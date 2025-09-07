// SPDX-License-Identifier: MPL-2.0

package main

import (
	_ "modernc.org/sqlite"

	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"shaper/server/comms"
	"shaper/server/core"
	"shaper/server/ingest"
	"shaper/server/snapshots"
	"shaper/server/util"
	"shaper/server/util/signals"
	"shaper/server/web"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/marcboeker/go-duckdb/v2"
	"github.com/nrednav/cuid2"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

const APP_NAME = "shaper"

// Embedding frontend files.
// Has to happen in main package because you cannot embed files from a parent directory.
// That's also the main reason why main.go is in the root of the project.
//
//go:embed dist
var frontendFS embed.FS

// Version is set during build time via ldflags
var Version = "dev"

const USAGE = `Version: {{.Version}}

  Shaper is a minimal data platform built on top of DuckDB and NATS to create analytics dashboards and embed them into your software.

  All configuration options can be set via command line flags, environment variables or config file.
  All options are optional.

  Environment variables must be prefixed with SHAPER_ and use uppercase letters and underscores.
  For example, --nats-token turns into SHAPER_NATS_TOKEN.

  The config file format is plain text, with one flag per line. The flag name and value are separated by whitespace.

  For more see: https://taleshape.com/shaper/docs

`

type Config struct {
	DeprecatedSchema           string
	SessionExp                 time.Duration
	InviteExp                  time.Duration
	Address                    string
	DataDir                    string
	ExecutableModTime          time.Time
	BasePath                   string
	CustomCSS                  string
	Favicon                    string
	JWTExp                     time.Duration
	NoPublicSharing            bool
	NoPasswordProtectedSharing bool
	NoTasks                    bool
	NodeIDFile                 string
	TLSDomain                  string
	TLSEmail                   string
	TLSCache                   string
	HTTPSHost                  string
	NatsServers                string
	NatsHost                   string
	NatsPort                   int
	NatsToken                  string
	NatsJSDir                  string
	NatsJSKey                  string
	NatsMaxStore               int64 // in bytes
	StateStreamName            string
	IngestStreamName           string
	ConfigKVBucketName         string
	IngestStreamMaxAge         time.Duration
	StateStreamMaxAge          time.Duration
	IngestConsumerNameFile     string
	IngestSubjectPrefix        string
	StateSubjectPrefix         string
	TasksStreamName            string
	TasksSubjectPrefix         string
	TaskQueueConsumerName      string
	TaskResultsStreamName      string
	TaskResultsSubjectPrefix   string
	TaskResultsStreamMaxAge    time.Duration
	TaskBroadcastSubject       string
	SQLiteDB                   string
	DuckDB                     string
	DuckDBExtDir               string
	InitSQL                    string
	InitSQLFile                string
	SnapshotTime               string
	SnapshotS3Bucket           string
	SnapshotS3Region           string
	SnapshotS3Endpoint         string
	SnapshotS3AccessKey        string
	SnapshotS3SecretKey        string
	SnapshotStream             string
	SnapshotConsumerName       string
	SnapshotSubjectPrefix      string
}

func main() {
	config := loadConfig()
	signals.HandleInterrupt(Run(config))
}

func loadConfig() Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	flags := ff.NewFlagSet(APP_NAME)
	help := flags.Bool('h', "help", "show help")
	version := flags.Bool('v', "version", "show version")
	addr := flags.StringLong("addr", "localhost:5454", "HTTP server address. Not used if --tls-domain is set. In that case, server is automatically listening on the ports 80 and 443.")
	dataDir := flags.String('d', "dir", path.Join(homeDir, ".shaper"), "directory to store data, by default set to /data in docker container)")
	customCSS := flags.StringLong("css", "", "CSS string to inject into the frontend")
	favicon := flags.StringLong("favicon", "", "path to override favicon. Must end .svg or .ico")
	initSQL := flags.StringLong("init-sql", "", "Execute SQL on startup. Supports environment variables in the format $VAR or ${VAR}")
	initSQLFile := flags.StringLong("init-sql-file", "", "Same as init-sql but read SQL from file. Docker by default tries to read /var/lib/shaper/init.sql (default: [--dir]/init.sql)")
	snapshotTime := flags.StringLong("snapshot-time", "01:00", "time to run daily snapshots, format: HH:MM")
	snapshotS3Bucket := flags.StringLong("snapshot-s3-bucket", "", "S3 bucket for snapshots (required for snapshots)")
	snapshotS3Region := flags.StringLong("snapshot-s3-region", "", "AWS region for S3 (optional)")
	snapshotS3Endpoint := flags.StringLong("snapshot-s3-endpoint", "", "S3 endpoint URL (required for snapshots)")
	snapshotS3AccessKey := flags.StringLong("snapshot-s3-access-key", "", "S3 access key (required for snapshots)")
	snapshotS3SecretKey := flags.StringLong("snapshot-s3-secret-key", "", "S3 secret key (required for snapshots)")
	noPublicSharing := flags.BoolLong("no-public-sharing", "Disable public sharing of dashboards")
	noPasswordProtectedSharing := flags.BoolLong("no-password-protected-sharing", "Disable sharing dashboards protected with a password")
	noTasks := flags.BoolLong("no-tasks", "Disable task functionality")
	tlsDomain := flags.StringLong("tls-domain", "", "Domain name for TLS certificate")
	tlsEmail := flags.StringLong("tls-email", "", "Email address for Let's Encrypt registration (optional, used for alerting about certificate expiration)")
	tlsCache := flags.StringLong("tls-cache", "", "Path to Let's Encrypt cache directory (default: [--dir]/letsencrypt-cache)")
	httpsHost := flags.StringLong("https-port", "", "Overwrite https hostname to not listen on all interfaces")
	basePath := flags.StringLong("basepath", "/", "Base URL path the frontend is served from. Override if you are using a reverse proxy and serve the frontend from a subpath.")
	natsHost := flags.StringLong("nats-host", "0.0.0.0", "NATS server host")
	natsPort := flags.Int('p', "nats-port", 0, "NATS server port. If not specified, NATS will not listen on any port.")
	natsToken := flags.String('t', "nats-token", "", "NATS authentication token")
	natsServers := flags.StringLong("nats-servers", "", "Use external NATS servers, specify as comma separated list")
	natsMaxStore := flags.StringLong("nats-max-store", "0", "Maximum storage in bytes, set to 0 for unlimited")
	natsJSKey := flags.StringLong("nats-js-key", "", "JetStream encryption key")
	natsJSDir := flags.StringLong("nats-dir", "", "Override JetStream storage directory (default: [--dir]/nats)")
	sqliteDB := flags.StringLong("sqlite", "", "Override sqlite DB file that is used for system state (default: [--dir]/shaper_internal.sqlite)")
	duckdb := flags.StringLong("duckdb", "", "Override duckdb DSN (default: [--dir]/shaper.duckdb)")
	duckdbExtDir := flags.StringLong("duckdb-ext-dir", "", "Override DuckDB extension directory, by default set to /data/duckdb_extensions in docker (default: ~/.duckdb/extensions/)")
	deprecatedSchema := flags.StringLong("schema", "_shaper", "DEPRECATED: Was used for system state in DuckDB, not used in Sqlite after data is migrated")
	jwtExp := flags.DurationLong("jwtexp", 15*time.Minute, "JWT expiration duration")
	sessionExp := flags.DurationLong("sessionexp", 30*24*time.Hour, "Session expiration duration")
	inviteExp := flags.DurationLong("inviteexp", 7*24*time.Hour, "Invite expiration duration")
	streamPrefix := flags.StringLong("stream-prefix", "", "Prefix for NATS stream and KV bucket names. Must be a valid NATS subject name")
	nodeIDFile := flags.StringLong("node-id-file", "", "File to store and lookup node ID (default: [--dir]/node-id.txt)")
	ingestStream := flags.StringLong("ingest-stream", "shaper-ingest", "NATS stream name for ingest messages")
	stateStream := flags.StringLong("state-stream", "shaper-state", "NATS stream name for state messages")
	configKVBucket := flags.StringLong("config-kv-bucket", "shaper-config", "Name for NATS config KV bucket")
	tasksStream := flags.StringLong("tasks-stream", "shaper-tasks", "NATS stream name for scheduled task execution")
	taskResultsStream := flags.StringLong("task-results-stream", "shaper-task-results", "NATS stream name for task results")
	ingestStreamMaxAge := flags.DurationLong("ingest-max-age", 0, "Maximum age of messages in the ingest stream. Set to 0 for indefinite retention")
	stateStreamMaxAge := flags.DurationLong("state-max-age", 0, "Maximum age of messages in the state stream. Set to 0 for indefinite retention")
	taskResultsStreamMaxAge := flags.DurationLong("task-results-max-age", 0, "Maximum age of messages in the task-results stream. Set to 0 for indefinite retention")
	ingestConsumerNameFile := flags.StringLong("ingest-consumer-name-file", "", "File to store and lookup name for ingest consumer (default: [--dir]/ingest-consumer-name.txt)")
	_ = flags.StringLong("state-consumer-name-file", "", "DEPRECATED: Using ephermal consumer and storing sequence in sqlite now")
	taskQueueConsumerName := flags.StringLong("task-queue-consumer-name", "shaper-task-queue-consumer", "Name for the task queue consumer")
	_ = flags.StringLong("task-result-consumer-name-file", "", "DEPRECATED: Now storing cursor in sqlite")
	snapshotStream := flags.StringLong("snapshot-stream", "shaper-snapshots", "NATS stream name for scheduled snapshots")
	snapshotConsumerName := flags.StringLong("snapshot-consumer-name", "shaper-snapshot-consumer", "Name for the snapshot consumer")
	subjectPrefix := flags.StringLong("subject-prefix", "", "prefix for NATS subjects. Must be a valid NATS subject name. Should probably end with a dot.")
	ingestSubjectPrefix := flags.StringLong("ingest-subject-prefix", "shaper.ingest.", "prefix for ingest NATS subjects")
	stateSubjectPrefix := flags.StringLong("state-subject-prefix", "shaper.state.", "prefix for state NATS subjects")
	tasksSubjectPrefix := flags.StringLong("tasks-subject-prefix", "shaper.tasks.", "prefix for tasks NATS subjects")
	taskResultsSubjectPrefix := flags.StringLong("task-results-subject-prefix", "shaper.task-results.", "prefix for task-results NATS subjects")
	taskBroadcastSubject := flags.StringLong("task-broadcast-subject", "shaper.task-broadcast", "subject to broadcast tasks to run on all nodes in a cluster when running manual task")
	snapshotSubjectPrefix := flags.StringLong("snapshots-subject-prefix", "shaper.snapshots.", "prefix for snapshots NATS subjects")
	flags.StringLong("config-file", "", "path to config file")

	err = ff.Parse(flags, os.Args[1:],
		ff.WithEnvVarPrefix("SHAPER"),
		ff.WithConfigFileFlag("config-file"),
		ff.WithConfigFileParser(ff.PlainParser),
	)
	if err != nil {
		fmt.Printf("Error parsing config: %v\n\nSee --help for config options\n\n", err)
		os.Exit(1)
	}
	if *help {
		usage := strings.Replace(USAGE, "{{.Version}}", Version, 1)
		fmt.Printf("%s\n", ffhelp.Flags(flags, usage))
		os.Exit(0)
	}
	if *version {
		fmt.Printf("%s version %s\n", APP_NAME, Version)
		os.Exit(0)
	}

	executableModTime, err := getExecutableModTime()
	if err != nil {
		fmt.Printf("Error getting executable modification time: %v\n", err)
		os.Exit(1)
	}

	if *tlsDomain != "" {
		if *addr != "localhost:5454" && *addr != ":5454" {
			fmt.Println("Cannot set addr and tls-domain at the same time.")
			os.Exit(1)
		}
		if *basePath != "/" {
			fmt.Println("Cannot set basepath and tls-domain at the same time.")
			os.Exit(1)
		}
	}

	tlsCacheDir := *tlsCache
	if tlsCacheDir == "" {
		tlsCacheDir = path.Join(*dataDir, "letsencrypt-cache")
	}

	// Parse natsMaxStore as int64
	maxStore, err := strconv.ParseInt(*natsMaxStore, 10, 64)
	if err != nil {
		fmt.Printf("Invalid value for nats-max-store: %v\n", err)
		os.Exit(1)
	}

	if *natsServers != "" {
		if *natsJSDir != "" || *natsJSKey != "" || maxStore > 0 {
			fmt.Println("when connecting to external NATS servers (nats-servers specified), nats-js-key, nats-dir and nats-max-store must not be specified")
			os.Exit(1)
		}
	}

	natsDir := path.Join(*dataDir, "nats")
	if *natsJSDir != "" {
		natsDir = *natsJSDir
	}

	bpath := *basePath
	if bpath == "" {
		bpath = "/"
	}
	if bpath[0] != '/' {
		bpath = "/" + bpath
	}
	if bpath[len(bpath)-1] != '/' {
		bpath += "/"
	}

	initSQLFilePath := path.Join(*dataDir, "init.sql")
	if *initSQLFile != "" {
		initSQLFilePath = *initSQLFile
	}

	config := Config{
		DeprecatedSchema:           *deprecatedSchema,
		Address:                    *addr,
		DataDir:                    *dataDir,
		ExecutableModTime:          executableModTime,
		BasePath:                   bpath,
		CustomCSS:                  *customCSS,
		Favicon:                    *favicon,
		JWTExp:                     *jwtExp,
		SessionExp:                 *sessionExp,
		InviteExp:                  *inviteExp,
		NoPublicSharing:            *noPublicSharing,
		NoPasswordProtectedSharing: *noPasswordProtectedSharing,
		NoTasks:                    *noTasks,
		NodeIDFile:                 *nodeIDFile,
		TLSDomain:                  *tlsDomain,
		TLSEmail:                   *tlsEmail,
		TLSCache:                   tlsCacheDir,
		HTTPSHost:                  *httpsHost,
		NatsServers:                *natsServers,
		NatsHost:                   *natsHost,
		NatsPort:                   *natsPort,
		NatsToken:                  *natsToken,
		NatsJSDir:                  natsDir,
		NatsJSKey:                  *natsJSKey,
		NatsMaxStore:               maxStore,
		StateStreamName:            *streamPrefix + *stateStream,
		IngestStreamName:           *streamPrefix + *ingestStream,
		ConfigKVBucketName:         *streamPrefix + *configKVBucket,
		IngestStreamMaxAge:         *ingestStreamMaxAge,
		StateStreamMaxAge:          *stateStreamMaxAge,
		IngestConsumerNameFile:     *ingestConsumerNameFile,
		IngestSubjectPrefix:        *subjectPrefix + *ingestSubjectPrefix,
		StateSubjectPrefix:         *subjectPrefix + *stateSubjectPrefix,
		TasksStreamName:            *streamPrefix + *tasksStream,
		TasksSubjectPrefix:         *subjectPrefix + *tasksSubjectPrefix,
		TaskQueueConsumerName:      *taskQueueConsumerName,
		TaskResultsStreamName:      *streamPrefix + *taskResultsStream,
		TaskResultsSubjectPrefix:   *subjectPrefix + *taskResultsSubjectPrefix,
		TaskResultsStreamMaxAge:    *taskResultsStreamMaxAge,
		TaskBroadcastSubject:       *subjectPrefix + *taskBroadcastSubject,
		SQLiteDB:                   *sqliteDB,
		DuckDB:                     *duckdb,
		DuckDBExtDir:               *duckdbExtDir,
		InitSQL:                    *initSQL,
		InitSQLFile:                initSQLFilePath,
		SnapshotTime:               *snapshotTime,
		SnapshotS3Bucket:           *snapshotS3Bucket,
		SnapshotS3Region:           *snapshotS3Region,
		SnapshotS3Endpoint:         *snapshotS3Endpoint,
		SnapshotS3AccessKey:        *snapshotS3AccessKey,
		SnapshotS3SecretKey:        *snapshotS3SecretKey,
		SnapshotStream:             *streamPrefix + *snapshotStream,
		SnapshotConsumerName:       *snapshotConsumerName,
		SnapshotSubjectPrefix:      *subjectPrefix + *snapshotSubjectPrefix,
	}
	return config
}

