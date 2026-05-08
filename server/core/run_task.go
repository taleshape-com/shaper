// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"shaper/server/util"
	"strings"
	"time"

	"github.com/duckdb/duckdb-go/v2"
	"github.com/jmoiron/sqlx"
)

type TaskQueryResult struct {
	SQL           string   `json:"sql"`
	Duration      int64    `json:"duration"`
	Error         *string  `json:"error,omitempty"`
	StopExecution bool     `json:"stopExecution,omitempty"`
	ResultColumns []string `json:"resultColumns"`
	ResultRows    [][]any  `json:"resultRows"`
}

type TaskResult struct {
	StartedAt    int64             `json:"startedAt"`
	NextRunAt    int64             `json:"nextRunAt,omitempty"`
	Success      bool              `json:"success"`
	TotalQueries int               `json:"totalQueries"`
	ScheduleType string            `json:"scheduleType,omitempty"`
	Queries      []TaskQueryResult `json:"queries"`
}

func getScheduleColumn(columns []*sql.ColumnType, rows Rows) (string, bool) {
	scheduleType := "single"
	col, _ := findColumnByTag(columns, "SCHEDULE")
	if col == nil {
		col, _ = findColumnByTag(columns, "SCHEDULE_ALL")
		if col == nil {
			return "", false
		}
		scheduleType = "all"
	}
	return scheduleType, (len(rows) == 0 || (len(rows) == 1 && len(rows[0]) == 1))
}

func needsNoTransaction(sql string) bool {
	upper := strings.ToUpper(strings.TrimSpace(sql))
	if strings.HasPrefix(upper, "ATTACH") || strings.HasPrefix(upper, "DETACH") {
		return true
	}
	if strings.HasPrefix(upper, "CREATE") && (strings.Contains(upper, "SECRET")) {
		// More precise check for CREATE SECRET
		parts := strings.Fields(upper)
		if len(parts) >= 2 && parts[0] == "CREATE" && parts[1] == "SECRET" {
			return true
		}
	}
	if strings.HasPrefix(upper, "INSTALL") || strings.HasPrefix(upper, "LOAD") {
		return true
	}
	return false
}

