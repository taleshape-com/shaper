// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func TestActorHasPermission(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	if err := initSQLite(db); err != nil {
		t.Fatalf("failed to init sqlite: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC()

	// 1. Test User actor (always has permission)
	userActor := Actor{Type: ActorUser, ID: "user-1"}
	if !userActor.HasPermission(ctx, db, "any-permission") {
		t.Errorf("User actor should always have permission")
	}

	// 2. Test API Key with specific permissions
	apiKeyID := "key-1"
	perms := []string{PermissionReadMetrics, PermissionIngestData}
	permsJSON, _ := json.Marshal(perms)
	_, err = db.Exec(`INSERT INTO api_keys (id, hash, salt, name, permissions, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		apiKeyID, "hash", "salt", "Test Key", string(permsJSON), now, now)
	if err != nil {
		t.Fatalf("failed to insert api key: %v", err)
	}

	keyActor := Actor{Type: ActorAPIKey, ID: apiKeyID}
	if !keyActor.HasPermission(ctx, db, PermissionReadMetrics) {
		t.Errorf("API Key should have PermissionReadMetrics")
	}
	if !keyActor.HasPermission(ctx, db, PermissionIngestData) {
		t.Errorf("API Key should have PermissionIngestData")
	}
	if keyActor.HasPermission(ctx, db, PermissionDeploy) {
		t.Errorf("API Key should NOT have PermissionDeploy")
	}

	// 3. Test API Key with NO permissions (empty array)
	emptyKeyID := "key-empty"
	emptyPermsJSON, _ := json.Marshal([]string{})
	_, err = db.Exec(`INSERT INTO api_keys (id, hash, salt, name, permissions, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		emptyKeyID, "hash", "salt", "Empty Key", string(emptyPermsJSON), now, now)
	if err != nil {
		t.Fatalf("failed to insert empty api key: %v", err)
	}

	emptyKeyActor := Actor{Type: ActorAPIKey, ID: emptyKeyID}
	if emptyKeyActor.HasPermission(ctx, db, PermissionReadMetrics) {
		t.Errorf("Empty API Key should NOT have any permissions")
	}

	// 4. Test API Key with null/empty string permissions (Legacy)
	// We manually insert them without using initSQLite again to see if HasPermission handles it correctly (it should return false now)
	legacyKeyID := "key-legacy"
	_, err = db.Exec(`INSERT INTO api_keys (id, hash, salt, name, permissions, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		legacyKeyID, "hash", "salt", "Legacy Key", nil, now, now)
	if err != nil {
		t.Fatalf("failed to insert legacy api key: %v", err)
	}

	legacyKeyActor := Actor{Type: ActorAPIKey, ID: legacyKeyID}
	if legacyKeyActor.HasPermission(ctx, db, PermissionReadMetrics) {
		t.Errorf("Legacy API Key (null) should NOT have permissions in HasPermission directly (it should be migrated by initSQLite)")
	}

	// 5. Test Migration in initSQLite
	// Re-run initSQLite to trigger migration
	if err := initSQLite(db); err != nil {
		t.Fatalf("failed to re-init sqlite: %v", err)
	}

	// Now legacyKey should have all permissions
	if !legacyKeyActor.HasPermission(ctx, db, PermissionReadMetrics) {
		t.Errorf("Legacy API Key should have all permissions after migration")
	}
	if !legacyKeyActor.HasPermission(ctx, db, PermissionDeploy) {
		t.Errorf("Legacy API Key should have all permissions after migration")
	}
}

func TestListAPIKeysLegacyMigration(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	// Initial init
	if err := initSQLite(db); err != nil {
		t.Fatalf("failed to init sqlite: %v", err)
	}

	app := &App{Sqlite: db}
	ctx := context.Background()
	now := time.Now().UTC()

	// Insert a legacy key manually (bypassing CreateAPIKey which would add permissions)
	legacyKeyID := "legacy-1"
	_, err = db.Exec(`INSERT INTO api_keys (id, hash, salt, name, permissions, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		legacyKeyID, "hash", "salt", "Legacy", "", now, now)
	if err != nil {
		t.Fatalf("failed to insert legacy key: %v", err)
	}

	// List keys before migration
	// Wait, ListAPIKeys we changed to return empty list for empty/null permissions
	keys, err := ListAPIKeys(app, ctx)
	if err != nil {
		t.Fatalf("ListAPIKeys failed: %v", err)
	}
	if len(keys.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys.Keys))
	}
	if len(keys.Keys[0].PermissionsList) != 0 {
		t.Errorf("expected 0 permissions for unmigrated legacy key, got %d", len(keys.Keys[0].PermissionsList))
	}

	// Run migration
	if err := initSQLite(db); err != nil {
		t.Fatalf("initSQLite failed: %v", err)
	}

	// List keys after migration
	keys, err = ListAPIKeys(app, ctx)
	if err != nil {
		t.Fatalf("ListAPIKeys failed: %v", err)
	}
	if len(keys.Keys[0].PermissionsList) != len(AllPermissions) {
		t.Errorf("expected %d permissions after migration, got %d", len(AllPermissions), len(keys.Keys[0].PermissionsList))
	}
}
