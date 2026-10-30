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

	jwt "github.com/golang-jwt/jwt/v5"
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

func TestRoutes_DeployDashboardIDRestriction(t *testing.T) {
	e := echo.New()
	secret := []byte("test-secret-key-32-bytes-long!")
	app := &core.App{
		LoginRequired: true,
		NoTasks:       true,
		JWTSecret:     secret,
		JWTExp:        time.Hour,
	}

	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}

	routes(e, app, mockFS, time.Now(), "", "", "http://localhost:8080", "2006-01-02")

	createJWT := func(claims jwt.MapClaims) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString(secret)
		require.NoError(t, err)
		return signed
	}

	t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
		tokenStr := createJWT(jwt.MapClaims{
			"userId":      "user-123",
			"dashboardId": "dash-456",
			"exp":         time.Now().Add(time.Hour).Unix(),
		})

		req := httptest.NewRequest(http.MethodPost, "/api/deploy", strings.NewReader(`{"apps":[]}`))
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		var res map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, "Unauthorized", res["error"])
	})

	t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
		tokenStr := createJWT(jwt.MapClaims{
			"userId": "user-123",
			"exp":    time.Now().Add(time.Hour).Unix(),
		})

		req := httptest.NewRequest(http.MethodPost, "/api/deploy", strings.NewReader(`{"apps":[]}`))
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var res map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, "apps array is required", res["error"])
	})
}

