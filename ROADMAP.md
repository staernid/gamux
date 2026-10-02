# Gamux Modernization & Performance Roadmap

This document outlines the phased roadmap for optimizing Gamux's performance, modernizing the frontend architecture to Svelte 5, and upgrading the runtime to Wails v3.

---

## ✅ Phase 0: Svelte 5 Runes Foundation *(Completed)*
- Migrated primary views (`GameHubView`, `DownloadsView`, `TopBar`, `TopBarProfileMenu`) and common UI library (`Icon`, `ConfirmModal`, `PosterArt`, `StepperRow`, `InheritSelect`, `Toast`) to Svelte 5 runes (`$state`, `$derived`, `$props`, `$effect`, `untrack`).
- Resolved infinite update loops on high-frequency download telemetry.
- Zero errors across `svelte-check`, Vitest (58/58 test suites passed), Go race tests, and GUI invariant audit.

---

## ✅ Milestone 1: IPC & Telemetry Throttling *(Completed)*
- **Delta Game Updates**:
  - Implemented `patchGameInStore(gameDir, delta)` in `frontend/src/stores/library/catalog.js` with reference preservation: avoids re-creating the `games` array or re-triggering derived stores when fields have not changed.
  - Optimized `setGameRunning` to route delta updates through `patchGameInStore` and memoize `runningGames` Set references to avoid spurious reactivity cascades.
- **Hard-Clamped Download Telemetry**:
  - Clamped task progress and step telemetry in `tasks/manager.go` to ~6.7 Hz (150ms) while guaranteeing delivery of initial ticks and terminal completion.
  - Filtered `gamux:task-event` in `gui/app.go` to only task/transfer events, and clamped IPC progress events crossing into WebKitGTK.
  - Stripped heavy `domain.Game` struct serialization from non-terminal task progress events.
  - Verified with new Go unit test `TestTasksManager_ProgressThrottling` and Vitest store test.

---

## ✅ Milestone 2: WebKitGTK Memory & Asset Optimization *(Completed)*
- **On-the-Fly Image Thumbnailing**:
  - Implemented high-quality area-average (box filter) image downscaling engine in `util/thumbnail.go` using Go's standard library `image` package.
  - Added thumbnail endpoint in `app.AssetHandler()` (`/art/thumb?id=...&w=300`) with bounded concurrency (semaphore max 4 decodes) and atomic disk caching in `<artDir>/.thumbs/`.
  - Added automatic cache invalidation based on source image modification time (`mtime`).
  - Integrated `getThumbnailUrl` helper and updated `PosterArt.svelte`, `CoverCard.svelte`, `TaskCard.svelte`, and `Carousel3D.svelte` to request appropriately sized thumbnails.
  - Verified with Go unit tests (`TestApp_AssetHandler_Thumbnail`, `TestResizeImage`, `TestGetOrGenerateThumbnail_*`) and Vitest test suite (`PosterArt.test.js`, `game.test.js`).
---

## ✅ Milestone 3: Store Layer Modernization (`.svelte.js` Runes) *(Completed)*
- **Retire `writable()` Stores**:
  - Migrated `frontend/src/stores/` (`catalog.svelte.js`, `tasks.svelte.js`, `profiles.svelte.js`, `config.svelte.js`) from Svelte 4 `writable()` to Svelte 5 rune classes (`$state`, `$derived`).
  - Added transparent backward-compatibility store bridges on catalog and tasks to maintain full compatibility with legacy store subscribers, derivations, and test harnesses.
  - Eliminated Rx-style `subscribe()` boilerplate and `$` store prefixing across components (`TopBar`, `TopBarProfileMenu`, `DownloadsView`, `AudioSettingsCard`, `HubDLCCard`, `ProfilesTab`, `CoopTab`, `CoopSeatCard`, `GameHubView`, `ToolsMaintenanceTab`, `Toast`, and `LibraryView`).
  - Verified with Vitest test suites (all 59 suites / 344 tests passing), zero Svelte diagnostics in `svelte-check`, and passed full `make check` verification gate (Go race tests, linters, frontend tests, GUI invariant audit, and binaries build).

---

## ⚡ Milestone 4: Library Grid Virtualization & View Decomposition
- **Decompose Heavy Settings Views**:
  - Break monolithic draft logic in `SettingsView` into decoupled section drafts managed directly by their respective tab components.

---

## ✅ Milestone 5: Wails v3 Backend Transition *(Completed)*
- **Application Bootstrap**:
  - Upgraded `cmd/gamux-gui/main.go` from Wails v2 to `github.com/wailsapp/wails/v3/pkg/application` (v3.0.0-beta.27).
  - Wired asset middleware for custom `/art/` and `/local-img/` paths and window closing hooks to preserve close-to-tray functionality.
- **Native Single-Instance & Tray**:
  - Replaced legacy D-Bus SNI engine (`tray_linux.go`, `tray_other.go`) with Wails v3 native cross-platform `SystemTray` in `gui/tray.go`.
  - Configured native `SingleInstanceOptions` with URL/command forwarding to active window.
- **Runtime Bridge & Build Tooling**:
  - Migrated `frontend/src/api/app.js` to `@wailsio/runtime` with transparent fallback for headless CI and Vitest.
  - Updated `Makefile` with `-tags desktop,production,gtk3,webkit2_41` and pruned `wails/v2` and legacy transitive dependencies from `go.mod`.
  - Verified cleanly via `make check` (Go race tests, linters, all 59 Vitest test suites / 344 tests passing, GUI invariant audit, and binary compilation).
