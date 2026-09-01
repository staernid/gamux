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

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/util"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Go backend adapter exposed to the Wails frontend.
type App struct {
	ctx    context.Context
	Config *config.Config
	Engine *engine.Engine
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

func safeEmit(ctx context.Context, eventName string, data ...any) {
	if ctx == nil || ctx.Value("events") == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("Ignoring panic during safeEmit (context likely closed)", "event", eventName, "recover", r)
		}
	}()
	wailsRuntime.EventsEmit(ctx, eventName, data...)
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

	// Bind central EventBus to Wails frontend event emitter
	if a.Engine != nil && a.Engine.EventBus != nil {
		a.Engine.EventBus.SubscribeFunc(func(ev events.Event) {
			safeEmit(a.ctx, "gamux:task-event", ev)
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
					"gameDir":   ev.GameDir,
					"isRunning": true,
					"pid":       parseMetaInt(ev.Metadata, "pid"),
				})
			}
			if ev.Type == events.EventGameExited {
				safeEmit(a.ctx, "gamux:game-status", map[string]any{
					"gameDir":         ev.GameDir,
					"isRunning":       false,
					"pid":             0,
					"sessionSeconds":  parseMetaInt64(ev.Metadata, "session_seconds"),
					"playtimeSeconds": parseMetaInt64(ev.Metadata, "playtime_seconds"),
					"lastPlayed":      parseMetaInt64(ev.Metadata, "last_played"),
				})
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

	if a.Engine != nil {
		// Bind desktop/toast notifications to EventBus
		a.Engine.Notifier = func(nCtx context.Context, title, message string) error {
			if a.Engine != nil && a.Engine.EventBus != nil {
				a.Engine.EventBus.Publish(events.NewNotification(title, message))
			}
			return nil
		}

		// Bind step progress updates to EventBus
		a.Engine.StepReporter = func(sCtx context.Context, prog engine.StepProgress) {
			if a.Engine != nil && a.Engine.EventBus != nil {
				a.Engine.EventBus.Publish(events.NewStepProgress(prog.Step, prog.Total, prog.Title, prog.Details))
			}
		}
	}

	slog.Info("Gamux GUI backend initialized")
}

// getContext returns the current context (useful for fallback and testing).
func (a *App) getContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// AssetHandler returns an http.Handler that serves cached artwork and local game images from disk.
func (a *App) AssetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/art/") {
			rel := strings.TrimPrefix(r.URL.Path, "/art/")
			rel = filepath.Clean(rel)
			if strings.HasPrefix(rel, "..") {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			artDir := ""
			if a.Config != nil && a.Config.ArtworkDir != "" {
				artDir = util.ExpandPath(a.Config.ArtworkDir)
			} else {
				home, _ := os.UserHomeDir()
				artDir = filepath.Join(home, ".local", "share", "gamux", "art")
			}
			filePath := filepath.Join(artDir, rel)
			if util.FileExists(filePath) {
				w.Header().Set("Cache-Control", "no-cache, must-revalidate")
				http.ServeFile(w, r, filePath)
				return
			}
		} else if strings.HasPrefix(r.URL.Path, "/local-img/") {
			rel := strings.TrimPrefix(r.URL.Path, "/local-img/")
			filePath := "/" + rel
			filePath = filepath.Clean(filePath)
			if util.FileExists(filePath) {
				w.Header().Set("Cache-Control", "no-cache, must-revalidate")
				http.ServeFile(w, r, filePath)
				return
			}
		}
		http.NotFound(w, r)
	})
}
