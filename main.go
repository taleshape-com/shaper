// TODO: authz to restrict access. Look into casbin and other existing solutions https://github.com/labstack/echo-contrib/tree/master/casbin
package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"
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

type Config struct {
	Address           string
	Port              int
	DBFile            string
	LoginToken        string
	Schema            string
	ExecutableModTime time.Time
	CustomCSS         string
	Favicon           string
	JWTSecret         []byte
	JWTExp            time.Duration
	NatsHost          string
	NatsPort          int
	NatsToken         string
	NatsJSDir         string
	NatsJSKey         string
	NatsMaxStore      int64 // in bytes
	NatsDontListen    bool
}

func main() {
	config := loadConfig()
	signals.HandleInterrupt(Run(config))
}

func loadConfig() Config {
	flags := ff.NewFlagSet("shaper")
	help := flags.Bool('h', "help", "show help")
	addr := flags.StringLong("addr", "0.0.0.0", "server address")
	port := flags.Int('p', "port", 3000, "port to listen on")
	dbFile := flags.String('d', "duckdb", "", "path to duckdb file (default: use in-memory db)")
	loginToken := flags.String('t', "token", "", "token used for login (required)")
	schema := flags.StringLong("schema", "_shaper", "DB schema name for internal tables")
	customCSS := flags.StringLong("css", "", "CSS string to inject into the frontend")
	favicon := flags.StringLong("favicon", "", "path to override favicon. Must end .svg or .ico")
	jwtSecret := flags.StringLong("jwtsecret", "", "JWT secret to sign auth tokens")
	jwtExp := flags.DurationLong("jwtexp", 15*time.Minute, "JWT expiration duration")
	natsHost := flags.StringLong("nats-host", "0.0.0.0", "NATS server host")
	natsPort := flags.IntLong("nats-port", 4222, "NATS server port")
	natsToken := flags.StringLong("nats-token", "", "NATS authentication token")
	natsJSDir := flags.String('n', "nats-dir", "", "JetStream storage directory (default: in-memory)")
	natsJSKey := flags.StringLong("nats-js-key", "", "JetStream encryption key")
	natsMaxStore := flags.StringLong("nats-max-store", "0", "Maximum storage in bytes (0 for unlimited)")
	natsDontListen := flags.BoolLong("nats-dont-listen", "Disable NATS from listening on any port")

	err := ff.Parse(flags, os.Args[1:],
		ff.WithEnvVarPrefix("SHAPER"),
		ff.WithConfigFileParser(ff.PlainParser),
	)
	if err == nil && *loginToken == "" {
		err = fmt.Errorf("--token must be set")
	}
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

	config := Config{
		Address:           *addr,
		Port:              *port,
		DBFile:            *dbFile,
		LoginToken:        *loginToken,
		Schema:            *schema,
		ExecutableModTime: executableModTime,
		CustomCSS:         *customCSS,
		Favicon:           *favicon,
		JWTSecret:         []byte(*jwtSecret),
		JWTExp:            *jwtExp,
		NatsHost:          *natsHost,
		NatsPort:          *natsPort,
		NatsToken:         *natsToken,
		NatsJSDir:         *natsJSDir,
		NatsJSKey:         *natsJSKey,
		NatsMaxStore:      maxStore,
		NatsDontListen:    *natsDontListen,
	}
	return config
}

func Run(config Config) func(context.Context) {
	if config.Favicon != "" {
		fmt.Println("⇨ custom favicon:", config.Favicon)
	}
	if config.CustomCSS != "" {
		fmt.Println("⇨ custom CSS injected into frontend")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// connect to duckdb
	dbConnector, err := duckdb.NewConnector(config.DBFile, nil)
	if err != nil {
		panic(err)
	}
	sqlDB := sql.OpenDB(dbConnector)
	db := sqlx.NewDb(sqlDB, "duckdb")
	if err != nil {
		panic(err)
	}
	if config.DBFile != "" {
		fmt.Println("⇨ connected to duckdb", config.DBFile)
	} else {
		fmt.Println("⇨ connected to in-memory duckdb")
	}

	c, err := comms.New(comms.Config{
		Logger:     logger.WithGroup("nats"),
		Host:       config.NatsHost,
		Port:       config.NatsPort,
		Token:      config.NatsToken,
		JSDir:      config.NatsJSDir,
		JSKey:      config.NatsJSKey,
		MaxStore:   config.NatsMaxStore,
		DontListen: config.NatsDontListen,
	})
	if err != nil {
		panic(err)
	}

	persistNATS := config.NatsJSDir != ""

	app, err := core.New(db, c.Conn, logger, config.LoginToken, config.Schema, config.JWTSecret, config.JWTExp, persistNATS)
	if err != nil {
		panic(err)
	}

	ingestConsumer, err := ingest.Start(dbConnector, db, logger.WithGroup("ingest"), c.Conn, persistNATS)
	if err != nil {
		panic(err)
	}

	e := web.Start(config.Address, config.Port, app, frontendFS, config.ExecutableModTime, config.CustomCSS, config.Favicon)

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
