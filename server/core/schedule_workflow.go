// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"fmt"
	"log/slog"
	"shaper/server/util"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

func scheduleWorkflow(app *App, ctx context.Context, workflowID, content string) {
	// cancel previous timer
	unscheduleWorkflow(app, workflowID)
	// Run first query
	cleanContent := util.StripSQLComments(content)
	sqls, err := splitSQLQueries(cleanContent)
	if err != nil {
		app.Logger.Error("Error splitting SQL queries", slog.Any("error", err), slog.String("workflowID", workflowID))
		return
	}
	if len(sqls) == 0 {
		app.Logger.Error("No SQL queries found in workflow content", slog.String("workflowID", workflowID))
		return
	}
	conn, err := app.DB.Connx(ctx)
	if err != nil {
		app.Logger.Error("Error getting database connection", slog.Any("error", err), slog.String("workflowID", workflowID))
		return
	}
	defer conn.Close()
	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		app.Logger.Error("Error starting transaction", slog.Any("error", err), slog.String("workflowID", workflowID))
		return
	}
	sqlString := strings.TrimSpace(sqls[0])
	if sqlString == "" {
		app.Logger.Error("First SQL query is empty", slog.String("workflowID", workflowID))
		return
	}
	rows, err := tx.QueryxContext(ctx, sqlString)
	if err != nil {
		app.Logger.Error("Error executing first SQL query", slog.Any("error", err), slog.String("workflowID", workflowID), slog.String("query", sqlString))
		return
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		app.Logger.Error("Error getting column types", slog.Any("error", err), slog.String("workflowID", workflowID))
		return
	}
	result := [][]any{}
	for rows.Next() {
		row, err := rows.SliceScan()
		if err != nil {
			app.Logger.Error("Error scanning row", slog.Any("error", err), slog.String("workflowID", workflowID))
			rows.Close()
			tx.Rollback()
			return
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		app.Logger.Error("Error iterating over rows", slog.Any("error", err), slog.String("workflowID", workflowID))
		tx.Rollback()
		return
	}
	if !isSchedule(colTypes, result) {
		app.Logger.Error("First SQL query is not a SCHEDULE query", slog.String("workflowID", workflowID), slog.String("query", sqlString))
		tx.Rollback()
		return
	}
	reloadValue := getReloadValue(result)
	if reloadValue <= 0 {
		app.Logger.Info("Workflow SCHEDULE set to NULL, skipping scheduling", slog.String("workflow", workflowID))
		tx.Rollback()
		return
	}
	nextRunAt := time.UnixMilli(reloadValue)
	// set next_run_at in DB
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO `+app.Schema+`.workflow_runs
		(workflow_id, next_run_at)
		VALUES ($1, $2)
		ON CONFLICT DO UPDATE SET next_run_at = $2`,
		workflowID,
		nextRunAt,
	)
	if err != nil {
		app.Logger.Error("Error inserting next run time into DB", slog.Any("error", err), slog.String("workflowID", workflowID))
		tx.Rollback()
		return
	}
	// schedule for later
	t := time.AfterFunc(time.Until(nextRunAt), func() {
		app.Logger.Info("Dispatching job", slog.String("workflow", workflowID))
		// Send message to NATS
		subject := app.JobsSubjectPrefix + workflowID
		msgID := fmt.Sprintf("%s-%d", workflowID, nextRunAt.UnixMilli())
		_, err := app.JetStream.Publish(ctx, subject, []byte{}, jetstream.WithMsgID(msgID))
		if err != nil {
			app.Logger.Error("failed to publish workflow run message", slog.Any("error", err), slog.String("subject", subject))
		}
	})
	app.WorkflowTimers[workflowID] = t
	err = tx.Commit()
	if err != nil {
		app.Logger.Error("Error committing transaction", slog.Any("error", err), slog.String("workflowID", workflowID))
		return
	}
	app.Logger.Info("Scheduling workflow", slog.String("workflow", workflowID), slog.Time("next", nextRunAt))
}

func unscheduleWorkflow(app *App, workflowID string) {
	if existingTimer, hasTimer := app.WorkflowTimers[workflowID]; hasTimer {
		existingTimer.Stop()
		delete(app.WorkflowTimers, workflowID)
	}
}

func (app *App) HandleJob(msg jetstream.Msg) {
	workflowID := strings.TrimPrefix(msg.Subject(), app.JobsSubjectPrefix)
	app.Logger.Info("Handling job", slog.String("workflow", workflowID))
	ctx := ContextWithActor(context.Background(), &Actor{Type: ActorJob})
	workflow, err := GetWorkflow(app, ctx, workflowID)
	if err != nil {
		app.Logger.Error("Error getting workflow", slog.String("workflow", workflowID), slog.Any("error", err))
		if err := msg.Nak(); err != nil {
			app.Logger.Error("Error nack message", slog.Any("error", err))
		}
		return
	}
	_, err = RunWorkflow(app, ctx, workflow.Content)
	scheduleWorkflow(app, ctx, workflowID, workflow.Content)
	if err != nil {
		app.Logger.Error("Error running workflow", slog.String("workflow", workflowID), slog.Any("error", err))
		if err := msg.Nak(); err != nil {
			app.Logger.Error("Error nack message", slog.Any("error", err))
		}
		return
	}
	err = msg.Ack()
	if err != nil {
		app.Logger.Error("Error acking message", slog.Any("error", err))
		return
	}
	app.Logger.Info("Workflow run completed", slog.String("workflowID", workflowID))
}

func scheduleExistingWorkflows(app *App, ctx context.Context) error {
	app.Logger.Info("Loading scheduled workflows")
	// Load scheduled workflow runs from database
	rows, err := app.DB.QueryxContext(
		ctx,
		`SELECT w.workflow_id, w.next_run_at, a.content
			FROM `+app.Schema+`.workflow_runs w
			JOIN `+app.Schema+`.apps a ON w.workflow_id = a.id
			WHERE next_run_at IS NOT NULL`,
	)
	if err != nil {
		rows.Close()
		return fmt.Errorf("failed to query scheduled workflows: %w", err)
	}
	for rows.Next() {
		var workflowID string
		var nextRunAt time.Time
		var content string
		if err := rows.Scan(&workflowID, &nextRunAt, &content); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan scheduled workflow: %w", err)
		}
		scheduleWorkflow(app, ctx, workflowID, content)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating over scheduled workflows: %w", err)
	}
	app.Logger.Info("Scheduled workflows loaded successfully")
	return nil
}
