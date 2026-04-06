// SPDX-License-Identifier: MPL-2.0

package core

import (
	"bytes"
	"context"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/jmoiron/sqlx"
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
			got, err := resolveDownloadQueryID(tt.sqls, tt.downloadType)
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
