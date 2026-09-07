// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/nrednav/cuid2"
	"golang.org/x/crypto/bcrypt"
)

var ErrUserSetupCompleted = errors.New("user setup already completed")

const setupClaimStaleTimeout = 30 * time.Second

func claimSetup(app *App, ctx context.Context) (bool, error) {
	if app.ConfigKV == nil {
		return false, nil
	}

	claimValue := []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
	_, err := app.ConfigKV.Create(ctx, CONFIG_KEY_SETUP_CLAIM, claimValue)
	if err == nil {
		return true, nil
	}

	if !errors.Is(err, jetstream.ErrKeyExists) {
		return false, fmt.Errorf("failed to claim setup: %w", err)
	}

	// Key already exists. Check if setup is already completed or if the claim is stale.
	var userCount int
	dbErr := app.Sqlite.GetContext(ctx, &userCount, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`)
	if dbErr == nil && userCount > 0 {
		return false, ErrUserSetupCompleted
	}

	entry, getErr := app.ConfigKV.Get(ctx, CONFIG_KEY_SETUP_CLAIM)
	if getErr != nil {
		if errors.Is(getErr, jetstream.ErrKeyNotFound) {
			// Key was purged between Create and Get; try claiming again
			_, retryErr := app.ConfigKV.Create(ctx, CONFIG_KEY_SETUP_CLAIM, claimValue)
			if retryErr == nil {
				return true, nil
			}
			if errors.Is(retryErr, jetstream.ErrKeyExists) {
				return false, ErrUserSetupCompleted
			}
			return false, fmt.Errorf("failed to claim setup on retry: %w", retryErr)
		}
		return false, fmt.Errorf("failed to check setup claim: %w", getErr)
	}

	if time.Since(entry.Created()) > setupClaimStaleTimeout {
		if app.Logger != nil {
			app.Logger.Warn("Stale setup claim detected; purging and reclaiming", slog.Duration("age", time.Since(entry.Created())))
		}
		_ = app.ConfigKV.Purge(ctx, CONFIG_KEY_SETUP_CLAIM)
		_, retryErr := app.ConfigKV.Create(ctx, CONFIG_KEY_SETUP_CLAIM, claimValue)
		if retryErr == nil {
			return true, nil
		}
		if errors.Is(retryErr, jetstream.ErrKeyExists) {
			return false, ErrUserSetupCompleted
		}
		return false, fmt.Errorf("failed to reclaim setup: %w", retryErr)
	}

	return false, ErrUserSetupCompleted
}

type User struct {
	ID           string     `db:"id" json:"id"`
	Email        string     `db:"email" json:"email"`
	Name         string     `db:"name" json:"name"`
	PasswordHash string     `db:"password_hash" json:"-"`
	CreatedAt    time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updatedAt"`
	DeletedAt    *time.Time `db:"deleted_at" json:"deletedAt,omitempty"`
	CreatedBy    *string    `db:"created_by" json:"-"`
	UpdatedBy    *string    `db:"updated_by" json:"-"`
	DeletedBy    *string    `db:"deleted_by" json:"-"`
}

type CreateUserPayload struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"passwordHash"`
	Timestamp    time.Time `json:"timestamp"`
	CreatedBy    string    `json:"createdBy"`
}

func CreateUser(app *App, ctx context.Context, email string, password string, name string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	name = strings.TrimSpace(name)
	if name == "" {
		name = email
	}

	// Check if any users exist
	var count int
	err := app.Sqlite.GetContext(ctx, &count, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`)
	if err != nil {
		return "", fmt.Errorf("failed to check existing users: %w", err)
	}
	if count > 0 {
		return "", ErrUserSetupCompleted
	}

	claimed, err := claimSetup(app, ctx)
	if err != nil {
		return "", err
	}

	var success bool
	if claimed {
		defer func() {
			if !success {
				rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = app.ConfigKV.Purge(rollbackCtx, CONFIG_KEY_SETUP_CLAIM)
			}
		}()
	}

	// Generate password hash
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	actor := ActorFromContext(ctx)
	createdBy := ""
	if actor != nil {
		createdBy = actor.String()
	}

	id := cuid2.Generate()
	payload := CreateUserPayload{
		ID:           id,
		Email:        email,
		Name:         name,
		PasswordHash: string(passwordHash),
		Timestamp:    time.Now(),
		CreatedBy:    createdBy,
	}

	err = app.SubmitState(ctx, "create_user", payload)
	if err != nil {
		return "", err
	}

	// Verify that this user was actually created and not rejected by invariant enforcement
	var exists bool
	err = app.Sqlite.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL)`, id)
	if err != nil {
		return "", fmt.Errorf("failed to verify user creation: %w", err)
	}
	if !exists {
		return "", ErrUserSetupCompleted
	}

	success = true
	return id, nil
}

func HandleCreateUser(app *App, data []byte) bool {
	var payload CreateUserPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		if app.Logger != nil {
			app.Logger.Error("failed to unmarshal create user payload", slog.Any("error", err))
		}
		return false
	}

	tx, err := app.Sqlite.Begin()
	if err != nil {
		if app.Logger != nil {
			app.Logger.Error("failed to begin transaction", slog.Any("error", err))
		}
		return false
	}
	defer func() { _ = tx.Rollback() }()

	// Ensure database invariant: only the first active user can be created via create_user
	var existingCount int
	err = tx.QueryRow(`SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND id != $1`, payload.ID).Scan(&existingCount)
	if err != nil {
		if app.Logger != nil {
			app.Logger.Error("failed to check existing users in HandleCreateUser", slog.Any("error", err))
		}
		return false
	}
	if existingCount > 0 {
		if app.Logger != nil {
			app.Logger.Warn("ignoring create_user event because an active user already exists", slog.String("id", payload.ID))
		}
		return true
	}

	_, err = tx.Exec(
		`INSERT OR IGNORE INTO users (
			id, email, name, password_hash, created_at, updated_at, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $5, $6, $6)`,
		payload.ID, payload.Email, payload.Name, payload.PasswordHash, payload.Timestamp, payload.CreatedBy,
	)
	if err != nil {
		if app.Logger != nil {
			app.Logger.Error("failed to insert user into DB", slog.Any("error", err))
		}
		return false
	}

	err = tx.Commit()
	if err != nil {
		if app.Logger != nil {
			app.Logger.Error("failed to commit transaction", slog.Any("error", err))
		}
		return false
	}

	if !app.LoginRequired {
		app.LoginRequired = true
		err := LoadJWTSecret(app)
		if err != nil {
			if app.Logger != nil {
				app.Logger.Error("Failed to load JWT secret", slog.Any("error", err))
			}
			return false
		}
	}
	return true
}

type DeleteInvitePayload struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
	// NOTE: Not used, but might want to log this in the future
	DeletedBy string `json:"deletedBy"`
}

func DeleteInvite(app *App, ctx context.Context, code string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}
	// Check if invite exists
	var exists bool
	err := app.Sqlite.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM invites WHERE code = $1)`,
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
		DeletedBy: actor.String(),
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

	_, err = app.Sqlite.Exec(
		`DELETE FROM invites WHERE code = $1`,
		payload.Code,
	)
	if err != nil {
		app.Logger.Error("failed to delete invite from DB", slog.Any("error", err))
		return false
	}
	return true
}

