# gamux

**gamux** (previously `gbe_fork_helper`) is a utility tool for managing Steam games on Linux. It streamlines DRM removal from your own clean Steam files using **Goldberg Steam Emulator** (`gbe_fork`), adds games to **Lutris**, and optionally creates shortcuts in **Steam** itself.

The goal is to be a one-stop post-download tool: fetch your game files clean from Steam, strip the DRM, configure it for offline/emulated use, and register it with your preferred launcher.

## Philosophy

gamux assumes you **own the game** on Steam. You download clean game files (e.g., from Steam's content servers), and gamux helps you:
1. **Remove DRM** — apply the Goldberg emulator to bypass Steam runtime checks.
2. **Register the game** — add the executable to Lutris (with artwork, config, etc.).
3. **Optionally add to Steam** — create a non-Steam game shortcut in Steam (useful for Steam Deck / Big Picture).

Future versions will fetch game metadata (titles, artwork, install paths) from the **Steam Store API** so you barely need to type an AppID.

## Features

- **DRM Removal (Goldberg Emulator)** — Applies `gbe_fork` patches to `steam_api` libraries on Linux, Windows (64/32-bit) targets. Backs up originals automatically.
- **DLC Configuration** — Fetches DLC lists from the Steam Store API in parallel and writes `configs.app.ini` so all owned DLCs are unlocked.
- **Native Archive Handling** — Uses pure-Go libraries for `.7z` and `.tar.bz2` extraction — no external `7z` or `tar` required.
- **Concurrent Operations** — Downloads & extracts Linux and Windows releases in parallel. Fetches DLC names concurrently with rate-limit awareness.
- **Lutris Integration** *(planned)* — Add game executables to Lutris with metadata.
- **Steam Shortcuts** *(planned)* — Optionally create non-Steam game shortcuts in Steam's `shortcuts.vdf`.
- **Interactive TUI** — Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for a guided workflow.
- **Dry-run Mode** — Preview all file changes before applying them.
- **Structured Logging** — Uses `log/slog` for clean, professional output.
- **Custom Configuration** — Supports custom paths via JSON config file.

## Installation

```bash
# Build from source
make build

# Install to ~/.local/bin
make install
```

## Usage

```bash
# Apply Goldberg emulator to a game (DRM removal)
gamux apply linux 123456

# Apply with dry-run (preview only)
gamux apply --dry-run linux 123456

# Update GBE fork from GitHub
gamux update

# Start interactive TUI
gamux tui

# Load custom config
gamux --config my_config.json apply linux 123456

# Display version
gamux --version
```

### Platform Targets

| Platform | Architecture | Steam API File |
|----------|-------------|----------------|
| `linux`  | 64-bit      | `libsteam_api.so` |
| `win64`  | 64-bit      | `steam_api64.dll` |
| `win32`  | 32-bit      | `steam_api.dll` |

The tool walks the current directory looking for matching Steam API files (including subdirectories) so you can run it from above a multi-part game.

## Roadmap

### Short Term
- [x] Core GBE apply/update/TUI workflow
- [x] Configuration file support for custom paths
- [x] Dry-run mode for `apply`
- [ ] **Lutris installer entry** — create and register a Lutris game entry from the CLI
- [ ] **Steam shortcut** — optionally write a non-Steam game shortcut to Steam's `shortcuts.vdf`
- [ ] **Steam API metadata** — look up game name, artwork, and default install paths by AppID

### Medium Term
- [ ] **Multiple-app support** — toggle between multiple games in the TUI
- [ ] **Batch mode** — process a directory tree of games in one pass
- [ ] **Rollback** — restore originals from backup files
- [ ] **Artwork download** — fetch capsule art, banners, and icons for Lutris/Steam

### Long Term
- [ ] **GUI** — evaluate Go GUI frameworks (Wails, Fyne) for a desktop application
- [ ] **Steam content server integration** — download clean files directly via Steam anonymous CDN
- [ ] **Proton/Wine prefix detection** — auto-detect Wine prefixes for Windows games on Linux

## Configuration

Configuration is loaded from `~/.config/gbe_fork_helper/config.json` (or a custom path via `--config`):

```json
{
  "gbe_dir": ".local/share/gbe_fork"
}
```

## Project Structure

```
cmd/gamux/             — CLI entry point (urfave/cli/v2)
config/                — Configuration structs and defaults
gbe/                   — GBE patch orchestration logic
github/                — GitHub release fetcher (gbe_fork updates)
steam/                 — Steam Store API client (app details, DLCs)
tui/                   — Bubble Tea interactive terminal UI
util/                  — Helpers (archive extraction, hashing, backups)
```

## Development

```bash
make test    # Run all tests
make lint    # Lint with golangci-lint
make build   # Build the binary
make run     # Run directly with go run
```

## License

MIT
