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
- [`domain`](file:///home/staernid/projects/gamux/domain): Canonical entity models (`domain.Game`) and domain state helpers (`IsClean`, `HasBackups`, `IsLoaderConfigured`).
- [`downloader`](file:///home/staernid/projects/gamux/downloader): High-level multi-depot downloader orchestrating key retrieval, manifest decryption, and chunk downloads.
- [`engine`](file:///home/staernid/projects/gamux/engine): High-level orchestrator coordinating game scanning, pipeline stage dispatching, and library queries.
- [`events`](file:///home/staernid/projects/gamux/events): Thread-safe pub/sub broker (`events.Bus`) distributing typed lifecycle events (`task:started`, `task:progress`, `task:stage_change`, `task:completed`, `task:failed`, `task:cancelled`).
- [`frontend`](file:///home/staernid/projects/gamux/frontend): Svelte 5 / Vite desktop UI with reactive task queues, Steam Deck-optimized navigation, and Vitest test suite.
- [`gamecfg`](file:///home/staernid/projects/gamux/gamecfg): Per-game configuration persistence (`steam_settings/launch_profile.json`).
- [`gbe`](file:///home/staernid/projects/gamux/gbe): Goldberg Steam Emulator setup (DLL/SO patching, loader env vars, DLC configuration, steam_settings generation).
- [`github`](file:///home/staernid/projects/gamux/github): GitHub API client for fetching and updating GBE fork release assets.
- [`gui`](file:///home/staernid/projects/gamux/gui): Wails desktop bridge adapter communicating between Go engine/events/jobs and the WebKit frontend.
- [`jobs`](file:///home/staernid/projects/gamux/jobs): Background task supervisor (`jobs.Manager`) managing concurrency limits, job lifecycles, and cooperative context cancellation.
- [`lutris`](file:///home/staernid/projects/gamux/lutris): Lutris game registration (YAML configuration generation and SQLite database updates).
- [`manifest`](file:///home/staernid/projects/gamux/manifest): Depot manifest parsing, Lua/RevoBD/SteamDepotDB key resolution, file checksum verification, and integrity reporting.
- [`pipeline`](file:///home/staernid/projects/gamux/pipeline): Composable game setup pipeline and stages (`pipeline.Stage`, `pipeline.Pipeline`, `pipeline.Context`).
- [`runner`](file:///home/staernid/projects/gamux/runner): Subprocess management, runner construction (Wine, Proton, UMU, Linux native), and process supervision.
- [`steam`](file:///home/staernid/projects/gamux/steam): Steam Web & Store API client for AppID searching, DLC metadata fetching, changelogs, and grid artwork downloads.
- [`steamless`](file:///home/staernid/projects/gamux/steamless): Steamless-rs automated extraction and SteamStub DRM executable unpacking.
- [`tools`](file:///home/staernid/projects/gamux/tools): Modular post-processing tool manager and update inspector.
- [`ui`](file:///home/staernid/projects/gamux/ui): CLI dashboards, guided wizards, help formatters, and terminal progress indicators.
- [`util`](file:///home/staernid/projects/gamux/util): File utilities (backup & atomic replacement, archive extraction, hash calculation, process execution).

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
│  • Job Supervisor (jobs/manager.go - Concurrency & Worker)  │
│  • Pub/Sub Broker (events/events.go - Typed Telemetry)      │
└──────────────┬──────────────────────────────────────────────┘
               │ Dispatches Context
               ▼
┌─────────────────────────────────────────────────────────────┐
│                 Domain & Pipeline Execution                 │
│  • Pipeline Runner (pipeline/runner.go)                     │
│  • Composable Stages (pipeline/stages_impl.go)              │
│    [Scan] ➔ [Manifest] ➔ [Normalize] ➔ [Steamless] ➔ [GBE] ➔ [Lutris]
│  • Canonical Domain Entity (domain.Game)                    │
└──────────────┬──────────────────────────────────────────────┘
               │ Operates On
               ▼
┌─────────────────────────────────────────────────────────────┐
│                 Low-Level Adapters & Services               │
│  • gbe  • steamless  • lutris  • downloader  • manifest     │
│  • steam (Store/Web API)  • github (Releases)  • runner     │
└─────────────────────────────────────────────────────────────┘
```

---

## Architectural Patterns & Concrete Recipes

### Recipe 1: Adding a New Game Operation Step
Implement a `pipeline.Stage` (and optionally `pipeline.RollbackableStage`) in [`pipeline/stages_impl.go`](file:///home/staernid/projects/gamux/pipeline/stages_impl.go):

```go
type MyCustomStage struct{}

func (s *MyCustomStage) Name() string {
    return "My Custom Setup Step"
}

func (s *MyCustomStage) Execute(ctx context.Context, pctx *Context) error {
    if pctx.DryRun {
        slog.Info("[DRY RUN] Would execute custom step", "game", pctx.Game.Name)
        return nil
    }
    // Perform mutations or checks against pctx.Game
    return nil
}

// Optional rollback on downstream stage failure:
func (s *MyCustomStage) Rollback(ctx context.Context, pctx *Context) error {
    slog.Info("Reverting custom setup step", "game", pctx.Game.Name)
    return nil
}
```

### Recipe 2: Submitting a Background Asynchronous Job
Instantiate a `jobs.Job` and submit it to the supervisor via `e.JobManager`:

```go
job := jobs.NewJob(
    jobs.OpDownload,
    "Download Cyberpunk 2077",
    "Cyberpunk 2077",
    targetDir,
    "1091500",
    func(ctx context.Context, reporter *jobs.ProgressReporter) error {
        reporter.UpdateStage("Downloading Chunks", "Connecting to Steam CDN...")
        reporter.UpdateProgress(50.0, 10485760, 5, 100, 200, 5000000, 10)
        return nil
    },
)
e.JobManager.Submit(job)
```

### Recipe 3: Testing Network Clients Safely
Inject scoped mock HTTP clients through `context.Context` without modifying global state:

```go
mockClient := testutil.NewMockHTTPClient(func(req *http.Request) (*http.Response, error) {
    return &http.Response{
        StatusCode: http.StatusOK,
        Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
        Header:     make(http.Header),
    }, nil
})

ctx := steam.WithHTTPClient(context.Background(), mockClient)
// All downstream Steam API calls using ctx will now use mockClient safely in parallel
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

### The Mandatory 4-Stage GUI Diagnostic Gate
Before making code edits on any GUI bug or async lifecycle issue:

1. **Phase 1: Trace Before Touch**:
   - Do NOT guess or blindly edit code.
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
   - Every GUI change MUST pass:
     - **Step 1 (Check)**: `cd frontend && npm run check` (Svelte type & syntax validation).
     - **Step 2 (Audit & Test)**: `make test-frontend` (Vitest test suite + automated invariant audit).
     - **Step 3 (Visual Snapshot)**: `make gui-snap` $\rightarrow$ inspect generated `.gui-snapshots/*.png` with `view_file` to visually verify UI alignment and states.
     - **Step 4 (Build)**: `make build-gui` (Full asset bundling and Go WebKit binary compilation).

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
make gui-snap          # Capture full suite of headless UI view snapshots into .gui-snapshots/
```

### Individual Target Commands
```bash
make test              # Run both Go and Frontend unit tests (including GUI invariant audit)
make test-go           # Run Go tests with race detection (go test -race ./...)
make test-frontend     # Run frontend Vitest suite and GUI invariant audit
make lint              # Run go vet and svelte-check
make build             # Compile bin/gamux and bin/gamux-gui
```

