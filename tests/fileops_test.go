package tests

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	h "vanish/internal/helpers"
	"vanish/internal/types"
)

// knownBug skips a test that asserts correct behaviour the code doesn't have
// yet. Run them with VANISH_KNOWN_BUGS=1; delete the call once the bug is fixed.
func knownBug(t *testing.T, why string) {
	t.Helper()
	if os.Getenv("VANISH_KNOWN_BUGS") == "" {
		t.Skip("known bug: " + why)
	}
}

func testCfg(t *testing.T) types.Config {
	t.Helper()
	cfg := getTestConfig()
	cfg.Cache.Directory = filepath.Join(t.TempDir(), "cache")
	cfg.Cache.Days = 10
	cfg.Logging.Directory = filepath.Join(t.TempDir(), "logs")
	if err := os.MkdirAll(cfg.Cache.Directory, 0755); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func wantFile(t *testing.T, path, content string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("expected %s to exist: %v", path, err)
	} else if string(got) != content {
		t.Errorf("%s = %q, want %q", path, got, content)
	}
}

func wantGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Errorf("expected %s to be gone (err=%v)", path, err)
	}
}

func wantLink(t *testing.T, path, target string) {
	t.Helper()
	if got, err := os.Readlink(path); err != nil || got != target {
		t.Errorf("Readlink(%s) = %q, %v; want %q", path, got, err, target)
	}
}

func symlink(t *testing.T, target, path string) {
	t.Helper()
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}

func loadItems(t *testing.T, cfg types.Config) []types.DeletedItem {
	t.Helper()
	index, err := h.LoadIndex(cfg)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	return index.Items
}

func saveItems(t *testing.T, cfg types.Config, items ...types.DeletedItem) {
	t.Helper()
	if err := h.SaveIndex(types.Index{Items: items}, cfg); err != nil {
		t.Fatal(err)
	}
}

// readOnlyTree creates root/sub/file.txt with every directory set to 0555, the
// shape of Go's module cache. Root ignores permission bits, so it skips there.
func readOnlyTree(t *testing.T, root string) {
	t.Helper()
	if os.Getuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	writeFile(t, filepath.Join(root, "sub", "file.txt"), "ro")
	for _, dir := range []string{filepath.Join(root, "sub"), root} {
		if err := os.Chmod(dir, 0555); err != nil {
			t.Fatal(err)
		}
	}
	// Runs before t.TempDir's cleanup, which couldn't remove the tree otherwise.
	t.Cleanup(func() { makeWritable(root) })
}

func makeWritable(root string) {
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			os.Chmod(p, 0755)
		}
		return nil
	})
}

func TestMoveSymlink(t *testing.T) {
	t.Run("keeps the target untouched", func(t *testing.T) {
		dir := t.TempDir()
		target, link, dst := filepath.Join(dir, "target"), filepath.Join(dir, "link"), filepath.Join(dir, "dst")
		writeFile(t, target, "precious")
		symlink(t, target, link)

		if err := h.MoveFile(link, dst); err != nil { // MoveFile must dispatch to MoveSymlink
			t.Fatal(err)
		}
		wantGone(t, link)
		wantLink(t, dst, target)
		wantFile(t, target, "precious")
	})

	t.Run("moves a dangling link", func(t *testing.T) {
		dir := t.TempDir()
		link, dst := filepath.Join(dir, "link"), filepath.Join(dir, "dst")
		symlink(t, "/nonexistent/target", link)

		if err := h.MoveSymlink(link, dst); err != nil {
			t.Fatal(err)
		}
		wantGone(t, link)
		wantLink(t, dst, "/nonexistent/target")
	})

	t.Run("keeps the source when the destination exists", func(t *testing.T) {
		dir := t.TempDir()
		link, dst := filepath.Join(dir, "link"), filepath.Join(dir, "dst")
		symlink(t, "somewhere", link)
		writeFile(t, dst, "occupied")

		if err := h.MoveSymlink(link, dst); err == nil {
			t.Error("expected an error")
		}
		wantLink(t, link, "somewhere")
		wantFile(t, dst, "occupied")
	})

	t.Run("errors on a regular file", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "file")
		writeFile(t, file, "x")
		if err := h.MoveSymlink(file, filepath.Join(dir, "dst")); err == nil {
			t.Error("expected an error")
		}
		wantFile(t, file, "x")
	})
}

