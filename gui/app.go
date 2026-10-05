package gui

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/util"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// App is the Go backend adapter exposed to the Wails frontend.
type App struct {
	ctx           context.Context
	mainWindow    *application.WebviewWindow
	Config        *config.Config
	Engine        *engine.Engine
	targetMu      sync.Mutex
	pendingTarget string
	trayMgr       *TrayManager
	gamesMu       sync.RWMutex
	cachedGames   []*domain.Game
	isQuitting    atomic.Bool
}

// NewApp creates a new App instance with configuration and services initialized.
func NewApp(cfg *config.Config) *App {
	if cfg == nil {
		var err error
		cfg, err = config.LoadConfig("")
		if err != nil {
			slog.Warn("Failed to load user config, using defaults", "error", err)
			cfg = config.DefaultConfig()
		}
	}
	eng := engine.New(cfg)
	return &App{
		Config: cfg,
		Engine: eng,
	}
}

// HandleOpenTarget handles an incoming open target request (URL or AppID) from IPC or CLI.
func (a *App) HandleOpenTarget(target string) {
	a.targetMu.Lock()
	defer a.targetMu.Unlock()
	if a.ctx == nil {
		a.pendingTarget = target
		return
	}
	safeEmit(a.ctx, "gamux:open-target", target)
}

// SetMainWindow registers the primary WebviewWindow reference.
func (a *App) SetMainWindow(w *application.WebviewWindow) {
	a.mainWindow = w
}

// GetMainWindow returns the primary WebviewWindow reference.
func (a *App) GetMainWindow() *application.WebviewWindow {
	return a.mainWindow
}

// GetInitialTarget returns and clears any initial target passed on cold start.
func (a *App) GetInitialTarget() string {
	a.targetMu.Lock()
	defer a.targetMu.Unlock()
	t := a.pendingTarget
	a.pendingTarget = ""
	return t
}

func safeEmit(ctx context.Context, eventName string, data ...any) {
	wailsApp := application.Get()
	if wailsApp == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("Ignoring panic during safeEmit", "event", eventName, "recover", r)
		}
	}()
	wailsApp.Event.Emit(eventName, data...)
}

func parseMetaInt(md map[string]string, key string) int {
	if md == nil {
		return 0
	}
	if val, ok := md[key]; ok {
		n, _ := strconv.Atoi(val)
		return n
	}
	return 0
}

func parseMetaInt64(md map[string]string, key string) int64 {
	if md == nil {
		return 0
	}
	if val, ok := md[key]; ok {
		n, _ := strconv.ParseInt(val, 10, 64)
		return n
	}
	return 0
}

func requireGamePath(gamePath string) (string, error) {
	c := util.CanonicalPath(gamePath)
	if c == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return c, nil
}

