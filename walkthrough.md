# Walkthrough - Tray Context Menu & Quit Handling

## Changes

### 1. Fix: System Tray "Quit" Intercepted by CloseToTray
- **Root Cause**: When the user had `CloseToTray` enabled in their settings, selecting "Quit" from the tray context menu called `wailsRuntime.Quit(...)`. Wails routes application quit events through `OnBeforeClose(ctx)`. Because `OnBeforeClose` only checked `app.Config.CloseToTray` without checking whether a termination was explicitly requested, it called `wailsRuntime.WindowHide(ctx)` and returned `true` (which cancels application shutdown in Wails).
- **Fix**:
  - In [`gui/app.go`](file:///home/staernid/projects/gamux/gui/app.go): Added `isQuitting atomic.Bool`, `SetQuitting(bool)`, and `IsQuitting() bool`. Updated `App.Quit()` to mark `isQuitting = true` before invoking `wailsRuntime.Quit`.
  - In [`cmd/gamux-gui/main.go`](file:///home/staernid/projects/gamux/cmd/gamux-gui/main.go):
    - `onQuit` explicitly calls `app.SetQuitting(true)` before requesting `wailsRuntime.Quit`.
    - Added a fallback safety timer to cleanly release single-instance guards and terminate the process if Wails/WebKitGTK gets stuck during teardown.
    - Updated `OnBeforeClose` to check `if app != nil && app.IsQuitting() { return false }` so intentional shutdown requests are never swallowed by `CloseToTray`.
  - In [`gui/tray_linux.go`](file:///home/staernid/projects/gamux/gui/tray_linux.go): Dispatched `m.tray.onQuit()` in a goroutine so the DBus `Event` call replies immediately to the session bus (e.g. Plasma workspace) before teardown starts.

### 2. Top Three Played Games in Context Menu
- Created [`gui/tray_types.go`](file:///home/staernid/projects/gamux/gui/tray_types.go):
  - `TrayGame`: Stores `Name`, `GameDir`, and `LastPlayed`.
  - `TrayCallbacks`: Encapsulates `OnActivate`, `OnQuit`, `OnLaunchGame`, and `GetRecentGames`.
  - `SortTrayGames(games []TrayGame, limit int) []TrayGame`: Omits unplayed games (`LastPlayed <= 0`), sorts descending by `LastPlayed` (least elapsed time first), breaks ties alphabetically by title, and limits results to at most 3 items.
  - `FilterAndSortRecentGames(games []*domain.Game, limit int) []TrayGame`: Converts domain game statuses into `TrayGame` records using `EffectivePlaytime()` and resolved titles.
- In [`gui/tray_linux.go`](file:///home/staernid/projects/gamux/gui/tray_linux.go):
  - Top 3 games appear at the top of the menu with mnemonic underscore escaping (`_` $\to$ `__`), followed by a separator, then **Show Gamux**, separator, and **Quit**.
  - Clicking a game launches it asynchronously via `QuickPlay`.
  - DBusMenu `AboutToShow(0)` dynamically refreshes recent games.
- In [`gui/app.go`](file:///home/staernid/projects/gamux/gui/app.go):
  - In-memory cache syncs when games are inspected or exit (`EventGameExited`).
  - Background scan warms cache on startup.

## Verification Results

### Automated Tests
1. `go test -race -v ./gui -run "TestTray.*|TestAppCached.*|TestAppQuit.*"`:
   - `TestTraySortingAndLimiting`: **PASS**
   - `TestTrayMenuItemsAndEvents`: **PASS**
   - `TestTrayRecentGamesMenuAndLaunch`: **PASS**
   - `TestAppCachedGamesAndRecentGames`: **PASS**
   - `TestAppQuitAndIsQuitting`: **PASS**
2. `go test -race ./gui/...`: **PASS** (12.29s, clean concurrency under race detector).
3. `make test`: **PASS** (All Go tests with race detection and all 217 frontend Vitest tests + invariant audit).
4. `make build-gui`: **PASS** (successfully compiled `bin/gamux-gui`).