func TestRestoreSymlink(t *testing.T) {
	t.Run("recreates missing parents and empties the cache", func(t *testing.T) {
		dir := t.TempDir()
		cached, original := filepath.Join(dir, "cached"), filepath.Join(dir, "a", "b", "link")
		symlink(t, "target", cached)

		if err := h.RestoreSymlink(cached, original); err != nil {
			t.Fatal(err)
		}
		wantLink(t, original, "target")
		wantGone(t, cached)
	})

	t.Run("keeps the cached link when the destination exists", func(t *testing.T) {
		dir := t.TempDir()
		cached, original := filepath.Join(dir, "cached"), filepath.Join(dir, "original")
		symlink(t, "target", cached)
		writeFile(t, original, "occupied")

		if err := h.RestoreSymlink(cached, original); err == nil {
			t.Error("expected an error")
		}
		wantLink(t, cached, "target")
		wantFile(t, original, "occupied")
	})
}

func TestIsSymlinkAndGetFileInfo(t *testing.T) {
	dir := t.TempDir()
	file, link := filepath.Join(dir, "file"), filepath.Join(dir, "link")
	writeFile(t, file, "x")
	symlink(t, "/nonexistent", link)

	for path, want := range map[string]bool{file: false, link: true} {
		if got, err := h.IsSymlink(path); err != nil || got != want {
			t.Errorf("IsSymlink(%s) = %v, %v; want %v", path, got, err, want)
		}
		if info, got, err := h.GetFileInfo(path); err != nil || got != want || info == nil {
			t.Errorf("GetFileInfo(%s) = %v, %v, %v; want %v", path, info, got, err, want)
		}
	}

	missing := filepath.Join(dir, "missing")
	if _, err := h.IsSymlink(missing); err == nil {
		t.Error("IsSymlink: expected an error for a missing path")
	}
	if _, _, err := h.GetFileInfo(missing); err == nil {
		t.Error("GetFileInfo: expected an error for a missing path")
	}
}

func TestMoveFileSafety(t *testing.T) {
	t.Run("preserves mode including read-only files", func(t *testing.T) {
		dir := t.TempDir()
		src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
		writeFile(t, src, "secret")
		if err := os.Chmod(src, 0400); err != nil {
			t.Fatal(err)
		}
		if err := h.MoveFile(src, dst); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(dst)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0400 {
			t.Errorf("mode = %o, want 400", info.Mode().Perm())
		}
		wantGone(t, src)
	})

	t.Run("keeps the source when the destination can't be created", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src")
		writeFile(t, src, "precious")
		if err := h.MoveFile(src, filepath.Join(dir, "missing", "dst")); err == nil {
			t.Error("expected an error")
		}
		wantFile(t, src, "precious")
	})

	t.Run("preserves modification time", func(t *testing.T) {
		dir := t.TempDir()
		src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
		writeFile(t, src, "x")
		old := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
		if err := os.Chtimes(src, old, old); err != nil {
			t.Fatal(err)
		}
		if err := h.MoveFile(src, dst); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(dst)
		if err != nil {
			t.Fatal(err)
		}
		if !info.ModTime().Equal(old) {
			t.Errorf("mtime = %v, want %v", info.ModTime(), old)
		}
	})

	t.Run("doesn't hang on a FIFO", func(t *testing.T) {
		dir := t.TempDir()
		src, dst := filepath.Join(dir, "fifo"), filepath.Join(dir, "dst")
		if err := syscall.Mkfifo(src, 0644); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- h.MoveFile(src, dst) }()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("MoveFile still blocked on the FIFO after 2s")
		}
		wantGone(t, src)
		if info, err := os.Lstat(dst); err != nil || info.Mode()&os.ModeNamedPipe == 0 {
			t.Errorf("FIFO not moved: %v, %v", info, err)
		}
	})

	t.Run("copying a FIFO errors instead of hanging", func(t *testing.T) {
		src, dst := filepath.Join(t.TempDir(), "src"), filepath.Join(t.TempDir(), "dst")
		if err := os.Mkdir(src, 0755); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(filepath.Join(src, "fifo"), 0644); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- h.CopyDirectory(src, dst) }()
		select {
		case err := <-done:
			if err == nil {
				t.Error("expected an error")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("CopyDirectory still blocked on the FIFO after 2s")
		}
	})
}

