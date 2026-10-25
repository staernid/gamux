# 002 — Remove snake_case reads from DownloadsView and GameHubView

**Area:** components · **Size:** S · **Depends on:** 001 (recommended)

## Context
Components must only see normalized camelCase entities (AGENTS.md → Frontend
Domain & State).

- `components/views/DownloadsView.svelte` around line 80:
  `String(g.appId || g.app_id)`. Library games are already normalized by
  `normalizeGame` in `utils/game.js`, so `g.appId` is enough.
- `components/views/GameHubView.svelte` around lines 69–92: an `$effect` reads
  `vTask.metadata?.integrity_report` and `t.operation`. **The backend never sets
  `integrity_report` in task metadata** (grep the Go code: nothing writes it),
  and `operation` is not a task field. This branch is dead code.

## Do
1. DownloadsView: drop `|| g.app_id`.
2. GameHubView: delete the dead `integrity_report` parsing and `t.operation`
   check. Keep the behaviour that clears `isCheckingIntegrity` and sets
   `integrityError` when the verify task completes or fails (ticket 007 turns
   this into a `$derived`; for now, only remove the dead parts).
3. Update the affected tests (`GameHubView.test.js`).

## Done when
`grep -rnE "\.(app_id|integrity_report|operation)\b" frontend/src --include=*.svelte`
has no hits in these two files, and `make check-fast` is green.
