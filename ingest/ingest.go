package ingest

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
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
	SUBJECT_PREFIX = "shaper.ingest."
	BATCH_SIZE     = 1000
	BATCH_TIMEOUT  = 2000 * time.Millisecond
	SLEEP_ON_ERROR = 60 * time.Second
)

type Ingest struct {
	ingestCancelFunc context.CancelFunc
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

func Start(dbConnector *duckdb.Connector, db *sqlx.DB, logger *slog.Logger, nc *nats.Conn, persist bool) (Ingest, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return Ingest{}, err
	}

	consumer, err := setupStreamAndConsumer(js, persist)
	if err != nil {
		return Ingest{}, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	go processMessages(ctx, js, consumer, logger, dbConnector, db, persist)
	return Ingest{ingestCancelFunc: cancel}, nil
}

func setupStreamAndConsumer(js jetstream.JetStream, persist bool) (jetstream.Consumer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	storageType := jetstream.MemoryStorage
	if persist {
		storageType = jetstream.FileStorage
	}

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "shaper-ingest",
		Subjects: []string{SUBJECT_PREFIX + ">"},
		Storage:  storageType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create/update stream: %w", err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable: "shaper-ingest",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create/update consumer: %w", err)
	}

	return consumer, nil
}

func processMessages(ctx context.Context, js jetstream.JetStream, consumer jetstream.Consumer, logger *slog.Logger, dbConnector *duckdb.Connector, db *sqlx.DB, persist bool) {
	tableCache := make(map[string]TableCache)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := handleMessageBatches(ctx, consumer, logger, dbConnector, db, tableCache)
			if err != nil {
				logger.Error("Message handling failed, attempting to recreate consumer", slog.Any("error", err), slog.Duration("sleep", SLEEP_ON_ERROR))
				time.Sleep(SLEEP_ON_ERROR)

				logger.Info("Attempting to recreate ingest consumer")
				newConsumer, err := setupStreamAndConsumer(js, persist)
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

func handleMessageBatches(ctx context.Context, c jetstream.Consumer, logger *slog.Logger, dbConnector *duckdb.Connector, db *sqlx.DB, tableCache map[string]TableCache) error {
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
					if err := processBatch(context.Background(), batch, tableCache, dbConnector, db); err != nil {
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
				if err := processBatch(context.Background(), batch, tableCache, dbConnector, db); err != nil {
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
				if err := processBatch(context.Background(), batch, tableCache, dbConnector, db); err != nil {
					logger.Error("Failed to process batch", slog.Any("error", err), slog.Int("size", len(batch)), slog.Duration("duration", time.Since(processStartTime)))
				} else {
					logger.Info("Processed ingest batch", slog.Int("size", len(batch)), slog.Duration("duration", time.Since(processStartTime)))
				}
				batch = make([]jetstream.Msg, 0, BATCH_SIZE)
			}

		case <-ctx.Done():
			// Process remaining messages before shutting down
			if len(batch) > 0 {
				if err := processBatch(context.Background(), batch, tableCache, dbConnector, db); err != nil {
					logger.Error("Failed to process final batch", slog.Any("error", err))
				}
			}
			wg.Wait()
			return nil
		}
	}
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

func processBatch(ctx context.Context, batch []jetstream.Msg, tableCache map[string]TableCache, dbConnector *duckdb.Connector, db *sqlx.DB) error {
	// Group messages by table
	tableMessages := make(map[string][]jetstream.Msg)
	for _, msg := range batch {
		tableName := strings.TrimPrefix(msg.Subject(), SUBJECT_PREFIX)
		tableMessages[tableName] = append(tableMessages[tableName], msg)
	}

	// Process each table's messages
	for tableName, messages := range tableMessages {
		// Get or update table schema from cache
		tableInfo, exists := tableCache[tableName]
		// TODO: Rethink how to cache table schemas
		if !exists || time.Since(tableInfo.lastUpdate) > time.Hour {
			columns, err := getTableColumns(ctx, db, tableName)
			if err != nil {
				return fmt.Errorf("failed to get table columns: %w", err)
			}
			tableCache[tableName] = TableCache{
				columns:    columns,
				lastUpdate: time.Now(),
			}
			tableInfo = tableCache[tableName]
		}

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
			var jsonData map[string]interface{}
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

// Helper function to parse timestamp strings
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
