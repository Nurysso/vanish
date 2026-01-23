package helpers

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	"vanish/internal/types"
)

func TestGetConfigPath(t *testing.T) {
	path := GetConfigPath()
	if path == "" || path == "could find Config File" {
		t.Error("GetConfigPath returned empty or error string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("Expected absolute path, got: %s", path)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"Zero bytes", 0, "0 B"},
		{"Less than KB", 512, "512 B"},
		{"Exactly 1 KB", 1024, "1.0 KB"},
		{"1.5 KB", 1536, "1.5 KB"},
		{"1 MB", 1048576, "1.0 MB"},
		{"1 GB", 1073741824, "1.0 GB"},
		{"1.5 GB", 1610612736, "1.5 GB"},
		{"1 TB", 1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %s; expected %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestCreateThemeStyles(t *testing.T) {
	config := getTestConfig()
	styles := CreateThemeStyles(config)

	// Check that styles are created (non-nil check through String() method)
	if styles.Title.String() == "" {
		t.Error("Title style not created")
	}
	if styles.Header.String() == "" {
		t.Error("Header style not created")
	}
	if styles.Success.String() == "" {
		t.Error("Success style not created")
	}
}

func TestRenderThemeAsString(t *testing.T) {
	config := getTestConfig()
	result := RenderThemeAsString(config)

	if result == "" {
		t.Error("RenderThemeAsString returned empty string")
	}
	if len(result) < 10 {
		t.Error("RenderThemeAsString result too short")
	}
}

func TestGetTerminalSize(t *testing.T) {
	width, height := GetTerminalSize()

	// Should return fallback values or actual terminal size
	if width <= 0 || height <= 0 {
		t.Errorf("Invalid terminal size: width=%d, height=%d", width, height)
	}

	// At minimum, should return fallback values
	if width < 80 || height < 24 {
		t.Logf("Terminal size smaller than default: width=%d, height=%d", width, height)
	}
}

func TestExpandPath(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Home prefix", "~/test", filepath.Join(homeDir, "test")},
		{"Relative path", "test", filepath.Join(homeDir, "test")},
		{"Absolute path", "/tmp/test", "/tmp/test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandPath(%s) = %s; expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCountFilesInDirectory(t *testing.T) {
	// Create temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files
	os.Create(filepath.Join(tmpDir, "file1.txt"))
	os.Create(filepath.Join(tmpDir, "file2.txt"))
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.Create(filepath.Join(tmpDir, "subdir", "file3.txt"))

	count, err := CountFilesInDirectory(tmpDir)
	if err != nil {
		t.Fatalf("CountFilesInDirectory failed: %v", err)
	}

	// Should count 2 files + 1 subdir + 1 file in subdir = 4
	if count != 4 {
		t.Errorf("Expected 4 items, got %d", count)
	}
}

func TestCountValidFiles(t *testing.T) {
	fileInfos := []types.FileInfo{
		{Path: "test1", Exists: true},
		{Path: "test2", Exists: false},
		{Path: "test3", Exists: true},
		{Path: "test4", Exists: false},
		{Path: "test5", Exists: true},
	}

	count := CountValidFiles(fileInfos)
	if count != 3 {
		t.Errorf("Expected 3 valid files, got %d", count)
	}
}

func TestFindNextValidFile(t *testing.T) {
	fileInfos := []types.FileInfo{
		{Path: "test1", Exists: false},
		{Path: "test2", Exists: false},
		{Path: "test3", Exists: true},
		{Path: "test4", Exists: false},
		{Path: "test5", Exists: true},
	}

	tests := []struct {
		name       string
		startIndex int
		expected   int
	}{
		{"From start", 0, 2},
		{"From middle", 3, 4},
		{"No valid files", 5, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindNextValidFile(fileInfos, tt.startIndex)
			if result != tt.expected {
				t.Errorf("FindNextValidFile(%d) = %d; expected %d", tt.startIndex, result, tt.expected)
			}
		})
	}
}

func TestMoveFile(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "destination.txt")

	// Create source file
	content := []byte("test content")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Move file
	if err := MoveFile(srcPath, dstPath); err != nil {
		t.Fatalf("MoveFile failed: %v", err)
	}

	// Check source doesn't exist
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Error("Source file still exists after move")
	}

	// Check destination exists with correct content
	destContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(destContent) != string(content) {
		t.Error("Destination file content doesn't match source")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "destination.txt")

	// Create source file
	content := []byte("test content")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Check both files exist
	if _, err := os.Stat(srcPath); err != nil {
		t.Error("Source file disappeared after copy")
	}

	destContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(destContent) != string(content) {
		t.Error("Destination file content doesn't match source")
	}
}

func TestCopyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "source")
	dstDir := filepath.Join(tmpDir, "destination")

	// Create source directory structure
	os.Mkdir(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644)
	os.Mkdir(filepath.Join(srcDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("content2"), 0644)

	// Copy directory
	if err := CopyDirectory(srcDir, dstDir); err != nil {
		t.Fatalf("CopyDirectory failed: %v", err)
	}

	// Verify destination structure
	if _, err := os.Stat(filepath.Join(dstDir, "file1.txt")); err != nil {
		t.Error("file1.txt not copied")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "subdir", "file2.txt")); err != nil {
		t.Error("subdir/file2.txt not copied")
	}
}

func TestMoveDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "source")
	dstDir := filepath.Join(tmpDir, "destination")

	// Create source directory
	os.Mkdir(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("content"), 0644)

	// Move directory
	if err := MoveDirectory(srcDir, dstDir); err != nil {
		t.Fatalf("MoveDirectory failed: %v", err)
	}

	// Check source doesn't exist
	if _, err := os.Stat(srcDir); !os.IsNotExist(err) {
		t.Error("Source directory still exists after move")
	}

	// Check destination exists
	if _, err := os.Stat(filepath.Join(dstDir, "file.txt")); err != nil {
		t.Error("Destination directory or file doesn't exist")
	}
}

func TestGetDirectorySize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with known sizes
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("12345"), 0644)      // 5 bytes
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("1234567890"), 0644) // 10 bytes
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file3.txt"), []byte("123"), 0644) // 3 bytes

	size, err := GetDirectorySize(tmpDir)
	if err != nil {
		t.Fatalf("GetDirectorySize failed: %v", err)
	}

	expectedSize := int64(18) // 5 + 10 + 3
	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}
}

func TestCheckFilesExist(t *testing.T) {
	tmpDir := t.TempDir()

	existingFile := filepath.Join(tmpDir, "exists.txt")
	os.WriteFile(existingFile, []byte("content"), 0644)

	existingDir := filepath.Join(tmpDir, "testdir")
	os.Mkdir(existingDir, 0755)
	os.WriteFile(filepath.Join(existingDir, "file1.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(existingDir, "file2.txt"), []byte("content"), 0644)

	nonExistingFile := filepath.Join(tmpDir, "not_exists.txt")

	cmd := CheckFilesExist([]string{existingFile, existingDir, nonExistingFile})
	msg := cmd()

	filesMsg, ok := msg.(types.FilesExistMsg)
	if !ok {
		t.Fatal("Expected FilesExistMsg")
	}

	if len(filesMsg.FileInfos) != 3 {
		t.Errorf("Expected 3 file infos, got %d", len(filesMsg.FileInfos))
	}

	// Check existing file
	if !filesMsg.FileInfos[0].Exists {
		t.Error("First file should exist")
	}
	if filesMsg.FileInfos[0].IsDirectory {
		t.Error("First file should not be a directory")
	}

	// Check existing directory
	if !filesMsg.FileInfos[1].Exists {
		t.Error("Second item (directory) should exist")
	}
	if !filesMsg.FileInfos[1].IsDirectory {
		t.Error("Second item should be a directory")
	}
	if filesMsg.FileInfos[1].FileCount != 2 {
		t.Errorf("Expected 2 files in directory, got %d", filesMsg.FileInfos[1].FileCount)
	}

	// Check non-existing file
	if filesMsg.FileInfos[2].Exists {
		t.Error("Third file should not exist")
	}
}

func TestLoadAndSaveIndex(t *testing.T) {
	tmpDir := t.TempDir()

	config := getTestConfig()
	config.Cache.Directory = tmpDir

	// Create test index
	testIndex := types.Index{
		Items: []types.DeletedItem{
			{
				ID:           "test1",
				OriginalPath: "/home/user/test.txt",
				DeleteDate:   time.Now(),
				CachePath:    filepath.Join(tmpDir, "test1"),
				IsDirectory:  false,
				Size:         1024,
			},
			{
				ID:           "test2",
				OriginalPath: "/home/user/folder",
				DeleteDate:   time.Now(),
				CachePath:    filepath.Join(tmpDir, "test2"),
				IsDirectory:  true,
				Size:         4096,
			},
		},
	}

	// Save index
	if err := SaveIndex(testIndex, config); err != nil {
		t.Fatalf("SaveIndex failed: %v", err)
	}

	// Load index
	loadedIndex, err := LoadIndex(config)
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Verify loaded index matches
	if len(loadedIndex.Items) != len(testIndex.Items) {
		t.Errorf("Expected %d items, got %d", len(testIndex.Items), len(loadedIndex.Items))
	}

	if loadedIndex.Items[0].ID != testIndex.Items[0].ID {
		t.Error("First item ID doesn't match")
	}

	if loadedIndex.Items[1].IsDirectory != testIndex.Items[1].IsDirectory {
		t.Error("Second item IsDirectory doesn't match")
	}
}

func TestClearAllCache(t *testing.T) {
	tmpDir := t.TempDir()

	config := getTestConfig()
	config.Cache.Directory = tmpDir
	config.Logging.Enabled = false

	// Create some files in cache
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("content"), 0644)

	// Create initial index
	index := types.Index{
		Items: []types.DeletedItem{
			{ID: "test", CachePath: filepath.Join(tmpDir, "test.txt")},
		},
	}
	SaveIndex(index, config)

	cmd := ClearAllCache(config)
	msg := cmd()

	clearMsg, ok := msg.(types.ClearMsg)
	if !ok {
		t.Fatal("Expected ClearMsg")
	}

	if clearMsg.Err != nil {
		t.Errorf("ClearAllCache failed: %v", clearMsg.Err)
	}

	// Verify index is empty
	loadedIndex, _ := LoadIndex(config)
	if len(loadedIndex.Items) != 0 {
		t.Errorf("Expected empty index, got %d items", len(loadedIndex.Items))
	}
}

func TestPurgeOldFiles(t *testing.T) {
	tmpDir := t.TempDir()

	config := getTestConfig()
	config.Cache.Directory = tmpDir
	config.Logging.Enabled = false

	// Create index with old and new items
	oldTime := time.Now().Add(-10 * 24 * time.Hour)
	newTime := time.Now()

	oldFile := filepath.Join(tmpDir, "old.txt")
	newFile := filepath.Join(tmpDir, "new.txt")
	os.WriteFile(oldFile, []byte("old"), 0644)
	os.WriteFile(newFile, []byte("new"), 0644)

	index := types.Index{
		Items: []types.DeletedItem{
			{ID: "old", CachePath: oldFile, DeleteDate: oldTime, IsDirectory: false},
			{ID: "new", CachePath: newFile, DeleteDate: newTime, IsDirectory: false},
		},
	}
	SaveIndex(index, config)

	cmd := PurgeOldFiles(config, "7")
	msg := cmd()

	purgeMsg, ok := msg.(types.PurgeMsg)
	if !ok {
		t.Fatal("Expected PurgeMsg")
	}

	if purgeMsg.Err != nil {
		t.Errorf("PurgeOldFiles failed: %v", purgeMsg.Err)
	}

	if purgeMsg.PurgedCount != 1 {
		t.Errorf("Expected 1 purged file, got %d", purgeMsg.PurgedCount)
	}

	// Check old file is gone
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("Old file should be purged")
	}

	// Check new file still exists
	if _, err := os.Stat(newFile); err != nil {
		t.Error("New file should still exist")
	}
}

