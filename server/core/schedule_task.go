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

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type TaskResultPayload struct {
	TaskID        string        `json:"taskId"`
	StartedAt     time.Time     `json:"startedAt"`
	Success       bool          `json:"success"`
	TotalDuration time.Duration `json:"totalDurationMs"`
	NextRunAt     *time.Time    `json:"nextRunAt,omitempty"`
	NextRunType   string        `json:"nextRunType,omitempty"` // "single" or "all"
}

type BroadCastPayload struct {
	NodeID  string `json:"nodeId"`
	Content string `json:"content"`
}

func getNextTaskRun(app *App, ctx context.Context, content string) (*time.Time, string, error) {
	// Run first query
	cleanContent := util.StripSQLComments(content)
	sqls, err := splitSQLQueries(cleanContent)
	if err != nil {
		return nil, "", fmt.Errorf("failed to split SQL queries: %w", err)
	}
	if len(sqls) == 0 {
		return nil, "", fmt.Errorf("no SQL queries found in task content")
	}
	conn, err := app.DuckDB.Connx(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get database connection: %w", err)
	}
	defer conn.Close()
	sqlString := strings.TrimSpace(sqls[0])
	if sqlString == "" {
		return nil, "", fmt.Errorf("first SQL query is empty")
	}
	rows, err := conn.QueryxContext(ctx, sqlString)
	if err != nil {
		return nil, "", fmt.Errorf("failed to execute first SQL query: %w", err)
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get column types: %w", err)
	}
	result := [][]any{}
	for rows.Next() {
		row, err := rows.SliceScan()
		if err != nil {
			rows.Close()
			return nil, "", fmt.Errorf("failed to scan row: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("error iterating over rows: %w", err)
	}
	scheduleType, isSchedule := getScheduleColumn(colTypes, result)
	if !isSchedule {
		return nil, "", fmt.Errorf("first SQL query is not a SCHEDULE query")
	}
	reloadValue := getReloadValue(result)
	if reloadValue <= 0 {
		return nil, scheduleType, nil
	}
	nextRunAt := time.UnixMilli(reloadValue)
	return &nextRunAt, scheduleType, nil
}

func scheduleAndTrackNextTaskRun(app *App, ctx context.Context, taskID string, content string) {
	unscheduleTask(app, taskID)
	nextRunAt, scheduleType, err := getNextTaskRun(app, ctx, content)
	if err != nil {
		app.Logger.Error("Error getting next task run", slog.Any("error", err), slog.String("task", taskID))
	}
	if nextRunAt == nil {
		return
	}
	// schedule task
	scheduleTask(app, ctx, taskID, *nextRunAt, scheduleType)
	// set next_run_at in DB
	_, err = app.Sqlite.ExecContext(
		ctx,
		`INSERT INTO task_runs
		(task_id, next_run_at, next_run_type)
		VALUES ($1, $2, $3)
		ON CONFLICT(task_id) DO UPDATE SET next_run_at = $2, next_run_type = $3`,
		taskID,
		nextRunAt,
		scheduleType,
	)
	if err != nil {
		app.Logger.Error("Error inserting next run time into DB", slog.Any("error", err), slog.String("task", taskID))
		return
	}
	app.Logger.Info("Scheduled task", slog.String("task", taskID), slog.Time("next", *nextRunAt))
}

// Schedule a timer, then publish task to NATS.
// Uses task ID + run time as NATS MSG ID to deduplicate messages
// since we have all nodes publishing the same message.
// All nodes do the scheduling so nodes can come and go.
func scheduleTask(app *App, ctx context.Context, taskID string, runAt time.Time, runType string) {
	t := time.AfterFunc(time.Until(runAt), func() {
		if runType == "all" {
			runAll(app, taskID, runAt)
			return
		}
		if runType != "single" {
			app.Logger.Warn("Invalid run type for task. Assuming single...", slog.String("task", taskID), slog.String("type", runType))
		}
		app.Logger.Info("Dispatching task", slog.String("task", taskID))
		msgID := fmt.Sprintf("%s-%d", taskID, runAt.UnixMilli())
		subject := app.TasksSubjectPrefix + taskID
		// Send message to NATS
		_, err := app.JetStream.Publish(ctx, subject, []byte{}, jetstream.WithMsgID(msgID))
		if err != nil {
			// Expected message dedup error
			if !strings.Contains(err.Error(), "code=503 err_code=10077") {
				app.Logger.Error("failed to publish task run message", slog.Any("error", err), slog.String("subject", subject))
			}
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
	msgID := msg.Headers().Get(jetstream.MsgIDHeader)
	app.Logger.Info("Running task", slog.String("task", taskID), slog.String("type", "single"))
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
	}
	// Ack even on error. Trying again next run.
	err = msg.Ack()
	if err != nil {
		app.Logger.Error("Error acking message", slog.Any("error", err))
		return
	}
	nextRunAt, scheduleType, err := getNextTaskRun(app, ctx, task.Content)
	if err != nil {
		app.Logger.Error("Error getting next task run", slog.String("task", taskID), slog.Any("error", err))
	}
	var totalDuration time.Duration
	for _, queryResult := range runResult.Queries {
		totalDuration += time.Duration(queryResult.Duration) * time.Millisecond
	}
	err = publishTaskRunResult(app, ctx, msgID, TaskResultPayload{
		TaskID:        taskID,
		StartedAt:     time.UnixMilli(runResult.StartedAt),
		Success:       runResult.Success,
		TotalDuration: totalDuration,
		NextRunAt:     nextRunAt,
		NextRunType:   scheduleType,
	})
	if err != nil {
		app.Logger.Error("Error publishing task run result", slog.Any("error", err), slog.String("task", taskID))
		return
	}
}

func runAll(app *App, taskID string, runTime time.Time) {
	msgID := fmt.Sprintf("%s-%d", taskID, runTime.UnixMilli())
	app.Logger.Info("Running task", slog.String("task", taskID), slog.String("type", "all"))
	ctx := ContextWithActor(context.Background(), &Actor{Type: ActorTask})
	task, err := GetTask(app, ctx, taskID)
	if err != nil {
		app.Logger.Error("Error getting task", slog.String("task", taskID), slog.Any("error", err))
		return
	}
	runResult, err := RunTask(app, ctx, task.Content)
	if err != nil {
		app.Logger.Error("Error running task", slog.String("task", taskID), slog.Any("error", err))
		return
	}
	nextRunAt, scheduleType, err := getNextTaskRun(app, ctx, task.Content)
	if err != nil {
		app.Logger.Error("Error getting next task run", slog.String("task", taskID), slog.Any("error", err))
	}
	var totalDuration time.Duration
	for _, queryResult := range runResult.Queries {
		totalDuration += time.Duration(queryResult.Duration) * time.Millisecond
	}
	err = publishTaskRunResult(app, ctx, msgID, TaskResultPayload{
		TaskID:        taskID,
		StartedAt:     time.UnixMilli(runResult.StartedAt),
		Success:       runResult.Success,
		TotalDuration: totalDuration,
		NextRunAt:     nextRunAt,
		NextRunType:   scheduleType,
	})
	if err != nil {
		app.Logger.Error("Error publishing task run result", slog.Any("error", err), slog.String("task", taskID))
	}
}

func publishTaskRunResult(app *App, ctx context.Context, msgID string, result TaskResultPayload) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal task result payload: %w", err)
	}
	_, err = app.JetStream.Publish(ctx, app.TaskResultsSubjectPrefix+result.TaskID, payload, jetstream.WithMsgID(msgID))
	if err != nil {
		return fmt.Errorf("failed to publish task result: %w", err)
	}
	return nil
}

func trackTaskRun(app *App, ctx context.Context, payload TaskResultPayload) {
	var nextRunSlogAttr slog.Attr
	var nextRunTypeSlogAttr slog.Attr
	if payload.NextRunAt != nil {
		nextRunSlogAttr = slog.Time("next_run", *payload.NextRunAt)
		nextRunTypeSlogAttr = slog.String("next_run_type", payload.NextRunType)
	}
	app.Logger.Info(
		"Task Result",
		slog.String("task", payload.TaskID),
		slog.Time("started_at", payload.StartedAt),
		slog.Bool("success", payload.Success),
		slog.Duration("duration", payload.TotalDuration),
		nextRunSlogAttr,
		nextRunTypeSlogAttr,
	)
	_, err := app.Sqlite.ExecContext(
		ctx,
		`INSERT INTO task_runs
			(task_id, last_run_at, last_run_success, last_run_duration, next_run_at, next_run_type)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT(task_id) DO UPDATE SET
				last_run_at = $2,
				last_run_success = $3,
				last_run_duration = $4,
				next_run_at = $5,
				next_run_type = $6`,
		payload.TaskID,
		payload.StartedAt,
		boolToInt(payload.Success),
		payload.TotalDuration.Milliseconds(),
		payload.NextRunAt,
		payload.NextRunType,
	)
	if err != nil {
		app.Logger.Error("Error saving task run", slog.Any("error", err), slog.String("task", payload.TaskID))
	}
}

func scheduleExistingTasks(app *App, ctx context.Context) error {
	app.Logger.Info("Loading scheduled tasks")
	// Load scheduled task runs from database
	rows, err := app.Sqlite.QueryxContext(
		ctx,
		`SELECT t.task_id, t.next_run_at, t.next_run_type
			FROM task_runs t
			WHERE next_run_at IS NOT NULL`,
	)
	if err != nil {
		rows.Close()
		return fmt.Errorf("failed to query scheduled tasks: %w", err)
	}
	for rows.Next() {
		var taskID string
		var nextRunAt string
		var nextRunType string
		if err := rows.Scan(&taskID, &nextRunAt, &nextRunType); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan scheduled task: %w", err)
		}
		t, err := time.Parse(time.RFC3339, nextRunAt)
		if err != nil {
			rows.Close()
			return fmt.Errorf("Error parsing time: %v", nextRunAt)
		}
		scheduleTask(app, ctx, taskID, t, nextRunType)
		app.Logger.Info("Scheduled task", slog.String("task", taskID), slog.Time("next", t), slog.String("type", nextRunType))
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
		scheduleTask(app, ctx, payload.TaskID, *payload.NextRunAt, payload.NextRunType)
	}
	trackTaskRun(app, ctx, payload)
	if err := msg.Ack(); err != nil {
		app.Logger.Error("Error acking message", slog.Any("error", err), slog.String("subject", msg.Subject()))
		return
	}
}

func ManualTaskRun(app *App, ctx context.Context, content string) (TaskResult, error) {
	_, scheduleType, err := getNextTaskRun(app, ctx, content)
	if err != nil {
		return TaskResult{}, fmt.Errorf("failed to get next task run: %w", err)
	}
	if scheduleType == "all" {
		broadcastManualTask(app, ctx, content)
	}
	return RunTask(app, ctx, content)
}

func broadcastManualTask(app *App, ctx context.Context, content string) {
	payload, err := json.Marshal(BroadCastPayload{
		// The ID is used to identify the current node but we regenerate it every time the node restarts. This is fine since broadcasts are not presistent.
		NodeID:  app.NodeID,
		Content: content,
	})
	if err != nil {
		app.Logger.Error("Error marshalling broadcast payload", slog.Any("error", err))
		return
	}
	err = app.NATSConn.Publish(app.TaskBroadcastSubject, payload)
	if err != nil {
		app.Logger.Error("Error publishing task broadcast", slog.Any("error", err), slog.String("subject", app.TaskBroadcastSubject))
		return
	}
}

func (app *App) HandleTaskBroadcast(msg *nats.Msg) {
	var payload BroadCastPayload
	err := json.Unmarshal(msg.Data, &payload)
	if err != nil {
		app.Logger.Error("Error unmarshalling task broadcast", slog.Any("error", err), slog.String("subject", msg.Subject))
		return
	}
	if payload.NodeID == app.NodeID {
		return
	}
	app.Logger.Info("Running broadcasted task", slog.String("origin", payload.NodeID))
	ctx := ContextWithActor(context.Background(), &Actor{Type: ActorTask})
	_, err = RunTask(app, ctx, payload.Content)
	if err != nil {
		app.Logger.Error("Error running broadcasted task", slog.Any("error", err))
		return
	}
}
