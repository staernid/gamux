# Future Explorations & Technical Spikes (`TO_EXPLORE.md`)

This document tracks complex, high-friction, or potentially iffy areas that require deeper research, user experience evaluation, and dedicated architectural spikes before implementation.

---

## 1. Game Dependencies & Redistributables Management

### Context & Problem Statement
Windows games frequently depend on external runtime libraries:
- Microsoft Visual C++ Redistributables (2005, 2008, 2010, 2012, 2013, 2015–2022 x86/x64).
- DirectX 9.0c / 11 user runtimes (`d3dx9_43.dll`, `xinput1_3.dll`, `d3dcompiler_47.dll`).
- OpenAL, PhysX, .NET Framework / dotNetFx.

When running under Wine/Proton, missing runtimes lead to silent crashes, missing audio, or crash dialogs on launch. Many Steam depot dumps include a `_CommonRedist/` or `redist/` directory with official installers.

### Why This is Iffy & High Risk
1. **Silent Installer Incompatibilities**: Running `.exe` installers (like `vcredist_x64.exe` or `dxsetup.exe`) non-interactively (`/quiet /norestart`) inside Wine/Proton prefixes sometimes hangs indefinitely or fails with obscure MSI error codes if the Wine prefix is being written to simultaneously.
2. **Proton vs. Wine Architecture Mismatches**: Proton's internal Wine prefix layout (`pfx`) handles 64-bit and 32-bit subsystems differently. Installing a 32-bit redistributable into a 64-bit Wine prefix can corrupt Wine's 64-bit DLL overrides if done incorrectly.
3. **Overlapping with UMU & Protonfixes**: UMU and Proton already have `umu-protonfixes` / `protonfixes` that automatically apply winetricks/verbs based on Steam `AppID`. Implementing our own redistributable installer might conflict with or duplicate what UMU already does behind the scenes.
4. **Winetricks / Protontricks Overhead**: Invoking `protontricks` or `winetricks` as a sub-process introduces external system dependencies that may not be available on all Linux distributions or Flatpak/Steam Deck environments.

### Questions & Areas to Explore
- [ ] Should Gamux strictly rely on UMU's built-in Protonfixes for known Steam AppIDs instead of executing `.exe` installers directly?
- [ ] Is simple CAB extraction or direct DLL placement (`winedllpath` or game directory placement) safer than running full MSI installers?
- [ ] Should we provide a manual "Run Executable in Game Prefix" developer tool instead of automated redistributable installation?

---

## 2. Library UX & Ergonomics at Scale

### Context & Problem Statement
The current primary library presentation is a 3D rotating carousel (`Carousel3D.svelte`) optimized for 5–15 games and console/handheld gamepad navigation. When users accumulate 50–200+ games, linear 1-by-1 carousel traversal becomes slow on desktop with mouse and keyboard.

### Why This is Iffy & High Risk
1. **Identity & Polish Trade-off**: The custom 3D carousel, physical reflections, sound design (`audio.js`), and floating HUD give Gamux its distinct arcade/handheld console personality. Adding standard grid/table views risks diluting the focused aesthetic into a generic launcher clone (like Heroic or Lutris).
2. **Gamepad Navigation Complexity**: Maintaining 100% full controller/gamepad navigation across both a 3D carousel and a 2D responsive poster grid introduces dual-focus management complexity, spatial navigation bugs, and tricky scroll boundaries.
3. **Auto-Save vs. Explicit Save Friction**:
   - In `GameHubView.svelte`, settings currently require an explicit "Save Changes" button.
   - While auto-saving on blur/toggle reduces clicks, it can accidentally persist unintended changes (e.g. accidentally changing a runner or operating mode while browsing tabs).
   - A dirty-state indicator with an explicit prompt on navigate-away might be safer than full silent auto-save.

### Questions & Areas to Explore
- [ ] Can we enhance the 3D Carousel with high-speed rotary scrolling, alphabetical jump indexes (like Steam Deck or iPod cover flow), and category filters without replacing the primary view?
- [ ] Should a dense Grid View remain as an on-demand modal overlay (`SearchGridModal.svelte`) enhanced with rich filter pills, rather than replacing the shelf view?
- [ ] Prototype a dirty-state draft indicator in `GameHubView` with seamless undo/revert instead of destructive silent auto-saving.