func Run(cfg Config) func(context.Context) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	logger.Info("Starting Shaper", slog.String("version", Version))
	logger.Info("For configuration options see --help or visit https://taleshape.com/shaper/docs for more")

	if cfg.Favicon != "" {
		logger.Info("Custom favicon: " + cfg.Favicon)
	}
	if cfg.CustomCSS != "" {
		logger.Info("Custom CSS injected into frontend")
	}

	// Make sure data directory exists
	if _, err := os.Stat(cfg.DataDir); os.IsNotExist(err) {
		err := os.Mkdir(cfg.DataDir, 0755)
		if err != nil {
			logger.Error("Failed to create data directory", slog.String("path", cfg.DataDir), slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("Created data directory", slog.Any("path", cfg.DataDir))
	}

	// connect to SQLite
	sqliteDBxFile := cfg.SQLiteDB
	if cfg.SQLiteDB == "" {
		sqliteDBxFile = path.Join(cfg.DataDir, "shaper_internal.sqlite")
	}
	sqliteDbx, err := sqlx.Connect("sqlite", sqliteDBxFile)
	if err != nil {
		logger.Error("Failed to connect to SQLite", slog.String("file", sqliteDBxFile), slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("SQLite opened", slog.Any("file", sqliteDBxFile))

	duckDBFile := cfg.DuckDB
	if cfg.DuckDB == "" {
		duckDBFile = path.Join(cfg.DataDir, "shaper.duckdb")
	}

	// connect to duckdb
	duckDBConnector, err := duckdb.NewConnector(duckDBFile, nil)
	if err != nil {
		logger.Error("Failed to create DuckDB connector", slog.String("file", duckDBFile), slog.Any("error", err))
		os.Exit(1)
	}
	duckDbSqlDb := sql.OpenDB(duckDBConnector)
	// This is important to avoid leaking variables or temp tables/views. Must not reuse connections.
	duckDbSqlDb.SetMaxIdleConns(0)
	duckdbSqlxDb := sqlx.NewDb(duckDbSqlDb, "duckdb")
	logger.Info("DuckDB opened", slog.Any("file", duckDBFile))

	if cfg.DuckDBExtDir != "" {
		_, err := duckdbSqlxDb.Exec("SET extension_directory = ?", cfg.DuckDBExtDir)
		if err != nil {
			logger.Error("Failed to set DuckDB extension directory", slog.String("path", cfg.DuckDBExtDir), slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("Set DuckDB extension directory", slog.Any("path", cfg.DuckDBExtDir))
	}

	if cfg.InitSQL != "" {
		logger.Info("Executing init-sql")
		// Substitute environment variables in the SQL
		sql := os.ExpandEnv(strings.TrimSpace(util.StripSQLComments(cfg.InitSQL)))
		if sql == "" {
			logger.Info("init-sql specified but empty, skipping")
		} else {
			_, err := duckdbSqlxDb.Exec(sql)
			if err != nil {
				logger.Error("Failed to execute init-sql", slog.String("sql", sql), slog.Any("error", err))
				os.Exit(1)
			}
		}
	}
	if cfg.InitSQLFile != "" {
		logger.Info("Loading init-sql-file", slog.Any("path", cfg.InitSQLFile))
		data, err := os.ReadFile(cfg.InitSQLFile)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				logger.Info("init-sql-file does not exist, skipping", slog.Any("path", cfg.InitSQLFile))
			} else {
				logger.Error("Failed to read init-sql-file", slog.String("path", cfg.InitSQLFile), slog.Any("error", err))
				os.Exit(1)
			}
		} else {
			sql := os.ExpandEnv(strings.TrimSpace(util.StripSQLComments(string(data))))
			if len(sql) == 0 {
				logger.Info("init-sql-file is empty, skipping", slog.Any("path", cfg.InitSQLFile))
			} else {
				logger.Info("Executing init-sql-file")
				// Substitute environment variables in the SQL file content
				_, err = duckdbSqlxDb.Exec(sql)
				if err != nil {
					logger.Error("Failed to execute init-sql-file", slog.String("path", cfg.InitSQLFile), slog.Any("error", err))
					os.Exit(1)
				}
			}
		}
	}

	nodeID := getOrGenerateNodeID(cfg.DataDir, cfg.NodeIDFile, "node-id.txt")
	ingestConsumerName := getOrGenerateConsumerName(cfg.DataDir, cfg.IngestConsumerNameFile, "ingest-consumer-name.txt", "shaper-ingest-consumer-", nodeID)

	app, err := core.New(
		APP_NAME,
		nodeID,
		sqliteDbx,
		duckdbSqlxDb,
		cfg.DeprecatedSchema,
		logger,
		cfg.BasePath,
		cfg.JWTExp,
		cfg.SessionExp,
		cfg.InviteExp,
		cfg.NoPublicSharing,
		cfg.NoPasswordProtectedSharing,
		cfg.NoTasks,
		cfg.IngestSubjectPrefix,
		cfg.StateSubjectPrefix,
		cfg.StateStreamName,
		cfg.StateStreamMaxAge,
		cfg.ConfigKVBucketName,
		cfg.TasksStreamName,
		cfg.TasksSubjectPrefix,
		cfg.TaskQueueConsumerName,
		cfg.TaskResultsStreamName,
		cfg.TaskResultsSubjectPrefix,
		cfg.TaskResultsStreamMaxAge,
		cfg.TaskBroadcastSubject,
	)
	if err != nil {
		logger.Error("Failed to create application core", slog.Any("error", err))
		os.Exit(1)
	}

	// TODO: refactor - comms should be part of core
	c, err := comms.New(comms.Config{
		Logger:              logger.WithGroup("nats"),
		Servers:             cfg.NatsServers,
		Host:                cfg.NatsHost,
		Port:                cfg.NatsPort,
		Token:               cfg.NatsToken,
		JSDir:               cfg.NatsJSDir,
		JSKey:               cfg.NatsJSKey,
		MaxStore:            cfg.NatsMaxStore,
		Sqlite:              sqliteDbx,
		IngestSubjectPrefix: cfg.IngestSubjectPrefix,
	})
	if err != nil {
		logger.Error("Failed to create NATS communication layer", slog.Any("error", err))
		os.Exit(1)
	}

	ingestConsumer, err := ingest.Start(
		cfg.IngestSubjectPrefix,
		duckDBConnector,
		duckdbSqlxDb,
		logger.WithGroup("ingest"),
		c.Conn,
		cfg.IngestStreamName,
		cfg.IngestStreamMaxAge,
		ingestConsumerName,
	)
	if err != nil {
		logger.Error("Failed to start ingest consumer", slog.Any("error", err))
		os.Exit(1)
	}

	err = app.Init(c.Conn)
	if err != nil {
		logger.Error("Failed to initialize application", slog.Any("error", err))
		os.Exit(1)
	}

	s := snapshots.Start(snapshots.Config{
		Logger:        logger.WithGroup("snapshots"),
		Sqlite:        sqliteDbx,
		DuckDB:        duckdbSqlxDb,
		Nats:          c.Conn,
		S3Bucket:      cfg.SnapshotS3Bucket,
		S3Region:      cfg.SnapshotS3Region,
		S3Endpoint:    cfg.SnapshotS3Endpoint,
		S3AccessKey:   cfg.SnapshotS3AccessKey,
		S3SecretKey:   cfg.SnapshotS3SecretKey,
		Stream:        cfg.SnapshotStream,
		ConsumerName:  cfg.SnapshotConsumerName,
		SubjectPrefix: cfg.SnapshotSubjectPrefix,
		ScheduledTime: cfg.SnapshotTime,
	})

	e := web.Start(
		cfg.Address,
		app,
		frontendFS,
		cfg.ExecutableModTime,
		cfg.CustomCSS,
		cfg.Favicon,
		cfg.TLSDomain,
		cfg.TLSEmail,
		cfg.TLSCache,
		cfg.HTTPSHost,
	)

	return func(ctx context.Context) {
		logger.Info("Initiating shutdown...")
		s.Stop()
		logger.Info("Stopping web server...")
		if err := e.Shutdown(ctx); err != nil {
			logger.ErrorContext(ctx, "Error stopping server", slog.Any("error", err))
		}
		logger.Info("Stopping NATS...")
		ingestConsumer.Close()
		c.Close()
		logger.Info("Closing DB connections...")
		if err := duckdbSqlxDb.Close(); err != nil {
			logger.ErrorContext(ctx, "Error closing database connection", slog.Any("error", err))
		}
	}
}

func getExecutableModTime() (time.Time, error) {
	ex, err := os.Executable()
	if err != nil {
		return time.Time{}, err
	}
	stat, err := os.Stat(ex)
	return stat.ModTime(), err
}

// Node ID is a CUID2 and it's stored in the given file.
// Binding the Node ID to the local file system means it resets when the file system is reset.
// This works well together with Docker containers.
func getOrGenerateNodeID(dataDir, nameFile, defaultFileName string) string {
	fileName := nameFile
	if fileName == "" {
		fileName = path.Join(dataDir, defaultFileName)
	}
	name := ""
	if _, err := os.Stat(fileName); err == nil {
		content, err := os.ReadFile(fileName)
		if err != nil {
			fmt.Printf("Failed to read node ID file %s: %v\n", fileName, err)
			os.Exit(1)
		}
		name = strings.TrimSpace(string(content))
	} else {
		name = cuid2.Generate()
		err := os.WriteFile(fileName, []byte(name), 0644)
		if err != nil {
			fmt.Printf("Failed to write node ID file %s: %v\n", fileName, err)
			os.Exit(1)
		}
	}
	return name
}

// Consumer name defaults to the Node ID with a prefix.
// Consumer names can also be read from a file to set them explicitly. This is for backwards compatibility from before the concept of Node IDs was introduced.
// Binding consumer names to the local file system means they reset when the file system is reset.
// This works well together with Docker containers.
func getOrGenerateConsumerName(dataDir, nameFile, defaultFileName, prefix, nodeID string) string {
	fileName := nameFile
	if fileName == "" {
		fileName = path.Join(dataDir, defaultFileName)
	}
	name := ""
	if _, err := os.Stat(fileName); err == nil {
		content, err := os.ReadFile(fileName)
		if err != nil {
			fmt.Printf("Failed to read consumer name file %s: %v\n", fileName, err)
			os.Exit(1)
		}
		name = strings.TrimSpace(string(content))
	} else {
		name = prefix + nodeID
	}
	return name
}
