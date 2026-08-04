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

	t.Run("Linechart with confidence band", func(t *testing.T) {
		dq := DashboardQuery{
			Content: `
				SELECT
					'2026-01-01'::TIMESTAMP::XAXIS AS ts,
					10.0::LINECHART AS val,
					8.0::BAND_LOWER AS confidence_lower,
					12.0::BAND_UPPER AS confidence_upper
			`,
			ID: "test-dash-band",
		}
		result, err := QueryDashboard(app, ctx, dq, url.Values{}, nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(result.Sections))
		assert.Equal(t, 1, len(result.Sections[0].Queries))
		q := result.Sections[0].Queries[0]
		assert.Equal(t, "linechart", q.Render.Type)
		
		// Verify tags are correct
		assert.Equal(t, "index", q.Columns[0].Tag)
		assert.Equal(t, "value", q.Columns[1].Tag)
		assert.Equal(t, "band_lower", q.Columns[2].Tag)
		assert.Equal(t, "band_upper", q.Columns[3].Tag)
	})

	t.Run("Scatterplot", func(t *testing.T) {
		dq := DashboardQuery{
			Content: `
				SELECT
					'2026-01-01'::TIMESTAMP::XAXIS AS ts,
					10.0::SCATTERPLOT AS val,
					'my-category'::SCATTERPLOT_CATEGORY AS cat,
					'#ff0000'::SCATTERPLOT_COLOR AS col
			`,
			ID: "test-dash-scatter",
		}
		result, err := QueryDashboard(app, ctx, dq, url.Values{}, nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(result.Sections))
		assert.Equal(t, 1, len(result.Sections[0].Queries))
		q := result.Sections[0].Queries[0]
		assert.Equal(t, "scatterplot", q.Render.Type)
		
		// Verify tags are correct
		assert.Equal(t, "index", q.Columns[0].Tag)
		assert.Equal(t, "value", q.Columns[1].Tag)
		assert.Equal(t, "category", q.Columns[2].Tag)
		assert.Equal(t, "color", q.Columns[3].Tag)
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

	t.Run("Detects unset variables accurately", func(t *testing.T) {
		dq := DashboardQuery{
			Content: `
				SELECT getvariable('already_set') AS v1, getvariable('missing_var1') AS v2;
				SET VARIABLE local_var = 'foo';
				SELECT getvariable('local_var') AS v3, getvariable('missing_var2') AS v4;
			`,
			ID: "test-unset-vars",
		}

		variables := map[string]any{"already_set": "hello"}
		result, err := QueryDashboard(app, ctx, dq, url.Values{}, variables)
		assert.NoError(t, err)
		assert.Equal(t, []string{"missing_var1", "missing_var2"}, result.UnsetVariables)
	})

	t.Run("TIMESTAMPTZ support in custom types and standalone columns", func(t *testing.T) {
		dq := DashboardQuery{
			Content: `
				SELECT '2026-08-04 10:00:00+00'::TIMESTAMPTZ::XLINE;
				SELECT
					'2026-08-04 12:00:00+00'::TIMESTAMPTZ::XAXIS AS ts_xaxis,
					10.0::LINECHART AS val;
				SELECT '2026-08-04 10:00:00+00'::TIMESTAMPTZ::YLINE;
				SELECT
					5.0::BARCHART AS bval,
					'2026-08-04 12:00:00+00'::TIMESTAMPTZ::YAXIS AS ts_yaxis;
				SELECT
					'2026-08-04 12:00:00+00'::TIMESTAMPTZ AS standalone_tz;
				SELECT
					'2026-08-04 12:00:00+00'::TIMESTAMPTZ::DATEPICKER AS dp;
				SELECT
					'2026-08-04 12:00:00+00'::TIMESTAMPTZ::RELOAD;
			`,
			ID: "test-timestamptz",
		}

		result, err := QueryDashboard(app, ctx, dq, url.Values{}, nil)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Sections), 1)

		// First query: linechart with XAXIS (TIMESTAMPTZ) and XLINE (TIMESTAMPTZ)
		q1 := result.Sections[0].Queries[0]
		assert.Equal(t, "linechart", q1.Render.Type)
		assert.Equal(t, "timestamp", q1.Columns[0].Type)
		assert.Equal(t, 1, len(q1.Render.MarkLines))
		assert.False(t, q1.Render.MarkLines[0].IsYaxis)
		assert.Equal(t, int64(1785837600000), q1.Render.MarkLines[0].Value) // 2026-08-04 10:00:00 UTC = 1785837600000 ms

		// Second query: barchartVertical with YAXIS (TIMESTAMPTZ) and YLINE (TIMESTAMPTZ)
		q2 := result.Sections[0].Queries[1]
		assert.Equal(t, "barchartVertical", q2.Render.Type)
		assert.Equal(t, "timestamp", q2.Columns[1].Type)
		assert.Equal(t, 1, len(q2.Render.MarkLines))
		assert.True(t, q2.Render.MarkLines[0].IsYaxis)

		// Third query: standalone TIMESTAMPTZ
		q3 := result.Sections[0].Queries[2]
		assert.Equal(t, "timestamp", q3.Columns[0].Type)

		// Fourth query: DATEPICKER with TIMESTAMPTZ
		q4 := result.Sections[1].Queries[0] // Header section for datepicker
		assert.Equal(t, "datepicker", q4.Render.Type)

		// Reload check
		assert.Equal(t, int64(1785844800000), result.ReloadAt) // 2026-08-04 12:00:00 UTC = 1785844800000 ms
	})
}