func TestCopyDirectoryPreservesMetadata(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	writeFile(t, filepath.Join(src, "sub", "file.sh"), "#!/bin/sh")
	writeFile(t, filepath.Join(src, ".hidden"), "h")
	symlink(t, "sub/file.sh", filepath.Join(src, "relative-link"))
	symlink(t, "/nonexistent", filepath.Join(src, "dangling-link"))
	if err := os.Mkdir(filepath.Join(src, "empty"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(src, "sub", "file.sh"), 0750); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-72 * time.Hour).Truncate(time.Second)
	for _, p := range []string{filepath.Join(src, "sub", "file.sh"), filepath.Join(src, "sub"), src} {
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}

	if err := h.CopyDirectory(src, dst); err != nil {
		t.Fatal(err)
	}

	wantFile(t, filepath.Join(dst, ".hidden"), "h")
	wantLink(t, filepath.Join(dst, "relative-link"), "sub/file.sh")
	wantLink(t, filepath.Join(dst, "dangling-link"), "/nonexistent")

	info, err := os.Stat(filepath.Join(dst, "sub", "file.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0750 {
		t.Errorf("file mode = %o, want 750", info.Mode().Perm())
	}
	for _, p := range []string{filepath.Join(dst, "sub", "file.sh"), filepath.Join(dst, "sub"), dst} {
		if info, err := os.Stat(p); err != nil || !info.ModTime().Equal(old) {
			t.Errorf("%s: mtime not preserved (%v, err=%v)", p, info, err)
		}
	}
	if info, err := os.Stat(filepath.Join(dst, "empty")); err != nil || info.Mode().Perm() != 0700 {
		t.Errorf("empty dir not copied with its mode: %v, %v", info, err)
	}
}

// Rename can't cross a mount, so these drive the real EXDEV copy-and-remove path.
func TestMovesAcrossFilesystems(t *testing.T) {
	shm, err := os.MkdirTemp("/dev/shm", "vanish-test-")
	if err != nil {
		t.Skipf("/dev/shm unavailable: %v", err)
	}
	t.Cleanup(func() {
		makeWritable(shm)
		os.RemoveAll(shm)
	})

	other := t.TempDir()
	var shmStat, otherStat syscall.Stat_t
	if syscall.Stat(shm, &shmStat) != nil || syscall.Stat(other, &otherStat) != nil || shmStat.Dev == otherStat.Dev {
		t.Skip("temp dir and /dev/shm are the same filesystem")
	}

	t.Run("directory with a symlink", func(t *testing.T) {
		src, dst := filepath.Join(shm, "dir"), filepath.Join(other, "dir")
		writeFile(t, filepath.Join(src, "sub", "file.txt"), "data")
		symlink(t, "sub/file.txt", filepath.Join(src, "link"))

		if err := h.MoveDirectory(src, dst); err != nil {
			t.Fatal(err)
		}
		wantGone(t, src)
		wantFile(t, filepath.Join(dst, "sub", "file.txt"), "data")
		wantLink(t, filepath.Join(dst, "link"), "sub/file.txt")
	})

	t.Run("read-only directory keeps its mode", func(t *testing.T) {
		src, dst := filepath.Join(shm, "ro"), filepath.Join(other, "ro")
		readOnlyTree(t, src)
		t.Cleanup(func() { makeWritable(dst) })

		if err := h.MoveDirectory(src, dst); err != nil {
			t.Fatal(err)
		}
		wantGone(t, src)
		wantFile(t, filepath.Join(dst, "sub", "file.txt"), "ro")
		if info, err := os.Stat(dst); err != nil || info.Mode().Perm() != 0555 {
			t.Errorf("mode not preserved: %v, %v", info, err)
		}
	})

	t.Run("file keeps its modification time", func(t *testing.T) {
		src, dst := filepath.Join(shm, "file"), filepath.Join(other, "file")
		writeFile(t, src, "data")
		old := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
		if err := os.Chtimes(src, old, old); err != nil {
			t.Fatal(err)
		}

		if err := h.MoveFile(src, dst); err != nil {
			t.Fatal(err)
		}
		wantGone(t, src)
		wantFile(t, dst, "data")
		if info, err := os.Stat(dst); err != nil || !info.ModTime().Equal(old) {
			t.Errorf("mtime not preserved: %v, %v", info, err)
		}
	})

	t.Run("FIFO errors and stays put", func(t *testing.T) {
		src := filepath.Join(shm, "fifo")
		if err := syscall.Mkfifo(src, 0644); err != nil {
			t.Fatal(err)
		}
		if err := h.MoveFile(src, filepath.Join(other, "fifo")); err == nil {
			t.Error("expected an error")
		}
		if _, err := os.Lstat(src); err != nil {
			t.Error(err)
		}
	})
}

func TestMoveDirectoryReadOnly(t *testing.T) {
	t.Run("failure keeps the source", func(t *testing.T) {
		dir := t.TempDir()
		src, blocker := filepath.Join(dir, "src"), filepath.Join(dir, "blocker")
		writeFile(t, filepath.Join(src, "file.txt"), "precious")
		writeFile(t, blocker, "not a dir")
		if err := h.MoveDirectory(src, filepath.Join(blocker, "dst")); err == nil {
			t.Error("expected an error")
		}
		wantFile(t, filepath.Join(src, "file.txt"), "precious")
	})

	t.Run("moves a 0555 tree to another parent", func(t *testing.T) {
		src, dst := filepath.Join(t.TempDir(), "src"), filepath.Join(t.TempDir(), "dst")
		readOnlyTree(t, src)
		t.Cleanup(func() { makeWritable(dst) })

		if err := h.MoveDirectory(src, dst); err != nil {
			t.Fatal(err)
		}
		wantGone(t, src)
		wantFile(t, filepath.Join(dst, "sub", "file.txt"), "ro")
	})
}

// `vx ~/.cache` asks to move the trash into itself.
func TestMoveDirectoryIntoItself(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	writeFile(t, filepath.Join(src, "keep.txt"), "precious")

	if err := h.MoveDirectory(src, filepath.Join(src, "cache")); err == nil {
		t.Error("expected an error")
	}
	wantFile(t, filepath.Join(src, "keep.txt"), "precious")

	t.Run("leaves no partial copy behind", func(t *testing.T) {
		wantGone(t, filepath.Join(src, "cache"))
	})
}

func TestRemoveAllWritable(t *testing.T) {
	t.Run("removes a read-only tree", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "root")
		readOnlyTree(t, root)
		if err := h.RemoveAllWritable(root); err != nil {
			t.Fatal(err)
		}
		wantGone(t, root)
	})

	t.Run("a missing path is not an error", func(t *testing.T) {
		if err := h.RemoveAllWritable(filepath.Join(t.TempDir(), "missing")); err != nil {
			t.Error(err)
		}
	})

	t.Run("removes a symlink without following it", func(t *testing.T) {
		dir := t.TempDir()
		target, link := filepath.Join(dir, "target", "file"), filepath.Join(dir, "link")
		writeFile(t, target, "precious")
		symlink(t, filepath.Dir(target), link)
		if err := h.RemoveAllWritable(link); err != nil {
			t.Fatal(err)
		}
		wantGone(t, link)
		wantFile(t, target, "precious")
	})
}

