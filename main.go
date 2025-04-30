package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path"
	"shaper/comms"
	"shaper/core"
	"shaper/ingest"
	"shaper/util/signals"
	"shaper/web"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/marcboeker/go-duckdb/v2"
	_ "github.com/marcboeker/go-duckdb/v2"
	"github.com/nrednav/cuid2"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

//go:embed dist
var frontendFS embed.FS

const APP_NAME = "shaper"

// TODO: Add a short description of what shaper does once I know how to explain it
const USAGE = `All options are optional.

All configuration options can be set via command line flags, environment variables or config file.

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
	IngestConsumerNameFile string
	StateConsumerNameFile  string
	IngestSubjectPrefix    string
	StateSubjectPrefix     string
	DuckDB                 string
	DuckDBExtDir           string
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
	addr := flags.StringLong("addr", "localhost:5454", "server address")
	dataDir := flags.String('d', "dir", path.Join(homeDir, ".shaper"), "directory to store data, by default set to /data in docker container)")
	schema := flags.StringLong("schema", "_shaper", "DB schema name for internal tables")
	basePath := flags.StringLong("basepath", "/", "Base URL path the frontend is served from. Override if you are using a reverse proxy and serve the frontend from a subpath.")
	customCSS := flags.StringLong("css", "", "CSS string to inject into the frontend")
	favicon := flags.StringLong("favicon", "", "path to override favicon. Must end .svg or .ico")
	jwtExp := flags.DurationLong("jwtexp", 15*time.Minute, "JWT expiration duration")
	sessionExp := flags.DurationLong("sessionexp", 30*24*time.Hour, "Session expiration duration")
	inviteExp := flags.DurationLong("inviteexp", 7*24*time.Hour, "Invite expiration duration")
	natsServers := flags.StringLong("nats-servers", "", "Use external NATS servers, specify as comma separated list")
	natsHost := flags.StringLong("nats-host", "0.0.0.0", "NATS server host")
	natsPort := flags.Int('p', "nats-port", 0, "NATS server port. If not specified, NATS will not listen on any port.")
	natsToken := flags.String('t', "nats-token", "", "NATS authentication token")
	natsJSDir := flags.StringLong("nats-dir", "", "Override JetStream storage directory (default: [--dir]/nats)")
	natsJSKey := flags.StringLong("nats-js-key", "", "JetStream encryption key")
	natsMaxStore := flags.StringLong("nats-max-store", "0", "Maximum storage in bytes, set to 0 for unlimited")
	streamPrefix := flags.StringLong("stream-prefix", "", "Prefix for NATS stream and KV bucket names. Must be a valid NATS subject name")
	stateStream := flags.StringLong("state-stream", "shaper-state", "NATS stream name for state messages")
	ingestStream := flags.StringLong("ingest-stream", "shaper-ingest", "NATS stream name for ingest messages")
	configKVBucket := flags.StringLong("config-kv-bucket", "shaper-config", "Name for NATS config KV bucket")
	ingestConsumerNameFile := flags.StringLong("ingest-consumer-name-file", "", "File to store and lookup name for ingest consumer (default: [--dir]/ingest-consumer-name.txt)")
	stateConsumerNameFile := flags.StringLong("state-consumer-name-file", "", "File to store and lookup name for state consumer (default: [--dir]/state-consumer-name.txt)")
	subjectPrefix := flags.StringLong("subject-prefix", "", "prefix for NATS subjects. Must be a valid NATS subject name. Should probably end with a dot.")
	ingestSubjectPrefix := flags.StringLong("ingest-subject-prefix", "shaper.ingest.", "prefix for ingest NATS subjects")
	stateSubjectPrefix := flags.StringLong("state-subject-prefix", "shaper.state.", "prefix for state NATS subjects")
	duckdb := flags.StringLong("duckdb", "", "Override duckdb DSN (default: [--dir]/shaper.duckdb)")
	duckdbExtDir := flags.StringLong("duckdb-ext-dir", "", "Override DuckDB extension directory, by default set to /data/duckdb_extensions in docker (default: ~/.duckdb/extensions/)")
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
		fmt.Printf("%s\n", ffhelp.Flags(flags, USAGE))
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

	config := Config{
		Address:                *addr,
		DataDir:                *dataDir,
		Schema:                 *schema,
		ExecutableModTime:      executableModTime,
		BasePath:               "/" + strings.TrimPrefix(strings.TrimSuffix(*basePath, "/"), "/"),
		CustomCSS:              *customCSS,
		Favicon:                *favicon,
		JWTExp:                 *jwtExp,
		SessionExp:             *sessionExp,
		InviteExp:              *inviteExp,
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
		IngestConsumerNameFile: *ingestConsumerNameFile,
		StateConsumerNameFile:  *stateConsumerNameFile,
		IngestSubjectPrefix:    *subjectPrefix + *ingestSubjectPrefix,
		StateSubjectPrefix:     *subjectPrefix + *stateSubjectPrefix,
		DuckDB:                 *duckdb,
		DuckDBExtDir:           *duckdbExtDir,
	}
	return config
}

func Run(cfg Config) func(context.Context) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	logger.Info("Starting Shaper")
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
	db := sqlx.NewDb(sqlDB, "duckdb")
	logger.Info("DuckDB opened", slog.Any("file", dbFile))

	if cfg.DuckDBExtDir != "" {
		_, err := db.Exec("SET extension_directory = ?", cfg.DuckDBExtDir)
		if err != nil {
			panic(err)
		}
		logger.Info("set DuckDB extension directory", slog.Any("path", cfg.DuckDBExtDir))
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
		cfg.IngestSubjectPrefix,
		cfg.StateSubjectPrefix,
		cfg.StateStreamName,
		stateConsumerName,
		cfg.ConfigKVBucketName,
	)
	if err != nil {
		panic(err)
	}

	// TODO: refactor - comms should be part of core
	c, err := comms.New(comms.Config{
		Logger:   logger.WithGroup("nats"),
		Servers:  cfg.NatsServers,
		Host:     cfg.NatsHost,
		Port:     cfg.NatsPort,
		Token:    cfg.NatsToken,
		JSDir:    cfg.NatsJSDir,
		JSKey:    cfg.NatsJSKey,
		MaxStore: cfg.NatsMaxStore,
		App:      app,
	})
	if err != nil {
		panic(err)
	}

	ingestConsumer, err := ingest.Start(cfg.IngestSubjectPrefix, dbConnector, db, logger.WithGroup("ingest"), c.Conn, cfg.IngestStreamName, ingestConsumerName)
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

// Helper function to get or generate consumer name
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
