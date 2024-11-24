// TODO: authz to restrict access. Look into casbin and other existing solutions https://github.com/labstack/echo-contrib/tree/master/casbin
package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"shaper/core"
	"shaper/server"
	"shaper/util/signals"
	"time"

	"github.com/jmoiron/sqlx"
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
	DashboardDir      string
	ExecutableModTime time.Time
	CustomCSS         string
	Favicon           string
}

func main() {
	config := loadConfig()
	signals.HandleInterrupt(Run(config))
}

func loadConfig() Config {
	flags := ff.NewFlagSet("shaper")
	help := flags.Bool('h', "help", "show help")
	addr := flags.String('a', "addr", "0.0.0.0", "server address")
	port := flags.Int('p', "port", 3000, "port to listen on")
	dbFile := flags.String('f', "duckdb", "", "path to duckdb file (default: use in-memory db)")
	loginToken := flags.String('t', "token", "", "token used for login (required)")
	dashboardDir := flags.String('d', "dashboards", "", "path to directory to read dashboard SQL files from (required)")
	customCSS := flags.String('c', "css", "", "CSS string to inject into the frontend")
	favicon := flags.String('i', "favicon", "", "path to override favicon. Must end .svg or .ico")

	err := ff.Parse(flags, os.Args[1:],
		ff.WithEnvVarPrefix("SHAPER"),
		ff.WithConfigFileParser(ff.PlainParser),
	)
	if err == nil && *loginToken == "" {
		err = fmt.Errorf("--token must be set")
	}
	if err == nil && *dashboardDir == "" {
		err = fmt.Errorf("--dashboards must be set")
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

	config := Config{
		Address:           *addr,
		Port:              *port,
		DBFile:            *dbFile,
		LoginToken:        *loginToken,
		DashboardDir:      *dashboardDir,
		ExecutableModTime: executableModTime,
		CustomCSS:         *customCSS,
		Favicon:           *favicon,
	}
	return config
}

func Run(config Config) func(context.Context) {
	fmt.Println("⇨ dashboard directory:", config.DashboardDir)
	if config.Favicon != "" {
		fmt.Println("⇨ custom favicon:", config.Favicon)
	}
	if config.CustomCSS != "" {
		fmt.Println("⇨ custom CSS injected into frontend")
	}

	// connect to duckdb
	db, err := sqlx.Connect("duckdb", config.DBFile)
	if err != nil {
		panic(err)
	}
	if config.DBFile != "" {
		fmt.Println("⇨ connected to duckdb", config.DBFile)
	} else {
		fmt.Println("⇨ connected to in-memory duckdb")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	app, err := core.New(db, logger, config.LoginToken, config.DashboardDir)
	if err != nil {
		panic(err)
	}
	e := server.Start(config.Address, config.Port, app, frontendFS, config.ExecutableModTime, config.CustomCSS, config.Favicon)

	return func(ctx context.Context) {
		if err := db.Close(); err != nil {
			logger.ErrorContext(ctx, "error closing database connection", slog.Any("error", err))
		}
		db.Close()
		if err := e.Shutdown(ctx); err != nil {
			logger.ErrorContext(ctx, "error stopping server", slog.Any("error", err))
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
