// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"fmt"
	"time"
)

type AppRecord struct {
	ID         string    `db:"id" json:"id"`
	Path       string    `db:"path" json:"path"`
	Name       string    `db:"name" json:"name"`
	Content    string    `db:"content" json:"content"`
	CreatedAt  time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt  time.Time `db:"updated_at" json:"updatedAt"`
	CreatedBy  *string   `db:"created_by" json:"createdBy,omitempty"`
	UpdatedBy  *string   `db:"updated_by" json:"updatedBy,omitempty"`
	Visibility *string   `db:"visibility" json:"visibility,omitempty"`
	TaskInfo   any       `db:"task_info" json:"taskInfo,omitempty"`
	Type       string    `db:"type" json:"type"`
}

type AppListResponse struct {
	Apps []AppRecord `json:"apps"`
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

	apps := []AppRecord{}
	optionalFilter := ""
	if app.NoTasks {
		optionalFilter = "WHERE type = 'dashboard'"
	}
	err := app.Sqlite.SelectContext(ctx, &apps,
		fmt.Sprintf(`SELECT a.id, a.path, a.name, a.content, a.created_at, a.updated_at, a.created_by, a.updated_by, a.visibility, a.type,
			CASE WHEN a.type = 'task' THEN
				json_object(
					'lastRunAt', t.last_run_at,
					'lastRunSuccess', t.last_run_success,
					'lastRunDuration', t.last_run_duration,
					'nextRunAt', t.next_run_at
				)
			END AS task_info
			FROM apps a
			LEFT JOIN task_runs t ON t.task_id = a.id AND a.type = 'task'
			%s
			ORDER BY %s %s`, optionalFilter, orderBy, order))
	if err != nil {
		err = fmt.Errorf("error listing apps: %w", err)
	}
	for i := range apps {
		if app.NoPublicSharing && apps[i].Visibility != nil && *apps[i].Visibility == "public" {
			apps[i].Visibility = nil
		}
		if app.NoPasswordProtectedSharing && apps[i].Visibility != nil && *apps[i].Visibility == "password-protected" {
			apps[i].Visibility = nil
		}
	}
	return AppListResponse{Apps: apps}, err
}
