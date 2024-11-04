package handler

import (
	"net/http"
	"shaper/core"
	"time"

	"github.com/labstack/echo/v4"
)

func CookieLogin(app *core.App) echo.HandlerFunc {
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
		// Set the cookie
		cookie := new(http.Cookie)
		cookie.Name = "shaper-token"
		cookie.Value = loginRequest.Token
		cookie.Expires = time.Now().Add(90 * 24 * time.Hour)
		cookie.HttpOnly = true
		cookie.Secure = true // Use this in production with HTTPS
		cookie.SameSite = http.SameSiteStrictMode
		cookie.Path = "/api"
		c.SetCookie(cookie)
		return c.JSON(http.StatusOK, map[string]string{"message": "Login successful"})
	}
}

func TestCookie(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "Cookie is valid"})
}
