// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Dawood Khan

// Package tui manages all the tui related code
package tui

import (
	"fmt"
	"os"

	"vanish/internal/helpers"
	"vanish/internal/types"
)

// ExecuteHeadless performs operations without the TUI
func ExecuteHeadless(filenames []string, operation string, cfg types.Config) error {
	switch operation {
	case "clear":
		return executeClearHeadless(cfg)
	case "purge":
		if len(filenames) == 0 {
			return fmt.Errorf("purge requires number of days")
		}
		return executePurgeHeadless(filenames[0], cfg)
	case "restore":
		return executeRestoreHeadless(filenames, cfg)
	default: // delete
		return executeDeleteHeadless(filenames, cfg)
	}
}

func executeClearHeadless(cfg types.Config) error {
	fmt.Println("Clearing cache...")

	if msg := helpers.ClearAllCache(cfg)().(types.ClearMsg); msg.Err != nil {
		return msg.Err
	}

	fmt.Println("✓ Cache cleared successfully")
	return nil
}

func executePurgeHeadless(daysStr string, cfg types.Config) error {
	msg := helpers.PurgeOldFiles(cfg, daysStr)().(types.PurgeMsg)
	if msg.Err != nil {
		return msg.Err
	}

	fmt.Printf("✓ Purged %d items\n", msg.PurgedCount)
	return nil
}

func executeRestoreHeadless(patterns []string, cfg types.Config) error {
	var items []types.DeletedItem
	switch msg := helpers.CheckRestoreItems(patterns, cfg)().(type) {
	case types.ErrorMsg:
		return fmt.Errorf("%s", string(msg))
	case types.RestoreItemsMsg:
		items = msg.Items
	}

	if len(items) == 0 {
		return fmt.Errorf("no matching items found in cache for restoration")
	}

	restoredCount := 0
	for _, item := range items {
		if msg := restoreFromCache(item, cfg)().(types.RestoreMsg); msg.Err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Failed to restore %s: %v\n", item.OriginalPath, msg.Err)
			continue
		}
		fmt.Printf("✓ Restored: %s\n", item.OriginalPath)
		restoredCount++
	}

	if restoredCount < len(items) {
		return fmt.Errorf("restored only %d of %d items", restoredCount, len(items))
	}
	return nil
}

func executeDeleteHeadless(filenames []string, cfg types.Config) error {
	// Check which files exist
	var validFiles []string
	for _, filename := range filenames {
		if _, err := os.Lstat(filename); err == nil {
			validFiles = append(validFiles, filename)
		} else {
			fmt.Fprintf(os.Stderr, "⚠ Skipping %s: does not exist\n", filename)
		}
	}

	if len(validFiles) == 0 {
		return fmt.Errorf("no valid files or directories found")
	}

	fmt.Printf("Moving %d items to cache...\n", len(validFiles))

	movedCount := 0
	for _, filename := range validFiles {
		if msg := moveFileToCache(filename, cfg)().(types.FileMoveMsg); msg.Err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Failed to move %s: %v\n", filename, msg.Err)
			continue
		}
		fmt.Printf("✓ Moved to cache: %s\n", filename)
		movedCount++
	}

	// Cleanup old files
	fmt.Println("Cleaning up old files...")
	cleanedCount, err := helpers.CleanupExpired(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Warning: cleanup failed: %v\n", err)
	}
	if cleanedCount > 0 {
		fmt.Printf("✓ Cleaned up %d old items\n", cleanedCount)
	}

	if movedCount < len(validFiles) {
		return fmt.Errorf("moved only %d of %d items", movedCount, len(validFiles))
	}
	fmt.Printf("✓ Successfully moved %d of %d items\n", movedCount, len(validFiles))
	return nil
}
