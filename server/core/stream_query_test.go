// SPDX-License-Identifier: MPL-2.0

package core

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/jmoiron/sqlx"
	"github.com/xuri/excelize/v2"
	_ "modernc.org/sqlite"
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

func TestStreamSQLToJSON(t *testing.T) {
	db, err := sqlx.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	app := &App{DuckDB: db}
	ctx := context.Background()
	buf := &bytes.Buffer{}

	sql := "SELECT 1 as id, 'hello' as name UNION ALL SELECT 2, 'world'"
	err = StreamSQLToJSON(app, ctx, sql, buf)
	if err != nil {
		t.Fatalf("StreamSQLToJSON failed: %v", err)
	}

	// json.Encoder adds newlines
	// Note: maps in Go are randomized, but since we only have 2 columns, it should be stable enough or we should check for contents
	// For this simple test with few columns it usually works, but it's better to be careful.
	// Actually for 1.21+ Go maps have some order but not guaranteed.
	// Let's just check if it contains expected strings.
	got := buf.String()
	if !strings.Contains(got, "{\"id\":1,\"name\":\"hello\"}") && !strings.Contains(got, "{\"name\":\"hello\",\"id\":1}") {
		t.Errorf("got %q, missing expected object", got)
	}
	if !strings.Contains(got, "{\"id\":2,\"name\":\"world\"}") && !strings.Contains(got, "{\"name\":\"world\",\"id\":2}") {
		t.Errorf("got %q, missing expected object", got)
	}
}

