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

		return c.JSONPretty(http.StatusOK, struct {
			Deleted bool `json:"deleted"`
		}{Deleted: true}, "  ")
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

		return c.JSONPretty(http.StatusCreated, invite, "  ")
	}
}

func GetInvite(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		code := c.Param("code")
		if code == "" {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invite code is required"}, "  ")
		}

		invite, err := core.GetInvite(app, c.Request().Context(), code)
		if err != nil {
			c.Logger().Error("error getting invite:", slog.Any("error", err))
			return c.JSONPretty(http.StatusNotFound, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusOK, invite, "  ")
	}
}

type ClaimInviteRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func ClaimInvite(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req ClaimInviteRequest
		if err := c.Bind(&req); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid request body"}, "  ")
		}

		if req.Name == "" {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Name is required"}, "  ")
		}

		if req.Password == "" {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Password is required"}, "  ")
		}

		code := c.Param("code")
		if code == "" {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invite code is required"}, "  ")
		}

		err := core.ClaimInvite(app, c.Request().Context(), code, req.Name, req.Password)
		if err != nil {
			c.Logger().Error("error claiming invite:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusOK, struct {
			Claimed bool `json:"claimed"`
		}{Claimed: true}, "  ")
	}
}

func DeleteInvite(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}

		code := c.Param("code")
		if code == "" {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invite code is required"}, "  ")
		}

		err := core.DeleteInvite(app, c.Request().Context(), code)
		if err != nil {
			c.Logger().Error("error deleting invite:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusOK, struct {
			Deleted bool `json:"deleted"`
		}{Deleted: true}, "  ")
	}
}