func TestRestorePathConflicts(t *testing.T) {
	tests := []struct {
		name     string
		original string
		existing []string
		want     string
	}{
		{"no conflict", "a.txt", nil, "a.txt"},
		{"file with extension", "a.txt", []string{"a.txt"}, "a1.txt"},
		{"file without extension", "a", []string{"a"}, "a1"},
		{"skips taken numbers", "a.txt", []string{"a.txt", "a1.txt", "a2.txt"}, "a3.txt"},
		{"only the last extension moves", "a.tar.gz", []string{"a.tar.gz"}, "a.tar1.gz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cfg := testCfg(t)
			for _, name := range tt.existing {
				writeFile(t, filepath.Join(dir, name), "existing")
			}
			saveItems(t, cfg, types.DeletedItem{ID: "1", OriginalPath: filepath.Join(dir, tt.original)})

			msg := h.CheckRestoreItems([]string{tt.original}, cfg)().(types.RestoreItemsMsg)
			if len(msg.Items) != 1 {
				t.Fatalf("got %d items, want 1", len(msg.Items))
			}
			if got := filepath.Base(msg.Items[0].OriginalPath); got != tt.want {
				t.Errorf("resolved to %s, want %s", got, tt.want)
			}
		})
	}

	t.Run("a dangling symlink counts as a conflict", func(t *testing.T) {
		dir := t.TempDir()
		cfg := testCfg(t)
		original := filepath.Join(dir, "a.txt")
		symlink(t, "/nonexistent", original)
		saveItems(t, cfg, types.DeletedItem{ID: "1", OriginalPath: original})

		msg := h.CheckRestoreItems([]string{"a.txt"}, cfg)().(types.RestoreItemsMsg)
		if len(msg.Items) != 1 || msg.Items[0].OriginalPath == original {
			t.Errorf("resolved to the occupied path: %+v", msg.Items)
		}
	})

	t.Run("doesn't hang when the parent became a file", func(t *testing.T) {
		dir := t.TempDir()
		cfg := testCfg(t)
		writeFile(t, filepath.Join(dir, "parent"), "now a file")
		saveItems(t, cfg, types.DeletedItem{ID: "1", OriginalPath: filepath.Join(dir, "parent", "a.txt")})

		done := make(chan tea.Msg, 1)
		go func() { done <- h.CheckRestoreItems([]string{"a.txt"}, cfg)() }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("still searching for a free name after 2s")
		}
	})
}

