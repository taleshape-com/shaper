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

	res, err := app.GetSchema(context.Background(), nil)
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

func TestGetSchema_Filtering(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shaper-test-filtering-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	duckdbPath := filepath.Join(tmpDir, "test.duckdb")
	duckdbDbx, err := sqlx.Connect("duckdb", duckdbPath)
	assert.NoError(t, err)
	defer duckdbDbx.Close()

	// Create test data in multiple schemas
	_, err = duckdbDbx.Exec(`
		CREATE SCHEMA s1;
		CREATE TABLE s1.t1 (id INTEGER);
		CREATE TABLE s1.t2 (id INTEGER);
		CREATE SCHEMA s2;
		CREATE TABLE s2.t3 (id INTEGER);
	`)
	assert.NoError(t, err)

	app := &App{DuckDB: duckdbDbx}

	// 1. Ignore specific table
	// We need to know the database name. DuckDB defaults to 'main' for the primary database.
	dbName := "test"
	// But let's check what it actually is
	checkRes, _ := app.GetSchema(context.Background(), nil)
	if len(checkRes.Databases) > 0 {
		dbName = checkRes.Databases[0].Name
	}

	res, err := app.GetSchema(context.Background(), []string{dbName + ".s1.t1"})
	assert.NoError(t, err)
	foundS1 := false
	for _, db := range res.Databases {
		if db.Name == dbName {
			for _, s := range db.Schemas {
				if s.Name == "s1" {
					foundS1 = true
					assert.Len(t, s.Tables, 1)
					assert.Equal(t, "t2", s.Tables[0].Name)
				}
			}
		}
	}
	assert.True(t, foundS1)

	// 2. Ignore whole schema
	res, err = app.GetSchema(context.Background(), []string{dbName + ".s1"})
	assert.NoError(t, err)
	for _, db := range res.Databases {
		if db.Name == dbName {
			for _, s := range db.Schemas {
				assert.NotEqual(t, "s1", s.Name)
			}
		}
	}

	// 3. Ignore whole database
	res, err = app.GetSchema(context.Background(), []string{dbName})
	assert.NoError(t, err)
	assert.Len(t, res.Databases, 0)
}
