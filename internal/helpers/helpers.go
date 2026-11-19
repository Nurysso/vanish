// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Dawood Khan

// Package helpers have all the helper function.

package helpers

import (
	"errors"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	// "log"
	"os"
	// "os/exec"
	"path/filepath"
	// "runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"vanish/internal/types"
)

// GetConfigPath returns the path to vanish.toml. It prefers
// os.UserConfigDir() (which respects XDG_CONFIG_HOME on Linux), and
// falls back to $HOME/.config if that's unavailable, and finally to a
// directory relative to the working directory if neither can be
// resolved. It never returns a non-path placeholder string.
func GetConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		homeDir, herr := os.UserHomeDir()
		if herr != nil || homeDir == "" {
			// Last resort: relative to the current working directory.
			return filepath.Join(".", "vanish", "vanish.toml")
		}
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "vanish", "vanish.toml")
}

// FormatBytes formats bytes and is used in
// cmd/commands[showList.go,showInfo.go]
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// SendNotification sends a desktop notification based on the provided title and message.
// It only sends notifications if the corresponding flags are enabled in the config.
// It's tested on only Linux tho it should also work on macOS, and Windows platforms.
// func SendNotification(title, message string, config types.Config) {
// 	if !config.Notifications.NotifySuccess && !config.Notifications.NotifyErrors {
// 		return
// 	}

// 	if config.Notifications.DesktopEnabled {
// 		// Run the notification in a separate goroutine to avoid blocking the UI.
// 		go func() {
// 			var err error

// 			switch runtime.GOOS {
// 			case "linux":
// 				err = exec.Command("notify-send", title, message).Run()
// 			case "darwin":
// 				script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
// 				err = exec.Command("osascript", "-e", script).Run()
// 			}

// 			if err != nil {
// 				log.Printf("failed to send notification: %v", err)
// 			}
// 		}()
// 	}
// }

// SetUpProgress defines progress bar style
func SetUpProgress(config types.Config) progress.Model {
	prog := progress.New()
	prog.Width = 50

	switch config.UI.Progress.Style {
	case "solid":
		prog = progress.New(progress.WithSolidFill(config.UI.Colors.Primary))
	case "rainbow":
		prog = progress.New(progress.WithGradient("#FF0000", "#9400D3")) //  "#FF7F00", "#FFFF00", "#00FF00", "#0000FF", "#4B0082",
	default: // gradient
		prog = progress.New(progress.WithGradient(config.UI.Colors.Primary, config.UI.Colors.Secondary))
	}
	return prog
}

// CreateThemeStyles create lipgloss themes
func CreateThemeStyles(config types.Config) types.ThemeStyles {
	colors := config.UI.Colors
	return types.ThemeStyles{
		Root: lipgloss.NewStyle().
			PaddingTop(1).
			PaddingRight(2),
		// PaddingBottom(2).
		// PaddingLeft(4),
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Text)).
			Bold(true).
			Padding(0, 2, 0, 2).
			MarginBottom(1),
		Header: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Primary)).
			Bold(true).
			Underline(true).
			MarginBottom(1),
		Question: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Primary)).
			Bold(true),
		Filename: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Highlight)).
			Bold(true).
			Underline(true),
		IconStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Highlight)).
			Bold(true),
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Success)).
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colors.Success)),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Error)).
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colors.Error)),
		Warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Warning)).
			Bold(true),
		Info: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Secondary)).
			Padding(0, 1),
		// Removed border and made it responsive
		// Border(lipgloss.NormalBorder()).
		// BorderForeground(lipgloss.Color(colors.Border))

		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Muted)).
			Italic(true).
			MarginTop(1),
		Progress: lipgloss.NewStyle().
			MarginTop(1).
			MarginBottom(1),
		Border: lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color(colors.Border)).
			Padding(1),
		List: lipgloss.NewStyle().
			MarginLeft(2).
			MarginTop(1).
			MarginBottom(1),
		StatusGood: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Success)),
		StatusBad: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Error)),
	}
}

