// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"shaper/server/util"
	"strings"
	"time"

	"github.com/duckdb/duckdb-go/v2"
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
	Version                    string
	Sqlite                     *sqlx.DB
	DuckDB                     *sqlx.DB
	DuckDBDSN                  string
	DuckDBExtDir               string
	DuckDBSecretDir            string
	InitSQL                    string
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
	NoEdit                     bool
	NoChromeSandbox            bool
	StateConsumeCtx            jetstream.ConsumeContext
	TaskConsumeCtx             jetstream.ConsumeContext
	TaskResultConsumeCtx       jetstream.ConsumeContext
	JetStream                  jetstream.JetStream
	ConfigKV                   jetstream.KeyValue
	TmpDashboardsKv            jetstream.KeyValue
	DownloadsKv                jetstream.KeyValue
	NATSConn                   *nats.Conn
	StateSubjectPrefix         string
	IngestSubjectPrefix        string
	StateStreamName            string
	StateStreamMaxAge          time.Duration
	StateConsumer              jetstream.Consumer
	ConfigKVBucketName         string
	TmpDashboardsKVBucketName  string
	TmpDashboardsTTL           time.Duration
	DownloadsKVBucketName      string
	DownloadsTTL               time.Duration
	TasksStreamName            string
	TasksSubjectPrefix         string
	TaskQueueConsumerName      string
	TaskResultsStreamName      string
	TaskResultsSubjectPrefix   string
	TaskResultsStreamMaxAge    time.Duration
	TaskBroadcastSubject       string
	TaskBroadcastSubscription  *nats.Subscription
	TaskTimers                 map[string]*time.Timer
	InternalDBName             string
}

