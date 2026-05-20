// SPDX-License-Identifier: MPL-2.0

package core

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/duckdb/duckdb-go/v2"
	"github.com/jmoiron/sqlx"
)

func migrateSystemData(sqliteDbx *sqlx.DB, duckDbx *sqlx.DB, deprecatedSchema string, logger *slog.Logger) error {
	for _, table := range []string{
		"apps",
		"api_keys",
		"users",
		"sessions",
		"invites",
		"task_runs",
	} {
		if err := migrateTableData(sqliteDbx, duckDbx, table, deprecatedSchema, logger); err != nil {
			return fmt.Errorf("failed to migrate table %s: %w", table, err)
		}
	}
	return nil
}

// The database contains little data so we can read all data into memory and write it to sqlite database.
// We don't delete data from duckdb after migrating.
// We only migrate data if the target table is empty.
func migrateTableData(sqliteDbx *sqlx.DB, duckDbx *sqlx.DB, table string, deprecatedSchema string, logger *slog.Logger) error {
	duckDbTable := "\"" + deprecatedSchema + "\"." + table
	var duckCount int
	err := duckDbx.Get(&duckCount, fmt.Sprintf(`SELECT count(*) FROM %s`, duckDbTable))
	// Skip if table not in DuckDB
	if err != nil {
		return nil
	}
	if duckCount == 0 {
		// DuckDB table has no data, skipping
		return nil
	}
	// Check if table already has data
	var liteCount int
	err = sqliteDbx.Get(&liteCount, fmt.Sprintf(`SELECT count(*) FROM %s`, table))
	if err != nil {
		return fmt.Errorf("failed to count rows in sqlite table %s: %w", table, err)
	}
	if liteCount > 0 {
		// Table already has data, skip migration
		return nil
	}
	logger.Info("Migrating table to SQLite", slog.String("table", table), slog.Int("rows", duckCount))
	rows, err := duckDbx.Queryx(fmt.Sprintf(`SELECT * FROM %s`, duckDbTable))
	if err != nil {
		return fmt.Errorf("failed to query duckdb table %s: %w", duckDbTable, err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns for duckdb table %s: %w", duckDbTable, err)
	}
	tx, err := sqliteDbx.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for sqlite table %s: %w", table, err)
	}
	// Migrate each row
	for rows.Next() {
		values := make([]any, len(cols))
		valuePtrs := make([]any, len(cols))
		for i := range cols {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to scan row for duckdb table %s: %w", duckDbTable, err)
		}
		placeholders := ""
		for i := range cols {
			if i > 0 {
				placeholders += ", "
			}
			placeholders += "?"
			if values[i] != nil && table == "task_runs" && cols[i] == "last_run_duration" {
				v, ok := values[i].(duckdb.Interval)
				if !ok {
					tx.Rollback()
					return fmt.Errorf("failed to convert last_run_duration to time for duckdb table %s", table)
				}
				values[i] = formatInterval(v)
			}
		}
		_, err := tx.Exec(
			fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, table, strings.Join(cols, ", "), placeholders),
			values...,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert row into sqlite table %s: %w", table, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for sqlite table %s: %w", table, err)
	}
	return nil
}
