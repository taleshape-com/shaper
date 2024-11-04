// TODO: JWT https://echo.labstack.com/docs/middleware/jwt
package core

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type App struct {
	db         *sqlx.DB
	Logger     *slog.Logger
	LoginToken string
}

func New(db *sqlx.DB, logger *slog.Logger, loginToken string) *App {
	return &App{db: db, Logger: logger, LoginToken: loginToken}
}
