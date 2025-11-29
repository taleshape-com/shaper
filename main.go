// SPDX-License-Identifier: MPL-2.0

package main

import (
	_ "modernc.org/sqlite"

	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"shaper/server/comms"
	"shaper/server/core"
	"shaper/server/dev"
	"shaper/server/ingest"
	"shaper/server/metrics"
	"shaper/server/snapshots"
	"shaper/server/util"
	"shaper/server/util/signals"
	"shaper/server/web"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/duckdb/duckdb-go/v2"
	"github.com/jmoiron/sqlx"
	"github.com/nrednav/cuid2"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

const APP_NAME = "shaper"

// Embedding frontend files.
// Has to happen in main package because you cannot embed files from a parent directory.
// That's also the main reason why main.go is in the root of the project.
//
//go:embed dist
var frontendFS embed.FS

// Version is set during build time via ldflags
var Version = "dev"

const USAGE = `Version: {{.Version}}

  Shaper is a minimal data platform built on top of DuckDB and NATS to create analytics dashboards and embed them into your software.

  All configuration options can be set via command line flags, environment variables or config file.
  All options are optional.

  Environment variables must be prefixed with SHAPER_ and use uppercase letters and underscores.
  For example, --nats-token turns into SHAPER_NATS_TOKEN.

  For S3 snapshots, Shaper supports AWS credential chain and auto-discovery. You can use AWS_ACCESS_KEY_ID,
  AWS_SECRET_ACCESS_KEY, and AWS_REGION environment variables, or use IAM roles, AWS credentials file,
  or other standard AWS credential methods. Command line flags can override environment variables.
  If no endpoint is specified, Shaper will automatically use AWS S3.

  The config file format is plain text, with one flag per line. The flag name and value are separated by whitespace.

  For more see: https://taleshape.com/shaper/docs

`

type Config struct {
	DeprecatedSchema           string
	LogLevel                   string
	SessionExp                 time.Duration
	InviteExp                  time.Duration
	Address                    string
	DataDir                    string
	ExecutableModTime          time.Time
	BasePath                   string
	CustomCSS                  string
	Favicon                    string
	JWTExp                     time.Duration
	NoPublicSharing            bool
	NoPasswordProtectedSharing bool
	NoTasks                    bool
	NodeIDFile                 string
	TLSDomain                  string
	TLSEmail                   string
	TLSCache                   string
	HTTPSHost                  string
	PdfDateFormat              string
	NatsServers                string
	NatsHost                   string
	NatsPort                   int
	NatsToken                  string
	NatsJSDir                  string
	NatsJSKey                  string
	NatsMaxStore               int64 // in bytes
	StateStreamName            string
	IngestStreamName           string
	ConfigKVBucketName         string
	TmpDashboardsKVBucketName  string
	TmpDashboardsTTL           time.Duration
	IngestStreamMaxAge         time.Duration
	StateStreamMaxAge          time.Duration
	IngestConsumerNameFile     string
	IngestSubjectPrefix        string
	StateSubjectPrefix         string
	TasksStreamName            string
	TasksSubjectPrefix         string
	TaskQueueConsumerName      string
	TaskResultsStreamName      string
	TaskResultsSubjectPrefix   string
	TaskResultsStreamMaxAge    time.Duration
	TaskBroadcastSubject       string
	SQLiteDB                   string
	DuckDB                     string
	DuckDBExtDir               string
	DuckDBSecretDir            string
	InitSQL                    string
	InitSQLFile                string
	SnapshotTime               string
	SnapshotS3Bucket           string
	SnapshotS3Region           string
	SnapshotS3Endpoint         string
	SnapshotS3AccessKey        string
	SnapshotS3SecretKey        string
	SnapshotStream             string
	SnapshotConsumerName       string
	SnapshotSubjectPrefix      string
	NoSnapshots                bool
	NoAutoRestore              bool
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rootCmd := buildRootCommand(ctx)

	err := rootCmd.ParseAndRun(ctx, os.Args[1:],
		ff.WithEnvVarPrefix("SHAPER"),
		ff.WithConfigFileFlag("config-file"),
		ff.WithConfigFileParser(ff.PlainParser),
	)
	if err != nil {
		// Check if this is a help-related error (help was shown, exit with code 0)
		if errors.Is(err, ff.ErrHelp) {
			os.Exit(0)
		}
		fmt.Println(err)
		os.Exit(1)
	}
}

func buildRootCommand(ctx context.Context) *ff.Command {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	rootFlags := ff.NewFlagSet(APP_NAME)
	rootCmd := &ff.Command{
		Name:      APP_NAME,
		Usage:     fmt.Sprintf("%s [FLAGS] | %s <subcommand> [FLAGS]", APP_NAME, APP_NAME),
		ShortHelp: "Shaper is a minimal data platform built on top of DuckDB and NATS to create analytics dashboards",
		Flags:     rootFlags,
	}

	// Add all server configuration flags
	help := rootFlags.Bool('h', "help", "show help")
	version := rootFlags.Bool('v', "version", "show version")
	logLevel := rootFlags.StringLong("log-level", "info", "log level: debug, info, warn, error")
	addr := rootFlags.StringLong("addr", "localhost:5454", "HTTP server address. Not used if --tls-domain is set. In that case, server is automatically listening on the ports 80 and 443.")
	dataDir := rootFlags.String('d', "dir", path.Join(homeDir, ".shaper"), "directory to store data, by default set to /data in docker container)")
	customCSS := rootFlags.StringLong("css", "", "CSS string to inject into the frontend")
	favicon := rootFlags.StringLong("favicon", "", "path to override favicon. Must end .svg or .ico")
	initSQL := rootFlags.StringLong("init-sql", "", "Execute SQL on startup. Supports environment variables in the format $VAR or ${VAR}")
	initSQLFile := rootFlags.StringLong("init-sql-file", "", "Same as init-sql but read SQL from file. Docker by default tries to read /var/lib/shaper/init.sql (default: [--dir]/init.sql)")
	snapshotS3Bucket := rootFlags.StringLong("snapshot-s3-bucket", "", "S3 bucket for snapshots (required for snapshots)")
	snapshotS3Endpoint := rootFlags.StringLong("snapshot-s3-endpoint", "", "S3 endpoint URL (optional, defaults to AWS S3 if not provided)")
	snapshotS3AccessKey := rootFlags.StringLong("snapshot-s3-access-key", "", "S3 access key (optional, can use AWS_ACCESS_KEY_ID environment variable)")
	snapshotS3SecretKey := rootFlags.StringLong("snapshot-s3-secret-key", "", "S3 secret key (optional, can use AWS_SECRET_ACCESS_KEY environment variable)")
	snapshotTime := rootFlags.StringLong("snapshot-time", "01:00", "time to run daily snapshots, format: HH:MM")
	snapshotS3Region := rootFlags.StringLong("snapshot-s3-region", "", "AWS region for S3 (optional, can use AWS_REGION environment variable)")
	noSnapshots := rootFlags.BoolLong("no-snapshots", "Disable automatic snapshots")
	noAutoRestore := rootFlags.BoolLong("no-auto-restore", "Disable automatic restore of latest snapshot on startup")
	noPublicSharing := rootFlags.BoolLong("no-public-sharing", "Disable public sharing of dashboards")
	noPasswordProtectedSharing := rootFlags.BoolLong("no-password-protected-sharing", "Disable sharing dashboards protected with a password")
	noTasks := rootFlags.BoolLong("no-tasks", "Disable task functionality")
	tlsDomain := rootFlags.StringLong("tls-domain", "", "Domain name for TLS certificate")
	tlsEmail := rootFlags.StringLong("tls-email", "", "Email address for Let's Encrypt registration (optional, used for alerting about certificate expiration)")
	tlsCache := rootFlags.StringLong("tls-cache", "", "Path to Let's Encrypt cache directory (default: [--dir]/letsencrypt-cache)")
	httpsHost := rootFlags.StringLong("https-port", "", "Overwrite https hostname to not listen on all interfaces")
	basePath := rootFlags.StringLong("basepath", "/", "Base URL path the frontend is served from. Override if you are using a reverse proxy and serve the frontend from a subpath.")
	pdfDateFormat := rootFlags.StringLong("pdf-date-format", "02.01.2006", "Date format for PDF exports, using Go time format, examples: '2006-01-02', '01/02/2006', '02.01.2006', 'Jan 2, 2006'")
	natsHost := rootFlags.StringLong("nats-host", "0.0.0.0", "NATS server host")
	natsPort := rootFlags.Int('p', "nats-port", 0, "NATS server port. If not specified, NATS will not listen on any port.")
	natsToken := rootFlags.String('t', "nats-token", "", "NATS authentication token")
	natsServers := rootFlags.StringLong("nats-servers", "", "Use external NATS servers, specify as comma separated list")
	natsMaxStore := rootFlags.StringLong("nats-max-store", "0", "Maximum storage in bytes, set to 0 for unlimited")
	natsJSKey := rootFlags.StringLong("nats-js-key", "", "JetStream encryption key")
	natsJSDir := rootFlags.StringLong("nats-dir", "", "Override JetStream storage directory (default: [--dir]/nats)")
	sqliteDB := rootFlags.StringLong("sqlite", "", "Override sqlite DB file that is used for system state (default: [--dir]/shaper_internal.sqlite)")
	duckdb := rootFlags.StringLong("duckdb", "", "Override duckdb DSN (default: [--dir]/shaper.duckdb)")
	duckdbExtDir := rootFlags.StringLong("duckdb-ext-dir", "", "Override DuckDB extension directory, by default set to /data/duckdb_extensions in docker (default: ~/.duckdb/extensions/)")
	duckdbSecretDir := rootFlags.StringLong("duckdb-secret-dir", "", "Override DuckDB secret directory (default: ~/.duckdb/stored_secrets/)")
	deprecatedSchema := rootFlags.StringLong("schema", "_shaper", "DEPRECATED: Was used for system state in DuckDB, not used in Sqlite after data is migrated")
	jwtExp := rootFlags.DurationLong("jwtexp", 15*time.Minute, "JWT expiration duration")
	sessionExp := rootFlags.DurationLong("sessionexp", 30*24*time.Hour, "Session expiration duration")
	inviteExp := rootFlags.DurationLong("inviteexp", 7*24*time.Hour, "Invite expiration duration")
	streamPrefix := rootFlags.StringLong("stream-prefix", "", "Prefix for NATS stream and KV bucket names. Must be a valid NATS subject name")
	nodeIDFile := rootFlags.StringLong("node-id-file", "", "File to store and lookup node ID (default: [--dir]/node-id.txt)")
	ingestStream := rootFlags.StringLong("ingest-stream", "shaper-ingest", "NATS stream name for ingest messages")
	stateStream := rootFlags.StringLong("state-stream", "shaper-state", "NATS stream name for state messages")
	configKVBucket := rootFlags.StringLong("config-kv-bucket", "shaper-config", "Name for NATS config KV bucket")
	tmpDashboardsKVBucket := rootFlags.StringLong("tmp-dashboards-kv-bucket", "shaper-tmp-dashboards", "Name for NATS KV bucket to store temporary dashboards")
	tmpDashboardsTTL := rootFlags.DurationLong("tmp-dashboards-ttl", 24*time.Hour, "TTL for temporary dashboards")
	tasksStream := rootFlags.StringLong("tasks-stream", "shaper-tasks", "NATS stream name for scheduled task execution")
	taskResultsStream := rootFlags.StringLong("task-results-stream", "shaper-task-results", "NATS stream name for task results")
	ingestStreamMaxAge := rootFlags.DurationLong("ingest-max-age", 0, "Maximum age of messages in the ingest stream. Set to 0 for indefinite retention")
	stateStreamMaxAge := rootFlags.DurationLong("state-max-age", 0, "Maximum age of messages in the state stream. Set to 0 for indefinite retention")
	taskResultsStreamMaxAge := rootFlags.DurationLong("task-results-max-age", 0, "Maximum age of messages in the task-results stream. Set to 0 for indefinite retention")
	ingestConsumerNameFile := rootFlags.StringLong("ingest-consumer-name-file", "", "File to store and lookup name for ingest consumer (default: [--dir]/ingest-consumer-name.txt)")
	_ = rootFlags.StringLong("state-consumer-name-file", "", "DEPRECATED: Using ephermal consumer and storing sequence in sqlite now")
	taskQueueConsumerName := rootFlags.StringLong("task-queue-consumer-name", "shaper-task-queue-consumer", "Name for the task queue consumer")
	_ = rootFlags.StringLong("task-result-consumer-name-file", "", "DEPRECATED: Now storing cursor in sqlite")
	snapshotStream := rootFlags.StringLong("snapshot-stream", "shaper-snapshots", "NATS stream name for scheduled snapshots")
	snapshotConsumerName := rootFlags.StringLong("snapshot-consumer-name", "shaper-snapshot-consumer", "Name for the snapshot consumer")
	subjectPrefix := rootFlags.StringLong("subject-prefix", "", "prefix for NATS subjects. Must be a valid NATS subject name. Should probably end with a dot.")
	ingestSubjectPrefix := rootFlags.StringLong("ingest-subject-prefix", "shaper.ingest.", "prefix for ingest NATS subjects")
	stateSubjectPrefix := rootFlags.StringLong("state-subject-prefix", "shaper.state.", "prefix for state NATS subjects")
	tasksSubjectPrefix := rootFlags.StringLong("tasks-subject-prefix", "shaper.tasks.", "prefix for tasks NATS subjects")
	taskResultsSubjectPrefix := rootFlags.StringLong("task-results-subject-prefix", "shaper.task-results.", "prefix for task-results NATS subjects")
	taskBroadcastSubject := rootFlags.StringLong("task-broadcast-subject", "shaper.task-broadcast", "subject to broadcast tasks to run on all nodes in a cluster when running manual task")
	snapshotSubjectPrefix := rootFlags.StringLong("snapshots-subject-prefix", "shaper.snapshots.", "prefix for snapshots NATS subjects")
	_ = rootFlags.StringLong("config-file", "", "path to config file")

	// Collect subcommands so we can include them in help output
	var subcommands []*ff.Command
	subcommands = append(subcommands,
		addDevSubcommand(rootCmd),
		addPullSubcommand(rootCmd),
		addDeploySubcommand(rootCmd),
	)

	// Set up the root command execution
	rootCmd.Exec = func(ctx context.Context, args []string) error {
		// Handle help and version flags
		if *help {
			usage := strings.Replace(USAGE, "{{.Version}}", Version, 1)
			fmt.Printf("%s\n", ffhelp.Flags(rootFlags, usage))
			fmt.Println("\nSUBCOMMANDS:")
			for _, cmd := range subcommands {
				fmt.Printf("  %-10s %s\n", cmd.Name, cmd.ShortHelp)
			}
			return nil
		}
		if *version {
			fmt.Printf("%s version %s\n", APP_NAME, Version)
			return nil
		}

		config := loadConfigFromFlags(
			homeDir,
			*logLevel, *addr, *dataDir, *customCSS, *favicon, *initSQL, *initSQLFile,
			*snapshotS3Bucket, *snapshotS3Endpoint, *snapshotS3AccessKey, *snapshotS3SecretKey,
			*snapshotTime, *snapshotS3Region, *noSnapshots, *noAutoRestore,
			*noPublicSharing, *noPasswordProtectedSharing, *noTasks,
			*tlsDomain, *tlsEmail, *tlsCache, *httpsHost, *basePath, *pdfDateFormat,
			*natsHost, *natsToken, *natsServers, *natsMaxStore, *natsJSKey, *natsJSDir,
			*sqliteDB, *duckdb, *duckdbExtDir, *duckdbSecretDir, *deprecatedSchema,
			*jwtExp, *sessionExp, *inviteExp,
			*streamPrefix, *nodeIDFile, *ingestStream, *stateStream, *configKVBucket,
			*tmpDashboardsKVBucket, *tmpDashboardsTTL, *tasksStream, *taskResultsStream,
			*ingestStreamMaxAge, *stateStreamMaxAge, *taskResultsStreamMaxAge,
			*ingestConsumerNameFile, *taskQueueConsumerName, *snapshotStream, *snapshotConsumerName,
			*subjectPrefix, *ingestSubjectPrefix, *stateSubjectPrefix, *tasksSubjectPrefix,
			*taskResultsSubjectPrefix, *taskBroadcastSubject, *snapshotSubjectPrefix,
			*natsPort,
		)
		signals.HandleInterrupt(Run(config))
		return nil
	}

	return rootCmd
}

func addDevSubcommand(rootCmd *ff.Command) *ff.Command {
	devFlags := ff.NewFlagSet("dev")
	help := devFlags.Bool('h', "help", "show help")
	devConfigPath := devFlags.StringLong("config", "./shaper.json", "Path to config file")
	devAuthFile := devFlags.StringLong("auth-file", ".shaper-auth", "Path to auth token file")

	usage := "watch local dashboard files and show preview"
	devCmd := &ff.Command{
		Name:      "dev",
		Usage:     "shaper dev [--config path] [--auth-file path]",
		ShortHelp: usage,
		Flags:     devFlags,
		Exec: func(ctx context.Context, args []string) error {
			if *help {
				fmt.Printf("%s\n", ffhelp.Flags(devFlags, usage))
				return nil
			}
			return runDevCommand(ctx, *devConfigPath, *devAuthFile)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, devCmd)
	return devCmd
}

func addPullSubcommand(rootCmd *ff.Command) *ff.Command {
	pullFlags := ff.NewFlagSet("pull")
	help := pullFlags.Bool('h', "help", "show help")
	pullConfigPath := pullFlags.StringLong("config", "./shaper.json", "Path to config file")
	pullAuthFile := pullFlags.StringLong("auth-file", ".shaper-auth", "Path to auth token file")

	usage := "pull dashboards from server to local files"
	pullCmd := &ff.Command{
		Name:      "pull",
		Usage:     "shaper pull [--config path] [--auth-file path]",
		ShortHelp: usage,
		Flags:     pullFlags,
		Exec: func(ctx context.Context, args []string) error {
			if *help {
				fmt.Printf("%s\n", ffhelp.Flags(pullFlags, usage))
				return nil
			}
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))
			return dev.RunPullCommand(ctx, *pullConfigPath, *pullAuthFile, logger)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, pullCmd)
	return pullCmd
}

func addDeploySubcommand(rootCmd *ff.Command) *ff.Command {
	deployFlags := ff.NewFlagSet("deploy")
	help := deployFlags.Bool('h', "help", "show help")
	deployConfigPath := deployFlags.StringLong("config", "./shaper.json", "Path to config file")

	usage := `Deploy dashboards from files using API key auth.

  Set SHAPER_DEPLOY_API_KEY to authenticate.`
	deployCmd := &ff.Command{
		Name:      "deploy",
		Usage:     "shaper deploy [--config path]",
		ShortHelp: usage,
		Flags:     deployFlags,
		Exec: func(ctx context.Context, args []string) error {
			if *help {
				fmt.Printf("%s\n", ffhelp.Flags(deployFlags, usage))
				return nil
			}
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))
			return dev.RunDeployCommand(ctx, *deployConfigPath, logger)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, deployCmd)
	return deployCmd
}

