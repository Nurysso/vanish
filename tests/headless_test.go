package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"vanish/internal/helpers"
	u "vanish/internal/tui"
	"vanish/internal/types"
)

func TestExecuteHeadless(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test config
	cfg := types.Config{}
	cfg.Cache.Directory = filepath.Join(tmpDir, "cache")
	cfg.Cache.Days = 1
	cfg.Logging.Enabled = false

	// Create cache directory
	if err := os.MkdirAll(cfg.Cache.Directory, 0755); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}

	t.Run("Delete operation - moves files to cache", func(t *testing.T) {
		// Create test files
		testDir := filepath.Join(tmpDir, "test-files")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("Failed to create test dir: %v", err)
		}

		file1 := filepath.Join(testDir, "file1.txt")
		file2 := filepath.Join(testDir, "file2.txt")
		if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
			t.Fatalf("Failed to create file1: %v", err)
		}
		if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
			t.Fatalf("Failed to create file2: %v", err)
		}

		// Test delete operation
		filenames := []string{file1, file2}
		err := u.ExecuteHeadless(filenames, "delete", cfg)
		if err != nil {
			t.Fatalf("ExecuteHeadless failed: %v", err)
		}

		// Check files are moved to cache
		cacheFiles, err := os.ReadDir(cfg.Cache.Directory)
		if err != nil {
			t.Fatalf("Failed to read cache dir: %v", err)
		}

		// Should have 2 files in cache (plus maybe index file)
		foundFiles := 0
		for _, f := range cacheFiles {
			if !f.IsDir() && f.Name() != "index.json" {
				foundFiles++
			}
		}
		if foundFiles != 2 {
			t.Errorf("Expected 2 files in cache, got %d", foundFiles)
		}

		// Check original files are gone
		if _, err := os.Stat(file1); !os.IsNotExist(err) {
			t.Error("Original file1 still exists")
		}
		if _, err := os.Stat(file2); !os.IsNotExist(err) {
			t.Error("Original file2 still exists")
		}
	})

	t.Run("Delete operation - handles non-existent files", func(t *testing.T) {
		// Create one valid file
		testDir := filepath.Join(tmpDir, "test-files2")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("Failed to create test dir: %v", err)
		}

		validFile := filepath.Join(testDir, "valid.txt")
		if err := os.WriteFile(validFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create valid file: %v", err)
		}

		nonExistent := filepath.Join(testDir, "nonexistent.txt")

		filenames := []string{validFile, nonExistent}
		err := u.ExecuteHeadless(filenames, "delete", cfg)
		if err != nil {
			t.Fatalf("ExecuteHeadless failed: %v", err)
		}

		// Should have moved the valid file
		cacheFiles, err := os.ReadDir(cfg.Cache.Directory)
		if err != nil {
			t.Fatalf("Failed to read cache dir: %v", err)
		}

		foundFiles := 0
		for _, f := range cacheFiles {
			if !f.IsDir() && f.Name() != "index.json" {
				foundFiles++
			}
		}
		if foundFiles < 1 {
			t.Error("Expected at least 1 file in cache")
		}
	})

	t.Run("Clear operation - removes all cached files", func(t *testing.T) {
		// Create a clean config with logging disabled
		clearCfg := cfg
		clearCfg.Logging.Enabled = false // Disable logging for this test

		// Create some files in cache first
		testFile := filepath.Join(clearCfg.Cache.Directory, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create index with items
		index := types.Index{
			Items: []types.DeletedItem{
				{
					ID:           "test1",
					OriginalPath: "/test/path",
					DeleteDate:   time.Now(),
					CachePath:    testFile,
				},
			},
		}
		if err := helpers.SaveIndex(index, clearCfg); err != nil {
			t.Fatalf("Failed to save index: %v", err)
		}

		err := u.ExecuteHeadless([]string{}, "clear", clearCfg)
		if err != nil {
			t.Fatalf("ExecuteHeadless clear failed: %v", err)
		}

		// Check cache directory only contains index.json
		cacheFiles, err := os.ReadDir(clearCfg.Cache.Directory)
		if err != nil {
			t.Fatalf("Failed to read cache dir: %v", err)
		}

		// The cache should only have index.json (the empty index)
		if len(cacheFiles) != 1 {
			t.Errorf("Expected 1 file (index.json) in cache, got %d files: %v", len(cacheFiles), cacheFiles)
		}

		if len(cacheFiles) > 0 && cacheFiles[0].Name() != "index.json" {
			t.Errorf("Expected index.json, got %s", cacheFiles[0].Name())
		}

		// Check index is empty
		loadedIndex, err := helpers.LoadIndex(clearCfg)
		if err != nil {
			t.Fatalf("Failed to load index: %v", err)
		}
		if len(loadedIndex.Items) != 0 {
			t.Errorf("Expected empty index, got %d items", len(loadedIndex.Items))
		}
	})

	t.Run("Purge operation - removes old files", func(t *testing.T) {
		// Create a clean cache directory for this test
		purgeCacheDir := filepath.Join(tmpDir, "purge-cache")
		if err := os.MkdirAll(purgeCacheDir, 0755); err != nil {
			t.Fatalf("Failed to create purge cache dir: %v", err)
		}

		// Create a clean config for this test
		purgeCfg := cfg
		purgeCfg.Cache.Directory = purgeCacheDir

		// Create old file (10 days old)
		oldFile := filepath.Join(purgeCacheDir, "old.txt")
		if err := os.WriteFile(oldFile, []byte("old"), 0644); err != nil {
			t.Fatalf("Failed to create old file: %v", err)
		}

		// Create new file
		newFile := filepath.Join(purgeCacheDir, "new.txt")
		if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
			t.Fatalf("Failed to create new file: %v", err)
		}

		// Create index with old and new items
		oldTime := time.Now().Add(-10 * 24 * time.Hour)
		newTime := time.Now()

		index := types.Index{
			Items: []types.DeletedItem{
				{
					ID:           "old",
					OriginalPath: "/old/path",
					DeleteDate:   oldTime,
					CachePath:    oldFile,
					IsDirectory:  false,
				},
				{
					ID:           "new",
					OriginalPath: "/new/path",
					DeleteDate:   newTime,
					CachePath:    newFile,
					IsDirectory:  false,
				},
			},
		}
		if err := helpers.SaveIndex(index, purgeCfg); err != nil {
			t.Fatalf("Failed to save index: %v", err)
		}

		// Purge files older than 7 days
		err := u.ExecuteHeadless([]string{"7"}, "purge", purgeCfg)
		if err != nil {
			t.Fatalf("ExecuteHeadless purge failed: %v", err)
		}

		// Check old file is gone
		if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
			t.Error("Old file should be purged")
		}

		// Check new file still exists
		if _, err := os.Stat(newFile); err != nil {
			t.Error("New file should still exist")
		}

		// Check index was updated
		loadedIndex, err := helpers.LoadIndex(purgeCfg)
		if err != nil {
			t.Fatalf("Failed to load index: %v", err)
		}

		if len(loadedIndex.Items) != 1 {
			t.Errorf("Expected 1 item in index, got %d", len(loadedIndex.Items))
		}

		if len(loadedIndex.Items) > 0 && loadedIndex.Items[0].ID != "new" {
			t.Error("Expected remaining item to be 'new'")
		}
	})

	t.Run("Purge operation - invalid days parameter", func(t *testing.T) {
		err := u.ExecuteHeadless([]string{"invalid"}, "purge", cfg)
		if err == nil {
			t.Error("Expected error for invalid days, got nil")
		}
	})

	t.Run("Delete operation with directory", func(t *testing.T) {
		// Create test directory with files
		testDir := filepath.Join(tmpDir, "test-dir")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("Failed to create test dir: %v", err)
		}

		subFile := filepath.Join(testDir, "subfile.txt")
		if err := os.WriteFile(subFile, []byte("subcontent"), 0644); err != nil {
			t.Fatalf("Failed to create subfile: %v", err)
		}

		err := u.ExecuteHeadless([]string{testDir}, "delete", cfg)
		if err != nil {
			t.Fatalf("ExecuteHeadless failed: %v", err)
		}

		// Check original directory is gone
		if _, err := os.Stat(testDir); !os.IsNotExist(err) {
			t.Error("Original directory should be gone")
		}

		// Check cache has the directory contents
		cacheFiles, err := os.ReadDir(cfg.Cache.Directory)
		if err != nil {
			t.Fatalf("Failed to read cache dir: %v", err)
		}

		// Should have one directory in cache (plus maybe index)
		foundDirs := 0
		for _, f := range cacheFiles {
			if f.IsDir() {
				foundDirs++
			}
		}
		if foundDirs != 1 {
			t.Errorf("Expected 1 directory in cache, got %d", foundDirs)
		}
	})

	t.Run("No valid files for deletion", func(t *testing.T) {
		err := u.ExecuteHeadless([]string{"/nonexistent/path1", "/nonexistent/path2"}, "delete", cfg)
		if err == nil {
			t.Error("Expected error when no valid files, got nil")
		}
	})
}