// RenderThemeAsString renders strings for dummy ui in --themes flags
// used in cmd/commands/showThemes.go
func RenderThemeAsString(cfg types.Config) string {
	styles := CreateThemeStyles(cfg)

	result := styles.Root.Render(
		styles.Question.Render("Are you sure you want to delete the following items?") + "\n\n" +
			styles.List.Render(
				"  📄 "+styles.Filename.Render("example.txt")+"\n"+
					"  📁 "+styles.Filename.Render("temp_folder/")+styles.Info.Render(" (5 items)")+"\n",
			) + "\n" +
			styles.Info.Render("Total items to delete: 2 | Recoverable for 10 days") + "\n\n" +
			func() string {
				prog := SetUpProgress(cfg)
				return styles.Progress.Render(prog.ViewAs(0.75))
			}() + "\n\n" +
			styles.Success.Render("✅ Success") + "\n" + " " +
			styles.Warning.Render("⚠ Warning") + "\n" +
			styles.Error.Render("❌ Error") + "\n" + " " +
			styles.Help.Render("Press 'y' to confirm, 'n' to cancel"),
	)

	return result + "\n"
}

// GetTerminalSize returns the current terminal width and height
func GetTerminalSize() (int, int) {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		width, height, err := term.GetSize(int(os.Stdout.Fd()))
		if err == nil {
			return width, height
		}
	}
	// Fallback to reasonable defaults
	return 80, 24
}

// ExpandPath expands a given path by resolving '~/' to the user's home directory
// and converting relative paths to absolute paths. If the path is already absolute,
// it is returned unchanged. If the home directory cannot be resolved, the path is
// returned unchanged (for "~/" paths) or made absolute relative to the working
// directory (for other relative paths), rather than silently producing a path
// rooted at an empty string.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil || homeDir == "" {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	if !filepath.IsAbs(path) {
		homeDir, err := os.UserHomeDir()
		if err != nil || homeDir == "" {
			if abs, aerr := filepath.Abs(path); aerr == nil {
				return abs
			}
			return path
		}
		return filepath.Join(homeDir, path)
	}
	return path
}

// ClearAllCache removes all cached files and directories, recreates the
// cache directory, resets the index, and logs the operation if logging
// is enabled. Returns a tea.Msg with any error encountered.
func ClearAllCache(config types.Config) tea.Cmd {
	return func() tea.Msg {
		cacheDir := ExpandPath(config.Cache.Directory)

		// Remove all files in cache directory
		if err := RemoveAllWritable(cacheDir); err != nil {
			return types.ClearMsg{Err: fmt.Errorf("failed to remove cache directory: %w", err)}
		}

		// Recreate cache directory
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return types.ClearMsg{Err: fmt.Errorf("failed to recreate cache directory: %w", err)}
		}

		// Create empty index
		index := types.Index{Items: []types.DeletedItem{}}
		if err := SaveIndex(index, config); err != nil {
			return types.ClearMsg{Err: fmt.Errorf("failed to save index: %w", err)}
		}

		// Log clear operation
		if config.Logging.Enabled {
			if err := logClearOperation(config); err != nil {
				// Log error but don't fail the entire operation
				// since the cache was successfully cleared
				return types.ClearMsg{Err: fmt.Errorf("cache cleared but logging failed: %w", err)}
			}
		}

		return types.ClearMsg{Err: nil}
	}
}

// PurgeOldFiles removes cached files and directories that are older than
// the specified number of days. Updates the index and logs each purge
// if logging is enabled. Returns a tea.Msg containing the purge results.
func PurgeOldFiles(config types.Config, daysStr string) tea.Cmd {
	return func() tea.Msg {
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return types.PurgeMsg{Err: fmt.Errorf("invalid days value: %s", daysStr)}
		}

		// Validate days is positive
		if days <= 0 {
			return types.PurgeMsg{Err: fmt.Errorf("days must be positive, got: %d", days)}
		}

		purged, err := purgeBefore(daysAgo(days), "PURGE", config)
		return types.PurgeMsg{PurgedCount: purged, Err: err}
	}
}

