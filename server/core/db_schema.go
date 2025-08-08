// SPDX-License-Identifier: MPL-2.0

package core

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

func initDB(db *sqlx.DB, schema string) error {
	// Create schema if not exists
	if _, err := db.Exec("CREATE SCHEMA IF NOT EXISTS " + schema); err != nil {
		return fmt.Errorf("error creating schema: %w", err)
	}

	// Create apps table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + schema + `.apps (
			id VARCHAR PRIMARY KEY,
			path VARCHAR NOT NULL,
			name VARCHAR NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_by VARCHAR,
			updated_by VARCHAR,
			visibility VARCHAR,
			type VARCHAR NOT NULL,
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating apps table: %w", err)
	}

	// TODO: Remove once ran for all active users
	_, err = db.Exec(`
		ALTER TABLE ` + schema + `.apps ADD COLUMN IF NOT EXISTS visibility VARCHAR
	`)
	if err != nil {
		return fmt.Errorf("error adding visibility column to apps table: %w", err)
	}

	// TODO: Remove once ran for all active users
	_, err = db.Exec(`
		ALTER TABLE ` + schema + `.apps ADD COLUMN IF NOT EXISTS type VARCHAR;
		UPDATE ` + schema + `.apps SET type = 'dashboard' WHERE type IS NULL;
		ALTER TABLE ` + schema + `.apps ALTER COLUMN type SET NOT NULL;
	`)
	if err != nil {
		return fmt.Errorf("error adding type column to apps table: %w", err)
	}

	// Create api_keys table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + schema + `.api_keys (
			id VARCHAR PRIMARY KEY,
			hash VARCHAR NOT NULL,
			salt VARCHAR NOT NULL,
			name VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_by VARCHAR,
			updated_by VARCHAR
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating config table: %w", err)
	}

	// Create users table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + schema + `.users (
			id VARCHAR PRIMARY KEY,
			email VARCHAR NOT NULL,
			password_hash VARCHAR,
			name VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_by VARCHAR,
			updated_by VARCHAR,
			deleted_at TIMESTAMP,
			deleted_by VARCHAR
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating users table: %w", err)
	}
	// Create sessions table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + schema + `.sessions (
			id VARCHAR PRIMARY KEY,
			user_id VARCHAR NOT NULL REFERENCES ` + schema + `.users(id),
			hash VARCHAR NOT NULL,
			salt VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating sessions table: %w", err)
	}

	// Create invites table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + schema + `.invites (
			code VARCHAR PRIMARY KEY,
			email VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL,
			created_by VARCHAR
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating invites table: %w", err)
	}

	// Create custom types
	for _, t := range dbTypes {
		if err := createType(db, t.Name, t.Definition); err != nil {
			return err
		}
	}
	return nil
}
