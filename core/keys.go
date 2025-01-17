package core

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"shaper/util"
	"strings"
	"time"

	"github.com/nrednav/cuid2"
)

const API_KEY_PREFIX = "shaperkey."

type APIKey struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Hash      string    `db:"hash" json:"-"`
	Salt      string    `db:"salt" json:"-"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	CreatedBy *string   `db:"created_by" json:"createdBy,omitempty"`
}

type APIKeyListResult struct {
	Keys []APIKey `json:"keys"`
}

type CreateAPIKeyPayload struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Name      string    `json:"name"`
	Hash      string    `json:"hash"`
	Salt      string    `json:"salt"`
}

func ListAPIKeys(app *App, ctx context.Context) (APIKeyListResult, error) {
	keys := []APIKey{}
	err := app.db.SelectContext(ctx, &keys,
		`SELECT id, name, created_at, created_by
		 FROM "`+app.Schema+`".api_keys
		 ORDER BY created_at desc`)
	if err != nil {
		err = fmt.Errorf("error listing api keys: %w", err)
	}
	return APIKeyListResult{Keys: keys}, err
}

func CreateAPIKey(app *App, ctx context.Context, name string) (string, string, error) {
	name = strings.TrimSpace(name)
	id := cuid2.Generate()
	suffix := util.GenerateRandomString(32)
	key := fmt.Sprintf("%s%s.%s", API_KEY_PREFIX, id, suffix)

	salt := util.GenerateRandomString(32)
	mac := hmac.New(sha256.New, []byte(salt))
	mac.Write([]byte(key))
	hash := hex.EncodeToString(mac.Sum(nil))

	payload := CreateAPIKeyPayload{
		ID:        id,
		Timestamp: time.Now(),
		Name:      name,
		Hash:      hash,
		Salt:      salt,
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
	// Insert into DB
	_, err = app.db.Exec(
		`INSERT OR IGNORE INTO `+app.Schema+`.api_keys (
			id, hash, salt, name, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $5)`,
		payload.ID, payload.Hash, payload.Salt, payload.Name, payload.Timestamp,
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
}

func DeleteAPIKey(app *App, ctx context.Context, id string) error {
	var count int
	err := app.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM `+app.Schema+`.api_keys WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to query api key: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("api key not found")
	}
	err = app.SubmitState(ctx, "delete_api_key", DeleteAPIKeyPayload{
		ID:        id,
		TimeStamp: time.Now(),
	})
	return err
}

func HandleDeleteAPIKey(app *App, data []byte) bool {
	var payload DeleteAPIKeyPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal delete api key payload", slog.Any("error", err))
		return false
	}
	_, err = app.db.Exec(
		`DELETE FROM `+app.Schema+`.api_keys WHERE id = $1`, payload.ID)
	if err != nil {
		app.Logger.Error("failed to execute DELETE statement", slog.Any("error", err))
		return false
	}
	return true
}
