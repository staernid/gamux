# 008 — Move placeholder-title metadata fetch out of GameHubView

**Area:** `GameHubView.svelte` → `stores/library/catalog.svelte.js` · **Size:** S

## Context
GameHubView ~157–175 has an `$effect` that runs when the selected game has a
placeholder name (`isPlaceholderTitle`). It calls `AppAPI.getGameMetadata`,
then writes to both `selectedGame` and the library store. Problems:
- A view component fires network requests and writes global state from an
  effect. It re-fires every time `game` changes identity, including after its
  own write.
- There's no dedupe: opening the same game twice fetches twice, and errors are
  silently swallowed (`.catch(() => {})`).
- It reads snake_case `meta.header_image` in a component.

## Do
1. Add a method to the catalog store, e.g. `catalog.resolvePlaceholderTitle(game)`,
   that keeps an in-flight `Set` keyed by `provider:id` (no concurrent duplicate
   fetches), normalizes the metadata response, and updates the game through the
   store's own update path (which also updates `selectedGame` if that's how
   selection works; check `stores/library/selection.js`).
2. Log failures with `console.warn` once. This is a background enrichment, so
   no toast.
3. In GameHubView, replace the effect with a one-line call to the store method
   when the game is selected (still an `$effect`, but with no state writes in
   the component).
4. Add a store unit test: two calls for the same game make one `AppAPI` call.

## Done when
GameHubView no longer imports `updateGameInStore` for this purpose, the test
exists, and `make check-fast` is green.