func TestCheckRestoreItems(t *testing.T) {
	tmpDir := t.TempDir()

	config := getTestConfig()
	config.Cache.Directory = tmpDir

	// Create test index
	index := types.Index{
		Items: []types.DeletedItem{
			{ID: "1", OriginalPath: "/home/user/document.txt", CachePath: filepath.Join(tmpDir, "doc1")},
			{ID: "2", OriginalPath: "/home/user/photo.jpg", CachePath: filepath.Join(tmpDir, "photo1")},
			{ID: "3", OriginalPath: "/home/user/report.pdf", CachePath: filepath.Join(tmpDir, "report1")},
		},
	}
	SaveIndex(index, config)

	cmd := CheckRestoreItems([]string{"document"}, config)
	msg := cmd()

	restoreMsg, ok := msg.(types.RestoreItemsMsg)
	if !ok {
		t.Fatal("Expected RestoreItemsMsg")
	}

	if len(restoreMsg.Items) != 1 {
		t.Errorf("Expected 1 matching item, got %d", len(restoreMsg.Items))
	}

	if len(restoreMsg.Items) > 0 && filepath.Base(restoreMsg.Items[0].OriginalPath) != "document.txt" {
		t.Error("Wrong item matched")
	}
}

func TestDeletedItemType(t *testing.T) {
	tests := []struct {
		name     string
		item     types.DeletedItem
		expected string
	}{
		{
			name:     "Regular file",
			item:     types.DeletedItem{IsSymlink: false, IsDirectory: false},
			expected: "file",
		},
		{
			name:     "Directory",
			item:     types.DeletedItem{IsSymlink: false, IsDirectory: true},
			expected: "directory",
		},
		{
			name:     "Symlink",
			item:     types.DeletedItem{IsSymlink: true, IsDirectory: false},
			expected: "symlink",
		},
		{
			name:     "Symlink to directory",
			item:     types.DeletedItem{IsSymlink: true, IsDirectory: true},
			expected: "symlink",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.ItemType()
			if result != tt.expected {
				t.Errorf("ItemType() = %s; expected %s", result, tt.expected)
			}
		})
	}
}

// Helper function to create a test config
func getTestConfig() types.Config {
	var config types.Config

	config.Cache.Directory = os.TempDir()
	config.Cache.Days = 30
	config.Cache.NoConfirm = false

	config.Logging.Enabled = false
	config.Logging.Directory = os.TempDir()

	config.UI.Theme = "default"
	config.UI.Colors.Primary = "#FF0000"
	config.UI.Colors.Secondary = "#00FF00"
	config.UI.Colors.Text = "#FFFFFF"
	config.UI.Colors.Highlight = "#FFFF00"
	config.UI.Colors.Success = "#00FF00"
	config.UI.Colors.Error = "#FF0000"
	config.UI.Colors.Warning = "#FFA500"
	config.UI.Colors.Muted = "#888888"
	config.UI.Colors.Border = "#CCCCCC"

	config.UI.Progress.Style = "gradient"
	config.UI.Progress.ShowEmoji = true
	config.UI.Progress.Animation = true

	return config
}