func RunTask(app *App, ctx context.Context, content string) (TaskResult, error) {
	result := TaskResult{
		StartedAt: time.Now().UnixMilli(),
		Queries:   []TaskQueryResult{},
	}

	cleanContent := util.StripSQLComments(content)
	sqls, err := util.SplitSQLQueries(cleanContent)
	if err != nil {
		return result, err
	}
	result.TotalQueries = len(sqls)

	db, cleanup, err := app.GetDuckDB(ctx)
	if err != nil {
		return result, fmt.Errorf("Error getting DB: %v", err)
	}
	defer cleanup()
	conn, err := db.Connx(ctx)
	if err != nil {
		return result, fmt.Errorf("Error getting conn: %v", err)
	}
	defer conn.Close()

	// Detect if it's an init task or contains transaction-breaking statements
	isInit := false
	anyNoTx := false
	for _, s := range sqls {
		if needsNoTransaction(s) {
			anyNoTx = true
			break
		}
	}

	if len(sqls) > 0 {
		sqlString := strings.TrimSpace(sqls[0])
		// We use a separate temporary query to check the schedule
		// to not interfere with the actual task execution state
		rows, err := conn.QueryxContext(ctx, sqlString)
		if err == nil {
			func() {
				defer rows.Close()
				colTypes, err := rows.ColumnTypes()
				if err != nil {
					app.Logger.Debug("Failed to get column types", slog.Any("error", err))
					return
				}
				resRows := [][]any{}
				if rows.Next() {
					row, err := rows.SliceScan()
					if err == nil {
						resRows = append(resRows, row)
					}
				}
				_, isSchedule := getScheduleColumn(colTypes, resRows)
				if isSchedule {
					timeVal := getScheduleTime(resRows)
					if timeVal == -1 {
						isInit = true
					}
				}
			}()
		} else {
			app.Logger.Debug("Failed to execute first query for init detection", slog.Any("error", err), slog.String("sql", sqlString))
		}
	}

	useTx := !isInit && !anyNoTx
	app.Logger.Debug("RunTask", slog.Bool("isInit", isInit), slog.Bool("anyNoTx", anyNoTx), slog.Int("queries", len(sqls)))

	var tx *sqlx.Tx
	if useTx {
		tx, err = conn.BeginTxx(ctx, nil)
		if err != nil {
			return result, fmt.Errorf("Error starting transaction: %v", err)
		}
	}

	success := true
	for sqlIndex, sqlString := range sqls {
		sqlString = strings.TrimSpace(sqlString)
		if sqlString == "" {
			continue
		}

		queryResult := TaskQueryResult{
			SQL:           sqlString,
			ResultColumns: []string{},
			ResultRows:    [][]any{},
		}

		if !IsAllowedTaskStatement(sqlString, isInit) {
			errMsg := "Statement not allowed in tasks (e.g., PRAGMA, SET configuration)"
			queryResult.Error = &errMsg
			success = false
			result.Queries = append(result.Queries, queryResult)
			break
		}

		start := time.Now()

		varPrefix, _ := buildVarPrefix(app, nil, nil)
		fullSQL := varPrefix + sqlString

		var rows *sqlx.Rows
		if !useTx {
			rows, err = conn.QueryxContext(ctx, fullSQL)
		} else {
			rows, err = tx.QueryxContext(ctx, fullSQL)
		}
		duration := time.Since(start).Milliseconds()
		queryResult.Duration = duration

		if err != nil {
			errorMessage := err.Error()
			queryResult.Error = &errorMessage
			success = false
			result.Queries = append(result.Queries, queryResult)
			break // Stop executing remaining queries on error
		}

		colTypes, err := rows.ColumnTypes()
		if err != nil {
			errorMessage := err.Error()
			queryResult.Error = &errorMessage
			success = false
			result.Queries = append(result.Queries, queryResult)
			rows.Close()
			break
		}

		for _, col := range colTypes {
			queryResult.ResultColumns = append(queryResult.ResultColumns, col.Name())
		}

		for rows.Next() {
			row, err := rows.SliceScan()
			if err != nil {
				errorMessage := err.Error()
				queryResult.Error = &errorMessage
				success = false
				break
			}
			// Convert DuckDB maps to Go maps so they can be JSON serialized
			for i, val := range row {
				if duckMap, ok := val.(duckdb.Map); ok {
					goMap := make(map[string]any, len(duckMap))
					for k, v := range duckMap {
						if kStr, ok := k.(string); ok {
							goMap[kStr] = v
						}
					}
					row[i] = goMap
				}
			}

			queryResult.ResultRows = append(queryResult.ResultRows, row)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			errorMessage := err.Error()
			queryResult.Error = &errorMessage
			success = false
		}

		// Check for early termination: single row, single column, boolean false
		if len(queryResult.ResultRows) == 1 && len(queryResult.ResultRows[0]) == 1 {
			if boolVal, ok := queryResult.ResultRows[0][0].(bool); ok && !boolVal {
				queryResult.StopExecution = true
			}
		}

		if scheduleType, isSchedule := getScheduleColumn(colTypes, queryResult.ResultRows); isSchedule {
			if result.NextRunAt != 0 {
				errMsg := "Multiple SCHEDULE queries in task"
				queryResult.Error = &errMsg
				success = false
				result.Queries = append(result.Queries, queryResult)
			} else {
				result.NextRunAt = getScheduleTime(queryResult.ResultRows)
				result.ScheduleType = scheduleType
				result.TotalQueries = len(sqls) - 1
			}
		} else {
			if sqlIndex == 0 {
				errMsg := "First query in task must define the schedule, for example:\nSELECT NULL::SCHEDULE;"
				queryResult.Error = &errMsg
				success = false
			}
			result.Queries = append(result.Queries, queryResult)
		}

		if !success || queryResult.StopExecution {
			break
		}
	}

	if success {
		if useTx {
			if err := tx.Commit(); err != nil {
				return result, fmt.Errorf("Error committing transaction: %v", err)
			}
		}
		result.Success = true
	} else {
		if useTx {
			if err := tx.Rollback(); err != nil {
				return result, fmt.Errorf("Error rolling back transaction: %v", err)
			}
		}
		result.Success = false
	}

	return result, nil
}
