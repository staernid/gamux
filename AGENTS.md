# Developer & AI Assistant Guide (`AGENTS.md`)

Essential guidelines, architecture patterns, and conventions for development on `gamux`.

## Project Overview & Architecture

`gamux` is a post-download Steam game manager for Linux (DRM removal via Goldberg Emulator, Lutris registration, and artwork management).

### Package Layout
- [`cache`](file:///home/staernid/projects/gamux/cache): Fast disk and in-memory TTL caching for Steam game metadata, AppIDs, and patch note changelogs.
- [`cmd/gamux`](file:///home/staernid/projects/gamux/cmd/gamux): CLI entry point (`bin/gamux`), flag definitions, and interactive terminal subcommands.
- [`cmd/gamux-gui`](file:///home/staernid/projects/gamux/cmd/gamux-gui): Wails desktop application entry point (`bin/gamux-gui`).
- [`config`](file:///home/staernid/projects/gamux/config): Global shared configuration loading (`config.json`), default paths, and data models.
- [`detector`](file:///home/staernid/projects/gamux/detector): Game directory scanner, AppID detection (via ACF manifests / `steam_appid.txt`), platform inspection (Linux native vs. Windows PE via Wine/Proton).
- [`downloader`](file:///home/staernid/projects/gamux/downloader): High-level multi-depot downloader orchestrating key retrieval, manifest decryption, and chunk downloads.
- [`engine`](file:///home/staernid/projects/gamux/engine): High-level orchestrator coordinating game detection, GBE patching, Lutris registration, step progress reporting, and depot verification.
- [`frontend`](file:///home/staernid/projects/gamux/frontend): Svelte/Vite web application assets and Go embed bridge.
- [`gbe`](file:///home/staernid/projects/gamux/gbe): Goldberg Steam Emulator setup (DLL/SO patching, loader env vars, DLC configuration, steam_settings generation).
- [`github`](file:///home/staernid/projects/gamux/github): GitHub API client for fetching and updating GBE fork release assets.
- [`gui`](file:///home/staernid/projects/gamux/gui): Wails desktop bridge adapter communicating between the Go engine/downloader and the WebKit frontend.
- [`lutris`](file:///home/staernid/projects/gamux/lutris): Lutris game registration (YAML configuration generation and SQLite database updates).
- [`manifest`](file:///home/staernid/projects/gamux/manifest): Depot manifest parsing, Lua/RevoBD/SteamDepotDB key resolution, file checksum verification, and integrity reporting.
- [`steam`](file:///home/staernid/projects/gamux/steam): Steam Web & Store API client for AppID searching, DLC metadata fetching, changelogs, and grid artwork downloads.
- [`steamless`](file:///home/staernid/projects/gamux/steamless): Steamless-rs automated extraction and SteamStub DRM executable unpacking.
- [`ui`](file:///home/staernid/projects/gamux/ui): CLI dashboards, guided wizards, help formatters, and terminal progress indicators.
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

6. **No Legacy Compatibility Hacks**:
   - Do not preserve legacy fallback code or redundant dual paths when breaking changes occur.
   - Always migrate local configuration files and game setups directly to the single current standard.

---

## Development Workflow

### Build & Verification Commands
```bash
make test    # Run unit tests across all packages
make lint    # Run go vet and golangci-lint
make build   # Compile native binary to ./gamux
```
