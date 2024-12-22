// TODO: JWT https://echo.labstack.com/docs/middleware/jwt
package core

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

type App struct {
	db         *sqlx.DB
	Logger     *slog.Logger
	LoginToken string
	Schema     string
	JWTSecret  []byte
	JWTExp     time.Duration
}

func New(db *sqlx.DB, logger *slog.Logger, loginToken string, schema string, jwtSecret []byte, jwtExp time.Duration) (*App, error) {
	if err := initDB(db, schema); err != nil {
		return nil, err
	}
	return &App{
		db:         db,
		Logger:     logger,
		LoginToken: loginToken,
		Schema:     schema,
		JWTSecret:  jwtSecret,
		JWTExp:     jwtExp,
	}, nil
}

func initDB(db *sqlx.DB, schema string) error {
	// Create schema if not exists
	if _, err := db.Exec("CREATE SCHEMA IF NOT EXISTS " + schema); err != nil {
		return fmt.Errorf("error creating schema: %w", err)
	}

	// Create dashboards table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + schema + `.dashboards (
			id VARCHAR PRIMARY KEY,
			path VARCHAR NOT NULL,
			name VARCHAR NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_by VARCHAR,
			updated_by VARCHAR
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating dashboards table: %w", err)
	}

	// Create custom types
	for _, t := range dbTypes {
		if err := createType(db, t.Name, t.Definition); err != nil {
			return err
		}
	}
	return nil
}
