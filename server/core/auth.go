// SPDX-License-Identifier: MPL-2.0

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
	"shaper/server/util"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nrednav/cuid2"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const ACTOR_CONTEXT_KEY contextKey = "actor"

type ActorType string

const (
	ActorUser   ActorType = "user"
	ActorAPIKey ActorType = "api_key"
	ActorNoAuth ActorType = "no_auth"
	ActorTask   ActorType = "task"
)

type Actor struct {
	Type ActorType
	ID   string
}

func (a Actor) String() string {
	if a.ID == "" {
		return string(a.Type)
	}
	return fmt.Sprintf("%s:%s", a.Type, a.ID)
}

func ActorFromContext(ctx context.Context) *Actor {
	if ctx == nil {
		return nil
	}
	if actor, ok := ctx.Value(ACTOR_CONTEXT_KEY).(*Actor); ok {
		return actor
	}
	return nil
}

func ActorFromString(s string) *Actor {
	if s == "" {
		return nil
	}
	parts := strings.SplitN(s, ":", 2)
	if len(parts) == 1 {
		return &Actor{
			Type: ActorType(parts[0]),
		}
	}
	return &Actor{
		Type: ActorType(parts[0]),
		ID:   parts[1],
	}
}

func ContextWithActor(ctx context.Context, actor *Actor) context.Context {
	return context.WithValue(ctx, ACTOR_CONTEXT_KEY, actor)
}

const SESSION_TOKEN_PREFIX = "shapersession."

type Session struct {
	Hash      string    `db:"hash"`
	Salt      string    `db:"salt"`
	CreatedAt time.Time `db:"created_at"`
}

type CreateSessionPayload struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Hash      string    `json:"hash"`
	Salt      string    `json:"salt"`
	Timestamp time.Time `json:"timestamp"`
}
type DeleteSessionPayload struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
}

type AuthInfo struct {
	Valid      bool
	IsUser     bool
	UserID     string
	UserEmail  string
	UserName   string
	SessionID  string
	APIKeyID   string
	APIKeyName string
}

func deleteExpiredSessions(app *App, userID string) (int64, error) {
	result, err := app.Sqlite.Exec(
		`DELETE FROM sessions WHERE user_id = $1 AND created_at < $2`,
		userID,
		time.Now().Add(-app.SessionExp),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return result.RowsAffected()
}

func HandleDeleteSession(app *App, data []byte) bool {
	var payload DeleteSessionPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal delete session payload", slog.Any("error", err))
		return false
	}

	_, err = app.Sqlite.Exec(`DELETE FROM sessions WHERE id = $1`, payload.ID)
	if err != nil {
		app.Logger.Error("failed to delete session from DB", slog.Any("error", err))
		return false
	}
	return true
}

func HandleCreateSession(app *App, data []byte) bool {
	var payload CreateSessionPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal create session payload", slog.Any("error", err))
		return false
	}

	// Delete expired sessions before creating new one
	deletedCount, err := deleteExpiredSessions(app, payload.UserID)
	if err != nil {
		app.Logger.Error("failed to delete expired sessions", slog.Any("error", err))
	}
	if deletedCount > 0 {
		app.Logger.Info("deleted expired sessions",
			slog.String("user_id", payload.UserID),
			slog.Int64("count", deletedCount))
	}

	_, err = app.Sqlite.Exec(
		`INSERT OR IGNORE INTO sessions (
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

func Logout(app *App, ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}

	payload := DeleteSessionPayload{
		ID:        sessionID,
		Timestamp: time.Now(),
	}

	return app.SubmitState(ctx, "delete_session", payload)
}

func Login(app *App, ctx context.Context, email string, password string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	var user struct {
		ID           string `db:"id"`
		PasswordHash string `db:"password_hash"`
	}

	err := app.Sqlite.GetContext(ctx, &user,
		`SELECT id, password_hash
		 FROM users
		 WHERE deleted_at IS NULL
		 AND email = $1`,
		email,
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

// ValidateAPIKey checks if an API key is valid by querying the database
func ValidateAPIKey(sdb *sqlx.DB, ctx context.Context, token string) (bool, error) {
	if !strings.HasPrefix(token, API_KEY_PREFIX) {
		return false, nil
	}

	id := GetAPIKeyID(token)
	if id == "" {
		return false, nil
	}

	var storedKey APIKey
	err := sdb.GetContext(ctx, &storedKey,
		`SELECT hash, salt FROM api_keys WHERE id = $1`, id)
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
	err := app.Sqlite.GetContext(ctx, &storedSession,
		`SELECT hash, salt, created_at FROM sessions WHERE id = $1`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	// Check if session has expired
	if time.Since(storedSession.CreatedAt) > app.SessionExp {
		return false, nil
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

func IsAPIKeyToken(token string) bool {
	return strings.HasPrefix(token, API_KEY_PREFIX)
}

func ValidToken(app *App, ctx context.Context, token string) (AuthInfo, error) {
	if !app.LoginRequired && token == "" {
		return AuthInfo{Valid: true}, nil
	}

	// Check session token first
	if strings.HasPrefix(token, SESSION_TOKEN_PREFIX) {
		sessionID := strings.Split(strings.TrimPrefix(token, SESSION_TOKEN_PREFIX), ".")[0]
		var user struct {
			ID    string `db:"id"`
			Email string `db:"email"`
			Name  string `db:"name"`
		}
		err := app.Sqlite.GetContext(ctx, &user,
			`SELECT u.id, u.email, u.name
			 FROM sessions s
			 JOIN users u ON s.user_id = u.id
			 WHERE s.id = $1`, sessionID)
		if err == nil {
			ok, err := validateSessionToken(app, ctx, token)
			if err != nil {
				return AuthInfo{}, err
			}
			if ok {
				return AuthInfo{
					Valid:     true,
					IsUser:    true,
					UserID:    user.ID,
					UserEmail: user.Email,
					UserName:  user.Name,
					SessionID: sessionID,
				}, nil
			}
		}
	}

	// Check API key
	if strings.HasPrefix(token, API_KEY_PREFIX) {
		id := GetAPIKeyID(token)
		var key struct {
			ID   string `db:"id"`
			Name string `db:"name"`
		}
		err := app.Sqlite.GetContext(ctx, &key,
			`SELECT id, name FROM api_keys WHERE id = $1`, id)
		if err == nil {
			ok, err := ValidateAPIKey(app.Sqlite, ctx, token)
			if err != nil {
				return AuthInfo{}, err
			}
			if ok {
				return AuthInfo{
					Valid:      true,
					IsUser:     false,
					APIKeyID:   key.ID,
					APIKeyName: key.Name,
				}, nil
			}
		}
	}

	return AuthInfo{Valid: false}, nil
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
			return fmt.Errorf("failed to reset JWT secret: %w", err)
		}
		app.JWTSecret = secret
	} else if err != nil {
		return fmt.Errorf("failed to get JWT secret from config KV: %w", err)
	} else {
		app.JWTSecret = entry.Value()
	}
	return nil
}
