// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v4"
)

func TestPreprocessArgsSubcommandFlagsBeforeSubcommand(t *testing.T) {
	ctx := context.Background()

	var schemaURL string
	var schemaConfig string

	schemaFlags := ff.NewFlagSet("schema")
	sConfig := schemaFlags.StringLong("config", "./shaper.json", "Path to config file")
	sURL := schemaFlags.StringLong("url", "", "Server URL")

	schemaCmd := &ff.Command{
		Name:  "schema",
		Flags: schemaFlags,
		Exec: func(ctx context.Context, args []string) error {
			schemaURL = *sURL
			schemaConfig = *sConfig
			return nil
		},
	}

	var previewURL string
	var previewFile string

	previewFlags := ff.NewFlagSet("preview")
	pConfig := previewFlags.StringLong("config", "./shaper.json", "Path to config file")
	pURL := previewFlags.StringLong("url", "", "Server URL")

	previewCmd := &ff.Command{
		Name:  "preview",
		Flags: previewFlags,
		Exec: func(ctx context.Context, args []string) error {
			previewURL = *pURL
			_ = *pConfig
			if len(args) == 1 {
				previewFile = args[0]
			}
			return nil
		},
	}

	rootFlags := ff.NewFlagSet("shaper")
	rLogLevel := rootFlags.StringLong("log-level", "info", "log level")

	rootCmd := &ff.Command{
		Name:        "shaper",
		Flags:       rootFlags,
		Subcommands: []*ff.Command{schemaCmd, previewCmd},
	}

	t.Run("shaper --url https://example.com schema", func(t *testing.T) {
		_ = rootCmd.Reset()
		schemaURL = ""

		inputArgs := []string{"--url", "https://example.com", "schema"}
		processedArgs := preprocessArgs(rootCmd, inputArgs)

		err := rootCmd.ParseAndRun(ctx, processedArgs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if schemaURL != "https://example.com" {
			t.Errorf("expected schemaURL 'https://example.com', got '%s'", schemaURL)
		}
	})

	t.Run("shaper --config ./myconfig.json --url https://example.com schema", func(t *testing.T) {
		_ = rootCmd.Reset()
		schemaURL = ""
		schemaConfig = ""

		inputArgs := []string{"--config", "./myconfig.json", "--url", "https://example.com", "schema"}
		processedArgs := preprocessArgs(rootCmd, inputArgs)

		err := rootCmd.ParseAndRun(ctx, processedArgs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if schemaURL != "https://example.com" {
			t.Errorf("expected schemaURL 'https://example.com', got '%s'", schemaURL)
		}
		if schemaConfig != "./myconfig.json" {
			t.Errorf("expected schemaConfig './myconfig.json', got '%s'", schemaConfig)
		}
	})

	t.Run("shaper --log-level debug --url https://example.com preview mydashboard.dashboard.sql", func(t *testing.T) {
		_ = rootCmd.Reset()
		previewURL = ""
		previewFile = ""

		inputArgs := []string{"--log-level", "debug", "--url", "https://example.com", "preview", "mydashboard.dashboard.sql"}
		processedArgs := preprocessArgs(rootCmd, inputArgs)

		err := rootCmd.ParseAndRun(ctx, processedArgs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if *rLogLevel != "debug" {
			t.Errorf("expected rLogLevel 'debug', got '%s'", *rLogLevel)
		}
		if previewURL != "https://example.com" {
			t.Errorf("expected previewURL 'https://example.com', got '%s'", previewURL)
		}
		if previewFile != "mydashboard.dashboard.sql" {
			t.Errorf("expected previewFile 'mydashboard.dashboard.sql', got '%s'", previewFile)
		}
	})
}

func TestCORSDomainsFlag(t *testing.T) {
	ctx := context.Background()
	rootCmd := buildRootCommand(ctx)

	flag, ok := rootCmd.Flags.GetFlag("cors-domains")
	if !ok {
		t.Fatal("expected --cors-domains flag to be registered on root command")
	}

	err := rootCmd.Flags.Parse([]string{"--cors-domains", "example.com,https://app.test.com"})
	if err != nil {
		t.Fatalf("unexpected error parsing --cors-domains flag: %v", err)
	}

	if flag.GetDefault() != "" {
		t.Errorf("expected default empty string, got %q", flag.GetDefault())
	}
}

func TestHealthcheckSubcommand(t *testing.T) {
	ctx := context.Background()
	rootCmd := buildRootCommand(ctx)

	var healthcheckCmd *ff.Command
	for _, sc := range rootCmd.Subcommands {
		if sc.Name == "healthcheck" {
			healthcheckCmd = sc
			break
		}
	}
	if healthcheckCmd == nil {
		t.Fatal("expected healthcheck subcommand to be registered")
	}

	for _, flagName := range []string{"url", "addr", "tls-domain", "timeout", "quiet"} {
		if _, ok := healthcheckCmd.Flags.GetFlag(flagName); !ok {
			t.Errorf("expected --%s flag on healthcheck subcommand", flagName)
		}
	}

	t.Run("preprocessArgs with healthcheck", func(t *testing.T) {
		inputArgs := []string{"--url", "http://localhost:5454", "healthcheck"}
		processedArgs := preprocessArgs(rootCmd, inputArgs)
		if len(processedArgs) < 2 || processedArgs[0] != "healthcheck" {
			t.Errorf("expected healthcheck subcommand first after preprocessing, got: %v", processedArgs)
		}
	})
}

func TestTLSFlagsDeprecated(t *testing.T) {
	ctx := context.Background()
	rootCmd := buildRootCommand(ctx)

	for _, flagName := range []string{"tls-domain", "tls-email", "tls-cache", "https-port"} {
		flag, ok := rootCmd.Flags.GetFlag(flagName)
		if !ok {
			t.Fatalf("expected --%s flag to be registered on root command", flagName)
		}
		if !strings.HasPrefix(flag.GetUsage(), "DEPRECATED:") {
			t.Errorf("expected --%s usage to start with 'DEPRECATED:', got %q", flagName, flag.GetUsage())
		}
	}
}