func New(
	name string,
	nodeID string,
	version string,
	sqliteDbx *sqlx.DB,
	duckDbx *sqlx.DB,
	duckDBDSN string,
	duckDBExtDir string,
	duckDBSecretDir string,
	initSQL string,
	deprecatedSchema string,
	logger *slog.Logger,
	baseURL string,
	jwtExp time.Duration,
	sessionExp time.Duration,
	inviteExp time.Duration,
	noPublicSharing bool,
	noPasswordProtectedSharing bool,
	noTasks bool,
	noEdit bool,
	noChromeSandbox bool,
	ingestSubjectPrefix string,
	stateSubjectPrefix string,
	stateStreamName string,
	stateStreamMaxAge time.Duration,
	configKVBucketName string,
	tmpDashboardsKVBucketName string,
	tmpDashboardsTTL time.Duration,
	downloadsKVBucketName string,
	downloadsTTL time.Duration,
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

	internalDBName := ""
	if duckDBDSN != ":memory:" {
		var err error
		internalDBName, err = initDuckDB(duckDbx)
		if err != nil {
			return nil, err
		}

		// TODO: Remove this once data is migrated for all active users
		if err := migrateSystemData(sqliteDbx, duckDbx, deprecatedSchema, logger); err != nil {
			return nil, err
		}

		if initSQL != "" {
			logger.Info("Executing init-sql")
			varPrefix, _ := buildVarPrefixWithDBName(internalDBName, nil, nil)

			conn, err := duckDbx.Conn(context.Background())
			if err != nil {
				return nil, fmt.Errorf("failed to get connection for init-sql: %w", err)
			}
			defer conn.Close()

			if err := duckdb.RegisterScalarUDF(conn, "getenv", &util.GetEnvFunc{}); err != nil {
				return nil, fmt.Errorf("failed to register getenv UDF: %w", err)
			}

			if _, err := conn.ExecContext(context.Background(), varPrefix+initSQL); err != nil {
				return nil, fmt.Errorf("failed to execute init-sql: %w", err)
			}
		}
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
	if noEdit {
		logger.Info("Dashboard editing via UI disabled.")
	}
	if noChromeSandbox {
		logger.Info("Chrome sandbox disabled for PDF/PNG generation.")
	}

	app := &App{
		Name:                       name,
		NodeID:                     nodeID,
		Version:                    version,
		Sqlite:                     sqliteDbx,
		DuckDB:                     duckDbx,
		DuckDBDSN:                  duckDBDSN,
		DuckDBExtDir:               duckDBExtDir,
		DuckDBSecretDir:            duckDBSecretDir,
		InitSQL:                    initSQL,
		Logger:                     logger,
		LoginRequired:              loginRequired,
		BasePath:                   baseURL,
		JWTExp:                     jwtExp,
		SessionExp:                 sessionExp,
		InviteExp:                  inviteExp,
		NoPublicSharing:            noPublicSharing,
		NoPasswordProtectedSharing: noPasswordProtectedSharing,
		NoTasks:                    noTasks,
		NoEdit:                     noEdit,
		NoChromeSandbox:            noChromeSandbox,
		IngestSubjectPrefix:        ingestSubjectPrefix,
		StateSubjectPrefix:         stateSubjectPrefix,
		StateStreamName:            stateStreamName,
		StateStreamMaxAge:          stateStreamMaxAge,
		ConfigKVBucketName:         configKVBucketName,
		TmpDashboardsKVBucketName:  tmpDashboardsKVBucketName,
		TmpDashboardsTTL:           tmpDashboardsTTL,
		DownloadsKVBucketName:      downloadsKVBucketName,
		DownloadsTTL:               downloadsTTL,
		TasksStreamName:            tasksStreamName,
		TasksSubjectPrefix:         tasksSubjectPrefix,
		TaskQueueConsumerName:      taskQueueConsumerName,
		TaskResultsStreamName:      taskResultsStreamName,
		TaskResultsSubjectPrefix:   taskResultsSubjectPrefix,
		TaskResultsStreamMaxAge:    taskResultsStreamMaxAge,
		TaskBroadcastSubject:       taskBroadcastSubject,
		TaskTimers:                 make(map[string]*time.Timer),
		InternalDBName:             internalDBName,
	}
	return app, nil
}

func (app *App) GetDuckDB(ctx context.Context) (*sqlx.DB, func(), error) {
	if app.DuckDBDSN != ":memory:" {
		return app.DuckDB, func() {}, nil
	}

	connector, err := duckdb.NewConnector(app.DuckDBDSN, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create DuckDB connector: %w", err)
	}
	db := sql.OpenDB(connector)
	db.SetMaxIdleConns(0)
	dbx := sqlx.NewDb(db, "duckdb")

	cleanup := func() {
		if err := dbx.Close(); err != nil {
			app.Logger.Error("failed to close in-memory DuckDB", slog.Any("error", err))
		}
	}

	// Always disable persistent secrets for in-memory mode to ensure isolation from central secret store
	if _, err := dbx.Exec("SET allow_persistent_secrets = false"); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to disable persistent secrets: %w", err)
	}

	if app.DuckDBExtDir != "" {
		_, err := dbx.Exec("SET extension_directory = ?", app.DuckDBExtDir)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("failed to set DuckDB extension directory: %w", err)
		}
	}

	dbName, err := initDuckDB(dbx)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to initialize DuckDB: %w", err)
	}
	app.InternalDBName = dbName

	if app.InitSQL != "" {
		varPrefix, _ := buildVarPrefix(app, nil, nil)

		conn, err := dbx.Conn(ctx)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("failed to get connection for init-sql: %w", err)
		}
		defer conn.Close()

		if err := duckdb.RegisterScalarUDF(conn, "getenv", &util.GetEnvFunc{}); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("failed to register getenv UDF: %w", err)
		}

		if _, err := conn.ExecContext(ctx, varPrefix+app.InitSQL); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("failed to execute init-sql: %w", err)
		}
	}

	if _, err := dbx.Exec("SET lock_configuration = true"); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to lock configuration: %w", err)
	}

	return dbx, cleanup, nil
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
		app.Logger.Info("Loaded scheduled tasks")
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
	// If NATS has less data than seq in DB we only process new messages in NATS.
	// This likely means we restored a snapshot with empty NATS.
	consumerCfg := jetstream.OrderedConsumerConfig{}
	if startSeq <= stream.CachedInfo().State.LastSeq {
		consumerCfg.DeliverPolicy = jetstream.DeliverByStartSequencePolicy
		consumerCfg.OptStartSeq = startSeq
	} else {
		consumerCfg.DeliverPolicy = jetstream.DeliverNewPolicy
	}
	stateConsumer, err := stream.OrderedConsumer(initCtx, consumerCfg)
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

	tmpDashboardsKV, err := app.JetStream.CreateOrUpdateKeyValue(initCtx, jetstream.KeyValueConfig{
		Bucket: app.TmpDashboardsKVBucketName,
		TTL:    app.TmpDashboardsTTL,
	})
	if err != nil {
		return fmt.Errorf("failed to create or update temporary dashboards KV: %w", err)
	}
	app.TmpDashboardsKv = tmpDashboardsKV

	downloadsKV, err := app.JetStream.CreateOrUpdateKeyValue(initCtx, jetstream.KeyValueConfig{
		Bucket: app.DownloadsKVBucketName,
		TTL:    app.DownloadsTTL,
	})
	if err != nil {
		return fmt.Errorf("failed to create or update downloads KV: %w", err)
	}
	app.DownloadsKv = downloadsKV

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
		consumerCfg := jetstream.OrderedConsumerConfig{}
		if taskResultStatSeq <= taskResultsStream.CachedInfo().State.LastSeq {
			consumerCfg.DeliverPolicy = jetstream.DeliverByStartSequencePolicy
			consumerCfg.OptStartSeq = taskResultStatSeq
		} else {
			consumerCfg.DeliverPolicy = jetstream.DeliverNewPolicy
		}
		taskResultConsumer, err := taskResultsStream.OrderedConsumer(initCtx, consumerCfg)
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