func runDevCommand(ctx context.Context, configPath, authFile string) error {
	cfg, err := dev.LoadOrPromptConfig(configPath)
	if err != nil {
		return err
	}

	watchDir, err := filepath.Abs(cfg.Directory)
	if err != nil {
		return fmt.Errorf("failed to resolve watch directory: %w", err)
	}
	if err := dev.EnsureDirExists(watchDir); err != nil {
		return err
	}

	if authFile == "" {
		authFile = ".shaper-auth"
	}
	authFilePath, err := filepath.Abs(authFile)
	if err != nil {
		return fmt.Errorf("failed to resolve auth file path: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	systemCfg, err := dev.FetchSystemConfig(ctx, cfg.URL)
	if err != nil {
		return err
	}

	authManager := dev.NewAuthManager(ctx, cfg.URL, authFilePath, systemCfg.LoginRequired, logger)
	if err := authManager.EnsureSession(); err != nil {
		return err
	}

	client, err := dev.NewAPIClient(ctx, cfg.URL, logger, authManager)
	if err != nil {
		return fmt.Errorf("failed to initialize API client: %w", err)
	}

	watcher, err := dev.Watch(dev.WatchConfig{
		WatchDirPath: watchDir,
		Client:       client,
		Logger:       logger,
		BaseURL:      cfg.URL,
	})
	if err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	logger.Info("Watching dashboards; press Ctrl+C to stop", slog.String("dir", watchDir), slog.String("url", cfg.URL))

	<-ctx.Done()
	watcher.Stop()
	logger.Info("Stopped dev watcher")
	return nil
}

func loadConfigFromFlags(
	homeDir string,
	logLevel, addr, dataDir, customCSS, favicon, initSQL, initSQLFile string,
	snapshotS3Bucket, snapshotS3Endpoint, snapshotS3AccessKey, snapshotS3SecretKey string,
	snapshotTime, snapshotS3Region string, noSnapshots, noAutoRestore bool,
	noPublicSharing, noPasswordProtectedSharing, noTasks bool,
	tlsDomain, tlsEmail, tlsCache, httpsHost, basePath, pdfDateFormat string,
	natsHost, natsToken, natsServers, natsMaxStore, natsJSKey, natsJSDir string,
	sqliteDB, duckdb, duckdbExtDir, duckdbSecretDir, deprecatedSchema string,
	jwtExp, sessionExp, inviteExp time.Duration,
	streamPrefix, nodeIDFile, ingestStream, stateStream, configKVBucket string,
	tmpDashboardsKVBucket string, tmpDashboardsTTL time.Duration,
	tasksStream, taskResultsStream string,
	ingestStreamMaxAge, stateStreamMaxAge, taskResultsStreamMaxAge time.Duration,
	ingestConsumerNameFile, taskQueueConsumerName, snapshotStream, snapshotConsumerName string,
	subjectPrefix, ingestSubjectPrefix, stateSubjectPrefix, tasksSubjectPrefix string,
	taskResultsSubjectPrefix, taskBroadcastSubject, snapshotSubjectPrefix string,
	natsPort int,
) Config {
	switch logLevel {
	case "debug", "info", "warn", "error":
		// valid
	default:
		fmt.Printf("Invalid log level '%s'. Valid options are: debug, info, warn, error\n", logLevel)
		os.Exit(1)
	}

	executableModTime, err := getExecutableModTime()
	if err != nil {
		fmt.Printf("Error getting executable modification time: %v\n", err)
		os.Exit(1)
	}

	if tlsDomain != "" {
		if addr != "localhost:5454" && addr != ":5454" {
			fmt.Println("Cannot set addr and tls-domain at the same time.")
			os.Exit(1)
		}
		if basePath != "/" {
			fmt.Println("Cannot set basepath and tls-domain at the same time.")
			os.Exit(1)
		}
	}

	tlsCacheDir := tlsCache
	if tlsCacheDir == "" {
		tlsCacheDir = path.Join(dataDir, "letsencrypt-cache")
	}

	// Parse natsMaxStore as int64
	maxStore, err := strconv.ParseInt(natsMaxStore, 10, 64)
	if err != nil {
		fmt.Printf("Invalid value for nats-max-store: %v\n", err)
		os.Exit(1)
	}

	if natsServers != "" {
		if natsJSDir != "" || natsJSKey != "" || maxStore > 0 {
			fmt.Println("when connecting to external NATS servers (nats-servers specified), nats-js-key, nats-dir and nats-max-store must not be specified")
			os.Exit(1)
		}
	}

	natsDir := path.Join(dataDir, "nats")
	if natsJSDir != "" {
		natsDir = natsJSDir
	}

	bpath := basePath
	if bpath == "" {
		bpath = "/"
	}
	if bpath[0] != '/' {
		bpath = "/" + bpath
	}
	if bpath[len(bpath)-1] != '/' {
		bpath += "/"
	}

	initSQLFilePath := path.Join(dataDir, "init.sql")
	if initSQLFile != "" {
		initSQLFilePath = initSQLFile
	}

	return Config{
		DeprecatedSchema:           deprecatedSchema,
		LogLevel:                   logLevel,
		Address:                    addr,
		DataDir:                    dataDir,
		ExecutableModTime:          executableModTime,
		BasePath:                   bpath,
		CustomCSS:                  customCSS,
		Favicon:                    favicon,
		JWTExp:                     jwtExp,
		SessionExp:                 sessionExp,
		InviteExp:                  inviteExp,
		NoPublicSharing:            noPublicSharing,
		NoPasswordProtectedSharing: noPasswordProtectedSharing,
		NoTasks:                    noTasks,
		NodeIDFile:                 nodeIDFile,
		TLSDomain:                  tlsDomain,
		TLSEmail:                   tlsEmail,
		TLSCache:                   tlsCacheDir,
		HTTPSHost:                  httpsHost,
		PdfDateFormat:              pdfDateFormat,
		NatsServers:                natsServers,
		NatsHost:                   natsHost,
		NatsPort:                   natsPort,
		NatsToken:                  natsToken,
		NatsJSDir:                  natsDir,
		NatsJSKey:                  natsJSKey,
		NatsMaxStore:               maxStore,
		StateStreamName:            streamPrefix + stateStream,
		IngestStreamName:           streamPrefix + ingestStream,
		ConfigKVBucketName:         streamPrefix + configKVBucket,
		TmpDashboardsKVBucketName:  streamPrefix + tmpDashboardsKVBucket,
		TmpDashboardsTTL:           tmpDashboardsTTL,
		IngestStreamMaxAge:         ingestStreamMaxAge,
		StateStreamMaxAge:          stateStreamMaxAge,
		IngestConsumerNameFile:     ingestConsumerNameFile,
		IngestSubjectPrefix:        subjectPrefix + ingestSubjectPrefix,
		StateSubjectPrefix:         subjectPrefix + stateSubjectPrefix,
		TasksStreamName:            streamPrefix + tasksStream,
		TasksSubjectPrefix:         subjectPrefix + tasksSubjectPrefix,
		TaskQueueConsumerName:      taskQueueConsumerName,
		TaskResultsStreamName:      streamPrefix + taskResultsStream,
		TaskResultsSubjectPrefix:   subjectPrefix + taskResultsSubjectPrefix,
		TaskResultsStreamMaxAge:    taskResultsStreamMaxAge,
		TaskBroadcastSubject:       subjectPrefix + taskBroadcastSubject,
		SQLiteDB:                   sqliteDB,
		DuckDB:                     duckdb,
		DuckDBExtDir:               duckdbExtDir,
		DuckDBSecretDir:            duckdbSecretDir,
		InitSQL:                    initSQL,
		InitSQLFile:                initSQLFilePath,
		SnapshotTime:               snapshotTime,
		SnapshotS3Bucket:           snapshotS3Bucket,
		SnapshotS3Region:           snapshotS3Region,
		SnapshotS3Endpoint:         snapshotS3Endpoint,
		SnapshotS3AccessKey:        snapshotS3AccessKey,
		SnapshotS3SecretKey:        snapshotS3SecretKey,
		SnapshotStream:             streamPrefix + snapshotStream,
		SnapshotConsumerName:       snapshotConsumerName,
		SnapshotSubjectPrefix:      subjectPrefix + snapshotSubjectPrefix,
		NoSnapshots:                noSnapshots,
		NoAutoRestore:              noAutoRestore,
	}
}

func Run(cfg Config) func(context.Context) {
	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handlerOptions := &slog.HandlerOptions{
		Level: logLevel,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, handlerOptions))

	logger.Info("Starting Shaper", slog.String("version", Version))
	logger.Info("For configuration options see --help or visit https://taleshape.com/shaper/docs for more")

	if cfg.Favicon != "" {
		logger.Info("Custom favicon: " + cfg.Favicon)
	}
	if cfg.CustomCSS != "" {
		logger.Info("Custom CSS injected into frontend")
	}

	// Make sure data directory exists
	if _, err := os.Stat(cfg.DataDir); os.IsNotExist(err) {
		err := os.Mkdir(cfg.DataDir, 0755)
		if err != nil {
			logger.Error("Failed to create data directory", slog.String("path", cfg.DataDir), slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("Created data directory", slog.Any("path", cfg.DataDir))
	}

	// connect to SQLite
	sqliteDBxFile := cfg.SQLiteDB
	if cfg.SQLiteDB == "" {
		sqliteDBxFile = path.Join(cfg.DataDir, "shaper_internal.sqlite")
	}

	duckDBFile := cfg.DuckDB
	if cfg.DuckDB == "" {
		duckDBFile = path.Join(cfg.DataDir, "shaper.duckdb")
	}

	// Attempt to restore snapshots if databases don't exist and snapshots are configured
	snapshotConfig := snapshots.Config{
		Logger:          logger,
		DuckDBExtDir:    cfg.DuckDBExtDir,
		DuckDBSecretDir: cfg.DuckDBSecretDir,
		InitSQL:         cfg.InitSQL,
		InitSQLFile:     cfg.InitSQLFile,
		S3Bucket:        cfg.SnapshotS3Bucket,
		S3Region:        cfg.SnapshotS3Region,
		S3Endpoint:      cfg.SnapshotS3Endpoint,
		S3AccessKey:     cfg.SnapshotS3AccessKey,
		S3SecretKey:     cfg.SnapshotS3SecretKey,
		EnableSnapshots: !cfg.NoSnapshots,
		EnableRestore:   !cfg.NoAutoRestore,
		Stream:          cfg.SnapshotStream,
		ConsumerName:    cfg.SnapshotConsumerName,
		SubjectPrefix:   cfg.SnapshotSubjectPrefix,
		ScheduledTime:   cfg.SnapshotTime,
	}
	if err := snapshots.RestoreLatestSnapshot(sqliteDBxFile, duckDBFile, snapshotConfig); err != nil {
		logger.Error("Failed to restore snapshots", slog.Any("error", err))
		os.Exit(1)
	}

	sqliteDbx, err := sqlx.Connect("sqlite", sqliteDBxFile)
	if err != nil {
		logger.Error("Failed to connect to SQLite", slog.String("file", sqliteDBxFile), slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("SQLite opened", slog.Any("file", sqliteDBxFile))

	// connect to duckdb
	duckDBConnector, err := duckdb.NewConnector(duckDBFile, nil)
	if err != nil {
		logger.Error("Failed to create DuckDB connector", slog.String("file", duckDBFile), slog.Any("error", err))
		os.Exit(1)
	}
	duckDbSqlDb := sql.OpenDB(duckDBConnector)
	// This is important to avoid leaking variables or temp tables/views. Must not reuse connections.
	duckDbSqlDb.SetMaxIdleConns(0)
	duckdbSqlxDb := sqlx.NewDb(duckDbSqlDb, "duckdb")
	logger.Info("DuckDB opened", slog.Any("file", duckDBFile))
	if cfg.DuckDBExtDir != "" {
		_, err := duckdbSqlxDb.Exec("SET extension_directory = ?", cfg.DuckDBExtDir)
		if err != nil {
			logger.Error("Failed to set DuckDB extension directory", slog.String("path", cfg.DuckDBExtDir), slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("Set DuckDB extension directory", slog.Any("path", cfg.DuckDBExtDir))
	}
	if cfg.DuckDBSecretDir != "" {
		_, err := duckdbSqlxDb.Exec("SET secret_directory = ?", cfg.DuckDBSecretDir)
		if err != nil {
			logger.Error("Failed to set DuckDB secret directory", slog.String("path", cfg.DuckDBSecretDir), slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("Set DuckDB secret directory", slog.Any("path", cfg.DuckDBSecretDir))
	}

	if cfg.InitSQL != "" {
		logger.Info("Executing init-sql")
		// Substitute environment variables in the SQL
		sql := os.ExpandEnv(strings.TrimSpace(util.StripSQLComments(cfg.InitSQL)))
		if sql == "" {
			logger.Info("init-sql specified but empty, skipping")
		} else {
			_, err := duckdbSqlxDb.Exec(sql)
			if err != nil {
				logger.Error("Failed to execute init-sql", slog.String("sql", sql), slog.Any("error", err))
				os.Exit(1)
			}
		}
	}
	if cfg.InitSQLFile != "" {
		logger.Info("Loading init-sql-file", slog.Any("path", cfg.InitSQLFile))
		data, err := os.ReadFile(cfg.InitSQLFile)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				logger.Info("init-sql-file does not exist, skipping", slog.Any("path", cfg.InitSQLFile))
			} else {
				logger.Error("Failed to read init-sql-file", slog.String("path", cfg.InitSQLFile), slog.Any("error", err))
				os.Exit(1)
			}
		} else {
			sql := os.ExpandEnv(strings.TrimSpace(util.StripSQLComments(string(data))))
			if len(sql) == 0 {
				logger.Info("init-sql-file is empty, skipping", slog.Any("path", cfg.InitSQLFile))
			} else {
				logger.Info("Executing init-sql-file")
				_, err = duckdbSqlxDb.Exec(sql)
				if err != nil {
					logger.Error("Failed to execute init-sql-file", slog.String("path", cfg.InitSQLFile), slog.Any("error", err))
					os.Exit(1)
				}
			}
		}
	}

	nodeID := getOrGenerateNodeID(cfg.DataDir, cfg.NodeIDFile, "node-id.txt")
	ingestConsumerName := getOrGenerateConsumerName(cfg.DataDir, cfg.IngestConsumerNameFile, "ingest-consumer-name.txt", "shaper-ingest-consumer-", nodeID)

	app, err := core.New(
		APP_NAME,
		nodeID,
		sqliteDbx,
		duckdbSqlxDb,
		cfg.DeprecatedSchema,
		logger,
		cfg.BasePath,
		cfg.JWTExp,
		cfg.SessionExp,
		cfg.InviteExp,
		cfg.NoPublicSharing,
		cfg.NoPasswordProtectedSharing,
		cfg.NoTasks,
		cfg.IngestSubjectPrefix,
		cfg.StateSubjectPrefix,
		cfg.StateStreamName,
		cfg.StateStreamMaxAge,
		cfg.ConfigKVBucketName,
		cfg.TmpDashboardsKVBucketName,
		cfg.TmpDashboardsTTL,
		cfg.TasksStreamName,
		cfg.TasksSubjectPrefix,
		cfg.TaskQueueConsumerName,
		cfg.TaskResultsStreamName,
		cfg.TaskResultsSubjectPrefix,
		cfg.TaskResultsStreamMaxAge,
		cfg.TaskBroadcastSubject,
	)
	if err != nil {
		logger.Error("Failed to create application core", slog.Any("error", err))
		os.Exit(1)
	}

	// TODO: refactor - comms should be part of core
	c, err := comms.New(comms.Config{
		Logger:              logger.WithGroup("nats"),
		Servers:             cfg.NatsServers,
		Host:                cfg.NatsHost,
		Port:                cfg.NatsPort,
		Token:               cfg.NatsToken,
		JSDir:               cfg.NatsJSDir,
		JSKey:               cfg.NatsJSKey,
		MaxStore:            cfg.NatsMaxStore,
		Sqlite:              sqliteDbx,
		IngestSubjectPrefix: cfg.IngestSubjectPrefix,
	})
	if err != nil {
		logger.Error("Failed to create NATS communication layer", slog.Any("error", err))
		os.Exit(1)
	}

	ingestConsumer, err := ingest.Start(
		cfg.IngestSubjectPrefix,
		duckDBConnector,
		duckdbSqlxDb,
		logger.WithGroup("ingest"),
		c.Conn,
		cfg.IngestStreamName,
		cfg.IngestStreamMaxAge,
		ingestConsumerName,
	)
	if err != nil {
		logger.Error("Failed to start ingest consumer", slog.Any("error", err))
		os.Exit(1)
	}

	err = app.Init(c.Conn)
	if err != nil {
		logger.Error("Failed to initialize application", slog.Any("error", err))
		os.Exit(1)
	}

	snapshotConfig.Sqlite = sqliteDbx
	snapshotConfig.DuckDB = duckdbSqlxDb
	snapshotConfig.Nats = c.Conn
	s, err := snapshots.Start(snapshotConfig)
	if err != nil {
		logger.Error("Failed to start snapshot service", slog.Any("error", err))
		os.Exit(1)
	}

	e := web.Start(
		cfg.Address,
		app,
		frontendFS,
		cfg.ExecutableModTime,
		cfg.CustomCSS,
		cfg.Favicon,
		cfg.TLSDomain,
		cfg.TLSEmail,
		cfg.TLSCache,
		cfg.HTTPSHost,
		cfg.PdfDateFormat,
	)

	metrics.Init()

	return func(ctx context.Context) {
		logger.Info("Initiating shutdown...")
		s.Stop()
		logger.Info("Stopping web server...")
		if err := e.Shutdown(ctx); err != nil {
			logger.ErrorContext(ctx, "Error stopping server", slog.Any("error", err))
		}
		logger.Info("Stopping NATS...")
		ingestConsumer.Close()
		c.Close()
		logger.Info("Closing DB connections...")
		if err := duckdbSqlxDb.Close(); err != nil {
			logger.ErrorContext(ctx, "Error closing database connection", slog.Any("error", err))
		}
	}
}

func getExecutableModTime() (time.Time, error) {
	ex, err := os.Executable()
	if err != nil {
		return time.Time{}, err
	}
	stat, err := os.Stat(ex)
	return stat.ModTime(), err
}

// Node ID is a CUID2 and it's stored in the given file.
// Binding the Node ID to the local file system means it resets when the file system is reset.
// This works well together with Docker containers.
func getOrGenerateNodeID(dataDir, nameFile, defaultFileName string) string {
	fileName := nameFile
	if fileName == "" {
		fileName = path.Join(dataDir, defaultFileName)
	}
	name := ""
	if _, err := os.Stat(fileName); err == nil {
		content, err := os.ReadFile(fileName)
		if err != nil {
			fmt.Printf("Failed to read node ID file %s: %v\n", fileName, err)
			os.Exit(1)
		}
		name = strings.TrimSpace(string(content))
	} else {
		name = cuid2.Generate()
		err := os.WriteFile(fileName, []byte(name), 0644)
		if err != nil {
			fmt.Printf("Failed to write node ID file %s: %v\n", fileName, err)
			os.Exit(1)
		}
	}
	return name
}

// Consumer name defaults to the Node ID with a prefix.
// Consumer names can also be read from a file to set them explicitly. This is for backwards compatibility from before the concept of Node IDs was introduced.
// Binding consumer names to the local file system means they reset when the file system is reset.
// This works well together with Docker containers.
func getOrGenerateConsumerName(dataDir, nameFile, defaultFileName, prefix, nodeID string) string {
	fileName := nameFile
	if fileName == "" {
		fileName = path.Join(dataDir, defaultFileName)
	}
	name := ""
	if _, err := os.Stat(fileName); err == nil {
		content, err := os.ReadFile(fileName)
		if err != nil {
			fmt.Printf("Failed to read consumer name file %s: %v\n", fileName, err)
			os.Exit(1)
		}
		name = strings.TrimSpace(string(content))
	} else {
		name = prefix + nodeID
	}
	return name
}
