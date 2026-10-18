// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestTasks_DashboardIDRestriction(t *testing.T) {
	e := echo.New()
	app := setupTestApp(t)

	t.Run("CreateTask", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := CreateTask(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := CreateTask(app)
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

	t.Run("GetTask", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodGet, "/api/tasks/task-1", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("task-1")

			h := GetTask(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodGet, "/api/tasks/task-1", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("task-1")

			h := GetTask(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("DeleteTask", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodDelete, "/api/tasks/task-1", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("task-1")

			h := DeleteTask(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodDelete, "/api/tasks/task-1", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("task-1")

			h := DeleteTask(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("SaveTaskContent", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPut, "/api/tasks/task-1/content", bytes.NewReader([]byte(`invalid json`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("task-1")

			h := SaveTaskContent(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPut, "/api/tasks/task-1/content", bytes.NewReader([]byte(`invalid json`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("task-1")

			h := SaveTaskContent(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 (bind error), got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("SaveTaskName", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPut, "/api/tasks/task-1/name", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("task-1")

			h := SaveTaskName(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPut, "/api/tasks/task-1/name", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("task-1")

			h := SaveTaskName(app)
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

	t.Run("RunTask", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPost, "/api/tasks/run", bytes.NewReader([]byte(`{"content":"test"}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := RunTask(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPost, "/api/tasks/run", bytes.NewReader([]byte(`{"content":"test"}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := RunTask(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})
}
