// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Dawood Khan

package command

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"vanish/internal/helpers"
	"vanish/internal/types"
)

// ShowUsageSmart that detects color support of terminal
func ShowUsageSmart(config types.Config) {
	// Check if terminal supports colors
	if helpers.IsColorTerminal() {
		ShowUsage(config)
	} else {
		ShowUsageFallback(config)
	}
}

// ShowUsage prints how to use vanish with example
func ShowUsage(config types.Config) {
	// Styles
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.UI.Colors.Primary)).
		Bold(true).
		Underline(true)

	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.UI.Colors.Success)).
		Bold(true).
		MarginTop(1)

	commandStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.UI.Colors.Highlight)).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.UI.Colors.Text))

	flagStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.UI.Colors.Secondary))

	exampleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.UI.Colors.Muted)).
		Italic(true).
		MarginLeft(2)

	configStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.UI.Colors.Warning)).
		MarginLeft(2)

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.UI.Colors.Muted)).
		Italic(true).
		Align(lipgloss.Center).
		MarginTop(1)

	// Helpers
	const flagColWidth = 28

	printFlag := func(flag, desc string) {
		fmt.Printf("  %-*s %s\n",
			flagColWidth,
			flagStyle.Render(flag),
			descStyle.Render(desc),
		)
	}

	printCmd := func(cmd, desc string) {
		fmt.Printf("  %-*s %s\n",
			flagColWidth,
			commandStyle.Render(cmd),
			descStyle.Render(desc),
		)
	}

	// Title
	fmt.Println(titleStyle.Render("Vanish (vx) — Safe file/directory removal tool"))
	fmt.Println()

	// Usage
	fmt.Println(sectionStyle.Render("USAGE"))
	printCmd("vx <file|dir> [...]", "Safely remove files or directories")
	fmt.Println()

	// File operations
	fmt.Println(sectionStyle.Render("FILE OPERATIONS"))
	printCmd("vx <files...>", "Remove files or directories")
	printFlag("-r, --restore <pattern>", "Restore cached items matching pattern(s)")
	printFlag("-c, --clear", "Clear entire cache immediately")
	printFlag("-pr, --purge <days>", "Delete files older than N days")
	fmt.Println()

	// Information
	fmt.Println(sectionStyle.Render("INFORMATION"))
	printFlag("-l, --list", "List all cached files")
	printFlag("-i, --info <pattern>", "Show detailed info for cached item(s)")
	printFlag("-s, --stats", "Show cache statistics")
	printFlag("-p, --path", "Print cache directory path")
	printFlag("-cp, --config-path", "Print config file path")
	fmt.Println()

	// Customization
	fmt.Println(sectionStyle.Render("CUSTOMIZATION"))
	printFlag("-t, --themes", "Interactive theme selector")
	fmt.Println()

	// Options
	fmt.Println(sectionStyle.Render("OPTIONS"))
	printFlag("-f, --noconfirm", "Skip confirmation prompts")
	printFlag("-h, --help", "Show this help message")
	printFlag("-v, --version", "Show version information")
	printFlag("-q, --quiet", "Run without UI (delete / clear only)")
	fmt.Println()

	// Examples
	fmt.Println(sectionStyle.Render("EXAMPLES"))
	fmt.Println(exampleStyle.Render("Basic usage:"))
	printCmd("vx file1.txt", "# Delete a file safely")
	printCmd("vx file1.txt dir1/ *.log", "# Delete multiple items")
	printCmd("vx -f *.tmp", "# Delete without confirmation")
	fmt.Println()

	fmt.Println(exampleStyle.Render("Recovery:"))
	printCmd("vx -r file1.txt", "# Restore specific file")
	printCmd(`vx -r "*project*"`, "# Restore matching files")
	printCmd(`vx -r "*.pdf" "backup-*"`, "# Restore multiple patterns")
	fmt.Println()

	fmt.Println(exampleStyle.Render("Maintenance:"))
	printCmd("vx -pr 30", "# Purge files older than 30 days")
	printCmd("vx -s", "# Show cache statistics")
	printCmd("vx -c", "# Clear entire cache")
	fmt.Println()

	// Current config
	fmt.Println(sectionStyle.Render("CURRENT CONFIGURATION"))
	fmt.Printf("  %s %s\n", configStyle.Render("Cache location:"), descStyle.Render(config.Cache.Directory))
	fmt.Printf("  %s %s\n", configStyle.Render("Retention period:"), descStyle.Render(fmt.Sprintf("%d days", config.Cache.Days)))
	fmt.Printf("  %s %s\n", configStyle.Render("Skip confirmations:"), descStyle.Render(fmt.Sprintf("%v", config.Cache.NoConfirm)))
	fmt.Printf("  %s %s\n", configStyle.Render("Current theme:"), descStyle.Render(config.UI.Theme))

	if config.Logging.Enabled {
		fmt.Printf("  %s %s\n",
			configStyle.Render("Logging:"),
			descStyle.Render("enabled → "+config.Logging.Directory),
		)
	} else {
		fmt.Printf("  %s %s\n",
			configStyle.Render("Logging:"),
			descStyle.Render("disabled"),
		)
	}

	fmt.Println(footerStyle.Render("For more information: https://github.com/Nurysoo/vanish"))
}

