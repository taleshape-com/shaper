// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"io"
	"log/slog"
	"net/url"
	"strings"
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

	t.Run("Rejects invalid variable names in token variables", func(t *testing.T) {
		dq := DashboardQuery{
			Content: "SELECT 1 AS val",
			ID:      "test-invalid-var-name",
		}
		// Semicolon/SQL injection attempt in var name
		invalidVars := map[string]any{"bad;name": "val"}
		_, err := QueryDashboard(app, ctx, dq, url.Values{}, invalidVars)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid variable name")

		// Name starting with number
		invalidVars2 := map[string]any{"123var": "val"}
		_, err = QueryDashboard(app, ctx, dq, url.Values{}, invalidVars2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid variable name")
	})

	t.Run("Rejects token variable string exceeding maximum length", func(t *testing.T) {
		dq := DashboardQuery{
			Content: "SELECT 1 AS val",
			ID:      "test-oversized-var",
		}
		oversizedVars := map[string]any{"my_var": strings.Repeat("a", 4097)}
		_, err := QueryDashboard(app, ctx, dq, url.Values{}, oversizedVars)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum allowed length")
	})

	t.Run("Rejects token variable array exceeding maximum count", func(t *testing.T) {
		dq := DashboardQuery{
			Content: "SELECT 1 AS val",
			ID:      "test-oversized-array",
		}
		hugeArray := make([]any, 501)
		for i := range hugeArray {
			hugeArray[i] = "opt"
		}
		oversizedVars := map[string]any{"my_multi_var": hugeArray}
		_, err := QueryDashboard(app, ctx, dq, url.Values{}, oversizedVars)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum allowed count")
	})

	t.Run("Rejects input variable exceeding maximum length in query param", func(t *testing.T) {
		dq := DashboardQuery{
			Content: `
				SELECT 'Enter name'::INPUT AS search_term;
				SELECT getvariable('search_term') AS res;
			`,
			ID: "test-oversized-input-param",
		}
		oversizedParam := url.Values{"search_term": []string{strings.Repeat("a", 4097)}}
		_, err := QueryDashboard(app, ctx, dq, oversizedParam, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum allowed length")
	})

	t.Run("Rejects invalid variable name in dashboard query column", func(t *testing.T) {
		dq := DashboardQuery{
			Content: `
				SELECT 'Enter name'::INPUT AS "invalid-name";
			`,
			ID: "test-invalid-col-name",
		}
		_, err := QueryDashboard(app, ctx, dq, url.Values{}, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid variable name")
	})

	t.Run("Accepts valid input variable within limit", func(t *testing.T) {
		dq := DashboardQuery{
			Content: `
				SELECT 'Enter name'::INPUT AS search_term;
				SELECT getvariable('search_term') AS res;
			`,
			ID: "test-valid-input-param",
		}
		validParam := url.Values{"search_term": []string{"hello world"}}
		result, err := QueryDashboard(app, ctx, dq, validParam, nil)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(result.Sections))
		assert.Equal(t, "hello world", result.Sections[1].Queries[0].Rows[0][0])
	})

	t.Run("Defaults unset input variable to empty string and does not flag as unset", func(t *testing.T) {
		dq := DashboardQuery{
			Content: `
				SELECT 'Enter name'::INPUT AS search_term;
				SELECT getvariable('search_term') AS res;
			`,
			ID: "test-unset-input-default",
		}
		result, err := QueryDashboard(app, ctx, dq, url.Values{}, nil)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(result.Sections))
		assert.Equal(t, "", result.Sections[1].Queries[0].Rows[0][0])
		assert.Empty(t, result.UnsetVariables)
	})

	t.Run("Section and Label with Subtitle", func(t *testing.T) {
		dq := DashboardQuery{
			Content: `
				SELECT 'My Section'::SECTION, 'This is my section'::SUBTITLE;
				SELECT 'My Label'::LABEL, 'Explanation for chart'::SUBTITLE;
				SELECT 1::XAXIS, 10::LINECHART;
			`,
			ID: "test-section-label-subtitle",
		}
		result, err := QueryDashboard(app, ctx, dq, url.Values{}, nil)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(result.Sections))

		// Check Section Title and Subtitle
		assert.Equal(t, "header", result.Sections[0].Type)
		assert.NotNil(t, result.Sections[0].Title)
		assert.Equal(t, "My Section", *result.Sections[0].Title)
		assert.NotNil(t, result.Sections[0].Subtitle)
		assert.Equal(t, "This is my section", *result.Sections[0].Subtitle)

		// Check Render Label and Subtitle on content query
		assert.Equal(t, "content", result.Sections[1].Type)
		assert.Equal(t, 1, len(result.Sections[1].Queries))
		q := result.Sections[1].Queries[0]
		assert.Equal(t, "linechart", q.Render.Type)
		assert.NotNil(t, q.Render.Label)
		assert.Equal(t, "My Label", *q.Render.Label)
		assert.NotNil(t, q.Render.Subtitle)
		assert.Equal(t, "Explanation for chart", *q.Render.Subtitle)
	})

	t.Run("Section and Label with reversed column order for Subtitle", func(t *testing.T) {
		dq := DashboardQuery{
			Content: `
				SELECT 'This is my section'::SUBTITLE, 'My Section'::SECTION;
				SELECT 'Explanation for chart'::SUBTITLE, 'My Label'::LABEL;
				SELECT 42;
			`,
			ID: "test-subtitle-order",
		}
		result, err := QueryDashboard(app, ctx, dq, url.Values{}, nil)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(result.Sections))

		// Check Section Title and Subtitle
		assert.NotNil(t, result.Sections[0].Title)
		assert.Equal(t, "My Section", *result.Sections[0].Title)
		assert.NotNil(t, result.Sections[0].Subtitle)
		assert.Equal(t, "This is my section", *result.Sections[0].Subtitle)

		// Check Render Label and Subtitle on value query
		q := result.Sections[1].Queries[0]
		assert.Equal(t, "value", q.Render.Type)
		assert.NotNil(t, q.Render.Label)
		assert.Equal(t, "My Label", *q.Render.Label)
		assert.NotNil(t, q.Render.Subtitle)
		assert.Equal(t, "Explanation for chart", *q.Render.Subtitle)
	})

	t.Run("Footer link without custom text", func(t *testing.T) {
		dq := DashboardQuery{
			Content: `SELECT 'https://example.com'::FOOTER_LINK`,
			ID:      "test-footer-link-default",
		}
		result, err := QueryDashboard(app, ctx, dq, url.Values{}, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result.FooterLink)
		assert.Equal(t, "https://example.com", *result.FooterLink)
		assert.Nil(t, result.FooterLinkText)
	})

	t.Run("Footer link with custom text using AS alias", func(t *testing.T) {
		dq := DashboardQuery{
			Content: `SELECT 'https://example.com'::FOOTER_LINK AS "my custom text"`,
			ID:      "test-footer-link-custom-text",
		}
		result, err := QueryDashboard(app, ctx, dq, url.Values{}, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result.FooterLink)
		assert.Equal(t, "https://example.com", *result.FooterLink)
		assert.NotNil(t, result.FooterLinkText)
		assert.Equal(t, "my custom text", *result.FooterLinkText)
	})

	t.Run("Charts with COLOR and no CATEGORY using VALUES", func(t *testing.T) {
		dq := DashboardQuery{
			Content: `
SELECT 'Fruit Sales'::LABEL;
SELECT
  fruit::XAXIS,
  sales::BARCHART,
  color::COLOR
FROM (VALUES
  ('Apples', 45, '#ef4444'),
  ('Bananas', 80, '#eab308'),
  ('Blueberries', 30, '#3b82f6'),
  ('Kiwis', 55, '#22c55e')
) AS t(fruit, sales, color);

SELECT 'Project Velocity'::LABEL;
SELECT
  x::XAXIS,
  y::SCATTERPLOT,
  color::COLOR
FROM (VALUES
  (10, 25, '#ef4444'),
  (20, 45, '#eab308'),
  (30, 15, '#3b82f6'),
  (40, 60, '#22c55e')
) AS t(x, y, color);

SELECT 'Server Latency'::LABEL;
SELECT
  time_val::XAXIS,
  latency::LINECHART,
  color::COLOR
FROM (VALUES
  ('2026-01-01'::DATE, 10, '#ef4444'),
  ('2026-01-02'::DATE, 25, '#eab308'),
  ('2026-01-03'::DATE, 18, '#3b82f6'),
  ('2026-01-04'::DATE, 30, '#22c55e')
) AS t(time_val, latency, color);
`,
			ID: "test-charts-color",
		}
		result, err := QueryDashboard(app, ctx, dq, url.Values{}, nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(result.Sections))
		assert.Equal(t, 3, len(result.Sections[0].Queries))

		// Bar chart
		qBar := result.Sections[0].Queries[0]
		assert.Equal(t, "barchartHorizontal", qBar.Render.Type)
		assert.Equal(t, "index", qBar.Columns[0].Tag)
		assert.Equal(t, "value", qBar.Columns[1].Tag)
		assert.Equal(t, "color", qBar.Columns[2].Tag)
		assert.Equal(t, 4, len(qBar.Rows))

		// Scatter plot
		qScatter := result.Sections[0].Queries[1]
		assert.Equal(t, "scatterplot", qScatter.Render.Type)
		assert.Equal(t, "index", qScatter.Columns[0].Tag)
		assert.Equal(t, "value", qScatter.Columns[1].Tag)
		assert.Equal(t, "color", qScatter.Columns[2].Tag)
		assert.Equal(t, 4, len(qScatter.Rows))

		// Line chart
		qLine := result.Sections[0].Queries[2]
		assert.Equal(t, "linechart", qLine.Render.Type)
		assert.Equal(t, "index", qLine.Columns[0].Tag)
		assert.Equal(t, "value", qLine.Columns[1].Tag)
		assert.Equal(t, "color", qLine.Columns[2].Tag)
		assert.Equal(t, 4, len(qLine.Rows))
	})
}

