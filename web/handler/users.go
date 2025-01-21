package handler

import (
	"log/slog"
	"net/http"

	"shaper/core"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func ListUsers(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		sort := c.QueryParam("sort")
		order := c.QueryParam("order")
		result, err := core.ListUsers(app, c.Request().Context(), sort, order)
		if err != nil {
			c.Logger().Error("error listing users:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}
		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}

func DeleteUser(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized,
				struct {
					Error string `json:"error"`
				}{Error: "Unauthorized"}, "  ")
		}

		err := core.DeleteUser(app, c.Request().Context(), c.Param("id"))
		if err != nil {
			c.Logger().Error("error deleting user:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
		}

		return c.NoContent(http.StatusOK)
	}
}