// daysAgo returns the cutoff for a retention period. The cap stops a huge value
// overflowing into a future date, which would make every item look expired.
func daysAgo(days int) time.Time {
	const maxDays = 1_000_000 // about 2,700 years, older than any deleted item
	return time.Now().AddDate(0, 0, -min(days, maxDays))
}

// CleanupExpired removes items older than the configured retention period.
// A non-positive retention never expires anything, because treating it as a
// cutoff would permanently delete the whole trash, including the file just moved.
func CleanupExpired(config types.Config) (int, error) {
	if config.Cache.Days <= 0 {
		return 0, nil
	}
	return purgeBefore(daysAgo(config.Cache.Days), "CLEANUP", config)
}

// purgeBefore permanently removes items deleted before cutoff and logs each one
// under the given operation name. An item that can't be removed stays in the
// index. Returns the number removed and any combined error.
func purgeBefore(cutoff time.Time, operation string, config types.Config) (int, error) {
	index, err := LoadIndex(config)
	if err != nil {
		return 0, fmt.Errorf("error loading index: %w", err)
	}

	// Pre-allocate slice with estimated capacity
	remainingItems := make([]types.DeletedItem, 0, len(index.Items))
	purgedCount := 0
	var purgeErrors []error

	for _, item := range index.Items {
		if !item.DeleteDate.Before(cutoff) {
			remainingItems = append(remainingItems, item)
			continue
		}

		// Remove the actual file or directory
		var removeErr error
		if item.IsDirectory {
			removeErr = RemoveAllWritable(item.CachePath)
		} else {
			removeErr = os.Remove(item.CachePath)
		}

		// Track errors but continue purging other files
		if removeErr != nil && !os.IsNotExist(removeErr) {
			purgeErrors = append(purgeErrors, fmt.Errorf("failed to remove %s: %w", item.CachePath, removeErr))
			// Keep item in index if we couldn't remove it
			remainingItems = append(remainingItems, item)
			continue
		}

		purgedCount++

		if config.Logging.Enabled {
			if err := LogOperation(operation, item, config); err != nil {
				// Log error but don't fail the operation
				purgeErrors = append(purgeErrors, fmt.Errorf("failed to log removal of %s: %w", item.OriginalPath, err))
			}
		}
	}

	// Update index with remaining items
	index.Items = remainingItems
	if err := SaveIndex(index, config); err != nil {
		return purgedCount, fmt.Errorf("error updating index: %w", err)
	}

	return purgedCount, errors.Join(purgeErrors...)
}

// CheckRestoreItems searches the index for deleted items that match any of the
// given patterns (see matchesPattern). Each item is returned once, with an
// OriginalPath that is free on disk and not shared with another returned item.
func CheckRestoreItems(patterns []string, config types.Config) tea.Cmd {
	return func() tea.Msg {
		index, err := LoadIndex(config)
		if err != nil {
			return types.ErrorMsg(fmt.Sprintf("Error loading index: %v", err))
		}

		var matchingItems []types.DeletedItem
		seen := make(map[string]bool)
		claimed := make(map[string]bool)
		for _, pattern := range patterns {
			for _, item := range index.Items {
				if seen[item.ID] || !matchesPattern(item.OriginalPath, pattern) {
					continue
				}
				seen[item.ID] = true
				item.OriginalPath = resolvePathConflict(item.OriginalPath, claimed)
				claimed[item.OriginalPath] = true
				matchingItems = append(matchingItems, item)
			}
		}

		return types.RestoreItemsMsg{Items: matchingItems}
	}
}

