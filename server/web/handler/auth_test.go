// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shaper/server/core"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func TestTokenAuth_DashboardIDRestriction(t *testing.T) {
	e := echo.New()
	secret := []byte("test-secret-key")
	app := &core.App{
		JWTSecret: secret,
		JWTExp:    time.Hour,
	}

	// Helper to create a signed JWT
	createJWT := func(claims jwt.MapClaims) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString(secret)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}
		return signed
	}

	// Helper to parse returned JWT
	parseJWT := func(tokenStr string) jwt.MapClaims {
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			return secret, nil
		})
		if err != nil || !token.Valid {
			t.Fatalf("failed to parse returned token: %v", err)
		}
		return token.Claims.(jwt.MapClaims)
	}

	t.Run("jwt header without dashboardId is allowed", func(t *testing.T) {
		tokenStr := createJWT(jwt.MapClaims{
			"userId": "user-123",
			"exp":    time.Now().Add(time.Hour).Unix(),
		})

		req := httptest.NewRequest(http.MethodPost, "/api/auth/token", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := TokenAuth(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("jwt header with dashboardId is rejected", func(t *testing.T) {
		tokenStr := createJWT(jwt.MapClaims{
			"userId":      "user-123",
			"dashboardId": "dash-456",
			"exp":         time.Now().Add(time.Hour).Unix(),
		})

		req := httptest.NewRequest(http.MethodPost, "/api/auth/token", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := TokenAuth(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("jwt in body with dashboardId is rejected", func(t *testing.T) {
		tokenStr := createJWT(jwt.MapClaims{
			"userId":      "user-123",
			"dashboardId": "dash-456",
			"exp":         time.Now().Add(time.Hour).Unix(),
		})

		body, _ := json.Marshal(map[string]string{
			"token": tokenStr,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/auth/token", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := TokenAuth(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("jwt in body without dashboardId is allowed", func(t *testing.T) {
		tokenStr := createJWT(jwt.MapClaims{
			"userId": "user-123",
			"exp":    time.Now().Add(time.Hour).Unix(),
		})

		body, _ := json.Marshal(map[string]string{
			"token": tokenStr,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/auth/token", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := TokenAuth(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("standard JWT refresh preserves original exp time", func(t *testing.T) {
		origExp := time.Now().Add(30 * time.Minute).Unix()
		tokenStr := createJWT(jwt.MapClaims{
			"userId": "user-123",
			"exp":    origExp,
		})

		body, _ := json.Marshal(map[string]any{
			"variables": map[string]any{"city": "Berlin"},
		})

		req := httptest.NewRequest(http.MethodPost, "/api/auth/token", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := TokenAuth(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var res struct {
			JWT string `json:"jwt"`
		}
		json.Unmarshal(rec.Body.Bytes(), &res)
		claims := parseJWT(res.JWT)

		intExp := int64(claims["exp"].(float64))
		if intExp != origExp {
			t.Errorf("expected exp to be preserved as %d, got %d", origExp, intExp)
		}
	})

	t.Run("short-lived JWT can generate long-lived token", func(t *testing.T) {
		tokenStr := createJWT(jwt.MapClaims{
			"userId": "user-123",
			"exp":    time.Now().Add(time.Hour).Unix(),
		})

		body, _ := json.Marshal(map[string]any{
			"longLived": true,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/auth/token", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := TokenAuth(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var res struct {
			JWT string `json:"jwt"`
		}
		json.Unmarshal(rec.Body.Bytes(), &res)
		claims := parseJWT(res.JWT)

		if claims["longLived"] != true {
			t.Errorf("expected longLived claim to be true")
		}
	})

	t.Run("long-lived JWT cannot generate another long-lived token", func(t *testing.T) {
		tokenStr := createJWT(jwt.MapClaims{
			"userId":    "user-123",
			"longLived": true,
			"exp":       time.Now().Add(30 * 24 * time.Hour).Unix(),
		})

		body, _ := json.Marshal(map[string]any{
			"longLived": true,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/auth/token", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := TokenAuth(app)
		if err := handler(c); err != nil {
			t.Fatalf("handler failed: %v", err)
		}

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
