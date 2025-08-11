// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"shaper/server/util"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type WorkflowQueryResult struct {
	SQL           string   `json:"sql"`
	Duration      int64    `json:"duration"`
	Error         *string  `json:"error,omitempty"`
	StopExecution bool     `json:"stopExecution,omitempty"`
	ResultColumns []string `json:"resultColumns"`
	ResultRows    [][]any  `json:"resultRows"`
}

type WorkflowResult struct {
	StartedAt    int64                 `json:"startedAt"`
	ReloadAt     int64                 `json:"reloadAt,omitempty"`
	Success      bool                  `json:"success"`
	TotalQueries int                   `json:"totalQueries"`
	Queries      []WorkflowQueryResult `json:"queries"`
}

type UpdateWorkflowRunPayload struct {
	ID        string         `json:"id"`
	UpdatedBy string         `json:"updatedBy"`
	RunResult WorkflowResult `json:"lastRunResult"`
}

func RunWorkflow(app *App, ctx context.Context, content string, workflowID string) (WorkflowResult, error) {
	result := WorkflowResult{
		StartedAt: time.Now().UnixMilli(),
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

		if isReload(colTypes, queryResult.ResultRows) {
			if result.ReloadAt != 0 {
				errMsg := "Multiple RELOAD queries in workflow"
				queryResult.Error = &errMsg
				success = false
				result.Queries = append(result.Queries, queryResult)
			} else {
				result.ReloadAt = getReloadValue(queryResult.ResultRows)
				result.TotalQueries = len(sqls) - 1
			}
		} else {
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

	if workflowID != "" {
		actor := ActorFromContext(ctx)
		if actor == nil {
			app.Logger.ErrorContext(ctx, "no actor in context")
		} else {
			err = app.SubmitState(ctx, "update_workflow_run", UpdateWorkflowRunPayload{
				ID:        workflowID,
				UpdatedBy: actor.String(),
				RunResult: result,
			})
			if err != nil {
				app.Logger.ErrorContext(ctx, "failed to update workflow run", slog.Any("error", err))
			}
		}
	}

	return result, nil
}

func HandleUpdateWorkflowRun(app *App, data []byte) bool {
	var payload UpdateWorkflowRunPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal update workflow run payload", slog.Any("error", err))
		return false
	}
	resultJSON, err := json.Marshal(payload.RunResult)
	if err != nil {
		app.Logger.Error("failed to marshal workflow run result", slog.Any("error", err))
		return false
	}
	lastRunAt := time.UnixMilli(payload.RunResult.StartedAt)
	var nextRunAt *time.Time
	if payload.RunResult.ReloadAt != 0 {
		t := time.UnixMilli(payload.RunResult.ReloadAt)
		nextRunAt = &t
	}
	_, err = app.DB.Exec(
		`INSERT INTO `+app.Schema+`.workflow_runs
			(workflow_id, last_run_at, last_run_by, last_run_result, next_run_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT DO UPDATE
		 	SET last_run_at = $2, last_run_by = $3, last_run_result = $4, next_run_at = $5`,
		payload.ID,
		lastRunAt,
		payload.UpdatedBy,
		resultJSON,
		nextRunAt,
	)
	if err != nil {
		app.Logger.Error("failed to execute UPDATE statement", slog.Any("error", err))
		return false
	}
	// cancel previous timer
	unscheduleWorkflow(app, payload.ID)
	// schedule for later
	if nextRunAt != nil {
		app.Logger.Info("Scheduled workflow", slog.String("workflow", payload.ID), slog.Time("run_time", *nextRunAt))
		t := time.AfterFunc(time.Until(*nextRunAt), func() {
			app.Logger.Info("Dispatching job", slog.String("workflow", payload.ID))
			// Send message to NATS
			ctx := context.Background()
			subject := app.JobsSubjectPrefix + payload.ID
			msgID := fmt.Sprintf("%s-%d", payload.ID, nextRunAt.UnixMilli())
			_, err := app.JetStream.Publish(ctx, subject, []byte{}, jetstream.WithMsgID(msgID))
			if err != nil {
				app.Logger.Error("failed to publish workflow run message", slog.Any("error", err), slog.String("subject", subject))
			}
		})
		app.WorkflowTimers[payload.ID] = t
	}
	return true
}
