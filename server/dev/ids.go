// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"context"
	"fmt"
)

func RunIdsCommand(ctx context.Context, configPath string) error {
	cfg, err := loadOrPromptConfig(configPath)
	if err != nil {
		return err
	}

	watchDir, err := resolveAbsolutePath(cfg.Directory)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}
	if err := ensureDirExists(watchDir); err != nil {
		return err
	}

	fmt.Printf("Adding missing IDs to dashboards in %s...\n", watchDir)

	fileCount, err := ensureShaperIDsForDir(watchDir)
	if err != nil {
		return err
	}

	pluralSuffix := ""
	if fileCount != 1 {
		pluralSuffix = "s"
	}
	fmt.Printf("\nDone. Processed %d dashboard%s.\n", fileCount, pluralSuffix)

	return nil
}