func initDuckDB(duckDbx *sqlx.DB) (string, error) {
	var name string
	if err := duckDbx.Get(&name, "SELECT current_database()"); err != nil {
		return "", err
	}
	// Create custom types
	for _, t := range dbTypes {
		if err := createType(duckDbx, t.Name, t.Definition); err != nil {
			return "", fmt.Errorf("failed to create custom type %s: %w", t.Name, err)
		}
	}
	if err := createBoxlotFunction(duckDbx); err != nil {
		return "", fmt.Errorf("failed to create BOXPLOT function: %w", err)
	}
	return name, nil
}

// buildVarPrefix prepends session state like search_path and variables to SQL queries.
func buildVarPrefix(app *App, singleVars map[string]string, multiVars map[string][]string) (string, string) {
	dbName := ""
	if app != nil {
		dbName = app.InternalDBName
	}
	return buildVarPrefixWithDBName(dbName, singleVars, multiVars)
}

func buildVarPrefixWithDBName(dbName string, singleVars map[string]string, multiVars map[string][]string) (string, string) {
	varPrefix := strings.Builder{}
	varCleanup := strings.Builder{}

	if dbName != "" {
		varPrefix.WriteString(fmt.Sprintf("SET search_path = 'main,\"%s\".main,system';\n", util.EscapeSQLIdentifier(dbName)))
	}

	for k, v := range singleVars {
		varPrefix.WriteString(fmt.Sprintf("SET VARIABLE \"%s\" = %s;\n", util.EscapeSQLIdentifier(k), v))
		varCleanup.WriteString(fmt.Sprintf("RESET VARIABLE \"%s\";\n", util.EscapeSQLIdentifier(k)))
	}
	for k, v := range multiVars {
		l := ""
		for i, p := range v {
			prefix := ", "
			if i == 0 {
				prefix = ""
			}
			l += fmt.Sprintf("%s'%s'", prefix, util.EscapeSQLString(p))
		}
		varPrefix.WriteString(fmt.Sprintf("SET VARIABLE \"%s\" = [%s]::VARCHAR[];\n", util.EscapeSQLIdentifier(k), l))
		varCleanup.WriteString(fmt.Sprintf("RESET VARIABLE \"%s\";\n", util.EscapeSQLIdentifier(k)))
	}
	return varPrefix.String(), varCleanup.String()
}
