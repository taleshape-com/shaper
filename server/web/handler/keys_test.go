// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestKeys_DashboardIDRestriction(t *testing.T) {
	e := echo.New()
	app := setupTestApp(t)

	t.Run("ListAPIKeys", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodGet, "/api/keys", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := ListAPIKeys(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodGet, "/api/keys", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := ListAPIKeys(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("CreateAPIKey", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPost, "/api/keys", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := CreateAPIKey(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPost, "/api/keys", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := CreateAPIKey(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 (validation error), got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("UpdateAPIKeyPermissions", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPut, "/api/keys/key-1", bytes.NewReader([]byte(`invalid json`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("key-1")

			h := UpdateAPIKeyPermissions(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPut, "/api/keys/key-1", bytes.NewReader([]byte(`invalid json`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("key-1")

			h := UpdateAPIKeyPermissions(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 (binding error), got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("DeleteAPIKey", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodDelete, "/api/keys/key-1", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("key-1")

			h := DeleteAPIKey(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodDelete, "/api/keys/key-1", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("key-1")

			h := DeleteAPIKey(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})
}
