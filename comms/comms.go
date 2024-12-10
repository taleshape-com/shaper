package comms

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
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// TODO: Move consts to config
const (
	SUBJECT_PREFIX         = "shaper.ingest."
	CONNECT_TIMEOUT        = 10 * time.Second
	EVENT_ID_COLUMN        = "event_id"
	EVENT_TIMESTAMP_COLUMN = "event_timestamp"
	BATCH_SIZE             = 1000
	BATCH_TIMEOUT          = 2000 * time.Millisecond
)

type Comms struct {
	nc               *nats.Conn
	ns               *server.Server
	ingestCancelFunc context.CancelFunc
	db               *sqlx.DB
}

type ColInfo struct {
	ColumnName string `db:"column_name"`
	Null       string `db:"null"`
}

type BatchMessage struct {
	msg      jetstream.Msg
	data     map[string]interface{}
	metadata *jetstream.MsgMetadata
	table    string
}

// TODO: remove type
type MessageBatch struct {
	messages []*BatchMessage
}

// TODO: remove type
type TableCache struct {
	columns    []ColInfo
	lastUpdate time.Time
}

func New(dbConnector *duckdb.Connector, db *sqlx.DB, logger *slog.Logger) (Comms, error) {
	// TODO: auth
	// TODO: allow changing nats host+port
	// TODO: support TLS
	// TODO: configure NATS logging
	// TODO: NATS prometheus metrics
	// TODO: JetStreamKey for disk encryption
	// TODO: Allow configuring stream retention
	opts := &server.Options{
		JetStream:              true,
		DisableJetStreamBanner: true,
		// TODO: DontListen as default. No NATS exposed. Only internally
		// DontListen:             true,
		// TODO: StoreDir
		StoreDir: "./jetstream",
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		return Comms{}, err
	}
	ns.ConfigureLogger()
	go ns.Start()
	if !ns.ReadyForConnections(CONNECT_TIMEOUT) {
		return Comms{}, err
	}
	clientOpts := []nats.Option{}
	// TODO: Make inprocess optional. Allow connecting to remote NATS
	clientOpts = append(clientOpts, nats.InProcessServer(ns))
	nc, err := nats.Connect(ns.ClientURL(), clientOpts...)
	if err != nil {
		return Comms{}, err
	}
	js, err := jetstream.New(nc)
	if err != nil {
		return Comms{}, err
	}
	initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer initCancel()
	stream, err := js.CreateOrUpdateStream(initCtx, jetstream.StreamConfig{
		Name:     "shaper-ingest",
		Subjects: []string{SUBJECT_PREFIX + ">"},
	})
	if err != nil {
		return Comms{}, err
	}
	// TODO: How to handle consumer errors? Should we set MaxDeliver? Any case we can fail writing to DuckDB?
	ingestConsumer, err := stream.CreateOrUpdateConsumer(initCtx, jetstream.ConsumerConfig{
		Durable: "shaper-ingest",
	})

	ctx, cancel := context.WithCancel(context.Background())
	go handleMessages(ctx, ingestConsumer, logger, dbConnector, db)
	return Comms{nc: nc, ns: ns, ingestCancelFunc: cancel, db: db}, err
}

func getTableColumns(ctx context.Context, db *sqlx.DB, tableName string) ([]ColInfo, error) {
	var columns []ColInfo
	query := "SELECT column_name, \"null\" FROM (DESCRIBE (FROM query_table($1)))"

	err := db.SelectContext(ctx, &columns, query, tableName)
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
	batch := &MessageBatch{
		messages: make([]*BatchMessage, 0, BATCH_SIZE),
	}

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
				if len(batch.messages) > 0 {
					if err := processBatch(context.Background(), batch, tableCache, dbConnector, db, logger); err != nil {
						logger.Error("Failed to process final batch", slog.Any("error", err))
					}
				}
				return
			}

			tableName := strings.TrimPrefix(msg.Subject(), SUBJECT_PREFIX)

			// Parse message
			var jsonData map[string]interface{}
			if err := json.Unmarshal(msg.Data(), &jsonData); err != nil {
				logger.Error("Failed to parse JSON message",
					slog.String("table", tableName),
					slog.Any("error", err))
				continue
			}

			metadata, err := msg.Metadata()
			if err != nil {
				logger.Error("Failed to get message metadata",
					slog.String("table", tableName),
					slog.Any("error", err))
				continue
			}

			// Add to batch
			batch.messages = append(batch.messages, &BatchMessage{
				msg:      msg,
				data:     jsonData,
				metadata: metadata,
				table:    tableName,
			})

			// Start/reset timer when we get the first message in a batch
			if len(batch.messages) == 1 {
				batchTimer.Reset(BATCH_TIMEOUT)
			}

			// Process if batch is full
			if len(batch.messages) >= BATCH_SIZE {
				if err := processBatch(context.Background(), batch, tableCache, dbConnector, db, logger); err != nil {
					logger.Error("Failed to process batch", slog.Any("error", err))
				}
				batch.messages = make([]*BatchMessage, 0, BATCH_SIZE)
				// Stop timer after processing
				if !batchTimer.Stop() {
					<-batchTimer.C
				}
			}

		case <-batchTimer.C:
			// Process non-empty batch
			if len(batch.messages) > 0 {
				if err := processBatch(context.Background(), batch, tableCache, dbConnector, db, logger); err != nil {
					logger.Error("Failed to process batch", slog.Any("error", err))
				}
				batch.messages = make([]*BatchMessage, 0, BATCH_SIZE)

			}

		case <-ctx.Done():
			// Process remaining messages before shutting down
			if len(batch.messages) > 0 {
				if err := processBatch(context.Background(), batch, tableCache, dbConnector, db, logger); err != nil {
					logger.Error("Failed to process final batch", slog.Any("error", err))
				}
			}
			wg.Wait()
			return
		}
	}
}

func processBatch(ctx context.Context, batch *MessageBatch, tableCache map[string]TableCache, dbConnector *duckdb.Connector, db *sqlx.DB, logger *slog.Logger) error {
	// Group messages by table
	tableMessages := make(map[string][]*BatchMessage)
	for _, msg := range batch.messages {
		tableMessages[msg.table] = append(tableMessages[msg.table], msg)
	}

	// Process each table's messages
	for tableName, messages := range tableMessages {
		// Get or update table schema from cache
		tableInfo, exists := tableCache[tableName]
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
		for _, batchMsg := range messages {
			values := make([]driver.Value, len(tableInfo.columns))
			for j, col := range tableInfo.columns {
				switch col.ColumnName {
				case EVENT_ID_COLUMN:
					if jsonValue, exists := batchMsg.data[EVENT_ID_COLUMN]; exists {
						values[j] = jsonValue
					} else {
						values[j] = batchMsg.metadata.Sequence
					}
				case EVENT_TIMESTAMP_COLUMN:
					if jsonValue, exists := batchMsg.data[EVENT_TIMESTAMP_COLUMN]; exists {
						values[j] = jsonValue
					} else {
						values[j] = batchMsg.metadata.Timestamp
					}
				default:
					value, exists := batchMsg.data[col.ColumnName]
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
			if err := batchMsg.msg.Ack(); err != nil {
				logger.Error("Failed to acknowledge message",
					slog.String("table", tableName),
					slog.Any("error", err))
			}
		}
	}

	return nil
}

func (c Comms) Close() {
	if c.ingestCancelFunc != nil {
		c.ingestCancelFunc()
	}
	c.ns.Shutdown()
	c.ns.WaitForShutdown()
}
