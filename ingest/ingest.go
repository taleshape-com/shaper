package ingest

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log/slog"
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
	SUBJECT_PREFIX         = "shaper.ingest."
	EVENT_ID_COLUMN        = "event_id"
	EVENT_TIMESTAMP_COLUMN = "event_timestamp"
	BATCH_SIZE             = 1000
	BATCH_TIMEOUT          = 2000 * time.Millisecond
)

type Ingest struct {
	ingestCancelFunc context.CancelFunc
}

type ColInfo struct {
	ColumnName string `db:"column_name"`
	Null       string `db:"null"`
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
	initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer initCancel()
	storageType := jetstream.MemoryStorage
	if persist {
		storageType = jetstream.FileStorage
	}
	stream, err := js.CreateOrUpdateStream(initCtx, jetstream.StreamConfig{
		Name:     "shaper-ingest",
		Subjects: []string{SUBJECT_PREFIX + ">"},
		Storage:  storageType,
	})
	if err != nil {
		return Ingest{}, err
	}
	ingestConsumer, err := stream.CreateOrUpdateConsumer(initCtx, jetstream.ConsumerConfig{
		Durable: "shaper-ingest",
	})
	ctx, cancel := context.WithCancel(context.Background())
	go handleMessages(ctx, ingestConsumer, logger, dbConnector, db)
	return Ingest{ingestCancelFunc: cancel}, err
}

const tableColumnsQuery = "SELECT column_name, \"null\" FROM (DESCRIBE (FROM query_table($1)))"

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

func handleMessages(ctx context.Context, c jetstream.Consumer, logger *slog.Logger, dbConnector *duckdb.Connector, db *sqlx.DB) {
	tableCache := make(map[string]TableCache)
	batch := make([]jetstream.Msg, 0, BATCH_SIZE)

	// Create buffered channel for messages
	msgChan := make(chan jetstream.Msg, BATCH_SIZE)

	// Start message consumer in separate goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(msgChan)

		msgs, err := c.Messages()
		if err != nil {
			logger.Error("Failed to get messages iterator", slog.Any("error", err))
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
					continue
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
	// Stop the timer initially since we don't have any messages yet
	if !batchTimer.Stop() {
		<-batchTimer.C
	}
	defer batchTimer.Stop()

	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				// Channel closed, process remaining messages and exit
				if len(batch) > 0 {
					processStartTime := time.Now()
					if err := processBatch(context.Background(), batch, tableCache, dbConnector, db); err != nil {
						logger.Error("Failed to process final batch", slog.Any("error", err), slog.Int("size", len(batch)), slog.Duration("duration", time.Since(processStartTime)))
					} else {
						logger.Info("Processed final ingest batch", slog.Int("size", len(batch)), slog.Duration("duration", time.Since(processStartTime)))
					}
				}
				return
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
			return
		}
	}
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
		defer appender.Close()

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
				switch col.ColumnName {
				case EVENT_ID_COLUMN:
					if jsonValue, exists := jsonData[EVENT_ID_COLUMN]; exists {
						values[j] = jsonValue
					} else {
						values[j] = metadata.Sequence
					}
				case EVENT_TIMESTAMP_COLUMN:
					if jsonValue, exists := jsonData[EVENT_TIMESTAMP_COLUMN]; exists {
						values[j] = jsonValue
					} else {
						values[j] = metadata.Timestamp
					}
				default:
					value, exists := jsonData[col.ColumnName]
					if !exists {
						if col.Null == "YES" {
							values[j] = nil
						} else {
							return fmt.Errorf("missing required column %s", col.ColumnName)
						}
					} else {
						values[j] = value
					}
				}
			}

			if err := appender.AppendRow(values...); err != nil {
				return fmt.Errorf("failed to append row: %w", err)
			}

			// Acknowledge message
			if err := msg.Ack(); err != nil {
				return fmt.Errorf("failed to acknowledge message: %w", err)
			}
		}
	}

	return nil
}

func (c Ingest) Close() {
	c.ingestCancelFunc()
}