func TestResolveDownloadQueryID(t *testing.T) {
	tests := []struct {
		name         string
		sqls         []string
		downloadType string
		want         int
		wantErr      bool
	}{
		{
			name: "single matching download type",
			sqls: []string{
				"SELECT 'Shaper Demo Dashboard'::SECTION;",
				"SELECT ('sessions-' || today())::DOWNLOAD_CSV AS CSV;",
				"SELECT * FROM dataset;",
			},
			downloadType: "csv",
			want:         2,
		},
		{
			name: "single matching json download type",
			sqls: []string{
				"SELECT 'Shaper Demo Dashboard'::SECTION;",
				"SELECT ('sessions-' || today())::DOWNLOAD_JSON AS JSON;",
				"SELECT * FROM dataset;",
			},
			downloadType: "json",
			want:         2,
		},
		{
			name: "single data query (no special types)",
			sqls: []string{
				"SELECT 'Label'::LABEL;",
				"SELECT 'Hello World';",
			},
			downloadType: "csv",
			want:         1,
		},
		{
			name: "multiple matching download types (fail)",
			sqls: []string{
				"SELECT ('sessions-' || today())::DOWNLOAD_CSV AS CSV;",
				"SELECT * FROM dataset;",
				"SELECT ('sessions-' || today())::DOWNLOAD_CSV AS CSV;",
				"SELECT * FROM dataset;",
			},
			downloadType: "csv",
			wantErr:      true,
		},
		{
			name: "multiple data queries (fail)",
			sqls: []string{
				"SELECT 1 as id",
				"SELECT 2 as id",
			},
			downloadType: "csv",
			wantErr:      true,
		},
		{
			name: "labels plural is fine",
			sqls: []string{
				"SELECT 'GAUGE with RANGE, LABELS and COLORS'::LABEL;",
				"SELECT 75::GAUGE, [0, 33, 66, 100]::RANGE, ['Bad', 'Okay', 'Good']::LABELS;",
			},
			downloadType: "csv",
			want:         1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveDownloadQueryID(nil, tt.sqls, tt.downloadType)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveDownloadQueryID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("resolveDownloadQueryID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetVarPrefixOnlyQueriesBeforeDownload(t *testing.T) {
	db, err := sqlx.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	if _, err := initDuckDB(db); err != nil {
		t.Fatalf("failed to init duckdb: %v", err)
	}

	app := &App{DuckDB: db}
	ctx := context.Background()
	conn, err := db.Connx(ctx)
	if err != nil {
		t.Fatalf("failed to get conn: %v", err)
	}
	defer conn.Close()

	sqls := []string{
		"SELECT 'bar'::DROPDOWN AS my_var;",
		"SELECT 1 AS result;",
		"SELECT INVALID SYNTAX HERE;",
	}

	queryID := 1 // download query is at index 1

	// When passing sqls[:queryID], the invalid query at index 2 should not be evaluated
	varPrefix, _, err := getVarPrefix(app, conn, ctx, sqls[:queryID], nil, nil)
	if err != nil {
		t.Fatalf("getVarPrefix failed: %v", err)
	}

	if !strings.Contains(varPrefix, "my_var") {
		t.Errorf("expected varPrefix to contain my_var, got: %q", varPrefix)
	}

	// Test SET VARIABLE defaults in getVarPrefix
	sqlsWithDefaults := []string{
		"SET VARIABLE cat = [];",
		"SELECT 'a'::DROPDOWN_MULTI AS cat UNION ALL SELECT 'b';",
		"SET VARIABLE drop_var = 'second';",
		"SELECT 'first'::DROPDOWN AS drop_var UNION ALL SELECT 'second';",
		"SET VARIABLE search_var = 'hello';",
		"SELECT 'Hint'::INPUT AS search_var;",
	}
	varPrefixDefaults, _, err := getVarPrefix(app, conn, ctx, sqlsWithDefaults, url.Values{}, nil)
	if err != nil {
		t.Fatalf("getVarPrefix with defaults failed: %v", err)
	}
	if !strings.Contains(varPrefixDefaults, `SET VARIABLE "cat" = []::VARCHAR[];`) {
		t.Errorf("expected varPrefix to have empty cat array, got: %q", varPrefixDefaults)
	}
	if !strings.Contains(varPrefixDefaults, `SET VARIABLE "drop_var" = 'second';`) {
		t.Errorf("expected varPrefix to have drop_var='second', got: %q", varPrefixDefaults)
	}
	if !strings.Contains(varPrefixDefaults, `SET VARIABLE "search_var" = 'hello';`) {
		t.Errorf("expected varPrefix to have search_var='hello', got: %q", varPrefixDefaults)
	}
}

func TestStreamQueryXLSXNumericTypes(t *testing.T) {
	sdb, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer sdb.Close()

	if err := InitSQLite(sdb); err != nil {
		t.Fatalf("failed to init sqlite: %v", err)
	}

	db, err := sqlx.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	if _, err := initDuckDB(db); err != nil {
		t.Fatalf("failed to init duckdb: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := &App{
		Sqlite: sdb,
		DuckDB: db,
		Logger: logger,
	}

	content := `SELECT 'T'::LABEL;
SELECT category, count(*) AS n, sum(qty) AS total_qty, sum(price) AS revenue
FROM (VALUES ('a', 2, 10.50::DECIMAL(10,2)), ('a', 3, 4.25::DECIMAL(10,2)), ('b', 1, 7.00::DECIMAL(10,2))) t(category, qty, price)
GROUP BY ALL ORDER BY ALL;`

	_, err = sdb.Exec(`INSERT INTO apps (id, type, name, content, created_at, updated_at, visibility)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'), ?)`,
		"xlsx-dash", "dashboard", "XLSX Test", content, "public")
	if err != nil {
		t.Fatalf("failed to insert dashboard: %v", err)
	}

	buf := &bytes.Buffer{}
	err = StreamQueryXLSX(app, context.Background(), "xlsx-dash", url.Values{}, -1, nil, buf)
	if err != nil {
		t.Fatalf("StreamQueryXLSX failed: %v", err)
	}

	xlsx, err := excelize.OpenReader(buf)
	if err != nil {
		t.Fatalf("failed to open generated xlsx: %v", err)
	}
	defer xlsx.Close()

	// Check cell types and values for row 2 ('a', 2, 5, 14.75) and row 3 ('b', 1, 1, 7)
	type expectedCell struct {
		cell     string
		wantVal  string
		wantType excelize.CellType
	}
	expected := []expectedCell{
		{cell: "A2", wantVal: "a", wantType: excelize.CellTypeSharedString},
		{cell: "B2", wantVal: "2", wantType: excelize.CellTypeUnset}, // numbers in Excel have unset or number cell type
		{cell: "C2", wantVal: "5", wantType: excelize.CellTypeUnset},
		{cell: "D2", wantVal: "14.75", wantType: excelize.CellTypeUnset},
		{cell: "A3", wantVal: "b", wantType: excelize.CellTypeSharedString},
		{cell: "B3", wantVal: "1", wantType: excelize.CellTypeUnset},
		{cell: "C3", wantVal: "1", wantType: excelize.CellTypeUnset},
		{cell: "D3", wantVal: "7", wantType: excelize.CellTypeUnset},
	}

	for _, exp := range expected {
		ct, err := xlsx.GetCellType("Sheet1", exp.cell)
		if err != nil {
			t.Fatalf("GetCellType(%s) failed: %v", exp.cell, err)
		}
		if ct != exp.wantType {
			t.Errorf("cell %s type = %v, want %v", exp.cell, ct, exp.wantType)
		}
		val, err := xlsx.GetCellValue("Sheet1", exp.cell)
		if err != nil {
			t.Fatalf("GetCellValue(%s) failed: %v", exp.cell, err)
		}
		if val != exp.wantVal {
			t.Errorf("cell %s value = %q, want %q", exp.cell, val, exp.wantVal)
		}
	}
}