func TestRoutes_PublicActorRestrictions(t *testing.T) {
	sdb, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer sdb.Close()
	require.NoError(t, core.InitSQLite(sdb))

	ddb, err := sqlx.Open("duckdb", "")
	require.NoError(t, err)
	defer ddb.Close()

	_, err = sdb.Exec(`
		INSERT INTO apps (id, name, content, created_at, updated_at, visibility, type)
		VALUES ('test-dash-id', 'Test Dashboard', 'SELECT 1;', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'public', 'dashboard');
	`)
	require.NoError(t, err)

	e := echo.New()
	secret := []byte("test-secret-key-32-bytes-long!")
	app := &core.App{
		LoginRequired: true,
		NoTasks:       false,
		JWTSecret:     secret,
		JWTExp:        time.Hour,
		Sqlite:        sdb,
		DuckDB:        ddb,
	}

	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}

	routes(e, app, mockFS, time.Now(), "", "", "http://localhost:8080", "2006-01-02")

	createJWT := func(claims jwt.MapClaims) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString(secret)
		require.NoError(t, err)
		return signed
	}

	publicToken := createJWT(jwt.MapClaims{
		"dashboardId": "test-dash-id",
		"public":      "public",
		"exp":         time.Now().Add(time.Hour).Unix(),
	})

	userToken := createJWT(jwt.MapClaims{
		"userId": "user-123",
		"exp":    time.Now().Add(time.Hour).Unix(),
	})

	t.Run("allowed route GET /dashboards/:id with public actor JWT", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/dashboards/test-dash-id", nil)
		req.Header.Set("Authorization", "Bearer "+publicToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("allowed route GET /api/dashboards/:id/download/:filename with public actor JWT", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/dashboards/test-dash-id/download/export.csv", nil)
		req.Header.Set("Authorization", "Bearer "+publicToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		// Must not be rejected with 401 Unauthorized by public actor route check
		assert.NotEqual(t, http.StatusUnauthorized, rec.Code)
	})

	forbiddenEndpoints := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"GET /api/dashboards/:id/info", http.MethodGet, "/api/dashboards/test-dash-id/info", ""},
		{"POST /api/dashboards/:id/query", http.MethodPost, "/api/dashboards/test-dash-id/query", `{"query":"SELECT 1"}`},
		{"POST /api/dashboards/:id/name", http.MethodPost, "/api/dashboards/test-dash-id/name", `{"name":"New Name"}`},
		{"POST /api/dashboards/:id/visibility", http.MethodPost, "/api/dashboards/test-dash-id/visibility", `{"visibility":"private"}`},
		{"POST /api/dashboards/:id/password", http.MethodPost, "/api/dashboards/test-dash-id/password", `{"password":"secret"}`},
		{"DELETE /api/dashboards/:id", http.MethodDelete, "/api/dashboards/test-dash-id", ""},
		{"GET /api/apps", http.MethodGet, "/api/apps", ""},
		{"POST /api/deploy", http.MethodPost, "/api/deploy", `{"apps":[]}`},
		{"POST /api/validate", http.MethodPost, "/api/validate", `{"apps":[]}`},
		{"GET /api/schema", http.MethodGet, "/api/schema", ""},
		{"POST /api/dashboards", http.MethodPost, "/api/dashboards", `{"name":"Test"}`},
		{"GET /api/users", http.MethodGet, "/api/users", ""},
		{"GET /api/version", http.MethodGet, "/api/version", ""},
		{"GET /api/keys", http.MethodGet, "/api/keys", ""},
		{"POST /api/folders", http.MethodPost, "/api/folders", `{"name":"Folder"}`},
		{"POST /api/admin/reset-jwt-secret", http.MethodPost, "/api/admin/reset-jwt-secret", ""},
	}

	for _, tc := range forbiddenEndpoints {
		t.Run("public actor JWT rejected on "+tc.name, func(t *testing.T) {
			var bodyReader *strings.Reader
			if tc.body != "" {
				bodyReader = strings.NewReader(tc.body)
			} else {
				bodyReader = strings.NewReader("")
			}
			req := httptest.NewRequest(tc.method, tc.path, bodyReader)
			req.Header.Set("Authorization", "Bearer "+publicToken)
			if tc.body != "" {
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			var res map[string]string
			err := json.Unmarshal(rec.Body.Bytes(), &res)
			assert.NoError(t, err)
			assert.Equal(t, "Unauthorized", res["error"])
		})
	}

	dashboardScopedUserToken := createJWT(jwt.MapClaims{
		"userId":      "user-123",
		"dashboardId": "test-dash-id",
		"exp":         time.Now().Add(time.Hour).Unix(),
	})

	t.Run("dashboard-scoped user JWT rejected on forbidden routes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
		req.Header.Set("Authorization", "Bearer "+dashboardScopedUserToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		var res map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, "Unauthorized", res["error"])
	})

	t.Run("dashboard-scoped user JWT allowed on GET /dashboards/:id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/dashboards/test-dash-id", nil)
		req.Header.Set("Authorization", "Bearer "+dashboardScopedUserToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("regular user JWT is allowed on general routes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestSetActor_InvalidOrMissingJWT(t *testing.T) {
	app := &core.App{
		LoginRequired: true,
	}

	nextCalled := false
	dummyNext := func(c echo.Context) error {
		nextCalled = true
		return c.NoContent(http.StatusOK)
	}

	t.Run("missing user in context returns 401", func(t *testing.T) {
		nextCalled = false
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := SetActor(app)(dummyNext)(c)
		assert.NoError(t, err)
		assert.False(t, nextCalled)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		var res map[string]string
		err = json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, "Unauthorized", res["error"])
	})

	t.Run("user is not *jwt.Token returns 401", func(t *testing.T) {
		nextCalled = false
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", "not-a-token")

		err := SetActor(app)(dummyNext)(c)
		assert.NoError(t, err)
		assert.False(t, nextCalled)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		var res map[string]string
		err = json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, "Unauthorized", res["error"])
	})

	t.Run("token with no actor claims returns 401", func(t *testing.T) {
		nextCalled = false
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		token := &jwt.Token{
			Claims: jwt.MapClaims{"foo": "bar"},
		}
		c.Set("user", token)

		err := SetActor(app)(dummyNext)(c)
		assert.NoError(t, err)
		assert.False(t, nextCalled)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		var res map[string]string
		err = json.Unmarshal(rec.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, "Unauthorized", res["error"])
	})
}

