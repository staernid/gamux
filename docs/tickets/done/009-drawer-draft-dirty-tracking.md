# 009 — Replace JSON.stringify-based autosave in GameHubView

**Area:** `GameHubView.svelte`, `stores/drawerDraft.js` · **Size:** M

## Context
GameHubView ~138–155 decides when to autosave the per-game config draft by
comparing `JSON.stringify($drawerDraft)` snapshots inside two `$effect`s
(`initialDraftJSON`, `triggerAutoSave` with a 400 ms debounce). This is fragile
(key order, the effects depend on each other) and the logic belongs in the
draft store, not the view.

## Do
1. Read `stores/drawerDraft.js` and its tests first.
2. Give the store explicit dirty tracking: `init(game, globalCfg)` stores a
   baseline, mutations mark it dirty, and add a `markSaved()`. Expose an
   `isDirty` value (a readable store, or a getter if ticket 012 has already
   moved it to runes).
3. Move the debounced autosave into the store or a small helper
   (`createAutosave(draft, saveFn, delay)`), so the view only wires it up.
4. Delete `initialDraftJSON` and both effects from GameHubView.
5. Test: changing a field triggers exactly one save after the debounce, and
   re-initialising for another game does not trigger a save. Use
   `vi.useFakeTimers()`.

## Done when
There are no `JSON.stringify` comparisons in GameHubView, the tests are added,
and `make check-fast` is green.
