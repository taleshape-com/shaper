// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"io"
	"log/slog"
	"net/url"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func TestValidateDashboardDownload(t *testing.T) {
	sdb, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer sdb.Close()

	if err := initSQLite(sdb); err != nil {
		t.Fatalf("failed to init sqlite: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := &App{
		Sqlite:    sdb,
		DuckDBDSN: ":memory:",
		Logger:    logger,
	}

	ctx := context.Background()

	// Insert a dashboard that has a DOWNLOAD_PDF button
	_, err = sdb.Exec(`INSERT INTO apps (id, type, name, content, created_at, updated_at, visibility)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'), ?)`,
		"source-dash", "dashboard", "Source", "SELECT 'target-dash'::ID, 'Download'::DOWNLOAD_PDF", "public")
	if err != nil {
		t.Fatalf("failed to insert dashboard: %v", err)
	}

	t.Run("Valid download reference", func(t *testing.T) {
		allowed, err := ValidateDashboardDownload(app, ctx, "source-dash", "target-dash", url.Values{}, nil)
		assert.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("Invalid download reference", func(t *testing.T) {
		allowed, err := ValidateDashboardDownload(app, ctx, "source-dash", "other-dash", url.Values{}, nil)
		assert.NoError(t, err)
		assert.False(t, allowed)
	})

	// Dashboard with variable
	_, err = sdb.Exec(`INSERT INTO apps (id, type, name, content, created_at, updated_at, visibility)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'), ?)`,
		"source-var-dash", "dashboard", "Source Var", "SELECT getvariable('target_id')::ID, 'Download'::DOWNLOAD_PDF", "public")
	if err != nil {
		t.Fatalf("failed to insert dashboard: %v", err)
	}

	t.Run("Valid download reference with variable", func(t *testing.T) {
		allowed, err := ValidateDashboardDownload(app, ctx, "source-var-dash", "target-dash", url.Values{}, map[string]any{"target_id": "target-dash"})
		assert.NoError(t, err)
		assert.True(t, allowed)
	})
}

func TestQueryDashboard(t *testing.T) {
	sdb, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer sdb.Close()

	if err := initSQLite(sdb); err != nil {
		t.Fatalf("failed to init sqlite: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := &App{
		Sqlite:    sdb,
		DuckDBDSN: ":memory:",
		Logger:    logger,
	}

	ctx := context.Background()

	t.Run("Basic query", func(t *testing.T) {
		dq := DashboardQuery{
			Content: "SELECT 1 AS val",
			ID:      "test-dash",
		}
		result, err := QueryDashboard(app, ctx, dq, url.Values{}, nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(result.Sections))
		assert.Equal(t, 1, len(result.Sections[0].Queries))
		assert.Equal(t, 1, len(result.Sections[0].Queries[0].Rows))
		// DuckDB returns int32 for small numbers by default in go-duckdb
		assert.Equal(t, int32(1), result.Sections[0].Queries[0].Rows[0][0])
	})

	t.Run("Query with variables", func(t *testing.T) {
		dq := DashboardQuery{
			Content: "SELECT getvariable('myvar') AS val",
			ID:      "test-dash-vars",
		}
		result, err := QueryDashboard(app, ctx, dq, url.Values{}, map[string]any{"myvar": "hello"})
		assert.NoError(t, err)
		assert.Equal(t, "hello", result.Sections[0].Queries[0].Rows[0][0])
	})

	t.Run("Variable precedence - query param must NOT overwrite variables", func(t *testing.T) {
		// This test fails if query params can overwrite secure variables.
		// We have two queries:
		// 1. A dropdown that defines 'myvar'
		// 2. A query that uses 'myvar'
		dq := DashboardQuery{
			Content: `
				SELECT 'secure_val'::DROPDOWN AS myvar, 'Secure'::LABEL AS label UNION ALL SELECT 'malicious_val', 'Malicious';
				SELECT getvariable('myvar') AS val;
			`,
			ID: "test-precedence",
		}

		// Secure variable set to 'secure_val'
		variables := map[string]any{"myvar": "secure_val"}
		// Query param tries to set 'myvar' to 'malicious_val'
		queryParams := url.Values{"myvar": []string{"malicious_val"}}

		result, err := QueryDashboard(app, ctx, dq, queryParams, variables)
		assert.NoError(t, err)

		// The second query (result.Sections[1]) should still see 'secure_val'
		assert.Equal(t, 2, len(result.Sections))
		assert.Equal(t, "secure_val", result.Sections[1].Queries[0].Rows[0][0], "Secure variable was overwritten by query parameter!")
	})

	t.Run("Variable precedence - normal query param should still work", func(t *testing.T) {
		dq := DashboardQuery{
			Content: `
				SELECT 'val1'::DROPDOWN AS myvar, 'Val 1'::LABEL AS label UNION ALL SELECT 'val2', 'Val 2';
				SELECT getvariable('myvar') AS val;
			`,
			ID: "test-normal",
		}

		// No secure variable
		variables := map[string]any{}
		// Query param sets 'myvar' to 'val2'
		queryParams := url.Values{"myvar": []string{"val2"}}

		result, err := QueryDashboard(app, ctx, dq, queryParams, variables)
		assert.NoError(t, err)

		// The second query should see 'val2'
		assert.Equal(t, 2, len(result.Sections))
		assert.Equal(t, "val2", result.Sections[1].Queries[0].Rows[0][0])
	})
}
