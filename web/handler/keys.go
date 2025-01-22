package handler

import (
	"log/slog"
	"net/http"
	"shaper/core"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func ListAPIKeys(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		result, err := core.ListAPIKeys(app, c.Request().Context())
		if err != nil {
			c.Logger().Error("error listing api keys:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}
		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}

func CreateAPIKey(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized,
				struct {
					Error string `json:"error"`
				}{Error: "Unauthorized"}, "  ")
		}

		var request struct {
			Name string `json:"name"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: "Invalid request"}, "  ")
		}

		// Validate dashboard name
		if request.Name == "" {
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: "Dashboard name is required"}, "  ")
		}

		id, key, err := core.CreateAPIKey(app, c.Request().Context(), request.Name)
		if err != nil {
			c.Logger().Error("error creating dashboard:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusOK, struct {
			ID  string `json:"id"`
			Key string `json:"key"`
		}{
			ID:  id,
			Key: key,
		}, "  ")
	}
}

func DeleteAPIKey(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized,
				struct {
					Error string `json:"error"`
				}{Error: "Unauthorized"}, "  ")
		}

		err := core.DeleteAPIKey(app, c.Request().Context(), c.Param("id"))
		if err != nil {
			c.Logger().Error("error deleting api key:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusOK, struct {
			Deleted bool `json:"deleted"`
		}{Deleted: true}, "  ")
	}
}
