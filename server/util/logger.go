// SPDX-License-Identifier: MPL-2.0

package util

import (
	"log/slog"
	"strings"
)

// RedactSensitiveAttrs redacts sensitive information from log attributes.
// It specifically handles URI/Path attributes by redacting download keys and invite codes.
func RedactSensitiveAttrs(groups []string, a slog.Attr) slog.Attr {
	if (a.Key == "uri" || a.Key == "path" || a.Key == "url") && a.Value.Kind() == slog.KindString {
		path := a.Value.String()

		// Redact download keys: /api/download/<key>/<filename>
		if strings.Contains(path, "/api/download/") {
			parts := strings.Split(path, "/")
			// We expect at least two segments after "download": the key and the filename.
			for i := 0; i < len(parts)-2; i++ {
				if parts[i] == "download" {
					parts[i+1] = "[REDACTED]"
					return slog.String(a.Key, strings.Join(parts, "/"))
				}
			}
		}

		// Redact invite codes: /api/invites/<code>
		if strings.Contains(path, "/api/invites/") {
			parts := strings.Split(path, "/")
			// We expect at least one segment after "invites": the code.
			for i := 0; i < len(parts)-1; i++ {
				if parts[i] == "invites" {
					parts[i+1] = "[REDACTED]"
					return slog.String(a.Key, strings.Join(parts, "/"))
				}
			}
		}
	}

	// Redact specific sensitive parameter keys if they are within a web or params group.
	// This helps redact them when they appear in a group of parameters.
	if (a.Key == "key" || a.Key == "code") && a.Value.Kind() == slog.KindString {
		for _, g := range groups {
			if g == "web" || g == "params" {
				return slog.String(a.Key, "[REDACTED]")
			}
		}
	}

	return a
}
