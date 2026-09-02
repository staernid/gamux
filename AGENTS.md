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
   - All Go unit tests must run cleanly under `go test -race ./...`.
   - Clean up temporary files with `defer os.RemoveAll(tmpDir)`.
   - Frontend tests use Vitest (`npm test` in `frontend/`).

6. **No Legacy Compatibility Hacks**:
   - Do not preserve legacy fallback code or redundant dual paths when breaking changes occur.
   - Always migrate local configuration files and game setups directly to the single current standard.

---

## GUI & Frontend Development Protocol

When touching anything under `frontend/` or the Wails bridge in `gui/`:

### UI & Design Guidelines

Keep the design serious, high-contrast, and monochrome:

1. **Monochrome Palette**:
   - Stick to neutral dark surfaces (`#09090b`, `#121214`, `#18181b`, `#222225`, `#27272a`, `#3f3f46`) and clean whites/grays.
   - **No neon aesthetic**: avoid cyan, purple, or pastel gradients, glows, and rainbow tab colors.
   - Color is reserved strictly for meaningful semantic states (e.g. green for active processes, red for destructive actions). Game cover art provides all organic color.

2. **Scalable Typography (`rem`)**:
   - **Never hardcode pixel font sizes** (e.g. `text-[10px]`, `text-[11px]`).
   - Always use `rem`-based Tailwind tokens (`text-xs`, `text-2xs`, `text-3xs`, `text-4xs`) so typography scales with the user's UI scale setting.

3. **Readability & Wrapping**:
   - Never truncate setting descriptions or helper copy with `truncate`. Allow text to wrap naturally (`leading-relaxed`) with readable contrast (`text-neutral-400` or lighter).

---

### The Mandatory 4-Stage GUI Diagnostic Gate

Before making code edits on GUI bugs or async lifecycle issues:

1. **Phase 1: Trace Before Touch**:
   - Try not to guess or blindly edit code.
   - Run `npm run gui:dump` (or `make gui-dump`) to inspect current store snapshots and event timelines.
   - Run `npm run gui:trace -- <scenario>` (or `make gui-trace`) to simulate the failing lifecycle and view the step-by-step event reduction table.
   - Evaluate expressions interactively via `npm run gui:eval -- "window.__GAMUX_DEV__.<expr>"`.

2. **Phase 2: Invariant Audit**:
   - Run `npm run gui:audit` (or `make gui-audit`).
   - Isolate whether the issue originates in:
     - Go backend event dispatch (`events.Bus` / `jobs.Manager`)
     - Wails bridge event routing (`gui/app.go` $\leftrightarrow$ `frontend/src/api/app.js`)
     - Svelte store state reducer (`frontend/src/stores/`)
     - Component-level reactivity or styling.

3. **Phase 3: Targeted Fix & Scenario Unit Test**:
   - Apply the minimal, targeted patch.
   - Add or update scenario/store tests in `frontend/src/debug/debug.test.js` or `frontend/src/**/*.test.js`.

4. **Phase 4: Visual & Invariant Verification Gate**:
     - **Check**: `cd frontend && npm run check` (Svelte type & syntax validation).
     - **Audit & Test**: `make test-frontend` (Vitest test suite + automated invariant audit).
     - **Build**: `make build-gui` (Full asset bundling and Go WebKit binary compilation).

---

## Development Workflow

### One-Command Full Verification

```bash
make check             # Runs linters, Go race tests, Svelte check, Vitest suite, GUI invariant audit, and builds binaries
```

### Diagnostic & Inspection Commands

```bash
make gui-audit         # Run automated GUI store and event invariant audit
make gui-trace         # Run lifecycle trace scenario in headless browser and print event table
make gui-dump          # Dump entire GUI store state, timeline, and audit report as JSON
```

### Individual Target Commands

```bash
make test              # Run both Go and Frontend unit tests (including GUI invariant audit)
make test-go           # Run Go tests with race detection (go test -race ./...)
make test-frontend     # Run frontend Vitest suite and GUI invariant audit
make lint              # Run go vet and svelte-check
make build             # Compile bin/gamux and bin/gamux-gui
```
