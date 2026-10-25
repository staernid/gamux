# 015 — Show user-facing errors instead of console-only warnings

**Area:** components · **Size:** S

## Context
These user-triggered loads fail silently. The user sees an empty or stale UI
and the only trace is a `console.warn`:

| File:line (approx.) | What fails |
|---|---|
| `components/stage/InstallModal.svelte:165` | Loading depots for the install dialog |
| `components/stage/InstallModal.svelte:284` | Creating library stub before download |
| `components/views/GameHubView.svelte:311` | Same stub creation (duplicated logic!) |
| `components/stage/CatalogModal.svelte:55` | Store search |
| `components/views/LibraryView.svelte:111` | Store search (duplicated logic?) |
| `components/views/SettingsView.svelte:199` | Loading initial settings |
| `components/views/hub/HubModsTab.svelte:84` | Loading mods |
| `components/domain/MetadataLinker.svelte:88` | (check) |
| `components/common/ModalHost.svelte:17` | (check: may be fine to keep) |

## Do
For each one, decide:
- **Blocking content failed to load** (depots, settings, mods): show an
  inline error state in that panel with a Retry button (text-xs, monochrome;
  only the error text uses semantic red, per AGENTS.md UI Design).
- **Action failed** (stub creation, search): `notificationStore.error(title, err)`.
- **Background enrichment**: keep `console.warn`.

Also dedupe: the "create library stub before download" code in InstallModal
and GameHubView, and the store search in CatalogModal and LibraryView, should
each become one function in `stores/library/actions.js`.

## Done when
Every row above is resolved (fixed, or justified in the commit message), the
duplicates are merged, and `make check-fast` is green.
