// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestListApps_DashboardIDRestriction(t *testing.T) {
	e := echo.New()
	app := setupTestApp(t)

	t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
		token := createDashboardScopedToken("dash-456")

		req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", token)

		h := ListApps(app)
		if err := h(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
		token := createUserToken()

		req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", token)

		h := ListApps(app)
		if err := h(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code == http.StatusUnauthorized {
			t.Errorf("expected non-401 status for user jwt without dashboardId, got %d: %s", rec.Code, rec.Body.String())
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