// matchesPattern reports whether pattern is a case-insensitive substring of
// path or a glob matching its base name. Both are tried so that a literal name
// like "photo[1].jpg" still finds itself.
func matchesPattern(path, pattern string) bool {
	path, pattern = strings.ToLower(path), strings.ToLower(pattern)
	if strings.Contains(path, pattern) {
		return true
	}
	matched, _ := filepath.Match(pattern, filepath.Base(path))
	return matched
}

// resolvePathConflict returns originalPath if it is free, otherwise the first
// free numbered variant. For example: test -> test1 -> test2, etc. A path is
// taken if it is in claimed or something exists there, including a dangling symlink.
func resolvePathConflict(originalPath string, claimed map[string]bool) string {
	taken := func(path string) bool {
		_, err := os.Lstat(path)
		return claimed[path] || err == nil
	}

	if !taken(originalPath) {
		return originalPath
	}

	// Extract directory, base name, and extension
	dir := filepath.Dir(originalPath)
	base := filepath.Base(originalPath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	// Try appending numbers until we find a free path
	for counter := 1; ; counter++ {
		newPath := filepath.Join(dir, fmt.Sprintf("%s%d%s", nameWithoutExt, counter, ext))
		if !taken(newPath) {
			return newPath
		}
	}
}

// CheckFilesExist checks if the specified files or directories exist on disk,
// gathers metadata about each, and returns a tea.Msg with the results.
func CheckFilesExist(filenames []string) tea.Cmd {
	return func() tea.Msg {
		fileInfos := make([]types.FileInfo, len(filenames))

		for i, filename := range filenames {
			// Lstat: a dangling symlink exists and can be deleted.
			stat, err := os.Lstat(filename)
			if err != nil {
				fileInfos[i] = types.FileInfo{
					Path:   filename,
					Exists: false,
					Error:  err.Error(),
				}
				continue
			}

			isDir := stat.IsDir()
			fileCount := 0

			if isDir {
				fileCount, _ = CountFilesInDirectory(filename)
			}

			fileInfos[i] = types.FileInfo{
				Path:        filename,
				IsDirectory: isDir,
				FileCount:   fileCount,
				Exists:      true,
			}
		}

		return types.FilesExistMsg{FileInfos: fileInfos}
	}
}

// CountFilesInDirectory returns the number of regular files (not including
// directories themselves) in the specified directory and its subdirectories.
// Errors encountered while walking the tree are ignored for individual
// entries (e.g. permission errors), but the entry itself is skipped.
func CountFilesInDirectory(dir string) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip problematic files
		}
		if path == dir {
			return nil
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	if err != nil {
		return 0, err // return the error from filepath.Walk
	}
	return count, nil
}

// CountValidFiles returns the number of FileInfo entries that represent
// existing files or directories.
func CountValidFiles(fileInfos []types.FileInfo) int {
	count := 0
	for _, info := range fileInfos {
		if info.Exists {
			count++
		}
	}
	return count
}

// FindNextValidFile returns the index of the next valid file (i.e., one that exists)
// in the given FileInfo slice, starting from startIndex. Returns -1 if none found.
func FindNextValidFile(fileInfos []types.FileInfo, startIndex int) int {
	for i := startIndex; i < len(fileInfos); i++ {
		if fileInfos[i].Exists {
			return i
		}
	}
	return -1
}

