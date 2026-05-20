// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/nrednav/cuid2"
	"github.com/stretchr/testify/assert"
	_ "github.com/duckdb/duckdb-go/v2"
	_ "modernc.org/sqlite"
)

func setupTestApp(t *testing.T) (*App, func()) {
	// Use unique name to avoid interference between tests
	dbName := cuid2.Generate()
	sdb, err := sqlx.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName))
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	if err := initSQLite(sdb); err != nil {
		t.Fatalf("failed to init sqlite: %v", err)
	}

	ddb, err := sqlx.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	ddb.SetMaxOpenConns(1) // Ensure sharing for :memory:

	_, err = initDuckDB(ddb)
	if err != nil {
		t.Fatalf("failed to init duckdb: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := &App{
		Sqlite:         sdb,
		DuckDB:         ddb,
		DuckDBDSN:      "test", // Bypass ":memory:" check in GetDuckDB to reuse ddb
		InternalDBName: "",     // Simplify varPrefix
		Logger:         logger,
		TaskTimers:     make(map[string]*time.Timer),
	}

	cleanup := func() {
		sdb.Close()
		ddb.Close()
	}

	return app, cleanup
}

func TestInitTask(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("Basic init task", func(t *testing.T) {
		content := `
			SELECT 'init'::SCHEDULE;
			CREATE TABLE init_test (id INTEGER);
			INSERT INTO init_test VALUES (42);
		`
		result, err := RunTask(app, ctx, content)
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, int64(-1), result.NextRunAt)
		assert.Equal(t, "all", result.ScheduleType)

		// Verify table was created and data inserted (since no transaction for init)
		var val int32
		err = app.DuckDB.Get(&val, "SELECT id FROM init_test")
		assert.NoError(t, err)
		assert.Equal(t, int32(42), val)
	})

	t.Run("Init task with ATTACH", func(t *testing.T) {
		content := `
			SELECT 'init'::SCHEDULE;
			ATTACH ':memory:' AS extra_db;
			CREATE TABLE extra_db.test (val TEXT);
			INSERT INTO extra_db.test VALUES ('hello');
		`
		result, err := RunTask(app, ctx, content)
		assert.NoError(t, err)
		assert.True(t, result.Success)

		var val string
		err = app.DuckDB.Get(&val, "SELECT val FROM extra_db.test")
		assert.NoError(t, err)
		assert.Equal(t, "hello", val)
	})

	t.Run("Non-init task can ATTACH", func(t *testing.T) {
		content := `
			SELECT (now() + interval '1 day')::SCHEDULE;
			ATTACH ':memory:' AS non_init_db;
			CREATE TABLE non_init_db.test (val TEXT);
			INSERT INTO non_init_db.test VALUES ('global');
		`
		result, err := RunTask(app, ctx, content)
		assert.NoError(t, err)
		assert.True(t, result.Success)

		var val string
		err = app.DuckDB.Get(&val, "SELECT val FROM non_init_db.test")
		assert.NoError(t, err)
		assert.Equal(t, "global", val)
	})

	t.Run("Task with SET fails", func(t *testing.T) {
		content := `
			SELECT 'init'::SCHEDULE;
			SET memory_limit='2GB';
		`
		result, err := RunTask(app, ctx, content)
		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, *result.Queries[0].Error, "Statement not allowed in tasks")
	})

	t.Run("Task without schedule works", func(t *testing.T) {
		content := `
			CREATE TABLE no_schedule (val INT);
			INSERT INTO no_schedule VALUES (42);
		`
		result, err := RunTask(app, ctx, content)
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, int64(0), result.NextRunAt)

		var val int
		err = app.DuckDB.Get(&val, "SELECT val FROM no_schedule")
		assert.NoError(t, err)
		assert.Equal(t, 42, val)
	})
}

func TestScheduleExistingInitTasks(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	ctx := context.Background()

	// Create a folder and an init task
	folderID := "f1"
	_, err := app.Sqlite.Exec(`INSERT INTO folders (id, name, created_at, updated_at) VALUES (?, ?, datetime('now'), datetime('now'))`, folderID, "folder1")
	assert.NoError(t, err)

	task1ID := "t1"
	task1Content := "SELECT 'init'::SCHEDULE; CREATE TABLE t1_ran (val BOOLEAN); INSERT INTO t1_ran VALUES (true);"
	_, err = app.Sqlite.Exec(`INSERT INTO apps (id, type, folder_id, name, content, created_at, updated_at) VALUES (?, 'task', ?, 'task1', ?, datetime('now'), datetime('now'))`, task1ID, folderID, task1Content)
	assert.NoError(t, err)

	_, err = app.Sqlite.Exec(`INSERT INTO task_runs (task_id, next_run_type) VALUES (?, 'init')`, task1ID)
	assert.NoError(t, err)

	// Top level task
	task2ID := "t2"
	task2Content := "SELECT 'init'::SCHEDULE; CREATE TABLE t2_ran (val BOOLEAN); INSERT INTO t2_ran VALUES (true);"
	_, err = app.Sqlite.Exec(`INSERT INTO apps (id, type, name, content, created_at, updated_at) VALUES (?, 'task', 'task2', ?, datetime('now'), datetime('now'))`, task2ID, task2Content)
	assert.NoError(t, err)

	_, err = app.Sqlite.Exec(`INSERT INTO task_runs (task_id, next_run_type) VALUES (?, 'init')`, task2ID)
	assert.NoError(t, err)

	err = scheduleExistingTasks(app, ctx)
	assert.NoError(t, err)

	// Verify both ran
	var exists bool
	err = app.DuckDB.Get(&exists, "SELECT count(*) > 0 FROM t1_ran")
	assert.NoError(t, err)
	assert.True(t, exists)

	err = app.DuckDB.Get(&exists, "SELECT count(*) > 0 FROM t2_ran")
	assert.NoError(t, err)
	assert.True(t, exists)
}
