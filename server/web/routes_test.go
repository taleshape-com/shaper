// SPDX-License-Identifier: MPL-2.0

package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shaper/server/core"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	_ "modernc.org/sqlite"
)

func TestLoginRateLimiter_AllowsUpToBurst(t *testing.T) {
	e := echo.New()
	limiter := LoginRateLimiter()

	e.POST("/api/login", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, limiter)

	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code, "request %d should succeed", i)
	}
}

func TestLoginAccountRateLimiterStore(t *testing.T) {
	store := LoginAccountRateLimiterStore()
	require.NotNil(t, store)

	// Burst is 5 per account
	for i := 1; i <= 5; i++ {
		allowed, err := store.Allow("192.168.1.1:alice@example.com")
		assert.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i)
	}

	// 6th request for same account should be denied
	allowed, err := store.Allow("192.168.1.1:alice@example.com")
	assert.NoError(t, err)
	assert.False(t, allowed, "6th request should be denied")

	// Different account from same IP should be allowed
	allowed, err = store.Allow("192.168.1.1:bob@example.com")
	assert.NoError(t, err)
	assert.True(t, allowed, "different account on same IP should be allowed")
}

func TestLoginRateLimiter_BlocksExcessRequests(t *testing.T) {
	e := echo.New()
	limiter := LoginRateLimiter()

	e.POST("/api/login", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, limiter)

	// Consume the 30 allowed burst requests
	for i := 1; i <= 30; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
		req.RemoteAddr = "192.168.1.101:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 31st request should be rejected with 429 Too Many Requests
	req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
	req.RemoteAddr = "192.168.1.101:12345"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "60", rec.Header().Get("Retry-After"))

	var res map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Too many login attempts, please try again later", res["error"])
}

