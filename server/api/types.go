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
	NextRunType     string     `json:"nextRunType,omitempty"`
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

// SchemaResponse represents the database schema.
type SchemaResponse struct {
	Databases  []Database  `json:"databases"`
	Extensions []Extension `json:"extensions,omitempty"`
	Secrets    []Secret    `json:"secrets,omitempty"`
}

// Database represents a database in the schema.
type Database struct {
	Name    string   `json:"name"`
	Schemas []Schema `json:"schemas"`
}

// Schema represents a schema in the database.
type Schema struct {
	Name   string  `json:"name"`
	Tables []Table `json:"tables"`
	Views  []View  `json:"views"`
	Enums  []Enum  `json:"enums"`
}

// Table represents a table in the schema.
type Table struct {
	Name        string       `json:"name"`
	Comment     string       `json:"comment,omitempty"`
	Columns     []Column     `json:"columns"`
	Constraints []Constraint `json:"constraints,omitempty"`
}

// View represents a view in the schema.
type View struct {
	Name       string   `json:"name"`
	Comment    string   `json:"comment,omitempty"`
	Columns    []Column `json:"columns"`
	Definition string   `json:"definition"` // The SQL DDL
}

// Column represents a column in a table or view.
type Column struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Nullable bool    `json:"nullable"`
	Default  *string `json:"default,omitempty"`
	Comment  string  `json:"comment,omitempty"`
}

// Constraint represents a constraint on a table.
type Constraint struct {
	Name            string   `json:"name"`
	Type            string   `json:"type"` // PRIMARY KEY, FOREIGN KEY, CHECK, UNIQUE
	Columns         []string `json:"columns"`
	ReferencedTable *string  `json:"referencedTable,omitempty"`
	ReferencedCols  []string `json:"referencedCols,omitempty"`
	CheckExpression *string  `json:"checkExpression,omitempty"`
}

// Enum represents an enum type.
type Enum struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// Extension represents a DuckDB extension.
type Extension struct {
	Name        string `json:"name"`
	Loaded      bool   `json:"loaded"`
	Installed   bool   `json:"installed"`
	Description string `json:"description,omitempty"`
}

// Secret represents a DuckDB secret.
type Secret struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Provider string   `json:"provider"`
	Scope    []string `json:"scope,omitempty"`
}
