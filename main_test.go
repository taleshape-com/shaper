// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
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
