# 011 — Migrate library stores (paths, selection, query, actions) to runes

**Area:** `stores/library/*.js` · **Size:** M · **Do after:** 010 (same pattern)

## Context
`stores/library/catalog.svelte.js` already uses runes, but its siblings
`paths.js`, `selection.js`, `query.js` and `actions.js` still use
`svelte/store`. `selectedGame` (from `selection.js`) is used in ~15 files, and
`libraryPaths` in ~7. `LibraryView.svelte` and `App.svelte` also import
`svelte/store` directly (`get`, `derived`).

Note: `paths.js` persists to `localStorage` through a `subscribe` side effect.
In runes, use an explicit setter that writes localStorage, not an `$effect`
inside the store module.

## Do
1. Migrate one file at a time and run `npm test` in `frontend/` after each.
2. Use the same class-with-`$state` + `subscribe` pattern as ticket 010.
3. Replace `get(store)` calls with direct property reads.
4. Keep exported names stable where you can, to limit churn.

## Done when
No file in `stores/library/` imports `svelte/store`, and `make check-fast` is
green.
