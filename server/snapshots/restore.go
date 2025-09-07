package snapshots

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/marcboeker/go-duckdb/v2"
	"github.com/minio/minio-go/v7"
	"github.com/nrednav/cuid2"
)

// RestoreLatestSnapsho attempts to restore the latest snapshots for SQLite and DuckDB
// if the local database files don't exist and snapshots are configured
func RestoreLatestSnapshot(sqlitePath, duckdbPath string, config Config) error {
	if !isRestoreEnabled(config) {
		return nil
	}
	ctx := context.Background()
	// Check if SQLite file exists, if not try to restore
	if _, err := os.Stat(sqlitePath); os.IsNotExist(err) {
		if err := restoreSQLiteSnapshot(ctx, sqlitePath, config); err != nil {
			return fmt.Errorf("failed to restore SQLite snapshot: %w", err)
		}
	}
	// Check if DuckDB file exists, if not try to restore
	if _, err := os.Stat(duckdbPath); os.IsNotExist(err) {
		if err := restoreDuckDBSnapshot(ctx, duckdbPath, config); err != nil {
			return fmt.Errorf("failed to restore DuckDB snapshot: %w", err)
		}
	}
	return nil
}

func isRestoreEnabled(config Config) bool {
	return config.EnableRestore &&
		config.S3Bucket != "" &&
		config.S3Endpoint != "" &&
		config.S3AccessKey != "" &&
		config.S3SecretKey != ""
}

func restoreSQLiteSnapshot(ctx context.Context, localPath string, config Config) error {
	config.Logger.Info("SQLite empty. Looking for SQLite snapshots in S3")
	minioClient, err := newMinioClient(config.S3Endpoint, config.S3Region, config.S3AccessKey, config.S3SecretKey)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}
	bucketExist, err := minioClient.BucketExists(ctx, config.S3Bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket %s: %w", config.S3Bucket, err)
	}
	if !bucketExist {
		return fmt.Errorf("bucket %s does not exist", config.S3Bucket)
	}
	latestSnapshot, err := getLatestSnapshot(ctx, minioClient, config.S3Bucket, "sqlite/")
	if err != nil {
		return fmt.Errorf("failed to find latest SQLite snapshot: %w", err)
	}
	if latestSnapshot == "" {
		config.Logger.Info("No SQLite snapshots found")
		return nil
	}
	startTime := time.Now()
	config.Logger.Info("Downloading SQLite snapshot", slog.String("snapshot", latestSnapshot))

	// Create temporary file for safe restore
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("shaper-sqlite-restore-%s.db", cuid2.Generate()))
	defer os.Remove(tempFile) // Cleanup temp file on exit

	// Download snapshot from S3
	obj, err := minioClient.GetObject(ctx, config.S3Bucket, latestSnapshot, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to download SQLite snapshot: %w", err)
	}
	defer obj.Close()

	// Create temporary file
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary SQLite file: %w", err)
	}
	defer file.Close()

	// Copy data to temporary file
	_, err = io.Copy(file, obj)
	if err != nil {
		return fmt.Errorf("failed to write SQLite snapshot to temp file: %w", err)
	}

	// Close the file before moving
	file.Close()

	// Atomically move temp file to final location
	err = os.Rename(tempFile, localPath)
	if err != nil {
		return fmt.Errorf("failed to move SQLite snapshot to final location: %w", err)
	}

	duration := time.Since(startTime)
	config.Logger.Info("SQLite snapshot restored successfully", slog.String("snapshot", latestSnapshot), slog.Duration("duration", duration))
	return nil
}

