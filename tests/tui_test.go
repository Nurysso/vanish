package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	h "vanish/internal/helpers"
	"vanish/internal/tui"
	"vanish/internal/types"
)

func newModel(cfg types.Config, op string, noConfirm bool, names ...string) *tui.Model {
	return &tui.Model{
		Filenames:  names,
		FileInfos:  make([]types.FileInfo, len(names)),
		State:      "checking",
		Progress:   h.SetUpProgress(cfg),
		Config:     cfg,
		Styles:     h.CreateThemeStyles(cfg),
		Operation:  op,
		NoConfirm:  noConfirm,
		TotalFiles: len(names),
	}
}

// drive stands in for the bubbletea runtime: it runs commands and feeds their
// messages to Update until none are left. Animation frames are dropped.
func drive(m *tui.Model, cmd tea.Cmd) {
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if next == nil {
			continue
		}
		switch msg := next().(type) {
		case tea.BatchMsg:
			queue = append(queue, msg...)
		case tea.QuitMsg, progress.FrameMsg:
		default:
			_, c := m.Update(msg)
			queue = append(queue, c)
		}
	}
}

func press(m *tui.Model, key string) {
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	drive(m, cmd)
}

func run(t *testing.T, cfg types.Config, op string, names ...string) *tui.Model {
	t.Helper()
	m := newModel(cfg, op, true, names...)
	drive(m, m.Init())
	return m
}

func wantDone(t *testing.T, m *tui.Model) {
	t.Helper()
	if m.State != "done" {
		t.Fatalf("state = %q, error = %q; want done", m.State, m.ErrorMsg)
	}
}

func TestTUIConfirmation(t *testing.T) {
	setup := func(t *testing.T) (*tui.Model, types.Config, string) {
		cfg := testCfg(t)
		file := filepath.Join(t.TempDir(), "file.txt")
		writeFile(t, file, "data")
		m := newModel(cfg, "delete", false, file)
		drive(m, m.Init())
		if m.State != "confirming" {
			t.Fatalf("state = %q, want confirming", m.State)
		}
		return m, cfg, file
	}

	t.Run("y deletes", func(t *testing.T) {
		m, cfg, file := setup(t)
		press(m, "y")
		wantDone(t, m)
		wantGone(t, file)
		if len(loadItems(t, cfg)) != 1 {
			t.Error("item not indexed")
		}
	})

	for _, key := range []string{"n", "q"} {
		t.Run(key+" leaves the file alone", func(t *testing.T) {
			m, cfg, file := setup(t)
			press(m, key)
			wantFile(t, file, "data")
			if m.Confirmed || len(loadItems(t, cfg)) != 0 {
				t.Error("cancelling changed state")
			}
		})
	}

	t.Run("y does nothing outside the prompt", func(t *testing.T) {
		cfg := testCfg(t)
		file := filepath.Join(t.TempDir(), "file.txt")
		writeFile(t, file, "data")
		m := newModel(cfg, "delete", false, file) // still "checking"
		press(m, "y")
		wantFile(t, file, "data")
	})
}

func TestTUIDeleteRestoreRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, path string)
		check func(t *testing.T, path string)
		want  types.DeletedItem // only the metadata fields below are compared
	}{
		{
			name: "file.txt",
			setup: func(t *testing.T, path string) {
				writeFile(t, path, "data")
				os.Chmod(path, 0640)
			},
			check: func(t *testing.T, path string) {
				wantFile(t, path, "data")
				if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0640 {
					t.Errorf("mode not restored: %v, %v", info, err)
				}
			},
			want: types.DeletedItem{Size: 4},
		},
		{
			name:  "empty",
			setup: func(t *testing.T, path string) { writeFile(t, path, "") },
			check: func(t *testing.T, path string) { wantFile(t, path, "") },
		},
		{
			name:  "spaces & ünïcødé\nnewline.txt",
			setup: func(t *testing.T, path string) { writeFile(t, path, "data") },
			check: func(t *testing.T, path string) { wantFile(t, path, "data") },
			want:  types.DeletedItem{Size: 4},
		},
		{
			name: "dir",
			setup: func(t *testing.T, path string) {
				writeFile(t, filepath.Join(path, "sub", "file"), "abc")
				writeFile(t, filepath.Join(path, ".hidden"), "de")
				symlink(t, "sub/file", filepath.Join(path, "link"))
			},
			check: func(t *testing.T, path string) {
				wantFile(t, filepath.Join(path, "sub", "file"), "abc")
				wantFile(t, filepath.Join(path, ".hidden"), "de")
				wantLink(t, filepath.Join(path, "link"), "sub/file")
			},
			want: types.DeletedItem{IsDirectory: true, FileCount: 3},
		},
		{
			name: "emptydir",
			setup: func(t *testing.T, path string) {
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
			},
			check: func(t *testing.T, path string) {
				if info, err := os.Stat(path); err != nil || !info.IsDir() {
					t.Errorf("not restored: %v, %v", info, err)
				}
			},
			want: types.DeletedItem{IsDirectory: true},
		},
		{
			name: "link-to-dir",
			setup: func(t *testing.T, path string) {
				target := filepath.Join(filepath.Dir(path), "target")
				writeFile(t, filepath.Join(target, "file"), "precious")
				symlink(t, target, path)
			},
			check: func(t *testing.T, path string) {
				target := filepath.Join(filepath.Dir(path), "target")
				wantLink(t, path, target)
				wantFile(t, filepath.Join(target, "file"), "precious")
			},
			want: types.DeletedItem{IsSymlink: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testCfg(t)
			path := filepath.Join(t.TempDir(), tt.name)
			tt.setup(t, path)

			wantDone(t, run(t, cfg, "delete", path))
			wantGone(t, path)

			items := loadItems(t, cfg)
			if len(items) != 1 {
				t.Fatalf("got %d index items, want 1", len(items))
			}
			item := items[0]
			if item.OriginalPath != path {
				t.Errorf("OriginalPath = %s, want %s", item.OriginalPath, path)
			}
			if item.IsDirectory != tt.want.IsDirectory || item.IsSymlink != tt.want.IsSymlink || item.FileCount != tt.want.FileCount {
				t.Errorf("metadata = %+v, want %+v", item, tt.want)
			}
			if tt.want.Size != 0 && item.Size != tt.want.Size {
				t.Errorf("Size = %d, want %d", item.Size, tt.want.Size)
			}
			if _, err := os.Lstat(item.CachePath); err != nil {
				t.Errorf("not in cache: %v", err)
			}

			wantDone(t, run(t, cfg, "restore", tt.name))
			tt.check(t, path)
			wantGone(t, item.CachePath)
			if left := loadItems(t, cfg); len(left) != 0 {
				t.Errorf("index not emptied: %+v", left)
			}
		})
	}

	t.Run("dangling symlink", func(t *testing.T) {
		cfg := testCfg(t)
		path := filepath.Join(t.TempDir(), "dangling")
		symlink(t, "/nonexistent", path)

		wantDone(t, run(t, cfg, "delete", path))
		wantGone(t, path)
		wantDone(t, run(t, cfg, "restore", "dangling"))
		wantLink(t, path, "/nonexistent")
	})

	t.Run("name too long for the cache prefix", func(t *testing.T) {
		cfg := testCfg(t)
		name := strings.Repeat("x", 250)
		path := filepath.Join(t.TempDir(), name)
		writeFile(t, path, "data")

		wantDone(t, run(t, cfg, "delete", path))
		wantGone(t, path)
		wantDone(t, run(t, cfg, "restore", name))
		wantFile(t, path, "data")
	})

	t.Run("non-UTF-8 name", func(t *testing.T) {
		knownBug(t, "the index is JSON, which replaces invalid UTF-8 in OriginalPath and CachePath, so the cached file can't be found again")
		cfg := testCfg(t)
		path := filepath.Join(t.TempDir(), "bad\xff.txt")
		writeFile(t, path, "data")

		wantDone(t, run(t, cfg, "delete", path))
		wantDone(t, run(t, cfg, "restore", "bad"))
		wantFile(t, path, "data")
	})
}

// A file the index doesn't know about can never be restored, so a failed index
// update must put everything back where it was.
func TestTUIDeleteUndoesMoveWhenIndexFails(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, path string)
		check func(t *testing.T, path string)
	}{
		{"file",
			func(t *testing.T, path string) { writeFile(t, path, "data") },
			func(t *testing.T, path string) { wantFile(t, path, "data") }},
		{"directory",
			func(t *testing.T, path string) { writeFile(t, filepath.Join(path, "f"), "data") },
			func(t *testing.T, path string) { wantFile(t, filepath.Join(path, "f"), "data") }},
		{"symlink",
			func(t *testing.T, path string) { symlink(t, "/nonexistent", path) },
			func(t *testing.T, path string) { wantLink(t, path, "/nonexistent") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testCfg(t)
			writeFile(t, h.GetIndexPath(cfg), "{not json")
			path := filepath.Join(t.TempDir(), "item")
			tt.setup(t, path)

			m := run(t, cfg, "delete", path)
			if m.State != "error" || !strings.Contains(m.ErrorMsg, "file left in place") {
				t.Errorf("state = %q, error = %q", m.State, m.ErrorMsg)
			}
			tt.check(t, path)
			wantFile(t, h.GetIndexPath(cfg), "{not json")
		})
	}
}

