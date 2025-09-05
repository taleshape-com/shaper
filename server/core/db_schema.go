// SPDX-License-Identifier: MPL-2.0

package core

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

func initSQLite(sdb *sqlx.DB) error {
	// Settings
	_, err := sdb.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA auto_vacuum = INCREMENTAL;
		PRAGMA foreign_keys = on;
		PRAGMA busy_timeout = 5000;
	`)
	if err != nil {
		return fmt.Errorf("error setting pragmas: %w", err)
	}

	// Create apps table
	_, err = sdb.Exec(`
		CREATE TABLE IF NOT EXISTS apps (
			id TEXT PRIMARY KEY,
			path TEXT NOT NULL,
			name TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			created_by TEXT,
			updated_by TEXT,
			visibility TEXT,
			type TEXT NOT NULL,
		  password_hash TEXT
		) STRICT
	`)
	if err != nil {
		return fmt.Errorf("error creating apps table: %w", err)
	}

	// Create api_keys table
	_, err = sdb.Exec(`
		CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			hash TEXT NOT NULL,
			salt TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			created_by TEXT,
			updated_by TEXT
		) STRICT
	`)
	if err != nil {
		return fmt.Errorf("error creating config table: %w", err)
	}

	// Create users table
	_, err = sdb.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL,
			password_hash TEXT,
			name TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			created_by TEXT,
			updated_by TEXT,
			deleted_at INTEGER,
			deleted_by TEXT
		) STRICT
	`)
	if err != nil {
		return fmt.Errorf("error creating users table: %w", err)
	}
	// Create sessions table
	_, err = sdb.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			hash TEXT NOT NULL,
			salt TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			FOREIGN KEY(user_id) REFERENCES users(id)
		) STRICT
	`)
	if err != nil {
		return fmt.Errorf("error creating sessions table: %w", err)
	}

	// Create invites table
	_, err = sdb.Exec(`
		CREATE TABLE IF NOT EXISTS invites (
			code TEXT PRIMARY KEY,
			email TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			created_by TEXT
		) STRICT
	`)
	if err != nil {
		return fmt.Errorf("error creating invites table: %w", err)
	}

	// Create task_runs table
	_, err = sdb.Exec(`
		CREATE TABLE IF NOT EXISTS task_runs (
			task_id TEXT PRIMARY KEY NOT NULL,
			last_run_at INTEGER,
			last_run_success INTEGER,
			last_run_duration INTEGER,
			next_run_at INTEGER,
			next_run_type TEXT NOT NULL DEFAULT 'single'
		) STRICT
	`)
	if err != nil {
		return fmt.Errorf("error creating task_runs table: %w", err)
	}

	return nil
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
