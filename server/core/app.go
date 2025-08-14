// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nrednav/cuid2"
)

const (
	CONFIG_KEY_JWT_SECRET = "jwt-secret"
)

// TODO: Rename App struct + file to Core to not confuse with apps data type
type App struct {
	Name                      string
	ProcessID                 string
	DB                        *sqlx.DB
	Logger                    *slog.Logger
	LoginRequired             bool
	BasePath                  string
	Schema                    string
	JWTSecret                 []byte
	JWTExp                    time.Duration
	SessionExp                time.Duration
	InviteExp                 time.Duration
	NoPublicSharing           bool
	NoTasks                   bool
	StateConsumeCtx           jetstream.ConsumeContext
	TaskConsumeCtx            jetstream.ConsumeContext
	TaskResultConsumeCtx      jetstream.ConsumeContext
	JetStream                 jetstream.JetStream
	ConfigKV                  jetstream.KeyValue
	NATSConn                  *nats.Conn
	StateSubjectPrefix        string
	IngestSubjectPrefix       string
	StateStreamName           string
	StateStreamMaxAge         time.Duration
	StateConsumerName         string
	ConfigKVBucketName        string
	TasksStreamName           string
	TasksSubjectPrefix        string
	TaskQueueConsumerName     string
	TaskResultsStreamName     string
	TaskResultsSubjectPrefix  string
	TaskResultsStreamMaxAge   time.Duration
	TaskResultConsumerName    string
	TaskBroadcastSubject      string
	TaskBroadcastSubscription *nats.Subscription
	TaskTimers                map[string]*time.Timer
}

func New(
	name string,
	db *sqlx.DB,
	logger *slog.Logger,
	baseURL string,
	schema string,
	jwtExp time.Duration,
	sessionExp time.Duration,
	inviteExp time.Duration,
	noPublicSharing bool,
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
	if err := initDB(db, schema); err != nil {
		return nil, err
	}

	loginRequired, err := isLoginRequired(db, schema)
	if err != nil {
		return nil, err
	}
	if !loginRequired {
		logger.Warn("No users found. Authentication is disabled until first user is created. Make sure you don't expose sensitive data publicly.")
	}

	if noPublicSharing {
		logger.Info("Publicly sharing dashboards is disabled.")
	}

	if noTasks {
		logger.Info("Tasks functionality disabled.")
	}

	app := &App{
		Name:                     name,
		ProcessID:                cuid2.Generate(),
		DB:                       db,
		Logger:                   logger,
		LoginRequired:            loginRequired,
		BasePath:                 baseURL,
		Schema:                   schema,
		JWTExp:                   jwtExp,
		SessionExp:               sessionExp,
		InviteExp:                inviteExp,
		NoPublicSharing:          noPublicSharing,
		NoTasks:                  noTasks,
		IngestSubjectPrefix:      ingestSubjectPrefix,
		StateSubjectPrefix:       stateSubjectPrefix,
		StateStreamName:          stateStreamName,
		StateStreamMaxAge:        stateStreamMaxAge,
		StateConsumerName:        stateConsumerName,
		ConfigKVBucketName:       configKVBucketName,
		TasksStreamName:          tasksStreamName,
		TasksSubjectPrefix:       tasksSubjectPrefix,
		TaskQueueConsumerName:    taskQueueConsumerName,
		TaskResultsStreamName:    taskResultsStreamName,
		TaskResultsSubjectPrefix: taskResultsSubjectPrefix,
		TaskResultsStreamMaxAge:  taskResultsStreamMaxAge,
		TaskResultConsumerName:   taskResultConsumerName,
		TaskBroadcastSubject:     taskBroadcastSubject,
		TaskTimers:               make(map[string]*time.Timer),
	}
	return app, nil
}

func (app *App) Init(nc *nats.Conn) error {
	app.NATSConn = nc
	js, err := jetstream.New(nc)
	app.JetStream = js
	if err != nil {
		return err
	}

	// Create stream and consumer
	if err := app.setupStreamAndConsumer(); err != nil {
		return err
	}

	if err := LoadJWTSecret(app); err != nil {
		return err
	}

	if !app.NoTasks {
		if err := scheduleExistingTasks(app, context.Background()); err != nil {
			return err
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
		return err
	}

	stateConsumer, err := stream.CreateOrUpdateConsumer(initCtx, jetstream.ConsumerConfig{
		Durable:       app.StateConsumerName,
		MaxAckPending: 1,
	})
	if err != nil {
		return err
	}

	// For now only the JWT secret is stored in NATS KV. It fits the persistence model nicely since it's fine if it resets.
	configKV, err := app.JetStream.CreateOrUpdateKeyValue(initCtx, jetstream.KeyValueConfig{
		Bucket: app.ConfigKVBucketName,
	})
	if err != nil {
		return err
	}
	app.ConfigKV = configKV

	if !app.NoTasks {
		taskBroadcastSub, err := app.NATSConn.Subscribe(app.TaskBroadcastSubject, app.HandleTaskBroadcast)
		if err != nil {
			return err
		}
		app.TaskBroadcastSubscription = taskBroadcastSub

		// Task invocations are coordinate via this NATS work queue stream to ensure that tasks only run on one node in a Shaper cluster.
		tasksStream, err := app.JetStream.CreateOrUpdateStream(initCtx, jetstream.StreamConfig{
			Name:      app.TasksStreamName,
			Subjects:  []string{app.TasksSubjectPrefix + ">"},
			Storage:   jetstream.FileStorage,
			Retention: jetstream.WorkQueuePolicy,
		})
		if err != nil {
			return err
		}
		taskConsumer, err := tasksStream.CreateOrUpdateConsumer(initCtx, jetstream.ConsumerConfig{
			Durable: app.TaskQueueConsumerName,
		})
		if err != nil {
			return err
		}
		taskConsumeCtx, err := taskConsumer.Consume(app.HandleTask)
		if err != nil {
			return err
		}
		app.TaskConsumeCtx = taskConsumeCtx

		// Task run results are published to all nodes in the cluster via this stream to ensure all nodes have task state in the database and schedule tasks.
		// We are not using the state stream for results since task results have different persistence requirements. Task results potentitally happen more frequently than state changes, but they do not need to be retained after each node processed them.
		taskResultsStream, err := app.JetStream.CreateOrUpdateStream(initCtx, jetstream.StreamConfig{
			Name:     app.TaskResultsStreamName,
			Subjects: []string{app.TaskResultsSubjectPrefix + ">"},
			Storage:  jetstream.FileStorage,
			MaxAge:   app.TaskResultsStreamMaxAge,
		})
		if err != nil {
			return err
		}
		taskResultConsumer, err := taskResultsStream.CreateOrUpdateConsumer(initCtx, jetstream.ConsumerConfig{
			Durable: app.TaskResultConsumerName,
		})
		if err != nil {
			return err
		}
		taskResultConsumeCtx, err := taskResultConsumer.Consume(app.HandleTaskResult)
		if err != nil {
			return err
		}
		app.TaskResultConsumeCtx = taskResultConsumeCtx
	}

	stateConsumeCtx, err := stateConsumer.Consume(app.HandleState)
	if err != nil {
		return err
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

func isLoginRequired(db *sqlx.DB, schema string) (bool, error) {
	var count int
	err := db.Get(&count, `
		SELECT count(*)
		FROM `+schema+`.users
		WHERE deleted_at IS NULL
		`)
	return count > 0, err
}