func TestTUIDeleteConfirmSkipsLeadingMissingFile(t *testing.T) {
	cfg := testCfg(t)
	real := filepath.Join(t.TempDir(), "real")
	writeFile(t, real, "x")

	m := newModel(cfg, "delete", false, "/nonexistent/a", real)
	drive(m, m.Init())
	press(m, "y")
	wantDone(t, m)
	wantGone(t, real)
}

func TestTUIDeleteRefusesMovingDirectoryIntoCache(t *testing.T) {
	cfg := testCfg(t)
	parent := filepath.Dir(cfg.Cache.Directory)
	writeFile(t, filepath.Join(parent, "keep"), "precious")

	m := run(t, cfg, "delete", parent)
	if m.State != "error" {
		t.Errorf("state = %q, want error", m.State)
	}
	wantFile(t, filepath.Join(parent, "keep"), "precious")
	if entries, _ := os.ReadDir(cfg.Cache.Directory); len(entries) != 0 {
		t.Errorf("cache not empty: %v", entries)
	}
}

func TestTUIDeleteEdgeCases(t *testing.T) {
	t.Run("only missing files is an error", func(t *testing.T) {
		m := run(t, testCfg(t), "delete", "/nonexistent/a")
		if m.State != "error" {
			t.Errorf("state = %q, want error", m.State)
		}
	})

	t.Run("a missing file after a real one", func(t *testing.T) {
		cfg := testCfg(t)
		real := filepath.Join(t.TempDir(), "real")
		writeFile(t, real, "x")
		wantDone(t, run(t, cfg, "delete", real, "/nonexistent/a"))
		wantGone(t, real)
	})

	t.Run("several files", func(t *testing.T) {
		cfg := testCfg(t)
		dir := t.TempDir()
		a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
		writeFile(t, a, "a")
		writeFile(t, b, "b")
		m := run(t, cfg, "delete", a, b)
		wantDone(t, m)
		if m.ProcessedFiles != 2 || len(loadItems(t, cfg)) != 2 {
			t.Errorf("processed %d, indexed %d", m.ProcessedFiles, len(loadItems(t, cfg)))
		}
	})

	t.Run("a missing file before a real one", func(t *testing.T) {
		cfg := testCfg(t)
		real := filepath.Join(t.TempDir(), "real")
		writeFile(t, real, "x")
		wantDone(t, run(t, cfg, "delete", "/nonexistent/a", real))
		wantGone(t, real)
	})

	t.Run("cleanup expires only old items", func(t *testing.T) {
		cfg := testCfg(t)
		oldFile := filepath.Join(cfg.Cache.Directory, "old")
		oldDir := filepath.Join(cfg.Cache.Directory, "olddir")
		recent := filepath.Join(cfg.Cache.Directory, "recent")
		writeFile(t, oldFile, "x")
		writeFile(t, filepath.Join(oldDir, "f"), "x")
		writeFile(t, recent, "x")
		expired := time.Now().Add(-11 * 24 * time.Hour)
		saveItems(t, cfg,
			types.DeletedItem{ID: "old", CachePath: oldFile, DeleteDate: expired},
			types.DeletedItem{ID: "olddir", CachePath: oldDir, DeleteDate: expired, IsDirectory: true},
			types.DeletedItem{ID: "recent", CachePath: recent, DeleteDate: time.Now().Add(-9 * 24 * time.Hour)},
		)
		file := filepath.Join(t.TempDir(), "new")
		writeFile(t, file, "x")

		wantDone(t, run(t, cfg, "delete", file))
		wantGone(t, oldFile)
		wantGone(t, oldDir)
		wantFile(t, recent, "x")
		if items := loadItems(t, cfg); len(items) != 2 {
			t.Errorf("items = %+v, want recent and new", items)
		}
	})

	t.Run("a move failure stops and keeps the source", func(t *testing.T) {
		cfg := testCfg(t)
		blocker := filepath.Join(t.TempDir(), "file")
		writeFile(t, blocker, "not a dir")
		cfg.Cache.Directory = filepath.Join(blocker, "cache")
		file := filepath.Join(t.TempDir(), "keep")
		writeFile(t, file, "precious")

		m := run(t, cfg, "delete", file)
		if m.State != "error" {
			t.Errorf("state = %q, want error", m.State)
		}
		wantFile(t, file, "precious")
	})
}

