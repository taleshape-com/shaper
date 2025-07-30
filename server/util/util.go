package util

import (
	"math/rand"
	"strings"
)

// Human-readable random string
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
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
	lines := strings.SplitSeq(sql, "\n")
	for line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			// Only take the part before the comment
			line = line[:idx]
		}
		if strings.TrimSpace(line) != "" {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}
	return result.String()
}