type UpdateUserPasswordPayload struct {
	UserID           string    `json:"userId"`
	PasswordHash     string    `json:"passwordHash"`
	Timestamp        time.Time `json:"timestamp"`
	UpdatedBy        string    `json:"updatedBy"`
	ExcludeSessionID string    `json:"excludeSessionId,omitempty"`
}

func getUser(app *App, ctx context.Context, id string) (*User, error) {
	var user User
	err := app.Sqlite.GetContext(ctx, &user, `SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

func UpdateUserPassword(app *App, ctx context.Context, userID string, currentPassword string, newPassword string, excludeSessionID string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}

	// Get current user to check password
	user, err := getUser(app, ctx, userID)
	if err != nil {
		return err
	}

	// Validate current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return fmt.Errorf("invalid current password")
	}

	// Generate new password hash
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	payload := UpdateUserPasswordPayload{
		UserID:           userID,
		PasswordHash:     string(passwordHash),
		Timestamp:        time.Now(),
		UpdatedBy:        actor.String(),
		ExcludeSessionID: excludeSessionID,
	}

	return app.SubmitState(ctx, "update_user_password", payload)
}

func HandleUpdateUserPassword(app *App, data []byte) bool {
	var payload UpdateUserPasswordPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal update user password payload", slog.Any("error", err))
		return false
	}

	tx, err := app.Sqlite.Begin()
	if err != nil {
		app.Logger.Error("failed to begin transaction", slog.Any("error", err))
		return false
	}
	defer func() { _ = tx.Rollback() }()

	// Update password hash
	_, err = tx.Exec(
		`UPDATE users SET password_hash = $1, updated_at = $2, updated_by = $3 WHERE id = $4`,
		payload.PasswordHash, payload.Timestamp, payload.UpdatedBy, payload.UserID,
	)
	if err != nil {
		app.Logger.Error("failed to update user password in DB", slog.Any("error", err))
		return false
	}

	// Invalidate other sessions for this user
	query := `DELETE FROM sessions WHERE user_id = $1`
	args := []any{payload.UserID}
	if payload.ExcludeSessionID != "" {
		query += ` AND id != $2`
		args = append(args, payload.ExcludeSessionID)
	}
	_, err = tx.Exec(query, args...)
	if err != nil {
		app.Logger.Error("failed to delete user sessions", slog.Any("error", err))
		return false
	}

	err = tx.Commit()
	if err != nil {
		app.Logger.Error("failed to commit transaction", slog.Any("error", err))
		return false
	}
	return true
}

type UpdateUserNamePayload struct {
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
	UpdatedBy string    `json:"updatedBy"`
}

func UpdateUserName(app *App, ctx context.Context, userID string, newName string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}

	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("name cannot be empty")
	}

	payload := UpdateUserNamePayload{
		UserID:    userID,
		Name:      newName,
		Timestamp: time.Now(),
		UpdatedBy: actor.String(),
	}

	return app.SubmitState(ctx, "update_user_name", payload)
}

func HandleUpdateUserName(app *App, data []byte) bool {
	var payload UpdateUserNamePayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal update user name payload", slog.Any("error", err))
		return false
	}

	_, err = app.Sqlite.Exec(
		`UPDATE users SET name = $1, updated_at = $2, updated_by = $3 WHERE id = $4`,
		payload.Name, payload.Timestamp, payload.UpdatedBy, payload.UserID,
	)
	if err != nil {
		app.Logger.Error("failed to update user name in DB", slog.Any("error", err))
		return false
	}

	return true
}

type UserList struct {
	Users                    []User   `json:"users"`
	Invites                  []Invite `json:"invites"`
	InviteValidTimeInSeconds int64    `json:"inviteValidTimeInSeconds"`
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
	err := app.Sqlite.SelectContext(ctx, &users,
		fmt.Sprintf(`SELECT *
		 FROM users
		 WHERE deleted_at IS NULL
		 ORDER BY %s %s`, orderBy, order))
	if err != nil {
		return UserList{}, fmt.Errorf("error listing users: %w", err)
	}

	// Get invites ordered by creation date
	invites := []Invite{}
	err = app.Sqlite.SelectContext(ctx, &invites,
		`SELECT code, email, created_at
		 FROM invites
		 ORDER BY created_at DESC`)
	if err != nil {
		return UserList{}, fmt.Errorf("error listing invites: %w", err)
	}

	return UserList{
		Users:                    users,
		Invites:                  invites,
		InviteValidTimeInSeconds: int64(app.InviteExp.Seconds()),
	}, nil
}

type DeleteUserPayload struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	DeletedBy string    `json:"deletedBy"`
}

func DeleteUser(app *App, ctx context.Context, id string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}
	var count int
	err := app.Sqlite.GetContext(ctx, &count, `SELECT COUNT(*) FROM users WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("failed to query user: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("user not found")
	}

	// Don't allow deleting the last active user
	err = app.Sqlite.GetContext(ctx, &count, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`)
	if err != nil {
		return fmt.Errorf("failed to check remaining users: %w", err)
	}
	if count <= 1 {
		return fmt.Errorf("cannot delete the last user")
	}

	err = app.SubmitState(ctx, "delete_user", DeleteUserPayload{
		ID:        id,
		Timestamp: time.Now(),
		DeletedBy: actor.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to submit delete user state: %w", err)
	}
	return nil
}

func HandleDeleteUser(app *App, data []byte) bool {
	var payload DeleteUserPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal delete user payload", slog.Any("error", err))
		return false
	}

	tx, err := app.Sqlite.Begin()
	if err != nil {
		app.Logger.Error("failed to begin transaction", slog.Any("error", err))
		return false
	}
	defer func() { _ = tx.Rollback() }()

	// Delete user's sessions first
	_, err = tx.Exec(
		`DELETE FROM sessions WHERE user_id = $1`,
		payload.ID,
	)
	if err != nil {
		app.Logger.Error("failed to delete user sessions", slog.Any("error", err))
		return false
	}

	// Then soft delete the user
	_, err = tx.Exec(
		`UPDATE users SET deleted_at = $1, deleted_by = $2 WHERE id = $3`,
		payload.Timestamp,
		payload.DeletedBy,
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

func isInviteExpired(createdAt time.Time, expiration time.Duration) bool {
	return time.Since(createdAt) > expiration
}

func GetInvite(app *App, ctx context.Context, code string) (*Invite, error) {
	var invite Invite
	err := app.Sqlite.GetContext(ctx, &invite,
		`SELECT code, email, created_at FROM invites WHERE code = $1`,
		code)
	if err != nil {
		return nil, fmt.Errorf("invite not found")
	}
	if isInviteExpired(invite.CreatedAt, app.InviteExp) {
		return nil, fmt.Errorf("invite has expired")
	}
	return &invite, nil
}

type CreateInvitePayload struct {
	Code      string    `json:"code"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
	CreatedBy string    `json:"createdBy"`
}

func CreateInvite(app *App, ctx context.Context, email string) (*Invite, error) {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return nil, fmt.Errorf("no actor in context")
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	// Check if email is already registered
	var existingUser bool
	err := app.Sqlite.GetContext(ctx, &existingUser,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`,
		email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser {
		return nil, fmt.Errorf("email is already registered")
	}

	// Check if there's already a pending invite
	var existingInvite bool
	err = app.Sqlite.GetContext(ctx, &existingInvite,
		`SELECT EXISTS(SELECT 1 FROM invites WHERE email = $1)`,
		email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing invite: %w", err)
	}
	if existingInvite {
		return nil, fmt.Errorf("invite already exists for this email")
	}

	code := generateInviteCode()
	var exists bool
	err = app.Sqlite.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM invites WHERE code = $1)`,
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
		CreatedBy: actor.String(),
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

	_, err = app.Sqlite.Exec(
		`INSERT OR IGNORE INTO invites (
			code, email, created_at, created_by
		) VALUES ($1, $2, $3, $4)`,
		payload.Code, payload.Email, payload.Timestamp, payload.CreatedBy,
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
			// This should never happen in practice - indicates a serious system error
			fmt.Printf("Error generating random string: %v\n", err)
			os.Exit(1)
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

func ClaimInvite(app *App, ctx context.Context, code string, name string, password string) (string, error) {
	// Get invite details
	var invite Invite
	err := app.Sqlite.GetContext(ctx, &invite,
		`SELECT * FROM invites WHERE code = $1`,
		code)
	if err != nil {
		return "", fmt.Errorf("invalid invite code")
	}
	if isInviteExpired(invite.CreatedAt, app.InviteExp) {
		return "", fmt.Errorf("invite has expired")
	}

	// Check if email is already registered
	var existingUser bool
	err = app.Sqlite.GetContext(ctx, &existingUser,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`,
		invite.Email)
	if err != nil {
		return "", fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser {
		return "", fmt.Errorf("email is already registered")
	}

	// Generate password hash
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
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
	if err != nil {
		return "", fmt.Errorf("failed to submit claim invite state: %w", err)
	}
	return userId, nil
}

func HandleClaimInvite(app *App, data []byte) bool {
	var payload ClaimInvitePayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal claim invite payload", slog.Any("error", err))
		return false
	}

	tx, err := app.Sqlite.Begin()
	if err != nil {
		app.Logger.Error("failed to begin transaction", slog.Any("error", err))
		return false
	}
	defer func() { _ = tx.Rollback() }()

	// Create the user
	_, err = tx.Exec(
		`INSERT OR IGNORE INTO users (
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
		`DELETE FROM invites WHERE code = $1`,
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
	err := app.Sqlite.SelectContext(ctx,
		&invites,
		`SELECT code, email, created_at
			 FROM invites
			 ORDER BY created_at DESC`)
	if err != nil {
		return InviteList{}, fmt.Errorf("failed to list invites: %w", err)
	}
	return InviteList{Invites: invites}, nil
}
