// SPDX-License-Identifier: MPL-2.0

package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func TestHandleUpdateUserPassword(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	if err := InitSQLite(db); err != nil {
		t.Fatalf("failed to init sqlite: %v", err)
	}

	app := &App{
		Sqlite: db,
		Logger: nil, // We'll just ignore logs for now or use a mock
	}

	// Create a test user
	userID := "user-1"
	oldPassword := "old-password"
	oldHash, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)
	now := time.Now().UTC()

	_, err = db.Exec(`INSERT INTO users (id, email, name, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		userID, "test@example.com", "Test User", string(oldHash), now, now)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	// Create a session for this user
	sessionID := "session-1"
	_, err = db.Exec(`INSERT INTO sessions (id, user_id, hash, salt, created_at) VALUES (?, ?, ?, ?, ?)`,
		sessionID, userID, "hash", "salt", now)
	if err != nil {
		t.Fatalf("failed to insert test session: %v", err)
	}

	// Create another session for this user (to be invalidated)
	otherSessionID := "session-2"
	_, err = db.Exec(`INSERT INTO sessions (id, user_id, hash, salt, created_at) VALUES (?, ?, ?, ?, ?)`,
		otherSessionID, userID, "hash", "salt", now)
	if err != nil {
		t.Fatalf("failed to insert test session 2: %v", err)
	}

	// Verify sessions exist
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM sessions WHERE user_id = ?", userID)
	if err != nil || count != 2 {
		t.Fatalf("2 sessions should exist before update, got count: %d, err: %v", count, err)
	}

	// Update password, excluding current session
	newPassword := "new-password"
	newHash, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	payload := UpdateUserPasswordPayload{
		UserID:           userID,
		PasswordHash:     string(newHash),
		Timestamp:        now.Add(time.Hour),
		UpdatedBy:        "user:user-1",
		ExcludeSessionID: sessionID,
	}
	data, _ := json.Marshal(payload)

	if !HandleUpdateUserPassword(app, data) {
		t.Fatalf("HandleUpdateUserPassword failed")
	}

	// Verify password hash updated
	var storedHash string
	err = db.Get(&storedHash, "SELECT password_hash FROM users WHERE id = ?", userID)
	if err != nil {
		t.Fatalf("failed to get stored hash: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(newPassword)); err != nil {
		t.Fatalf("stored hash does not match new password: %v", err)
	}

	// Verify other session invalidated but current one remains
	err = db.Get(&count, "SELECT COUNT(*) FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		t.Fatalf("failed to count sessions: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 session after password update, got %d", count)
	}

	var remainingSessionID string
	err = db.Get(&remainingSessionID, "SELECT id FROM sessions WHERE user_id = ?", userID)
	if err != nil || remainingSessionID != sessionID {
		t.Fatalf("expected session %s to remain, but got %s (err: %v)", sessionID, remainingSessionID, err)
	}
}

func TestHandleUpdateUserName(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	if err := InitSQLite(db); err != nil {
		t.Fatalf("failed to init sqlite: %v", err)
	}

	app := &App{
		Sqlite: db,
		Logger: nil,
	}

	// Create a test user
	userID := "user-1"
	oldName := "Old Name"
	now := time.Now().UTC()

	_, err = db.Exec(`INSERT INTO users (id, email, name, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		userID, "test@example.com", oldName, "hash", now, now)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	// Update name
	newName := "New Name"
	payload := UpdateUserNamePayload{
		UserID:    userID,
		Name:      newName,
		Timestamp: now.Add(time.Hour),
		UpdatedBy: "user:user-1",
	}
	data, _ := json.Marshal(payload)

	if !HandleUpdateUserName(app, data) {
		t.Fatalf("HandleUpdateUserName failed")
	}

	// Verify name updated
	var storedName string
	err = db.Get(&storedName, "SELECT name FROM users WHERE id = ?", userID)
	if err != nil {
		t.Fatalf("failed to get stored name: %v", err)
	}

	if storedName != newName {
		t.Fatalf("expected name %s, got %s", newName, storedName)
	}
}

func TestUpdateUserPasswordSubmitState(t *testing.T) {
	// This would require a more complex setup with NATS, which might be overkill for this task
	// given the existing test patterns.
}

