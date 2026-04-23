// SPDX-License-Identifier: MPL-2.0

package core

import (
	"fmt"
	"strings"
)

// These SQL statements are used only for their side effects and not to display anything.
// They are not visible in the dashboard output.
var sideEffectSQLStatements = [][]string{
	{"USE"},
	{"SET", "VARIABLE"},
	{"BEGIN"},
	{"COMMIT"},
	{"ROLLBACK"},
	{"ABORT"},
	{"CALL"},
	{"RESET", "VARIABLE"},
	{"CREATE", "TEMPORARY", "TABLE"},
	{"CREATE", "TEMPORARY", "VIEW"},
	{"CREATE", "TEMP", "TABLE"},
	{"CREATE", "TEMP", "VIEW"},
	{"CREATE", "OR", "REPLACE", "TEMPORARY", "TABLE"},
	{"CREATE", "OR", "REPLACE", "TEMPORARY", "VIEW"},
	{"CREATE", "OR", "REPLACE", "TEMP", "TABLE"},
	{"CREATE", "OR", "REPLACE", "TEMP", "VIEW"},
	{"CREATE", "TEMP", "MACRO"},
	{"CREATE", "TEMP", "FUNCTION"},
	{"CREATE", "TEMPORARY", "MACRO"},
	{"CREATE", "TEMPORARY", "FUNCTION"},
	{"CREATE", "TEMP", "MACRO", "IF", "NOT", "EXISTS"},
	{"CREATE", "TEMP", "FUNCTION", "IF", "NOT", "EXISTS"},
	{"CREATE", "TEMPORARY", "MACRO", "IF", "NOT", "EXISTS"},
	{"CREATE", "TEMPORARY", "FUNCTION", "IF", "NOT", "EXISTS"},
	{"CREATE", "OR", "REPLACE", "TEMP", "MACRO"},
	{"CREATE", "OR", "REPLACE", "TEMP", "FUNCTION"},
	{"CREATE", "OR", "REPLACE", "TEMPORARY", "MACRO"},
	{"CREATE", "OR", "REPLACE", "TEMPORARY", "FUNCTION"},
}

// These SQL statements are allowed to read data.
var allowedDataReadingStatements = [][]string{
	{"SELECT"},
	{"FROM"},
	{"VALUES"},
	{"SUMMARIZE"},
	{"DESC"},
	{"DESCRIBE"},
	{"SHOW", "TABLES"},
	{"SHOW", "ALL", "TABLES"},
	{"PIVOT"},
	{"UNPIVOT"},
	{"EXPLAIN"},
}

var disallowedTaskStatements = [][]string{
	{"INSTALL"},
	{"LOAD"},
	{"PRAGMA"},
}

// Some SQL statements are only used for their side effects and should not be shown on the dashboard.
// We ignore case and extra whitespace when matching the statements
func isSideEffect(app *App, sqlString string) bool {
	upper := strings.ToUpper(strings.TrimSpace(sqlString))
	if app != nil && app.DuckDBDSN == ":memory:" && strings.HasPrefix(upper, "ATTACH") {
		return true
	}
	for _, stmt := range sideEffectSQLStatements {
		if matchesPrefix(upper, stmt) {
			return true
		}
	}
	return false
}

func matchesPrefix(upperSql string, prefix []string) bool {
	sub := upperSql
	for _, s := range prefix {
		if !strings.HasPrefix(sub, s) {
			return false
		}
		// Check that it's a whole word
		after := sub[len(s):]
		if len(after) > 0 && !isSpace(after[0]) && after[0] != '(' && after[0] != ';' && after[0] != ',' {
			return false
		}
		sub = strings.TrimSpace(after)
	}
	return true
}

