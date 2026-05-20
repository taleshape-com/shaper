// SPDX-License-Identifier: MPL-2.0

package snapshots

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nrednav/cuid2"
)

const (
	SECRET_NAME                   = "_shaper_snapshot_secret"
	SNAPSHOT_SUBJECT              = "snapshot"
	SNAPSHOT_PREFIX               = "shaper-snapshots/"
	SNAPSHOT_SQLITE_FILE_PREFIX   = SNAPSHOT_PREFIX + "shaper-sqlite-"
	SNAPSHOT_DUCKDB_FOLDER_PREFIX = SNAPSHOT_PREFIX + "shaper-duckdb-"
)

// Config holds the configuration for the snapshots service
type Config struct {
	Logger          *slog.Logger
	Sqlite          *sqlx.DB
	DuckDB          *sqlx.DB
	Nats            *nats.Conn
	DuckDBExtDir    string
	DuckDBSecretDir string
	InitSQL         string
	InitSQLFile     string
	S3Bucket        string
	S3Region        string
	S3Endpoint      string
	S3AccessKey     string // Optional - if empty, will use credential chain
	S3SecretKey     string // Optional - if empty, will use credential chain
	EnableSnapshots bool
	EnableRestore   bool
	Stream          string
	ConsumerName    string
	SubjectPrefix   string
	ScheduledTime   string // Format: "HH:MM"
}

type Service struct {
	config     Config
	js         jetstream.JetStream
	consumer   jetstream.Consumer
	consumeCtx jetstream.ConsumeContext
	timer      *time.Timer
	stopChan   chan struct{}
	enabled    bool
}

func Start(config Config) (*Service, error) {
	s := &Service{
		config:   config,
		stopChan: make(chan struct{}),
		enabled:  hasConfig(config) && config.EnableSnapshots,
	}
	if !s.enabled {
		config.Logger.Info("Snapshots disabled")
		return s, nil
	}
	// Create S3 secret
	// We leave the secret in DuckDB so the user has the option to inspect and work with snapshot data
	if err := createDuckDBSecret(context.Background(), s.config.DuckDB, SECRET_NAME, s.config); err != nil {
		return s, fmt.Errorf("failed to create DuckDB S3 secret: %w", err)
	}

	js, err := jetstream.New(config.Nats)
	if err != nil {
		return s, fmt.Errorf("failed to create JetStream: %w", err)
	}
	s.js = js
	if err := s.setupStreamAndConsumer(); err != nil {
		return s, fmt.Errorf("failed to setup stream and consumer: %w", err)
	}
	s.scheduleNext()
	return s, nil
}

func hasConfig(config Config) bool {
	return config.S3Bucket != ""
}

func (s *Service) setupStreamAndConsumer() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stream, err := s.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:                 s.config.Stream,
		Subjects:             []string{s.config.SubjectPrefix + ">"},
		Storage:              jetstream.FileStorage,
		DiscardNewPerSubject: true,
		Discard:              jetstream.DiscardNew,
		MaxMsgsPerSubject:    1,
		Retention:            jetstream.WorkQueuePolicy,
	})
	if err != nil {
		return fmt.Errorf("failed to create snapshot stream: %w", err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable: s.config.ConsumerName,
	})
	if err != nil {
		return fmt.Errorf("failed to create snapshot consumer: %w", err)
	}
	s.consumer = consumer
	consumeCtx, err := consumer.Consume(s.handleSnapshot, jetstream.PullMaxMessages(1))
	if err != nil {
		return fmt.Errorf("failed to consume snapshots: %w", err)
	}
	s.consumeCtx = consumeCtx
	return nil
}

func (s *Service) scheduleNext() {
	if !s.enabled {
		return
	}
	now := time.Now()
	scheduledTime, err := time.Parse("15:04", s.config.ScheduledTime)
	if err != nil {
		s.config.Logger.Error("Invalid snapshot time format", slog.String("time", s.config.ScheduledTime), slog.Any("error", err))
		return
	}
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), scheduledTime.Hour(), scheduledTime.Minute(), 0, 0, now.Location())
	if nextRun.Before(now) {
		nextRun = nextRun.Add(24 * time.Hour) // Schedule for tomorrow
	}
	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(time.Until(nextRun), func() {
		s.triggerSnapshot("snapshot-" + nextRun.Format("2006-01-02_15-04-05"))
	})
	s.config.Logger.Info("Next snapshot scheduled", slog.Time("next_run", nextRun))
}

