# 001 — Normalize Task entities to a single wire shape

**Area:** `frontend/src/stores/tasks.svelte.js` · **Size:** S

## Context
The Go wire type is `tasks.Task` in `tasks/types.go`. Its JSON fields are
`id`, `type`, `title`, `game_name`, `game_dir`, `app_id`, `status`, `stage`,
`details`, `capabilities`, `progress`, `error`, `created_at`, `started_at`,
`finished_at`, `metadata`, `game`. That list is the complete contract.

`normalizeTask()` (around line 102) guesses at many aliases that the backend
never sends:
```js
const appId = String(t.app_id ?? t.appId ?? t.game_id ?? '');
const gameDir = t.game_dir || t.gameDir || t.target_dir || t.targetDir || '';
id: t.id || t.task_id,
const gameName = t.game_name || t.gameName || t.title || 'Task';
```
The *Single Path Forward* tenet in AGENTS.md rules out guessing like this.

## Do
1. Rewrite `normalizeTask` so it reads only the snake_case fields listed above
   and outputs camelCase. Keep the derived fields (percent, speed, eta, ...).
2. Check whether anything calls `normalizeTask` with an already-normalized
   object (grep for `normalizeTask(`). If so, fix that caller so it doesn't
   normalize twice. Don't add camelCase fallbacks.
3. Fix test fixtures in `stores/tasks.test.js` (and any other tests) that
   relied on the removed aliases. Update `MOCK_TASKS` in `api/mockData.js` to
   the Go shape.
4. Check `extractTaskProgress` and `isDownloadTask` in the same file for the
   same kind of alias guessing (`t.operation` is not a Go field).

## Done when
- No `task_id`, `target_dir`, `game_id`, `t.operation` or camelCase-input
  fallbacks remain in `tasks.svelte.js`.
- `make check-fast` is green.
