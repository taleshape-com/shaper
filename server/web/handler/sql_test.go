// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shaper/server/core"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "github.com/duckdb/duckdb-go/v2"
)

func TestExecuteSQL(t *testing.T) {
	e := echo.New()
	
	db, err := sqlx.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	app := &core.App{DuckDB: db}
	
	t.Run("valid single query", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"sql": "SELECT 1 as id, 'hello' as name",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/sql", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := ExecuteSQL(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		expected := "id,name\n1,hello\n"
		if rec.Body.String() != expected {
			t.Errorf("expected %q, got %q", expected, rec.Body.String())
		}
	})

	t.Run("multiple queries forbidden", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"sql": "SELECT 1; SELECT 2;",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/sql", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := ExecuteSQL(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
		
		if !strings.Contains(rec.Body.String(), "Only one SQL query is allowed") {
			t.Errorf("unexpected error message: %s", rec.Body.String())
		}
	})

	t.Run("disallowed SQL statement", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"sql": "DELETE FROM users",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/sql", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := ExecuteSQL(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		if !strings.Contains(rec.Body.String(), "disallowed SQL statement") {
			t.Errorf("unexpected error message: %s", rec.Body.String())
		}
	})

	t.Run("allowed side effect", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"sql": "SET VARIABLE x = 1",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/sql", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := ExecuteSQL(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("empty SQL", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"sql": "",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/sql", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := ExecuteSQL(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})
}
