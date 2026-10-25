# 006 — Shared helpers for "task belongs to game" and status checks

**Area:** `stores/tasks.svelte.js`, several components · **Size:** S

## Context
`stores/tasks.svelte.js` already exports `isActiveStatus`, `isQueuedStatus`,
`isPausedStatus` and `isTerminalStatus`. Several components ignore them and
write out status lists by hand:
- `components/views/GameHubView.svelte` ~61 and ~250
- `components/views/hub/HubBetasTab.svelte` ~334
- `components/views/settings/ToolsMaintenanceTab.svelte` ~49
- `debug/debugBridge.js` ~157, ~179

The "does this task belong to this game?" check (same `gameDir` via
`isSamePath`, or same non-zero `appId`) is also copied in GameHubView twice and
in DownloadsView.

## Do
1. In `stores/tasks.svelte.js`, add and export:
   - `isLiveStatus(status)`: active, queued or paused (the list GameHubView
     uses for "has a live task").
   - `taskMatchesGame(task, game)`: gameDir via `isSamePath`, else appId
     (ignore `''` and `'0'`, per AGENTS.md tenet 4 "absence is not an error").
2. Replace every hand-written status list and matcher listed above.
3. Add unit tests for both helpers in `stores/tasks.test.js`.

## Done when
`grep -rn "'downloading'" frontend/src --include=*.svelte` returns nothing
(the JSDoc comment in PlayActionCluster may stay), and `make check-fast` is
green.
