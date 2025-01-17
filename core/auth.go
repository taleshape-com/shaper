package core

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"shaper/util"
	"strings"

	"github.com/nats-io/nats.go/jetstream"
)

func ValidLogin(app *App, ctx context.Context, token string) (bool, error) {
	// First check if it matches token from the flag
	if subtle.ConstantTimeCompare([]byte(token), []byte(app.LoginToken)) == 1 {
		return true, nil
	}

	// Check if it's an API key
	if !strings.HasPrefix(token, API_KEY_PREFIX) {
		return false, nil
	}

	parts := strings.Split(strings.TrimPrefix(token, API_KEY_PREFIX), ".")
	if len(parts) != 2 {
		return false, nil
	}

	id := parts[0]

	var storedKey APIKey
	err := app.db.GetContext(ctx, &storedKey,
		`SELECT hash, salt FROM `+app.Schema+`.api_keys WHERE id = $1`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	// Validate using HMAC with stored salt
	mac := hmac.New(sha256.New, []byte(storedKey.Salt))
	mac.Write([]byte(token))
	hash := hex.EncodeToString(mac.Sum(nil))

	return subtle.ConstantTimeCompare([]byte(hash), []byte(storedKey.Hash)) == 1, nil
}

func ResetJWTSecret(app *App, ctx context.Context) ([]byte, error) {
	secret := util.GenerateRandomString(64)
	b := []byte(secret)
	_, err := app.ConfigKV.Put(ctx, CONFIG_KEY_JWT_SECRET, b)
	return b, err
}

func loadJWTSecret(app *App) error {
	entry, err := app.ConfigKV.Get(context.Background(), CONFIG_KEY_JWT_SECRET)
	if err == jetstream.ErrKeyNotFound {
		secret, err := ResetJWTSecret(app, context.Background())
		if err != nil {
			return err
		}
		app.JWTSecret = secret
	} else if err != nil {
		return err
	} else {
		app.JWTSecret = entry.Value()
	}
	return nil
}
