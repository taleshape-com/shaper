// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
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
	StateConsumer              jetstream.Consumer
	ConfigKVBucketName         string
	TasksStreamName            string
	TasksSubjectPrefix         string
	TaskQueueConsumerName      string
	TaskResultsStreamName      string
	TaskResultsSubjectPrefix   string
	TaskResultsStreamMaxAge    time.Duration
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
	configKVBucketName string,
	tasksStreamName string,
	tasksSubjectPrefix string,
	taskQueueConsumerName string,
	taskResultsStreamName string,
	taskResultsSubjectPrefix string,
	taskResultsStreamMaxAge time.Duration,
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
		ConfigKVBucketName:         configKVBucketName,
		TasksStreamName:            tasksStreamName,
		TasksSubjectPrefix:         tasksSubjectPrefix,
		TaskQueueConsumerName:      taskQueueConsumerName,
		TaskResultsStreamName:      taskResultsStreamName,
		TaskResultsSubjectPrefix:   taskResultsSubjectPrefix,
		TaskResultsStreamMaxAge:    taskResultsStreamMaxAge,
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

	startSeq, err := getConsumerStartSeq(initCtx, app, INTERNAL_STATE_CONSUMER_NAME)
	if err != nil {
		return fmt.Errorf("failed to get state consumer start sequence: %w", err)
	}
	stateConsumer, err := stream.OrderedConsumer(initCtx, jetstream.OrderedConsumerConfig{
		DeliverPolicy: jetstream.DeliverByStartSequencePolicy,
		OptStartSeq:   startSeq,
	})
	if err != nil {
		return fmt.Errorf("failed to create or update state consumer: %w", err)
	}
	app.StateConsumer = stateConsumer

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
		taskResultStatSeq, err := getConsumerStartSeq(initCtx, app, INTERNAL_TASK_RESULTS_CONSUMER_NAME)
		if err != nil {
			return fmt.Errorf("failed to get task result consumer start sequence: %w", err)
		}
		taskResultConsumer, err := taskResultsStream.OrderedConsumer(initCtx, jetstream.OrderedConsumerConfig{
			DeliverPolicy: jetstream.DeliverByStartSequencePolicy,
			OptStartSeq:   taskResultStatSeq,
		})
		if err != nil {
			return fmt.Errorf("failed to create or update task result consumer: %w", err)
		}
		taskResultConsumeCtx, err := taskResultConsumer.Consume(app.HandleTaskResult)
		if err != nil {
			return fmt.Errorf("failed to consume task results: %w", err)
		}
		app.TaskResultConsumeCtx = taskResultConsumeCtx

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
		taskConsumeCtx, err := taskConsumer.Consume(app.HandleTask, jetstream.PullMaxMessages(1))
		if err != nil {
			return fmt.Errorf("failed to consume tasks: %w", err)
		}
		app.TaskConsumeCtx = taskConsumeCtx
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
