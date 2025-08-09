// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"fmt"
	"shaper/server/util"
	"strings"
	"time"
)

type WorkflowQueryResult struct {
	SQL           string  `json:"sql"`
	Duration      int64   `json:"duration"`
	Error         *string `json:"error,omitempty"`
	StopExecution bool    `json:"stopExecution,omitempty"`
	Result        any     `json:"result,omitempty"`
}

type WorkflowResult struct {
	StartTime    time.Time             `json:"startTime"`
	Success      bool                  `json:"success"`
	TotalQueries int                   `json:"totalQueries"`
	Queries      []WorkflowQueryResult `json:"queries"`
}

func RunWorkflow(app *App, ctx context.Context, content string) (WorkflowResult, error) {
	result := WorkflowResult{
		StartTime: time.Now(),
		Queries:   []WorkflowQueryResult{},
	}

	cleanContent := util.StripSQLComments(content)
	sqls, err := splitSQLQueries(cleanContent)
	if err != nil {
		return result, err
	}
	result.TotalQueries = len(sqls)

	conn, err := app.DB.Connx(ctx)
	if err != nil {
		return result, fmt.Errorf("Error getting conn: %v", err)
	}
	defer conn.Close()

	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("Error starting transaction: %v", err)
	}

	success := true
	for _, sqlString := range sqls {
		sqlString = strings.TrimSpace(sqlString)
		if sqlString == "" {
			continue
		}

		queryResult := WorkflowQueryResult{
			SQL: sqlString,
		}

		start := time.Now()

		rows, err := tx.QueryContext(ctx, sqlString)
		duration := time.Since(start).Milliseconds()
		queryResult.Duration = duration

		if err != nil {
			errorMessage := err.Error()
			queryResult.Error = &errorMessage
			success = false
			result.Queries = append(result.Queries, queryResult)
			break // Stop executing remaining queries on error
		} else {
			var rowData []map[string]any

			if rows != nil {
				columns, err := rows.Columns()
				if err != nil {
					errorMessage := err.Error()
					queryResult.Error = &errorMessage
					success = false
					result.Queries = append(result.Queries, queryResult)
					rows.Close()
					break
				}

				for rows.Next() {
					values := make([]any, len(columns))
					valuePtrs := make([]any, len(columns))
					for i := range values {
						valuePtrs[i] = &values[i]
					}

					if err := rows.Scan(valuePtrs...); err != nil {
						errorMessage := err.Error()
						queryResult.Error = &errorMessage
						success = false
						result.Queries = append(result.Queries, queryResult)
						rows.Close()
						break
					}

					rowMap := make(map[string]any)
					for i, col := range columns {
						if values[i] != nil {
							if b, ok := values[i].([]byte); ok {
								rowMap[col] = string(b)
							} else {
								rowMap[col] = values[i]
							}
						} else {
							rowMap[col] = nil
						}
					}
					rowData = append(rowData, rowMap)
				}
				rows.Close()

				if !success {
					break
				}
			}

			queryResult.Result = rowData

			// Check for early termination: single row, single column, boolean false
			if len(rowData) == 1 && len(rowData[0]) == 1 {
				for _, value := range rowData[0] {
					if boolVal, ok := value.(bool); ok && !boolVal {
						queryResult.StopExecution = true
						result.Queries = append(result.Queries, queryResult)
						break
					}
				}
				if !success || queryResult.StopExecution {
					break
				}
			}
		}

		result.Queries = append(result.Queries, queryResult)
	}

	if success {
		if err := tx.Commit(); err != nil {
			return result, fmt.Errorf("Error committing transaction: %v", err)
		}
		result.Success = true
	} else {
		if err := tx.Rollback(); err != nil {
			return result, fmt.Errorf("Error rolling back transaction: %v", err)
		}
		result.Success = false
	}

	return result, nil
}
