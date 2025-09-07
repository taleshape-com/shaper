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
	SNAPSHOT_SUBJECT = "snapshot"
)

// Config holds the configuration for the snapshots service
type Config struct {
	Logger        *slog.Logger
	Sqlite        *sqlx.DB
	DuckDB        *sqlx.DB
	Nats          *nats.Conn
	S3Bucket      string
	S3Region      string
	S3Endpoint    string
	S3AccessKey   string
	S3SecretKey   string
	Stream        string
	ConsumerName  string
	SubjectPrefix string
	ScheduledTime string // Format: "HH:MM"
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

func Start(config Config) *Service {
	s := &Service{
		config:   config,
		stopChan: make(chan struct{}),
		enabled:  isSnapshotsEnabled(config),
	}
	if !s.enabled {
		config.Logger.Info("Snapshots disabled - missing required S3 configuration")
		return s
	}
	js, err := jetstream.New(config.Nats)
	if err != nil {
		config.Logger.Error("Failed to create JetStream", slog.Any("error", err))
		return s
	}
	s.js = js
	if err := s.setupStreamAndConsumer(); err != nil {
		config.Logger.Error("Failed to setup stream and consumer", slog.Any("error", err))
		return s
	}
	s.scheduleNext()
	return s
}

func isSnapshotsEnabled(config Config) bool {
	return config.S3Bucket != "" &&
		config.S3Endpoint != "" &&
		config.S3AccessKey != "" &&
		config.S3SecretKey != ""
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
	// Use VACUUM INTO to create backup
	_, err := s.config.Sqlite.ExecContext(ctx, "VACUUM INTO ?", tempFile)
	if err != nil {
		s.config.Logger.Error("Failed to create SQLite backup", slog.Any("error", err))
		return false
	}
	// Upload to S3
	s3Key := fmt.Sprintf("sqlite/shaper-sqlite-%s.db", timestamp)
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
		s.config.Logger.Error("Failed to upload SQLite backup to S3", slog.Any("error", err))
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
	secretName := fmt.Sprintf("shaper_backup_secret_%s", strings.ReplaceAll(timestamp, "-", "_"))
	s3Path := fmt.Sprintf("s3://%s/duckdb/shaper-duckdb-%s", s.config.S3Bucket, timestamp)
	cleanEndpoint := strings.TrimPrefix(s.config.S3Endpoint, "http://")
	cleanEndpoint = strings.TrimPrefix(cleanEndpoint, "https://")
	useSSL := !strings.HasPrefix(s.config.S3Endpoint, "http://")
	// Create temporary S3 secret
	createSecretSQL := fmt.Sprintf(`
		CREATE SECRET %s (
			TYPE S3,
			KEY_ID '%s',
			SECRET '%s',
			REGION '%s',
			ENDPOINT '%s',
			URL_STYLE 'path',
			USE_SSL %t,
			SCOPE '%s'
		)`, secretName, s.config.S3AccessKey, s.config.S3SecretKey, s.config.S3Region, cleanEndpoint, useSSL, s3Path)
	_, err := s.config.DuckDB.ExecContext(ctx, createSecretSQL)
	if err != nil {
		s.config.Logger.Error("Failed to create DuckDB S3 secret", slog.Any("error", err))
		return false
	}
	// Ensure secret is cleaned up
	defer func() {
		dropSecretSQL := fmt.Sprintf("DROP SECRET IF EXISTS %s", secretName)
		_, err := s.config.DuckDB.ExecContext(context.Background(), dropSecretSQL)
		if err != nil {
			s.config.Logger.Error("Failed to drop DuckDB S3 secret", slog.Any("error", err))
		}
	}()

	// Export database to S3
	exportSQL := fmt.Sprintf("EXPORT DATABASE '%s' (FORMAT parquet, COMPRESSION zstd)", s3Path)
	_, err = s.config.DuckDB.ExecContext(ctx, exportSQL)
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
