// SPDX-License-Identifier: MPL-2.0

package api

import "time"

// Request represents a deploy request containing multiple app operations.
type Request struct {
	Apps []AppRequest `json:"apps"`
}

// AppRequest represents a single app operation in a deploy request.
type AppRequest struct {
	Operation string        `json:"operation"`
	Type      string        `json:"type"`
	Data      DashboardData `json:"data"`
}

// DashboardData represents the data for a dashboard operation.
type DashboardData struct {
	ID      *string `json:"id"`
	Path    *string `json:"path"`
	Name    *string `json:"name"`
	Content *string `json:"content"`
}

// TaskInfo represents task execution information.
type TaskInfo struct {
	LastRunAt       *time.Time `json:"lastRunAt,omitempty"`
	LastRunSuccess  *bool      `json:"lastRunSuccess,omitempty"`
	LastRunDuration *int64     `json:"lastRunDuration,omitempty"`
	NextRunAt       *time.Time `json:"nextRunAt,omitempty"`
}

// App represents an app (dashboard or task) from the API.
type App struct {
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	FolderID   *string   `json:"folderId,omitempty"`
	Name       string    `json:"name"`
	Content    string    `json:"content,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	CreatedBy  *string   `json:"createdBy,omitempty"`
	UpdatedBy  *string   `json:"updatedBy,omitempty"`
	Visibility *string   `json:"visibility,omitempty"`
	TaskInfo   *TaskInfo `json:"taskInfo,omitempty"`
	Type       string    `json:"type"`
}

// AppsResponse represents a paginated response containing apps.
type AppsResponse struct {
	Apps     []App `json:"apps"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}
