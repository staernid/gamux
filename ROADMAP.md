
## 2. Frontend: Legacy Svelte Syntax (`run(...)` from `'svelte/legacy'`)
- **Locations**:
  - `frontend/src/lib/components/hub/HubAboutCard.svelte`
  - `frontend/src/lib/components/hub/HubConfigTab.svelte`
- **Context & Issue**:
  Gamux is migrating to Svelte 5 runes (`$state`, `$derived`, `$effect`, `$props`). However, several Hub subcomponents still import `run` from `'svelte/legacy'` to simulate reactive declarations (`$: ...`).
- **Proposed Direction**:
  Refactor these remaining legacy reactive declarations into native Svelte 5 `$effect` or `$derived` runes, eliminating all legacy shims from `frontend/src/lib/components/hub/`.
---
## 3. Backend: Standalone / Unlinked Game Prefix Goldberg Artifact Isolation
- **Locations**:
  - `pipeline/gbe.go`
  - `runner/wine.go`
  - `compat/profile.go`
- **Context & Issue**:
  When titles are run in loader mode or when Goldberg emulation is bootstrapped into a Wine prefix (`drive_c/Program Files (x86)/Steam/`), standalone or non-Steam games without a valid Steam App ID can receive empty or placeholder configuration files. Furthermore, if a game is later unlinked or shifted to native/different runner modes, remnants in the prefix can cause conflicting Steam API hook behavior.
- **Proposed Direction**:
  Ensure clean lifecycle management of emulation artifacts in the prefix:
  - If a title lacks a Steam App ID or is marked unlinked, bypass Goldberg loader injection unless explicitly toggled.
  - Provide an idempotent prefix scrubber or ensure symlinks/DLL overrides are only mounted when emulation is active.
---
## 4. Manifest vs. Heuristic Executable Detection Priority
- **Locations**:
  - `pipeline/normalize.go`
  - `manifest/loader.go`
  - `engine/library.go`
- **Context & Issue**:
  In certain scanned directories with pirated or third-party wrappers, files like `ColdClientLoader.ini` or auxiliary crack launchers might be picked up by heuristic detection even when an authoritative upstream manifest (e.g. from a known distribution structure) specifies the canonical game executable.
- **Proposed Direction**:
  Strengthen executable scoring in `pipeline/normalize.go`:
  - Prioritize manifest-declared executables and canonical root executables over loader/wrapper binaries.
  - Allow the user to pick an alternate entry point without having the heuristic re-override it during deep library rescans.
---
## 5. Storage: Outdated Artwork Garbage Collection (`.gamux/art/`)
- **Locations**:
  - `store/artwork.go`
  - `engine/library.go`
- **Context & Issue**:
  As users search, link, and swap custom artwork (covers, banners, icons, logos) across various providers (SteamGridDB, Steam CDN, GOG, RAWG), downloaded image assets are cached locally in the game's `.gamux/art/` or system cache directories under hashed filenames. When custom artwork is replaced or reset, the previous asset files are orphaned on disk.
- **Proposed Direction**:
  Implement an artwork garbage-collection helper:
  - Prune unreferenced images in `.gamux/art/` during an explicit library maintenance routine or when replacing a slot's image.
