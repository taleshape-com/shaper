package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"shaper/core"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func TokenLogin(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Parse the request body
		var loginRequest struct {
			Token     string                 `json:"token"`
			Variables map[string]interface{} `json:"variables"`
		}
		if err := c.Bind(&loginRequest); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}
		// Check if the token is valid
		if ok, err := core.ValidLogin(app, c.Request().Context(), loginRequest.Token); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct{ Error error }{Error: err}, "  ")
		} else if !ok {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
		}
		claims := jwt.MapClaims{
			"exp": time.Now().Add(app.JWTExp).Unix(),
		}
		if len(loginRequest.Variables) > 0 {
			err := validateVariables(loginRequest.Variables)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error":     "Invalid variables format: " + err.Error(),
					"variables": loginRequest.Variables,
				})
			}
			claims["variables"] = loginRequest.Variables
		}
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := jwtToken.SignedString(app.JWTSecret)
		if err != nil {
			c.Logger().Error("Failed to sign token", slog.Any("error", err))
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to sign token"})
		}
		return c.JSON(http.StatusOK, map[string]string{"jwt": tokenString})
	}
}

func validateVariables(mixedMap map[string]interface{}) error {
	for k, v := range mixedMap {
		switch val := v.(type) {
		case string:
			continue
		case []interface{}:
			for _, item := range val {
				if _, ok := item.(string); !ok {
					return fmt.Errorf("invalid type in array for key %s: %T", k, item)
				}
			}
		default:
			return fmt.Errorf("unsupported type for key %s: %T", k, v)
		}
	}
	return nil
}

func TokenAuth(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Parse the request body
		var loginRequest struct {
			Token       string                 `json:"token"`
			DashboardID string                 `json:"dashboardId"`
			Variables   map[string]interface{} `json:"variables"`
		}
		if err := c.Bind(&loginRequest); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}
		// Check if the token is valid
		if ok, err := core.ValidLogin(app, c.Request().Context(), loginRequest.Token); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct{ Error error }{Error: err}, "  ")
		} else if !ok {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
		}
		claims := jwt.MapClaims{
			"exp": time.Now().Add(app.JWTExp).Unix(),
		}
		// TODO: remove after migration
		if loginRequest.DashboardID == "embed" {
			loginRequest.DashboardID = "ja1ce8t8x53fkpd8dsmh8qrt"
		}
		if loginRequest.DashboardID != "" {
			claims["dashboardId"] = loginRequest.DashboardID
		}
		if len(loginRequest.Variables) > 0 {
			err := validateVariables(loginRequest.Variables)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error":     "Invalid variables format: " + err.Error(),
					"variables": loginRequest.Variables,
				})
			}
			claims["variables"] = loginRequest.Variables
		}

		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := jwtToken.SignedString(app.JWTSecret)
		if err != nil {
			c.Logger().Error("Failed to sign token", slog.Any("error", err))
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to sign token"})
		}
		return c.JSON(http.StatusOK, map[string]string{"jwt": tokenString})
	}
}

func ResetJWTSecret(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct{ Error string }{Error: "Unauthorized"}, "  ")
		}
		_, err := core.ResetJWTSecret(app, c.Request().Context())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to reset JWT secret"})
		}
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}