func (s *Service) triggerSnapshot(id string) {
	if !s.enabled {
		return
	}
	ctx := context.Background()
	subject := s.config.SubjectPrefix + SNAPSHOT_SUBJECT
	s.config.Logger.Info("Triggering snapshot", slog.String("msg_id", id))
	_, err := s.js.Publish(ctx, subject, []byte{}, jetstream.WithMsgID(id))
	if err != nil {
		if !strings.Contains(err.Error(), "code=503 err_code=10077") {
			s.config.Logger.Error("Failed to publish snapshot message", slog.Any("error", err))
		}
	}
	s.scheduleNext()
}

func (s *Service) handleSnapshot(msg jetstream.Msg) {
	startTime := time.Now()
	if err := msg.Ack(); err != nil {
		s.config.Logger.Error("Failed to ack snapshot message", slog.Any("error", err))
		return
	}
	if !s.enabled {
		s.config.Logger.Warn("Snapshots disabled")
		return
	}
	s.config.Logger.Info("Processing snapshot")
	success := s.performSnapshot()
	duration := time.Since(startTime)
	metricSnapshotTotalDuration.Observe(duration.Seconds())
	if success {
		metricSnapshotCounter.WithLabelValues("success").Inc()
		s.config.Logger.Info("Snapshot completed successfully", slog.Duration("duration", duration))
	} else {
		metricSnapshotCounter.WithLabelValues("failed").Inc()
		s.config.Logger.Error("Snapshot failed", slog.Duration("duration", duration))
	}
}

func (s *Service) performSnapshot() bool {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	ctx := context.Background()
	sqliteSuccess := s.snapshotSQLite(ctx, timestamp)
	duckdbSuccess := s.snapshotDuckDB(ctx, timestamp)
	return sqliteSuccess && duckdbSuccess
}

func (s *Service) snapshotSQLite(ctx context.Context, timestamp string) bool {
	s.config.Logger.Info("Starting SQLite snapshot")
	startTime := time.Now()
	// Create temporary file
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("shaper-sqlite-%s-%s.db", timestamp, cuid2.Generate()))
	defer os.Remove(tempFile)
	_, err := s.config.Sqlite.ExecContext(ctx, "VACUUM INTO ?", tempFile)
	if err != nil {
		s.config.Logger.Error("Failed to create SQLite snapshot", slog.Any("error", err))
		return false
	}
	// Upload to S3
	s3Key := fmt.Sprintf("%s%s.db", SNAPSHOT_SQLITE_FILE_PREFIX, timestamp)
	err = uploadFileToS3(
		ctx,
		tempFile,
		s.config.S3Bucket,
		s3Key,
		s.config.S3Endpoint,
		s.config.S3Region,
		s.config.S3AccessKey,
		s.config.S3SecretKey,
	)
	if err != nil {
		s.config.Logger.Error("Failed to upload SQLite snapshot to S3", slog.Any("error", err))
		return false
	}
	duration := time.Since(startTime)
	metricSqliteSnapshotDuration.Observe(duration.Seconds())
	s.config.Logger.Info("SQLite snapshot completed", slog.Duration("duration", duration), slog.String("s3_key", s3Key))
	return true
}

// snapshotDuckDB creates a snapshot of the DuckDB database
func (s *Service) snapshotDuckDB(ctx context.Context, timestamp string) bool {
	s.config.Logger.Info("Starting DuckDB snapshot")
	startTime := time.Now()
	s3Path := fmt.Sprintf("s3://%s/%s%s", s.config.S3Bucket, SNAPSHOT_DUCKDB_FOLDER_PREFIX, timestamp)

	// Export database to S3
	exportSQL := fmt.Sprintf("EXPORT DATABASE '%s' (FORMAT parquet, COMPRESSION zstd)", s3Path)
	_, err := s.config.DuckDB.ExecContext(ctx, exportSQL)
	if err != nil {
		s.config.Logger.Error("Failed to export DuckDB to S3", slog.Any("error", err))
		return false
	}

	duration := time.Since(startTime)
	metricDuckdbSnapshotDuration.Observe(duration.Seconds())
	s.config.Logger.Info("DuckDB snapshot completed", slog.Duration("duration", duration), slog.String("s3_path", s3Path))

	return true
}

func (s *Service) Stop() {
	if !s.enabled {
		return
	}
	s.config.Logger.Info("Stopping snapshots service")
	if s.timer != nil {
		s.timer.Stop()
	}
	if s.consumeCtx != nil {
		s.consumeCtx.Drain()
		<-s.consumeCtx.Closed()
	}
	close(s.stopChan)
	s.config.Logger.Info("Snapshots service stopped")
}

