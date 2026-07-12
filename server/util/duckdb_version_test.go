// SPDX-License-Identifier: MPL-2.0

package util

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"golang.org/x/mod/semver"
)

type MotherDuckVersions struct {
	DuckDB struct {
		MotherDuckRegions map[string]struct {
			Min string `json:"min"`
			Max string `json:"max"`
		} `json:"motherduck_regions"`
	} `json:"duckdb"`
}

func TestMotherDuckCompatibility(t *testing.T) {
	// 1. Get our DuckDB version
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	var ourVersion string
	err = db.QueryRow("SELECT version()").Scan(&ourVersion)
	if err != nil {
		t.Fatalf("failed to query version: %v", err)
	}

	// Ensure our version has the 'v' prefix for semver comparison
	if len(ourVersion) > 0 && ourVersion[0] != 'v' {
		ourVersion = "v" + ourVersion
	}

	t.Logf("Our DuckDB version: %s", ourVersion)

	// 2. Fetch MotherDuck supported versions
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://motherduck.com/docs/duckdb-versions.json")
	if err != nil {
		if os.Getenv("CI") == "true" {
			t.Fatalf("Failed to fetch MotherDuck versions JSON in CI: %v", err)
		}
		t.Skipf("Skipping compatibility check; failed to fetch MotherDuck versions: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if os.Getenv("CI") == "true" {
			t.Fatalf("MotherDuck versions endpoint returned status %d", resp.StatusCode)
		}
		t.Skipf("Skipping compatibility check; endpoint returned status %d", resp.StatusCode)
	}

	var mdVersions MotherDuckVersions
	if err := json.NewDecoder(resp.Body).Decode(&mdVersions); err != nil {
		t.Fatalf("failed to decode MotherDuck versions JSON: %v", err)
	}

	// Get global region compatibility range
	global, exists := mdVersions.DuckDB.MotherDuckRegions["global"]
	if !exists {
		t.Fatalf("global region not found in MotherDuck versions JSON")
	}

	minVersion := "v" + global.Min
	maxVersion := "v" + global.Max

	t.Logf("MotherDuck supported range (global): %s to %s", minVersion, maxVersion)

	// Check if our version is within range
	// semver.Compare(v, w) returns:
	//   -1 if v < w
	//    0 if v == w
	//   +1 if v > w
	if semver.Compare(ourVersion, minVersion) < 0 {
		t.Errorf("Incompatible DuckDB version: our version %s is older than the minimum supported MotherDuck version %s", ourVersion, minVersion)
	} else if semver.Compare(ourVersion, maxVersion) > 0 {
		t.Errorf("Incompatible DuckDB version: our version %s is newer than the maximum supported MotherDuck version %s", ourVersion, maxVersion)
	} else {
		t.Logf("DuckDB version %s is fully compatible with MotherDuck!", ourVersion)
	}
}
