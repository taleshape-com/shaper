// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"shaper/server/core"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "modernc.org/sqlite"
)

func setupTestApp(t *testing.T) *core.App {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if err := core.InitSQLite(db); err != nil {
		t.Fatalf("failed to init sqlite: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
		VALUES ('test-user-id', 'test@example.com', 'Test User', 'test-password-hash', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
		INSERT INTO sessions (id, user_id, hash, salt, created_at)
		VALUES ('test-session-id', 'test-user-id', 'hash', 'salt', CURRENT_TIMESTAMP);
	`)
	if err != nil {
		t.Fatalf("failed to seed test user and session: %v", err)
	}

	duckdb, err := sqlx.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	t.Cleanup(func() {
		duckdb.Close()
	})

	return &core.App{
		Sqlite:      db,
		DuckDB:      duckdb,
		JWTSecret:   []byte("test-secret-key-32-bytes-long!"),
		JWTExp:      time.Hour,
		ConfigKV:    &MockKV{data: make(map[string][]byte)},
		DownloadsKv: &MockKV{data: make(map[string][]byte)},
		Logger:      slog.Default(),
	}
}

func createDashboardScopedToken(dashboardId string) *jwt.Token {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId":      "test-user-id",
		"sessionId":   "test-session-id",
		"pwdRev":      core.ComputePasswordRevision("test-password-hash"),
		"dashboardId": dashboardId,
		"exp":         time.Now().Add(time.Hour).Unix(),
	})
}

func createUserToken() *jwt.Token {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId":    "test-user-id",
		"sessionId": "test-session-id",
		"pwdRev":    core.ComputePasswordRevision("test-password-hash"),
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
}

func signToken(token *jwt.Token, secret []byte) string {
	str, _ := token.SignedString(secret)
	return str
}

func newContextWithUser(e *echo.Echo, req *http.Request, rec *httptest.ResponseRecorder, token *jwt.Token) echo.Context {
	actor := &core.Actor{Type: core.ActorUser, ID: "test-user-id"}
	req = req.WithContext(core.ContextWithActor(req.Context(), actor))
	c := e.NewContext(req, rec)
	c.Set("user", token)
	return c
}
