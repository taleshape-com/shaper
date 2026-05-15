// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shaper/server/core"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "github.com/duckdb/duckdb-go/v2"
)

func TestValidate(t *testing.T) {
	e := echo.New()
	
	db, err := sqlx.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	app := &core.App{DuckDB: db}
	
	t.Run("valid dashboard", func(t *testing.T) {
		reqBody, _ := json.Marshal(ValidateRequest{
			Type: "dashboard",
			SQL:  "SELECT 1",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/validate", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := Validate(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp ValidateResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if !resp.Valid {
			t.Errorf("expected valid to be true, got false. Error: %s", resp.Error)
		}
	})

	t.Run("invalid dashboard SQL", func(t *testing.T) {
		reqBody, _ := json.Marshal(ValidateRequest{
			Type: "dashboard",
			SQL:  "SELECT * FROM non_existent_table",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/validate", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := Validate(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp ValidateResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp.Valid {
			t.Errorf("expected valid to be false, got true")
		}
		if resp.Error == "" {
			t.Errorf("expected error message, got empty")
		}
	})

	t.Run("task validation unsupported", func(t *testing.T) {
		reqBody, _ := json.Marshal(ValidateRequest{
			Type: "task",
			SQL:  "SELECT 1",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/validate", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := Validate(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		reqBody, _ := json.Marshal(ValidateRequest{
			Type: "invalid",
			SQL:  "SELECT 1",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/validate", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := Validate(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})
}
