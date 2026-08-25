# Gamux Domain Context

Gamux is a post-download Steam game manager for Linux, orchestrating depot acquisition, DRM removal (Goldberg Emulator & Steamless), directory normalization, and Lutris registration.

## Language

**Game**:
An installed or downloaded game title containing executable binaries, assets, and optional Steamworks libraries.
_Avoid_: App, product, package

**Depot**:
A Steam content distribution unit containing file entries, chunks, and metadata identified by a unique DepotID.
_Avoid_: Package, archive, repo

**Manifest**:
An official Steam metadata record (binary `.manifest` or KeyValues `appmanifest_<appid>.acf`) mapping file paths, sizes, SHA1 checksums, and encrypted chunk IDs.
_Avoid_: Index, catalog

**Loader Mode**:
Goldberg Steam Emulator execution configuration keeping official game binaries 100% untouched, hooking Steamworks interfaces via Wine registry overrides or loader helper executables.
_Avoid_: Non-portable mode, injection mode

**Portable Mode**:
Goldberg Steam Emulator execution mode that backs up original `steam_api.dll`/`libsteam_api.so` to `.ORIGINAL` and replaces them in-place in the game folder.
_Avoid_: Direct mode, patch mode

**Lutris Registration**:
Generating Lutris game configuration YAML files, updating Lutris's SQLite database (`pga.db`), and creating desktop/application menu shortcuts.
_Avoid_: Desktop shortcutting, runner configuration

**Steamless**:
Automated SteamStub DRM executable unpacker that strips DRM wrappers and preserves original binaries.
_Avoid_: DRM cracker, patcher

**Operating Mode**:
The top-level operational strategy for a game: `Lutris Helper` (companion mode generating Lutris configs and delegating launch to Lutris) or `Standalone Launcher` (direct game execution bypassing Lutris).
_Avoid_: Execution style, play mode

**Standalone Launcher**:
An operating mode that executes games directly on Linux using `umu-launcher` (`umu-run`), Wine, Proton, or native binaries, managing child process lifecycles and runtime environments.
_Avoid_: Direct play, custom runner

**Lutris Helper**:
An operating mode acting as a companion to Lutris: generating Lutris YAML configs, updating the Lutris SQLite database, and delegating execution to `lutris lutris:rungame/<slug>`.
_Avoid_: Lutris bridge, Lutris runner mode

**UMU Runner**:
The Unified Linux Wine Game Launcher (`umu-run` from Open-Wine-Components) used in Standalone Launcher mode for Proton/Wine compatibility, Wine prefix management, and environment setup.
_Avoid_: UMU wrapper, GE launcher

**Launch Profile**:
A structured configuration defining executable targets, launch arguments, Wine prefix location, runner selection, and performance/compatibility environment variables (`MANGOHUD`, `WINE_FULLSCREEN_FSR`, `PROTON_ENABLE_NVAPI`, `DXVK_HUD`).
_Avoid_: Run config, start script

**Pipeline**:
An ordered, sequential executor coordinating game processing through composable, isolated steps with cooperative cancellation and automatic failure rollback.
_Avoid_: Task runner, workflow, script

**Stage**:
A discrete, single-responsibility step within the setup pipeline (e.g. Scan, Manifest, Normalize, Steamless, GBE, Lutris) that mutates or inspects a `Game` entity.
_Avoid_: Step, action, middleware

**Job**:
An asynchronous, cancellable background unit of work tracked by the supervisor worker pool (`jobs.Manager`) with lifecycle states (`Pending`, `Running`, `Completed`, `Failed`, `Cancelled`).
_Avoid_: Background thread, queue item, worker process

**Event / Telemetry**:
A typed, non-blocking lifecycle message emitted across `events.Bus` carrying stage status and quantitative progress (`TaskProgress`: throughput, ETA, item counts, percentage).
_Avoid_: Notification, log stream, console output

**Directory Normalization**:
Renaming a downloaded game folder to match Steam's official 1:1 `installdir` from its ACF manifest or Steam API metadata.
_Avoid_: Folder renaming, sanitization, path cleanup

