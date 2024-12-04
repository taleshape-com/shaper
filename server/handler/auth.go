package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"shaper/core"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
)

func TokenLogin(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Parse the request body
		var loginRequest struct {
			Token string `json:"token"`
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
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"exp": time.Now().Add(time.Second * 10).Unix(),
		})
		tokenString, err := jwtToken.SignedString(app.JWTSecret)
		if err != nil {
			app.Logger.Error("Failed to sign token", slog.Any("error", err))
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to sign token"})
		}
		return c.JSON(http.StatusOK, map[string]string{"jwt": tokenString})
	}
}
func normalizeVariables(mixedMap map[string]interface{}) (map[string][]string, error) {
	result := make(map[string][]string)
	for k, v := range mixedMap {
		switch val := v.(type) {
		case string:
			result[k] = []string{val}
		case []interface{}:
			strSlice := make([]string, 0, len(val))
			for _, item := range val {
				if str, ok := item.(string); ok {
					strSlice = append(strSlice, str)
				} else {
					return nil, fmt.Errorf("invalid type in array for key %s: %T", k, item)
				}
			}
			result[k] = strSlice
		default:
			return nil, fmt.Errorf("unsupported type for key %s: %T", k, v)
		}
	}

	return result, nil
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
			"exp": time.Now().Add(time.Second * 10).Unix(),
		}
		if loginRequest.DashboardID != "" {
			claims["dashboardId"] = loginRequest.DashboardID
		}
		if len(loginRequest.Variables) > 0 {
			normalizedVars, err := normalizeVariables(loginRequest.Variables)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error":     "Invalid variables format: " + err.Error(),
					"variables": loginRequest.Variables,
				})
			}
			claims["variables"] = normalizedVars
		}

		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := jwtToken.SignedString(app.JWTSecret)
		if err != nil {
			app.Logger.Error("Failed to sign token", slog.Any("error", err))
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to sign token"})
		}
		return c.JSON(http.StatusOK, map[string]string{"jwt": tokenString})
	}
}
