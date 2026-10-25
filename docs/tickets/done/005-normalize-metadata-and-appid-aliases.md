# 005 — Remove appId/metadata alias chains (InstallModal, catalog, MetadataLinker)

**Area:** several · **Size:** M

## Context
More snake_case and alias guessing outside the normalizers:
- `components/stage/InstallModal.svelte` lines ~146 and ~248:
  `game?.appId || game?.app_id || game?.AppID`. Find out what `game` is there:
  a normalized library game, or a raw store search result
  (`AppAPI.searchStoreGames`). Normalize at the source.
- `stores/library/catalog.svelte.js`:
  - `setGameRunning(... extra)` around line 40 reads `extra.profileId ||
    extra.profile_id`, `extra.appId || extra.app_id`, etc. Find the callers
    (likely the `gamux:game-status` event handler in `App.svelte`) and
    normalize the event payload once, there.
  - `updateGameInStore` delta handling around line 130 maps `last_played`,
    `is_patched`, `game_dir`, `app_id` by hand. Callers should pass
    `normalizeGame(raw)` output (or a camelCase partial) instead.
- `components/domain/MetadataLinker.svelte` ~62/180: `display_name` from
  `listMetadataProviders`. Add a `normalizeMetadataProvider`.
- `components/views/GameHubView.svelte` ~165: `meta.header_image` from
  `AppAPI.getGameMetadata`. Normalize the metadata result. Ticket 008 moves
  this logic into the catalog store; if you do 008 first, normalize there.

## Do
Handle each bullet: one normalizer per entity, called where the data enters,
no `a || b_c || C` chains. Add or update unit tests for each new normalizer.

## Done when
`grep -rnE "(app_id|AppID|profile_id|seat_id|instance_id|game_name|display_name|header_image)" frontend/src --include=*.svelte`
has no hits, and every remaining hit in `.js` is inside a `normalize*`
function. Also update the comment at the top of `types/domain.d.ts`: it still
says "backwards-compatible snake_case aliases", which contradicts AGENTS.md.
`make check-fast` is green.
