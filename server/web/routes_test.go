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

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
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

func TestLoginRateLimiter_BlocksExcessRequests(t *testing.T) {
	e := echo.New()
	limiter := LoginRateLimiter()

	e.POST("/api/login", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, limiter)

	// Consume the 5 allowed burst requests
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
		req.RemoteAddr = "192.168.1.101:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 6th request should be rejected with 429 Too Many Requests
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

	// Exhaust IP 1
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// IP 1 is blocked on 6th attempt
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

	// Send 5 invalid login attempts from same IP
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader("not-json"))
		req.RemoteAddr = "192.168.2.1:12345"
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		// Should reach handler and return 400 (invalid request body), but NOT 429
		assert.Equal(t, http.StatusBadRequest, rec.Code, "request %d should return 400", i)
	}

	// 6th attempt should be rate limited with 429
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader("not-json"))
	req.RemoteAddr = "192.168.2.1:12345"
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	var res map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Too many login attempts, please try again later", res["error"])
}

func TestPublicAuthRateLimiter(t *testing.T) {
	e := echo.New()
	limiter := PublicAuthRateLimiter()

	e.POST("/api/auth/public", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, limiter)

	// Burst is 10
	for i := 1; i <= 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/public", nil)
		req.RemoteAddr = "192.168.3.1:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should succeed", i)
	}

	// 11th request rejected
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

func TestInviteRateLimiter_SharedAcrossEndpoints(t *testing.T) {
	e := echo.New()
	limiter := InviteRateLimiter()

	e.GET("/api/invites/:code", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, limiter)
	e.POST("/api/invites/:code/claim", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, limiter)

	// Send 3 GET requests and 2 POST requests (total 5 = burst)
	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/invites/testcode", nil)
		req.RemoteAddr = "192.168.4.1:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
	for i := 1; i <= 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/invites/testcode/claim", nil)
		req.RemoteAddr = "192.168.4.1:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 6th request (either GET or POST) should be blocked by rate limit
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
}

func TestDownloadRateLimiter(t *testing.T) {
	e := echo.New()
	limiter := DownloadRateLimiter()

	e.GET("/api/download/:key/:filename", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, limiter)

	// Burst is 5
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/download/sample-key/report.pdf", nil)
		req.RemoteAddr = "192.168.5.1:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 6th request rejected
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

func TestRoutes_PublicAuthRateLimiterIntegration(t *testing.T) {
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

	// Burst is 10 for public auth
	for i := 1; i <= 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/public", strings.NewReader("not-json"))
		req.RemoteAddr = "192.168.6.1:12345"
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "request %d should return 400", i)
	}

	// 11th attempt should return 429
	req := httptest.NewRequest(http.MethodPost, "/api/auth/public", strings.NewReader("not-json"))
	req.RemoteAddr = "192.168.6.1:12345"
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	var res map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Too many requests, please try again later", res["error"])
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

	// Burst is 5 for invite routes. Send 5 POST requests with invalid body to consume bucket
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/invites/dummycode/claim", strings.NewReader("not-json"))
		req.RemoteAddr = "192.168.8.1:12345"
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "request %d should return 400", i)
	}

	// 6th attempt (sent to GET endpoint) should be blocked with 429 because the limiter is shared!
	req := httptest.NewRequest(http.MethodGet, "/api/invites/dummycode", nil)
	req.RemoteAddr = "192.168.8.1:12345"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	var res map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Too many invite requests, please try again later", res["error"])
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

	// Send 5 requests to /api/download/:key/:filename (burst = 5)
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/download/dummy-key/test.pdf", nil)
		req.RemoteAddr = "192.168.7.1:12345"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		// Reaches handler and returns 404 (key not found), but NOT 429
		assert.Equal(t, http.StatusNotFound, rec.Code, "request %d should return 404", i)
	}

	// 6th attempt should return 429
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

