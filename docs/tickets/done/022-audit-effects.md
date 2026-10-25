# 022 — Audit remaining $effect usages

**Area:** components / stores · **Size:** M · **Do after:** 007–009

## Context
There are 41 `$effect(` calls in `frontend/src`. In Svelte 5, `$effect` is for
syncing with the outside world (DOM APIs, timers, subscriptions). Effects that
only compute state from other state should be `$derived`. Effects that react
to user actions should be event handlers.

Files with the most: `GameHubView` (4), `HubOverviewTab` (4), `CatalogModal`
(3), `HubCommunityTab` (3), `CoopTab` (3), `SettingsView` (3).

## Do
For each `$effect`:
1. If it only assigns state derived from other state → convert to
   `$derived` / `$derived.by`.
2. If it fires because a prop or state changed which was itself set by a click
   → call the logic from the click handler instead.
3. If it does real side-effect work (fetch on id change, timers, listeners)
   → keep it. Make sure it returns a cleanup function where relevant and
   guards against stale async results (check that the id still matches before
   assigning).
Write a one-line comment on any effect you keep that isn't obviously necessary.

## Done when
The effect count drops meaningfully, there are no "assign state in effect"
patterns left that could be derived, and `make check-fast` is green.