func TestLoginRateLimiter_IndependentIPs(t *testing.T) {
	e := echo.New()
	limiter := LoginRateLimiter()

	e.POST("/api/login", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, limiter)

	// Exhaust IP 1 with 30 requests
	for i := 1; i <= 30; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// IP 1 is blocked on 31st attempt
	req1 := httptest.NewRequest(http.MethodPost, "/api/login", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusTooManyRequests, rec1.Code)

	// IP 2 is not affected and succeeds
	req2 := httptest.NewRequest(http.MethodPost, "/api/login", nil)
	req2.RemoteAddr = "10.0.0.2:12345"
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestLoginRateLimiterWithConfig_CustomLimits(t *testing.T) {
	e := echo.New()
	// Custom limiter with burst of 2
	limiter := LoginRateLimiterWithConfig(rate.Limit(1.0/60.0), 2, time.Minute)

	e.POST("/api/login", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, limiter)

	// 2 requests allowed
	for i := 1; i <= 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
		req.RemoteAddr = "172.16.0.1:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 3rd request rejected
	req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
	req.RemoteAddr = "172.16.0.1:12345"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRoutes_LoginRateLimiterIntegration(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			deleted_at DATETIME
		);
		INSERT INTO users (id, email, password_hash)
		VALUES ('user-alice', 'alice@example.com', '$2a$10$abcdefghijklmnopqrstuvwxyz012345678901234567890123456789');
		INSERT INTO users (id, email, password_hash)
		VALUES ('user-bob', 'bob@example.com', '$2a$10$abcdefghijklmnopqrstuvwxyz012345678901234567890123456789');
	`)
	require.NoError(t, err)

	e := echo.New()
	app := &core.App{
		LoginRequired: true,
		NoTasks:       true,
		JWTSecret:     []byte("test-secret-key-32-bytes-long!"),
		JWTExp:        time.Hour,
		Sqlite:        db,
	}

	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}

	routes(e, app, mockFS, time.Now(), "", "", "http://localhost:8080", "2006-01-02")

	// Send 5 invalid login attempts for alice@example.com from IP 192.168.2.1
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"email":"alice@example.com","password":"wrong"}`))
		req.RemoteAddr = "192.168.2.1:12345"
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		// Should reach handler and return 400 (no db configured in app), but NOT 429
		assert.Equal(t, http.StatusBadRequest, rec.Code, "request %d should return 400", i)
	}

	// 6th attempt for alice@example.com from same IP should be rate limited with 429
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"email":"alice@example.com","password":"wrong"}`))
	req.RemoteAddr = "192.168.2.1:12345"
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	var res map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Too many login attempts, please try again later", res["error"])

	// Colleagues on the same company IP (bob@example.com) should NOT be blocked
	reqBob := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"email":"bob@example.com","password":"wrong"}`))
	reqBob.RemoteAddr = "192.168.2.1:12345"
	reqBob.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recBob := httptest.NewRecorder()
	e.ServeHTTP(recBob, reqBob)
	assert.Equal(t, http.StatusBadRequest, recBob.Code, "different user on same IP should not be rate limited")

	// Alice trying from a different IP should NOT be blocked
	reqAliceOtherIP := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"email":"alice@example.com","password":"wrong"}`))
	reqAliceOtherIP.RemoteAddr = "192.168.2.2:12345"
	reqAliceOtherIP.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recAliceOtherIP := httptest.NewRecorder()
	e.ServeHTTP(recAliceOtherIP, reqAliceOtherIP)
	assert.Equal(t, http.StatusBadRequest, recAliceOtherIP.Code, "same user on different IP should not be rate limited")
}

func TestPublicAuthRateLimiter(t *testing.T) {
	e := echo.New()
	limiter := PublicAuthRateLimiter()

	e.POST("/api/auth/public", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, limiter)

	// Burst is 25
	for i := 1; i <= 25; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/public", nil)
		req.RemoteAddr = "192.168.3.1:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should succeed", i)
	}

	// 26th request rejected
	req := httptest.NewRequest(http.MethodPost, "/api/auth/public", nil)
	req.RemoteAddr = "192.168.3.1:12345"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "60", rec.Header().Get("Retry-After"))
	var res map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Too many requests, please try again later", res["error"])
}

func TestPublicAuthRateLimiterStore(t *testing.T) {
	store := PublicAuthRateLimiterStore()
	require.NotNil(t, store)

	// Burst is 25
	for i := 1; i <= 25; i++ {
		allowed, err := store.Allow("192.168.10.1")
		assert.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i)
	}

	// 26th request should be denied
	allowed, err := store.Allow("192.168.10.1")
	assert.NoError(t, err)
	assert.False(t, allowed, "26th request should be denied")
}

func TestInviteRateLimiter_SharedAcrossEndpoints(t *testing.T) {
	e := echo.New()
	limiter := InviteRateLimiter()

	e.GET("/api/invites/:code", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, limiter)
	e.POST("/api/invites/:code/claim", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, limiter)

	// Send 10 GET requests and 5 POST requests (total 15 = burst for testcode)
	for i := 1; i <= 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/invites/testcode", nil)
		req.RemoteAddr = "192.168.4.1:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/invites/testcode/claim", nil)
		req.RemoteAddr = "192.168.4.1:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 16th request for testcode should be blocked by rate limit
	req := httptest.NewRequest(http.MethodGet, "/api/invites/testcode", nil)
	req.RemoteAddr = "192.168.4.1:12345"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "60", rec.Header().Get("Retry-After"))
	var res map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Too many invite requests, please try again later", res["error"])

	// Another invite code from same IP should NOT be blocked
	reqOther := httptest.NewRequest(http.MethodGet, "/api/invites/othercode", nil)
	reqOther.RemoteAddr = "192.168.4.1:12345"
	recOther := httptest.NewRecorder()
	e.ServeHTTP(recOther, reqOther)
	assert.Equal(t, http.StatusOK, recOther.Code)
}

func TestDownloadRateLimiter(t *testing.T) {
	e := echo.New()
	limiter := DownloadRateLimiter()

	e.GET("/api/download/:key/:filename", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, limiter)

	// Burst is 10 for unauthenticated requests
	for i := 1; i <= 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/download/sample-key/report.pdf", nil)
		req.RemoteAddr = "192.168.5.1:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 11th request rejected
	req := httptest.NewRequest(http.MethodGet, "/api/download/sample-key/report.pdf", nil)
	req.RemoteAddr = "192.168.5.1:12345"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "60", rec.Header().Get("Retry-After"))
	var res map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Too many download requests, please try again later", res["error"])
}

func TestDownloadRateLimiter_AuthenticatedActorsSeparateBuckets(t *testing.T) {
	e := echo.New()
	limiter := DownloadRateLimiter()

	e.GET("/api/test-download", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, limiter)

	// Exhaust actor1 on IP 192.168.5.2 (10 requests)
	for i := 1; i <= 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/test-download", nil)
		req.RemoteAddr = "192.168.5.2:12345"
		ctx := core.ContextWithActor(req.Context(), &core.Actor{Type: core.ActorUser, ID: "user-1"})
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 11th request for actor1 is blocked
	req1 := httptest.NewRequest(http.MethodGet, "/api/test-download", nil)
	req1.RemoteAddr = "192.168.5.2:12345"
	req1 = req1.WithContext(core.ContextWithActor(req1.Context(), &core.Actor{Type: core.ActorUser, ID: "user-1"}))
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusTooManyRequests, rec1.Code)

	// actor2 from the SAME IP is NOT blocked
	req2 := httptest.NewRequest(http.MethodGet, "/api/test-download", nil)
	req2.RemoteAddr = "192.168.5.2:12345"
	req2 = req2.WithContext(core.ContextWithActor(req2.Context(), &core.Actor{Type: core.ActorUser, ID: "user-2"}))
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code, "different actor on same IP should not be blocked")
}

func TestRoutes_PublicAuthRateLimiterIntegration(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE apps (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			created_by TEXT,
			updated_by TEXT,
			visibility TEXT,
			type TEXT NOT NULL,
			password_hash TEXT,
			folder_id TEXT
		);
		INSERT INTO apps (id, name, content, created_at, updated_at, visibility, type)
		VALUES ('public-dash', 'Public Dashboard', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'public', 'dashboard');

		INSERT INTO apps (id, name, content, created_at, updated_at, visibility, type, password_hash)
		VALUES ('protected-dash', 'Protected Dashboard', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'password-protected', 'dashboard', '$2a$10$abcdefghijklmnopqrstuvwxyz012345678901234567890123456789');

		INSERT INTO apps (id, name, content, created_at, updated_at, visibility, type, password_hash)
		VALUES ('protected-dash-2', 'Protected Dashboard 2', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'password-protected', 'dashboard', '$2a$10$abcdefghijklmnopqrstuvwxyz012345678901234567890123456789');
	`)
	require.NoError(t, err)

	e := echo.New()
	app := &core.App{
		LoginRequired: true,
		NoTasks:       true,
		JWTSecret:     []byte("test-secret-key-32-bytes-long!"),
		JWTExp:        time.Hour,
		Sqlite:        db,
	}

	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}

	routes(e, app, mockFS, time.Now(), "", "", "http://localhost:8080", "2006-01-02")

	// 1. Completely public dashboard: should NOT be rate limited even with 30 requests (e.g. documentation embeds)
	for i := 1; i <= 30; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/public", strings.NewReader(`{"dashboardId":"public-dash"}`))
		req.RemoteAddr = "192.168.6.1:12345"
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "public dashboard request %d should succeed with 200", i)
	}

	// 2. Password-protected dashboard: burst is 25 for password attempts
	for i := 1; i <= 25; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/public", strings.NewReader(`{"dashboardId":"protected-dash","password":"wrong-password"}`))
		req.RemoteAddr = "192.168.6.1:12345"
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "password request %d should return 401 Unauthorized", i)
	}

	// 26th attempt on password-protected dashboard should return 429
	req := httptest.NewRequest(http.MethodPost, "/api/auth/public", strings.NewReader(`{"dashboardId":"protected-dash","password":"wrong-password"}`))
	req.RemoteAddr = "192.168.6.1:12345"
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "60", rec.Header().Get("Retry-After"))
	var res map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Too many requests, please try again later", res["error"])

	// Another password-protected dashboard from the same IP should NOT be blocked
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/public", strings.NewReader(`{"dashboardId":"protected-dash-2","password":"wrong-password"}`))
	req2.RemoteAddr = "192.168.6.1:12345"
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusUnauthorized, rec2.Code, "different protected dashboard on same IP should not be blocked")
}

