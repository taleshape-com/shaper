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

type TaskResultPayload struct {
	TaskID        string        `json:"taskId"`
	StartedAt     time.Time     `json:"startedAt"`
	Success       bool          `json:"success"`
	TotalDuration time.Duration `json:"totalDurationMs"`
	NextRunAt     *time.Time    `json:"nextRunAt,omitempty"`
}

func getNextTaskRun(app *App, ctx context.Context, taskID, content string) *time.Time {
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
	return &nextRunAt
}

func scheduleAndTrackNextTaskRun(app *App, ctx context.Context, taskID string, content string) {
	unscheduleTask(app, taskID)
	nextRunAt := getNextTaskRun(app, ctx, taskID, content)
	if nextRunAt == nil {
		return
	}
	// schedule task
	scheduleTask(app, ctx, taskID, *nextRunAt)
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

func scheduleTask(app *App, ctx context.Context, taskID string, runAt time.Time) {
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
	nextRunAt := getNextTaskRun(app, ctx, taskID, task.Content)
	var totalDuration time.Duration
	for _, queryResult := range runResult.Queries {
		totalDuration += time.Duration(queryResult.Duration) * time.Millisecond
	}
	err = publishTaskRunResult(app, ctx, TaskResultPayload{
		TaskID:        taskID,
		StartedAt:     time.UnixMilli(runResult.StartedAt),
		Success:       runResult.Success,
		TotalDuration: totalDuration,
		NextRunAt:     nextRunAt,
	})
	if err != nil {
		app.Logger.Error("Error publishing task run result", slog.Any("error", err), slog.String("task", taskID))
		return
	}
}

func publishTaskRunResult(app *App, ctx context.Context, result TaskResultPayload) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = app.JetStream.Publish(ctx, app.TaskResultsSubjectPrefix+result.TaskID, payload)
	if err != nil {
		return err
	}
	return nil
}

func trackTaskRun(app *App, ctx context.Context, payload TaskResultPayload) {
	var nextRunSlogAttr slog.Attr
	if payload.NextRunAt != nil {
		nextRunSlogAttr = slog.Time("next_run", *payload.NextRunAt)
	}
	app.Logger.Info(
		"Task completed",
		slog.String("task", payload.TaskID),
		slog.Time("started_at", payload.StartedAt),
		slog.Bool("success", payload.Success),
		slog.Duration("duration", payload.TotalDuration),
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
		payload.TaskID,
		payload.StartedAt,
		payload.Success,
		fmt.Sprintf("%dms", payload.TotalDuration.Milliseconds()),
		payload.NextRunAt,
	)
	if err != nil {
		app.Logger.Error("Error saving task run", slog.Any("error", err), slog.String("task", payload.TaskID))
	}
}

func scheduleExistingTasks(app *App, ctx context.Context) error {
	app.Logger.Info("Loading scheduled tasks")
	// Load scheduled task runs from database
	rows, err := app.DB.QueryxContext(
		ctx,
		`SELECT t.task_id, t.next_run_at, a.content
			FROM `+app.Schema+`.task_runs t
			JOIN `+app.Schema+`.apps a ON t.task_id = a.id
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
		scheduleTask(app, ctx, taskID, nextRunAt)
		app.Logger.Info("Scheduled task", slog.String("task", taskID), slog.Time("next", nextRunAt))
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating over scheduled tasks: %w", err)
	}
	app.Logger.Info("All tasks scheduled")
	return nil
}

func (app *App) HandleTaskResult(msg jetstream.Msg) {
	var payload TaskResultPayload
	err := json.Unmarshal(msg.Data(), &payload)
	if err != nil {
		app.Logger.Error("Error unmarshalling task result", slog.Any("error", err), slog.String("subject", msg.Subject()))
		if err := msg.Nak(); err != nil {
			app.Logger.Error("Error nack message", slog.Any("error", err))
		}
		return
	}
	unscheduleTask(app, payload.TaskID)
	ctx := ContextWithActor(context.Background(), &Actor{Type: ActorTask})
	if payload.NextRunAt != nil {
		scheduleTask(app, ctx, payload.TaskID, *payload.NextRunAt)
	}
	trackTaskRun(app, ctx, payload)
	if err := msg.Ack(); err != nil {
		app.Logger.Error("Error acking message", slog.Any("error", err), slog.String("subject", msg.Subject()))
		return
	}
}