func TestTUIRestoreEdgeCases(t *testing.T) {
	deleted := func(t *testing.T, cfg types.Config, path, content string) {
		t.Helper()
		writeFile(t, path, content)
		wantDone(t, run(t, cfg, "delete", path))
	}

	t.Run("doesn't overwrite a file recreated at the original path", func(t *testing.T) {
		cfg := testCfg(t)
		path := filepath.Join(t.TempDir(), "a.txt")
		deleted(t, cfg, path, "old")
		writeFile(t, path, "new")

		wantDone(t, run(t, cfg, "restore", "a.txt"))
		wantFile(t, path, "new")
		wantFile(t, filepath.Join(filepath.Dir(path), "a1.txt"), "old")
	})

	t.Run("recreates a deleted parent directory", func(t *testing.T) {
		cfg := testCfg(t)
		parent := filepath.Join(t.TempDir(), "parent")
		path := filepath.Join(parent, "a.txt")
		deleted(t, cfg, path, "data")
		if err := os.RemoveAll(parent); err != nil {
			t.Fatal(err)
		}

		wantDone(t, run(t, cfg, "restore", "a.txt"))
		wantFile(t, path, "data")
	})

	t.Run("no match is an error", func(t *testing.T) {
		if m := run(t, testCfg(t), "restore", "absent"); m.State != "error" {
			t.Errorf("state = %q, want error", m.State)
		}
	})

	t.Run("a missing cache file is an error and keeps the entry", func(t *testing.T) {
		cfg := testCfg(t)
		saveItems(t, cfg, types.DeletedItem{ID: "1", OriginalPath: "/nonexistent/a.txt", CachePath: filepath.Join(cfg.Cache.Directory, "gone")})

		m := run(t, cfg, "restore", "a.txt")
		if m.State != "error" || !strings.Contains(m.ErrorMsg, "cached file not found") {
			t.Errorf("state = %q, error = %q", m.State, m.ErrorMsg)
		}
		if len(loadItems(t, cfg)) != 1 {
			t.Error("entry was dropped")
		}
	})

	t.Run("two versions of one path both restore", func(t *testing.T) {
		cfg := testCfg(t)
		path := filepath.Join(t.TempDir(), "a.txt")
		deleted(t, cfg, path, "first")
		deleted(t, cfg, path, "second")

		wantDone(t, run(t, cfg, "restore", "a.txt"))
		wantFile(t, path, "first")
		wantFile(t, filepath.Join(filepath.Dir(path), "a1.txt"), "second")
	})
}

func TestTUIOtherOperations(t *testing.T) {
	t.Run("clear", func(t *testing.T) {
		cfg := testCfg(t)
		file := filepath.Join(cfg.Cache.Directory, "item")
		writeFile(t, file, "x")
		saveItems(t, cfg, types.DeletedItem{ID: "1", CachePath: file})

		wantDone(t, run(t, cfg, "clear", ""))
		wantGone(t, file)
		if len(loadItems(t, cfg)) != 0 {
			t.Error("index not emptied")
		}
	})

	t.Run("purge reports its count", func(t *testing.T) {
		cfg := testCfg(t)
		file := filepath.Join(cfg.Cache.Directory, "item")
		writeFile(t, file, "x")
		saveItems(t, cfg, types.DeletedItem{ID: "1", CachePath: file, DeleteDate: time.Now().Add(-30 * 24 * time.Hour)})

		m := run(t, cfg, "purge", "7")
		wantDone(t, m)
		if m.ProcessedFiles != 1 {
			t.Errorf("ProcessedFiles = %d, want 1", m.ProcessedFiles)
		}
	})

	t.Run("purge with bad days shows an error", func(t *testing.T) {
		if m := run(t, testCfg(t), "purge", "0"); m.State != "error" {
			t.Errorf("state = %q, want error", m.State)
		}
	})
}

// Rendering is checked only for panics and key text, not layout.
func TestTUIViewRendersEveryState(t *testing.T) {
	item := types.DeletedItem{ID: "1", OriginalPath: "/o/file.txt", CachePath: "/c/file.txt", DeleteDate: time.Now()}
	infos := []types.FileInfo{
		{Path: "/o/file.txt", Exists: true},
		{Path: "/o/dir", Exists: true, IsDirectory: true, FileCount: 2},
		{Path: "/o/missing", Error: "does not exist"},
	}

	for _, emoji := range []bool{true, false} {
		for _, op := range []string{"delete", "restore", "clear", "purge"} {
			for _, state := range []string{"checking", "confirming", "moving", "restoring", "cleanup", "clearing", "purging", "done", "error"} {
				cfg := testCfg(t)
				cfg.UI.Progress.ShowEmoji = emoji
				m := newModel(cfg, op, false, "/o/file.txt", "/o/dir", "/o/missing")
				m.State, m.FileInfos, m.ErrorMsg = state, infos, "boom"
				m.RestoreItems = []types.DeletedItem{item}
				m.ProcessedItems = []types.DeletedItem{item}
				m.ProcessedFiles = 1

				view := m.View()
				if view == "" {
					t.Errorf("%s/%s: empty view", op, state)
				}
				if state == "error" && !strings.Contains(view, "boom") {
					t.Errorf("%s: error text missing:\n%s", op, view)
				}
				if state == "confirming" && (op == "delete" || op == "restore") && !strings.Contains(view, "/o/file.txt") {
					t.Errorf("%s: confirmation doesn't list the file:\n%s", op, view)
				}
			}
		}
	}
}