type mockKV struct {
	jetstream.KeyValue
}

func (m *mockKV) Get(ctx context.Context, key string) (jetstream.KeyValueEntry, error) {
	return nil, jetstream.ErrKeyNotFound
}

func TestRoutes_InviteRateLimiterIntegration(t *testing.T) {
	e := echo.New()
	app := &core.App{
		LoginRequired: true,
		NoTasks:       true,
		JWTSecret:     []byte("test-secret-key-32-bytes-long!"),
		JWTExp:        time.Hour,
	}

	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}

	routes(e, app, mockFS, time.Now(), "", "", "http://localhost:8080", "2006-01-02")

	// Burst is 15 for invite routes. Send 15 POST requests with invalid body to consume bucket
	for i := 1; i <= 15; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/invites/dummycode/claim", strings.NewReader("not-json"))
		req.RemoteAddr = "192.168.8.1:12345"
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "request %d should return 400", i)
	}

	// 16th attempt (sent to GET endpoint) should be blocked with 429 because the limiter is shared for dummycode!
	req := httptest.NewRequest(http.MethodGet, "/api/invites/dummycode", nil)
	req.RemoteAddr = "192.168.8.1:12345"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	var res map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Too many invite requests, please try again later", res["error"])

	// Another invite code from the same IP should NOT be blocked
	reqOther := httptest.NewRequest(http.MethodPost, "/api/invites/othercode/claim", strings.NewReader("not-json"))
	reqOther.RemoteAddr = "192.168.8.1:12345"
	reqOther.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recOther := httptest.NewRecorder()
	e.ServeHTTP(recOther, reqOther)
	assert.Equal(t, http.StatusBadRequest, recOther.Code)
}

func TestRoutes_DownloadRateLimiterIntegration(t *testing.T) {
	e := echo.New()
	app := &core.App{
		LoginRequired: true,
		NoTasks:       true,
		JWTSecret:     []byte("test-secret-key-32-bytes-long!"),
		JWTExp:        time.Hour,
		DownloadsKv:   &mockKV{},
	}

	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}

	routes(e, app, mockFS, time.Now(), "", "", "http://localhost:8080", "2006-01-02")

	// Send 10 requests to /api/download/:key/:filename (burst = 10)
	for i := 1; i <= 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/download/dummy-key/test.pdf", nil)
		req.RemoteAddr = "192.168.7.1:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		// Reaches handler and returns 404 (key not found), but NOT 429
		assert.Equal(t, http.StatusNotFound, rec.Code, "request %d should return 404", i)
	}

	// 11th attempt should return 429
	req := httptest.NewRequest(http.MethodGet, "/api/download/dummy-key/test.pdf", nil)
	req.RemoteAddr = "192.168.7.1:12345"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	var res map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Too many download requests, please try again later", res["error"])
}

