# Handover Briefing & Remaining Task: Ticket 023

## Summary of Work Completed This Session

### 1. Ticket 014: Split Oversized View Components (`docs/tickets/done/014-split-oversized-components.md`)
- **`HubBetasTab.svelte`** (originally 568 lines -> **281 lines**):
  - Extracted [HubBetasStatusBanner.svelte](file:///home/staernid/projects/gamux/frontend/src/components/views/hub/HubBetasStatusBanner.svelte)
  - Extracted [HubBranchList.svelte](file:///home/staernid/projects/gamux/frontend/src/components/views/hub/HubBranchList.svelte)
  - Extracted [HubPrivateBetasCard.svelte](file:///home/staernid/projects/gamux/frontend/src/components/views/hub/HubPrivateBetasCard.svelte)
  - Pure sorting, filtering, and absence-aware build mismatch logic extracted into [betas.js](file:///home/staernid/projects/gamux/frontend/src/utils/betas.js) with 100% test coverage in [betas.test.js](file:///home/staernid/projects/gamux/frontend/src/utils/betas.test.js) and [HubBetasComponents.test.js](file:///home/staernid/projects/gamux/frontend/src/components/views/hub/HubBetasComponents.test.js).
- **`SettingsView.svelte`** (originally 684 lines -> **242 lines**):
  - Extracted [SettingsHeader.svelte](file:///home/staernid/projects/gamux/frontend/src/components/views/settings/SettingsHeader.svelte)
  - Extracted [SettingsNav.svelte](file:///home/staernid/projects/gamux/frontend/src/components/views/settings/SettingsNav.svelte)
  - Extracted [SettingsAdvancedTab.svelte](file:///home/staernid/projects/gamux/frontend/src/components/views/settings/SettingsAdvancedTab.svelte)
  - Tab routing parameters, adjacency cycling, and default draft constants moved to [settings.js](file:///home/staernid/projects/gamux/frontend/src/utils/settings.js) with test coverage in [settings.test.js](file:///home/staernid/projects/gamux/frontend/src/utils/settings.test.js) and [SettingsViewComponents.test.js](file:///home/staernid/projects/gamux/frontend/src/components/views/settings/SettingsViewComponents.test.js).
- **`GameHubView.svelte`** (originally 717 lines -> **288 lines**):
  - Extracted [HubNavTabs.svelte](file:///home/staernid/projects/gamux/frontend/src/components/views/hub/HubNavTabs.svelte)
  - Extracted [HubTabContent.svelte](file:///home/staernid/projects/gamux/frontend/src/components/views/hub/HubTabContent.svelte)
  - Navigation tab cycling and task matching pure logic moved to [hub.js](file:///home/staernid/projects/gamux/frontend/src/utils/hub.js) with unit tests in [hub.test.js](file:///home/staernid/projects/gamux/frontend/src/utils/hub.test.js).
  - Launching, downloading, trashing, updating, and metadata operations moved to [hubActions.js](file:///home/staernid/projects/gamux/frontend/src/utils/hubActions.js).
  - Component tests added in [HubNavTabs.test.js](file:///home/staernid/projects/gamux/frontend/src/components/views/hub/HubNavTabs.test.js).
- **Verification**: `make check` passed with zero errors, race tests passing, and all components well below the 350-line target. Committed as `feat(ui): split oversized view components (ticket 014)`.

---

### 2. Ticket 019: Tailwind v3 vs v4 Discrepancy (`docs/tickets/done/019-tailwind-version-discrepancy.md`)
- Migrated frontend build to **Tailwind CSS v4** (`tailwindcss@^4`) with `@tailwindcss/vite`.
- Converted design configuration to CSS-first `@theme` block in [style.css](file:///home/staernid/projects/gamux/frontend/src/style.css):
  - Theme variables defined for `--color-surface-*` tokens and small font sizing (`--text-2xs`, `--text-3xs`, `--text-4xs`).
  - Added `@custom-variant dark (&:where(.dark, .dark *));` to support class-based dark mode (`html.dark`).
- Removed `postcss.config.js`, `tailwind.config.js`, `postcss`, and `autoprefixer`.
- Migrated classes across 70 template files via `@tailwindcss/upgrade`.
- Verified full test suite (`npm test`: 78 test files, 463 tests passing) and `make gui-audit` (0 errors, 0 warnings). Committed as `feat(ui): migrate to Tailwind CSS v4 and @tailwindcss/vite (ticket 019)`.

---

## Remaining Task: Ticket 023 — Generate Typed Wails v3 Bindings

- **Ticket File**: [`docs/tickets/023-typed-generated-bindings.md`](file:///home/staernid/projects/gamux/docs/tickets/023-typed-generated-bindings.md)
- **Goal**: Replace the hand-rolled bindings manifest (`frontend/src/api/bindings.gen.js` and `gui/bindings_manifest_test.go`) with generated typed Wails v3 bindings committed under `frontend/src/api/gen/`, while keeping the `AppAPI` wrapper and Vitest test suite working seamlessly.

### Environment & Tool State
- `wails3` CLI `v3.0.0-beta.27` is **installed and functional** in `/home/staernid/.local/share/go/bin/wails3` (compiled with `-tags gtk3,webkit2_41`).
- The project requires build tags `-tags desktop,production,gtk3,webkit2_41` because host environment provides GTK3 and WebKit2 4.1.

### Step-by-Step Implementation Guide for Next Session

1. **Investigate bound types / service registration in `gui/`**:
   - Inspect [gui/app.go](file:///home/staernid/projects/gamux/gui/app.go) and [cmd/gamux-gui/main.go](file:///home/staernid/projects/gamux/cmd/gamux-gui/main.go) to see how `gui.App` is bound to the Wails application:
     ```go
     // In Wails v3, services are bound with application.NewService(&gui.App{})
     ```
   - Test generating bindings with `wails3`:
     ```bash
     wails3 generate bindings -ts -d frontend/src/api/gen -f "-tags desktop,production,gtk3,webkit2_41" ./cmd/gamux-gui ./gui
     ```
   - Check the generated TypeScript files in `frontend/src/api/gen/`.

2. **Adapt `frontend/src/api/app.js`**:
   - Inspect [frontend/src/api/app.js](file:///home/staernid/projects/gamux/frontend/src/api/app.js).
   - Currently, `AppAPI` wraps `window.wails?.Call` or `runtime.Call` and checks against `bindings.gen.js`.
   - Update `AppAPI` to delegate calls to the generated service methods (or use them directly), ensuring that Vitest mocks (`vi.spyOn(AppAPI, ...)`) continue to work across all 78 existing test files.

3. **Delete legacy manifest & test**:
   - Delete `frontend/src/api/bindings.gen.js`.
   - Delete `gui/bindings_manifest_test.go`.
   - In accordance with Core Tenet 3 (*Single Path Forward*), do not keep redundant validation layers.

4. **Update `Makefile`**:
   - Update the `bindings` recipe in [Makefile](file:///home/staernid/projects/gamux/Makefile) to run:
     ```makefile
     bindings:
     	wails3 generate bindings -ts -d frontend/src/api/gen -f "-tags desktop,production,gtk3,webkit2_41" ./cmd/gamux-gui ./gui
     ```

5. **Verify and Finalize**:
   - Run `make check-fast` and `make check`.
   - Move `docs/tickets/023-typed-generated-bindings.md` to `docs/tickets/done/023-typed-generated-bindings.md`.
   - Commit all changes: `git add -A && git commit -m "feat(ipc): generate typed Wails v3 bindings (ticket 023)"`.
