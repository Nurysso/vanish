package tests

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	command "vanish/cmd/commands"
	c "vanish/internal/config"
	h "vanish/internal/helpers"
	"vanish/internal/tui"
	"vanish/internal/types"
)

func TestParseArgs(t *testing.T) {
	cfg := getTestConfig()
	tests := []struct {
		name string
		args []string
		want command.ParsedArgs
	}{
		{"delete", []string{"a", "b"}, command.ParsedArgs{Operation: "delete", Filenames: []string{"a", "b"}}},
		{"delete with -f", []string{"-f", "a"}, command.ParsedArgs{Operation: "delete", Filenames: []string{"a"}, NoConfirm: true}},
		{"delete with --noconfirm", []string{"--noconfirm", "a"}, command.ParsedArgs{Operation: "delete", Filenames: []string{"a"}, NoConfirm: true}},
		{"quiet implies no confirm", []string{"-q", "a"}, command.ParsedArgs{Operation: "delete", Filenames: []string{"a"}, NoConfirm: true, Headless: true}},
		{"--quiet", []string{"--quiet", "a"}, command.ParsedArgs{Operation: "delete", Filenames: []string{"a"}, NoConfirm: true, Headless: true}},
		{"restore", []string{"-r", "a", "b"}, command.ParsedArgs{Operation: "restore", Filenames: []string{"a", "b"}}},
		{"--restore", []string{"--restore", "a"}, command.ParsedArgs{Operation: "restore", Filenames: []string{"a"}}},
		{"-f then restore", []string{"-f", "-r", "a"}, command.ParsedArgs{Operation: "restore", Filenames: []string{"a"}, NoConfirm: true}},
		{"clear", []string{"-c"}, command.ParsedArgs{Operation: "clear", Filenames: []string{""}}},
		{"--clear with -f", []string{"-f", "--clear"}, command.ParsedArgs{Operation: "clear", Filenames: []string{""}, NoConfirm: true}},
		{"purge", []string{"-pr", "7"}, command.ParsedArgs{Operation: "purge", Filenames: []string{"7"}}},
		{"--purge quiet", []string{"-q", "--purge", "7"}, command.ParsedArgs{Operation: "purge", Filenames: []string{"7"}, NoConfirm: true, Headless: true}},
		{"restore then -f", []string{"-r", "a", "-f", "b"}, command.ParsedArgs{Operation: "restore", Filenames: []string{"a", "b"}, NoConfirm: true}},
		{"restore then --quiet", []string{"-r", "--quiet", "a"}, command.ParsedArgs{Operation: "restore", Filenames: []string{"a"}, NoConfirm: true, Headless: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := command.ParseArgs(tt.args, cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseArgs(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}

	t.Run("README's --restore --noconfirm", func(t *testing.T) {
		want := command.ParsedArgs{Operation: "restore", Filenames: []string{"x"}, NoConfirm: true}
		if got := command.ParseArgs([]string{"--restore", "--noconfirm", "x"}, cfg); !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
}

func TestHeadlessMetadataAndSymlinks(t *testing.T) {
	cfg := testCfg(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "target", "file")
	writeFile(t, target, "precious")
	link := filepath.Join(dir, "link")
	dangling := filepath.Join(dir, "dangling")
	symlink(t, filepath.Dir(target), link)
	symlink(t, "/nonexistent", dangling)

	if err := tui.ExecuteHeadless([]string{link, dangling}, "delete", cfg); err != nil {
		t.Fatal(err)
	}
	wantGone(t, link)
	wantGone(t, dangling)
	wantFile(t, target, "precious")

	items := loadItems(t, cfg)
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	for _, item := range items {
		if !item.IsSymlink || item.LinkTarget == "" {
			t.Errorf("symlink metadata missing: %+v", item)
		}
		wantLink(t, item.CachePath, item.LinkTarget)
	}
}

func TestHeadlessLogs(t *testing.T) {
	cfg := testCfg(t)
	cfg.Logging.Enabled = true
	file := filepath.Join(t.TempDir(), "file")
	writeFile(t, file, "x")

	if err := tui.ExecuteHeadless([]string{file}, "delete", cfg); err != nil {
		t.Fatal(err)
	}
	log, _ := os.ReadFile(filepath.Join(cfg.Logging.Directory, "vanish.log"))
	if !strings.Contains(string(log), "DELETE: "+file) {
		t.Errorf("delete not logged: %q", log)
	}

	if err := tui.ExecuteHeadless(nil, "clear", cfg); err != nil {
		t.Fatal(err)
	}
	log, _ = os.ReadFile(filepath.Join(cfg.Logging.Directory, "vanish.log"))
	if !strings.Contains(string(log), "CLEAR_ALL") {
		t.Errorf("clear not logged: %q", log)
	}
}

func TestHeadlessPurgeRequiresDays(t *testing.T) {
	if err := tui.ExecuteHeadless(nil, "purge", testCfg(t)); err == nil {
		t.Error("expected an error")
	}
}

func TestHeadlessCleanupExpiresOldItems(t *testing.T) {
	cfg := testCfg(t)
	old := filepath.Join(cfg.Cache.Directory, "old")
	writeFile(t, old, "x")
	saveItems(t, cfg, types.DeletedItem{ID: "old", CachePath: old, DeleteDate: time.Now().Add(-11 * 24 * time.Hour)})
	file := filepath.Join(t.TempDir(), "file")
	writeFile(t, file, "x")

	if err := tui.ExecuteHeadless([]string{file}, "delete", cfg); err != nil {
		t.Fatal(err)
	}
	wantGone(t, old)
	if items := loadItems(t, cfg); len(items) != 1 || items[0].ID == "old" {
		t.Errorf("items = %+v", items)
	}
}

func TestHeadlessSafety(t *testing.T) {
	old := time.Now().Add(-30 * 24 * time.Hour)

	for _, days := range []string{"0", "-1"} {
		t.Run("purge rejects days="+days, func(t *testing.T) {
			cfg := testCfg(t)
			recent := filepath.Join(cfg.Cache.Directory, "recent")
			writeFile(t, recent, "precious")
			saveItems(t, cfg, types.DeletedItem{ID: "1", CachePath: recent, DeleteDate: old})

			if err := tui.ExecuteHeadless([]string{days}, "purge", cfg); err == nil {
				t.Error("expected an error")
			}
			wantFile(t, recent, "precious")
		})
	}

	t.Run("restore doesn't delete its argument", func(t *testing.T) {
		cfg := testCfg(t)
		file := filepath.Join(t.TempDir(), "notes.txt")
		writeFile(t, file, "precious")

		if err := tui.ExecuteHeadless([]string{file}, "restore", cfg); err == nil {
			t.Error("expected an error for no matches")
		}
		wantFile(t, file, "precious")
	})

	t.Run("restore round trip", func(t *testing.T) {
		cfg := testCfg(t)
		file := filepath.Join(t.TempDir(), "notes.txt")
		writeFile(t, file, "data")
		if err := tui.ExecuteHeadless([]string{file}, "delete", cfg); err != nil {
			t.Fatal(err)
		}
		wantGone(t, file)

		if err := tui.ExecuteHeadless([]string{"*.txt"}, "restore", cfg); err != nil {
			t.Fatal(err)
		}
		wantFile(t, file, "data")
		if items := loadItems(t, cfg); len(items) != 0 {
			t.Errorf("items = %+v", items)
		}
	})

	t.Run("restore reports a partial failure", func(t *testing.T) {
		cfg := testCfg(t)
		dir := t.TempDir()
		cached := filepath.Join(cfg.Cache.Directory, "b")
		writeFile(t, cached, "b")
		saveItems(t, cfg,
			types.DeletedItem{ID: "1", OriginalPath: filepath.Join(dir, "a.txt"), CachePath: filepath.Join(cfg.Cache.Directory, "gone")},
			types.DeletedItem{ID: "2", OriginalPath: filepath.Join(dir, "b.txt"), CachePath: cached},
		)
		if err := tui.ExecuteHeadless([]string{"*.txt"}, "restore", cfg); err == nil {
			t.Error("expected an error")
		}
		wantFile(t, filepath.Join(dir, "b.txt"), "b")
	})

	t.Run("clear handles a read-only tree", func(t *testing.T) {
		cfg := testCfg(t)
		readOnlyTree(t, filepath.Join(cfg.Cache.Directory, "item"))
		if err := tui.ExecuteHeadless(nil, "clear", cfg); err != nil {
			t.Error(err)
		}
	})

	t.Run("purge keeps the entry when removal fails", func(t *testing.T) {
		cfg := testCfg(t)
		locked := filepath.Join(cfg.Cache.Directory, "locked")
		readOnlyTree(t, locked)
		cached := filepath.Join(locked, "sub", "file.txt")
		saveItems(t, cfg, types.DeletedItem{ID: "1", CachePath: cached, DeleteDate: old})

		tui.ExecuteHeadless([]string{"7"}, "purge", cfg)
		wantFile(t, cached, "ro")
		if len(loadItems(t, cfg)) != 1 {
			t.Error("entry dropped while its file still exists")
		}
	})

	t.Run("a corrupt index doesn't orphan files", func(t *testing.T) {
		cfg := testCfg(t)
		writeFile(t, h.GetIndexPath(cfg), "{not json")
		file := filepath.Join(t.TempDir(), "file")
		writeFile(t, file, "precious")

		if err := tui.ExecuteHeadless([]string{file}, "delete", cfg); err == nil {
			t.Error("expected an error")
		}
		wantFile(t, file, "precious")
	})

	t.Run("deletes a name near NAME_MAX", func(t *testing.T) {
		cfg := testCfg(t)
		file := filepath.Join(t.TempDir(), strings.Repeat("x", 250))
		writeFile(t, file, "x")

		if err := tui.ExecuteHeadless([]string{file}, "delete", cfg); err != nil {
			t.Error(err)
		}
		wantGone(t, file)
	})
}

func TestLoadConfigFollowsXDG(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	writeFile(t, h.GetConfigPath(), "[cache]\ndays = 3\n")

	cfg, err := c.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Cache.Days != 3 {
		t.Errorf("Days = %d, want 3 from %s", cfg.Cache.Days, h.GetConfigPath())
	}
}

func TestLoadConfigInvalidToml(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	writeFile(t, filepath.Join(home, ".config", "vanish", "vanish.toml"), "[cache\nnot toml")
	if _, err := c.LoadConfig(); err == nil {
		t.Error("expected an error")
	}
}

// TestCLI runs the real binary against a throwaway $HOME, covering the os.Exit
// paths and main.go. Interactive flags (--list, --stats, --info) need a TTY.
func TestCLI(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "vx")
	build := exec.Command("go", "build", "-o", bin, "vanish")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	type setup struct{ home, work string }
	newEnv := func(t *testing.T) setup { return setup{t.TempDir(), t.TempDir()} }
	vx := func(t *testing.T, s setup, args ...string) (string, int) {
		t.Helper()
		cmd := exec.Command(bin, args...)
		cmd.Dir = s.work
		cmd.Env = []string{"HOME=" + s.home, "TERM=dumb"}
		var out bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &out
		err := cmd.Run()
		if _, isExit := err.(*exec.ExitError); err != nil && !isExit {
			t.Fatal(err)
		}
		return out.String(), cmd.ProcessState.ExitCode()
	}
	index := func(t *testing.T, s setup) types.Index {
		t.Helper()
		var idx types.Index
		data, err := os.ReadFile(filepath.Join(s.home, ".cache", "vanish", "index.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &idx); err != nil {
			t.Fatal(err)
		}
		return idx
	}

	t.Run("quiet delete of a relative path records the absolute one", func(t *testing.T) {
		s := newEnv(t)
		writeFile(t, filepath.Join(s.work, "rel.txt"), "x")

		out, code := vx(t, s, "-q", "rel.txt")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		if !strings.Contains(out, "Moved to cache: rel.txt\n") { // guards the old Println("%s") bug
			t.Errorf("output = %q", out)
		}
		wantGone(t, filepath.Join(s.work, "rel.txt"))
		items := index(t, s).Items
		if len(items) != 1 || items[0].OriginalPath != filepath.Join(s.work, "rel.txt") {
			t.Errorf("items = %+v", items)
		}
	})

	t.Run("first run creates the default config", func(t *testing.T) {
		s := newEnv(t)
		vx(t, s, "-v")
		if _, err := os.Stat(filepath.Join(s.home, ".config", "vanish", "vanish.toml")); err != nil {
			t.Error(err)
		}
	})

	t.Run("default config round-trips into the same cache directory", func(t *testing.T) {
		s := newEnv(t)
		vx(t, s, "-v") // writes the default config, which the next run parses
		out, code := vx(t, s, "-p")
		if want := filepath.Join(s.home, ".cache", "vanish") + "\n"; code != 0 || out != want {
			t.Errorf("exit %d, output %q, want %q", code, out, want)
		}
	})

	t.Run("quiet clear and purge", func(t *testing.T) {
		s := newEnv(t)
		writeFile(t, filepath.Join(s.work, "a"), "x")
		vx(t, s, "-q", "a")

		if out, code := vx(t, s, "-q", "-pr", "7"); code != 0 {
			t.Errorf("purge: exit %d: %s", code, out)
		}
		if len(index(t, s).Items) != 1 {
			t.Error("purge removed a recent item")
		}
		if out, code := vx(t, s, "-q", "-c"); code != 0 {
			t.Errorf("clear: exit %d: %s", code, out)
		}
		if len(index(t, s).Items) != 0 {
			t.Error("clear left items behind")
		}
	})

	exits := []struct {
		name string
		args []string
		code int
		want string
	}{
		{"version", []string{"-v"}, 0, command.VERSION},
		{"long version", []string{"--version"}, 0, command.VERSION},
		{"config path", []string{"-cp"}, 0, "vanish.toml"},
		{"help", []string{"-h"}, 0, "--restore"},
		{"no arguments shows usage", nil, 0, "--restore"},
		{"themes", []string{"-t"}, 0, "OCEAN"},
		{"quiet delete of a missing file", []string{"-q", "/nonexistent/a"}, 1, "no valid files"},
		{"quiet purge with bad days", []string{"-q", "-pr", "abc"}, 1, "invalid days"},
		{"restore without a pattern", []string{"-r"}, 1, "requires at least one pattern"},
		{"restore with only flags", []string{"-r", "-f"}, 1, "requires at least one pattern"},
		{"quiet restore with no match", []string{"-q", "-r", "absent"}, 1, "no matching items"},
		{"purge without days", []string{"-pr"}, 1, "requires number of days"},
		{"info without a pattern", []string{"-i"}, 1, "requires a pattern"},
		{"flags without an operation", []string{"-f"}, 1, ""},
	}
	for _, tt := range exits {
		t.Run(tt.name, func(t *testing.T) {
			out, code := vx(t, newEnv(t), tt.args...)
			if code != tt.code || !strings.Contains(out, tt.want) {
				t.Errorf("exit %d, want %d; output missing %q:\n%s", code, tt.code, tt.want, out)
			}
		})
	}

	t.Run("a broken config exits non-zero", func(t *testing.T) {
		s := newEnv(t)
		writeFile(t, filepath.Join(s.home, ".config", "vanish", "vanish.toml"), "[cache\nnot toml")
		if out, code := vx(t, s, "-v"); code != 1 || !strings.Contains(out, "parsing config") {
			t.Errorf("exit %d: %s", code, out)
		}
	})

	t.Run("quiet restore doesn't trash the named file", func(t *testing.T) {
		s := newEnv(t)
		file := filepath.Join(s.work, "notes.txt")
		writeFile(t, file, "precious")
		if out, code := vx(t, s, "-q", "-r", "notes.txt"); code != 1 {
			t.Errorf("exit %d: %s", code, out)
		}
		wantFile(t, file, "precious")
	})

	t.Run("quiet delete then restore", func(t *testing.T) {
		s := newEnv(t)
		file := filepath.Join(s.work, "notes.txt")
		writeFile(t, file, "data")
		vx(t, s, "-q", "notes.txt")
		wantGone(t, file)

		if out, code := vx(t, s, "--restore", "--quiet", "*.txt"); code != 0 {
			t.Fatalf("exit %d: %s", code, out)
		}
		wantFile(t, file, "data")
	})

	t.Run("quiet delete fails when nothing could be moved", func(t *testing.T) {
		s := newEnv(t)
		file := filepath.Join(s.work, "notes.txt")
		writeFile(t, file, "data")
		writeFile(t, filepath.Join(s.home, ".cache", "vanish", "index.json"), "{not json")

		if out, code := vx(t, s, "-q", "notes.txt"); code != 1 {
			t.Errorf("exit %d: %s", code, out)
		}
		wantFile(t, file, "data")
	})

	t.Run("--config-path names the file actually loaded", func(t *testing.T) {
		s := newEnv(t)
		xdg := filepath.Join(s.home, "xdg")
		cmd := exec.Command(bin, "-cp")
		cmd.Env = []string{"HOME=" + s.home, "XDG_CONFIG_HOME=" + xdg}
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(xdg, "vanish", "vanish.toml")
		if strings.TrimSpace(string(out)) != want {
			t.Errorf("--config-path = %q, want %q", out, want)
		}
		if _, err := os.Stat(want); err != nil {
			t.Errorf("default config not created there: %v", err)
		}
	})
}
