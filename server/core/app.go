// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	CONFIG_KEY_JWT_SECRET              = "jwt-secret"
	RECREATE_STREAM_AND_CONSUMER_DELAY = 60 * time.Second
)

// TODO: Rename App struct + file to Core to not confuse with apps data type
type App struct {
	Name                 string
	DB                   *sqlx.DB
	Logger               *slog.Logger
	LoginRequired        bool
	BasePath             string
	Schema               string
	JWTSecret            []byte
	JWTExp               time.Duration
	SessionExp           time.Duration
	InviteExp            time.Duration
	NoPublicSharing      bool
	NoWorkflows          bool
	StateConsumeCtx      jetstream.ConsumeContext
	JetStream            jetstream.JetStream
	ConfigKV             jetstream.KeyValue
	NATSConn             *nats.Conn
	StateSubjectPrefix   string
	IngestSubjectPrefix  string
	StateStreamName      string
	StateStreamMaxAge    time.Duration
	StateConsumerName    string
	ConfigKVBucketName   string
	JobsStreamName       string
	JobsSubjectPrefix    string
	JobQueueConsumerName string
	WorkflowTimers       map[string]*time.Timer
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
	noWorkflows bool,
	ingestSubjectPrefix string,
	stateSubjectPrefix string,
	stateStreamName string,
	stateStreamMaxAge time.Duration,
	stateConsumerName string,
	configKVBucketName string,
	jobsStreamName string,
	jobsSubjectPrefix string,
	jobQueueConsumerName string,
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

	if noWorkflows {
		logger.Info("Workflows functionality disabled.")
	}

	app := &App{
		Name:                 name,
		DB:                   db,
		Logger:               logger,
		LoginRequired:        loginRequired,
		BasePath:             baseURL,
		Schema:               schema,
		JWTExp:               jwtExp,
		SessionExp:           sessionExp,
		InviteExp:            inviteExp,
		NoPublicSharing:      noPublicSharing,
		NoWorkflows:          noWorkflows,
		IngestSubjectPrefix:  ingestSubjectPrefix,
		StateSubjectPrefix:   stateSubjectPrefix,
		StateStreamName:      stateStreamName,
		StateStreamMaxAge:    stateStreamMaxAge,
		StateConsumerName:    stateConsumerName,
		ConfigKVBucketName:   configKVBucketName,
		JobsStreamName:       jobsStreamName,
		JobsSubjectPrefix:    jobsSubjectPrefix,
		JobQueueConsumerName: jobQueueConsumerName,
		WorkflowTimers:       make(map[string]*time.Timer),
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

	// Start message processing
	go app.processStateMessages()

	return LoadJWTSecret(app)
}

// TODO: allow setting MaxMsg, MaxBytes per stream
func (app *App) setupStreamAndConsumer() error {
	initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer initCancel()

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

	configKV, err := app.JetStream.CreateOrUpdateKeyValue(initCtx, jetstream.KeyValueConfig{
		Bucket: app.ConfigKVBucketName,
	})
	if err != nil {
		return err
	}
	app.ConfigKV = configKV

	if !app.NoWorkflows {
		jobsStream, err := app.JetStream.CreateOrUpdateStream(initCtx, jetstream.StreamConfig{
			Name:      app.JobsStreamName,
			Subjects:  []string{app.JobsSubjectPrefix + ">"},
			Storage:   jetstream.FileStorage,
			Retention: jetstream.WorkQueuePolicy,
		})
		if err != nil {
			return err
		}
		jobConsumer, err := jobsStream.CreateOrUpdateConsumer(initCtx, jetstream.ConsumerConfig{
			Durable: app.JobQueueConsumerName,
		})
		if err != nil {
			return err
		}
		jobConsumer.Consume(app.HandleJob)
	}

	stateConsumeCtx, err := stateConsumer.Consume(app.HandleState)
	if err != nil {
		return err
	}
	app.StateConsumeCtx = stateConsumeCtx

	return nil
}

func (app *App) processStateMessages() {
	const sleepOnError = 60 * time.Second

	for {
		select {
		case <-app.StateConsumeCtx.Closed():
			app.Logger.Info("State consumer context done, attempting to recreate")
			time.Sleep(RECREATE_STREAM_AND_CONSUMER_DELAY)
			if err := app.setupStreamAndConsumer(); err != nil {
				app.Logger.Error("Failed to recreate stream/consumer", slog.Any("error", err))
				time.Sleep(sleepOnError)
				continue
			}
			app.Logger.Info("Successfully recreated state stream and consumer")
		}
	}
}

func (app *App) Close() {
	if app.StateConsumeCtx != nil {
		app.StateConsumeCtx.Drain()
		<-app.StateConsumeCtx.Closed()
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
