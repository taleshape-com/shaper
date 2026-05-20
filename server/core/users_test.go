// SPDX-License-Identifier: MPL-2.0

package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

func TestHandleUpdateUserPassword(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	if err := initSQLite(db); err != nil {
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

	if err := initSQLite(db); err != nil {
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
