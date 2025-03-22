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
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/marcboeker/go-duckdb"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

//go:embed dist
var frontendFS embed.FS

const APP_NAME = "shaper"

type Config struct {
	SessionExp          time.Duration
	InviteExp           time.Duration
	Address             string
	DataDir             string
	Schema              string
	ExecutableModTime   time.Time
	CustomCSS           string
	Favicon             string
	JWTExp              time.Duration
	NatsHost            string
	NatsPort            int
	NatsToken           string
	NatsJSDir           string
	NatsJSKey           string
	NatsMaxStore        int64 // in bytes
	NatsDontListen      bool
	IngestSubjectPrefix string
	DuckDB              string
	DuckDBExtDir        string
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

	flags := ff.NewFlagSet("shaper")
	help := flags.Bool('h', "help", "show help")
	addr := flags.StringLong("addr", "localhost:3000", "server address")
	dataDir := flags.String('d', "dir", path.Join(homeDir, ".shaper"), "directory to store data, by default set to /data in docker container)")
	schema := flags.StringLong("schema", "_shaper", "DB schema name for internal tables")
	customCSS := flags.StringLong("css", "", "CSS string to inject into the frontend")
	favicon := flags.StringLong("favicon", "", "path to override favicon. Must end .svg or .ico")
	jwtExp := flags.DurationLong("jwtexp", 15*time.Minute, "JWT expiration duration")
	sessionExp := flags.DurationLong("sessionexp", 30*24*time.Hour, "Session expiration duration")
	inviteExp := flags.DurationLong("inviteexp", 7*24*time.Hour, "Invite expiration duration")
	natsHost := flags.StringLong("nats-host", "0.0.0.0", "NATS server host")
	natsPort := flags.IntLong("nats-port", 4222, "NATS server port")
	natsToken := flags.StringLong("nats-token", "", "NATS authentication token")
	natsJSDir := flags.StringLong("nats-dir", "", "Override JetStream storage directory (default: [--dir]/nats)")
	natsJSKey := flags.StringLong("nats-js-key", "", "JetStream encryption key")
	natsMaxStore := flags.StringLong("nats-max-store", "0", "Maximum storage in bytes, set to 0 for unlimited")
	natsDontListen := flags.BoolLong("nats-dont-listen", "Disable NATS from listening on any port")
	ingestSubjectPrefix := flags.StringLong("ingest-subject-prefix", "shaper.ingest.", "prefix for ingest subjects")
	duckdb := flags.StringLong("duckdb", "", "Override duckdb DSN (default: [--dir]/data.duckdb)")
	duckdbExtDir := flags.StringLong("duckdb-ext-dir", "", "Override DuckDB extension directory, by default set to /data/duckdb_extensions in docker (default: ~/.duckdb/extensions/)")
	flags.StringLong("config-file", "", "path to config file")

	err = ff.Parse(flags, os.Args[1:],
		ff.WithEnvVarPrefix("SHAPER"),
		ff.WithConfigFileFlag("config-file"),
		ff.WithConfigFileParser(ff.PlainParser),
	)
	if err != nil {
		fmt.Printf("%s\n", ffhelp.Flags(flags))
		fmt.Printf("err=%v\n", err)
		os.Exit(1)
	}
	if *help {
		fmt.Printf("%s\n", ffhelp.Flags(flags))
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

	natsDir := *dataDir + "/nats"
	if *natsJSDir != "" {
		natsDir = *natsJSDir
	}

	config := Config{
		Address:             *addr,
		DataDir:             *dataDir,
		Schema:              *schema,
		ExecutableModTime:   executableModTime,
		CustomCSS:           *customCSS,
		Favicon:             *favicon,
		JWTExp:              *jwtExp,
		SessionExp:          *sessionExp,
		InviteExp:           *inviteExp,
		NatsHost:            *natsHost,
		NatsPort:            *natsPort,
		NatsToken:           *natsToken,
		NatsJSDir:           natsDir,
		NatsJSKey:           *natsJSKey,
		NatsMaxStore:        maxStore,
		NatsDontListen:      *natsDontListen,
		IngestSubjectPrefix: *ingestSubjectPrefix,
		DuckDB:              *duckdb,
		DuckDBExtDir:        *duckdbExtDir,
	}
	return config
}

func Run(cfg Config) func(context.Context) {
	if cfg.Favicon != "" {
		fmt.Println("⇨ custom favicon:", cfg.Favicon)
	}
	if cfg.CustomCSS != "" {
		fmt.Println("⇨ custom CSS injected into frontend")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Make sure data directory exists
	if _, err := os.Stat(cfg.DataDir); os.IsNotExist(err) {
		err := os.Mkdir(cfg.DataDir, 0755)
		logger.Info("created data directory", slog.Any("path", cfg.DataDir))
		if err != nil {
			panic(err)
		}
	}

	dbFile := cfg.DuckDB
	if cfg.DuckDB == "" {
		dbFile = path.Join(cfg.DataDir, "data.duckdb")
	}

	// connect to duckdb
	dbConnector, err := duckdb.NewConnector(dbFile, nil)
	if err != nil {
		panic(err)
	}
	sqlDB := sql.OpenDB(dbConnector)
	db := sqlx.NewDb(sqlDB, "duckdb")
	logger.Info("connected to duckdb", slog.Any("file", dbFile))

	if cfg.DuckDBExtDir != "" {
		_, err := db.Exec("SET extension_directory = ?", cfg.DuckDBExtDir)
		if err != nil {
			panic(err)
		}
		logger.Info("set DuckDB extension directory", slog.Any("path", cfg.DuckDBExtDir))
	}

	app, err := core.New(
		APP_NAME,
		db,
		logger,
		cfg.Schema,
		cfg.JWTExp,
		cfg.SessionExp,
		cfg.InviteExp,
		cfg.IngestSubjectPrefix,
	)
	if err != nil {
		panic(err)
	}

	// TODO: refactor - comms should be part of core
	c, err := comms.New(comms.Config{
		Logger:     logger.WithGroup("nats"),
		Host:       cfg.NatsHost,
		Port:       cfg.NatsPort,
		Token:      cfg.NatsToken,
		JSDir:      cfg.NatsJSDir,
		JSKey:      cfg.NatsJSKey,
		MaxStore:   cfg.NatsMaxStore,
		DontListen: cfg.NatsDontListen,
		App:        app,
	})
	if err != nil {
		panic(err)
	}

	ingestConsumer, err := ingest.Start(cfg.IngestSubjectPrefix, dbConnector, db, logger.WithGroup("ingest"), c.Conn)
	if err != nil {
		panic(err)
	}

	err = app.Init(c.Conn)
	if err != nil {
		panic(err)
	}

	e := web.Start(cfg.Address, app, frontendFS, cfg.ExecutableModTime, cfg.CustomCSS, cfg.Favicon)

	return func(ctx context.Context) {
		logger.Info("initiating shutdown...")
		logger.Info("stopping web server...")
		if err := e.Shutdown(ctx); err != nil {
			logger.ErrorContext(ctx, "error stopping server", slog.Any("error", err))
		}
		logger.Info("stopping NATS...")
		ingestConsumer.Close()
		c.Close()
		logger.Info("closing DB connections...")
		if err := db.Close(); err != nil {
			logger.ErrorContext(ctx, "error closing database connection", slog.Any("error", err))
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
