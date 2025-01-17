// TODO: JWT https://echo.labstack.com/docs/middleware/jwt
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	NATS_SUBJECT_PREFIX   = "shaper.state."
	NATS_KV_CONFIG_BUCKET = "shaper-config"
	CONFIG_KEY_JWT_SECRET = "jwt-secret"
)

type App struct {
	db              *sqlx.DB
	Logger          *slog.Logger
	LoginToken      string
	Schema          string
	JWTSecret       []byte
	JWTExp          time.Duration
	StateConsumeCtx jetstream.ConsumeContext
	JetStream       jetstream.JetStream
	ConfigKV        jetstream.KeyValue
	NATSConn        *nats.Conn
}

func New(
	db *sqlx.DB,
	nc *nats.Conn,
	logger *slog.Logger,
	loginToken string,
	schema string,
	jwtExp time.Duration,
	persist bool,
) (*App, error) {
	if err := initDB(db, schema); err != nil {
		return nil, err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}
	initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer initCancel()
	storageType := jetstream.MemoryStorage
	if persist {
		storageType = jetstream.FileStorage
	}
	stream, err := js.CreateOrUpdateStream(initCtx, jetstream.StreamConfig{
		Name:     "shaper-state",
		Subjects: []string{NATS_SUBJECT_PREFIX + ">"},
		Storage:  storageType,
	})
	if err != nil {
		return nil, err
	}
	stateConsumer, err := stream.CreateOrUpdateConsumer(initCtx, jetstream.ConsumerConfig{
		Durable:       "shaper-state",
		MaxAckPending: 1,
	})
	if err != nil {
		return nil, err
	}
	configKV, err := js.CreateOrUpdateKeyValue(initCtx, jetstream.KeyValueConfig{
		Bucket:  NATS_KV_CONFIG_BUCKET,
		Storage: storageType,
	})
	if err != nil {
		return nil, err
	}
	app := &App{
		db:         db,
		Logger:     logger,
		LoginToken: loginToken,
		Schema:     schema,
		JWTExp:     jwtExp,
		NATSConn:   nc,
		JetStream:  js,
		ConfigKV:   configKV,
	}

	stateConsumeCtx, err := stateConsumer.Consume(app.HandleState)
	if err != nil {
		return nil, err
	}
	app.StateConsumeCtx = stateConsumeCtx

	loadJWTSecret(app)

	return app, nil
}

func (app *App) Close() {
	if app.StateConsumeCtx != nil {
		app.StateConsumeCtx.Drain()
		<-app.StateConsumeCtx.Closed()
	}
}

func (app *App) HandleState(msg jetstream.Msg) {
	event := strings.TrimPrefix(msg.Subject(), NATS_SUBJECT_PREFIX)
	data := msg.Data()
	handler := func(app *App, data []byte) bool {
		app.Logger.Error("Unknown state message subject", slog.String("event", event))
		return false
	}
	switch event {
	case "create_dashboard":
		handler = HandleCreateDashboard
	case "update_dashboard_content":
		handler = HandleUpdateDashboardContent
	case "update_dashboard_name":
		handler = HandleUpdateDashboardName
	case "delete_dashboard":
		handler = HandleDeleteDashboard
	case "create_api_key":
		handler = HandleCreateAPIKey
	case "delete_api_key":
		handler = HandleDeleteAPIKey
	}
	app.Logger.Info("Handling shaper state change", slog.String("event", event))
	ok := handler(app, data)
	if ok {
		err := msg.Ack()
		if err != nil {
			app.Logger.Error("Error acking message", slog.Any("error", err))
		}
	}
}

func (app *App) SubmitState(ctx context.Context, action string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	// We listen on the ACK subject for the consumer to know when the message has been processed
	// We need to subscribe before publishing the message to avoid missing the ACK
	sub, err := app.NATSConn.SubscribeSync("$JS.ACK.shaper-state.shaper-state.>")
	if err != nil {
		return err
	}
	ack, err := app.JetStream.Publish(ctx, NATS_SUBJECT_PREFIX+action, payload)
	if err != nil {
		return err
	}
	ackSeq := strconv.FormatUint(ack.Sequence, 10)
	// Wait for the ACK
	// If context is cancelled, we return an error
	for {
		msg, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			return err
		}
		// The sequence number is the part of the subject after the container of how many deliveries have been made
		// We trust the shape of the subject to be correct and panic otherwise
		seq := strings.Split(strings.TrimPrefix(msg.Subject, "$JS.ACK.shaper-state.shaper-state."), ".")[1]
		if seq == ackSeq {
			return nil
		}
	}
}

func initDB(db *sqlx.DB, schema string) error {
	// Create schema if not exists
	if _, err := db.Exec("CREATE SCHEMA IF NOT EXISTS " + schema); err != nil {
		return fmt.Errorf("error creating schema: %w", err)
	}

	// Create dashboards table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + schema + `.dashboards (
			id VARCHAR PRIMARY KEY,
			path VARCHAR NOT NULL,
			name VARCHAR NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_by VARCHAR,
			updated_by VARCHAR
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating dashboards table: %w", err)
	}

	// Create api_keys table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + schema + `.api_keys (
			id VARCHAR PRIMARY KEY,
			hash VARCHAR NOT NULL,
			name VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_by VARCHAR,
			updated_by VARCHAR
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating config table: %w", err)
	}

	// Create custom types
	for _, t := range dbTypes {
		if err := createType(db, t.Name, t.Definition); err != nil {
			return err
		}
	}
	return nil
}