func IsAllowedStatement(app *App, sql string) bool {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return true
	}
	upper := strings.ToUpper(sql)

	// Handle WITH
	if strings.HasPrefix(upper, "WITH") {
		remaining, ctes, err := splitWithStatement(sql)
		if err != nil {
			return false
		}
		for _, cte := range ctes {
			if !IsAllowedStatement(app, cte) {
				return false
			}
		}
		return IsAllowedStatement(app, remaining)
	}

	// Handle parenthesized queries like (SELECT 1)
	if strings.HasPrefix(upper, "(") {
		inner, remaining, err := splitParenthesized(sql)
		if err != nil {
			return false
		}
		if !IsAllowedStatement(app, inner) {
			return false
		}
		remaining = strings.TrimSpace(remaining)
		if remaining == "" {
			return true
		}
		remUpper := strings.ToUpper(remaining)
		operators := []string{"UNION", "INTERSECT", "EXCEPT"}
		for _, op := range operators {
			if strings.HasPrefix(remUpper, op) {
				pos := len(op)
				rest := strings.TrimSpace(remaining[pos:])
				restUpper := strings.ToUpper(rest)
				if strings.HasPrefix(restUpper, "ALL") {
					rest = strings.TrimSpace(rest[len("ALL"):])
				} else if strings.HasPrefix(restUpper, "DISTINCT") {
					rest = strings.TrimSpace(rest[len("DISTINCT"):])
				}
				return IsAllowedStatement(app, rest)
			}
		}
		// Also handle ORDER BY, LIMIT etc which can follow a parenthesized query
		if strings.HasPrefix(remUpper, "ORDER") || strings.HasPrefix(remUpper, "LIMIT") || strings.HasPrefix(remUpper, "OFFSET") || strings.HasPrefix(remUpper, "FETCH") {
			return true // These are reading-only modifiers
		}
		return false
	}

	// Check against side effects
	if isSideEffect(app, sql) {
		return true
	}

	// Check against allowed reading statements
	for _, stmt := range allowedDataReadingStatements {
		if matchesPrefix(upper, stmt) {
			// If it's EXPLAIN, check what follows
			if stmt[0] == "EXPLAIN" {
				rest := strings.TrimSpace(sql[len("EXPLAIN"):])
				if rest == "" {
					return true
				}
				// Handle EXPLAIN ANALYZE
				if strings.HasPrefix(strings.ToUpper(rest), "ANALYZE") {
					rest = strings.TrimSpace(rest[len("ANALYZE"):])
				}
				if rest == "" {
					return true
				}
				return IsAllowedStatement(app, rest)
			}
			return true
		}
	}

	return false
}

func IsAllowedTaskStatement(sql string) bool {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return true
	}
	upper := strings.ToUpper(sql)

	// Handle WITH
	if strings.HasPrefix(upper, "WITH") {
		remaining, ctes, err := splitWithStatement(sql)
		if err != nil {
			return false
		}
		for _, cte := range ctes {
			if !IsAllowedTaskStatement(cte) {
				return false
			}
		}
		return IsAllowedTaskStatement(remaining)
	}

	// Handle parenthesized queries like (SELECT 1)
	if strings.HasPrefix(upper, "(") {
		inner, remaining, err := splitParenthesized(sql)
		if err != nil {
			return false
		}
		if !IsAllowedTaskStatement(inner) {
			return false
		}
		remaining = strings.TrimSpace(remaining)
		if remaining == "" {
			return true
		}
		remUpper := strings.ToUpper(remaining)
		operators := []string{"UNION", "INTERSECT", "EXCEPT"}
		for _, op := range operators {
			if strings.HasPrefix(remUpper, op) {
				pos := len(op)
				rest := strings.TrimSpace(remaining[pos:])
				restUpper := strings.ToUpper(rest)
				if strings.HasPrefix(restUpper, "ALL") {
					rest = strings.TrimSpace(rest[len("ALL"):])
				} else if strings.HasPrefix(restUpper, "DISTINCT") {
					rest = strings.TrimSpace(rest[len("DISTINCT"):])
				}
				return IsAllowedTaskStatement(rest)
			}
		}
		// Also handle ORDER BY, LIMIT etc which can follow a parenthesized query
		if strings.HasPrefix(remUpper, "ORDER") || strings.HasPrefix(remUpper, "LIMIT") || strings.HasPrefix(remUpper, "OFFSET") || strings.HasPrefix(remUpper, "FETCH") {
			return true
		}
		return false
	}

	// Block specific statements
	for _, stmt := range disallowedTaskStatements {
		if matchesPrefix(upper, stmt) {
			return false
		}
	}

	// Handle SET config: if it's SET but not SET VARIABLE
	if strings.HasPrefix(upper, "SET") {
		if matchesPrefix(upper, []string{"SET", "VARIABLE"}) {
			return true
		}
		return false
	}
	// Handle RESET config: if it's RESET but not RESET VARIABLE
	if strings.HasPrefix(upper, "RESET") {
		if matchesPrefix(upper, []string{"RESET", "VARIABLE"}) {
			return true
		}
		return false
	}

	return true
}

