package handler

import (
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
