// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"net/http"
	"shaper/server/core"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

func GetQueryExecutions(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		tracker := core.GetQueryTracker()

		filter := core.QueryFilter{}

		if typeParam := c.QueryParam("type"); typeParam != "" {
			types := strings.Split(typeParam, ",")
			filter.Types = make([]core.QueryExecutionType, 0, len(types))
			for _, t := range types {
				switch strings.ToLower(strings.TrimSpace(t)) {
				case "dashboard":
					filter.Types = append(filter.Types, core.QueryTypeDashboard)
				case "task":
					filter.Types = append(filter.Types, core.QueryTypeTask)
				case "sql_api":
					filter.Types = append(filter.Types, core.QueryTypeSQLAPI)
				case "download":
					filter.Types = append(filter.Types, core.QueryTypeDownload)
				}
			}
		}

		if statusParam := c.QueryParam("status"); statusParam != "" {
			statuses := strings.Split(statusParam, ",")
			filter.Status = make([]core.QueryExecutionStatus, 0, len(statuses))
			for _, s := range statuses {
				switch strings.ToLower(strings.TrimSpace(s)) {
				case "pending":
					filter.Status = append(filter.Status, core.QueryStatusPending)
				case "running":
					filter.Status = append(filter.Status, core.QueryStatusRunning)
				case "success":
					filter.Status = append(filter.Status, core.QueryStatusSuccess)
				case "failed":
					filter.Status = append(filter.Status, core.QueryStatusFailed)
				case "cancelled":
					filter.Status = append(filter.Status, core.QueryStatusCancelled)
				case "timed_out":
					filter.Status = append(filter.Status, core.QueryStatusTimedOut)
				}
			}
		}

		if limitParam := c.QueryParam("limit"); limitParam != "" {
			if limit, err := strconv.Atoi(limitParam); err == nil && limit > 0 {
				filter.Limit = limit
			}
		}
		if filter.Limit <= 0 {
			filter.Limit = 100
		}

		var executions []*core.QueryExecution
		if slowParam := c.QueryParam("slow"); strings.ToLower(slowParam) == "true" {
			executions = tracker.GetSlowQueries(filter)
		} else {
			executions = tracker.GetRecentExecutions(filter)
		}

		sanitized := make([]*core.QueryExecution, len(executions))
		for i, exec := range executions {
			sanitized[i] = exec.SanitizeForResponse()
		}

		return c.JSON(http.StatusOK, map[string]any{
			"executions": sanitized,
			"total":      len(sanitized),
		})
	}
}
