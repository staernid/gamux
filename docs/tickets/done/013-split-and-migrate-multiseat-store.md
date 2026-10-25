# 013 — Split multiseat.js (870 lines) and migrate to runes

**Area:** `stores/multiseat.js` · **Size:** L (split into two sessions: split first, migrate second)

## Context
`stores/multiseat.js` is the largest store. It mixes:
- pure helpers (`getActiveSession`, `resolveMultiGamepadAssignments`,
  `resolveGamepadAssignments`, `hydrateStagingFromConfig`, lines 1–~140)
- one big `createMultiseatStore()` factory (~140–860) built on `writable`
  (device polling, staging config, session launch/stop)

The Coop cards in `components/views/settings/cards/` and `CoopTab.svelte`
consume it.

## Session A: split (no behaviour change)
1. Move the pure helpers to `stores/multiseat/assignments.js` (plus a test
   file; move the relevant cases from `multiseat.test.js`).
2. Break the factory into logical parts (devices, staging, session) inside
   `stores/multiseat/`, re-exported from `stores/multiseat/index.js`, so
   `import { multiseatStore } from '.../stores/multiseat.js'` callers switch to
   the new path with no other changes.

## Session B: runes
Convert each part to a `$state` class as in ticket 010. Keep `subscribe` for
`$multiseatStore` users. Polling timers must be started and stopped explicitly
(`start()`/`stop()` called from `CoopTab` mount/unmount), not by module-level
side effects.

## Done when
No file over ~300 lines in `stores/multiseat/`, no `svelte/store` import, and
`make check-fast` is green.
