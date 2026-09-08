# Developer & AI Assistant Guide (`AGENTS.md`)

Essential guidelines, architecture patterns, and conventions for development on `gamux`.

## Project Overview & Architecture

`gamux` is a game manager for Linux (DRM removal, direct host runners via UMU/Wine/Proton, artwork management, savegame management).

---

## Architecture & Data Flow

```text
┌─────────────────────────────────────────────────────────────┐
│                    Presentation Layer                       │
│  • CLI (cmd/gamux, ui)     • GUI Frontend (Svelte 5 / Vite) │
└──────────────┬───────────────────────────────▲──────────────┘
               │ Invokes                       │ Subscribes
               ▼                               │ (gamux:task-event)
┌──────────────────────────────────────────────┴──────────────┐
│                    Orchestration Layer                      │
│  • Engine (engine/engine.go)  • Wails Bridge (gui/app.go)   │
│  • Task Supervisor (tasks/manager.go - Concurrency & Worker)│
│  • Pub/Sub Broker (events/events.go - Typed Telemetry)      │
└──────────────┬──────────────────────────────────────────────┘
               │ Dispatches Context
               ▼
┌─────────────────────────────────────────────────────────────┐
│                 Domain & Pipeline Execution                 │
│  • Pipeline Runner (pipeline/runner.go)                     │
│  • Composable Stages (pipeline/stages_impl.go)              │
│    [Scan] ➔ [Manifest] ➔ [Normalize] ➔ [Steamless] ➔ [GBE]
│  • Canonical Domain Entity (domain.Game)                    │
└──────────────┬──────────────────────────────────────────────┘
               │ Operates On
               ▼
┌─────────────────────────────────────────────────────────────┐
│                 Low-Level Adapters & Services               │
│  • gbe  • steamless  • store  • downloader  • manifest      │
│  • steam (Store/Web API)  • github (Releases)  • runner     │
└─────────────────────────────────────────────────────────────┘
```

---

## Core Tenets

1. **Opinionated Simplicity**: Minimal user decisions; sensible defaults over toggles. Discard legacy launcher conventions freely.
2. **Unified Lifecycle**: Previews, stubs, and installed titles share one entity model. Degrade missing features gracefully; never lock users out of metadata or configuration.
3. **Single Path Forward**: Migrate schemas and configs directly. No legacy fallbacks, shims, or dual execution paths.
4. **Absence is Not an Error**: Treat `0`, `""`, and `unknown` as absence, not failure. Never show mismatch alerts on unpopulated sentinels.
5. **Idempotent & Reversible**: Pipeline stages must not overwrite user config. Always back up files before patching (`util.BackupAndReplace`).

---

## Conventions

- **Logging**: Use `log/slog` structured logging (`slog.Info`, `slog.Warn`, `slog.Error`). No `fmt.Println` or stdlib `log` in library modules.
- **Errors**: Wrap with context: `fmt.Errorf("...: %w", err)`.
- **Concurrency**: Use `golang.org/x/sync/errgroup` with worker limits (`g.SetLimit(10)`). Pass `context.Context` through async/network chains.
- **Testing**: All Go unit tests must pass under `go test -race ./...`. Clean up temp files with `defer os.RemoveAll(tmpDir)`. Frontend tests use Vitest (`npm test` in `frontend/`).
- **UI Design**: Monochrome dark surfaces (`#09090b` to `#3f3f46`). Semantic color only (active/destructive states). Game cover art provides organic color.
- **Typography & Layout**: Never hardcode pixel font sizes. Use Tailwind `rem` tokens (`text-xs`, `text-2xs`). Never truncate descriptions; allow natural wrapping (`leading-relaxed`).

---

## Diagnostics & Verification

### Inspection & Diagnostics
- `make gui-dump` — Dump GUI store state, event timeline, and invariant report as JSON.
- `make gui-trace` — Run lifecycle trace scenario in headless browser and print event table.
- `make gui-audit` — Run automated GUI store and event invariant audit.

### Full Verification
```bash
make check             # Runs linters, Go race tests, svelte-check, Vitest suite, GUI invariant audit, and builds binaries
```

### Individual Commands
- `make test-go` — Run Go tests with race detection (`go test -race ./...`).
- `make test-frontend` — Run Vitest suite and GUI invariant audit.
- `make lint` — Run `go vet` and `svelte-check`.
- `make build` — Compile `bin/gamux` and `bin/gamux-gui`.

