// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"net/url"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryTruncation(t *testing.T) {
	db, err := sqlx.Open("duckdb", "")
	require.NoError(t, err, "failed to open duckdb")
	defer db.Close()

	app := &App{DuckDB: db}
	ctx := context.Background()

	t.Run("query with less than max rows should not be truncated", func(t *testing.T) {
		sql := "SELECT * FROM generate_series(1, 100) AS t(id)"
		dashboardQuery := DashboardQuery{
			Content: sql,
			ID:      "test-dashboard",
		}

		result, err := QueryDashboard(app, ctx, dashboardQuery, url.Values{}, map[string]any{})
		require.NoError(t, err, "QueryDashboard failed")

		require.Len(t, result.Sections, 1, "expected 1 section")
		require.Len(t, result.Sections[0].Queries, 1, "expected 1 query")

		query := result.Sections[0].Queries[0]
		assert.False(t, query.IsTruncated, "query should not be truncated")
		assert.Len(t, query.Rows, 100, "expected 100 rows")
	})

	t.Run("query with exactly max rows should not be truncated", func(t *testing.T) {
		sql := "SELECT * FROM generate_series(1, 3000) AS t(id)"
		dashboardQuery := DashboardQuery{
			Content: sql,
			ID:      "test-dashboard",
		}

		result, err := QueryDashboard(app, ctx, dashboardQuery, url.Values{}, map[string]any{})
		require.NoError(t, err, "QueryDashboard failed")

		require.Len(t, result.Sections, 1, "expected 1 section")
		require.Len(t, result.Sections[0].Queries, 1, "expected 1 query")

		query := result.Sections[0].Queries[0]
		assert.False(t, query.IsTruncated, "query should not be truncated")
		assert.Len(t, query.Rows, 3000, "expected 3000 rows")
	})

	t.Run("query with more than max rows should be truncated", func(t *testing.T) {
		sql := "SELECT * FROM generate_series(1, 5000) AS t(id)"
		dashboardQuery := DashboardQuery{
			Content: sql,
			ID:      "test-dashboard",
		}

		result, err := QueryDashboard(app, ctx, dashboardQuery, url.Values{}, map[string]any{})
		require.NoError(t, err, "QueryDashboard failed")

		require.Len(t, result.Sections, 1, "expected 1 section")
		require.Len(t, result.Sections[0].Queries, 1, "expected 1 query")

		query := result.Sections[0].Queries[0]
		assert.True(t, query.IsTruncated, "query should be truncated")
		assert.Len(t, query.Rows, 3000, "expected 3000 rows after truncation")

		firstRow := query.Rows[0][0].(int64)
		lastRow := query.Rows[2999][0].(int64)
		assert.Equal(t, int64(1), firstRow, "first row should be 1")
		assert.Equal(t, int64(3000), lastRow, "last row should be 3000")
	})

	t.Run("multiple queries, some truncated", func(t *testing.T) {
		sql := `
SELECT * FROM generate_series(1, 100) AS t(id);
SELECT * FROM generate_series(1, 5000) AS t(id);
SELECT * FROM generate_series(1, 2000) AS t(id);
`
		dashboardQuery := DashboardQuery{
			Content: sql,
			ID:      "test-dashboard",
		}

		result, err := QueryDashboard(app, ctx, dashboardQuery, url.Values{}, map[string]any{})
		require.NoError(t, err, "QueryDashboard failed")

		require.Len(t, result.Sections, 1, "expected 1 section")
		require.Len(t, result.Sections[0].Queries, 3, "expected 3 queries")

		assert.False(t, result.Sections[0].Queries[0].IsTruncated, "first query should not be truncated")
		assert.Len(t, result.Sections[0].Queries[0].Rows, 100, "first query should have 100 rows")

		assert.True(t, result.Sections[0].Queries[1].IsTruncated, "second query should be truncated")
		assert.Len(t, result.Sections[0].Queries[1].Rows, 3000, "second query should have 3000 rows")

		assert.False(t, result.Sections[0].Queries[2].IsTruncated, "third query should not be truncated")
		assert.Len(t, result.Sections[0].Queries[2].Rows, 2000, "third query should have 2000 rows")
	})
}
