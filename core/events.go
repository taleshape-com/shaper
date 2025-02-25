package core

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/nats-io/nats.go"
	"github.com/nrednav/cuid2"
)

var tableNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,127}$`)

type EventResponse struct {
	ID string `json:"id"`
}

func PublishEvent(ctx context.Context, app *App, tableName string, payload interface{}) (*EventResponse, error) {
	if !tableNameRegex.MatchString(tableName) {
		return nil, fmt.Errorf("invalid table name. Must be alphanumeric with underscores, max 64 characters")
	}

	// Verify it can be marshaled to JSON
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON structure: %w", err)
	}

	id := cuid2.Generate()
	msg := nats.NewMsg(app.IngestSubjectPrefix + tableName)
	msg.Data = jsonBytes
	msg.Header.Set("Nats-Msg-Id", id)

	_, err = app.JetStream.PublishMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to publish event (id: %s): %w", id, err)
	}

	return &EventResponse{ID: id}, nil
}
