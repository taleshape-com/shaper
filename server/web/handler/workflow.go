// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"log/slog"
	"net/http"
	"shaper/server/core"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func RunWorkflow(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		var request struct {
			Content string `json:"content"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}

		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}

		result, err := core.RunWorkflow(app, c.Request().Context(), request.Content)

		if err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}

func GetWorkflow(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		idParam := c.Param("id")
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		result, err := core.GetWorkflow(app, c.Request().Context(), idParam)
		if err != nil {
			c.Logger().Error("error getting workflow:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}
		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}

func CreateWorkflow(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized,
				struct {
					Error string `json:"error"`
				}{Error: "Unauthorized"}, "  ")
		}

		var request struct {
			Name    string `json:"name"`
			Content string `json:"content"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: "Invalid request"}, "  ")
		}

		// Validate workflow name
		if request.Name == "" {
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: "Workflow name is required"}, "  ")
		}

		id, err := core.CreateWorkflow(app, c.Request().Context(), request.Name, request.Content)
		if err != nil {
			c.Logger().Error("error creating workflow:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusCreated, struct {
			ID string `json:"id"`
		}{
			ID: id,
		}, "  ")
	}
}

func SaveWorkflowContent(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		var request struct {
			Content string `json:"content"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid request"}, "  ")
		}
		err := core.SaveWorkflowContent(app, c.Request().Context(), c.Param("id"), request.Content)
		if err != nil {
			c.Logger().Error("error saving workflow query:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func SaveWorkflowName(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		var request struct {
			Name string `json:"name"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid request"}, "  ")
		}
		if request.Name == "" {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Workflow name is required"}, "  ")
		}
		err := core.SaveWorkflowName(app, c.Request().Context(), c.Param("id"), request.Name)
		if err != nil {
			c.Logger().Error("error saving workflow name:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func DeleteWorkflow(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized,
				struct {
					Error string `json:"error"`
				}{Error: "Unauthorized"}, "  ")
		}
		err := core.DeleteWorkflow(app, c.Request().Context(), c.Param("id"))
		if err != nil {
			c.Logger().Error("error deleting workflow:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
		}
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}
