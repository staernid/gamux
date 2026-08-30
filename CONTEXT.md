# Gamux Domain Context

Gamux is a post-download Steam game manager for Linux, orchestrating depot acquisition, DRM removal (Goldberg Emulator & Steamless), directory normalization, and standalone execution.

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

**Steamless**:
Automated SteamStub DRM executable unpacker that strips DRM wrappers and preserves original binaries.
_Avoid_: DRM cracker, patcher

**Standalone Launcher**:
The primary execution engine executing games directly on Linux using `umu-launcher` (`umu-run`), Wine, Proton, or native binaries, managing child process lifecycles and runtime environments.
_Avoid_: Direct play, custom runner

**UMU Runner**:
The Unified Linux Wine Game Launcher (`umu-run` from Open-Wine-Components) used for Proton/Wine compatibility, Wine prefix management, and environment setup.
_Avoid_: UMU wrapper, GE launcher

**Launch Profile**:
A structured configuration defining executable targets, launch arguments, Wine prefix location, runner selection, and performance/compatibility environment variables (`MANGOHUD`, `WINE_FULLSCREEN_FSR`, `PROTON_ENABLE_NVAPI`, `DXVK_HUD`).
_Avoid_: Run config, start script

**Pipeline**:
An ordered, sequential executor coordinating game processing through composable, isolated stages with cooperative cancellation, step-level telemetry, and automatic fail-fast LIFO failure rollback.
_Avoid_: Task runner, workflow, script

**Pipeline Recipe**:
A pre-assembled, ordered sequence of `Stage` instances configured for a discrete domain lifecycle action (Setup, Verify, Update, Rollback, Normalize).
_Avoid_: Workflow definition, job script, task sequence

**Stage**:
A discrete, single-responsibility step within a pipeline recipe (e.g. Scan, Manifest, Normalize, Steamless, GBE, Verify, Update, RevertDRM) that mutates or inspects a `Game` entity.
_Avoid_: Step, action, middleware

**Job**:
An asynchronous, cancellable background unit of work tracked by the supervisor worker pool (`tasks.Manager`) with lifecycle states (`Queued`, `Running`, `Paused`, `Completed`, `Failed`, `Cancelled`).
_Avoid_: Background thread, queue item, worker process

**Pipeline Operation**:
A direct, first-class binding of a `Pipeline Recipe` and execution `Context` satisfying `tasks.Operation`, bridging pipeline stage lifecycles, progress compositing, and detached context rollback to the supervisor worker pool without intermediary adapters.
_Avoid_: Task adapter, job wrapper, worker closure

**Event / Telemetry**:
A typed, non-blocking lifecycle message emitted across `events.Bus` carrying stage status and quantitative progress (`TaskProgress`: throughput, ETA, item counts, percentage).
_Avoid_: Notification, log stream, console output

**Directory Normalization**:
Renaming a downloaded game folder to match Steam's official 1:1 `installdir` from its ACF manifest or Steam API metadata.
_Avoid_: Folder renaming, sanitization, path cleanup

**Save Snapshot**:
A point-in-time, self-contained compressed archive (`.zip`) containing semantic mappings, file checksums, and on-disk save/configuration data for a game across Linux native, Wine prefix, and GSE Cloud locations.
_Avoid_: Save zip, backup file, save export

