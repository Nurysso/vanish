// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Dawood Khan

// Package tui manages all the tui related code
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

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
		// TODO : Add restore
	// case "restore":
	// 	return executeRestoreHeadless(filenames, cfg)
	default: // delete
		return executeDeleteHeadless(filenames, cfg)
	}
}

func executeClearHeadless(cfg types.Config) error {
	fmt.Println("Clearing cache...")

	cacheDir := helpers.ExpandPath(cfg.Cache.Directory)

	// Remove all files in cache directory
	if err := os.RemoveAll(cacheDir); err != nil {
		return fmt.Errorf("failed to remove cache directory: %w", err)
	}

	// Recreate cache directory
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create empty index
	index := types.Index{Items: []types.DeletedItem{}}
	if err := helpers.SaveIndex(index, cfg); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	// Log clear operation
	if cfg.Logging.Enabled {
		logDir := helpers.ExpandPath(cfg.Logging.Directory)
		if err := os.MkdirAll(logDir, 0755); err == nil {
			logPath := filepath.Join(logDir, "vanish.log")
			if logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				defer logFile.Close()
				logFile.WriteString(fmt.Sprintf("%s CLEAR_ALL Cache cleared\n",
					time.Now().Format("2006-01-02 15:04:05")))
			}
		}
	}

	fmt.Println("✓ Cache cleared successfully")
	return nil
}

func executePurgeHeadless(daysStr string, cfg types.Config) error {
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		return fmt.Errorf("invalid days value: %s", daysStr)
	}

	fmt.Printf("Purging files older than %d days...\n", days)

	cutoffDays := time.Duration(days) * 24 * time.Hour
	cutoff := time.Now().Add(-cutoffDays)

	index, err := helpers.LoadIndex(cfg)
	if err != nil {
		return fmt.Errorf("error loading index: %w", err)
	}

	var remainingItems []types.DeletedItem
	purgedCount := 0

	for _, item := range index.Items {
		if item.DeleteDate.Before(cutoff) {
			// Remove the actual file or directory
			if item.IsDirectory {
				os.RemoveAll(item.CachePath)
			} else {
				os.Remove(item.CachePath)
			}
			purgedCount++

			// Log purge
			if cfg.Logging.Enabled {
				helpers.LogOperation("PURGE", item, cfg)
			}
		} else {
			remainingItems = append(remainingItems, item)
		}
	}

	// Update index
	index.Items = remainingItems
	if err := helpers.SaveIndex(index, cfg); err != nil {
		return fmt.Errorf("error updating index: %w", err)
	}

	fmt.Printf("✓ Purged %d items\n", purgedCount)
	return nil
}

func executeDeleteHeadless(filenames []string, cfg types.Config) error {
	// Ensure cache directory exists
	cacheDir := helpers.ExpandPath(cfg.Cache.Directory)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

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
		// Get file info using Lstat (doesn't follow symlinks)
		stat, err := os.Lstat(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Skipping %s: %v\n", filename, err)
			continue
		}

		// Get absolute path
		absPath, err := filepath.Abs(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Skipping %s: failed to get absolute path: %v\n", filename, err)
			continue
		}

		// Generate unique ID and cache filename
		now := time.Now()
		id := fmt.Sprintf("%d", now.UnixNano())
		timestamp := now.Format("2006-01-02-15-04-05")
		baseFilename := filepath.Base(filename)
		cacheFilename := fmt.Sprintf("%s-%s-%s", id, timestamp, baseFilename)
		cachePath := filepath.Join(cacheDir, cacheFilename)

		// Determine file type
		isSymlink := stat.Mode()&os.ModeSymlink != 0
		isDir := stat.IsDir()
		fileCount := 0
		size := stat.Size()
		linkTarget := ""

		// Handle different file types
		var moveErr error
		if isSymlink {
			linkTarget, err = os.Readlink(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠ Skipping %s: failed to read symlink: %v\n", filename, err)
				continue
			}
			moveErr = helpers.MoveSymlink(filename, cachePath)
			size = stat.Size()
		} else if isDir {
			fileCount, _ = helpers.CountFilesInDirectory(filename)
			size, _ = helpers.GetDirectorySize(filename)
			moveErr = helpers.MoveDirectory(filename, cachePath)
		} else {
			moveErr = helpers.MoveFile(filename, cachePath)
		}

		if moveErr != nil {
			fmt.Fprintf(os.Stderr, "⚠ Failed to move %s: %v\n", filename, moveErr)
			continue
		}

		// Create deleted item with all metadata
		item := types.DeletedItem{
			ID:           id,
			OriginalPath: absPath,
			DeleteDate:   now,
			CachePath:    cachePath,
			IsDirectory:  isDir,
			IsSymlink:    isSymlink,
			LinkTarget:   linkTarget,
			FileCount:    fileCount,
			Size:         size,
		}

		// Update index
		if err := helpers.AddToIndex(item, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Warning: failed to update index for %s: %v\n", filename, err)
		}

		// Log the operation
		if cfg.Logging.Enabled {
			helpers.LogOperation("DELETE", item, cfg)
		}

		fmt.Println("✓ Moved to cache: %s", filename)
		movedCount++
	}

	// Cleanup old files
	fmt.Println("Cleaning up old files...")
	cutoffDays := time.Duration(cfg.Cache.Days) * 24 * time.Hour
	cutoff := time.Now().Add(-cutoffDays)

	index, err := helpers.LoadIndex(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Warning: failed to load index for cleanup: %v\n", err)
	} else {
		var remainingItems []types.DeletedItem
		cleanedCount := 0

		for _, item := range index.Items {
			if item.DeleteDate.Before(cutoff) {
				if item.IsDirectory {
					os.RemoveAll(item.CachePath)
				} else {
					os.Remove(item.CachePath)
				}
				cleanedCount++

				if cfg.Logging.Enabled {
					helpers.LogOperation("CLEANUP", item, cfg)
				}
			} else {
				remainingItems = append(remainingItems, item)
			}
		}

		if cleanedCount > 0 {
			index.Items = remainingItems
			if err := helpers.SaveIndex(index, cfg); err != nil {
				fmt.Fprintf(os.Stderr, "⚠ Warning: failed to update index: %v\n", err)
			}
			fmt.Printf("✓ Cleaned up %d old items\n", cleanedCount)
		}
	}

	fmt.Printf("✓ Successfully moved %d of %d items\n", movedCount, len(validFiles))
	return nil
}
