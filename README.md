<div align="center">

# 🗑️ Vanish (vx)

### *A modern, safe file deletion tool with recovery capabilities*

[![Release](https://img.shields.io/github/v/release/Nurysso/vanish?include_prereleases&style=flat-square)](https://github.com/Nurysso/vanish/releases/tag/v0.9.0)
[![License](https://img.shields.io/github/license/Nurysso/vanish?style=flat-square)](LICENSE)

[Features](#-features) • [Installation](#-installation) • [Quick Start](#-quick-start) • [Themes](#-themes--customization) • [Documentation](#-command-reference)

</div>

---

## 🎥 See It In Action

<div align="center">

<img src ="https://raw.githubusercontent.com/Nurysso/Hermes/main/vanish/vanish.gif" width="900">

</div>


---

## 🌟 Why Vanish?

Accidentally deleted an important file? Vanish gives you peace of mind with a **smart cache system** that lets you recover files easily. Say goodbye to permanent deletion anxiety and hello to confident file management.

### ✨ Features

<table>
<tr>
<td width="50%">

🛡️ **Safe Deletion**
Files move to cache, never truly deleted

🔄 **Pattern-based Recovery**
Restore using wildcards and flexible matching

📊 **Rich Statistics**
Track cache usage and file metrics

🎨 **Beautiful TUI**
8 stunning built-in themes

</td>
<td width="50%">

⚡ **Blazing Fast**
Handles large directories effortlessly

🔧 **Highly Configurable**
Customize via simple TOML config

📝 **Audit Trails**
Complete operation logging

🧹 **Auto Cleanup**
Configurable retention policies

</td>
</tr>
</table>

---

## 🚀 Installation

### Quick Install (Recommended)

**Using curl:**
```bash
curl -LsSf https://raw.githubusercontent.com/Nurysso/vanish/main/install.sh | sh
```

**Using wget:**
```bash
wget -qO- https://raw.githubusercontent.com/Nurysso/vanish/main/install.sh | sh
```

### Install Specific Version

Replace `<tag>` with your desired version (e.g., `v0.9.0`):

```bash
curl -LsSf https://raw.githubusercontent.com/Nurysso/vanish/<tag>/install.sh | sh
```

### Build from Source

```bash
git clone https://github.com/Nurysso/vanish.git
cd vanish && make build
sudo mv vx /usr/local/bin/
```

---

## 📖 Quick Start

### Basic Operations

```bash
# Delete files/directories safely
vx file.txt folder/ *.log

# List everything in cache
vx --list

# Restore files by pattern
vx --restore "*.txt" "project-*"

# Get detailed file info
vx --info "important-file"

# Clear entire cache
vx --clear

# Remove files older than 30 days
vx --purge 30

# View cache statistics
vx --stats

# Skip confirmations (use with caution!)
vx --restore --noconfirm "*.backup"
```

---

## 🎨 Themes & Customization

Vanish includes **8 gorgeous themes** designed for different moods and environments:

<table>
<tr>
<td align="center" width="25%">

**Default**
🎯 Clean & Professional

</td>
<td align="center" width="25%">

**Dark**
🌑 High Contrast

</td>
<td align="center" width="25%">

**Light**
☀️ Bright & Minimal

</td>
<td align="center" width="25%">

**Cyberpunk**
🌆 Neon Futuristic

</td>
</tr>
<tr>
<td align="center" width="25%">

**Minimal**
✨ Distraction-Free

</td>
<td align="center" width="25%">

**Ocean**
🌊 Calming Blues

</td>
<td align="center" width="25%">

**Forest**
🌲 Natural Greens

</td>
<td align="center" width="25%">

**Sunset**
🌅 Warm & Cozy

</td>
</tr>
</table>

### Try Them Out

```bash
# Interactive theme selector
vx --themes
```

Customize further via the [configuration file](https://github.com/Nurysso/vanish/blob/main/docs/configuration/default-config.md).

---

## 📋 Command Reference

| Command | Shorthand | Description |
|---------|-----------|-------------|
| `vx <files...>` | — | Move files/directories to cache |
| `--restore <pattern>` | `-r` | Restore files matching pattern |
| `--list` | `-l` | Show all cached files |
| `--info <pattern>` | `-i` | Detailed info about items |
| `--clear` | `-c` | Empty entire cache |
| `--purge <days>` | `-pr` | Remove files older than N days |
| `--stats` | `-s` | Display cache statistics |
| `--path` | `-p` | Show cache directory location |
| `--themes` | `-t` | Interactive theme browser |
| `--config-path` | `-cp` | Show config file location |
| `--noconfirm` | `-f` | Skip all confirmation prompts |
| `--help` | `-h` | Show help information |
| `--version` | `-v` | Display version |

---

## 🎯 Pattern Matching Examples

Vanish supports powerful glob patterns for precise file restoration:

```bash
# Exact match
vx --restore "document.pdf"

# All text files
vx --restore "*.txt"

# Files starting with 'backup'
vx --restore "backup-*"

# Multiple patterns at once
vx --restore "*.log" "config.*" "test-*"

# Year-based restoration
vx --restore "*-2024-*"
```

---

## 🛡️ Safety Features

<table>
<tr>
<td width="50%">

✅ **Atomic Operations**
Prevents data corruption during moves

✅ **Path Validation**
Comprehensive conflict prevention

✅ **Collision Detection**
Smart naming conflict resolution

</td>
<td width="50%">

✅ **Permission Preservation**
Maintains original file attributes

✅ **Transaction Logging**
Complete audit trail

✅ **Integrity Checks**
Verification during restoration

</td>
</tr>
</table>

---

## ⚙️ Configuration

Vanish uses **TOML** for easy, human-readable configuration. Customize cache location, retention policies, themes, and more.

📚 **[View Full Configuration Guide →](https://github.com/Nurysso/vanish/blob/main/docs/configuration/condig.md)**

```bash
# Show config file location
vx --config-path
```

---

## ⚠️ Important Notes

### Cache Directory Warning

**Never manually modify the cache directory structure.** To change the cache location:

1. Update the configuration file
2. Run `vx --clear` to empty the old location
3. Restart using the new location

### Security Considerations

- Original file permissions are preserved
- Respects filesystem ACLs and extended attributes
- Symbolic links preserved but not followed
- Hidden files require explicit specification

---

## 🤝 Contributing

We welcome contributions of all kinds! Here's how to get started:

1. **Fork** the repository
2. **Create** your feature branch: `git checkout -b feature/AmazingFeature`
3. **Lint** your code: `make lint`
4. **Commit** your changes: `git commit -m 'Add AmazingFeature'`
5. **Push** to the branch: `git push origin feature/AmazingFeature`
6. **Open** a Pull Request

### Report Bugs or Request Features

- 🐛 [Report a Bug](https://github.com/Nurysso/vanish/issues)
- 💡 [Request a Feature](https://github.com/Nurysso/vanish/discussions)

---

## [LICENSE](LICENSE).

---

## Acknowledgments

Built with using:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Powerful TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Beautiful terminal styling

---

<div align="center">

### 🔗 Links

**[Homepage](https://dwukn.vercel.app/)** • **[Documentation](https://dwukn.vercel.app/)** • **[Releases](https://github.com/Nurysso/vanish/releases)** • **[Discussions](https://github.com/Nurysso/vanish/discussions)**

---

Made with ❤️ by [Nurysso](https://github.com/Nurysso)

⭐ **Star this repo if Vanish made your life easier!**

</div>
