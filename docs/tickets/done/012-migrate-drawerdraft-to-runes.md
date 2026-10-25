# 012 — Migrate drawerDraft.js to runes

**Area:** `stores/drawerDraft.js` · **Size:** S–M

## Context
The per-game config draft is a `writable` with custom methods (`init`, field
setters, ...) and has good test coverage (`drawerDraft.test.js`, 262 lines).
It's used in 5 files. Ticket 009 adds dirty tracking to it, so coordinate: if
009 is done, keep its API; if not, do this first and 009 becomes simpler.

## Do
Rename it to `drawerDraft.svelte.js`, reimplement it with `$state` (deep
reactivity means nested field writes are tracked, so drop the immutable-spread
updates), keep `subscribe` for `$drawerDraft` users, and update imports.

## Done when
It doesn't import `svelte/store`, the existing tests pass, and
`make check-fast` is green.
