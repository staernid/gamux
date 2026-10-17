
# Gamux Modernization & Architecture Roadmap

## ✅ 2. Frontend: Legacy Svelte Syntax Elimination *(Completed)*
- **Locations**:
  - `frontend/src/components/views/hub/cards/HubAboutCard.svelte`
  - `frontend/src/components/views/hub/HubConfigTab.svelte`
- **Accomplished**:
  - Refactored remaining legacy reactive declarations using `run` from `'svelte/legacy'` into native Svelte 5 `$effect` runes.
  - Removed unused and redundant lifecycle hooks (`onMount`), ensuring reactive metadata loading and launch command preview updates adhere to Svelte 5 rune idioms.
  - Verified with full Vitest test suite (`HubAboutCard.test.js`, `GameHubView.test.js`) and `svelte-check`.

---

## ✅ 3. Backend: Standalone / Unlinked Game Prefix Goldberg Artifact Isolation *(Completed)*
- **Locations**:
  - `compat/prefix.go`
  - `compat/profile.go`
  - `pipeline/stages_impl.go`
  - `runner/command.go`
- **Accomplished**:
  - Added `ForceEmulation *bool` to `domain.LaunchProfile` to allow explicit opt-in/opt-out per profile or game.
  - Implemented `compat.ShouldEmulate(game)` requiring a valid Steam App ID unless explicitly forced.
  - Implemented `compat.ScrubPrefixEmulation(prefixDir)` to cleanly wipe Goldberg emulator libraries (`steamclient.dll`, `steamclient64.dll`, etc.) and `steam_settings` links from Wine prefixes when emulation is inactive or removed.
  - Guarded `GBEStage` execution, seat prefix preparation, metadata unlinking, and `WINEDLLOVERRIDES` generation to prevent Steam hook collisions on standalone/unlinked titles.

---

## ✅ 4. Manifest vs. Heuristic Executable Detection Priority *(Completed)*
- **Locations**:
  - `detector/exe_priority.go`
  - `detector/detector.go`
  - `detector/exe_priority_test.go`
- **Accomplished**:
  - Manifest-declared executables now strictly outrank heuristic detection, `.ini` configuration files, and generic directory scans.
  - Demoted known crack/loader wrappers (e.g. `ColdClientLoader.ini`, launcher stubs) in `ScoreExecutable` to prevent auxiliary launchers from overriding primary game binaries.
  - Preserved user-selected entry points across deep library rescans.

---

## ✅ 5. Storage: Outdated Artwork Garbage Collection *(Completed)*
- **Locations**:
  - `store/artwork.go`
  - `engine/task.go`
  - `store/store_test.go`
- **Accomplished**:
  - Implemented `store.PruneOrphans(game)` to sweep `.gamux/art/` and prune orphaned, unreferenced image files.
  - Integrated orphan pruning into `ApplyCustomArtwork` on replace or reset of covers, banners, icons, and logos.
  - Wired orphan garbage collection into batch library artwork maintenance tasks.
