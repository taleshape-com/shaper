// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"shaper/server/comms"
	"shaper/server/core"
	"testing"
	"time"

	"github.com/duckdb/duckdb-go/v2"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func setupTestApp(t *testing.T) (*core.App, func()) {
	t.Helper()

	dataDir := t.TempDir()
	sqliteDBFile := filepath.Join(dataDir, "test.sqlite")
	sqliteDbx, err := sqlx.Connect("sqlite", sqliteDBFile)
	if err != nil {
		t.Fatalf("failed to connect sqlite: %v", err)
	}

	duckdbConnector, err := duckdb.NewConnector("", nil)
	if err != nil {
		t.Fatalf("failed to create duckdb connector: %v", err)
	}
	duckdbSqlDb := sql.OpenDB(duckdbConnector)
	duckdbSqlxDb := sqlx.NewDb(duckdbSqlDb, "duckdb")

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	app, err := core.New(
		"shaper-test",
		"test-node",
		"dev",
		sqliteDbx,
		duckdbSqlxDb,
		":memory:",
		"",
		"",
		"",
		"_shaper",
		logger,
		"/",
		15*time.Minute,
		24*time.Hour,
		7*24*time.Hour,
		false,
		false,
		true, // noTasks
		false,
		true, // noChromeSandbox
		"shaper.ingest.",
		"shaper.state.",
		"shaper-state-test-"+filepath.Base(dataDir),
		0,
		"shaper-config-test-"+filepath.Base(dataDir),
		"shaper-tmp-dashboards-test-"+filepath.Base(dataDir),
		24*time.Hour,
		"shaper-downloads-test-"+filepath.Base(dataDir),
		10*time.Minute,
		"shaper-tasks-test-"+filepath.Base(dataDir),
		"shaper.tasks.",
		"shaper-task-queue-consumer-test-"+filepath.Base(dataDir),
		"shaper-task-results-test-"+filepath.Base(dataDir),
		"shaper.task-results.",
		0,
		"shaper.task-broadcast",
		"",
		"",
	)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	natsDir := filepath.Join(dataDir, "nats")
	c, err := comms.New(comms.Config{
		Logger:              logger.WithGroup("nats"),
		Host:                "127.0.0.1",
		Port:                0,
		JSDir:               natsDir,
		Sqlite:              sqliteDbx,
		IngestSubjectPrefix: "shaper.ingest.",
	})
	if err != nil {
		t.Fatalf("failed to start comms: %v", err)
	}

	if err := app.Init(c.Conn); err != nil {
		c.Close()
		t.Fatalf("failed to init app with NATS: %v", err)
	}

	cleanup := func() {
		app.Close()
		c.Close()
		sqliteDbx.Close()
		duckdbSqlxDb.Close()
	}

	return app, cleanup
}

func TestDeployOnStartup(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	ctx := context.Background()

	// Test 1: non-existent directory
	err := DeployOnStartup(ctx, app, "/non/existent/dir/path/xyz")
	if err == nil {
		t.Fatalf("expected error for non-existent directory")
	}

	// Test 2: valid directory with a dashboard
	tmpDir := t.TempDir()
	dashPath := filepath.Join(tmpDir, "sales.dashboard.sql")
	dashContent := "-- shaperid:dash123\n\nSELECT * FROM sales;"
	if err := os.WriteFile(dashPath, []byte(dashContent), 0o644); err != nil {
		t.Fatalf("failed to write dashboard file: %v", err)
	}

	if err := DeployOnStartup(ctx, app, tmpDir); err != nil {
		t.Fatalf("DeployOnStartup failed: %v", err)
	}

	// Verify dashboard in DB
	resp, err := core.ListApps(app, ctx, core.ListAppsOptions{IncludeContent: true, Path: "/"})
	if err != nil {
		t.Fatalf("ListApps failed: %v", err)
	}
	if len(resp.Apps) != 1 {
		t.Fatalf("expected 1 app in DB, got %d", len(resp.Apps))
	}
	if resp.Apps[0].ID != "dash123" {
		t.Fatalf("expected app ID 'dash123', got %q", resp.Apps[0].ID)
	}
	if resp.Apps[0].Name != "sales" {
		t.Fatalf("expected app Name 'sales', got %q", resp.Apps[0].Name)
	}
	if resp.Apps[0].Content != "SELECT * FROM sales;" {
		t.Fatalf("expected stripped content, got %q", resp.Apps[0].Content)
	}

	// Test 3: re-run DeployOnStartup without changes
	if err := DeployOnStartup(ctx, app, tmpDir); err != nil {
		t.Fatalf("DeployOnStartup second run failed: %v", err)
	}

	// Test 4: update dashboard content locally and re-deploy
	updatedContent := "-- shaperid:dash123\n\nSELECT id, amount FROM sales;"
	if err := os.WriteFile(dashPath, []byte(updatedContent), 0o644); err != nil {
		t.Fatalf("failed to update dashboard file: %v", err)
	}

	if err := DeployOnStartup(ctx, app, tmpDir); err != nil {
		t.Fatalf("DeployOnStartup after update failed: %v", err)
	}

	resp, err = core.ListApps(app, ctx, core.ListAppsOptions{IncludeContent: true, Path: "/"})
	if err != nil {
		t.Fatalf("ListApps failed: %v", err)
	}
	if resp.Apps[0].Content != "SELECT id, amount FROM sales;" {
		t.Fatalf("expected updated content in DB, got %q", resp.Apps[0].Content)
	}
}
