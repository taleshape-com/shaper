// SPDX-License-Identifier: MPL-2.0

package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

const (
	// MaxVariableNameLength is the maximum allowed length for SQL/DuckDB variable names.
	MaxVariableNameLength = 64
	// MaxInputVariableLength is the maximum allowed string length for free-text input variables.
	MaxInputVariableLength = 4096
	// MaxMultiVariableCount is the maximum number of items allowed in a multi-variable array.
	MaxMultiVariableCount = 500
	// MaxMultiVariableItemLength is the maximum string length for each item in a multi-variable array.
	MaxMultiVariableItemLength = 4096
)

var validVariableNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// IsValidVariableName checks if a string is a valid SQL/DuckDB variable name
// (alphanumeric and underscore, starting with a letter or underscore, at most MaxVariableNameLength chars).
func IsValidVariableName(name string) bool {
	if len(name) == 0 || len(name) > MaxVariableNameLength {
		return false
	}
	return validVariableNameRegex.MatchString(name)
}


// Human-readable random string
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	max := big.NewInt(int64(len(charset)))
	for i := range b {
		num, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(fmt.Sprintf("failed to generate random string: %v", err))
		}
		b[i] = charset[num.Int64()]
	}
	return string(b)
}

func EscapeSQLString(str string) string {
	escaped := strings.ReplaceAll(str, "'", "''")
	escaped = strings.ReplaceAll(escaped, "\x00", "") // Remove null bytes
	escaped = strings.ReplaceAll(escaped, "\n", " ")  // Replace newlines
	escaped = strings.ReplaceAll(escaped, "\r", " ")  // Replace carriage returns
	escaped = strings.ReplaceAll(escaped, "\x1a", "") // Remove ctrl+Z
	return escaped
}

func EscapeSQLIdentifier(str string) string {
	escaped := strings.ReplaceAll(str, "\"", "\"\"")
	escaped = strings.ReplaceAll(escaped, "\x00", "") // Remove null bytes
	escaped = strings.ReplaceAll(escaped, "\n", " ")  // Replace newlines
	escaped = strings.ReplaceAll(escaped, "\r", " ")  // Replace carriage returns
	escaped = strings.ReplaceAll(escaped, "\x1a", "") // Remove ctrl+Z
	return escaped
}

func StripSQLComments(sql string) string {
	var result strings.Builder
	var inSingleQuote bool
	var inDoubleQuote bool

	for i := 0; i < len(sql); i++ {
		c := sql[i]

		// Handle single quotes
		if c == '\'' && !inDoubleQuote {
			if i+1 < len(sql) && sql[i+1] == '\'' {
				// Escaped single quote
				result.WriteByte(c)
				result.WriteByte(sql[i+1])
				i++
				continue
			}
			inSingleQuote = !inSingleQuote
			result.WriteByte(c)
			continue
		}

		// Handle double quotes
		if c == '"' && !inSingleQuote {
			if i+1 < len(sql) && sql[i+1] == '"' {
				// Escaped double quote
				result.WriteByte(c)
				result.WriteByte(sql[i+1])
				i++
				continue
			}
			inDoubleQuote = !inDoubleQuote
			result.WriteByte(c)
			continue
		}

		// Check for comment start (--) only when not inside quotes
		if c == '-' && !inSingleQuote && !inDoubleQuote {
			if i+1 < len(sql) && sql[i+1] == '-' {
				// Found comment start, skip to end of line
				for i < len(sql) && sql[i] != '\n' {
					i++
				}
				// Write the newline if we found one
				if i < len(sql) {
					result.WriteByte(sql[i])
				}
				continue
			}
		}

		result.WriteByte(c)
	}

	return result.String()
}

// Split by ; and handle ; inside single and double quotes
func SplitSQLQueries(sql string) ([]string, error) {
	var queries []string
	var currentQuery strings.Builder
	var inSingleQuote bool
	var inDoubleQuote bool
	var lineNum int = 1
	var quoteStartLine int

	for i := 0; i < len(sql); i++ {
		c := sql[i]
		currentQuery.WriteByte(c)

		// Track line numbers
		if c == '\n' {
			lineNum++
		}

		// Handle single quotes
		if c == '\'' && !inDoubleQuote {
			if i+1 < len(sql) && sql[i+1] == '\'' {
				currentQuery.WriteByte(sql[i+1])
				i++
				continue
			}
			if !inSingleQuote {
				quoteStartLine = lineNum
			}
			inSingleQuote = !inSingleQuote
			continue
		}

		// Handle double quotes
		if c == '"' && !inSingleQuote {
			if i+1 < len(sql) && sql[i+1] == '"' {
				currentQuery.WriteByte(sql[i+1])
				i++
				continue
			}
			if !inDoubleQuote {
				quoteStartLine = lineNum
			}
			inDoubleQuote = !inDoubleQuote
			continue
		}

		// Handle semicolon
		if c == ';' && !inSingleQuote && !inDoubleQuote {
			query := strings.TrimSpace(currentQuery.String())
			if len(query) > 0 {
				queries = append(queries, query[:len(query)-1]) // Remove the semicolon
			}
			currentQuery.Reset()
		}
	}

	if inSingleQuote {
		return nil, fmt.Errorf("unclosed single quote starting in line %d", quoteStartLine+1)
	}
	if inDoubleQuote {
		return nil, fmt.Errorf("unclosed double quote starting in line %d", quoteStartLine+1)
	}

	lastQuery := strings.TrimSpace(currentQuery.String())
	if lastQuery != "" {
		queries = append(queries, lastQuery)
	}

	return queries, nil
}
