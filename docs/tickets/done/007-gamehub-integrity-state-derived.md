# 007 — GameHubView: derive integrity-check state instead of syncing it in $effect

**Area:** `components/views/GameHubView.svelte` · **Size:** S · **Depends on:** 002, 006

## Context
GameHubView keeps `isCheckingIntegrity` / `integrityError` in sync with the
task list from inside an `$effect` (~lines 69–92). Writing state from an
effect is the Svelte 5 anti-pattern: it causes extra renders and ordering bugs.
`HubIntegrityCard.svelte` already does the same thing correctly:
`isVerifying = $derived(isCheckingIntegrity || (verifyTask && isActiveStatus(verifyTask.status)))`.

## Do
1. Add `let verifyTask = $derived(...)`: the latest verify task (`t.type ===
   'verify'`) matching the game via `taskMatchesGame` (ticket 006).
2. Derive the error: `verifyTask?.status === 'failed' ? (verifyTask.error ||
   'Integrity check failed.') : localError`.
3. Keep local `$state` only for things the user starts (e.g. "I clicked
   verify, waiting for the task to appear").
4. Delete the effect. Check what HubIntegrityCard receives as props; avoid
   computing the same thing twice in both components.

## Done when
GameHubView has no `$effect` touching integrity state, existing
`GameHubView.test.js` integrity tests pass (adjust them only if they tested
implementation details), and `make check-fast` is green.
