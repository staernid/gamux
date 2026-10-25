# 010 — Migrate notifications.js and modal.js to runes

**Area:** `stores/notifications.js`, `stores/modal.js` · **Size:** M

## Context
The codebase mixes two state systems. `svelte/store` `writable`s are in 11
modules; rune classes in `*.svelte.js` are in 5 (see `stores/profiles.svelte.js`,
`stores/tasks.svelte.js`, `stores/config.svelte.js` for the target pattern:
a class with `$state` fields, exported as a singleton). The rune stores also
keep a `subscribe()` method for `$store` compatibility. Check how
`ProfilesState` does it and copy that.

`notificationStore` is used in ~25 files and `modalStore` in ~17, mostly via
method calls (`notificationStore.error(...)`, `modalStore.open('confirm', ...)`).
Keep the method API the same so callers don't change.

## Do
1. Rename `notifications.js` → `notifications.svelte.js` (runes only work in
   `.svelte.js`) and reimplement it with `$state`. Same for `modal.js`.
2. Update imports (`grep -rn "stores/notifications.js\|stores/modal.js"`).
3. Keep `subscribe` for now if components use `$notificationStore` / `$modalStore`.
   Removing it is part of ticket 013.
4. Existing tests (`notifications.test.js`, `ConfirmModal.test.js`,
   `Toast.test.js`) must pass unchanged, apart from import paths.

## Done when
Neither file imports `svelte/store`, and `make check-fast` is green.
