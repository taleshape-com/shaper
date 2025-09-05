// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"fmt"
	"time"
)

type AppListItem struct {
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	CreatedBy  *string   `json:"createdBy,omitempty"`
	UpdatedBy  *string   `json:"updatedBy,omitempty"`
	Visibility *string   `json:"visibility,omitempty"`
	TaskInfo   *TaskInfo `json:"taskInfo,omitempty"`
	Type       string    `json:"type"`
}

type AppDbRecord struct {
	ID              string     `db:"id"`
	Path            string     `db:"path"`
	Name            string     `db:"name"`
	Content         string     `db:"content"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
	CreatedBy       *string    `db:"created_by"`
	UpdatedBy       *string    `db:"updated_by"`
	Visibility      *string    `db:"visibility"`
	Type            string     `db:"type"`
	LastRunAt       *time.Time `db:"last_run_at"`
	LastRunSuccess  *bool      `db:"last_run_success"`
	LastRunDuration *int64     `db:"last_run_duration"`
	NextRunAt       *time.Time `db:"next_run_at"`
}

type TaskInfo struct {
	LastRunAt       *time.Time `json:"lastRunAt,omitempty"`
	LastRunSuccess  *bool      `json:"lastRunSuccess,omitempty"`
	LastRunDuration *int64     `json:"lastRunDuration,omitempty"`
	NextRunAt       *time.Time `json:"nextRunAt,omitempty"`
}

type AppListResponse struct {
	Apps []AppListItem `json:"apps"`
}

func ListApps(app *App, ctx context.Context, sort string, order string) (AppListResponse, error) {
	var orderBy string
	switch sort {
	case "created":
		orderBy = "created_at"
	case "name":
		orderBy = "name"
	default:
		orderBy = "updated_at"
	}

	if order != "asc" && order != "desc" {
		order = "desc"
	}

	dbApps := []AppDbRecord{}
	optionalFilter := ""
	if app.NoTasks {
		optionalFilter = "WHERE type = 'dashboard'"
	}
	err := app.Sqlite.SelectContext(ctx, &dbApps,
		fmt.Sprintf(`SELECT
			a.id,
			a.path,
			a.name,
			a.content,
			a.created_at,
			a.updated_at,
			a.created_by,
			a.updated_by,
			a.visibility,
			a.type,
			t.last_run_at,
			t.last_run_success,
			t.last_run_duration,
			t.next_run_at
			FROM apps a
			LEFT JOIN task_runs t ON t.task_id = a.id AND a.type = 'task'
			%s
			ORDER BY %s %s`, optionalFilter, orderBy, order))
	if err != nil {
		err = fmt.Errorf("error listing apps: %w", err)
	}
	apps := make([]AppListItem, len(dbApps))
	for i, a := range dbApps {
		apps[i] = AppListItem{
			ID:         a.ID,
			Path:       a.Path,
			Name:       a.Name,
			Content:    a.Content,
			CreatedAt:  a.CreatedAt,
			UpdatedAt:  a.UpdatedAt,
			CreatedBy:  a.CreatedBy,
			UpdatedBy:  a.UpdatedBy,
			Visibility: a.Visibility,
			Type:       a.Type,
		}
		apps[i].TaskInfo = &TaskInfo{
			LastRunAt:       a.LastRunAt,
			LastRunSuccess:  a.LastRunSuccess,
			LastRunDuration: a.LastRunDuration,
			NextRunAt:       a.NextRunAt,
		}
		if app.NoPublicSharing && apps[i].Visibility != nil && *apps[i].Visibility == "public" {
			apps[i].Visibility = nil
		}
		if app.NoPasswordProtectedSharing && apps[i].Visibility != nil && *apps[i].Visibility == "password-protected" {
			apps[i].Visibility = nil
		}
	}
	return AppListResponse{Apps: apps}, err
}
