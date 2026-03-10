// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"net/http"
	"shaper/server/core"
	"shaper/server/util"
	"strings"

	"github.com/labstack/echo/v4"
)

func ExecuteSQL(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		var request struct {
			SQL string `json:"sql"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: "Invalid request body"}, "  ")
		}

		sql := strings.TrimSpace(request.SQL)
		if sql == "" {
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: "SQL is required"}, "  ")
		}

		// Strip comments and split queries to ensure only one query
		cleanSQL := util.StripSQLComments(sql)
		queries, err := util.SplitSQLQueries(cleanSQL)
		if err != nil {
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
		}

		if len(queries) != 1 {
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: "Only one SQL query is allowed"}, "  ")
		}

		// Set headers for CSV file download
		c.Response().Header().Set(echo.HeaderContentType, "text/csv")
		c.Response().Header().Set("X-Content-Type-Options", "nosniff")
		c.Response().Header().Set("Transfer-Encoding", "chunked")

		writer := c.Response().Writer

		err = core.StreamSQLToCSV(app, c.Request().Context(), queries[0]+";", writer)
		if err != nil {
			if !c.Response().Committed {
				return c.JSONPretty(http.StatusInternalServerError,
					struct {
						Error string `json:"error"`
					}{Error: err.Error()}, "  ")
			}
			// If already committed, we can't do much but log
			c.Logger().Error("Error streaming SQL result:", err)
			return err
		}

		return nil
	}
}