// restoreDuckDBSnapshot restores the latest DuckDB snapshot from S3
func restoreDuckDBSnapshot(ctx context.Context, duckdbPath string, config Config) error {
	config.Logger.Info("DuckDB empty. Looking for DuckDB snapshots in S3")
	minioClient, err := newMinioClient(config.S3Endpoint, config.S3Region, config.S3AccessKey, config.S3SecretKey)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}
	latestSnapshot, err := getLatestSnapshot(ctx, minioClient, config.S3Bucket, "duckdb/")
	if err != nil {
		return fmt.Errorf("failed to find latest DuckDB snapshot: %w", err)
	}
	if latestSnapshot == "" {
		config.Logger.Info("No DuckDB snapshots found")
		return nil
	}
	if hasData, err := checkDuckdbBackupHasData(ctx, minioClient, config.S3Bucket, latestSnapshot); err != nil {
		return fmt.Errorf("failed to check DuckDB backup data: %w", err)
	} else if !hasData {
		config.Logger.Info("Found DuckDB snapshot but it contains no load.sql, skipping restore", slog.String("snapshot", latestSnapshot))
		return nil
	}
	startTime := time.Now()
	config.Logger.Info("Restoring DuckDB snapshot", slog.String("snapshot", latestSnapshot))

	// Create temporary DuckDB file for safe restore
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("shaper-duckdb-restore-%s.db", cuid2.Generate()))

	// Connect to temporary DuckDB database
	tempDB, err := sqlx.Connect("duckdb", tempFile)
	if err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to connect to temporary DuckDB: %w", err)
	}

	// Extract timestamp from snapshot path for secret naming
	timestamp := strings.TrimSuffix(filepath.Base(latestSnapshot), filepath.Ext(filepath.Base(latestSnapshot)))
	timestamp = strings.ReplaceAll(timestamp, "-", "_")
	secretName := fmt.Sprintf("shaper_restore_secret_%s", timestamp)
	s3Path := fmt.Sprintf("s3://%s/%s", config.S3Bucket, latestSnapshot)

	// Create S3 secret for DuckDB
	err = createDuckDBSecret(ctx, tempDB, secretName, config, s3Path)
	if err != nil {
		os.Remove(tempFile)
		tempDB.Close()
		return fmt.Errorf("failed to create DuckDB S3 secret: %w", err)
	}

	// Import database from S3 into temporary database
	importSQL := fmt.Sprintf("IMPORT DATABASE '%s'; CHECKPOINT;", s3Path)
	_, err = tempDB.ExecContext(ctx, importSQL)
	dropDuckDBSecret(context.Background(), tempDB, secretName, config.Logger)
	if err != nil {
		tempDB.Close()
		os.Remove(tempFile)
		return fmt.Errorf("failed to import DuckDB from S3: %w", err)
	}

	// Close the temporary database connection
	tempDB.Close()

	// Atomically move temp file to final location
	err = os.Rename(tempFile, duckdbPath)
	if err != nil {
		return fmt.Errorf("failed to move DuckDB snapshot to final location: %w", err)
	}

	duration := time.Since(startTime)
	config.Logger.Info("DuckDB snapshot restored successfully", slog.String("snapshot", latestSnapshot), slog.Duration("duration", duration))
	return nil
}

// getLatestSnapshot finds the most recent snapshot in the specified S3 prefix
func getLatestSnapshot(ctx context.Context, client *minio.Client, bucket, prefix string) (string, error) {
	objectCh := client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: prefix})
	var latestKey string
	var latestTimestamp time.Time
	var found bool
	for object := range objectCh {
		if object.Err != nil {
			return "", fmt.Errorf("error listing objects: %w", object.Err)
		}

		// Extract timestamp from filename
		key := strings.TrimSuffix(object.Key, "/")
		timestamp := extractTimestampFromKey(key)
		if timestamp.IsZero() {
			continue // Skip objects without valid timestamps
		}

		if !found || timestamp.After(latestTimestamp) {
			latestKey = key
			latestTimestamp = timestamp
			found = true
		}
	}
	if !found {
		return "", nil
	}
	return latestKey, nil
}

func extractTimestampFromKey(key string) time.Time {
	filename := filepath.Base(key)
	if ext := filepath.Ext(filename); ext != "" {
		filename = strings.TrimSuffix(filename, ext)
	}
	// Look for timestamp pattern YYYY-MM-DD_HH-MM-SS at the end
	// Expected format: shaper-sqlite-2006-01-02_15-04-05 or shaper-duckdb-2006-01-02_15-04-05
	parts := strings.Split(filename, "-")
	if len(parts) >= 5 {
		// Try to parse the last 5 parts as timestamp: YYYY-MM-DD_HH-MM-SS
		timestampParts := parts[len(parts)-5:]
		timestampStr := strings.Join(timestampParts, "-")
		if timestamp, err := time.Parse("2006-01-02_15-04-05", timestampStr); err == nil {
			return timestamp
		}
	}
	return time.Time{}
}

// Check if load.sql file exists in the backup
func checkDuckdbBackupHasData(ctx context.Context, client *minio.Client, bucket, snapshotPath string) (bool, error) {
	objectCh := client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: snapshotPath + "/"})
	for object := range objectCh {
		if object.Err != nil {
			return false, fmt.Errorf("error listing objects: %w", object.Err)
		}
		if strings.HasSuffix(object.Key, "load.sql") {
			return true, nil
		}
	}
	return false, nil
}
