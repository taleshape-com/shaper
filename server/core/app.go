// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/marcboeker/go-duckdb/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	CONFIG_KEY_JWT_SECRET = "jwt-secret"
)

// TODO: Rename App struct + file to Core to not confuse with apps data type
type App struct {
	Name                       string
	NodeID                     string
	Sqlite                     *sqlx.DB
	DuckDB                     *sqlx.DB
	Logger                     *slog.Logger
	LoginRequired              bool
	BasePath                   string
	JWTSecret                  []byte
	JWTExp                     time.Duration
	SessionExp                 time.Duration
	InviteExp                  time.Duration
	NoPublicSharing            bool
	NoPasswordProtectedSharing bool
	NoTasks                    bool
	StateConsumeCtx            jetstream.ConsumeContext
	TaskConsumeCtx             jetstream.ConsumeContext
	TaskResultConsumeCtx       jetstream.ConsumeContext
	JetStream                  jetstream.JetStream
	ConfigKV                   jetstream.KeyValue
	NATSConn                   *nats.Conn
	StateSubjectPrefix         string
	IngestSubjectPrefix        string
	StateStreamName            string
	StateStreamMaxAge          time.Duration
	StateConsumerName          string
	ConfigKVBucketName         string
	TasksStreamName            string
	TasksSubjectPrefix         string
	TaskQueueConsumerName      string
	TaskResultsStreamName      string
	TaskResultsSubjectPrefix   string
	TaskResultsStreamMaxAge    time.Duration
	TaskResultConsumerName     string
	TaskBroadcastSubject       string
	TaskBroadcastSubscription  *nats.Subscription
	TaskTimers                 map[string]*time.Timer
}

func New(
	name string,
	nodeID string,
	sqliteDbx *sqlx.DB,
	duckDbx *sqlx.DB,
	deprecatedSchema string,
	logger *slog.Logger,
	baseURL string,
	jwtExp time.Duration,
	sessionExp time.Duration,
	inviteExp time.Duration,
	noPublicSharing bool,
	noPasswordProtectedSharing bool,
	noTasks bool,
	ingestSubjectPrefix string,
	stateSubjectPrefix string,
	stateStreamName string,
	stateStreamMaxAge time.Duration,
	stateConsumerName string,
	configKVBucketName string,
	tasksStreamName string,
	tasksSubjectPrefix string,
	taskQueueConsumerName string,
	taskResultsStreamName string,
	taskResultsSubjectPrefix string,
	taskResultsStreamMaxAge time.Duration,
	taskResultConsumerName string,
	taskBroadcastSubject string,
) (*App, error) {
	if err := initSQLite(sqliteDbx); err != nil {
		return nil, err
	}

	if err := initDuckDB(duckDbx); err != nil {
		return nil, err
	}

	// TODO: Remove this once data is migrated for all active users
	if err := migrateSystemData(sqliteDbx, duckDbx, deprecatedSchema, logger); err != nil {
		return nil, err
	}

	loginRequired, err := isLoginRequired(sqliteDbx)
	if err != nil {
		return nil, err
	}
	if !loginRequired {
		logger.Warn("No users found. Authentication is disabled until first user is created. Make sure you don't expose sensitive data publicly.")
	}

	if noPublicSharing {
		logger.Info("Publicly sharing dashboards is disabled.")
	}
	if noPasswordProtectedSharing {
		logger.Info("Sharing dashboards protected with a password is disabled.")
	}
	if noTasks {
		logger.Info("Tasks functionality disabled.")
	}

	app := &App{
		Name:                       name,
		NodeID:                     nodeID,
		Sqlite:                     sqliteDbx,
		DuckDB:                     duckDbx,
		Logger:                     logger,
		LoginRequired:              loginRequired,
		BasePath:                   baseURL,
		JWTExp:                     jwtExp,
		SessionExp:                 sessionExp,
		InviteExp:                  inviteExp,
		NoPublicSharing:            noPublicSharing,
		NoPasswordProtectedSharing: noPasswordProtectedSharing,
		NoTasks:                    noTasks,
		IngestSubjectPrefix:        ingestSubjectPrefix,
		StateSubjectPrefix:         stateSubjectPrefix,
		StateStreamName:            stateStreamName,
		StateStreamMaxAge:          stateStreamMaxAge,
		StateConsumerName:          stateConsumerName,
		ConfigKVBucketName:         configKVBucketName,
		TasksStreamName:            tasksStreamName,
		TasksSubjectPrefix:         tasksSubjectPrefix,
		TaskQueueConsumerName:      taskQueueConsumerName,
		TaskResultsStreamName:      taskResultsStreamName,
		TaskResultsSubjectPrefix:   taskResultsSubjectPrefix,
		TaskResultsStreamMaxAge:    taskResultsStreamMaxAge,
		TaskResultConsumerName:     taskResultConsumerName,
		TaskBroadcastSubject:       taskBroadcastSubject,
		TaskTimers:                 make(map[string]*time.Timer),
	}
	return app, nil
}

