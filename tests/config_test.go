package tests

import (
	"os"
	"path/filepath"
	"testing"
	c "vanish/internal/config"
)

func TestLoadConfig(t *testing.T) {
	// Save original home dir and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Create a temporary home directory
	tmpHome, err := os.MkdirTemp("", "test-home")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	os.Setenv("HOME", tmpHome)

	t.Run("Creates default config when no config file exists", func(t *testing.T) {
		config, err := c.LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		// Check default values
		expectedCacheDir := filepath.Join(tmpHome, ".cache", "vanish")
		if config.Cache.Directory != expectedCacheDir {
			t.Errorf("Expected cache dir %s, got %s", expectedCacheDir, config.Cache.Directory)
		}

		if config.Cache.Days != 10 {
			t.Errorf("Expected cache days 10, got %d", config.Cache.Days)
		}

		if !config.Logging.Enabled {
			t.Error("Expected logging to be enabled by default")
		}

		expectedLogDir := filepath.Join(tmpHome, ".cache", "vanish", "logs")
		if config.Logging.Directory != expectedLogDir {
			t.Errorf("Expected log dir %s, got %s", expectedLogDir, config.Logging.Directory)
		}

		// Check that default config file was created
		configPath := filepath.Join(tmpHome, ".config", "vanish", "vanish.toml")
		if _, err := os.Stat(configPath); err != nil {
			t.Error("Default config file was not created")
		}
	})

	t.Run("Loads custom config file", func(t *testing.T) {
		// Create config directory
		configDir := filepath.Join(tmpHome, ".config", "vanish")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("Failed to create config dir: %v", err)
		}

		// Write custom config
		configPath := filepath.Join(configDir, "vanish.toml")
		customConfig := `
[cache]
directory = "/custom/cache"
days = 20

[logging]
enabled = false
directory = "/custom/logs"

[ui]
theme = "dark"

[ui.colors]
primary = "#123456"
success = "#654321"
`
		if err := os.WriteFile(configPath, []byte(customConfig), 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		config, err := c.LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		// Check custom values
		if config.Cache.Directory != "/custom/cache" {
			t.Errorf("Expected cache dir /custom/cache, got %s", config.Cache.Directory)
		}

		if config.Cache.Days != 20 {
			t.Errorf("Expected cache days 20, got %d", config.Cache.Days)
		}

		if config.Logging.Enabled {
			t.Error("Expected logging to be disabled")
		}

		if config.Logging.Directory != "/custom/logs" {
			t.Errorf("Expected log dir /custom/logs, got %s", config.Logging.Directory)
		}

		if config.UI.Theme != "dark" {
			t.Errorf("Expected theme dark, got %s", config.UI.Theme)
		}

		if config.UI.Colors.Primary != "#123456" {
			t.Errorf("Expected primary color #123456, got %s", config.UI.Colors.Primary)
		}
	})

	t.Run("Falls back to default theme when unknown theme specified", func(t *testing.T) {
		configDir := filepath.Join(tmpHome, ".config", "vanish")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("Failed to create config dir: %v", err)
		}

		configPath := filepath.Join(configDir, "vanish.toml")
		customConfig := `
[ui]
theme = "nonexistent"
`
		if err := os.WriteFile(configPath, []byte(customConfig), 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		config, err := c.LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		// When an unknown theme is specified, the function falls back to default
		// and replaces the entire UI config with the default theme's UI
		if config.UI.Theme != "default" {
			t.Errorf("Expected theme to be 'default' after fallback, got %s", config.UI.Theme)
		}

		// Colors should be from default theme
		defaultThemes := c.GetDefaultThemes()
		defaultTheme := defaultThemes["default"]

		if config.UI.Colors.Primary != defaultTheme.UI.Colors.Primary {
			t.Errorf("Expected primary color %s from default theme, got %s",
				defaultTheme.UI.Colors.Primary, config.UI.Colors.Primary)
		}

		if config.UI.Colors.Success != defaultTheme.UI.Colors.Success {
			t.Errorf("Expected success color %s from default theme, got %s",
				defaultTheme.UI.Colors.Success, config.UI.Colors.Success)
		}
	})

	t.Run("Applies custom colors on top of theme", func(t *testing.T) {
		configDir := filepath.Join(tmpHome, ".config", "vanish")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("Failed to create config dir: %v", err)
		}

		configPath := filepath.Join(configDir, "vanish.toml")
		customConfig := `
[ui]
theme = "default"

[ui.colors]
primary = "#FF0000"
success = "#00FF00"
`
		if err := os.WriteFile(configPath, []byte(customConfig), 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		config, err := c.LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if config.UI.Colors.Primary != "#FF0000" {
			t.Errorf("Expected primary color #FF0000, got %s", config.UI.Colors.Primary)
		}

		if config.UI.Colors.Success != "#00FF00" {
			t.Errorf("Expected success color #00FF00, got %s", config.UI.Colors.Success)
		}
	})
}
