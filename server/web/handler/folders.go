// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"log/slog"
	"net/http"
	"shaper/server/core"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func CreateFolder(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}

		var req core.CreateFolderRequest
		if err := c.Bind(&req); err != nil {
			c.Logger().Error("error binding request:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid request body"}, "  ")
		}

		// Validate required fields
		if req.Name == "" {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Folder name is required"}, "  ")
		}

		result, err := core.CreateFolder(app, c.Request().Context(), req)
		if err != nil {
			c.Logger().Error("error creating folder:", slog.Any("error", err))
			// Check if it's a duplicate folder error
			if strings.Contains(err.Error(), "already exists") {
				return c.JSONPretty(http.StatusConflict, struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
			}
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusCreated, result, "  ")
	}
}

func DeleteFolder(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}

		id := c.Param("id")
		if id == "" {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Folder ID is required"}, "  ")
		}

		err := core.DeleteFolder(app, c.Request().Context(), id)
		if err != nil {
			c.Logger().Error("error deleting folder:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusOK, struct {
			OK bool `json:"ok"`
		}{OK: true}, "  ")
	}
}

func MoveItems(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}

		var req core.MoveItemsRequest
		if err := c.Bind(&req); err != nil {
			c.Logger().Error("error binding request:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid request body"}, "  ")
		}

		// Validate request
		if len(req.Apps) == 0 && len(req.Folders) == 0 {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "No items to move"}, "  ")
		}

		err := core.MoveItems(app, c.Request().Context(), req)
		if err != nil {
			c.Logger().Error("error moving items:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusOK, struct {
			OK bool `json:"ok"`
		}{OK: true}, "  ")
	}
}