func (app *App) Init(nc *nats.Conn) error {
	app.NATSConn = nc
	js, err := jetstream.New(nc)
	app.JetStream = js
	if err != nil {
		return fmt.Errorf("failed to create jetstream: %w", err)
	}

	// Create stream and consumer
	if err := app.setupStreamAndConsumer(); err != nil {
		return fmt.Errorf("failed to setup streams and consumers: %w", err)
	}

	if err := LoadJWTSecret(app); err != nil {
		return fmt.Errorf("failed to load JWT secret: %w", err)
	}

	if !app.NoTasks {
		if err := scheduleExistingTasks(app, context.Background()); err != nil {
			return fmt.Errorf("failed to schedule existing tasks: %w", err)
		}
	}

	return nil
}

// TODO: allow setting MaxMsg, MaxBytes per stream
func (app *App) setupStreamAndConsumer() error {
	initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer initCancel()

	// All app changes go through state stream. Think event sourcing.
	stream, err := app.JetStream.CreateOrUpdateStream(initCtx, jetstream.StreamConfig{
		Name:     app.StateStreamName,
		Subjects: []string{app.StateSubjectPrefix + ">"},
		Storage:  jetstream.FileStorage,
		MaxAge:   app.StateStreamMaxAge,
	})
	if err != nil {
		return fmt.Errorf("failed to create or update state stream: %w", err)
	}

	stateConsumer, err := stream.CreateOrUpdateConsumer(initCtx, jetstream.ConsumerConfig{
		Durable:       app.StateConsumerName,
		MaxAckPending: 1,
	})
	if err != nil {
		return fmt.Errorf("failed to create or update state consumer: %w", err)
	}

	// For now only the JWT secret is stored in NATS KV. It fits the persistence model nicely since it's fine if it resets.
	configKV, err := app.JetStream.CreateOrUpdateKeyValue(initCtx, jetstream.KeyValueConfig{
		Bucket: app.ConfigKVBucketName,
	})
	if err != nil {
		return fmt.Errorf("failed to create or update config KV: %w", err)
	}
	app.ConfigKV = configKV

	if !app.NoTasks {
		taskBroadcastSub, err := app.NATSConn.Subscribe(app.TaskBroadcastSubject, app.HandleTaskBroadcast)
		if err != nil {
			return fmt.Errorf("failed to subscribe to task broadcast: %w", err)
		}
		app.TaskBroadcastSubscription = taskBroadcastSub

		// Task invocations are coordinate via this NATS work queue stream to ensure that tasks only run on one node in a Shaper cluster.
		tasksStream, err := app.JetStream.CreateOrUpdateStream(initCtx, jetstream.StreamConfig{
			Name:                 app.TasksStreamName,
			Subjects:             []string{app.TasksSubjectPrefix + ">"},
			Storage:              jetstream.FileStorage,
			DiscardNewPerSubject: true,
			Discard:              jetstream.DiscardNew,
			MaxMsgsPerSubject:    1,
			Retention:            jetstream.WorkQueuePolicy,
		})
		if err != nil {
			return fmt.Errorf("failed to create or update tasks stream: %w", err)
		}
		taskConsumer, err := tasksStream.CreateOrUpdateConsumer(initCtx, jetstream.ConsumerConfig{
			Durable: app.TaskQueueConsumerName,
		})
		if err != nil {
			return fmt.Errorf("failed to create or update task consumer: %w", err)
		}
		taskConsumeCtx, err := taskConsumer.Consume(app.HandleTask)
		if err != nil {
			return fmt.Errorf("failed to consume tasks: %w", err)
		}
		app.TaskConsumeCtx = taskConsumeCtx

		// Task run results are published to all nodes in the cluster via this stream to ensure all nodes have task state in the database and schedule tasks.
		// We are not using the state stream for results since task results have different persistence requirements. Task results potentitally happen more frequently than state changes, but they do not need to be retained after each node processed them.
		taskResultsStream, err := app.JetStream.CreateOrUpdateStream(initCtx, jetstream.StreamConfig{
			Name:              app.TaskResultsStreamName,
			Subjects:          []string{app.TaskResultsSubjectPrefix + ">"},
			Storage:           jetstream.FileStorage,
			MaxAge:            app.TaskResultsStreamMaxAge,
			MaxMsgsPerSubject: 1,
		})
		if err != nil {
			return fmt.Errorf("failed to create or update task results stream: %w", err)
		}
		taskResultConsumer, err := taskResultsStream.CreateOrUpdateConsumer(initCtx, jetstream.ConsumerConfig{
			Durable: app.TaskResultConsumerName,
		})
		if err != nil {
			return fmt.Errorf("failed to create or update task result consumer: %w", err)
		}
		taskResultConsumeCtx, err := taskResultConsumer.Consume(app.HandleTaskResult)
		if err != nil {
			return fmt.Errorf("failed to consume task results: %w", err)
		}
		app.TaskResultConsumeCtx = taskResultConsumeCtx
	}

	stateConsumeCtx, err := stateConsumer.Consume(app.HandleState)
	if err != nil {
		return fmt.Errorf("failed to consume state: %w", err)
	}
	app.StateConsumeCtx = stateConsumeCtx

	return nil
}

