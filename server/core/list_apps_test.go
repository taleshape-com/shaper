// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"shaper/server/api"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func setupListAppsTestApp(t *testing.T) *App {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	schema := []string{
		`CREATE TABLE folders (
			id TEXT PRIMARY KEY,
			parent_folder_id TEXT,
			name TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_by TEXT,
			updated_by TEXT
		);`,
		`CREATE TABLE apps (
			id TEXT PRIMARY KEY,
			folder_id TEXT,
			name TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_by TEXT,
			updated_by TEXT,
			visibility TEXT,
			type TEXT NOT NULL
		);`,
		`CREATE TABLE task_runs (
			task_id TEXT PRIMARY KEY,
			last_run_at TIMESTAMP,
			last_run_success BOOLEAN,
			last_run_duration INTEGER,
			next_run_at TIMESTAMP
		);`,
	}

	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("failed to exec schema: %v", err)
		}
	}

	now := time.Now().UTC()

	if _, err := db.Exec(`INSERT INTO folders (id, parent_folder_id, name, created_at, updated_at) VALUES
		('folderA', NULL, 'FolderA', ?, ?),
		('folderB', 'folderA', 'FolderB', ?, ?)`, now, now, now, now); err != nil {
		t.Fatalf("failed to seed folders: %v", err)
	}

	appInserts := []struct {
		id       string
		folderID *string
		name     string
	}{
		{id: "dash-root", folderID: nil, name: "Root Dashboard"},
		{id: "dash-a", folderID: ptr("folderA"), name: "Folder A Dashboard"},
		{id: "dash-b", folderID: ptr("folderB"), name: "Folder B Dashboard"},
	}

	for _, app := range appInserts {
		if _, err := db.Exec(`INSERT INTO apps (id, folder_id, name, content, created_at, updated_at, type)
			VALUES (?, ?, ?, '', ?, ?, 'dashboard')`,
			app.id, app.folderID, app.name, now, now); err != nil {
			t.Fatalf("failed to seed apps: %v", err)
		}
	}

	return &App{
		Sqlite: db,
	}
}

func ptr[T any](v T) *T {
	return &v
}

func dashboardsOnly(items []api.App) []api.App {
	var dashboards []api.App
	for _, item := range items {
		if item.Type != "_folder" {
			dashboards = append(dashboards, item)
		}
	}
	return dashboards
}

func TestListAppsRecursiveIncludesSubfolders(t *testing.T) {
	app := setupListAppsTestApp(t)
	ctx := context.Background()

	resp, err := ListApps(app, ctx, ListAppsOptions{
		Path:  "/",
		Sort:  "name",
		Order: "asc",
	})
	if err != nil {
		t.Fatalf("ListApps non-recursive: %v", err)
	}

	dashboards := dashboardsOnly(resp.Apps)
	if len(dashboards) != 1 {
		t.Fatalf("expected 1 root dashboard, got %d", len(dashboards))
	}
	if dashboards[0].ID != "dash-root" {
		t.Fatalf("expected dash-root, got %s", dashboards[0].ID)
	}

	respRecursive, err := ListApps(app, ctx, ListAppsOptions{
		Path:              "/",
		IncludeSubfolders: true,
		Sort:              "name",
		Order:             "asc",
	})
	if err != nil {
		t.Fatalf("ListApps recursive: %v", err)
	}

	dashboards = dashboardsOnly(respRecursive.Apps)
	if len(dashboards) != 3 {
		t.Fatalf("expected 3 dashboards with recursion, got %d", len(dashboards))
	}

	found := map[string]bool{}
	for _, d := range dashboards {
		found[d.ID] = true
	}
	for _, id := range []string{"dash-root", "dash-a", "dash-b"} {
		if !found[id] {
			t.Fatalf("missing dashboard %s in recursive results", id)
		}
	}
}

func TestListAppsPagination(t *testing.T) {
	app := setupListAppsTestApp(t)
	ctx := context.Background()

	resp, err := ListApps(app, ctx, ListAppsOptions{
		Path:              "/",
		IncludeSubfolders: true,
		Sort:              "name",
		Order:             "asc",
		Limit:             2,
		Offset:            3,
	})
	if err != nil {
		t.Fatalf("ListApps paginated: %v", err)
	}

	dashboards := dashboardsOnly(resp.Apps)
	if len(dashboards) != 2 {
		t.Fatalf("expected 2 dashboards from pagination, got %d", len(dashboards))
	}

	if dashboards[0].Name != "Folder B Dashboard" || dashboards[1].Name != "Root Dashboard" {
		t.Fatalf("unexpected pagination order: %+v", dashboards)
	}
}
