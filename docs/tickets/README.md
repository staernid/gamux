# Frontend Improvement Tickets

Self-contained work items from a frontend review (2026-10). Each ticket can be
done in one session. Pick a few, finish them, delete the ticket file (or move it
to `docs/tickets/done/`) in the same commit.

## Before you start (every ticket)

1. Read `AGENTS.md` at the repo root. Especially: Core Tenet 3 (*Single Path
   Forward*, no fallbacks/shims) and the *Frontend Domain & State* convention.
2. Frontend lives in `frontend/src`. Svelte 5 runes, plain JS + JSDoc (no TS
   files), Tailwind **v3** (see ticket 019 for the v4 discrepancy).
3. Verify with `make check-fast` before finishing. It must be green.
4. Don't widen scope. If you find something else, write a new ticket here.

## How the IPC boundary works now (read once)

- `frontend/src/api/app.js` exports `AppAPI`. Calls map to Go methods on
  `gui.App` (`gui/*.go`): `AppAPI.inspectGame(a, b)` → `App.InspectGame`.
- `frontend/src/api/gen/` contains **generated typed Wails v3 bindings** (`make bindings`). It lists
  every bound method, typed parameters, return models, and its exact argument count. Unknown methods are
  `undefined`; wrong argument counts throw, in tests too (Wails v3 is strict).
- After adding/removing/changing a Go `App` method signature: run `make bindings`.
- In Vitest/browser mode calls go to `DEFAULT_MOCKS` in `app.js`. A bound method
  without a mock throws. Add a mock if a component needs one in tests.
- Backend JSON is **snake_case** (Go `json:"game_dir"` tags; also persisted on
  disk, so do NOT change Go tags). Convert to camelCase in store normalizers.

## Suggested order

Tickets are numbered roughly by priority. Those in the same group can be done
in any order.

| Group | Tickets |
|---|---|
| Normalize at the boundary | 001–005 |
| Shared task helpers / GameHubView cleanup | 006–009 |
| Store migration to runes | 010–013 |
| Component splits | 014 |
| UX / quality | 015–018, 020–022 |
| Needs a human decision first | 019, 023 |
