// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shaper/server/api"
	"shaper/server/core"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func TestDeploy_NoTasksRejectsTaskOperations(t *testing.T) {
	e := echo.New()
	app := setupTestApp(t)
	app.NoTasks = true

	token := createUserToken()

	t.Run("deploy create task rejected when tasks disabled", func(t *testing.T) {
		body := `{"apps":[{"operation":"create","type":"task","data":{"name":"audit-task-ddl","path":"/","content":"SELECT 'init'::SCHEDULE;\nCREATE TABLE audit_task_probe AS SELECT 42 AS answer;"}}]}`
		req := httptest.NewRequest(http.MethodPost, "/api/deploy", bytes.NewReader([]byte(body)))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := newContextWithUser(e, req, rec, token)

		h := Deploy(app)
		if err := h(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
		}
		var res map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if !strings.Contains(res["error"], "tasks are disabled") {
			t.Errorf("expected error to mention tasks are disabled, got %q", res["error"])
		}

		// Verify task was not inserted and DDL did not execute
		var count int
		if err := app.Sqlite.Get(&count, "SELECT COUNT(*) FROM apps WHERE type = 'task'"); err != nil {
			t.Fatalf("failed querying sqlite: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 tasks in DB, got %d", count)
		}

		var tableCount int
		if err := app.DuckDB.Get(&tableCount, "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'audit_task_probe'"); err != nil {
			t.Fatalf("failed querying duckdb: %v", err)
		}
		if tableCount != 0 {
			t.Errorf("expected audit_task_probe not to exist, got count %d", tableCount)
		}
	})

	t.Run("ProcessDeployOperation rejects create, update, and delete for task when NoTasks is true", func(t *testing.T) {
		ctx := core.ContextWithActor(context.Background(), &core.Actor{Type: core.ActorUser, ID: "user-123"})
		id := "task-123"
		name := "my-task"
		path := "/"
		content := "SELECT 1;"

		for _, op := range []string{"create", "update", "delete"} {
			_, err := ProcessDeployOperation(ctx, app, 0, api.AppRequest{
				Operation: op,
				Type:      "task",
				Data: api.DashboardData{
					ID:      &id,
					Name:    &name,
					Path:    &path,
					Content: &content,
				},
			})
			if err == nil {
				t.Errorf("expected error for operation %q when NoTasks is true, got nil", op)
			} else if !strings.Contains(err.Error(), "tasks are disabled") {
				t.Errorf("expected 'tasks are disabled' error for operation %q, got %v", op, err)
			}
		}
	})
}

func TestDeploy_DashboardIDRestriction(t *testing.T) {
	e := echo.New()
	app := &core.App{}

	t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"userId":      "user-123",
			"dashboardId": "dash-456",
			"exp":         time.Now().Add(time.Hour).Unix(),
		})

		req := httptest.NewRequest(http.MethodPost, "/api/deploy", bytes.NewReader([]byte(`{"apps":[]}`)))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", token)

		h := Deploy(app)
		if err := h(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"userId": "user-123",
			"exp":    time.Now().Add(time.Hour).Unix(),
		})

		req := httptest.NewRequest(http.MethodPost, "/api/deploy", bytes.NewReader([]byte(`{"apps":[]}`)))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", token)

		h := Deploy(app)
		if err := h(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code == http.StatusUnauthorized {
			t.Errorf("expected non-401 status for valid user jwt, got %d: %s", rec.Code, rec.Body.String())
		}
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 (apps array required), got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("api key auth without user in context is not rejected with 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/deploy", bytes.NewReader([]byte(`{"apps":[]}`)))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		h := Deploy(app)
		if err := h(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code == http.StatusUnauthorized {
			t.Errorf("expected non-401 status for api key request, got %d: %s", rec.Code, rec.Body.String())
		}
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 (apps array required), got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
