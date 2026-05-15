// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"shaper/server/api"
	"strings"
)

func RunSchemaCommand(ctx context.Context, configPath, authFile string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	sysCfg, err := fetchSystemConfig(ctx, cfg.URL)
	if err != nil {
		return err
	}

	authFilePath, err := resolvePathRelativeToConfig(authFile, configPath)
	if err != nil {
		return err
	}

	auth := NewAuthManager(ctx, cfg.URL, authFilePath, sysCfg.LoginRequired)
	client, err := NewAPIClient(ctx, cfg.URL, auth)
	if err != nil {
		return err
	}

	resp, err := client.DoRequest(ctx, http.MethodGet, "/api/schema", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decodeAPIError(resp)
	}

	var schema api.SchemaResponse
	if err := json.NewDecoder(resp.Body).Decode(&schema); err != nil {
		return fmt.Errorf("failed to decode schema response: %w", err)
	}

	output := formatSchemaMarkdown(schema)
	fmt.Print(output)

	return nil
}

func formatSchemaMarkdown(schema api.SchemaResponse) string {
	var sb strings.Builder
	sb.WriteString("# Database Schema\n\n")

	if len(schema.Extensions) > 0 {
		sb.WriteString("## Loaded Extensions\n\n")
		headers := []string{"Extension", "Description"}
		rows := make([][]string, 0, len(schema.Extensions))
		for _, e := range schema.Extensions {
			rows = append(rows, []string{e.Name, e.Description})
		}
		sb.WriteString(formatPaddedTable(headers, rows))
		sb.WriteString("\n")
	}

	if len(schema.Secrets) > 0 {
		sb.WriteString("## Secrets\n\n")
		headers := []string{"Name", "Type", "Provider", "Scope"}
		rows := make([][]string, 0, len(schema.Secrets))
		for _, s := range schema.Secrets {
			rows = append(rows, []string{s.Name, s.Type, s.Provider, strings.Join(s.Scope, ", ")})
		}
		sb.WriteString(formatPaddedTable(headers, rows))
		sb.WriteString("\n")
	}

	for _, db := range schema.Databases {
		dbHasContent := false
		var dbSb strings.Builder
		dbSb.WriteString(fmt.Sprintf("## Database: %s\n\n", db.Name))

		for _, s := range db.Schemas {
			if len(s.Tables) == 0 && len(s.Views) == 0 && len(s.Enums) == 0 {
				continue
			}
			dbHasContent = true
			dbSb.WriteString(fmt.Sprintf("### Schema: %s\n\n", s.Name))

			if len(s.Enums) > 0 {
				dbSb.WriteString("#### Enums\n\n")
				dbSb.WriteString("```sql\n")
				for _, e := range s.Enums {
					values := make([]string, len(e.Values))
					for i, v := range e.Values {
						values[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
					}
					dbSb.WriteString(fmt.Sprintf("CREATE TYPE \"%s\".\"%s\" AS ENUM (%s);\n", s.Name, e.Name, strings.Join(values, ", ")))
				}
				dbSb.WriteString("```\n\n")
			}

			if len(s.Tables) > 0 {
				dbSb.WriteString("#### Tables\n\n")
				for _, t := range s.Tables {
					if t.Comment != "" {
						dbSb.WriteString(t.Comment + "\n\n")
					}

					// Filter out NOT NULL constraints as they are handled inline
					var activeConstraints []api.Constraint
					for _, c := range t.Constraints {
						if c.Type != "NOT NULL" {
							activeConstraints = append(activeConstraints, c)
						}
					}

					dbSb.WriteString("```sql\n")
					if t.Comment != "" {
						dbSb.WriteString(fmt.Sprintf("-- %s\n", t.Comment))
					}
					dbSb.WriteString(fmt.Sprintf("CREATE TABLE \"%s\".\"%s\".\"%s\" (\n", db.Name, s.Name, t.Name))
					for i, col := range t.Columns {
						if col.Comment != "" {
							dbSb.WriteString(fmt.Sprintf("    -- %s\n", col.Comment))
						}
						line := fmt.Sprintf("    \"%s\" %s", col.Name, col.Type)
						if !col.Nullable {
							line += " NOT NULL"
						}
						if col.Default != nil {
							line += fmt.Sprintf(" DEFAULT %s", *col.Default)
						}
						if i < len(t.Columns)-1 || len(activeConstraints) > 0 {
							line += ","
						}
						dbSb.WriteString(line + "\n")
					}
					for i, c := range activeConstraints {
						line := "    "
						switch c.Type {
						case "PRIMARY KEY":
							line += fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(quoteIdentifiers(c.Columns), ", "))
						case "FOREIGN KEY":
							refTable := ""
							if c.ReferencedTable != nil {
								refTable = *c.ReferencedTable
							}
							line += fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s", strings.Join(quoteIdentifiers(c.Columns), ", "), refTable)
						case "UNIQUE":
							line += fmt.Sprintf("UNIQUE (%s)", strings.Join(quoteIdentifiers(c.Columns), ", "))
						case "CHECK":
							expr := ""
							if c.CheckExpression != nil {
								expr = *c.CheckExpression
							}
							line += fmt.Sprintf("CHECK (%s)", expr)
						default:
							line += fmt.Sprintf("CONSTRAINT \"%s\" %s", c.Name, c.Type)
						}
						if i < len(activeConstraints)-1 {
							line += ","
						}
						dbSb.WriteString(line + "\n")
					}
					dbSb.WriteString(");\n")
					dbSb.WriteString("```\n\n")
				}
			}

			if len(s.Views) > 0 {
				dbSb.WriteString("#### Views\n\n")
				for _, v := range s.Views {
					dbSb.WriteString(fmt.Sprintf("##### View: \"%s\".\"%s\"\n\n", s.Name, v.Name))
					if v.Comment != "" {
						dbSb.WriteString(v.Comment + "\n\n")
					}

					// Show columns table
					if len(v.Columns) > 0 {
						headers := []string{"Column", "Type", "Comment"}
						rows := make([][]string, 0, len(v.Columns))
						for _, col := range v.Columns {
							rows = append(rows, []string{
								col.Name,
								col.Type,
								col.Comment,
							})
						}
						dbSb.WriteString(formatPaddedTable(headers, rows))
						dbSb.WriteString("\n")
					}

					dbSb.WriteString("```sql\n")
					if v.Definition != "" {
						dbSb.WriteString(v.Definition + "\n")
					} else {
						// Fallback if definition is missing
						dbSb.WriteString(fmt.Sprintf("CREATE VIEW \"%s\".\"%s\".\"%s\" AS ...\n", db.Name, s.Name, v.Name))
					}
					dbSb.WriteString("```\n\n")
				}
			}
		}

		if dbHasContent {
			sb.WriteString(dbSb.String())
		}
	}

	return sb.String()
}

func quoteIdentifiers(ids []string) []string {
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = fmt.Sprintf("\"%s\"", id)
	}
	return quoted
}

func formatPaddedTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for _, row := range rows {
		for i, val := range row {
			if i < len(widths) && len(val) > widths[i] {
				widths[i] = len(val)
			}
		}
	}

	var sb strings.Builder

	// Header
	sb.WriteString("|")
	for i, h := range headers {
		sb.WriteString(fmt.Sprintf(" %-*s |", widths[i], h))
	}
	sb.WriteString("\n")

	// Separator
	sb.WriteString("|")
	for _, w := range widths {
		sb.WriteString(fmt.Sprintf(" %s |", strings.Repeat("-", w)))
	}
	sb.WriteString("\n")

	// Rows
	for _, row := range rows {
		sb.WriteString("|")
		for i, val := range row {
			if i < len(widths) {
				sb.WriteString(fmt.Sprintf(" %-*s |", widths[i], val))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
