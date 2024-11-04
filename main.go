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

	"github.com/jmoiron/sqlx"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

//go:embed dist
var frontendFS embed.FS

type Config struct {
	Address string
	Port    int
	DBFile  string
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

	err := ff.Parse(flags, os.Args[1:],
		ff.WithEnvVarPrefix("SHAPER"),
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

	config := Config{
		Address: *addr,
		Port:    *port,
		DBFile:  *dbFile,
	}
	return config
}

func Run(config Config) func(context.Context) {
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

	app := core.New(db, logger)
	e := server.Start(config.Address, config.Port, app, frontendFS)

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
