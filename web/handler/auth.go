package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"shaper/core"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func Logout(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)

		// Only allow logout for user sessions
		if _, hasAPIKey := claims["apiKeyId"]; hasAPIKey {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Cannot logout with API key"})
		}

		sessionID, ok := claims["sessionId"].(string)
		if !ok || sessionID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "No session ID in token"})
		}

		if err := core.Logout(app, c.Request().Context(), sessionID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete session"})
		}

		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func Login(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Parse the request body
		var loginRequest struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.Bind(&loginRequest); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}
		// If a token is provided, validate it
		sessionToken, err := core.Login(app, c.Request().Context(), loginRequest.Email, loginRequest.Password)
		if err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		} else if sessionToken == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid email or password"})
		}
		return c.JSON(http.StatusOK, map[string]string{"token": sessionToken})
	}
}

func validateVariables(mixedMap map[string]any) error {
	for k, v := range mixedMap {
		switch val := v.(type) {
		case string:
			continue
		case []any:
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

func LoginEnabled(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]bool{
			"enabled": app.LoginRequired,
		})
	}
}

func TokenAuth(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Parse the request body
		var loginRequest struct {
			Token       string         `json:"token"`
			DashboardID string         `json:"dashboardId"`
			Variables   map[string]any `json:"variables"`
		}
		if err := c.Bind(&loginRequest); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}
		// Check if the token is valid and get auth info
		authInfo, err := core.ValidToken(app, c.Request().Context(), loginRequest.Token)
		if err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct{ Error error }{Error: err}, "  ")
		}
		if !authInfo.Valid {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
		}

		// Add user or API key info to claims
		claims := jwt.MapClaims{
			"exp": time.Now().Add(app.JWTExp).Unix(),
		}
		if authInfo.IsUser {
			claims["userId"] = authInfo.UserID
			claims["userEmail"] = authInfo.UserEmail
			claims["userName"] = authInfo.UserName
			claims["sessionId"] = authInfo.SessionID
		} else if authInfo.APIKeyID != "" {
			claims["apiKeyId"] = authInfo.APIKeyID
			claims["apiKeyName"] = authInfo.APIKeyName
		}
		if loginRequest.DashboardID != "" {
			claims["dashboardId"] = loginRequest.DashboardID
		}
		if len(loginRequest.Variables) > 0 {
			err := validateVariables(loginRequest.Variables)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]any{
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

func PublicAuth(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		if app.NoPublicSharing {
			c.Logger().Warn("public sharing disabled")
			return c.JSONPretty(http.StatusNotFound, struct {
				Error string `json:"error"`
			}{Error: "not found"}, "  ")
		}
		// Parse the request body
		var loginRequest struct {
			DashboardID string `json:"dashboardId"`
		}
		if err := c.Bind(&loginRequest); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}
		if loginRequest.DashboardID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing dashboardId"})
		}

		dashboard, err := core.GetDashboardQuery(app, c.Request().Context(), loginRequest.DashboardID)
		if err != nil {
			c.Logger().Error("error getting dashboard:", slog.Any("error", err))
			return c.JSONPretty(http.StatusNotFound, struct {
				Error string `json:"error"`
			}{Error: "not found"}, "  ")
		}
		if dashboard.Visibility == nil || *dashboard.Visibility != "public" {
			return c.JSONPretty(http.StatusNotFound, struct {
				Error string `json:"error"`
			}{Error: "not found"}, "  ")
		}

		// Add user or API key info to claims
		claims := jwt.MapClaims{
			"exp": time.Now().Add(app.JWTExp).Unix(),
		}
		claims["dashboardId"] = loginRequest.DashboardID

		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := jwtToken.SignedString(app.JWTSecret)
		if err != nil {
			c.Logger().Error("Failed to sign token", slog.Any("error", err))
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to sign token"})
		}
		return c.JSON(http.StatusOK, map[string]string{"jwt": tokenString})
	}
}

func Setup(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		var setupRequest struct {
			Email    string `json:"email"`
			Name     string `json:"name"`
			Password string `json:"password"`
		}
		if err := c.Bind(&setupRequest); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}

		if setupRequest.Email == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email is required"})
		}
		if setupRequest.Password == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Password is required"})
		}

		id, err := core.CreateUser(app, c.Request().Context(), setupRequest.Email, setupRequest.Password, setupRequest.Name)
		if err != nil {
			if errors.Is(err, core.ErrUserSetupCompleted) {
				return c.JSON(http.StatusConflict, map[string]string{"error": "User setup already completed"})
			}
			c.Logger().Error("Failed to create user", slog.Any("error", err))
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
		}

		return c.JSON(http.StatusOK, map[string]string{"id": id})
	}
}

func ResetJWTSecret(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		_, err := core.ResetJWTSecret(app, c.Request().Context())
		if err != nil {
			app.Logger.Error("Failed to reset JWT secret", slog.Any("error", err))
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to reset JWT secret"})
		}
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}