// createDuckDBSecret creates a temporary S3 secret for DuckDB operations
func createDuckDBSecret(ctx context.Context, duckDbx *sqlx.DB, secretName string, config Config) error {
	s3Path := fmt.Sprintf("s3://%s/%s", config.S3Bucket, SNAPSHOT_PREFIX)

	// Determine if we're using AWS S3
	useAWS := shouldUseAWSEndpoint(config.S3Endpoint)

	// Get region - use provided region or discover it
	region := config.S3Region
	if region == "" {
		region = discoverAWSRegion()
	}

	// For AWS S3, we don't need to specify endpoint in DuckDB secret
	// For other S3-compatible services, we need the endpoint
	var cleanEndpoint string
	var useSSL bool
	var urlStyle string

	if useAWS {
		// AWS S3 - no endpoint needed, use vhost style
		urlStyle = "vhost"
		useSSL = true
	} else {
		// Non-AWS S3-compatible service
		cleanEndpoint = strings.TrimPrefix(config.S3Endpoint, "http://")
		cleanEndpoint = strings.TrimPrefix(cleanEndpoint, "https://")
		useSSL = !strings.HasPrefix(config.S3Endpoint, "http://")
		urlStyle = "path"
	}

	// If access key and secret key are provided, create a secret with explicit credentials
	if config.S3AccessKey != "" && config.S3SecretKey != "" {
		var createSecretSQL string
		if useAWS {
			// AWS S3 with explicit credentials
			createSecretSQL = fmt.Sprintf(`
				CREATE OR REPLACE SECRET %s (
					TYPE S3,
					KEY_ID '%s',
					SECRET '%s',
					REGION '%s',
					URL_STYLE '%s',
					USE_SSL %t,
					SCOPE '%s'
				)`, secretName, config.S3AccessKey, config.S3SecretKey, region, urlStyle, useSSL, s3Path)
		} else {
			// Non-AWS with explicit credentials
			if region != "" {
				createSecretSQL = fmt.Sprintf(`
					CREATE OR REPLACE SECRET %s (
						TYPE S3,
						KEY_ID '%s',
						SECRET '%s',
						REGION '%s',
						ENDPOINT '%s',
						URL_STYLE '%s',
						USE_SSL %t,
						SCOPE '%s'
					)`, secretName, config.S3AccessKey, config.S3SecretKey, region, cleanEndpoint, urlStyle, useSSL, s3Path)
			} else {
				createSecretSQL = fmt.Sprintf(`
					CREATE OR REPLACE SECRET %s (
						TYPE S3,
						KEY_ID '%s',
						SECRET '%s',
						ENDPOINT '%s',
						URL_STYLE '%s',
						USE_SSL %t,
						SCOPE '%s'
					)`, secretName, config.S3AccessKey, config.S3SecretKey, cleanEndpoint, urlStyle, useSSL, s3Path)
			}
		}
		_, err := duckDbx.ExecContext(ctx, createSecretSQL)
		return err
	}

	// For credential chain, create a secret with PROVIDER credential_chain
	var createSecretSQL string
	if useAWS {
		// AWS S3 with credential chain
		createSecretSQL = fmt.Sprintf(`
			CREATE OR REPLACE SECRET %s (
				TYPE S3,
				PROVIDER credential_chain,
				REGION '%s',
				URL_STYLE '%s',
				USE_SSL %t,
				SCOPE '%s'
			)`, secretName, region, urlStyle, useSSL, s3Path)
	} else {
		// Non-AWS with credential chain
		if region != "" {
			createSecretSQL = fmt.Sprintf(`
				CREATE OR REPLACE SECRET %s (
					TYPE S3,
					PROVIDER credential_chain,
					REGION '%s',
					ENDPOINT '%s',
					URL_STYLE '%s',
					USE_SSL %t,
					SCOPE '%s'
				)`, secretName, region, cleanEndpoint, urlStyle, useSSL, s3Path)
		} else {
			createSecretSQL = fmt.Sprintf(`
				CREATE OR REPLACE SECRET %s (
					TYPE S3,
					PROVIDER credential_chain,
					ENDPOINT '%s',
					URL_STYLE '%s',
					USE_SSL %t,
					SCOPE '%s'
				)`, secretName, cleanEndpoint, urlStyle, useSSL, s3Path)
		}
	}
	_, err := duckDbx.ExecContext(ctx, createSecretSQL)
	return err
}

// discoverAWSRegion attempts to discover the AWS region from various sources
func discoverAWSRegion() string {
	// Check AWS_REGION environment variable first
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}

	// Check AWS_DEFAULT_REGION environment variable
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region
	}

	// Default to us-east-1 if no region is found
	return "us-east-1"
}

// getDefaultAWSEndpoint returns the default AWS S3 endpoint for a given region
func getDefaultAWSEndpoint(region string) string {
	return fmt.Sprintf("s3.%s.amazonaws.com", region)
}

// shouldUseAWSEndpoint determines if we should use AWS S3 based on the endpoint
func shouldUseAWSEndpoint(endpoint string) bool {
	return endpoint == "" || strings.Contains(endpoint, "amazonaws.com")
}