func TestCheckRestoreItemsMatching(t *testing.T) {
	cfg := testCfg(t)
	saveItems(t, cfg,
		types.DeletedItem{ID: "1", OriginalPath: "/nonexistent/Report.PDF"},
		types.DeletedItem{ID: "2", OriginalPath: "/nonexistent/notes.txt"},
		types.DeletedItem{ID: "3", OriginalPath: "/nonexistent/todo.txt"},
	)
	match := func(patterns ...string) []types.DeletedItem {
		return h.CheckRestoreItems(patterns, cfg)().(types.RestoreItemsMsg).Items
	}

	if got := match("report.pdf"); len(got) != 1 || got[0].ID != "1" {
		t.Errorf("case-insensitive match failed: %+v", got)
	}
	if got := match("absent"); len(got) != 0 {
		t.Errorf("expected no matches, got %+v", got)
	}
	if got := match("notes", "todo"); len(got) != 2 {
		t.Errorf("expected 2 matches for 2 patterns, got %+v", got)
	}

	t.Run("README globs match", func(t *testing.T) {
		if got := match("*.txt"); len(got) != 2 {
			t.Errorf("expected 2 matches for *.txt, got %+v", got)
		}
		if got := match("*.PDF"); len(got) != 1 {
			t.Errorf("expected a case-insensitive glob match, got %+v", got)
		}
		if got := match("not*"); len(got) != 1 || got[0].ID != "2" {
			t.Errorf("not* matched %+v", got)
		}
		if got := match("nonexistent*"); len(got) != 0 {
			t.Errorf("a glob must match the base name, not the directory: %+v", got)
		}
	})

	t.Run("a malformed glob still works as a literal", func(t *testing.T) {
		cfg := testCfg(t)
		saveItems(t, cfg, types.DeletedItem{ID: "1", OriginalPath: "/nonexistent/photo[1].jpg"})
		for _, pattern := range []string{"photo[1].jpg", "photo["} {
			if got := h.CheckRestoreItems([]string{pattern}, cfg)().(types.RestoreItemsMsg).Items; len(got) != 1 {
				t.Errorf("%q matched %+v", pattern, got)
			}
		}
	})

	t.Run("overlapping patterns match an item once", func(t *testing.T) {
		if got := match("notes", ".txt"); len(got) != 2 {
			t.Errorf("expected notes.txt and todo.txt once each, got %d items", len(got))
		}
	})

	t.Run("a corrupt index is reported", func(t *testing.T) {
		cfg := testCfg(t)
		writeFile(t, h.GetIndexPath(cfg), "{not json")
		if _, ok := h.CheckRestoreItems([]string{"x"}, cfg)().(types.ErrorMsg); !ok {
			t.Error("expected ErrorMsg")
		}
	})
}

