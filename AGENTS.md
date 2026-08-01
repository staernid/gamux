# Agentic Developer Guide (`agents.md`)

Welcome, AI Agent! This repository is optimized for autonomous and agentic development. This document serves as your developer handbook to help you understand the codebase, its architecture, tooling commands, and best practices.

---

## 1. Project Overview — **gamux**

**gamux** (previously `gbe_fork_helper`) is a utility tool written in Go for managing Steam games on Linux. It performs three core tasks:

1. **DRM Removal** — Applies the **Goldberg Steam Emulator** (`gbe_fork`) to clean Steam files, removing the Steam runtime requirement.
2. **Lutris Registration** — Adds game executables to Lutris with metadata (artwork, config, Wine prefix support).
3. **Steam Shortcuts** — Optionally creates non-Steam game shortcuts in Steam's `shortcuts.vdf`.

The project started as a pure `gbe_fork_helper` but is expanding into a complete post-download game management tool for Linux gamers who own their games on Steam.

### Key Features
*   **Native Archive Handling**: Extracts `.tar.bz2` (Linux) and `.7z` (Windows) releases natively without external command-line dependencies.
*   **Concurrent DLC Fetching**: Queries the Steam Store/CDN APIs and extracts/configures DLC names in parallel using Go routines.
*   **Dry-run Mode**: Validates what files would be copied/replaced before performing any writes.
*   **TUI & CLI Dual Interface**: Operates via normal command-line execution or an interactive terminal UI.
*   **Lutris Integration** *(in progress)*: Creates game entries in Lutris via its CLI or YAML-based game installers.
*   **Steam Shortcut Management** *(planned)*: Writes to `shortcuts.vdf` for non-Steam game entries.

---

## 2. Directory Layout & Architecture

Here is the structural outline of the codebase:

```mermaid
graph TD
    Main[cmd/gamux/main.go] --> Config[config/config.go]
    Main --> GBE[gbe/gbe.go]
    Main --> TUI[tui/tui.go]
    Main --> Github[github/github.go]
    Main --> Lutris[lutris/]              # TODO: Lutris integration module
    Main --> SteamShortcut[steamshortcut/] # TODO: Steam shortcut module

    GBE --> Steam[steam/steam.go]
    GBE --> Util[util/util.go]
    Github --> Util
    Steam --> Util
```

### Module Responsibilities

*   **`cmd/gamux/main.go`**
    *   The entry point. Defines and registers commands (`apply`, `update`, `tui`, future: `lutris-add`, `steam-add`) using `urfave/cli/v2`.
    *   Sets up standard text structured logging (`log/slog`).
    *   Currently at `cmd/gbe_fork_helper/` — will be restructured as project scope expands.
*   **`config/config.go`**
    *   Defines configuration structs, default paths (default GBE dir: `~/.local/share/gbe_fork`), and OS/Platform configuration mappings (targets, architectures, additional files like `steamclient.so`).
*   **`gbe/gbe.go`**
    *   Orchestrates the emulator patching process. Walks directories to find target `steam_api` libraries, checks SHA256 hashes to prevent redundant writes, backs up originals, copies Goldberg files, runs interfaces generators, and triggers DLC retrieval.
*   **`github/github.com`**
    *   Fetches the latest release info for Goldberg Steam Emulator from GitHub.
    *   Handles parallel download and extraction of Windows and Linux release assets.
*   **`steam/`**
    *   `steam.go`: Fetches game names and DLC lists from the Steam Store API, and writes `steam_appid.txt` and `configs.app.ini`.
    *   `achievements.go`: (Optional/Helper) Downloads achievements icons/images from Steam CDN.
*   **`tui/tui.go`**
    *   Provides the interactive Bubble Tea terminal user interface.
*   **`util/util.go`**
    *   Contains helper functions: native command runner, file hashers (SHA256), atomic backup-and-replace, and native extraction logic (`tar.bz2` and `7z`).

### Future Directories

| Directory | Status | Purpose |
|-----------|--------|---------|
| `lutris/` | Planned | Lutris game entry creation, Wine prefix auto-detection, YAML installer generation |
| `steamshortcut/` | Planned | Reading/writing Steam `shortcuts.vdf` for non-Steam game entries |
| `metadata/` | Planned | Fetching artwork, descriptions, and metadata from Steam Store API |

---

## 3. Tooling & Commands

Use the provided `Makefile` in the root of the project for common development workflows.

| Command | Action | Description |
| :--- | :--- | :--- |
| `make build` | `go build -o gamux ./cmd/gamux` | Compiles the binary to the root directory |
| `make test` | `go test -v ./...` | Runs all unit and package tests |
| `make lint` | `golangci-lint run` | Lints the codebase |
| `make tidy` | `go mod tidy` | Cleans up and locks Go modules |
| `make run` | `go run ./cmd/gamux` | Runs gamux instantly |
| `make clean` | `go clean && rm -f gamux` | Removes build artifacts |
| `make install` | Installs binary to `~/.local/bin` | Installs gamux locally |

---

## 4. Development Guidelines & Conventions

When proposing changes, fixing bugs, or implementing new features, you **must** adhere to the following conventions:

### A. Logging Strategy
*   Use standard `log/slog` for structured logging.
*   **Do not use basic `fmt.Println` or standard loggers** in libraries (`gbe`, `steam`, `util`, `lutris`, etc.) for status updates. Always use `slog.Info`, `slog.Warn`, or `slog.Error` with key-value contexts.
*   *TUI Safety*: Note that in TUI mode, `slog` output is redirected to `gamux.log` (or `io.Discard` on error) to prevent drawing corruption over the bubbletea UI.

### B. Error Handling
*   Wrap errors using `%w` rather than discarding the context, example:
    ```go
    return fmt.Errorf("failed to extract file %s: %w", filename, err)
    ```
*   Keep errors descriptive and localized.

### C. Concurrency Control
*   We use `golang.org/x/sync/errgroup` for running concurrent network calls or operations.
*   **Rate Limits**: When hammering external APIs (like Steam Store or GitHub CDN), always enforce worker limits. Use `g.SetLimit(10)` or equivalent to avoid rate limits (HTTP 429).
*   Always respect and pass through `context.Context` for proper cancellation propagation.

### D. File-System Operations and Safety
*   **Backup First**: Never overwrite a target library (e.g., `steam_api64.dll` or `libsteam_api.so`) without creating a backup copy. Use `util.BackupAndReplace(src, dest)` which automatically saves the original to `<dest>.<timestamp>.ORIGINAL`.
*   **Dry-run Support**: Any function modifying the filesystem must support dry-run flags. Output planned mutations using `slog.Info("[DRY RUN] ...")` without changing state.

### E. Test Practices
*   Ensure critical business logic is covered by unit tests.
*   **Mock Network Calls**: Mock HTTP requests using custom `http.RoundTripper` functions in tests (see `steam/steam_test.go` for examples). Avoid hitting real Steam/GitHub endpoints in unit tests.
*   **Clean Up**: Always clean up temporary directories using `defer os.RemoveAll(tmpDir)`.

---

## 5. Key Dependencies Reference

Keep these versions and tools in mind when updating `go.mod` or modifying library integrations:
*   **TUI/CLI**: `github.com/urfave/cli/v2` (CLI structure) & `github.com/charmbracelet/bubbletea` (TUI loop).
*   **TUI Styling**: `github.com/charmbracelet/lipgloss` (UI element styles).
*   **Markdown UI**: `github.com/charmbracelet/glamour` (terminal markdown rendering for release details).
*   **Extraction**: `github.com/bodgit/sevenzip` (pure Go 7z decoder).
