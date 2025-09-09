package snapshots

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"shaper/server/util"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/marcboeker/go-duckdb/v2"
	"github.com/minio/minio-go/v7"
	"github.com/nrednav/cuid2"
)

var restoreInitSQLStatements = []string{
	"INSTALL",
	"LOAD",
	"ATTACH",
}

// RestoreLatestSnapsho attempts to restore the latest snapshots for SQLite and DuckDB
// if the local database files don't exist and snapshots are configured
func RestoreLatestSnapshot(sqlitePath, duckdbPath string, config Config) error {
	if !hasConfig(config) || !config.EnableRestore {
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
	latestSnapshot, err := getLatestSnapshot(ctx, minioClient, config.S3Bucket, SNAPSHOT_SQLITE_FILE_PREFIX)
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
	latestSnapshot, err := getLatestSnapshot(ctx, minioClient, config.S3Bucket, SNAPSHOT_DUCKDB_FOLDER_PREFIX)
	if err != nil {
		return fmt.Errorf("failed to find latest DuckDB snapshot: %w", err)
	}
	if latestSnapshot == "" {
		config.Logger.Info("No DuckDB snapshots found")
		return nil
	}
	if hasData, err := checkDuckdbSnapshotHasData(ctx, minioClient, config.S3Bucket, latestSnapshot); err != nil {
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

	if config.DuckDBExtDir != "" {
		_, err := tempDB.Exec("SET extension_directory = ?", config.DuckDBExtDir)
		if err != nil {
			return fmt.Errorf("failed to set DuckDB extension directory: %w", err)
		}
	}

	if config.InitSQL != "" {
		// Substitute environment variables in the SQL
		sql, err := prepSQL(config.InitSQL)
		if err != nil {
			return fmt.Errorf("failed to prepare init-sql: %w", err)
		}
		if sql != "" {
			_, err := tempDB.Exec(sql)
			if err != nil {
				return fmt.Errorf("failed to execute init-sql: %w", err)
			}
		}
	}
	if config.InitSQLFile != "" {
		data, err := os.ReadFile(config.InitSQLFile)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to read init-sql-file: %w", err)
			}
		} else {
			sql, err := prepSQL(string(data))
			if err != nil {
				return fmt.Errorf("failed to prepare init-sql-file: %w", err)
			}
			if len(sql) != 0 {
				// Substitute environment variables in the SQL file content
				_, err = tempDB.Exec(sql)
				if err != nil {
					return fmt.Errorf("failed to execute init-sql-file: %w", err)
				}
			}
		}
	}

	// Create S3 secret
	if err := createDuckDBSecret(ctx, tempDB, SECRET_NAME, config); err != nil {
		return fmt.Errorf("failed to create DuckDB S3 secret: %w", err)
	}

	// Extract timestamp from snapshot path for secret naming
	timestamp := strings.TrimSuffix(filepath.Base(latestSnapshot), filepath.Ext(filepath.Base(latestSnapshot)))
	timestamp = strings.ReplaceAll(timestamp, "-", "_")
	s3Path := fmt.Sprintf("s3://%s/%s", config.S3Bucket, latestSnapshot)

	// Import database from S3 into temporary database
	importSQL := fmt.Sprintf("IMPORT DATABASE '%s'; CHECKPOINT;", s3Path)
	_, err = tempDB.ExecContext(ctx, importSQL)
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

// Check if load.sql file exists in the snapshot
func checkDuckdbSnapshotHasData(ctx context.Context, client *minio.Client, bucket, snapshotPath string) (bool, error) {
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

func prepSQL(sql string) (string, error) {
	sql = util.StripSQLComments(sql)
	parts, err := util.SplitSQLQueries(sql)
	if err != nil {
		return "", err
	}
	filteredSql := ""
	for _, p := range parts {
		normalized := strings.ToUpper(strings.TrimSpace(p))
		for _, stmt := range restoreInitSQLStatements {
			if strings.HasPrefix(normalized, stmt) {
				filteredSql += p + ";\n"
				break
			}
		}
	}
	sql = filteredSql
	sql = strings.TrimSpace(sql)
	sql = os.ExpandEnv(sql)
	return sql, nil
}