func TestIndexOperations(t *testing.T) {
	t.Run("RemoveFromIndex removes only the matching ID", func(t *testing.T) {
		cfg := testCfg(t)
		saveItems(t, cfg, types.DeletedItem{ID: "a"}, types.DeletedItem{ID: "b"})

		if err := h.RemoveFromIndex("a", cfg); err != nil {
			t.Fatal(err)
		}
		if err := h.RemoveFromIndex("absent", cfg); err != nil {
			t.Fatal(err)
		}
		if items := loadItems(t, cfg); len(items) != 1 || items[0].ID != "b" {
			t.Errorf("items = %+v", items)
		}
	})

	t.Run("a corrupt index is never overwritten", func(t *testing.T) {
		cfg := testCfg(t)
		writeFile(t, h.GetIndexPath(cfg), "{not json")

		if _, err := h.LoadIndex(cfg); err == nil {
			t.Error("LoadIndex: expected an error")
		}
		if err := h.AddToIndex(types.DeletedItem{ID: "a"}, cfg); err == nil {
			t.Error("AddToIndex: expected an error")
		}
		if err := h.RemoveFromIndex("a", cfg); err == nil {
			t.Error("RemoveFromIndex: expected an error")
		}
		wantFile(t, h.GetIndexPath(cfg), "{not json")
	})

	t.Run("concurrent adds lose nothing", func(t *testing.T) {
		cfg := testCfg(t)
		var wg sync.WaitGroup
		for i := range 50 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := h.AddToIndex(types.DeletedItem{ID: fmt.Sprint(i)}, cfg); err != nil {
					t.Error(err)
				}
			}()
		}
		wg.Wait()
		if items := loadItems(t, cfg); len(items) != 50 {
			t.Errorf("got %d items, want 50", len(items))
		}
	})

	t.Run("metadata survives a round trip", func(t *testing.T) {
		cfg := testCfg(t)
		now := time.Now().Truncate(time.Second)
		item := types.DeletedItem{ID: "1", OriginalPath: "/a b/ünï\ncødé", DeleteDate: now, CachePath: "/c", IsSymlink: true, LinkTarget: "/t", FileCount: 3, Size: 7}
		saveItems(t, cfg, item)
		got := loadItems(t, cfg)
		if len(got) != 1 || !got[0].DeleteDate.Equal(now) {
			t.Fatalf("items = %+v", got)
		}
		got[0].DeleteDate = now
		if got[0] != item {
			t.Errorf("got %+v, want %+v", got[0], item)
		}
	})
}

func TestLogging(t *testing.T) {
	readLog := func(t *testing.T, cfg types.Config) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(cfg.Logging.Directory, "vanish.log"))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	t.Run("disabled writes nothing", func(t *testing.T) {
		cfg := testCfg(t)
		if err := h.LogOperation("DELETE", types.DeletedItem{}, cfg); err != nil {
			t.Error(err)
		}
		if err := h.LogSimpleOperation("OP", "msg", cfg); err != nil {
			t.Error(err)
		}
		wantGone(t, cfg.Logging.Directory)
	})

	t.Run("entries append and create the directory", func(t *testing.T) {
		cfg := testCfg(t)
		cfg.Logging.Enabled = true
		h.LogOperation("DELETE", types.DeletedItem{OriginalPath: "/o/file", CachePath: "/c/file"}, cfg)
		h.LogOperation("RESTORE", types.DeletedItem{OriginalPath: "/o/dir", CachePath: "/c/dir", IsDirectory: true}, cfg)
		h.LogSimpleOperation("NOTE", "hello", cfg)

		log := readLog(t, cfg)
		for _, want := range []string{"[FILE] DELETE: /o/file -> /c/file\n", "[DIR] RESTORE: /o/dir -> /c/dir\n", "[NOTE] hello\n"} {
			if !strings.Contains(log, want) {
				t.Errorf("log missing %q:\n%s", want, log)
			}
		}
	})

	t.Run("reports an unusable directory", func(t *testing.T) {
		cfg := testCfg(t)
		cfg.Logging.Enabled = true
		blocker := filepath.Join(t.TempDir(), "file")
		writeFile(t, blocker, "not a dir")
		cfg.Logging.Directory = filepath.Join(blocker, "logs")
		if err := h.LogOperation("DELETE", types.DeletedItem{}, cfg); err == nil {
			t.Error("LogOperation: expected an error")
		}
		if err := h.LogSimpleOperation("OP", "m", cfg); err == nil {
			t.Error("LogSimpleOperation: expected an error")
		}
	})

	t.Run("ClearAllCache logs and reports log failures", func(t *testing.T) {
		cfg := testCfg(t)
		cfg.Logging.Enabled = true
		if msg := h.ClearAllCache(cfg)().(types.ClearMsg); msg.Err != nil {
			t.Fatal(msg.Err)
		}
		if log := readLog(t, cfg); !strings.Contains(log, "CLEAR_ALL") {
			t.Errorf("log = %q", log)
		}

		blocker := filepath.Join(t.TempDir(), "file")
		writeFile(t, blocker, "not a dir")
		cfg.Logging.Directory = filepath.Join(blocker, "logs")
		if msg := h.ClearAllCache(cfg)().(types.ClearMsg); msg.Err == nil {
			t.Error("expected the logging failure to be reported")
		}
	})
}