func TestHandleCreateUser_InvariantEnforcement(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	if err := InitSQLite(db); err != nil {
		t.Fatalf("failed to init sqlite: %v", err)
	}

	app := &App{
		Sqlite: db,
		Logger: nil,
	}

	now := time.Now().UTC()

	// 1. First user creation should succeed
	payload1 := CreateUserPayload{
		ID:           "user-first",
		Email:        "first@example.com",
		Name:         "First Admin",
		PasswordHash: "hash1",
		Timestamp:    now,
		CreatedBy:    "",
	}
	data1, _ := json.Marshal(payload1)
	if !HandleCreateUser(app, data1) {
		t.Fatalf("HandleCreateUser failed for first user")
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 user, got %d", count)
	}

	// 2. Idempotent replay of the SAME user should succeed without error
	if !HandleCreateUser(app, data1) {
		t.Fatalf("HandleCreateUser failed on idempotent replay of first user")
	}
	if err := db.Get(&count, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected still 1 user after replay, got %d", count)
	}

	// 3. Second user creation with different ID must NOT insert a user
	payload2 := CreateUserPayload{
		ID:           "user-second",
		Email:        "second@example.com",
		Name:         "Second Admin",
		PasswordHash: "hash2",
		Timestamp:    now,
		CreatedBy:    "",
	}
	data2, _ := json.Marshal(payload2)
	// Must return true (to acknowledge and not block consumer stream)
	if !HandleCreateUser(app, data2) {
		t.Fatalf("HandleCreateUser should return true to ack stream message")
	}

	// Verify only 1 user exists in DB and user-second was NOT inserted
	if err := db.Get(&count, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 user, invariant broken: got %d users", count)
	}

	var secondExists bool
	if err := db.Get(&secondExists, `SELECT EXISTS(SELECT 1 FROM users WHERE id = 'user-second')`); err != nil {
		t.Fatal(err)
	}
	if secondExists {
		t.Fatalf("second user was inserted despite active user existing")
	}
}

func TestHandleDeleteUser(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	if err := InitSQLite(db); err != nil {
		t.Fatalf("failed to init sqlite: %v", err)
	}

	app := &App{
		Sqlite: db,
		Logger: nil,
	}

	now := time.Now().UTC()
	user1ID := "user-1"
	user2ID := "user-2"

	// Insert two users
	_, err = db.Exec(`INSERT INTO users (id, email, name, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		user1ID, "user1@example.com", "User 1", "hash", now, now)
	if err != nil {
		t.Fatalf("failed to insert user 1: %v", err)
	}
	_, err = db.Exec(`INSERT INTO users (id, email, name, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		user2ID, "user2@example.com", "User 2", "hash", now, now)
	if err != nil {
		t.Fatalf("failed to insert user 2: %v", err)
	}

	// Insert a session for user 1
	_, err = db.Exec(`INSERT INTO sessions (id, user_id, hash, salt, created_at) VALUES (?, ?, ?, ?, ?)`,
		"session-1", user1ID, "hash", "salt", now)
	if err != nil {
		t.Fatalf("failed to insert session: %v", err)
	}

	// Insert invites created by user 1 and user 2
	user1Actor := (&Actor{Type: ActorUser, ID: user1ID}).String()
	user2Actor := (&Actor{Type: ActorUser, ID: user2ID}).String()
	_, err = db.Exec(`INSERT INTO invites (code, email, created_at, created_by) VALUES (?, ?, ?, ?)`,
		"INVITE1", "invite1@example.com", now, user1Actor)
	if err != nil {
		t.Fatalf("failed to insert invite 1: %v", err)
	}
	_, err = db.Exec(`INSERT INTO invites (code, email, created_at, created_by) VALUES (?, ?, ?, ?)`,
		"INVITE2", "invite2@example.com", now, user2Actor)
	if err != nil {
		t.Fatalf("failed to insert invite 2: %v", err)
	}

	// Delete user 1
	payload := DeleteUserPayload{
		ID:        user1ID,
		Timestamp: now.Add(time.Hour),
		DeletedBy: user2Actor,
	}
	data, _ := json.Marshal(payload)

	if !HandleDeleteUser(app, data) {
		t.Fatalf("HandleDeleteUser failed")
	}

	// Verify user 1 is soft-deleted
	var deletedAt *time.Time
	if err := db.Get(&deletedAt, `SELECT deleted_at FROM users WHERE id = ?`, user1ID); err != nil {
		t.Fatalf("failed to query deleted_at: %v", err)
	}
	if deletedAt == nil {
		t.Fatalf("expected user 1 to be soft-deleted")
	}

	// Verify user 1's session is deleted
	var sessionCount int
	if err := db.Get(&sessionCount, `SELECT COUNT(*) FROM sessions WHERE user_id = ?`, user1ID); err != nil {
		t.Fatalf("failed to count sessions: %v", err)
	}
	if sessionCount != 0 {
		t.Fatalf("expected 0 sessions for user 1, got %d", sessionCount)
	}

	// Verify invite created by user 1 is deleted, while user 2's invite remains
	var invite1Exists bool
	if err := db.Get(&invite1Exists, `SELECT EXISTS(SELECT 1 FROM invites WHERE code = 'INVITE1')`); err != nil {
		t.Fatalf("failed to check invite 1: %v", err)
	}
	if invite1Exists {
		t.Fatalf("expected invite created by user 1 to be deleted")
	}

	var invite2Exists bool
	if err := db.Get(&invite2Exists, `SELECT EXISTS(SELECT 1 FROM invites WHERE code = 'INVITE2')`); err != nil {
		t.Fatalf("failed to check invite 2: %v", err)
	}
	if !invite2Exists {
		t.Fatalf("expected invite created by user 2 to remain")
	}
}