// MoveFile moves a file from the source path to the destination path.
// It handles regular files, symlinks, and special files appropriately.
func MoveFile(src, dst string) error {
	// Check if it's a symlink first (before opening)
	isSymlink, err := IsSymlink(src)
	if err != nil {
		return fmt.Errorf("failed to check if symlink: %w", err)
	}

	if isSymlink {
		return MoveSymlink(src, dst)
	}

	// Rename keeps metadata and handles FIFOs and sockets, which can't be copied.
	err = os.Rename(src, dst)
	if !errors.Is(err, syscall.EXDEV) {
		return err
	}

	if err := CopyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

// MoveDirectory moves a directory from src to dst. Attempts an atomic move
// using os.Rename first, and falls back to a copy-and-remove approach
// if that fails. Properly handles symlinks within directories.
func MoveDirectory(src, dst string) error {
	// Use os.Rename for atomic operation when possible (same filesystem)
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Moving a directory into itself. Copying would recurse into its own output.
	if errors.Is(err, syscall.EINVAL) {
		return err
	}

	// Fallback to copy + remove for cross-filesystem moves, and for read-only
	// directories, which can't be renamed to a new parent
	if err := CopyDirectory(src, dst); err != nil {
		return err
	}

	return RemoveAllWritable(src)
}

// RemoveAllWritable removes the file or directory tree at path, first
// walking it to add the owner-write bit to any directory that's missing it.
// os.RemoveAll requires write+execute permission on a directory to unlink
// its children, and read-only trees - such as Go's module cache, which
// chmods directories to dr-xr-xr-x - would otherwise fail partway through
// with "permission denied". Read-only directories can still be walked
// since the execute bit (needed to list contents) is normally present.
func RemoveAllWritable(path string) error {
	_ = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			// Can't stat this entry; let RemoveAll surface the error.
			return nil
		}
		if info.IsDir() && info.Mode()&0200 == 0 {
			if chmodErr := os.Chmod(p, info.Mode()|0700); chmodErr != nil {
				return chmodErr
			}
		}
		return nil
	})
	return os.RemoveAll(path)
}

// CopyDirectory recursively copies the contents of the source directory to the
// destination directory. Preserves file and directory modes and modification
// times. Returns an error if any operation fails.
func CopyDirectory(src, dst string) error {
	// Use Lstat to not follow symlinks when checking source
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}

	// Create destination directory, writable until it's filled
	if err := os.MkdirAll(dst, srcInfo.Mode().Perm()|0700); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// Check if entry is a symlink
		isSymlink, err := IsSymlink(srcPath)
		if err != nil {
			return err
		}

		if isSymlink {
			// Handle symlink
			linkTarget, err := os.Readlink(srcPath)
			if err != nil {
				return fmt.Errorf("failed to read symlink %s: %w", srcPath, err)
			}
			if err := os.Symlink(linkTarget, dstPath); err != nil {
				return fmt.Errorf("failed to create symlink %s: %w", dstPath, err)
			}
		} else if entry.IsDir() {
			// Handle directory
			if err := CopyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Handle regular file
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to preserve mode for directory %s: %w", dst, err)
	}

	// Preserve the directory's modification time. This must happen after
	// all entries have been written, since creating entries inside dst
	// updates dst's mtime.
	modTime := srcInfo.ModTime()
	if err := os.Chtimes(dst, modTime, modTime); err != nil {
		return fmt.Errorf("failed to preserve mod time for directory %s: %w", dst, err)
	}

	return nil
}

// CopyFile copies a file from src to dst, preserving its permissions and
// modification time. Returns an error if opening, copying, or creating fails.
// Only regular files can be copied: opening a FIFO would block forever, and
// symlinks need MoveSymlink.
func CopyFile(src, dst string) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("cannot copy %s: not a regular file", src)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if err := dstFile.Chmod(srcInfo.Mode()); err != nil {
		return err
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Preserve modification time so restored/copied files don't appear
	// to have been created "now".
	modTime := srcInfo.ModTime()
	if err := os.Chtimes(dst, modTime, modTime); err != nil {
		return fmt.Errorf("failed to preserve mod time for %s: %w", dst, err)
	}

	return nil
}

// GetDirectorySize returns the total size in bytes of all non-directory
// files within the specified directory and its subdirectories.
func GetDirectorySize(dir string) (int64, error) {
	var size int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, err // return the error from filepath.Walk
	}
	return size, nil
}
