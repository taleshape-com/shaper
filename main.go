// SPDX-License-Identifier: MPL-2.0

package main

import (
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
	SessionExp             time.Duration
	InviteExp              time.Duration
	Address                string
	DataDir                string
	Schema                 string
	ExecutableModTime      time.Time
	BasePath               string
	CustomCSS              string
	Favicon                string
	JWTExp                 time.Duration
	NoPublicSharing        bool
	NatsServers            string
	NatsHost               string
	NatsPort               int
	NatsToken              string
	NatsJSDir              string
	NatsJSKey              string
	NatsMaxStore           int64 // in bytes
	StateStreamName        string
	IngestStreamName       string
	ConfigKVBucketName     string
	IngestStreamMaxAge     time.Duration
	StateStreamMaxAge      time.Duration
	IngestConsumerNameFile string
	StateConsumerNameFile  string
	IngestSubjectPrefix    string
	StateSubjectPrefix     string
	DuckDB                 string
	DuckDBExtDir           string
	InitSQL                string
	InitSQLFile            string
}

func main() {
	config := loadConfig()
	signals.HandleInterrupt(Run(config))
}

func loadConfig() Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	flags := ff.NewFlagSet(APP_NAME)
	help := flags.Bool('h', "help", "show help")
	version := flags.Bool('v', "version", "show version")
	addr := flags.StringLong("addr", "localhost:5454", "server address")
	dataDir := flags.String('d', "dir", path.Join(homeDir, ".shaper"), "directory to store data, by default set to /data in docker container)")
	customCSS := flags.StringLong("css", "", "CSS string to inject into the frontend")
	favicon := flags.StringLong("favicon", "", "path to override favicon. Must end .svg or .ico")
	initSQL := flags.StringLong("init-sql", "", "Execute SQL on startup. Supports environment variables in the format $VAR or ${VAR}")
	initSQLFile := flags.StringLong("init-sql-file", "", "Same as init-sql but read SQL from file. Docker by default tries to read /var/lib/shaper/init.sql (default: [--dir]/init.sql)")
	noPublicSharing := flags.BoolLong("no-public-sharing", "Disable public sharing of dashboards")
	basePath := flags.StringLong("basepath", "/", "Base URL path the frontend is served from. Override if you are using a reverse proxy and serve the frontend from a subpath.")
	natsHost := flags.StringLong("nats-host", "0.0.0.0", "NATS server host")
	natsPort := flags.Int('p', "nats-port", 0, "NATS server port. If not specified, NATS will not listen on any port.")
	natsToken := flags.String('t', "nats-token", "", "NATS authentication token")
	natsServers := flags.StringLong("nats-servers", "", "Use external NATS servers, specify as comma separated list")
	natsMaxStore := flags.StringLong("nats-max-store", "0", "Maximum storage in bytes, set to 0 for unlimited")
	natsJSKey := flags.StringLong("nats-js-key", "", "JetStream encryption key")
	natsJSDir := flags.StringLong("nats-dir", "", "Override JetStream storage directory (default: [--dir]/nats)")
	duckdb := flags.StringLong("duckdb", "", "Override duckdb DSN (default: [--dir]/shaper.duckdb)")
	duckdbExtDir := flags.StringLong("duckdb-ext-dir", "", "Override DuckDB extension directory, by default set to /data/duckdb_extensions in docker (default: ~/.duckdb/extensions/)")
	schema := flags.StringLong("schema", "_shaper", "DB schema name for internal tables")
	jwtExp := flags.DurationLong("jwtexp", 15*time.Minute, "JWT expiration duration")
	sessionExp := flags.DurationLong("sessionexp", 30*24*time.Hour, "Session expiration duration")
	inviteExp := flags.DurationLong("inviteexp", 7*24*time.Hour, "Invite expiration duration")
	streamPrefix := flags.StringLong("stream-prefix", "", "Prefix for NATS stream and KV bucket names. Must be a valid NATS subject name")
	ingestStream := flags.StringLong("ingest-stream", "shaper-ingest", "NATS stream name for ingest messages")
	stateStream := flags.StringLong("state-stream", "shaper-state", "NATS stream name for state messages")
	configKVBucket := flags.StringLong("config-kv-bucket", "shaper-config", "Name for NATS config KV bucket")
	ingestStreamMaxAge := flags.DurationLong("ingest-max-age", 0, "Maximum age of messages in the ingest stream. Set to 0 for indefinite retention")
	stateStreamMaxAge := flags.DurationLong("state-max-age", 0, "Maximum age of messages in the state stream. Set to 0 for indefinite retention")
	ingestConsumerNameFile := flags.StringLong("ingest-consumer-name-file", "", "File to store and lookup name for ingest consumer (default: [--dir]/ingest-consumer-name.txt)")
	stateConsumerNameFile := flags.StringLong("state-consumer-name-file", "", "File to store and lookup name for state consumer (default: [--dir]/state-consumer-name.txt)")
	subjectPrefix := flags.StringLong("subject-prefix", "", "prefix for NATS subjects. Must be a valid NATS subject name. Should probably end with a dot.")
	ingestSubjectPrefix := flags.StringLong("ingest-subject-prefix", "shaper.ingest.", "prefix for ingest NATS subjects")
	stateSubjectPrefix := flags.StringLong("state-subject-prefix", "shaper.state.", "prefix for state NATS subjects")
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
		panic(err)
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
		Address:                *addr,
		DataDir:                *dataDir,
		Schema:                 *schema,
		ExecutableModTime:      executableModTime,
		BasePath:               bpath,
		CustomCSS:              *customCSS,
		Favicon:                *favicon,
		JWTExp:                 *jwtExp,
		SessionExp:             *sessionExp,
		InviteExp:              *inviteExp,
		NoPublicSharing:        *noPublicSharing,
		NatsServers:            *natsServers,
		NatsHost:               *natsHost,
		NatsPort:               *natsPort,
		NatsToken:              *natsToken,
		NatsJSDir:              natsDir,
		NatsJSKey:              *natsJSKey,
		NatsMaxStore:           maxStore,
		StateStreamName:        *streamPrefix + *stateStream,
		IngestStreamName:       *streamPrefix + *ingestStream,
		ConfigKVBucketName:     *streamPrefix + *configKVBucket,
		IngestStreamMaxAge:     *ingestStreamMaxAge,
		StateStreamMaxAge:      *stateStreamMaxAge,
		IngestConsumerNameFile: *ingestConsumerNameFile,
		StateConsumerNameFile:  *stateConsumerNameFile,
		IngestSubjectPrefix:    *subjectPrefix + *ingestSubjectPrefix,
		StateSubjectPrefix:     *subjectPrefix + *stateSubjectPrefix,
		DuckDB:                 *duckdb,
		DuckDBExtDir:           *duckdbExtDir,
		InitSQL:                *initSQL,
		InitSQLFile:            initSQLFilePath,
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
		logger.Info("Created data directory", slog.Any("path", cfg.DataDir))
		if err != nil {
			panic(err)
		}
	}

	dbFile := cfg.DuckDB
	if cfg.DuckDB == "" {
		dbFile = path.Join(cfg.DataDir, "shaper.duckdb")
	}

	// connect to duckdb
	dbConnector, err := duckdb.NewConnector(dbFile, nil)
	if err != nil {
		panic(err)
	}
	sqlDB := sql.OpenDB(dbConnector)
	// This is important to avoid leaking variables or temp tables/views. Must not reuse connections.
	sqlDB.SetMaxIdleConns(0)
	db := sqlx.NewDb(sqlDB, "duckdb")
	logger.Info("DuckDB opened", slog.Any("file", dbFile))

	if cfg.DuckDBExtDir != "" {
		_, err := db.Exec("SET extension_directory = ?", cfg.DuckDBExtDir)
		if err != nil {
			panic(errors.New("failed to set extension directory: " + err.Error()))
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
			_, err := db.Exec(sql)
			if err != nil {
				panic(errors.New("failed to execute init-sql: " + err.Error()))
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
				panic(errors.New("failed to read init-sql-file: " + err.Error()))
			}
		} else {
			sql := os.ExpandEnv(strings.TrimSpace(util.StripSQLComments(string(data))))
			if len(sql) == 0 {
				logger.Info("init-sql-file is empty, skipping", slog.Any("path", cfg.InitSQLFile))
			} else {
				logger.Info("Executing init-sql-file")
				// Substitute environment variables in the SQL file content
				_, err = db.Exec(sql)
				if err != nil {
					panic(errors.New("failed to execute init-sql-file: " + err.Error()))
				}
			}
		}
	}

	// Get or generate consumer names
	ingestConsumerName := getOrGenerateConsumerName(cfg.DataDir, cfg.IngestConsumerNameFile, "ingest-consumer-name.txt", "shaper-ingest-consumer-")
	stateConsumerName := getOrGenerateConsumerName(cfg.DataDir, cfg.StateConsumerNameFile, "state-consumer-name.txt", "shaper-state-consumer-")

	app, err := core.New(
		APP_NAME,
		db,
		logger,
		cfg.BasePath,
		cfg.Schema,
		cfg.JWTExp,
		cfg.SessionExp,
		cfg.InviteExp,
		cfg.NoPublicSharing,
		cfg.IngestSubjectPrefix,
		cfg.StateSubjectPrefix,
		cfg.StateStreamName,
		cfg.StateStreamMaxAge,
		stateConsumerName,
		cfg.ConfigKVBucketName,
	)
	if err != nil {
		panic(err)
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
		DB:                  db,
		Schema:              cfg.Schema,
		IngestSubjectPrefix: cfg.IngestSubjectPrefix,
	})
	if err != nil {
		panic(err)
	}

	ingestConsumer, err := ingest.Start(
		cfg.IngestSubjectPrefix,
		dbConnector,
		db,
		logger.WithGroup("ingest"),
		c.Conn,
		cfg.IngestStreamName,
		cfg.IngestStreamMaxAge,
		ingestConsumerName,
	)
	if err != nil {
		panic(err)
	}

	err = app.Init(c.Conn)
	if err != nil {
		panic(err)
	}

	e := web.Start(cfg.Address, app, frontendFS, cfg.ExecutableModTime, cfg.CustomCSS, cfg.Favicon)

	return func(ctx context.Context) {
		logger.Info("Initiating shutdown...")
		logger.Info("Stopping web server...")
		if err := e.Shutdown(ctx); err != nil {
			logger.ErrorContext(ctx, "Error stopping server", slog.Any("error", err))
		}
		logger.Info("Stopping NATS...")
		ingestConsumer.Close()
		c.Close()
		logger.Info("Closing DB connections...")
		if err := db.Close(); err != nil {
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

// Consumer name is a CUID2 with a prefix and it's stored in the given file
// Binding consumer names to the local file system means they reset when the file system is reset.
// This works well together with Docker containers.
func getOrGenerateConsumerName(dataDir, nameFile, defaultFileName string, prefix string) string {
	fileName := nameFile
	if fileName == "" {
		fileName = path.Join(dataDir, defaultFileName)
	}

	name := ""
	if _, err := os.Stat(fileName); err == nil {
		content, err := os.ReadFile(fileName)
		if err != nil {
			panic(err)
		}
		name = strings.TrimSpace(string(content))
	} else {
		name = prefix + cuid2.Generate()
		err := os.WriteFile(fileName, []byte(name), 0644)
		if err != nil {
			panic(err)
		}
	}
	return name
}
