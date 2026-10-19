// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"shaper/server/core"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

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