func TestRoutes_NoAPIKeys(t *testing.T) {
	sdb, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer sdb.Close()
	require.NoError(t, core.InitSQLite(sdb))

	// Insert a valid API key into SQLite
	apiKeyID := "testkeyid123"
	apiKeyToken := "shaperkey." + apiKeyID + ".testsuffix1234567890123456789012"
	salt := "testsalt123456789012345678901234"
	// Compute HMAC-SHA256 matching core.ValidateAPIKey
	ok, err := core.ValidateAPIKey(sdb, context.Background(), apiKeyToken)
	require.NoError(t, err)
	require.False(t, ok)

	// Insert key using core.HandleCreateAPIKey
	payloadBytes, err := json.Marshal(core.CreateAPIKeyPayload{
		ID:          apiKeyID,
		Timestamp:   time.Now(),
		Name:        "Test Key",
		Hash:        "560f0c2fb9ab780f657996401d9e4d3cf73475c8dc5c5d87d5fd52882f9cf277",
		Salt:        salt,
		Permissions: core.AllPermissions,
		CreatedBy:   "user:user-123",
	})
	require.NoError(t, err)
	require.True(t, core.HandleCreateAPIKey(&core.App{Sqlite: sdb}, payloadBytes))

	secret := []byte("test-secret-key-32-bytes-long!")
	app := &core.App{
		LoginRequired: true,
		NoTasks:       true,
		NoAPIKeys:     true,
		JWTSecret:     secret,
		JWTExp:        time.Hour,
		Sqlite:        sdb,
	}

	e := echo.New()
	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}
	routes(e, app, mockFS, time.Now(), "", "", "http://localhost:8080", "2006-01-02")

	createJWT := func(claims jwt.MapClaims) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString(secret)
		require.NoError(t, err)
		return signed
	}

	apiKeyJWT := createJWT(jwt.MapClaims{
		"apiKeyId": apiKeyID,
		"exp":      time.Now().Add(time.Hour).Unix(),
	})

	t.Run("system config reports apiKeysEnabled false", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/system/config", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var res map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		require.NoError(t, err)
		assert.Equal(t, false, res["apiKeysEnabled"])
	})

	t.Run("key management and API-key-only endpoints are not registered", func(t *testing.T) {
		registeredPaths := make(map[string]bool)
		for _, r := range e.Routes() {
			registeredPaths[r.Method+" "+r.Path] = true
		}
		disabledRoutes := []string{
			http.MethodGet + " /api/keys",
			http.MethodPost + " /api/keys",
			http.MethodPost + " /api/keys/:id/permissions",
			http.MethodDelete + " /api/keys/:id",
			http.MethodGet + " /metrics",
			http.MethodPost + " /api/data/:table_name",
			http.MethodPost + " /api/sql",
			http.MethodPost + " /api/download/:filename",
		}
		for _, route := range disabledRoutes {
			assert.False(t, registeredPaths[route], "route %s should not be registered when NoAPIKeys is true", route)
		}
	})

	t.Run("token exchange with API key is rejected with 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/token", strings.NewReader(`{"token":"`+apiKeyToken+`"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("token refresh with API key JWT is rejected with 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/token", strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer "+apiKeyJWT)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("API key JWT is rejected by SetActor with 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
		req.Header.Set("Authorization", "Bearer "+apiKeyJWT)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("direct API key token is rejected on API routes", func(t *testing.T) {
		for _, path := range []string{"/api/apps", "/api/schema"} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Authorization", "Bearer "+apiKeyToken)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code, "GET %s with API key should be 401", path)
		}
	})

	t.Run("direct API key token is rejected even when LoginRequired is false", func(t *testing.T) {
		noAuthApp := *app
		noAuthApp.LoginRequired = false
		eNoAuth := echo.New()
		routes(eNoAuth, &noAuthApp, mockFS, time.Now(), "", "", "http://localhost:8080", "2006-01-02")

		for _, path := range []string{"/api/apps", "/api/schema"} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Authorization", "Bearer "+apiKeyToken)
			rec := httptest.NewRecorder()
			eNoAuth.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code, "GET %s with API key should be 401 when LoginRequired is false", path)
		}
	})
}

func TestRoutes_NoIngest(t *testing.T) {
	sdb, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer sdb.Close()
	require.NoError(t, core.InitSQLite(sdb))

	app := &core.App{
		LoginRequired: false,
		NoTasks:       true,
		NoIngest:      true,
		JWTSecret:     []byte("test-secret-key-32-bytes-long!"),
		JWTExp:        time.Hour,
		Sqlite:        sdb,
	}

	e := echo.New()
	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}
	routes(e, app, mockFS, time.Now(), "", "", "http://localhost:8080", "2006-01-02")

	t.Run("system config reports ingestEnabled false", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/system/config", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var res map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		require.NoError(t, err)
		assert.Equal(t, false, res["ingestEnabled"])
	})

	t.Run("POST /api/data/:table_name route is not registered", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/data/events", strings.NewReader(`{"foo":"bar"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.NotEqual(t, http.StatusAccepted, rec.Code)
		assert.NotEqual(t, http.StatusOK, rec.Code)
	})
}

func TestRoutes_SetupRejectedWhenAdminEmailSet(t *testing.T) {
	sdb, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer sdb.Close()
	require.NoError(t, core.InitSQLite(sdb))

	app := &core.App{
		LoginRequired: true,
		AdminEmail:    "admin@example.com",
		AdminPassword: "secret",
		NoTasks:       true,
		JWTSecret:     []byte("test-secret-key-32-bytes-long!"),
		JWTExp:        time.Hour,
		Sqlite:        sdb,
	}

	e := echo.New()
	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}
	routes(e, app, mockFS, time.Now(), "", "", "http://localhost:8080", "2006-01-02")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/setup", strings.NewReader(`{"email":"attacker@example.com","password":"pass","name":"Attacker"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	var res map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	assert.Equal(t, "User setup already completed", res["error"])
}

