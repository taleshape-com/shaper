// SPDX-License-Identifier: MPL-2.0

package core

import (
	"bytes"
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/duckdb/duckdb-go/v2"
)

func TestStreamSQLToCSV(t *testing.T) {
	db, err := sqlx.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	app := &App{DuckDB: db}
	ctx := context.Background()
	buf := &bytes.Buffer{}

	sql := "SELECT 1 as id, 'hello' as name UNION ALL SELECT 2, 'world'"
	err = StreamSQLToCSV(app, ctx, sql, buf)
	if err != nil {
		t.Fatalf("StreamSQLToCSV failed: %v", err)
	}

	// CSV output uses \r\n on some systems or just \n. encoding/csv uses \n by default.
	expected := "id,name\n1,hello\n2,world\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}
