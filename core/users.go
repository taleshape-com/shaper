package core

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
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
	CreatedBy    *time.Time `db:"created_by" json:"-"`
	UpdatedBy    *time.Time `db:"updated_by" json:"-"`
	DeletedBy    *time.Time `db:"deleted_by" json:"-"`
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

type DeleteInvitePayload struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

func DeleteInvite(app *App, ctx context.Context, code string) error {
	// Check if invite exists
	var exists bool
	err := app.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM "`+app.Schema+`".invites WHERE code = $1)`,
		code)
	if err != nil {
		return fmt.Errorf("failed to check invite existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("invite not found")
	}

	payload := DeleteInvitePayload{
		Code:      code,
		Timestamp: time.Now(),
	}

	return app.SubmitState(ctx, "delete_invite", payload)
}

func HandleDeleteInvite(app *App, data []byte) bool {
	var payload DeleteInvitePayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal delete invite payload", slog.Any("error", err))
		return false
	}

	_, err = app.db.Exec(
		`DELETE FROM "`+app.Schema+`".invites WHERE code = $1`,
		payload.Code,
	)
	if err != nil {
		app.Logger.Error("failed to delete invite from DB", slog.Any("error", err))
		return false
	}
	return true
}

type UserList struct {
	Users   []User   `json:"users"`
	Invites []Invite `json:"invites"`
}

func ListUsers(app *App, ctx context.Context, sort string, order string) (UserList, error) {
	var orderBy string
	switch sort {
	case "name":
		orderBy = "name"
	case "email":
		orderBy = "email"
	default:
		orderBy = "created_at"
	}

	if order != "asc" && order != "desc" {
		order = "desc"
	}

	users := []User{}
	err := app.db.SelectContext(ctx, &users,
		fmt.Sprintf(`SELECT *
		 FROM %s.users
		 WHERE deleted_at IS NULL
		 ORDER BY %s %s`, app.Schema, orderBy, order))
	if err != nil {
		return UserList{}, fmt.Errorf("error listing users: %w", err)
	}

	// Get invites ordered by creation date
	invites := []Invite{}
	err = app.db.SelectContext(ctx, &invites,
		`SELECT code, email, created_at
		 FROM "`+app.Schema+`".invites
		 ORDER BY created_at DESC`)
	if err != nil {
		return UserList{}, fmt.Errorf("error listing invites: %w", err)
	}

	return UserList{
		Users:   users,
		Invites: invites,
	}, nil
}

type DeleteUserPayload struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
}

func DeleteUser(app *App, ctx context.Context, id string) error {
	var count int
	err := app.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM "`+app.Schema+`".users WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("failed to query user: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("user not found")
	}

	// Don't allow deleting the last active user
	err = app.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM "`+app.Schema+`".users WHERE deleted_at IS NULL`)
	if err != nil {
		return fmt.Errorf("failed to check remaining users: %w", err)
	}
	if count <= 1 {
		return fmt.Errorf("cannot delete the last user")
	}

	err = app.SubmitState(ctx, "delete_user", DeleteUserPayload{
		ID:        id,
		Timestamp: time.Now(),
	})
	return err
}

func HandleDeleteUser(app *App, data []byte) bool {
	var payload DeleteUserPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal delete user payload", slog.Any("error", err))
		return false
	}

	tx, err := app.db.Begin()
	if err != nil {
		app.Logger.Error("failed to begin transaction", slog.Any("error", err))
		return false
	}
	defer tx.Rollback()

	// Delete user's sessions first
	_, err = tx.Exec(
		`DELETE FROM "`+app.Schema+`".sessions WHERE user_id = $1`,
		payload.ID,
	)
	if err != nil {
		app.Logger.Error("failed to delete user sessions", slog.Any("error", err))
		return false
	}

	// Then soft delete the user
	_, err = tx.Exec(
		`UPDATE "`+app.Schema+`".users SET deleted_at = $1 WHERE id = $2`,
		payload.Timestamp,
		payload.ID,
	)
	if err != nil {
		app.Logger.Error("failed to soft delete user", slog.Any("error", err))
		return false
	}

	err = tx.Commit()
	if err != nil {
		app.Logger.Error("failed to commit transaction", slog.Any("error", err))
		return false
	}
	return true
}

type Invite struct {
	Code      string    `db:"code" json:"code"`
	Email     string    `db:"email" json:"email"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	CreatedBy *string   `db:"created_by" json:"-"`
}

func GetInvite(app *App, ctx context.Context, code string) (*Invite, error) {
	var invite Invite
	err := app.db.GetContext(ctx, &invite,
		`SELECT code, email, created_at FROM "`+app.Schema+`".invites WHERE code = $1`,
		code)
	if err != nil {
		return nil, fmt.Errorf("invite not found")
	}
	return &invite, nil
}

type CreateInvitePayload struct {
	Code      string    `json:"code"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
}

