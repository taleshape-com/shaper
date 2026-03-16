// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"shaper/server/util"
	"strings"
	"time"

	"github.com/nrednav/cuid2"
)

const API_KEY_PREFIX = "shaperkey."

const (
	PermissionReadMetrics = "metrics"
	PermissionIngestData  = "data:ingest"
	PermissionDeploy      = "deploy"
	PermissionQueryData   = "data:query"
)

var AllPermissions = []string{
	PermissionReadMetrics,
	PermissionIngestData,
	PermissionDeploy,
	PermissionQueryData,
}

type APIKey struct {
	ID              string    `db:"id" json:"id"`
	Name            string    `db:"name" json:"name"`
	Hash            string    `db:"hash" json:"-"`
	Salt            string    `db:"salt" json:"-"`
	Permissions     *string   `db:"permissions" json:"-"`
	PermissionsList []string  `db:"-" json:"permissions"`
	CreatedAt       time.Time `db:"created_at" json:"createdAt"`
	CreatedBy       *string   `db:"created_by" json:"createdBy,omitempty"`
}

func (k APIKey) HasPermission(permission string) bool {
	if k.Permissions == nil || *k.Permissions == "" {
		return true // Default to all permissions for legacy keys
	}
	var perms []string
	err := json.Unmarshal([]byte(*k.Permissions), &perms)
	if err != nil {
		return false
	}
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

type APIKeyListResult struct {
	Keys []APIKey `json:"keys"`
}

type CreateAPIKeyPayload struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Name        string    `json:"name"`
	Hash        string    `json:"hash"`
	Salt        string    `json:"salt"`
	Permissions []string  `json:"permissions"`
	CreatedBy   string    `json:"createdBy"`
}

func ListAPIKeys(app *App, ctx context.Context) (APIKeyListResult, error) {
	keys := []APIKey{}
	err := app.Sqlite.SelectContext(ctx, &keys,
		`SELECT id, name, permissions, created_at, created_by
		 FROM api_keys
		 ORDER BY created_at desc`)
	if err != nil {
		return APIKeyListResult{}, fmt.Errorf("error listing api keys: %w", err)
	}

	for i := range keys {
		if keys[i].Permissions == nil || *keys[i].Permissions == "" {
			keys[i].PermissionsList = AllPermissions
		} else {
			_ = json.Unmarshal([]byte(*keys[i].Permissions), &keys[i].PermissionsList)
		}
	}

	return APIKeyListResult{Keys: keys}, nil
}

func CreateAPIKey(app *App, ctx context.Context, name string, permissions []string) (string, string, error) {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return "", "", fmt.Errorf("no actor in context")
	}
	name = strings.TrimSpace(name)
	id := cuid2.Generate()
	suffix := util.GenerateRandomString(32)
	key := fmt.Sprintf("%s%s.%s", API_KEY_PREFIX, id, suffix)

	salt := util.GenerateRandomString(32)
	mac := hmac.New(sha256.New, []byte(salt))
	mac.Write([]byte(key))
	hash := hex.EncodeToString(mac.Sum(nil))

	if permissions == nil {
		permissions = AllPermissions
	}

	payload := CreateAPIKeyPayload{
		ID:          id,
		Timestamp:   time.Now(),
		Name:        name,
		Hash:        hash,
		Salt:        salt,
		Permissions: permissions,
		CreatedBy:   actor.String(),
	}
	err := app.SubmitState(ctx, "create_api_key", payload)
	return id, key, err
}

func HandleCreateAPIKey(app *App, data []byte) bool {
	var payload CreateAPIKeyPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal create api key payload", slog.Any("error", err))
		return false
	}

	perms, err := json.Marshal(payload.Permissions)
	if err != nil {
		app.Logger.Error("failed to marshal api key permissions", slog.Any("error", err))
		return false
	}

	// Insert into DB
	_, err = app.Sqlite.Exec(
		`INSERT OR IGNORE INTO api_keys (
			id, hash, salt, name, permissions, created_at, updated_at, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $7)`,
		payload.ID, payload.Hash, payload.Salt, payload.Name, string(perms), payload.Timestamp, payload.CreatedBy,
	)
	if err != nil {
		app.Logger.Error("failed to insert api key into DB", slog.Any("error", err))
		return false
	}
	return true
}

type DeleteAPIKeyPayload struct {
	ID        string    `json:"id"`
	TimeStamp time.Time `json:"timestamp"`
	// NOTE: Not used, but might want to log this in the future
	DeletedBy string `json:"deletedBy"`
}

func DeleteAPIKey(app *App, ctx context.Context, id string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}
	var count int
	err := app.Sqlite.GetContext(ctx, &count, `SELECT COUNT(*) FROM api_keys WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to query api key: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("api key not found")
	}
	err = app.SubmitState(ctx, "delete_api_key", DeleteAPIKeyPayload{
		ID:        id,
		TimeStamp: time.Now(),
		DeletedBy: actor.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to submit API key deletion: %w", err)
	}
	return nil
}

func HandleDeleteAPIKey(app *App, data []byte) bool {
	var payload DeleteAPIKeyPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal delete api key payload", slog.Any("error", err))
		return false
	}
	_, err = app.Sqlite.Exec(
		`DELETE FROM api_keys WHERE id = $1`, payload.ID)
	if err != nil {
		app.Logger.Error("failed to execute DELETE statement", slog.Any("error", err))
		return false
	}
	return true
}

type UpdateAPIKeyPermissionsPayload struct {
	ID          string    `json:"id"`
	Permissions []string  `json:"permissions"`
	Timestamp   time.Time `json:"timestamp"`
	UpdatedBy   string    `json:"updatedBy"`
}

func UpdateAPIKeyPermissions(app *App, ctx context.Context, id string, permissions []string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}

	payload := UpdateAPIKeyPermissionsPayload{
		ID:          id,
		Permissions: permissions,
		Timestamp:   time.Now(),
		UpdatedBy:   actor.String(),
	}

	return app.SubmitState(ctx, "update_api_key_permissions", payload)
}

func HandleUpdateAPIKeyPermissions(app *App, data []byte) bool {
	var payload UpdateAPIKeyPermissionsPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal update api key permissions payload", slog.Any("error", err))
		return false
	}

	perms, err := json.Marshal(payload.Permissions)
	if err != nil {
		app.Logger.Error("failed to marshal api key permissions", slog.Any("error", err))
		return false
	}

	_, err = app.Sqlite.Exec(
		`UPDATE api_keys SET permissions = $1, updated_at = $2, updated_by = $3 WHERE id = $4`,
		string(perms), payload.Timestamp, payload.UpdatedBy, payload.ID,
	)
	if err != nil {
		app.Logger.Error("failed to update api key permissions in DB", slog.Any("error", err))
		return false
	}
	return true
}
