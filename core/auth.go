package core

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"shaper/util"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/nrednav/cuid2"
	"golang.org/x/crypto/bcrypt"
)

const SESSION_TOKEN_PREFIX = "shapersession."

type Session struct {
	Hash string `db:"hash"`
	Salt string `db:"salt"`
}

type CreateSessionPayload struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Hash      string    `json:"hash"`
	Salt      string    `json:"salt"`
	Timestamp time.Time `json:"timestamp"`
}

func HandleCreateSession(app *App, data []byte) bool {
	var payload CreateSessionPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal create session payload", slog.Any("error", err))
		return false
	}

	_, err = app.db.Exec(
		`INSERT INTO `+app.Schema+`.sessions (
			id, user_id, hash, salt, created_at
		) VALUES ($1, $2, $3, $4, $5)`,
		payload.ID, payload.UserID, payload.Hash, payload.Salt, payload.Timestamp,
	)
	if err != nil {
		app.Logger.Error("failed to insert session into DB", slog.Any("error", err))
		return false
	}
	return true
}

func Login(app *App, ctx context.Context, email string, password string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	var user struct {
		ID           string `db:"id"`
		PasswordHash string `db:"password_hash"`
	}

	err := app.db.GetContext(ctx, &user,
		`SELECT id, password_hash
         FROM `+app.Schema+`.users
         WHERE deleted_at IS NULL
				 AND email = $1`, email,
	)
	if err != nil {
		return "", fmt.Errorf("error finding user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid password")
	}

	// Generate session token
	id := cuid2.Generate()
	suffix := util.GenerateRandomString(32)
	token := fmt.Sprintf("%s%s.%s", SESSION_TOKEN_PREFIX, id, suffix)

	salt := util.GenerateRandomString(32)
	mac := hmac.New(sha256.New, []byte(salt))
	mac.Write([]byte(token))
	hash := hex.EncodeToString(mac.Sum(nil))

	payload := CreateSessionPayload{
		ID:        id,
		UserID:    user.ID,
		Hash:      hash,
		Salt:      salt,
		Timestamp: time.Now(),
	}

	if err := app.SubmitState(ctx, "create_session", payload); err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return token, nil
}

func ValidateAPIKey(app *App, ctx context.Context, token string) (bool, error) {
	if !strings.HasPrefix(token, API_KEY_PREFIX) {
		return false, nil
	}

	id := GetAPIKeyID(token)
	if id == "" {
		return false, nil
	}

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

func validateSessionToken(app *App, ctx context.Context, token string) (bool, error) {
	if !strings.HasPrefix(token, SESSION_TOKEN_PREFIX) {
		return false, nil
	}

	parts := strings.Split(strings.TrimPrefix(token, SESSION_TOKEN_PREFIX), ".")
	if len(parts) != 2 {
		return false, nil
	}
	id := parts[0]

	var storedSession Session
	err := app.db.GetContext(ctx, &storedSession,
		`SELECT hash, salt FROM `+app.Schema+`.sessions WHERE id = $1`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	// Validate using HMAC with stored salt
	mac := hmac.New(sha256.New, []byte(storedSession.Salt))
	mac.Write([]byte(token))
	hash := hex.EncodeToString(mac.Sum(nil))

	return subtle.ConstantTimeCompare([]byte(hash), []byte(storedSession.Hash)) == 1, nil
}

func GetAPIKeyID(token string) string {
	parts := strings.Split(strings.TrimPrefix(token, API_KEY_PREFIX), ".")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

func ValidToken(app *App, ctx context.Context, token string) (bool, error) {
	if !app.LoginRequired && token == "" {
		return true, nil
	}
	ok, err := validateSessionToken(app, ctx, token)
	if err != nil || ok {
		return ok, err
	}
	return ValidateAPIKey(app, ctx, token)
}

func ResetJWTSecret(app *App, ctx context.Context) ([]byte, error) {
	secret := util.GenerateRandomString(64)
	b := []byte(secret)
	_, err := app.ConfigKV.Put(ctx, CONFIG_KEY_JWT_SECRET, b)
	return b, err
}

func LoadJWTSecret(app *App) error {
	// Empty secret until login is required
	if !app.LoginRequired {
		app.JWTSecret = []byte{}
		return nil
	}
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
