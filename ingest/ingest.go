package ingest

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"shaper/util"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/marcboeker/go-duckdb"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// TODO: Move consts to config
const (
	BATCH_SIZE     = 3000
	BATCH_TIMEOUT  = 2000 * time.Millisecond
	SLEEP_ON_ERROR = 10 * time.Second

	SQL_TYPE_BOOLEAN   = "BOOLEAN"
	SQL_TYPE_DOUBLE    = "DOUBLE"
	SQL_TYPE_TIMESTAMP = "TIMESTAMP"
	SQL_TYPE_DATE      = "DATE"
	SQL_TYPE_VARCHAR   = "VARCHAR"
	SQL_TYPE_JSON      = "JSON"
)

// Add timestamp formats to try
var timestampFormats = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05.000Z07:00",
	"2006-01-02", // Simple date format
	"01/02/2006", // MM/DD/YYYY
	"02/01/2006", // DD/MM/YYYY
}

type Ingest struct {
	ingestCancelFunc context.CancelFunc
	subjectPrefix    string
}

type ColInfo struct {
	ColumnName string `db:"column_name"`
	Null       string `db:"null"`
	Type       string `db:"column_type"`
}

type TableCache struct {
	columns    []ColInfo
	lastUpdate time.Time
}

func Start(subjectPrefix string, dbConnector *duckdb.Connector, db *sqlx.DB, logger *slog.Logger, nc *nats.Conn) (Ingest, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return Ingest{}, err
	}

	consumer, err := setupStreamAndConsumer(js, subjectPrefix)
	if err != nil {
		return Ingest{}, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	go processMessages(ctx, js, consumer, logger, dbConnector, db, subjectPrefix)
	return Ingest{
		ingestCancelFunc: cancel,
		subjectPrefix:    subjectPrefix,
	}, nil
}

func setupStreamAndConsumer(js jetstream.JetStream, subjectPrefix string) (jetstream.Consumer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "shaper-ingest",
		Subjects: []string{subjectPrefix + ">"},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create/update stream: %w", err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       "shaper-ingest",
		MaxAckPending: BATCH_SIZE,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create/update consumer: %w", err)
	}

	return consumer, nil
}

func processMessages(ctx context.Context, js jetstream.JetStream, consumer jetstream.Consumer, logger *slog.Logger, dbConnector *duckdb.Connector, db *sqlx.DB, subjectPrefix string) {
	tableCache := make(map[string]TableCache)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := handleMessageBatches(ctx, consumer, logger, dbConnector, db, tableCache, subjectPrefix)
			if err != nil {
				logger.Error("Message handling failed, attempting to recreate consumer", slog.Any("error", err), slog.Duration("sleep", SLEEP_ON_ERROR))
				time.Sleep(SLEEP_ON_ERROR)

				logger.Info("Attempting to recreate ingest consumer")
				newConsumer, err := setupStreamAndConsumer(js, subjectPrefix)
				if err != nil {
					logger.Error("Failed to recreate ingest consumer", slog.Any("error", err))
					os.Exit(1)
				}
				logger.Info("Recreated ingest consumer")
				consumer = newConsumer
			}
		}
	}
}

