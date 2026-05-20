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