// Startup is called when the Wails application starts up.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.monitorSystemThemeChanges(ctx)

	// Bind central EventBus to Wails frontend event emitter
	if a.Engine != nil && a.Engine.EventBus != nil {
		var (
			progressMu   sync.Mutex
			lastProgress = make(map[string]int64)
		)

		a.Engine.EventBus.SubscribeFunc(func(ev events.Event) {
			isTaskOrTransfer := strings.HasPrefix(string(ev.Type), "task:") || strings.HasPrefix(string(ev.Type), "transfer:")
			if isTaskOrTransfer {
				// Throttle high-frequency progress events crossing the Wails IPC boundary
				if ev.Type == events.EventTaskProgress || ev.Type == events.EventTransferProgress {
					now := time.Now().UnixMilli()
					progressMu.Lock()
					last := lastProgress[ev.TaskID]
					isTerminalProgress := ev.Progress.Percent >= 100 || (ev.Progress.TotalBytes > 0 && ev.Progress.CurrentBytes >= ev.Progress.TotalBytes)
					if !isTerminalProgress && (now-last) < 150 {
						progressMu.Unlock()
						return
					}
					lastProgress[ev.TaskID] = now
					progressMu.Unlock()
				} else if ev.Type == events.EventTaskCompleted || ev.Type == events.EventTaskFailed || ev.Type == events.EventTaskCancelled ||
					ev.Type == events.EventTransferCompleted || ev.Type == events.EventTransferFailed || ev.Type == events.EventTransferCancelled {
					progressMu.Lock()
					delete(lastProgress, ev.TaskID)
					progressMu.Unlock()
				}

				safeEmit(a.ctx, "gamux:task-event", ev)
			}

			if ev.Type == events.EventNotification {
				title := ev.Stage
				if t, ok := ev.Metadata["title"]; ok && t != "" {
					title = t
				}
				msg := ev.Details
				if m, ok := ev.Metadata["message"]; ok && m != "" {
					msg = m
				}
				safeEmit(a.ctx, "gamux:notification", map[string]string{
					"title":   title,
					"message": msg,
				})
			}
			if ev.Type == events.EventGameLaunched {
				safeEmit(a.ctx, "gamux:game-status", map[string]any{
					"gameDir":    ev.GameDir,
					"isRunning":  true,
					"pid":        parseMetaInt(ev.Metadata, "pid"),
					"profileId":  ev.ProfileID,
					"seatId":     ev.SeatID,
					"instanceId": ev.InstanceID,
					"gameName":   ev.GameName,
					"appId":      ev.AppID,
				})
				a.handleGameLaunchWindow(true)
			}
			if ev.Type == events.EventGameExited {
				ps := parseMetaInt64(ev.Metadata, "playtime_seconds")
				lp := parseMetaInt64(ev.Metadata, "last_played")
				safeEmit(a.ctx, "gamux:game-status", map[string]any{
					"gameDir":         ev.GameDir,
					"isRunning":       false,
					"pid":             0,
					"profileId":       ev.ProfileID,
					"seatId":          ev.SeatID,
					"instanceId":      ev.InstanceID,
					"gameName":        ev.GameName,
					"appId":           ev.AppID,
					"sessionSeconds":  parseMetaInt64(ev.Metadata, "session_seconds"),
					"playtimeSeconds": ps,
					"lastPlayed":      lp,
				})
				a.handleGameLaunchWindow(false)
				if ev.GameDir != "" {
					if lp <= 0 {
						lp = time.Now().Unix()
					}
					a.RecordGamePlayed(ev.GameDir, lp, ps)
				}
			}
			if ev.Type == events.EventTaskProgress && ev.Metadata != nil && ev.Metadata["step"] != "" {
				safeEmit(a.ctx, "gamux:step-progress", map[string]any{
					"step":    parseMetaInt(ev.Metadata, "step"),
					"total":   parseMetaInt(ev.Metadata, "total"),
					"title":   ev.Stage,
					"details": ev.Details,
					"percent": ev.Progress.Percent,
				})
			}
		})
	}

	// Pre-warm game cache for tray menu and fast lookups
	go func() {
		dirs := a.Config.ResolvedLibraryDirs()
		if len(dirs) > 0 && a.Engine != nil {
			if games, err := a.Engine.BatchInspectLibraries(context.Background(), dirs); err == nil && len(games) > 0 {
				a.UpdateCachedGames(games)
			}
		}
	}()

	slog.Info("Gamux GUI backend initialized")
}

func (a *App) handleGameLaunchWindow(launched bool) {
	if a.ctx == nil || a.Config == nil {
		return
	}
	behavior := strings.ToLower(strings.TrimSpace(a.Config.LaunchWindowBehavior))
	if behavior == "" || behavior == "keep" {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("Ignoring panic during window state change", "behavior", behavior, "launched", launched, "recover", r)
		}
	}()

	if launched {
		switch behavior {
		case "minimize":
			if a.mainWindow != nil {
				a.mainWindow.Minimise()
			}
		case "hide":
			if a.mainWindow != nil {
				a.mainWindow.Hide()
			}
		}
	} else {
		switch behavior {
		case "minimize":
			if a.mainWindow != nil {
				a.mainWindow.Restore()
			}
		case "hide":
			if a.mainWindow != nil {
				a.mainWindow.Show()
				a.mainWindow.Restore()
			}
		}
	}
}

// IsQuitting returns whether an application shutdown was explicitly requested.
func (a *App) IsQuitting() bool {
	return a.isQuitting.Load()
}

// SetQuitting marks the application as explicitly shutting down.
func (a *App) SetQuitting(quitting bool) {
	a.isQuitting.Store(quitting)
}

// Quit unconditionally terminates the GUI application.
func (a *App) Quit() {
	a.SetQuitting(true)
	wailsApp := application.Get()
	if wailsApp != nil {
		wailsApp.Quit()
	}
}

// Shutdown cleanly terminates all active multi-seat background sessions and resources.
func (a *App) Shutdown(ctx context.Context) {
	if a.Engine != nil && a.Engine.Multiseat != nil {
		a.Engine.Multiseat.StopAll()
	}
}