func handleMessageBatches(ctx context.Context, c jetstream.Consumer, logger *slog.Logger, dbConnector *duckdb.Connector, db *sqlx.DB, tableCache map[string]TableCache, subjectPrefix string) error {
	batch := make([]jetstream.Msg, 0, BATCH_SIZE)
	msgChan := make(chan jetstream.Msg, BATCH_SIZE)
	errChan := make(chan error, 1)

	// Start message consumer in separate goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(msgChan)

		msgs, err := c.Messages()
		if err != nil {
			logger.Error("Failed to get messages iterator", slog.Any("error", err))
			errChan <- fmt.Errorf("failed to get messages iterator: %w", err)
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := msgs.Next()
				if err != nil {
					if err == context.Canceled {
						return
					}
					logger.Error("Failed to get next message", slog.Any("error", err))
					errChan <- fmt.Errorf("failed to get next message: %w", err)
					return
				}
				select {
				case msgChan <- msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Create timer but don't start it yet
	batchTimer := time.NewTimer(BATCH_TIMEOUT)
	if !batchTimer.Stop() {
		<-batchTimer.C
	}
	defer batchTimer.Stop()

	for {
		select {
		case err := <-errChan:
			return err
		case msg, ok := <-msgChan:
			if !ok {
				// Channel closed, process remaining messages and return
				if len(batch) > 0 {
					processStartTime := time.Now()
					if err := processBatch(context.Background(), batch, tableCache, dbConnector, db, subjectPrefix); err != nil {
						return fmt.Errorf("failed to process final batch: %w", err)
					}
					logger.Info("Processed final ingest batch", slog.Int("size", len(batch)), slog.Duration("duration", time.Since(processStartTime)))
				}
				wg.Wait()
				return nil
			}

			// Add to batch
			batch = append(batch, msg)

			// Start/reset timer when we get the first message in a batch
			if len(batch) == 1 {
				batchTimer.Reset(BATCH_TIMEOUT)
			}

			// Process if batch is full
			if len(batch) >= BATCH_SIZE {
				processStartTime := time.Now()
				if err := processBatch(context.Background(), batch, tableCache, dbConnector, db, subjectPrefix); err != nil {
					logger.Error("Failed to process batch", slog.Any("error", err), slog.Int("size", len(batch)), slog.Duration("duration", time.Since(processStartTime)))
				} else {
					logger.Info("Processed ingest batch", slog.Int("size", len(batch)), slog.Duration("duration", time.Since(processStartTime)))
				}
				batch = make([]jetstream.Msg, 0, BATCH_SIZE)
				// Stop timer after processing
				if !batchTimer.Stop() {
					<-batchTimer.C
				}
			}

		case <-batchTimer.C:
			// Process non-empty batch
			if len(batch) > 0 {
				processStartTime := time.Now()
				if err := processBatch(context.Background(), batch, tableCache, dbConnector, db, subjectPrefix); err != nil {
					logger.Error("Failed to process batch", slog.Any("error", err), slog.Int("size", len(batch)), slog.Duration("duration", time.Since(processStartTime)))
				} else {
					logger.Info("Processed ingest batch", slog.Int("size", len(batch)), slog.Duration("duration", time.Since(processStartTime)))
				}
				batch = make([]jetstream.Msg, 0, BATCH_SIZE)
			}

		case <-ctx.Done():
			// Process remaining messages before shutting down
			if len(batch) > 0 {
				if err := processBatch(context.Background(), batch, tableCache, dbConnector, db, subjectPrefix); err != nil {
					logger.Error("Failed to process final batch", slog.Any("error", err))
				}
			}
			wg.Wait()
			return nil
		}
	}
}

func tableExists(ctx context.Context, db *sqlx.DB, tableName string) (bool, error) {
	query := "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?"
	var count int
	err := db.GetContext(ctx, &count, query, tableName)
	if err != nil {
		return false, fmt.Errorf("failed to check if table exists: %w", err)
	}
	return count > 0, nil
}

const tableColumnsQuery = "SELECT column_name, \"null\", column_type FROM (DESCRIBE (FROM query_table($1)))"

func getTableColumns(ctx context.Context, db *sqlx.DB, tableName string) ([]ColInfo, error) {
	var columns []ColInfo
	err := db.SelectContext(ctx, &columns, tableColumnsQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get table columns: %w", err)
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("table %s not found or has no columns", tableName)
	}
	return columns, nil
}

func detectSchemaFromBatch(messages []jetstream.Msg) (map[string]string, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("cannot detect schema from empty batch")
	}

	// First pass: collect all field names and sample values
	columnSamples := make(map[string][]any)
	for _, msg := range messages {
		var jsonData map[string]any
		if err := json.Unmarshal(msg.Data(), &jsonData); err != nil {
			return nil, fmt.Errorf("failed to parse JSON message: %w", err)
		}

		// Collect all field names and sample values
		for field, value := range jsonData {
			if _, exists := columnSamples[field]; !exists {
				columnSamples[field] = make([]any, 0, len(messages))
			}
			columnSamples[field] = append(columnSamples[field], value)
		}
	}

	// Second pass: determine the best type for each column
	columnTypes := make(map[string]string)
	for field, samples := range columnSamples {
		columnTypes[field] = determineColumnType(samples)
	}

	return columnTypes, nil
}

// Function to determine the best SQL type for a column based on samples
func determineColumnType(samples []any) string {
	if len(samples) == 0 {
		return SQL_TYPE_JSON // Default to JSON for empty samples
	}

	// Track what types of values we've seen
	var hasTimestamp bool
	var hasDate bool
	var hasString bool
	var hasNumber bool
	var hasBoolean bool
	var hasComplexType bool

	for _, sample := range samples {
		if sample == nil {
			continue
		}

		val := reflect.ValueOf(sample)
		kind := val.Kind()

		switch kind {
		case reflect.Bool:
			hasBoolean = true
		case reflect.Float64:
			hasNumber = true
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			hasNumber = true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			hasNumber = true
		case reflect.String:
			hasString = true
			// Try to parse as timestamp or date
			strVal := sample.(string)
			if isTimestamp(strVal) {
				hasTimestamp = true
			} else if isDate(strVal) {
				hasDate = true
			}
		case reflect.Map, reflect.Slice:
			hasComplexType = true
		}
	}

	if hasBoolean && !hasString && !hasNumber && !hasComplexType {
		return SQL_TYPE_BOOLEAN
	}

	if hasNumber && !hasString && !hasBoolean && !hasComplexType {
		return SQL_TYPE_DOUBLE
	}

	if hasString && !hasNumber && !hasBoolean && !hasComplexType {
		if hasTimestamp && !hasDate {
			return SQL_TYPE_TIMESTAMP
		}
		if hasDate && !hasTimestamp {
			return SQL_TYPE_DATE
		}
		return SQL_TYPE_VARCHAR
	}

	// Default to JSON for anything else
	return SQL_TYPE_JSON
}

func createTable(ctx context.Context, db *sqlx.DB, tableName string, columnTypes map[string]string) error {
	if len(columnTypes) == 0 {
		return fmt.Errorf("cannot create table with no columns")
	}

	// Escape and quote the table name
	escapedTableName := fmt.Sprintf("\"%s\"", util.EscapeSQLIdentifier(tableName))

	// Build CREATE TABLE statement
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", escapedTableName))

	i := 0
	for column, dataType := range columnTypes {
		if i > 0 {
			sb.WriteString(",\n")
		}
		// Escape and quote the column name
		escapedColumnName := fmt.Sprintf("\"%s\"", util.EscapeSQLIdentifier(column))
		sb.WriteString(fmt.Sprintf("  %s %s", escapedColumnName, dataType))
		i++
	}
	sb.WriteString("\n)")

	// Execute the CREATE TABLE statement
	_, err := db.ExecContext(ctx, sb.String())
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

func processBatch(ctx context.Context, batch []jetstream.Msg, tableCache map[string]TableCache, dbConnector *duckdb.Connector, db *sqlx.DB, subjectPrefix string) error {
	// Group messages by table
	tableMessages := make(map[string][]jetstream.Msg)
	for _, msg := range batch {
		tableName := strings.TrimPrefix(msg.Subject(), subjectPrefix)
		tableMessages[tableName] = append(tableMessages[tableName], msg)
	}

	// Process each table's messages
	for tableName, messages := range tableMessages {
		// Check if table exists
		exists, err := tableExists(ctx, db, tableName)
		if err != nil {
			return fmt.Errorf("failed to check if table exists: %w", err)
		}

		// Detect schema from batch
		columnTypes, err := detectSchemaFromBatch(messages)
		if err != nil {
			return fmt.Errorf("failed to detect schema for table %s: %w", tableName, err)
		}

		if !exists {
			// Create the table if it doesn't exist
			err = createTable(ctx, db, tableName, columnTypes)
			if err != nil {
				return fmt.Errorf("failed to create table %s: %w", tableName, err)
			}
		} else {
			// Table exists - check for new columns
			columns, err := getTableColumns(ctx, db, tableName)
			if err != nil {
				return fmt.Errorf("failed to get table columns: %w", err)
			}

			// Build a map of existing columns
			existingColumns := make(map[string]bool)
			for _, col := range columns {
				existingColumns[col.ColumnName] = true
			}

			// Check for new columns in the detected schema
			for column, dataType := range columnTypes {
				if !existingColumns[column] {
					// New column found - add it to the table
					escapedTableName := fmt.Sprintf("\"%s\"", util.EscapeSQLIdentifier(tableName))
					escapedColumnName := fmt.Sprintf("\"%s\"", util.EscapeSQLIdentifier(column))
					alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", escapedTableName, escapedColumnName, dataType)
					if _, err := db.ExecContext(ctx, alterSQL); err != nil {
						return fmt.Errorf("failed to add new column %s: %w", column, err)
					}
				}
			}
		}

		// Get or update table schema from cache
		tableCache[tableName] = TableCache{
			columns:    nil, // Force refresh
			lastUpdate: time.Time{},
		}

		// Now get the updated columns
		columns, err := getTableColumns(ctx, db, tableName)
		if err != nil {
			return fmt.Errorf("failed to get table columns: %w", err)
		}
		tableCache[tableName] = TableCache{
			columns:    columns,
			lastUpdate: time.Now(),
		}
		tableInfo := tableCache[tableName]

		// Get DB connection
		conn, err := dbConnector.Connect(ctx)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer conn.Close()

		// Create appender
		appender, err := duckdb.NewAppenderFromConn(conn, "", tableName)
		if err != nil {
			return fmt.Errorf("failed to create appender: %w", err)
		}

		// Process messages for this table
		for _, msg := range messages {
			// Parse message
			var jsonData map[string]any
			if err := json.Unmarshal(msg.Data(), &jsonData); err != nil {
				return fmt.Errorf("failed to parse JSON message: %w", err)
			}

			metadata, err := msg.Metadata()
			if err != nil {
				return fmt.Errorf("failed to get message metadata: %w", err)
			}

			values := make([]driver.Value, len(tableInfo.columns))
			for j, col := range tableInfo.columns {
				value, exists := jsonData[col.ColumnName]
				if !exists {
					if col.Null == "YES" {
						values[j] = nil
					} else {
						return fmt.Errorf("missing required column %s (SEQ %d)", col.ColumnName, metadata.Sequence.Stream)
					}
				} else {
					if strings.Contains(strings.ToUpper(col.Type), "TIMESTAMP") {
						switch v := value.(type) {
						case string:
							// Try parsing the timestamp string
							ts, err := parseTimestamp(v)
							if err != nil {
								return fmt.Errorf("failed to parse timestamp for column %s: %w (SEQ %d)", col.ColumnName, err, metadata.Sequence.Stream)
							}
							values[j] = ts
						case float64:
							// Assume Unix timestamp in seconds or milliseconds
							values[j] = parseUnixTimestamp(v)
						default:
							return fmt.Errorf("unsupported timestamp format for column %s (SEQ %d)", col.ColumnName, metadata.Sequence.Stream)
						}
					} else if strings.Contains(strings.ToUpper(col.Type), "DATE") {
						switch v := value.(type) {
						case string:
							// Try parsing the date string
							date, err := parseDate(v)
							if err != nil {
								return fmt.Errorf("failed to parse date for column %s: %w (SEQ %d)", col.ColumnName, err, metadata.Sequence.Stream)
							}
							values[j] = date
						default:
							return fmt.Errorf("unsupported date format for column %s (SEQ %d)", col.ColumnName, metadata.Sequence.Stream)
						}
					} else if strings.Contains(strings.ToUpper(col.Type), "JSON") && (value != nil) {
						// For JSON columns, convert the value to a JSON string
						jsonBytes, err := json.Marshal(value)
						if err != nil {
							return fmt.Errorf("failed to marshal JSON for column %s: %w (SEQ %d)", col.ColumnName, err, metadata.Sequence.Stream)
						}
						values[j] = string(jsonBytes)
					} else {
						values[j] = value
					}
				}
			}

			if err := appender.AppendRow(values...); err != nil {
				return fmt.Errorf("failed to append row: %w (SEQ %d)", err, metadata.Sequence.Stream)
			}
		}

		err = appender.Close()
		if err != nil {
			return fmt.Errorf("failed to close appender: %w", err)
		}

		// Acknowledge messages after appender is closed
		for _, msg := range messages {
			if err := msg.Ack(); err != nil {
				return fmt.Errorf("failed to acknowledge message: %w", err)
			}
		}
	}

	return nil
}

func isTimestamp(value string) bool {
	for _, format := range timestampFormats {
		if _, err := time.Parse(format, value); err == nil {
			// Only consider it a timestamp if it has time component
			return strings.Contains(format, "15:04:05")
		}
	}
	return false
}

func isDate(value string) bool {
	for _, format := range timestampFormats {
		if _, err := time.Parse(format, value); err == nil {
			// Consider it a date if it doesn't have time component
			return !strings.Contains(format, "15:04:05")
		}
	}
	return false
}

func parseDate(value string) (time.Time, error) {
	// Try date formats
	for _, format := range timestampFormats {
		if !strings.Contains(format, "15:04:05") { // Date formats don't have time component
			if t, err := time.Parse(format, value); err == nil {
				return t, nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", value)
}

func parseTimestamp(value string) (time.Time, error) {
	// Try common timestamp formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.000Z07:00",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", value)
}

// Helper function to parse Unix timestamps
func parseUnixTimestamp(value float64) time.Time {
	// If the number is too large to be seconds, assume it's milliseconds
	if value > 1e11 {
		return time.UnixMilli(int64(value))
	}
	return time.Unix(int64(value), 0)
}

func (c Ingest) Close() {
	c.ingestCancelFunc()
}
