// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"database/sql"
	"fmt"
	"shaper/server/util"
	"strings"
	"time"

	"github.com/duckdb/duckdb-go/v2"
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

func RunTask(app *App, ctx context.Context, content string) (TaskResult, error) {
	tracker := GetQueryTracker()
	exec := tracker.Start(
		ctx,
		QueryTypeTask,
		nil,
		nil,
		nil,
		content,
	)

	result := TaskResult{
		StartedAt: time.Now().UnixMilli(),
		Queries:   []TaskQueryResult{},
	}

	cleanContent := util.StripSQLComments(content)
	sqls, err := util.SplitSQLQueries(cleanContent)
	if err != nil {
		tracker.Complete(exec, 0, err)
		return result, err
	}
	result.TotalQueries = len(sqls)

	db, cleanup, err := app.GetDuckDB(ctx)
	if err != nil {
		tracker.Complete(exec, 0, err)
		return result, fmt.Errorf("Error getting DB: %v", err)
	}
	defer cleanup()
	conn, err := db.Connx(ctx)
	if err != nil {
		tracker.Complete(exec, 0, err)
		return result, fmt.Errorf("Error getting conn: %v", err)
	}
	defer conn.Close()

	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		tracker.Complete(exec, 0, err)
		return result, fmt.Errorf("Error starting transaction: %v", err)
	}

	success := true
	var totalRows int64 = 0
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

		if !IsAllowedTaskStatement(sqlString) {
			errMsg := "Statement not allowed in tasks (e.g., INSTALL, LOAD, SET configuration)"
			queryResult.Error = &errMsg
			success = false
			result.Queries = append(result.Queries, queryResult)
			break
		}

		queryCtx, cancelQuery := WithQueryTimeout(ctx)
		defer cancelQuery()

		qIndex := sqlIndex
		queryExec := tracker.Start(
			queryCtx,
			QueryTypeTask,
			nil,
			nil,
			&qIndex,
			sqlString,
		)

		var queryRowCount int64 = 0
		var queryErr error = nil
		queryResultAdded := false

		func() {
			start := time.Now()

			varPrefix, _ := buildVarPrefix(app, nil, nil)
			rows, err := tx.QueryxContext(queryCtx, varPrefix+sqlString)
			duration := time.Since(start).Milliseconds()
			queryResult.Duration = duration

			if err != nil {
				queryErr = err
				return
			}
			defer rows.Close()

			colTypes, err := rows.ColumnTypes()
			if err != nil {
				queryErr = err
				return
			}

			for _, col := range colTypes {
				queryResult.ResultColumns = append(queryResult.ResultColumns, col.Name())
			}

			for rows.Next() {
				row, err := rows.SliceScan()
				if err != nil {
					queryErr = err
					return
				}
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
				queryRowCount++
				totalRows++
			}

			if rowsErr := rows.Err(); rowsErr != nil {
				queryErr = rowsErr
				return
			}

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
					queryResultAdded = true
				} else {
					result.NextRunAt = getReloadValue(queryResult.ResultRows)
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
				queryResultAdded = true
			}
		}()

		if queryErr != nil {
			errorMessage := queryErr.Error()
			queryResult.Error = &errorMessage
			success = false
			if !queryResultAdded {
				result.Queries = append(result.Queries, queryResult)
			}
		}

		CompleteQueryExecution(queryExec, queryRowCount, queryErr)

		if !success || queryResult.StopExecution {
			break
		}
	}

	if success {
		if err := tx.Commit(); err != nil {
			CompleteQueryExecution(exec, totalRows, err)
			return result, fmt.Errorf("Error committing transaction: %v", err)
		}
		result.Success = true
	} else {
		if err := tx.Rollback(); err != nil {
			CompleteQueryExecution(exec, totalRows, err)
			return result, fmt.Errorf("Error rolling back transaction: %v", err)
		}
		result.Success = false
	}

	var finalErr error = nil
	if !success {
		finalErr = fmt.Errorf("task execution failed")
	}
	CompleteQueryExecution(exec, totalRows, finalErr)
	return result, nil
}
