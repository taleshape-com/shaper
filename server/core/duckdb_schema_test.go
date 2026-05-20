// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	_ "github.com/duckdb/duckdb-go/v2"
)

func TestGetSchema(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shaper-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	sqlitePath := filepath.Join(tmpDir, "test.sqlite")
	duckdbPath := filepath.Join(tmpDir, "test.duckdb")

	sqliteDbx, err := sqlx.Connect("sqlite", sqlitePath)
	assert.NoError(t, err)
	defer sqliteDbx.Close()

	err = initSQLite(sqliteDbx)
	assert.NoError(t, err)

	duckdbDbx, err := sqlx.Connect("duckdb", duckdbPath)
	assert.NoError(t, err)
	defer duckdbDbx.Close()

	// Create some test data in DuckDB
	_, err = duckdbDbx.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		COMMENT ON TABLE users IS 'A table of users';
		COMMENT ON COLUMN users.name IS 'The user''s full name';

		CREATE VIEW active_users AS SELECT * FROM users WHERE name IS NOT NULL;
		COMMENT ON VIEW active_users IS 'Only active users';

		CREATE TYPE mood AS ENUM ('happy', 'sad', 'ok');
		CREATE TABLE feelings (
			user_id INTEGER,
			current_mood mood,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
	`)
	assert.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := &App{
		Sqlite:    sqliteDbx,
		DuckDB:    duckdbDbx,
		DuckDBDSN: duckdbPath,
		Logger:    logger,
	}

	res, err := app.GetSchema(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// Verify the schema structure
	foundMain := false
	for _, db := range res.Databases {
		if db.Name == "test" || db.Name == "memory" || db.Name == "test.duckdb" || db.Name == "main" {
			foundMain = true
			foundDefault := false
			for _, s := range db.Schemas {
				if s.Name == "main" {
					foundDefault = true
					
					// Check Tables
					assert.Len(t, s.Tables, 2)
					var usersTable *struct{Name string}
					for _, table := range s.Tables {
						if table.Name == "users" {
							usersTable = &struct{Name string}{Name: table.Name}
							assert.Equal(t, "A table of users", table.Comment)
							assert.Len(t, table.Columns, 4)
							assert.Equal(t, "id", table.Columns[0].Name)
							assert.Equal(t, "name", table.Columns[1].Name)
							assert.Equal(t, "The user's full name", table.Columns[1].Comment)
							
							// Check Constraints
							assert.NotEmpty(t, table.Constraints)
						}
					}
					assert.NotNil(t, usersTable)

					// Check Views
					assert.Len(t, s.Views, 1)
					assert.Equal(t, "active_users", s.Views[0].Name)
					assert.Contains(t, s.Views[0].Definition, "SELECT")

					// Check Enums
					assert.Len(t, s.Enums, 1)
					assert.Equal(t, "mood", s.Enums[0].Name)
					assert.ElementsMatch(t, []string{"happy", "sad", "ok"}, s.Enums[0].Values)
				}
			}
			assert.True(t, foundDefault)
		}
	}
	assert.True(t, foundMain)
}