// getContext returns the current context (useful for fallback and testing).
func (a *App) getContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// AssetHandler returns an http.Handler that serves cached artwork, local game images, and on-the-fly thumbnails.
func (a *App) AssetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		artDir := ""
		if a.Config != nil && a.Config.ArtworkDir != "" {
			artDir = util.ExpandPath(a.Config.ArtworkDir)
		} else {
			home, _ := os.UserHomeDir()
			artDir = filepath.Join(home, ".local", "share", "gamux", "art")
		}

		if r.URL.Path == "/art/thumb" || r.URL.Path == "/art/thumb/" {
			a.handleThumbnail(w, r, artDir)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/art/") {
			rel := strings.TrimPrefix(r.URL.Path, "/art/")
			rel = filepath.Clean(rel)
			if strings.HasPrefix(rel, "..") {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			filePath := filepath.Join(artDir, rel)
			if util.FileExists(filePath) {
				// The ?t=<mtime> query string handles cache busting, so we can
				// cache aggressively and avoid redundant revalidation requests.
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				http.ServeFile(w, r, filePath)
				return
			}
		} else if strings.HasPrefix(r.URL.Path, "/local-img/") {
			rel := strings.TrimPrefix(r.URL.Path, "/local-img/")
			filePath := "/" + rel
			filePath = filepath.Clean(filePath)
			if strings.Contains(filePath, "..") {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			if util.FileExists(filePath) {
				w.Header().Set("Cache-Control", "no-cache, must-revalidate")
				http.ServeFile(w, r, filePath)
				return
			}
		}
		http.NotFound(w, r)
	})
}

