// TODO: JWT https://echo.labstack.com/docs/middleware/jwt
package core

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type App struct {
	db     *sqlx.DB
	Logger *slog.Logger
}

func New(db *sqlx.DB, logger *slog.Logger) *App {
	return &App{db: db, Logger: logger}
}