func splitWithStatement(sql string) (string, []string, error) {
	upper := strings.ToUpper(sql)
	if !strings.HasPrefix(upper, "WITH") {
		return "", nil, fmt.Errorf("not a WITH statement")
	}

	pos := len("WITH")
	// Skip RECURSIVE
	restUpper := strings.TrimSpace(upper[pos:])
	if strings.HasPrefix(restUpper, "RECURSIVE") {
		pos += strings.Index(upper[pos:], "RECURSIVE") + len("RECURSIVE")
	}

	var ctes []string
	for {
		// Skip spaces
		for pos < len(sql) && isSpace(sql[pos]) {
			pos++
		}
		if pos >= len(sql) {
			return "", nil, fmt.Errorf("unexpected end of WITH statement")
		}

		// Skip CTE name and optional column list
		newPos, err := skipIdentifier(sql, pos)
		if err != nil {
			return "", nil, err
		}
		pos = newPos

		// Optional column list (col1, col2, ...)
		for pos < len(sql) && isSpace(sql[pos]) {
			pos++
		}
		if pos < len(sql) && sql[pos] == '(' {
			endParen, err := findClosingParen(sql, pos)
			if err != nil {
				return "", nil, err
			}
			pos = endParen + 1
		}

		// Expect AS
		for pos < len(sql) && isSpace(sql[pos]) {
			pos++
		}
		if !strings.HasPrefix(strings.ToUpper(sql[pos:]), "AS") {
			return "", nil, fmt.Errorf("missing AS in WITH clause")
		}
		pos += 2

		// Expect (
		for pos < len(sql) && isSpace(sql[pos]) {
			pos++
		}
		if pos >= len(sql) || sql[pos] != '(' {
			return "", nil, fmt.Errorf("missing ( after AS in WITH clause")
		}

		// Find matching )
		endParen, err := findClosingParen(sql, pos)
		if err != nil {
			return "", nil, err
		}

		ctes = append(ctes, sql[pos+1:endParen])
		pos = endParen + 1

		// Check for comma or main query
		for pos < len(sql) && isSpace(sql[pos]) {
			pos++
		}
		if pos >= len(sql) {
			return "", nil, fmt.Errorf("unexpected end after CTE")
		}
		if sql[pos] == ',' {
			pos++
			continue
		} else {
			// The rest is the main query
			return sql[pos:], ctes, nil
		}
	}
}

func splitParenthesized(sql string) (inner string, remaining string, err error) {
	sql = strings.TrimSpace(sql)
	if !strings.HasPrefix(sql, "(") {
		return "", "", fmt.Errorf("not a parenthesized statement")
	}
	endParen, err := findClosingParen(sql, 0)
	if err != nil {
		return "", "", err
	}
	return sql[1:endParen], sql[endParen+1:], nil
}

func findClosingParen(sql string, startPos int) (int, error) {
	var inSingleQuote bool
	var inDoubleQuote bool
	depth := 0
	for i := startPos; i < len(sql); i++ {
		c := sql[i]
		if c == '\'' && !inDoubleQuote {
			if i+1 < len(sql) && sql[i+1] == '\'' {
				i++ // Escaped quote
				continue
			}
			inSingleQuote = !inSingleQuote
			continue
		}
		if c == '"' && !inSingleQuote {
			if i+1 < len(sql) && sql[i+1] == '"' {
				i++ // Escaped quote
				continue
			}
			inDoubleQuote = !inDoubleQuote
			continue
		}
		if inSingleQuote || inDoubleQuote {
			continue
		}
		if c == '(' {
			depth++
		} else if c == ')' {
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return -1, fmt.Errorf("unmatched parenthesis")
}

func skipIdentifier(sql string, pos int) (int, error) {
	if pos >= len(sql) {
		return pos, nil
	}
	if sql[pos] == '"' {
		// Quoted identifier
		for i := pos + 1; i < len(sql); i++ {
			if sql[i] == '"' {
				if i+1 < len(sql) && sql[i+1] == '"' {
					i++ // Escaped quote
					continue
				}
				return i + 1, nil
			}
		}
		return -1, fmt.Errorf("unclosed double quote")
	}
	// Unquoted identifier
	for i := pos; i < len(sql); i++ {
		c := sql[i]
		if isSpace(c) || c == '(' || c == ')' || c == ',' || c == ';' || c == '.' {
			return i, nil
		}
	}
	return len(sql), nil
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\n' || c == '\t' || c == '\r'
}
