# Developer & AI Assistant Guide (`AGENTS.md`)

Essential guidelines, architecture patterns, and conventions for development on `gamux`.

## System Architecture

- **Presentation**: CLI (`cmd/gamux`, `ui`) & GUI (Wails v3, Svelte 5, Tailwind v4).
- **Orchestration**: `Engine` (`engine/`), `App` (`gui/`), task workers (`tasks/`), telemetry bus (`events/`), and profile management (`profiles/`).
- **Domain & Pipeline**: Composable pipeline operations (`pipeline/`: `[Scan] -> [Manifest] -> [Normalize] -> [Steamless] -> [GBE]`) operating on canonical domain entities (`domain.Game`, `domain.Profile`).
- **Adapters & Services**: `gbe`, `steamless`, `store`, `downloader`, `manifest`, `steam`, `github`, `runner`, `multiseat` (bwrap/audio/gamescope), `saves`.

---

## Core Tenets & Invariants

1. **Opinionated Simplicity**: Minimal user decisions; sensible defaults over toggles. Discard legacy launcher conventions freely.
2. **Unified Lifecycle**: Previews, stubs, and installed titles share one entity model (`domain.Game`). Degrade missing features gracefully; never lock users out of metadata or configuration.
3. **Single Path Forward**: Migrate schemas and configs directly. No legacy fallbacks, shims, or dual execution paths.
4. **Absence is Not an Error**: Treat `0`, `""`, and `unknown` as absence, not failure. Never show mismatch alerts on unpopulated sentinels.
5. **Idempotent & Reversible**: Pipeline stages must not overwrite user config. Always back up files before patching (`util.BackupAndReplace`).
6. **Single Source of Truth**: Steam persona and emulation configs live exclusively in Goldberg's `configs.user.ini` and are symlinked into Wine prefixes (`AppData/Roaming/GSE Saves`). Never duplicate persona fields in `config.json`.
7. **Thread Safety & Immutability**: Cached entities returned by `cache.GetCachedGame` / `detector.Inspect` are shared pointers across concurrent workers. Pipeline operations and mutating tasks must call `game.Clone()` before modifying state.
8. **Prefix Emulation Setup**: In loader mode, Wine prefixes must have `<prefix>/drive_c/Program Files (x86)/Steam/` populated with Goldberg libraries (`steamclient.dll`, `steamclient64.dll`, `GameOverlayRenderer.dll`, `GameOverlayRenderer64.dll`, `steam_settings` link), coupled with `WINEDLLOVERRIDES`.
9. **Bubblewrap Mount Destinations**: `bwrap` fails with `Can't mount on symlink destination` if a bind-mount target is a symlink. Profile and seat sandboxes must use physical directories or environment pointers (`STEAM_EXTRA_COMPAT_TOOLS_PATHS`), never symlinks as mount targets.
10. **Depot Incompatibility & Conflict Reconciliation**: Candidate depot manifests must pass through `manifest.ReconcileDepots` before initiating downloads to resolve mutually exclusive regional depots and deduplicate identical paths. Downloader workers must abort early if conflicting file versions exist for any normalized relative path.

---

## Conventions

- **Logging**: Use `log/slog` structured logging (`slog.Info`, `slog.Warn`, `slog.Error`). No `fmt.Println` or stdlib `log` in library modules.
- **Concurrency**: Use `golang.org/x/sync/errgroup` with worker limits (`g.SetLimit(10)`). Pass `context.Context` through async/network chains.
- **Testing**: All Go unit tests must pass under `go test -race ./...`. Frontend tests use Vitest (`npm test` in `frontend/`).
- **Frontend Domain & State**: Entities crossing the IPC boundary (`Game`, `RunningInstance`, `Profile`, `Task`) must be normalized at store ingestion to canonical `camelCase` schemas defined in `frontend/src/types/domain.d.ts`. In accordance with Core Tenet 3 (*Single Path Forward*), internal frontend entities must not retain redundant legacy `snake_case` aliases.
- **Profiles & Prefix Isolation**: Main profile uses host `$HOME` and `~/Games/umu`. Secondary profiles use isolated sandboxes in `$XDG_DATA_HOME/gamux/profiles/<id>/home` where their prefixes live in `$HOME/Games/umu`. Profile definitions live in `$XDG_CONFIG_HOME/gamux/profiles.json`. The top-right circular avatar is the sole global profile switcher; Settings tab edits the active profile.
- **Modals & Dialogs**: Destructive or critical user prompts must use `modalStore.open('confirm', ...)` with `ConfirmModal.svelte` (`danger: true` auto-focuses cancel). Avoid bespoke inline confirmation buttons.
- **UI Design**: Monochrome dark surfaces (`#09090b` to `#3f3f46`). Semantic color only (active/destructive states). Game cover art provides organic color.
- **Typography & Layout**: Use Tailwind `rem` tokens (`text-xs`, `text-2xs`). Never truncate descriptions; allow natural wrapping (`leading-relaxed`).

---

## Diagnostics & Verification

- `make check` — Primary verification gate: runs linters (`go vet`, `svelte-check`), Go race tests, Vitest suite, GUI invariant audit, and builds binaries.
- `make check-fast` — Rapid iteration gate: runs `check-go` + `check-ui` without binary compilation overhead (<5s).
- `make check-go` — Go formatting check, `go vet`, `golangci-lint`, and fast unit tests.
- `make check-ui` — Svelte type check (`svelte-check`), full Vitest suite, and GUI invariant audit.
- `make test-go-fast` — Runs all Go unit tests without `-race` overhead.
- `make gui-dump` — Dump GUI store state, event timeline, and invariant report as JSON.
- `make gui-trace` — Run lifecycle trace scenario in headless browser and print event table.
- `make gui-audit` — Run automated GUI store and event invariant audit.

---

## Subsystem Invariants & Reference Docs

- [`docs/notes/performance-and-isolation-invariants.md`](docs/notes/performance-and-isolation-invariants.md) — Operational invariants for downloader chunk I/O, warm cache sentinel signatures, and multiseat evdev permissions.
- [`docs/notes/workshop-mod-management.md`](docs/notes/workshop-mod-management.md) — Workshop and mod structure design notes.

