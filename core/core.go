// TODO: JWT https://echo.labstack.com/docs/middleware/jwt
package core

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type App struct {
	db           *sqlx.DB
	Logger       *slog.Logger
	LoginToken   string
	DashboardDir string
}

func New(db *sqlx.DB, logger *slog.Logger, loginToken string, dashboardDir string) *App {
	return &App{db: db, Logger: logger, LoginToken: loginToken, DashboardDir: dashboardDir}
}
