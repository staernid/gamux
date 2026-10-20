# Workshop & Mod Dependency Management Design & Implementation

> Status: Implemented (Automated engine mod topology detection, non-destructive toggling, Steam Workshop metadata tracking, update alerts, CLI subcommands, and dedicated GUI Mods Tab).

## Overview
Gamux provides automated, integrated mod and Steam Workshop management across Linux standalone and Proton titles.

## Implemented Architecture

1. **Mod Directory Placement & Engine Topologies (`detector/mods.go`)**:
   - Auto-detects game engines and standard mod directory layouts:
     - **Unity Engine**: `BepInEx/plugins/` (supported formats: `.dll`, `.zip`).
     - **Unreal Engine**: `<Project>/Content/Paks/~mods` (supported formats: `.pak`, `.ucas`, `.utoc`, `.zip`).
     - **Generic / Custom**: `mods/` in game root directory.
   - Preserves user files and standard paths across filesystem types.

2. **Non-Destructive Mod Toggling**:
   - Mods are toggled active/inactive via atomic toggle rename with `.disabled` suffix (`filename` <-> `filename.disabled`).
   - Works across all filesystems without deleting files, requiring no symlink capabilities or root permissions, and safely respected by Bubblewrap sandbox bind-mounts.

3. **Workshop Update Tracking & Metadata**:
   - Installed Workshop item metadata is persisted in `<gameDir>/.gamux/mods.json`.
   - Update verification queries `ISteamRemoteStorage/GetPublishedFileDetails/v1` to compare local `time_updated` against remote timestamps.
   - Outdated mods trigger update notification alerts in both CLI and GUI.

4. **CLI Commands**:
   - `gamux workshop <id_or_url>`: Downloads and installs Workshop mod directly into the detected engine topology directory.
   - `gamux mods list [--dir <dir>]`: Lists installed mods, engine topology, and update availability.
   - `gamux mods toggle <id> [--enable/--disable]`: Toggles mods without deleting files.
   - `gamux mods check-updates [--dir <dir>]`: Queries Steam Workshop for updates.

5. **Dedicated GUI Mods Tab (`frontend/src/components/views/hub/HubModsTab.svelte`)**:
   - Surface in Game Hub navigation (`['overview', 'mods', 'config', 'files', 'saves']`).
   - Displays topology badges, target folder quick-opener, workshop install input bar, update alert banners, and non-destructive toggle switches (`SeamlessSwitch.svelte`).

