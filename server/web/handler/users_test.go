// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestUsers_DashboardIDRestriction(t *testing.T) {
	e := echo.New()
	app := setupTestApp(t)

	t.Run("ListUsers", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := ListUsers(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := ListUsers(app)
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

	t.Run("DeleteUser", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodDelete, "/api/users/user-1", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("user-1")

			h := DeleteUser(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodDelete, "/api/users/user-1", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("user-1")

			h := DeleteUser(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("UpdatePassword", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPut, "/api/users/user-1/password", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("user-1")

			h := UpdatePassword(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPut, "/api/users/user-1/password", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("user-1")

			h := UpdatePassword(app)
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

	t.Run("UpdateName", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPut, "/api/users/user-1/name", bytes.NewReader([]byte(`invalid json`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("user-1")

			h := UpdateName(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPut, "/api/users/user-1/name", bytes.NewReader([]byte(`invalid json`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("user-1")

			h := UpdateName(app)
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

	t.Run("CreateInvite", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPost, "/api/invites", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := CreateInvite(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPost, "/api/invites", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := CreateInvite(app)
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

	t.Run("DeleteInvite", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodDelete, "/api/invites/", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("code")
			c.SetParamValues("")

			h := DeleteInvite(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodDelete, "/api/invites/", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("code")
			c.SetParamValues("")

			h := DeleteInvite(app)
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
}
