# Performance, Cache & Isolation Invariants

Essential subsystem architectural invariants and operational knowledge for `gamux`.

---

## 1. Downloader & Verification I/O

### The Dual-Hash Trap
Steam depot manifests frequently provide both a top-level file `SHA1` and a collection of `Chunks` (each ~1MB).
- **Anti-Pattern**: Evaluating `if entry.SHA1 != ""` in `needsDownload()` before chunk inspection forces a full-file read (e.g. 50GB) to compute SHA-1 upfront. If the hash mismatches (or during re-downloads), the chunk downloader reads the entire 50GB file *again* chunk-by-chunk to identify stale blocks, and a third pass computes SHA-1 post-download.
- **Invariant**: For entries with chunks (`len(entry.Chunks) > 0`), size-matching files must defer integrity checking to in-place chunk verification in `downloadChunked()`.

### Zero-Pending Chunk Bypass
- When `downloadChunked()` inspects an existing file on disk, `pendingChunks()` checks every chunk against its expected hash.
- **Invariant**: If all chunks match on disk (`len(pending) == 0`), zero network chunks were fetched or written. In this case, post-download SHA-1 re-hashing must be completely skipped—every block has already been validated against its chunk hash.

### Decoupled Integrity Scanning
- In `ScanDepotIntegrityWithProgress`, filesystem tree walking (`WalkDir`) must never execute hashing synchronously inside the walk callback.
- **Invariant**: The directory traversal quickly identifies missing files, size mismatches (corrupt without reading bytes), and collects size-matching official candidates. Content hashing is then dispatched across a bounded worker pool (`min(runtime.NumCPU(), 8)`) using `errgroup.Group`, updating progress under a mutex.

---

## 2. Warm Library Cache Invalidation

### The Root Dir Mtime Anti-Pattern
In Linux filesystems, a directory's `mtime` updates whenever any file is created, deleted, or renamed directly inside it.
- **Anti-Pattern**: Using the root game directory's modification timestamp (`dirFi.ModTime().UnixNano()`) as the cache key causes Wine, Proton, or Steam runtime logs (`wine.log`, `ShaderCache`, `.dxvk-cache`, `crash.dmp`) to invalidate the warm cache constantly. This forces redundant recursive traversals across 50,000+ files on subsequent inspections.
- Conversely, modifying executables or configs inside subdirectories (`bin/`, `steam_settings/`) does *not* bump root directory mtime, risking stale cache hits.

### Sentinel Signatures & Executable Stat Checking
- **Invariant**: Game cache identity is keyed on `detector.ComputeSentinelSignature(gameDir)`:
  - Configuration files: `.gamux/launch_profile.json`, `steam_settings/launch_profile.json`, `configs.*.ini`, `steam_appid.txt`
  - Manifest & mod files: `.gamux/manifest.json`, `.gamux/mods.json`, `steam_settings/mods.json`, `mods/`, `BepInEx/plugins/`, `[Manifests]`, and `appmanifest_*.acf` (in the game dir and parent `steamapps/`)
- In `cache.GetCachedGame`, cache hits further verify the primary executable binary (`cached.ExePath` mtime and size via a single `os.Stat`). If the binary was upgraded, unpacked with Steamless, or replaced, the cache entry is invalidated immediately.

---

## 3. Multiseat Evdev Input Isolation

### Tool Existence vs. Kernel Device Permissions
Checking binary presence (`exec.LookPath`) is necessary but not sufficient for multi-seat support.
- Hardware input isolation relies on kernel-level exclusive grabs via `EVIOCGRAB` on `/dev/input/event*` device nodes.
- Without read permissions on `/dev/input/event*` (typically membership in group `input` or active udev seat tags), `os.OpenFile` returns `EACCES` or `ioctl` fails with `EPERM`.

### Fail Fast vs. Silent Input Leakage
- **Anti-Pattern**: Swallowing grab failures and proceeding to launch Seat 2 leaves the hardware devices ungrabbed. The host OS and Player 1 continue receiving Player 2's keystrokes, and Seat 2 receives no input.
- **Invariant**: Capabilities probing (`multiseat.CheckCapabilities`) tests read accessibility to `/dev/input/event*` (`HasInputPermissions`). Seat launch (`prepareSeat`) must fail fast if hardware grabs fail, surfacing actionable remediation:
  ```bash
  sudo usermod -aG input $USER
  ```
