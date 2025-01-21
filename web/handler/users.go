package handler

import (
	"log/slog"
	"net/http"
	"time"

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

type CreateInviteRequest struct {
	Email string `json:"email"`
}

type CreateInviteResponse struct {
	Code      string    `json:"code"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

func CreateInvite(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}

		var req CreateInviteRequest
		if err := c.Bind(&req); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid request body"}, "  ")
		}

		if req.Email == "" {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Email is required"}, "  ")
		}

		invite, err := core.CreateInvite(app, c.Request().Context(), req.Email)
		if err != nil {
			c.Logger().Error("error creating invite:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusOK, invite, "  ")
	}
}
