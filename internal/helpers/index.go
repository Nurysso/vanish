// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Dawood Khan

// Package helpers have all the helper function.
package helpers

import (
	"encoding/json"
	// "log"
	"os"
	// "os/exec"
	"path/filepath"
	// "runtime"
	"sync"
	"vanish/internal/types"
)

// --- Index Helpers ---

// indexMu guards reads and writes to index.json. SaveIndex/LoadIndex are
// called from multiple tea.Cmds (ClearAllCache, PurgeOldFiles, AddToIndex,
// RemoveFromIndex) which may run concurrently in separate goroutines under
// bubbletea. Without this lock, two concurrent read-modify-write cycles can
// race and silently drop one side's changes.
var indexMu sync.RWMutex

// SaveIndex serializes the provided index to JSON and writes it to disk
// at the location specified by the given config. Returns an error if
// marshalling or writing to file fails.
func SaveIndex(index types.Index, config types.Config) error {
	indexMu.Lock()
	defer indexMu.Unlock()
	return saveIndexLocked(index, config)
}

// saveIndexLocked is the unexported implementation of SaveIndex that
// assumes indexMu is already held. Used internally by helpers that need
// to perform a locked read-modify-write without double-locking.
func saveIndexLocked(index types.Index, config types.Config) error {
	indexPath := GetIndexPath(config)
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(indexPath, data, 0644)
}

// GetIndexPath returns the full path to the index.json file used to
// store metadata about cached files, based on the provided config.
func GetIndexPath(config types.Config) string {
	cacheDir := ExpandPath(config.Cache.Directory)
	return filepath.Join(cacheDir, "index.json")
}

// LoadIndex reads and unmarshals the index.json file into an Index struct.
// If the file does not exist, it returns an empty Index. Returns an error
// if reading or unmarshalling fails.
func LoadIndex(config types.Config) (types.Index, error) {
	indexMu.RLock()
	defer indexMu.RUnlock()
	return loadIndexLocked(config)
}

// loadIndexLocked is the unexported implementation of LoadIndex that
// assumes indexMu is already held (for read or write).
func loadIndexLocked(config types.Config) (types.Index, error) {
	var index types.Index
	indexPath := GetIndexPath(config)
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty index if file doesn't exist
			return types.Index{Items: []types.DeletedItem{}}, nil
		}
		return index, err
	}
	err = json.Unmarshal(data, &index)
	return index, err
}

// AddToIndex adds a DeletedItem to the index and saves the updated
// index to disk using the provided config. Returns an error if loading
// or saving the index fails.
func AddToIndex(item types.DeletedItem, config types.Config) error {
	indexMu.Lock()
	defer indexMu.Unlock()

	index, err := loadIndexLocked(config)
	if err != nil {
		return err
	}
	index.Items = append(index.Items, item)
	return saveIndexLocked(index, config)
}

// RemoveFromIndex removes a DeletedItem with the specified ID from the
// index and saves the updated index to disk. Returns an error if loading
// or saving the index fails.
func RemoveFromIndex(itemID string, config types.Config) error {
	indexMu.Lock()
	defer indexMu.Unlock()

	index, err := loadIndexLocked(config)
	if err != nil {
		return err
	}

	remainingItems := make([]types.DeletedItem, 0, len(index.Items))
	for _, item := range index.Items {
		if item.ID != itemID {
			remainingItems = append(remainingItems, item)
		}
	}

	index.Items = remainingItems
	return saveIndexLocked(index, config)
}
