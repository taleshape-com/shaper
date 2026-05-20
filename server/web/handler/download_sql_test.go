// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shaper/server/core"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go/jetstream"
	_ "github.com/duckdb/duckdb-go/v2"
)

// MockKV implements jetstream.KeyValue for testing
type MockKV struct {
	jetstream.KeyValue
	data map[string][]byte
}

func (m *MockKV) Put(ctx context.Context, key string, value []byte) (uint64, error) {
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[key] = value
	return 0, nil
}

type mockEntry struct {
	value []byte
}
func (e *mockEntry) Bucket() string { return "" }
func (e *mockEntry) Key() string { return "" }
func (e *mockEntry) Value() []byte { return e.value }
func (e *mockEntry) Revision() uint64 { return 0 }
func (e *mockEntry) Created() time.Time { return time.Now() }
func (e *mockEntry) Delta() uint64 { return 0 }
func (e *mockEntry) Operation() jetstream.KeyValueOp { return jetstream.KeyValuePut }

func (m *MockKV) Get(ctx context.Context, key string) (jetstream.KeyValueEntry, error) {
	if v, ok := m.data[key]; ok {
		return &mockEntry{value: v}, nil
	}
	return nil, jetstream.ErrKeyNotFound
}

func TestDownloadSQL(t *testing.T) {
	e := echo.New()
	
	db, err := sqlx.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	kv := &MockKV{data: make(map[string][]byte)}
	app := &core.App{
		DuckDB:          db,
		TmpDashboardsKv: kv,
		DownloadsKv:     kv,
		JWTSecret:       []byte("secret"),
		JWTExp:          time.Hour,
	}

	t.Run("valid SQL download", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"sql": "SELECT 1 as id, 'hello' as name",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/download/test.csv", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("filename")
		c.SetParamValues("test.csv")

		// Set actor
		actor := &core.Actor{Type: core.ActorAPIKey, ID: "test-key"}
		ctx := core.ContextWithActor(req.Context(), actor)
		c.SetRequest(req.WithContext(ctx))

		handler := DownloadSQL(app, "http://localhost", "2006-01-02")
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		expected := "id,name\n1,hello\n"
		if rec.Body.String() != expected {
			t.Errorf("expected %q, got %q", expected, rec.Body.String())
		}
	})

	t.Run("mode=url", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"sql": "SELECT 1 as id, 'hello' as name",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/download/test.csv?mode=url", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("filename")
		c.SetParamValues("test.csv")

		// Set actor
		actor := &core.Actor{Type: core.ActorAPIKey, ID: "test-key"}
		ctx := core.ContextWithActor(req.Context(), actor)
		c.SetRequest(req.WithContext(ctx))

		handler := DownloadSQL(app, "http://localhost", "2006-01-02")
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var result struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if !strings.HasPrefix(result.URL, "/api/download/") || !strings.HasSuffix(result.URL, "/test.csv") {
			t.Errorf("unexpected URL: %s", result.URL)
		}
	})

	t.Run("invalid file type", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"sql": "SELECT 1",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/download/test.exe", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("filename")
		c.SetParamValues("test.exe")

		// Set actor
		actor := &core.Actor{Type: core.ActorAPIKey, ID: "test-key"}
		ctx := core.ContextWithActor(req.Context(), actor)
		c.SetRequest(req.WithContext(ctx))

		handler := DownloadSQL(app, "http://localhost", "2006-01-02")
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
		
		if !strings.Contains(rec.Body.String(), "Invalid file type") {
			t.Errorf("unexpected error message: %s", rec.Body.String())
		}
	})

	t.Run("missing SQL", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"sql": "",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/download/test.csv", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("filename")
		c.SetParamValues("test.csv")

		// Set actor
		actor := &core.Actor{Type: core.ActorAPIKey, ID: "test-key"}
		ctx := core.ContextWithActor(req.Context(), actor)
		c.SetRequest(req.WithContext(ctx))

		handler := DownloadSQL(app, "http://localhost", "2006-01-02")
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
		
		if !strings.Contains(rec.Body.String(), "SQL is required") {
			t.Errorf("unexpected error message: %s", rec.Body.String())
		}
	})
}
