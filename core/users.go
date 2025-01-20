package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nrednav/cuid2"
	"golang.org/x/crypto/bcrypt"
)

var ErrUserSetupCompleted = errors.New("user setup already completed")

type User struct {
	ID           string     `db:"id" json:"id"`
	Email        string     `db:"email" json:"email"`
	Name         string     `db:"name" json:"name"`
	PasswordHash string     `db:"password_hash" json:"-"`
	CreatedAt    time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updatedAt"`
	DeletedAt    *time.Time `db:"deleted_at" json:"deletedAt,omitempty"`
}

type CreateUserPayload struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"passwordHash"`
	Timestamp    time.Time `json:"timestamp"`
}

func CreateUser(app *App, ctx context.Context, email string, password string, name string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	name = strings.TrimSpace(name)
	if name == "" {
		name = email
	}

	// Check if any users exist
	var count int
	err := app.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM "`+app.Schema+`".users WHERE deleted_at IS NULL`)
	if err != nil {
		return "", fmt.Errorf("failed to check existing users: %w", err)
	}
	if count > 0 {
		return "", ErrUserSetupCompleted
	}

	// Generate password hash
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	id := cuid2.Generate()
	payload := CreateUserPayload{
		ID:           id,
		Email:        email,
		Name:         name,
		PasswordHash: string(passwordHash),
		Timestamp:    time.Now(),
	}

	err = app.SubmitState(ctx, "create_user", payload)
	return id, err
}

func HandleCreateUser(app *App, data []byte) bool {
	var payload CreateUserPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal create user payload", slog.Any("error", err))
		return false
	}

	_, err = app.db.Exec(
		`INSERT INTO "`+app.Schema+`".users (
			id, email, name, password_hash, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $5)`,
		payload.ID, payload.Email, payload.Name, payload.PasswordHash, payload.Timestamp,
	)
	if err != nil {
		app.Logger.Error("failed to insert user into DB", slog.Any("error", err))
		return false
	}
	if !app.LoginRequired {
		app.LoginRequired = true
		err := LoadJWTSecret(app)
		if err != nil {
			panic(err)
		}
	}
	return true
}
