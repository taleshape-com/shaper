package core

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/nats-io/nats.go"
	"github.com/nrednav/cuid2"
)

const ID_COLUMN = "_id"

// TODO: Support schema prefix in the table name
var tableNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,127}$`)

func PublishEvent(ctx context.Context, app *App, tableName string, payload any) (string, error) {
	if !tableNameRegex.MatchString(tableName) {
		return "", fmt.Errorf("invalid table name. Must be alphanumeric with underscores, max 64 characters")
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
	if !tableNameRegex.MatchString(tableName) {
		return nil, fmt.Errorf("invalid table name. Must be alphanumeric with underscores, max 64 characters")
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
