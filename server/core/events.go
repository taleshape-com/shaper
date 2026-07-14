// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/nrednav/cuid2"
)

const ID_COLUMN = "_id"

var partRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,127}$`)

func validateTableName(tableName string) bool {
	parts := strings.Split(tableName, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return false
	}
	for _, part := range parts {
		if !partRegex.MatchString(part) {
			return false
		}
	}
	return true
}

func PublishEvent(ctx context.Context, app *App, tableName string, payload any) (string, error) {
	if !validateTableName(tableName) {
		return "", fmt.Errorf("invalid table name. Each part must be alphanumeric with underscores (max 128 characters) and separated by dots (up to 3 parts)")
	}

	// Verify it can be marshaled to JSON
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("invalid JSON structure: %w", err)
	}

	id := cuid2.Generate()
	msg := nats.NewMsg(app.IngestSubjectPrefix + tableName)
	msg.Data = jsonBytes
	msg.Header.Set("Nats-Msg-Id", id)

	_, err = app.JetStream.PublishMsg(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("failed to publish event (id: %s): %w", id, err)
	}

	return id, nil
}

func PublishEvents(ctx context.Context, app *App, tableName string, payloads []map[string]any) ([]string, error) {
	if !validateTableName(tableName) {
		return nil, fmt.Errorf("invalid table name. Each part must be alphanumeric with underscores (max 128 characters) and separated by dots (up to 3 parts)")
	}

	ids := make([]string, 0, len(payloads))

	for _, payload := range payloads {
		// Verify it can be marshaled to JSON
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			return ids, fmt.Errorf("invalid JSON structure: %w", err)
		}

		id := ""
		idCol, ok := payload[ID_COLUMN]
		if ok {
			if idColStr, ok := idCol.(string); ok && idColStr != "" {
				id = idColStr
			} else {
				id = cuid2.Generate()
			}
		} else {
			id = cuid2.Generate()
		}
		msg := nats.NewMsg(app.IngestSubjectPrefix + tableName)
		msg.Data = jsonBytes
		msg.Header.Set("Nats-Msg-Id", id)

		_, err = app.JetStream.PublishMsg(ctx, msg)
		if err != nil {
			return ids, fmt.Errorf("failed to publish event (id: %s): %w", id, err)
		}

		ids = append(ids, id)
	}

	return ids, nil
}
