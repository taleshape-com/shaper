package core

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"log/slog"
	"math/rand"
	"time"
)

type SetJWTSecretPayload struct {
	Secret    string    `json:"secret"`
	TimeStamp time.Time `json:"timestamp"`
}

func ValidLogin(app *App, ctx context.Context, token string) (bool, error) {
	return subtle.ConstantTimeCompare([]byte(token), []byte(app.LoginToken)) == 1, nil
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func ResetJWTSecret(app *App, ctx context.Context) ([]byte, error) {
	secret := generateRandomString(64)
	err := app.SubmitState(ctx, "set_jwt_secret", SetJWTSecretPayload{
		Secret:    secret,
		TimeStamp: time.Now(),
	})
	return []byte(secret), err
}

func loadJWTSecret(app *App) error {
	var secret string
	err := app.db.Get(&secret, "SELECT value FROM "+app.Schema+".config WHERE key=$1", CONFIG_KEY_JWT_SECRET)
	if err == sql.ErrNoRows {
		secret, err := ResetJWTSecret(app, context.Background())
		if err != nil {
			return err
		}
		app.JWTSecret = secret
	}
	if err != nil {
		return err
	}
	app.JWTSecret = []byte(secret)
	return nil
}

func HandleSetJWTSecret(app *App, data []byte) bool {
	var payload SetJWTSecretPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal set JWT secret payload", slog.Any("error", err))
		return false
	}
	_, err = app.db.Exec(`
		INSERT INTO `+app.Schema+`.config (key, value, created_at, updated_at)
		VALUES ($1, $2, $3, $3)
		ON CONFLICT DO UPDATE SET value=EXCLUDED.value, updated_at=$3`, CONFIG_KEY_JWT_SECRET, payload.Secret, payload.TimeStamp)
	if err != nil {
		app.Logger.Error("failed to execute INSERT/UPDATE statement", slog.Any("error", err))
		return false
	}
	app.JWTSecret = []byte(payload.Secret)
	return true
}
