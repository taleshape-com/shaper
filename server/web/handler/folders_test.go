// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestFolders_DashboardIDRestriction(t *testing.T) {
	e := echo.New()
	app := setupTestApp(t)

	t.Run("CreateFolder", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPost, "/api/folders", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := CreateFolder(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPost, "/api/folders", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := CreateFolder(app)
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

	t.Run("DeleteFolder", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodDelete, "/api/folders/", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("")

			h := DeleteFolder(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodDelete, "/api/folders/", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("")

			h := DeleteFolder(app)
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

	t.Run("RenameFolder", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPut, "/api/folders/", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("")

			h := RenameFolder(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPut, "/api/folders/", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("")

			h := RenameFolder(app)
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

	t.Run("MoveItems", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPost, "/api/move", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := MoveItems(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPost, "/api/move", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := MoveItems(app)
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
