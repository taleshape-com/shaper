// TODO: config via https://github.com/peterbourgon/ff
// TODO: authz to restrict access. Look into casbin and other existing solutions https://github.com/labstack/echo-contrib/tree/master/casbin
package main

import (
	"context"
	"duckshape/core"
	"duckshape/server"
	"duckshape/util/signals"
	"log/slog"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	signals.HandleInterrupt(Run())
}

func Run() func(context.Context) {
	// connect to duckdb
	db, err := sqlx.Connect("duckdb", "")
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	app := core.New(db, logger)
	e := server.Start(app)

	return func(ctx context.Context) {
		if err := db.Close(); err != nil {
			logger.ErrorContext(ctx, "error closing database connection", err)
		}
		db.Close()
		if err := e.Shutdown(ctx); err != nil {
			logger.ErrorContext(ctx, "error stopping server", err)
		}
	}
}