func TestClearAllCacheReadOnlyTree(t *testing.T) {
	cfg := testCfg(t)
	readOnlyTree(t, filepath.Join(cfg.Cache.Directory, "item"))

	if msg := h.ClearAllCache(cfg)().(types.ClearMsg); msg.Err != nil {
		t.Fatal(msg.Err)
	}
	wantGone(t, filepath.Join(cfg.Cache.Directory, "item"))
	if items := loadItems(t, cfg); len(items) != 0 {
		t.Errorf("items = %+v", items)
	}
}

func TestPurgeOldFilesEdgeCases(t *testing.T) {
	old := time.Now().Add(-30 * 24 * time.Hour)
	purge := func(cfg types.Config, days string) types.PurgeMsg {
		return h.PurgeOldFiles(cfg, days)().(types.PurgeMsg)
	}

	t.Run("huge day counts don't overflow into a future cutoff", func(t *testing.T) {
		for _, days := range []string{"106752", "999999999", "9223372036854775807"} {
			cfg := testCfg(t)
			file := filepath.Join(cfg.Cache.Directory, "item")
			writeFile(t, file, "x")
			saveItems(t, cfg, types.DeletedItem{ID: "1", CachePath: file, DeleteDate: old})

			purge(cfg, days)
			wantFile(t, file, "x")
		}
	})

	for _, days := range []string{"abc", "", "0", "-1", "1.5"} {
		t.Run("rejects days="+days+" without touching anything", func(t *testing.T) {
			cfg := testCfg(t)
			file := filepath.Join(cfg.Cache.Directory, "item")
			writeFile(t, file, "x")
			saveItems(t, cfg, types.DeletedItem{ID: "1", CachePath: file, DeleteDate: old})

			if msg := purge(cfg, days); msg.Err == nil {
				t.Error("expected an error")
			}
			wantFile(t, file, "x")
			if len(loadItems(t, cfg)) != 1 {
				t.Error("index changed")
			}
		})
	}

	t.Run("removes directories, read-only trees and already-missing items", func(t *testing.T) {
		cfg := testCfg(t)
		dir := filepath.Join(cfg.Cache.Directory, "dir")
		ro := filepath.Join(cfg.Cache.Directory, "ro")
		writeFile(t, filepath.Join(dir, "file"), "x")
		readOnlyTree(t, ro)
		saveItems(t, cfg,
			types.DeletedItem{ID: "dir", CachePath: dir, DeleteDate: old, IsDirectory: true},
			types.DeletedItem{ID: "ro", CachePath: ro, DeleteDate: old, IsDirectory: true},
			types.DeletedItem{ID: "missing", CachePath: filepath.Join(cfg.Cache.Directory, "missing"), DeleteDate: old},
		)

		msg := purge(cfg, "7")
		if msg.Err != nil || msg.PurgedCount != 3 {
			t.Errorf("msg = %+v, want 3 purged", msg)
		}
		wantGone(t, dir)
		wantGone(t, ro)
		if items := loadItems(t, cfg); len(items) != 0 {
			t.Errorf("items = %+v", items)
		}
	})

	t.Run("keeps the index entry when removal fails", func(t *testing.T) {
		cfg := testCfg(t)
		locked := filepath.Join(cfg.Cache.Directory, "locked")
		readOnlyTree(t, locked)
		cached := filepath.Join(locked, "sub", "file.txt")
		saveItems(t, cfg, types.DeletedItem{ID: "1", CachePath: cached, DeleteDate: old})

		msg := purge(cfg, "7")
		if msg.Err == nil || msg.PurgedCount != 0 {
			t.Errorf("msg = %+v, want an error and 0 purged", msg)
		}
		wantFile(t, cached, "ro")
		if len(loadItems(t, cfg)) != 1 {
			t.Error("entry was dropped from the index while its file still exists")
		}
	})

	t.Run("logs each purge", func(t *testing.T) {
		cfg := testCfg(t)
		cfg.Logging.Enabled = true
		saveItems(t, cfg, types.DeletedItem{ID: "1", OriginalPath: "/o/file", CachePath: filepath.Join(cfg.Cache.Directory, "x"), DeleteDate: old})
		if msg := purge(cfg, "7"); msg.Err != nil {
			t.Fatal(msg.Err)
		}
		log, _ := os.ReadFile(filepath.Join(cfg.Logging.Directory, "vanish.log"))
		if !strings.Contains(string(log), "PURGE: /o/file") {
			t.Errorf("log = %q", log)
		}
	})
}

