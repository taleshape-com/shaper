// SPDX-License-Identifier: MPL-2.0

package util

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactSensitiveAttrs(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		val      string
		expected string
		groups   []string
	}{
		{
			name:     "Redact download key",
			key:      "uri",
			val:      "/api/download/SECRET_KEY/report.pdf",
			expected: "/api/download/[REDACTED]/report.pdf",
		},
		{
			name:     "Redact download key in path",
			key:      "path",
			val:      "/api/download/SECRET_KEY/report.pdf",
			expected: "/api/download/[REDACTED]/report.pdf",
		},
		{
			name:     "Do not redact download without key",
			key:      "uri",
			val:      "/api/download/report.pdf",
			expected: "/api/download/report.pdf",
		},
		{
			name:     "Redact invite code",
			key:      "uri",
			val:      "/api/invites/SECRET_CODE",
			expected: "/api/invites/[REDACTED]",
		},
		{
			name:     "Redact invite code with claim",
			key:      "uri",
			val:      "/api/invites/SECRET_CODE/claim",
			expected: "/api/invites/[REDACTED]/claim",
		},
		{
			name:     "Do not redact non-sensitive path",
			key:      "uri",
			val:      "/api/dashboards/123",
			expected: "/api/dashboards/123",
		},
		{
			name:     "Do not redact non-path attribute",
			key:      "msg",
			val:      "/api/download/SECRET_KEY/report.pdf",
			expected: "/api/download/SECRET_KEY/report.pdf",
		},
		{
			name:     "Redact key in params group",
			key:      "key",
			val:      "SECRET_KEY",
			expected: "[REDACTED]",
			groups:   []string{"web", "params"},
		},
		{
			name:     "Redact code in params group",
			key:      "code",
			val:      "SECRET_CODE",
			expected: "[REDACTED]",
			groups:   []string{"web", "params"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := slog.String(tt.key, tt.val)
			redacted := RedactSensitiveAttrs(tt.groups, attr)
			assert.Equal(t, tt.expected, redacted.Value.String())
		})
	}
}
