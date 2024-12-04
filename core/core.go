// TODO: JWT https://echo.labstack.com/docs/middleware/jwt
package core

import (
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

type App struct {
	db           *sqlx.DB
	Logger       *slog.Logger
	LoginToken   string
	DashboardDir string
	JWTSecret    []byte
	JWTExp       time.Duration
}

func New(db *sqlx.DB, logger *slog.Logger, loginToken string, dashboardDir string, jwtSecret []byte, jwtExp time.Duration) (*App, error) {
	if err := initDB(db); err != nil {
		return nil, err
	}
	return &App{
		db:           db,
		Logger:       logger,
		LoginToken:   loginToken,
		DashboardDir: dashboardDir,
		JWTSecret:    jwtSecret,
		JWTExp:       jwtExp,
	}, nil
}

func initDB(db *sqlx.DB) error {
	for _, t := range dbTypes {
		if err := createType(db, t.Name, t.Definition); err != nil {
			return err
		}
	}
	return nil
}