// ShowUsageFallback is a alternative fallback to simple output if colors are not available
func ShowUsageFallback(config types.Config) {
	fmt.Println("Vanish (vx) - Safe file/directory removal tool")
	fmt.Println("=" + strings.Repeat("=", 47))
	fmt.Println()

	fmt.Println("USAGE:")
	fmt.Println("  vx <file|directory> [file2] [dir2] ...        Remove files/directories safely")
	fmt.Println()

	fmt.Println("FILE OPERATIONS:")
	fmt.Println("  vx <files...>                                 Remove files/directories safely")
	fmt.Println("  -r, --restore <pattern>...                   Restore files matching patterns")
	fmt.Println("  -c, --clear                                   Clear all cached files immediately")
	fmt.Println("  -pr, --purge <days>                           Delete files older than N days")
	fmt.Println()

	fmt.Println("INFORMATION:")
	fmt.Println("  -l, --list                                    Show all cached files")
	fmt.Println("  -i, --info <pattern>                          Show detailed info about cached item(s)")
	fmt.Println("  -s, --stats                                   Show cache statistics")
	fmt.Println("  -p, --path                                    Print cache directory path")
	fmt.Println("  -cp, --config-path                            Print config file path")
	fmt.Println()

	fmt.Println("CUSTOMIZATION:")
	fmt.Println("  -t, --themes                                  Interactive theme selector")
	// fmt.Println("  --completion <shell>                          Generate shell completion")
	fmt.Println()

	fmt.Println("OPTIONS:")
	fmt.Println("  -f, --noconfirm                               Skip confirmation prompts")
	fmt.Println("  -h, --help                                    Show this help message")
	fmt.Println("  -v, --version                                 Show version information")
	fmt.Println()

	fmt.Println("EXAMPLES:")
	fmt.Println("  vx file1.txt                                  # Delete file safely")
	fmt.Println("  vx -r \"*project*\"                             # Restore files matching pattern")
	fmt.Println("  vx -pr 30                                     # Purge files older than 30 days")
	fmt.Println()

	fmt.Println("CURRENT CONFIGURATION:")
	fmt.Printf("  Cache location: %s\n", config.Cache.Directory)
	fmt.Printf("  Retention period: %d days\n", config.Cache.Days)
	fmt.Printf("  Current theme: %s\n", config.UI.Theme)
	fmt.Printf("  Logging: %v\n", config.Logging.Enabled)
	// fmt.Printf("  Notifications: %v\n", config.Notifications.DesktopEnabled)
	fmt.Println("\n\nFor more information visit: https://github.com/Nurysoo/vanish")
}
