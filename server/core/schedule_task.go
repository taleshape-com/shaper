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

func scheduleTask(app *App, ctx context.Context, taskID, content string) *time.Time {
	// cancel previous timer
	unscheduleTask(app, taskID)
	// Run first query
	cleanContent := util.StripSQLComments(content)
	sqls, err := splitSQLQueries(cleanContent)
	if err != nil {
		app.Logger.Error("Error splitting SQL queries", slog.Any("error", err), slog.String("task", taskID))
		return nil
	}
	if len(sqls) == 0 {
		app.Logger.Error("No SQL queries found in task content", slog.String("task", taskID))
		return nil
	}
	conn, err := app.DB.Connx(ctx)
	if err != nil {
		app.Logger.Error("Error getting database connection", slog.Any("error", err), slog.String("task", taskID))
		return nil
	}
	defer conn.Close()
	sqlString := strings.TrimSpace(sqls[0])
	if sqlString == "" {
		app.Logger.Error("First SQL query is empty", slog.String("task", taskID))
		return nil
	}
	rows, err := conn.QueryxContext(ctx, sqlString)
	if err != nil {
		app.Logger.Error("Error executing first SQL query", slog.Any("error", err), slog.String("task", taskID), slog.String("query", sqlString))
		return nil
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		app.Logger.Error("Error getting column types", slog.Any("error", err), slog.String("task", taskID))
		return nil
	}
	result := [][]any{}
	for rows.Next() {
		row, err := rows.SliceScan()
		if err != nil {
			app.Logger.Error("Error scanning row", slog.Any("error", err), slog.String("task", taskID))
			rows.Close()
			return nil
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		app.Logger.Error("Error iterating over rows", slog.Any("error", err), slog.String("task", taskID))
		return nil
	}
	if !isSchedule(colTypes, result) {
		app.Logger.Error("First SQL query is not a SCHEDULE query", slog.String("task", taskID), slog.String("query", sqlString))
		return nil
	}
	reloadValue := getReloadValue(result)
	if reloadValue <= 0 {
		app.Logger.Info("Task SCHEDULE set to NULL, skipping scheduling", slog.String("task", taskID))
		return nil
	}
	nextRunAt := time.UnixMilli(reloadValue)
	// schedule for later
	runTaskAt(app, ctx, taskID, nextRunAt)
	return &nextRunAt
}

func scheduleAndTrackNextTaskRun(app *App, ctx context.Context, taskID string, content string) {
	nextRunAt := scheduleTask(app, ctx, taskID, content)
	if nextRunAt == nil {
		return
	}
	// set next_run_at in DB
	_, err := app.DB.ExecContext(
		ctx,
		`INSERT INTO `+app.Schema+`.task_runs
		(task_id, next_run_at)
		VALUES ($1, $2)
		ON CONFLICT DO UPDATE SET next_run_at = $2`,
		taskID,
		nextRunAt,
	)
	if err != nil {
		app.Logger.Error("Error inserting next run time into DB", slog.Any("error", err), slog.String("task", taskID))
		return
	}
	app.Logger.Info("Scheduled task", slog.String("task", taskID), slog.Time("next", *nextRunAt))
}

func runTaskAt(app *App, ctx context.Context, taskID string, runAt time.Time) {
	t := time.AfterFunc(time.Until(runAt), func() {
		app.Logger.Info("Dispatching task", slog.String("task", taskID))
		subject := app.TasksSubjectPrefix + taskID
		msgID := fmt.Sprintf("%s-%d", taskID, runAt.UnixMilli())
		// Send message to NATS
		_, err := app.JetStream.Publish(ctx, subject, []byte{}, jetstream.WithMsgID(msgID))
		if err != nil {
			app.Logger.Error("failed to publish task run message", slog.Any("error", err), slog.String("subject", subject))
		}
	})
	app.TaskTimers[taskID] = t
}

func unscheduleTask(app *App, taskID string) {
	if existingTimer, hasTimer := app.TaskTimers[taskID]; hasTimer {
		existingTimer.Stop()
		delete(app.TaskTimers, taskID)
	}
}

func (app *App) HandleTask(msg jetstream.Msg) {
	taskID := strings.TrimPrefix(msg.Subject(), app.TasksSubjectPrefix)
	app.Logger.Info("Running task", slog.String("task", taskID))
	ctx := ContextWithActor(context.Background(), &Actor{Type: ActorTask})
	task, err := GetTask(app, ctx, taskID)
	if err != nil {
		app.Logger.Error("Error getting task", slog.String("task", taskID), slog.Any("error", err))
		if err := msg.Nak(); err != nil {
			app.Logger.Error("Error nack message", slog.Any("error", err))
		}
		return
	}
	runResult, err := RunTask(app, ctx, task.Content)
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
	// TODO: Not sure if we need to call schedule Task here or just call runTaskAt with runResult.NextRunAt. The difference is if the runAt time is calulated before or after the task ran.
	nextRunAt := scheduleTask(app, ctx, taskID, task.Content)
	trackTaskRun(app, ctx, taskID, runResult, nextRunAt)
}

func trackTaskRun(app *App, ctx context.Context, taskID string, result TaskResult, nextRunAt *time.Time) {
	var totalDurationMs int64
	for _, queryResult := range result.Queries {
		totalDurationMs += queryResult.Duration
	}
	var nextRunSlogAttr slog.Attr
	if nextRunAt != nil {
		nextRunSlogAttr = slog.Time("next_run", *nextRunAt)
	}
	app.Logger.Info(
		"Task completed",
		slog.String("task", taskID),
		slog.Time("started_at", time.UnixMilli(result.StartedAt)),
		slog.Bool("success", result.Success),
		slog.Duration("duration", time.Duration(totalDurationMs)*time.Millisecond),
		nextRunSlogAttr,
	)
	_, err := app.DB.ExecContext(
		ctx,
		`INSERT INTO `+app.Schema+`.task_runs
			(task_id, last_run_at, last_run_success, last_run_duration, next_run_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT DO UPDATE SET
				last_run_at = $2,
				last_run_success = $3,
				last_run_duration = $4,
				next_run_at = $5`,
		taskID,
		time.UnixMilli(result.StartedAt),
		result.Success,
		fmt.Sprintf("%dms", totalDurationMs),
		nextRunAt,
	)
	if err != nil {
		app.Logger.Error("Error tracking task run", slog.Any("error", err), slog.String("task", taskID))
	}
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
		runTaskAt(app, ctx, taskID, nextRunAt)
		app.Logger.Info("Scheduled task", slog.String("task", taskID), slog.Time("next", nextRunAt))
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating over scheduled tasks: %w", err)
	}
	app.Logger.Info("All tasks scheduled")
	return nil
}