func CreateInvite(app *App, ctx context.Context, email string) (*Invite, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	// Check if email is already registered
	var existingUser bool
	err := app.db.GetContext(ctx, &existingUser,
		`SELECT EXISTS(SELECT 1 FROM "`+app.Schema+`".users WHERE email = $1 AND deleted_at IS NULL)`,
		email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser {
		return nil, fmt.Errorf("email is already registered")
	}

	// Check if there's already a pending invite
	var existingInvite bool
	err = app.db.GetContext(ctx, &existingInvite,
		`SELECT EXISTS(SELECT 1 FROM "`+app.Schema+`".invites WHERE email = $1)`,
		email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing invite: %w", err)
	}
	if existingInvite {
		return nil, fmt.Errorf("invite already exists for this email")
	}

	code := generateInviteCode()
	var exists bool
	err = app.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM "`+app.Schema+`".invites WHERE code = $1)`,
		code)
	if err != nil {
		return nil, fmt.Errorf("failed to check invite code uniqueness: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("failed to generate unique invite code, please try again")
	}

	payload := CreateInvitePayload{
		Code:      code,
		Email:     email,
		Timestamp: time.Now(),
	}

	err = app.SubmitState(ctx, "create_invite", payload)
	if err != nil {
		return nil, err
	}

	return &Invite{
		Code:      code,
		Email:     email,
		CreatedAt: payload.Timestamp,
	}, nil
}

func HandleCreateInvite(app *App, data []byte) bool {
	var payload CreateInvitePayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal create invite payload", slog.Any("error", err))
		return false
	}

	_, err = app.db.Exec(
		`INSERT INTO "`+app.Schema+`".invites (
			code, email, created_at
		) VALUES ($1, $2, $3)`,
		payload.Code, payload.Email, payload.Timestamp,
	)
	if err != nil {
		app.Logger.Error("failed to insert invite into DB", slog.Any("error", err))
		return false
	}
	return true
}

// generateInviteCode creates a secure random 12-character invite code using
// characters that are unambiguous (no 0/O or 1/l confusion)
func generateInviteCode() string {
	const charset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	const length = 12

	b := make([]byte, length)
	for i := range b {
		// Use crypto/rand for secure random numbers
		randomNum, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(err) // This should never happen in practice
		}
		b[i] = charset[randomNum.Int64()]
	}
	return string(b)
}

type ClaimInvitePayload struct {
	Code         string    `json:"code"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"passwordHash"`
	UserId       string    `json:"userId"`
	Timestamp    time.Time `json:"timestamp"`
}

func ClaimInvite(app *App, ctx context.Context, code string, name string, password string) error {
	// Get invite details
	var invite Invite
	err := app.db.GetContext(ctx, &invite,
		`SELECT * FROM "`+app.Schema+`".invites WHERE code = $1`,
		code)
	if err != nil {
		return fmt.Errorf("invalid invite code")
	}

	// Check if email is already registered
	var existingUser bool
	err = app.db.GetContext(ctx, &existingUser,
		`SELECT EXISTS(SELECT 1 FROM "`+app.Schema+`".users WHERE email = $1 AND deleted_at IS NULL)`,
		invite.Email)
	if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser {
		return fmt.Errorf("email is already registered")
	}

	// Generate password hash
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	userId := cuid2.Generate()
	payload := ClaimInvitePayload{
		Code:         code,
		Email:        invite.Email,
		Name:         name,
		PasswordHash: string(passwordHash),
		UserId:       userId,
		Timestamp:    time.Now(),
	}

	err = app.SubmitState(ctx, "claim_invite", payload)
	return err
}

func HandleClaimInvite(app *App, data []byte) bool {
	var payload ClaimInvitePayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal claim invite payload", slog.Any("error", err))
		return false
	}

	tx, err := app.db.Begin()
	if err != nil {
		app.Logger.Error("failed to begin transaction", slog.Any("error", err))
		return false
	}
	defer tx.Rollback()

	// Create the user
	_, err = tx.Exec(
		`INSERT INTO "`+app.Schema+`".users (
			id, email, name, password_hash, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $5)`,
		payload.UserId, payload.Email, payload.Name, payload.PasswordHash, payload.Timestamp,
	)
	if err != nil {
		app.Logger.Error("failed to insert user into DB", slog.Any("error", err))
		return false
	}

	// Delete the invite
	_, err = tx.Exec(
		`DELETE FROM "`+app.Schema+`".invites WHERE code = $1`,
		payload.Code,
	)
	if err != nil {
		app.Logger.Error("failed to delete invite", slog.Any("error", err))
		return false
	}

	err = tx.Commit()
	if err != nil {
		app.Logger.Error("failed to commit transaction", slog.Any("error", err))
		return false
	}

	return true
}

type InviteList struct {
	Invites []Invite `json:"invites"`
}

func ListInvites(app *App, ctx context.Context) (InviteList, error) {
	var invites []Invite
	err := app.db.SelectContext(ctx,
		&invites,
		`SELECT code, email, created_at
			 FROM "`+app.Schema+`".invites
			 ORDER BY created_at DESC`)
	if err != nil {
		return InviteList{}, fmt.Errorf("failed to list invites: %w", err)
	}
	return InviteList{Invites: invites}, nil
}