func (app *App) Close() {
	if app.StateConsumeCtx != nil {
		app.StateConsumeCtx.Drain()
		<-app.StateConsumeCtx.Closed()
	}
	if app.TaskConsumeCtx != nil {
		app.TaskConsumeCtx.Drain()
		<-app.TaskConsumeCtx.Closed()
	}
	if app.TaskResultConsumeCtx != nil {
		app.TaskResultConsumeCtx.Drain()
		<-app.TaskResultConsumeCtx.Closed()
	}
	if app.TaskBroadcastSubscription != nil {
		app.TaskBroadcastSubscription.Unsubscribe()
	}
}

func isLoginRequired(sdb *sqlx.DB) (bool, error) {
	var count int
	err := sdb.Get(&count, `
		SELECT count(*)
		FROM users
		WHERE deleted_at IS NULL
	`)
	return count > 0, err
}

func initDuckDB(duckDbx *sqlx.DB) error {
	// Create custom types
	for _, t := range dbTypes {
		if err := createType(duckDbx, t.Name, t.Definition); err != nil {
			return fmt.Errorf("failed to create custom type %s: %w", t.Name, err)
		}
	}
	return nil
}

func migrateSystemData(sqliteDbx *sqlx.DB, duckDbx *sqlx.DB, deprecatedSchema string, logger *slog.Logger) error {
	for _, table := range []string{
		"apps",
		"api_keys",
		"users",
		"sessions",
		"invites",
		"task_runs",
	} {
		if err := migrateTableData(sqliteDbx, duckDbx, table, deprecatedSchema, logger); err != nil {
			return fmt.Errorf("failed to migrate table %s: %w", table, err)
		}
	}
	return nil
}

// The database contains little data so we can read all data into memory and write it to sqlite database.
// We don't delete data from duckdb after migrating.
// We only migrate data if the target table is empty.
func migrateTableData(sqliteDbx *sqlx.DB, duckDbx *sqlx.DB, table string, deprecatedSchema string, logger *slog.Logger) error {
	duckDbTable := "\"" + deprecatedSchema + "\"." + table
	var count int
	err := duckDbx.Get(&count, fmt.Sprintf(`SELECT count(*) FROM %s`, duckDbTable))
	// Skip if table not in DuckDB
	if err != nil {
		return nil
	}
	if count == 0 {
		// DuckDB table has no data, skipping
		return nil
	}
	// Check if table already has data
	err = sqliteDbx.Get(&count, fmt.Sprintf(`SELECT count(*) FROM %s`, table))
	if err != nil {
		return fmt.Errorf("failed to count rows in sqlite table %s: %w", table, err)
	}
	if count > 0 {
		// Table already has data, skip migration
		return nil
	}
	logger.Info("Migrating table to SQLite", slog.String("table", table))
	rows, err := duckDbx.Queryx(fmt.Sprintf(`SELECT * FROM %s`, duckDbTable))
	if err != nil {
		return fmt.Errorf("failed to query duckdb table %s: %w", duckDbTable, err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns for duckdb table %s: %w", duckDbTable, err)
	}
	tx, err := sqliteDbx.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for sqlite table %s: %w", table, err)
	}
	// Migrate each row
	for rows.Next() {
		values := make([]any, len(cols))
		valuePtrs := make([]any, len(cols))
		for i := range cols {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to scan row for duckdb table %s: %w", duckDbTable, err)
		}
		placeholders := ""
		for i := range cols {
			if i > 0 {
				placeholders += ", "
			}
			placeholders += "?"
			if values[i] != nil {
				if v, ok := values[i].(time.Time); ok {
					values[i] = v.UnixMilli()
				}
				if table == "task_runs" {
					if cols[i] == "last_run_duration" {
						v, ok := values[i].(duckdb.Interval)
						if !ok {
							tx.Rollback()
							return fmt.Errorf("failed to convert last_run_duration to time for duckdb table %s", table)
						}
						values[i] = formatInterval(v)
					} else if cols[i] == "last_run_success" {
						v, ok := values[i].(bool)
						if !ok {
							tx.Rollback()
							return fmt.Errorf("failed to convert last_run_success to bool for duckdb table %s", table)
						}
						if v {
							values[i] = 1
						} else {
							values[i] = 0
						}
					}
				}
			}
		}
		_, err := tx.Exec(
			fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, table, strings.Join(cols, ", "), placeholders),
			values...,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert row into sqlite table %s: %w", table, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for sqlite table %s: %w", table, err)
	}
	return nil
}
