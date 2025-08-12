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

func scheduleTask(app *App, ctx context.Context, taskID, content string) {
	// cancel previous timer
	unscheduleTask(app, taskID)
	// Run first query
	cleanContent := util.StripSQLComments(content)
	sqls, err := splitSQLQueries(cleanContent)
	if err != nil {
		app.Logger.Error("Error splitting SQL queries", slog.Any("error", err), slog.String("task", taskID))
		return
	}
	if len(sqls) == 0 {
		app.Logger.Error("No SQL queries found in task content", slog.String("task", taskID))
		return
	}
	conn, err := app.DB.Connx(ctx)
	if err != nil {
		app.Logger.Error("Error getting database connection", slog.Any("error", err), slog.String("task", taskID))
		return
	}
	defer conn.Close()
	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		app.Logger.Error("Error starting transaction", slog.Any("error", err), slog.String("task", taskID))
		return
	}
	sqlString := strings.TrimSpace(sqls[0])
	if sqlString == "" {
		app.Logger.Error("First SQL query is empty", slog.String("task", taskID))
		return
	}
	rows, err := tx.QueryxContext(ctx, sqlString)
	if err != nil {
		app.Logger.Error("Error executing first SQL query", slog.Any("error", err), slog.String("task", taskID), slog.String("query", sqlString))
		return
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		app.Logger.Error("Error getting column types", slog.Any("error", err), slog.String("task", taskID))
		return
	}
	result := [][]any{}
	for rows.Next() {
		row, err := rows.SliceScan()
		if err != nil {
			app.Logger.Error("Error scanning row", slog.Any("error", err), slog.String("task", taskID))
			rows.Close()
			tx.Rollback()
			return
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		app.Logger.Error("Error iterating over rows", slog.Any("error", err), slog.String("task", taskID))
		tx.Rollback()
		return
	}
	if !isSchedule(colTypes, result) {
		app.Logger.Error("First SQL query is not a SCHEDULE query", slog.String("task", taskID), slog.String("query", sqlString))
		tx.Rollback()
		return
	}
	reloadValue := getReloadValue(result)
	if reloadValue <= 0 {
		app.Logger.Info("Task SCHEDULE set to NULL, skipping scheduling", slog.String("task", taskID))
		tx.Rollback()
		return
	}
	nextRunAt := time.UnixMilli(reloadValue)
	// set next_run_at in DB
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO `+app.Schema+`.task_runs
		(task_id, next_run_at)
		VALUES ($1, $2)
		ON CONFLICT DO UPDATE SET next_run_at = $2`,
		taskID,
		nextRunAt,
	)
	if err != nil {
		app.Logger.Error("Error inserting next run time into DB", slog.Any("error", err), slog.String("taskID", taskID))
		tx.Rollback()
		return
	}
	// schedule for later
	t := time.AfterFunc(time.Until(nextRunAt), func() {
		app.Logger.Info("Dispatching job", slog.String("task", taskID))
		// Send message to NATS
		subject := app.JobsSubjectPrefix + taskID
		msgID := fmt.Sprintf("%s-%d", taskID, nextRunAt.UnixMilli())
		_, err := app.JetStream.Publish(ctx, subject, []byte{}, jetstream.WithMsgID(msgID))
		if err != nil {
			app.Logger.Error("failed to publish task run message", slog.Any("error", err), slog.String("subject", subject))
		}
	})
	app.TaskTimers[taskID] = t
	err = tx.Commit()
	if err != nil {
		app.Logger.Error("Error committing transaction", slog.Any("error", err), slog.String("taskID", taskID))
		return
	}
	app.Logger.Info("Scheduling task", slog.String("task", taskID), slog.Time("next", nextRunAt))
}

func unscheduleTask(app *App, taskID string) {
	if existingTimer, hasTimer := app.TaskTimers[taskID]; hasTimer {
		existingTimer.Stop()
		delete(app.TaskTimers, taskID)
	}
}

func (app *App) HandleJob(msg jetstream.Msg) {
	taskID := strings.TrimPrefix(msg.Subject(), app.JobsSubjectPrefix)
	app.Logger.Info("Handling job", slog.String("task", taskID))
	ctx := ContextWithActor(context.Background(), &Actor{Type: ActorJob})
	task, err := GetTask(app, ctx, taskID)
	if err != nil {
		app.Logger.Error("Error getting task", slog.String("task", taskID), slog.Any("error", err))
		if err := msg.Nak(); err != nil {
			app.Logger.Error("Error nack message", slog.Any("error", err))
		}
		return
	}
	_, err = RunTask(app, ctx, task.Content)
	scheduleTask(app, ctx, taskID, task.Content)
	if err != nil {
		app.Logger.Error("Error running task", slog.String("task", taskID), slog.Any("error", err))
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
	app.Logger.Info("Task run completed", slog.String("taskID", taskID))
}

func scheduleExistingTasks(app *App, ctx context.Context) error {
	app.Logger.Info("Loading scheduled tasks")
	// Load scheduled task runs from database
	rows, err := app.DB.QueryxContext(
		ctx,
		`SELECT w.task_id, w.next_run_at, a.content
			FROM `+app.Schema+`.task_runs w
			JOIN `+app.Schema+`.apps a ON w.task_id = a.id
			WHERE next_run_at IS NOT NULL`,
	)
	if err != nil {
		rows.Close()
		return fmt.Errorf("failed to query scheduled tasks: %w", err)
	}
	for rows.Next() {
		var taskID string
		var nextRunAt time.Time
		var content string
		if err := rows.Scan(&taskID, &nextRunAt, &content); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan scheduled task: %w", err)
		}
		scheduleTask(app, ctx, taskID, content)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating over scheduled tasks: %w", err)
	}
	app.Logger.Info("Scheduled tasks loaded successfully")
	return nil
}
