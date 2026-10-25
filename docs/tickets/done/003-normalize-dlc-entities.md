# 003 — Normalize DLC entities

**Area:** `components/views/hub/cards/HubDLCCard.svelte` · **Size:** S

## Context
HubDLCCard renders backend DLC objects directly and reads `dlc.app_id` (around
lines 413–417, inside `{#each dlcs as dlc (dlc.app_id)}`). Components should
only get camelCase entities.

## Do
1. Find where `dlcs` comes from (an `AppAPI.*` call, or a field on the game).
   Find the Go struct behind it (`grep -rn "type .*DLC" --include=*.go`) and
   note its JSON tags.
2. Add `normalizeDlc(raw)` in `frontend/src/utils/game.js`, next to
   `normalizeGame`, mapping every field the UI uses to camelCase (`appId`,
   ...). Add a `DLC` interface to `frontend/src/types/domain.d.ts`.
3. Normalize where the data enters (store or the fetch in the card), then use
   camelCase in the template.
4. Update `HubDLCCard.test.js` fixtures: raw snake_case goes into the mock,
   camelCase comes out.

## Done when
No snake_case property access remains in HubDLCCard, there's a unit test for
`normalizeDlc` in `utils/game.test.js`, and `make check-fast` is green.
