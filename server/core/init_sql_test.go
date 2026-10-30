// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/jmoiron/sqlx"
	"github.com/nrednav/cuid2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestInitSQLConfigurationOption(t *testing.T) {
	tempDir := t.TempDir()
	duckdbPath := filepath.Join(tempDir, "test.duckdb")

	dbName := cuid2.Generate()
	sdb, err := sqlx.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName))
	require.NoError(t, err)
	defer sdb.Close()

	ddb, err := sqlx.Open("duckdb", duckdbPath)
	require.NoError(t, err)
	defer ddb.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// initSQL includes setting a configuration option
	initSQL := "SET threads = 2;\n"

	app, err := New(
		"test",
		"node-1",
		"dev",
		sdb,
		ddb,
		duckdbPath,
		"",
		"",
		"",
		false,
		initSQL,
		"_shaper",
		logger,
		"/",
		15*time.Minute,
		24*time.Hour,
		7*24*time.Hour,
		false,
		false,
		true,
		false,
		false,
		false,
		true,
		"shaper.ingest.",
		"shaper.state.",
		"state-stream",
		0,
		"config-bucket",
		"dash-bucket",
		24*time.Hour,
		"dl-bucket",
		10*time.Minute,
		"tasks-stream",
		"shaper.tasks.",
		"task-consumer",
		"results-stream",
		"shaper.task-results.",
		0,
		"broadcast",
		"",
		"",
		"",
		"",
	)
	require.NoError(t, err)
	require.NotNil(t, app)

	// Verify the option was set
	var threads int64
	err = ddb.Get(&threads, "SELECT current_setting('threads')::BIGINT")
	require.NoError(t, err)
	assert.Equal(t, int64(2), threads)

	// Verify configuration is locked afterwards
	_, err = ddb.Exec("SET threads = 4")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "the configuration has been locked")
}

func TestInitSQLConfigurationOptionMemoryMode(t *testing.T) {
	dbName := cuid2.Generate()
	sdb, err := sqlx.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName))
	require.NoError(t, err)
	defer sdb.Close()

	ddb, err := sqlx.Open("duckdb", ":memory:")
	require.NoError(t, err)
	defer ddb.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	initSQL := "SET threads = 2;\n"

	app, err := New(
		"test",
		"node-1",
		"dev",
		sdb,
		ddb,
		":memory:",
		"",
		"",
		"",
		false,
		initSQL,
		"_shaper",
		logger,
		"/",
		15*time.Minute,
		24*time.Hour,
		7*24*time.Hour,
		false,
		false,
		true,
		false,
		false,
		false,
		true,
		"shaper.ingest.",
		"shaper.state.",
		"state-stream",
		0,
		"config-bucket",
		"dash-bucket",
		24*time.Hour,
		"dl-bucket",
		10*time.Minute,
		"tasks-stream",
		"shaper.tasks.",
		"task-consumer",
		"results-stream",
		"shaper.task-results.",
		0,
		"broadcast",
		"",
		"",
		"",
		"",
	)
	require.NoError(t, err)
	require.NotNil(t, app)

	// In memory mode, GetDuckDB provisions a fresh DB per session and runs initSQL
	db, cleanup, err := app.GetDuckDB(context.Background())
	require.NoError(t, err)
	defer cleanup()

	var threads int64
	err = db.Get(&threads, "SELECT current_setting('threads')::BIGINT")
	require.NoError(t, err)
	assert.Equal(t, int64(2), threads)

	// Verify configuration is locked on the memory DB
	_, err = db.Exec("SET threads = 4")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "the configuration has been locked")

	// Verify the base ddb passed to New is also locked
	_, err = ddb.Exec("SET threads = 4")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "the configuration has been locked")
}

func TestNewAdminEmailValidation(t *testing.T) {
	dbName := cuid2.Generate()
	sdb, err := sqlx.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName))
	require.NoError(t, err)
	defer sdb.Close()

	ddb, err := sqlx.Open("duckdb", ":memory:")
	require.NoError(t, err)
	defer ddb.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	newApp := func(ssoLoginURL, adminEmail, adminPassword string) (*App, error) {
		return New(
			"test",
			"node-1",
			"dev",
			sdb,
			ddb,
			":memory:",
			"",
			"",
			"",
			false,
			"",
			"_shaper",
			logger,
			"/",
			15*time.Minute,
			24*time.Hour,
			7*24*time.Hour,
			false,
			false,
			true,
			false,
			false,
			false,
			true,
			"shaper.ingest.",
			"shaper.state.",
			"state-stream",
			0,
			"config-bucket",
			"dash-bucket",
			24*time.Hour,
			"dl-bucket",
			10*time.Minute,
			"tasks-stream",
			"shaper.tasks.",
			"task-consumer",
			"results-stream",
			"shaper.task-results.",
			0,
			"broadcast",
			ssoLoginURL,
			"",
			adminEmail,
			adminPassword,
		)
	}

	t.Run("fails when only adminEmail is set", func(t *testing.T) {
		_, err := newApp("", "admin@example.com", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "both admin-email and admin-password must be set")
	})

	t.Run("fails when only adminPassword is set", func(t *testing.T) {
		_, err := newApp("", "", "secret")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "both admin-email and admin-password must be set")
	})

	t.Run("fails when adminEmail is not a valid email address", func(t *testing.T) {
		for _, invalid := range []string{"admin", "not-an-email@", "@example.com", "Admin <admin@example.com>"} {
			_, err := newApp("", invalid, "secret")
			require.Error(t, err, "expected error for invalid email %q", invalid)
			assert.Contains(t, err.Error(), "admin-email must be a valid email address")
		}
	})

	t.Run("fails when adminEmail and ssoLoginURL are both set", func(t *testing.T) {
		_, err := newApp("https://sso.example.com/login", "admin@example.com", "secret")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot set admin-email and sso-login-url at the same time")
	})

	t.Run("succeeds and sets LoginRequired true when both adminEmail and adminPassword are set", func(t *testing.T) {
		app, err := newApp("", "admin@example.com", "secret")
		require.NoError(t, err)
		require.NotNil(t, app)
		assert.True(t, app.LoginRequired)
		assert.Equal(t, "admin@example.com", app.AdminEmail)
		assert.Equal(t, "secret", app.AdminPassword)
	})
}

