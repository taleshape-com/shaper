// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"net/http"
	"net/http/httptest"
	"shaper/server/core"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "github.com/duckdb/duckdb-go/v2"
)

func TestGetSchema(t *testing.T) {
	e := echo.New()
	
	db, err := sqlx.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	app := &core.App{DuckDB: db}
	
	req := httptest.NewRequest(http.MethodGet, "/api/schema", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := GetSchema(app)
	if err := handler(c); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get(echo.HeaderContentType)
	if contentType != echo.MIMEApplicationJSON && contentType != echo.MIMEApplicationJSONCharsetUTF8 {
		t.Errorf("expected json content type, got %s", contentType)
	}
}

func TestGetSchema_WithIgnore(t *testing.T) {
	e := echo.New()
	
	db, err := sqlx.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	// Create a table to ignore
	_, err = db.Exec("CREATE TABLE to_ignore (id INTEGER)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	_, err = db.Exec("CREATE TABLE to_keep (id INTEGER)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	app := &core.App{DuckDB: db}
	
	// We need to know the database name, usually 'memory' or 'main' for in-memory duckdb
	var dbName string
	err = db.Get(&dbName, "SELECT database_name FROM duckdb_databases() WHERE NOT internal LIMIT 1")
	if err != nil {
		t.Fatalf("failed to get db name: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/schema?ignore="+dbName+".main.to_ignore", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := GetSchema(app)
	if err := handler(c); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify the table is ignored in the response body would be good but for now we just check it doesn't crash
}