func (a *App) handleThumbnail(w http.ResponseWriter, r *http.Request, artDir string) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		id = strings.TrimSpace(r.URL.Query().Get("path"))
	}
	if id == "" {
		id = strings.TrimSpace(r.URL.Query().Get("src"))
	}
	if id == "" {
		http.Error(w, "missing id parameter", http.StatusBadRequest)
		return
	}

	if strings.Contains(id, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	targetWidth := 300
	if wStr := r.URL.Query().Get("w"); wStr != "" {
		if parsed, err := strconv.Atoi(wStr); err == nil && parsed > 0 {
			targetWidth = parsed
		}
	}

	sourcePath := a.resolveThumbnailSource(id, artDir)
	if sourcePath == "" || !util.FileExists(sourcePath) {
		http.NotFound(w, r)
		return
	}

	cacheDir := filepath.Join(artDir, ".thumbs")
	thumbPath, err := util.GetOrGenerateThumbnail(sourcePath, targetWidth, cacheDir)
	if err != nil {
		slog.Warn("Failed to generate thumbnail, serving source", "source", sourcePath, "error", err)
		http.ServeFile(w, r, sourcePath)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if strings.HasSuffix(strings.ToLower(thumbPath), ".png") {
		w.Header().Set("Content-Type", "image/png")
	} else {
		w.Header().Set("Content-Type", "image/jpeg")
	}
	http.ServeFile(w, r, thumbPath)
}

func (a *App) resolveThumbnailSource(id, artDir string) string {
	// 1. Direct local filesystem image
	if strings.HasPrefix(id, "/local-img/") {
		rel := strings.TrimPrefix(id, "/local-img/")
		candidate := "/" + filepath.Clean(rel)
		if !strings.Contains(candidate, "..") && util.FileExists(candidate) {
			return candidate
		}
	} else if strings.HasPrefix(id, "/") && util.FileExists(id) {
		return id
	}

	cleanID := strings.TrimPrefix(id, "/art/")
	cleanID = strings.TrimPrefix(cleanID, "/")
	cleanID = filepath.Clean(cleanID)
	if strings.HasPrefix(cleanID, "..") {
		return ""
	}

	// 2. Direct file in artDir (e.g. half-life-2/cover.jpg)
	direct := filepath.Join(artDir, cleanID)
	if fi, err := os.Stat(direct); err == nil && !fi.IsDir() {
		return direct
	}

	// 3. Slug folder in artDir -> check candidates
	candidates := []string{"cover.jpg", "cover.png", "hero.jpg", "banner.jpg"}
	for _, c := range candidates {
		candPath := filepath.Join(direct, c)
		if util.FileExists(candPath) {
			return candPath
		}
	}

	// 4. Check cachedGames by AppID or slug
	a.gamesMu.RLock()
	var matchedGame *domain.Game
	for _, g := range a.cachedGames {
		if g == nil {
			continue
		}
		if g.AppID == cleanID || util.Slugify(g.Name) == cleanID || g.GameDir == cleanID {
			matchedGame = g
			break
		}
	}
	a.gamesMu.RUnlock()

	if matchedGame != nil {
		if matchedGame.LocalCover != "" && util.FileExists(matchedGame.LocalCover) {
			return matchedGame.LocalCover
		}
		slug := util.Slugify(matchedGame.Name)
		if slug != "" {
			for _, c := range candidates {
				candPath := filepath.Join(artDir, slug, c)
				if util.FileExists(candPath) {
					return candPath
				}
			}
		}
		for _, c := range candidates {
			candPath := filepath.Join(artDir, "game-"+matchedGame.AppID, c)
			if util.FileExists(candPath) {
				return candPath
			}
		}
	}

	// 5. Check artDir/game-<cleanID>/cover.jpg
	for _, c := range candidates {
		candPath := filepath.Join(artDir, "game-"+cleanID, c)
		if util.FileExists(candPath) {
			return candPath
		}
	}

	return ""
}

// SetTrayManager registers the TrayManager with the application so menu items stay updated.
func (a *App) SetTrayManager(tray *TrayManager) {
	a.trayMgr = tray
	if tray != nil {
		tray.SetRecentGames(a.GetRecentGames(3))
	}
}

// UpdateCachedGames refreshes the in-memory cache of games and pushes updates to the system tray.
func (a *App) UpdateCachedGames(games []*domain.Game) {
	if len(games) == 0 {
		return
	}
	a.gamesMu.Lock()
	a.cachedGames = games
	a.gamesMu.Unlock()

	if a.trayMgr != nil {
		a.trayMgr.SetRecentGames(a.GetRecentGames(3))
	}
}

// UpdateCachedGame updates or inserts a single game into the in-memory cache and updates the tray.
func (a *App) UpdateCachedGame(game *domain.Game) {
	if game == nil || game.GameDir == "" {
		return
	}
	cDir := util.CanonicalPath(game.GameDir)
	a.gamesMu.Lock()
	found := false
	for i, g := range a.cachedGames {
		if g != nil && util.CanonicalPath(g.GameDir) == cDir {
			a.cachedGames[i] = game
			found = true
			break
		}
	}
	if !found {
		a.cachedGames = append(a.cachedGames, game)
	}
	a.gamesMu.Unlock()

	if a.trayMgr != nil {
		a.trayMgr.SetRecentGames(a.GetRecentGames(3))
	}
}

// RemoveCachedGame removes a game from the in-memory cache and updates the tray.
func (a *App) RemoveCachedGame(gameDir string) {
	if gameDir == "" {
		return
	}
	cDir := util.CanonicalPath(gameDir)
	a.gamesMu.Lock()
	filtered := make([]*domain.Game, 0, len(a.cachedGames))
	for _, g := range a.cachedGames {
		if g != nil && util.CanonicalPath(g.GameDir) != cDir {
			filtered = append(filtered, g)
		}
	}
	a.cachedGames = filtered
	a.gamesMu.Unlock()

	if a.trayMgr != nil {
		a.trayMgr.SetRecentGames(a.GetRecentGames(3))
	}
}

// RecordGamePlayed updates the last played and playtime timestamps for a game in the cache.
func (a *App) RecordGamePlayed(gameDir string, lastPlayed int64, playtimeSeconds int64) {
	if gameDir == "" {
		return
	}
	cDir := util.CanonicalPath(gameDir)
	a.gamesMu.Lock()
	found := false
	for _, g := range a.cachedGames {
		if g != nil && util.CanonicalPath(g.GameDir) == cDir {
			found = true
			if g.LaunchProfile == nil {
				g.LaunchProfile = &domain.LaunchProfile{GameDir: g.GameDir}
			}
			g.LaunchProfile.LastPlayed = lastPlayed
			g.LastPlayed = lastPlayed
			if playtimeSeconds > 0 {
				g.LaunchProfile.PlaytimeSeconds = playtimeSeconds
				g.PlaytimeSeconds = playtimeSeconds
			}
			break
		}
	}
	a.gamesMu.Unlock()

	if !found && a.Engine != nil {
		if st, err := a.Engine.InspectStatus(context.Background(), cDir); err == nil && st != nil {
			a.gamesMu.Lock()
			a.cachedGames = append(a.cachedGames, st)
			a.gamesMu.Unlock()
		}
	}

	if a.trayMgr != nil {
		a.trayMgr.SetRecentGames(a.GetRecentGames(3))
	}
}

// GetRecentGames returns up to limit games sorted by least time since last play session to most.
func (a *App) GetRecentGames(limit int) []TrayGame {
	a.gamesMu.RLock()
	cached := make([]*domain.Game, len(a.cachedGames))
	copy(cached, a.cachedGames)
	a.gamesMu.RUnlock()

	if len(cached) == 0 && a.Engine != nil {
		dirs := a.Config.ResolvedLibraryDirs()
		if len(dirs) > 0 {
			if games, err := a.Engine.BatchInspectLibraries(context.Background(), dirs); err == nil && len(games) > 0 {
				a.gamesMu.Lock()
				a.cachedGames = games
				a.gamesMu.Unlock()
				cached = games
			}
		}
	}

	return FilterAndSortRecentGames(cached, limit)
}

