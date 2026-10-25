# 014 — Split oversized view components

**Area:** components · **Size:** M per component. Do one component per session.

## Context
These files are big enough that they're hard to change safely:

| File | Lines |
|---|---|
| `components/views/GameHubView.svelte` | ~770 (shrinks after 007–009) |
| `components/views/hub/cards/HubDLCCard.svelte` | ~690 |
| `components/views/SettingsView.svelte` | ~680 |
| `components/views/hub/HubBetasTab.svelte` | ~570 |
| `components/views/settings/ToolsMaintenanceTab.svelte` | ~550 |

## Do (per component)
1. Read the component and its test. Find the 2–4 visually or logically
   separate sections (e.g. a list plus its toolbar plus an empty state).
2. Extract each into a child component in the same folder, passing data down
   with `$props()` and callbacks as props (`onsomething`). No new global stores.
3. Move pure logic (filtering, formatting) into `utils/` with unit tests.
4. Keep the existing test passing. Add a small test for any extracted child
   that has logic.

## Done when
The parent is under ~350 lines, behaviour is unchanged, and
`make check-fast` is green. Do GameHubView last (after 007–009).
