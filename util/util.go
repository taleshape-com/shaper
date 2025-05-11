package util

import (
	"math/rand"
	"strings"
)

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func EscapeSQLString(str string) string {
	// Replace single quotes with doubled single quotes
	escaped := strings.Replace(str, "'", "''", -1)

	// Optional: Replace other potentially dangerous characters
	escaped = strings.Replace(escaped, "\x00", "", -1) // Remove null bytes
	escaped = strings.Replace(escaped, "\n", " ", -1)  // Replace newlines
	escaped = strings.Replace(escaped, "\r", " ", -1)  // Replace carriage returns
	escaped = strings.Replace(escaped, "\x1a", "", -1) // Remove ctrl+Z

	return escaped
}

func EscapeSQLIdentifier(str string) string {
	// Replace single quotes with doubled single quotes
	escaped := strings.Replace(str, "\"", "\"\"", -1)

	// Optional: Replace other potentially dangerous characters
	escaped = strings.Replace(escaped, "\x00", "", -1) // Remove null bytes
	escaped = strings.Replace(escaped, "\n", " ", -1)  // Replace newlines
	escaped = strings.Replace(escaped, "\r", " ", -1)  // Replace carriage returns
	escaped = strings.Replace(escaped, "\x1a", "", -1) // Remove ctrl+Z

	return escaped
}

func StripSQLComments(sql string) string {
	var result strings.Builder
	lines := strings.Split(sql, "\n")

	for _, line := range lines {
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
