package core

import (
	"context"
	"fmt"
	"shaper/server/api"
)

func (app *App) GetSchema(ctx context.Context) (*api.SchemaResponse, error) {
	db, cleanup, err := app.GetDuckDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DuckDB connection: %w", err)
	}
	defer cleanup()

	// 1. Fetch Databases
	var databases []struct {
		Name string `db:"database_name"`
	}
	err = db.SelectContext(ctx, &databases, `
		SELECT database_name
		FROM duckdb_databases()
		WHERE NOT internal AND database_name NOT IN ('sqlite', 'system', 'temp')
		ORDER BY database_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch databases: %w", err)
	}

	res := &api.SchemaResponse{
		Databases:  make([]api.Database, 0, len(databases)),
		Extensions: make([]api.Extension, 0),
		Secrets:    make([]api.Secret, 0),
	}

	// Fetch Extensions
	var extensions []struct {
		Name        string `db:"extension_name"`
		Loaded      bool   `db:"loaded"`
		Installed   bool   `db:"installed"`
		Description string `db:"description"`
	}
	_ = db.SelectContext(ctx, &extensions, `
		SELECT extension_name, loaded, installed, description
		FROM duckdb_extensions()
		WHERE loaded AND installed AND extension_name NOT IN ('autocomplete', 'core_functions', 'icu', 'jemalloc', 'json', 'parquet')
		ORDER BY extension_name
	`)
	for _, e := range extensions {
		res.Extensions = append(res.Extensions, api.Extension{
			Name:        e.Name,
			Loaded:      e.Loaded,
			Installed:   e.Installed,
			Description: e.Description,
		})
	}

	// Fetch Secrets
	rows, err := db.QueryxContext(ctx, `
		SELECT name, type, provider, scope
		FROM duckdb_secrets()
		ORDER BY name
	`)
	if err == nil {
		for rows.Next() {
			m := make(map[string]interface{})
			if err := rows.MapScan(m); err == nil {
				s := api.Secret{}
				if val, ok := m["name"]; ok && val != nil {
					s.Name = fmt.Sprint(val)
				}
				if val, ok := m["type"]; ok && val != nil {
					s.Type = fmt.Sprint(val)
				}
				if val, ok := m["provider"]; ok && val != nil {
					s.Provider = fmt.Sprint(val)
				}
				if val, ok := m["scope"]; ok && val != nil {
					if scopes, ok := val.([]interface{}); ok {
						for _, scope := range scopes {
							s.Scope = append(s.Scope, fmt.Sprint(scope))
						}
					}
				}
				res.Secrets = append(res.Secrets, s)
			}
		}
		rows.Close()
	}

	for _, d := range databases {
		database := api.Database{
			Name:    d.Name,
			Schemas: make([]api.Schema, 0),
		}

		// 2. Fetch Schemas for this database
		var schemas []struct {
			Name string `db:"schema_name"`
		}
		err = db.SelectContext(ctx, &schemas, `
			SELECT schema_name
			FROM duckdb_schemas()
			WHERE database_name = ? AND schema_name NOT IN ('information_schema', 'pg_catalog')
			ORDER BY schema_name
		`, d.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch schemas for %s: %w", d.Name, err)
		}

		for _, s := range schemas {
			schema := api.Schema{
				Name:   s.Name,
				Tables: make([]api.Table, 0),
				Views:  make([]api.View, 0),
				Enums:  make([]api.Enum, 0),
			}

			// 3. Fetch Enums for this schema
			// Note: duckdb_types() doesn't have database_name in some versions, but let's assume it does or use current
			var enums []struct {
				Name string `db:"type_name"`
			}
			db.SelectContext(ctx, &enums, `
				SELECT type_name
				FROM duckdb_types()
				WHERE schema_name = ? AND logical_type = 'ENUM' AND NOT internal
				ORDER BY type_name
			`, s.Name)
			// We ignore error here as duckdb_types might not have database_name or fail for other reasons,
			// it's not critical for the whole schema

			for _, e := range enums {
				var values []string
				// enum_range returns an array, which sqlx might struggle to scan directly into []string
				// if not careful, but let's try.
				query := fmt.Sprintf("SELECT enum_range(NULL::\"%s\".\"%s\")", s.Name, e.Name)
				// DuckDB array to Go slice might need special handling if sqlx doesn't do it.
				// For now let's just try to get it as a string if it fails.
				var val interface{}
				if err := db.GetContext(ctx, &val, query); err == nil {
					if slice, ok := val.([]interface{}); ok {
						for _, v := range slice {
							if s, ok := v.(string); ok {
								values = append(values, s)
							}
						}
					}
				}
				schema.Enums = append(schema.Enums, api.Enum{
					Name:   e.Name,
					Values: values,
				})
			}

			// 4. Fetch Tables
			var tables []struct {
				Name    string  `db:"table_name"`
				Comment *string `db:"comment"`
			}
			err = db.SelectContext(ctx, &tables, `
				SELECT table_name, comment
				FROM duckdb_tables()
				WHERE database_name = ? AND schema_name = ? AND NOT internal
				ORDER BY table_name
			`, d.Name, s.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch tables for %s.%s: %w", d.Name, s.Name, err)
			}

			for _, t := range tables {
				table := api.Table{
					Name:    t.Name,
					Columns: make([]api.Column, 0),
				}
				if t.Comment != nil {
					table.Comment = *t.Comment
				}

				// Fetch Columns
				var columns []struct {
					Name     string  `db:"column_name"`
					Type     string  `db:"data_type"`
					Nullable bool    `db:"is_nullable"`
					Default  *string `db:"column_default"`
					Comment  *string `db:"comment"`
				}
				err = db.SelectContext(ctx, &columns, `
					SELECT column_name, data_type, is_nullable, column_default, comment
					FROM duckdb_columns()
					WHERE database_name = ? AND schema_name = ? AND table_name = ?
					ORDER BY column_index
				`, d.Name, s.Name, t.Name)
				if err != nil {
					return nil, fmt.Errorf("failed to fetch columns for %s.%s.%s: %w", d.Name, s.Name, t.Name, err)
				}

				for _, c := range columns {
					col := api.Column{
						Name:     c.Name,
						Type:     c.Type,
						Nullable: c.Nullable,
						Default:  c.Default,
					}
					if c.Comment != nil {
						col.Comment = *c.Comment
					}
					table.Columns = append(table.Columns, col)
				}

				// Fetch Constraints
				rows, err := db.QueryxContext(ctx, `
					SELECT *
					FROM duckdb_constraints()
					WHERE database_name = ? AND schema_name = ? AND table_name = ?
				`, d.Name, s.Name, t.Name)
				if err != nil {
					// Fallback if database_name is not available (older duckdb?)
					rows, err = db.QueryxContext(ctx, `
						SELECT *
						FROM duckdb_constraints()
						WHERE schema_name = ? AND table_name = ?
					`, s.Name, t.Name)
				}

				if err == nil {
					for rows.Next() {
						m := make(map[string]interface{})
						if err := rows.MapScan(m); err == nil {
							c := api.Constraint{}
							if val, ok := m["constraint_name"]; ok && val != nil {
								c.Name = fmt.Sprint(val)
							}
							if val, ok := m["constraint_type"]; ok && val != nil {
								c.Type = fmt.Sprint(val)
							}
							// Try different possible column names for columns array
							colKey := ""
							if _, ok := m["constraint_column_names"]; ok {
								colKey = "constraint_column_names"
							} else if _, ok := m["column_names"]; ok {
								colKey = "column_names"
							}

							if colKey != "" {
								if cols, ok := m[colKey].([]interface{}); ok {
									for _, col := range cols {
										c.Columns = append(c.Columns, fmt.Sprint(col))
									}
								}
							}

							if val, ok := m["referenced_table"]; ok && val != nil {
								s := fmt.Sprint(val)
								c.ReferencedTable = &s
							}

							table.Constraints = append(table.Constraints, c)
						}
					}
					rows.Close()
				}

				schema.Tables = append(schema.Tables, table)
			}

			// 5. Fetch Views
			var views []struct {
				Name       string  `db:"view_name"`
				Comment    *string `db:"comment"`
				Definition string  `db:"sql"`
			}
			err = db.SelectContext(ctx, &views, `
				SELECT view_name, comment, sql
				FROM duckdb_views()
				WHERE database_name = ? AND schema_name = ? AND NOT internal
				ORDER BY view_name
			`, d.Name, s.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch views for %s.%s: %w", d.Name, s.Name, err)
			}

			for _, v := range views {
				view := api.View{
					Name:       v.Name,
					Definition: v.Definition,
					Columns:    make([]api.Column, 0),
				}
				if v.Comment != nil {
					view.Comment = *v.Comment
				}

				// Fetch Columns for View
				var columns []struct {
					Name     string  `db:"column_name"`
					Type     string  `db:"data_type"`
					Nullable bool    `db:"is_nullable"`
					Default  *string `db:"column_default"`
					Comment  *string `db:"comment"`
				}
				err = db.SelectContext(ctx, &columns, `
					SELECT column_name, data_type, is_nullable, column_default, comment
					FROM duckdb_columns()
					WHERE database_name = ? AND schema_name = ? AND table_name = ?
					ORDER BY column_index
				`, d.Name, s.Name, v.Name)
				if err == nil {
					for _, c := range columns {
						col := api.Column{
							Name:     c.Name,
							Type:     c.Type,
							Nullable: c.Nullable,
							Default:  c.Default,
						}
						if c.Comment != nil {
							col.Comment = *c.Comment
						}
						view.Columns = append(view.Columns, col)
					}
				}

				schema.Views = append(schema.Views, view)
			}

			database.Schemas = append(database.Schemas, schema)
		}

		res.Databases = append(res.Databases, database)
	}

	return res, nil
}
