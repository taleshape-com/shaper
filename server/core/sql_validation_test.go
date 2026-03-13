// SPDX-License-Identifier: MPL-2.0

package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAllowedStatement(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected bool
	}{
		// Basic Allowed
		{"Select", "SELECT * FROM users", true},
		{"Summarize", "SUMMARIZE SELECT 1", true},
		{"Describe", "DESCRIBE users", true},
		{"Show Tables", "SHOW TABLES", true},
		{"Show All Tables", "SHOW ALL TABLES", true},
		{"Explain", "EXPLAIN SELECT 1", true},
		{"Explain Analyze", "EXPLAIN ANALYZE SELECT 1", true},
		{"Pivot", "PIVOT sales ON region USING SUM(amount)", true},
		{"Unpivot", "UNPIVOT sales ON region USING SUM(amount)", true},

		// Basic Disallowed
		{"Drop Table", "DROP TABLE users", false},
		{"Delete", "DELETE FROM users", false},
		{"Insert", "INSERT INTO users VALUES (1)", false},
		{"Update", "UPDATE users SET name = 'foo'", false},
		{"Create Table", "CREATE TABLE users (id INT)", false},
		{"Alter Table", "ALTER TABLE users ADD COLUMN name TEXT", false},

		// Side Effects (Allowed)
		{"Set", "SET VARIABLE x = 1", true},
		{"Attach", "ATTACH 'file.db' AS other", false},
		{"Use", "USE other", false},
		{"Create Temp Table", "CREATE TEMPORARY TABLE foo AS SELECT 1", true},
		{"Begin", "BEGIN TRANSACTION", true},
		{"Commit", "COMMIT", true},

		// WITH Statements
		{"With Select", "WITH t AS (SELECT 1) SELECT * FROM t", true},
		{"With Recursive", "WITH RECURSIVE t AS (SELECT 1) SELECT * FROM t", true},
		{"With Multiple CTEs", "WITH t1 AS (SELECT 1), t2 AS (SELECT 2) SELECT * FROM t1, t2", true},
		{"With Disallowed in CTE", "WITH t AS (DROP TABLE x) SELECT 1", false},
		{"With Disallowed in Main", "WITH t AS (SELECT 1) DROP TABLE x", false},
		{"With CTE Column List", "WITH t(a, b) AS (SELECT 1, 2) SELECT * FROM t", true},
		{"With Quoted CTE", "WITH \"my table\" AS (SELECT 1) SELECT * FROM \"my table\"", true},

		// Nested Queries
		{"Parenthesized Select", "(SELECT 1)", true},
		{"Union", "(SELECT 1) UNION SELECT 2", true},
		{"Union All", "(SELECT 1) UNION ALL SELECT 2", true},
		{"Nested Union", "((SELECT 1) UNION (SELECT 2))", true},
		{"Union with Disallowed", "(SELECT 1) UNION (DROP TABLE x)", false},
		{"Disallowed in Parens", "(DROP TABLE x)", false},

		// Explain cases
		{"Explain only", "EXPLAIN", true},
		{"Explain Analyze only", "EXPLAIN ANALYZE", true},
		{"Explain Disallowed", "EXPLAIN DROP TABLE x", false},
		{"Explain Analyze Disallowed", "EXPLAIN ANALYZE DROP TABLE x", false},

		// Edge Cases
		{"Leading Spaces", "   SELECT 1", true},
		{"Newlines", "\nSELECT\n1", true},
		{"Semicolon", "SELECT 1;", true},
		{"Quoted keywords in identifiers", "SELECT \"DROP\" FROM t", true},
		{"False match for keyword prefix", "SETTINGS", false},
		{"Empty string", "", true},
		{"Space string", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isAllowedStatement(tt.sql), "SQL: %s", tt.sql)
		})
	}
}

func TestSplitWithStatement(t *testing.T) {
	tests := []struct {
		name         string
		sql          string
		expectedMain string
		expectedCtes []string
		expectErr    bool
	}{
		{
			"Simple WITH",
			"WITH t AS (SELECT 1) SELECT * FROM t",
			" SELECT * FROM t",
			[]string{"SELECT 1"},
			false,
		},
		{
			"WITH Recursive",
			"WITH RECURSIVE t AS (SELECT 1) SELECT * FROM t",
			" SELECT * FROM t",
			[]string{"SELECT 1"},
			false,
		},
		{
			"Multiple CTEs",
			"WITH t1 AS (SELECT 1), t2 AS (SELECT 2) SELECT * FROM t1",
			" SELECT * FROM t1",
			[]string{"SELECT 1", "SELECT 2"},
			false,
		},
		{
			"CTE with columns",
			"WITH t(a, b) AS (SELECT 1, 2) SELECT * FROM t",
			" SELECT * FROM t",
			[]string{"SELECT 1, 2"},
			false,
		},
		{
			"Quoted CTE name",
			"WITH \"table\" AS (SELECT 1) SELECT * FROM \"table\"",
			" SELECT * FROM \"table\"",
			[]string{"SELECT 1"},
			false,
		},
		{
			"CTE with nested parens",
			"WITH t AS (SELECT (SELECT 1)) SELECT * FROM t",
			" SELECT * FROM t",
			[]string{"SELECT (SELECT 1)"},
			false,
		},
		{
			"CTE with strings containing parens",
			"WITH t AS (SELECT '(parenthesized)') SELECT * FROM t",
			" SELECT * FROM t",
			[]string{"SELECT '(parenthesized)'"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			main, ctes, err := splitWithStatement(tt.sql)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, strings.TrimSpace(tt.expectedMain), strings.TrimSpace(main))
				assert.Equal(t, tt.expectedCtes, ctes)
			}
		})
	}
}
