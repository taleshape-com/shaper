// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"log/slog"
	"net/http"
	"shaper/server/core"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func ListApps(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctxUser := c.Get("user")
		if ctxUser != nil {
			claims := ctxUser.(*jwt.Token).Claims.(jwt.MapClaims)
			if _, hasId := claims["dashboardId"]; hasId {
				return c.JSONPretty(http.StatusUnauthorized, struct {
					Error string `json:"error"`
				}{Error: "Unauthorized"}, "  ")
			}
		}
		sort := c.QueryParam("sort")
		order := c.QueryParam("order")
		path := c.QueryParam("path")

		includeSubfolders := false
		if recursive := c.QueryParam("recursive"); recursive != "" {
			parsed, err := strconv.ParseBool(recursive)
			if err != nil {
				return c.JSONPretty(http.StatusBadRequest, struct {
					Error string `json:"error"`
				}{Error: "invalid recursive value"}, "  ")
			}
			includeSubfolders = parsed
		}

		var limit int
		if limitParam := c.QueryParam("limit"); limitParam != "" {
			parsed, err := strconv.Atoi(limitParam)
			if err != nil || parsed < 0 {
				return c.JSONPretty(http.StatusBadRequest, struct {
					Error string `json:"error"`
				}{Error: "invalid limit value"}, "  ")
			}
			limit = parsed
		}

		var offset int
		if offsetParam := c.QueryParam("offset"); offsetParam != "" {
			parsed, err := strconv.Atoi(offsetParam)
			if err != nil || parsed < 0 {
				return c.JSONPretty(http.StatusBadRequest, struct {
					Error string `json:"error"`
				}{Error: "invalid offset value"}, "  ")
			}
			offset = parsed
		}

		includeContent := false
		if includeContentParam := c.QueryParam("include_content"); includeContentParam != "" {
			parsed, err := strconv.ParseBool(includeContentParam)
			if err != nil {
				return c.JSONPretty(http.StatusBadRequest, struct {
					Error string `json:"error"`
				}{Error: "invalid include_content value"}, "  ")
			}
			includeContent = parsed
		}

		result, err := core.ListApps(app, c.Request().Context(), core.ListAppsOptions{
			Sort:              sort,
			Order:             order,
			Path:              path,
			IncludeSubfolders: includeSubfolders,
			IncludeContent:    includeContent,
			Limit:             limit,
			Offset:            offset,
		})
		if err != nil {
			c.Logger().Error("error listing apps:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}
		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}