func TestCleanupExpired(t *testing.T) {
	old := time.Now().Add(-30 * 24 * time.Hour)

	t.Run("removes expired read-only trees and logs them", func(t *testing.T) {
		cfg := testCfg(t)
		cfg.Logging.Enabled = true
		ro := filepath.Join(cfg.Cache.Directory, "ro")
		readOnlyTree(t, ro)
		saveItems(t, cfg, types.DeletedItem{ID: "1", OriginalPath: "/o/ro", CachePath: ro, DeleteDate: old, IsDirectory: true})

		if n, err := h.CleanupExpired(cfg); n != 1 || err != nil {
			t.Errorf("CleanupExpired = %d, %v", n, err)
		}
		wantGone(t, ro)
		log, _ := os.ReadFile(filepath.Join(cfg.Logging.Directory, "vanish.log"))
		if !strings.Contains(string(log), "CLEANUP: /o/ro") {
			t.Errorf("log = %q", log)
		}
	})

	for _, days := range []int{0, -1, math.MaxInt} {
		t.Run(fmt.Sprintf("days=%d expires nothing", days), func(t *testing.T) {
			cfg := testCfg(t)
			cfg.Cache.Days = days
			file := filepath.Join(cfg.Cache.Directory, "item")
			writeFile(t, file, "x")
			saveItems(t, cfg, types.DeletedItem{ID: "1", CachePath: file, DeleteDate: old})

			if n, err := h.CleanupExpired(cfg); n != 0 || err != nil {
				t.Errorf("CleanupExpired = %d, %v", n, err)
			}
			wantFile(t, file, "x")
		})
	}
}

func TestPathsWithoutHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	if got := h.ExpandPath("~/x"); got != "~/x" {
		t.Errorf("ExpandPath(~/x) = %s, want it unchanged", got)
	}
	abs, err := filepath.Abs("x")
	if err != nil {
		t.Fatal(err)
	}
	if got := h.ExpandPath("x"); got != abs {
		t.Errorf("ExpandPath(x) = %s, want %s", got, abs)
	}
	if got := h.GetConfigPath(); got != filepath.Join("vanish", "vanish.toml") {
		t.Errorf("GetConfigPath() = %s", got)
	}
}

func TestGetConfigPathRespectsXDG(t *testing.T) {
	t.Setenv("HOME", "/home/someone")

	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	if got := h.GetConfigPath(); got != "/xdg/vanish/vanish.toml" {
		t.Errorf("with XDG_CONFIG_HOME: %s", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	if got := h.GetConfigPath(); got != "/home/someone/.config/vanish/vanish.toml" {
		t.Errorf("without XDG_CONFIG_HOME: %s", got)
	}
}
