// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nrednav/cuid2"
)

type Task struct {
	ID              string  `db:"id" json:"id"`
	Path            string  `db:"path" json:"path"`
	Name            string  `db:"name" json:"name"`
	Content         string  `db:"content" json:"content"`
	CreatedAt       int64   `db:"created_at" json:"createdAt"`
	UpdatedAt       int64   `db:"updated_at" json:"updatedAt"`
	CreatedBy       *string `db:"created_by" json:"createdBy,omitempty"`
	UpdatedBy       *string `db:"updated_by" json:"updatedBy,omitempty"`
	NextRunAt       *int64  `db:"next_run_at" json:"nextRunAt,omitempty"`
	LastRunAt       *int64  `db:"last_run_at" json:"lastRunAt,omitempty"`
	LastRunSuccess  *bool   `db:"last_run_success" json:"lastRunSuccess,omitempty"`
	LastRunDuration *int64  `db:"last_run_duration" json:"lastRunDuration,omitempty"`
}

type CreateTaskPayload struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedBy string    `json:"createdBy"`
}

type UpdateTaskContentPayload struct {
	ID        string    `json:"id"`
	TimeStamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
	UpdatedBy string    `json:"updatedBy"`
}

type UpdateTaskNamePayload struct {
	ID        string    `json:"id"`
	TimeStamp time.Time `json:"timestamp"`
	Name      string    `json:"name"`
	UpdatedBy string    `json:"updatedBy"`
}

type DeleteTaskPayload struct {
	ID        string    `json:"id"`
	TimeStamp time.Time `json:"timestamp"`
	// NOTE: Not used, but might want to log this in the future
	DeletedBy string `json:"deletedBy"`
}

func GetTask(app *App, ctx context.Context, id string) (Task, error) {
	var task Task
	err := app.Sqlite.GetContext(ctx, &task,
		`SELECT a.id, a.path, a.name, a.content, a.created_at, a.updated_at, a.created_by, a.updated_by,
						CAST(round(unixepoch(tr.next_run_at, 'subsec')*1000) AS INTEGER) AS next_run_at,
						CAST(round(unixepoch(tr.last_run_at, 'subsec')*1000) AS INTEGER) AS last_run_at,
		        tr.last_run_success,
		        tr.last_run_duration
		 FROM apps a
		 LEFT JOIN task_runs tr ON tr.task_id = a.id
		 WHERE a.id = $1 AND a.type = 'task'`, id)
	if err != nil {
		return task, fmt.Errorf("failed to get task: %w", err)
	}
	return task, nil
}

func CreateTask(app *App, ctx context.Context, name string, content string) (string, error) {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return "", fmt.Errorf("no actor in context")
	}
	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("task name cannot be empty")
	}
	id := cuid2.Generate()
	err := app.SubmitState(ctx, "create_task", CreateTaskPayload{
		ID:        id,
		Timestamp: time.Now(),
		Path:      "/",
		Name:      name,
		Content:   content,
		CreatedBy: actor.String(),
	})
	return id, err
}

func HandleCreateTask(app *App, data []byte) bool {
	var payload CreateTaskPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal create task payload", slog.Any("error", err))
		return false
	}
	ctx := ContextWithActor(context.Background(), ActorFromString(payload.CreatedBy))
	// Insert into DB
	_, err = app.Sqlite.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO apps (
			id, path, name, content, created_at, updated_at, created_by, updated_by, type
		) VALUES ($1, $2, $3, $4, $5, $5, $6, $6, 'task')`,
		payload.ID, payload.Path, payload.Name, payload.Content, payload.Timestamp, payload.CreatedBy,
	)
	if err != nil {
		app.Logger.Error("failed to insert task into DB", slog.Any("error", err))
		return false
	}
	scheduleAndTrackNextTaskRun(app, ctx, payload.ID, payload.Content)
	return true
}

func SaveTaskContent(app *App, ctx context.Context, id string, content string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}
	var count int
	err := app.Sqlite.GetContext(ctx, &count, `SELECT COUNT(*) FROM apps WHERE id = $1 AND type = 'task'`, id)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("task not found")
	}
	err = app.SubmitState(ctx, "update_task_content", UpdateTaskContentPayload{
		ID:        id,
		TimeStamp: time.Now(),
		Content:   content,
		UpdatedBy: actor.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to submit task content update: %w", err)
	}
	return nil
}

func HandleUpdateTaskContent(app *App, data []byte) bool {
	var payload UpdateTaskContentPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal update task content payload", slog.Any("error", err))
		return false
	}
	ctx := ContextWithActor(context.Background(), ActorFromString(payload.UpdatedBy))
	_, err = app.Sqlite.ExecContext(
		ctx,
		`UPDATE apps
		 SET content = $1, updated_at = $2, updated_by = $3
		 WHERE id = $4 AND type = 'task'`,
		payload.Content, payload.TimeStamp, payload.UpdatedBy, payload.ID)
	if err != nil {
		app.Logger.Error("failed to execute UPDATE statement", slog.Any("error", err))
		return false
	}
	scheduleAndTrackNextTaskRun(app, ctx, payload.ID, payload.Content)
	return true
}

func SaveTaskName(app *App, ctx context.Context, id string, name string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}
	var count int
	err := app.Sqlite.GetContext(ctx, &count, `SELECT COUNT(*) FROM apps WHERE id = $1 AND type = 'task'`, id)
	if err != nil {
		return fmt.Errorf("failed to query task: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("task not found")
	}
	err = app.SubmitState(ctx, "update_task_name", UpdateTaskNamePayload{
		ID:        id,
		TimeStamp: time.Now(),
		Name:      name,
		UpdatedBy: actor.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to submit task name update: %w", err)
	}
	return nil
}

func HandleUpdateTaskName(app *App, data []byte) bool {
	var payload UpdateTaskNamePayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal update task name payload", slog.Any("error", err))
		return false
	}
	_, err = app.Sqlite.Exec(
		`UPDATE apps
		 SET name = $1, updated_at = $2, updated_by = $3
		 WHERE id = $4 AND type = 'task'`,
		payload.Name, payload.TimeStamp, payload.UpdatedBy, payload.ID)
	if err != nil {
		app.Logger.Error("failed to execute UPDATE statement", slog.Any("error", err))
		return false
	}
	return true
}

func DeleteTask(app *App, ctx context.Context, id string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}
	var count int
	err := app.Sqlite.GetContext(ctx, &count, `SELECT COUNT(*) FROM apps WHERE id = $1 AND type = 'task'`, id)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("task not found")
	}
	err = app.SubmitState(ctx, "delete_task", DeleteTaskPayload{
		ID:        id,
		TimeStamp: time.Now(),
		DeletedBy: actor.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to submit task deletion: %w", err)
	}
	return nil
}

func HandleDeleteTask(app *App, data []byte) bool {
	var payload DeleteTaskPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal delete task payload", slog.Any("error", err))
		return false
	}
	unscheduleTask(app, payload.ID)
	_, err = app.Sqlite.Exec(
		`DELETE FROM apps WHERE id = $1 AND type = 'task'`, payload.ID)
	if err != nil {
		app.Logger.Error("failed to execute DELETE statement", slog.Any("error", err))
		return false
	}
	return true
}
