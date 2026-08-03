# Developer & AI Assistant Guide (`AGENTS.md`)

Essential guidelines, architecture patterns, and conventions for development on `gamux`.

## Project Overview & Architecture

`gamux` is a post-download Steam game manager for Linux (DRM removal via Goldberg Emulator, Lutris registration, Steam shortcuts, and artwork management).

### Package Layout
- `cmd` / [`main.go`](file:///home/staernid/projects/gamux/main.go): CLI entry point using `urfave/cli/v2`. Parses flags and delegates to `engine`.
- [`config`](file:///home/staernid/projects/gamux/config): Global configuration loading (`config.yaml`), default paths, and data models.
- [`detector`](file:///home/staernid/projects/gamux/detector): Game directory scanner, AppID detection (via ACF manifests / `steam_appid.txt`), platform inspection (Linux native vs. Windows PE via Wine/Proton).
- [`engine`](file:///home/staernid/projects/gamux/engine): High-level orchestrator coordinating game detection, GBE patching, Lutris registration, and Steam shortcuts.
- [`gbe`](file:///home/staernid/projects/gamux/gbe): Goldberg Steam Emulator setup (DLL/SO patching, loader env vars, DLC configuration, steam_settings generation).
- [`github`](file:///home/staernid/projects/gamux/github): GitHub API client for fetching and updating GBE fork release assets.
- [`lutris`](file:///home/staernid/projects/gamux/lutris): Lutris game registration (YAML configuration generation and SQLite database updates).
- [`steam`](file:///home/staernid/projects/gamux/steam): Steam Web API client for AppID searching, DLC metadata fetching, and grid artwork downloads.
- [`steamshortcut`](file:///home/staernid/projects/gamux/steamshortcut): Steam non-Steam shortcut writer (binary VDF serialization & fallback `.desktop` entry generation).
- [`util`](file:///home/staernid/projects/gamux/util): File utilities (backup & atomic replacement, archive extraction, hash calculation, process execution).

---

## Core Conventions

1. **Logging Strategy**:
   - Use `log/slog` structured logging (`slog.Info`, `slog.Warn`, `slog.Error`).
   - Do not use `fmt.Println` or standard `log` package in library modules.

2. **Error Handling**:
   - Wrap errors with context: `fmt.Errorf("...: %w", err)`.

3. **Concurrency & Rate Limits**:
   - Use `golang.org/x/sync/errgroup` with worker caps (e.g. `g.SetLimit(10)`) when querying external APIs.
   - Pass `context.Context` through all async/network chains.

4. **File Safety**:
   - Always back up libraries before patching (`util.BackupAndReplace`).
   - Respect `--dry-run` flags; log mutations with `[DRY RUN]` without modifying the filesystem.

5. **Testing**:
   - Mock HTTP requests using custom `http.RoundTripper`.
   - Clean up temporary files with `defer os.RemoveAll(tmpDir)`.

---

## Development Workflow

### Build & Verification Commands
```bash
make test    # Run unit tests across all packages
make lint    # Run go vet and golangci-lint
make build   # Compile native binary to ./gamux
```
