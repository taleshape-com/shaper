// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"database/sql"
	"fmt"
	"shaper/server/util"
	"strings"
	"time"
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
	result := TaskResult{
		StartedAt: time.Now().UnixMilli(),
		Queries:   []TaskQueryResult{},
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

		start := time.Now()

		rows, err := tx.QueryxContext(ctx, sqlString)
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
			queryResult.ResultRows = append(queryResult.ResultRows, row)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			errorMessage := err.Error()
			queryResult.Error = &errorMessage
			success = false
		}

		if scheduleType, isSchedule := getScheduleColumn(colTypes, queryResult.ResultRows); isSchedule {
			if result.NextRunAt != 0 {
				errMsg := "Multiple SCHEDULE queries in task"
				queryResult.Error = &errMsg
				success = false
				result.Queries = append(result.Queries, queryResult)
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
		}

		if !success {
			break
		}

		// Check for early termination: single row, single column, boolean false
		if len(queryResult.ResultRows) == 1 && len(queryResult.ResultRows[0]) == 1 {
			if boolVal, ok := queryResult.ResultRows[0][0].(bool); ok && !boolVal {
				queryResult.StopExecution = true
				break
			}
		}
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
