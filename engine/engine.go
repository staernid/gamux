package engine

import (
	"context"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/multiseat"
	"github.com/staernid/gamux/profiles"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/saves"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/store"
	"github.com/staernid/gamux/tasks"
	"github.com/staernid/gamux/tools"
)

// Engine is the deep orchestration module for all game management, library scanning,
// task supervision, DRM setup, execution, savegame backup, and telemetry operations.
type Engine struct {
	Config         *config.Config
	HTTPClient     *http.Client
	EventBus       *events.Bus
	Tasks          *tasks.Manager
	RunnerMgr      *runner.Manager
	StoreRegistry  *store.Registry
	ArtworkService *store.ArtworkService
	Saves          *saves.Service
	Multiseat      *multiseat.Supervisor
	Profiles       *profiles.Manager
	toolManager    *tools.Manager
}

// New creates a new Engine with all domain and infrastructure services initialized.
func New(cfg *config.Config) *Engine {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	bus := events.NewBus()
	tasksFile := filepath.Join(filepath.Dir(config.GetConfigPath("")), "tasks.json")
	storeAdapter := tasks.NewFileStore(tasksFile)
	eventSink := tasks.NewBusEventSink(bus)
	limiter := tasks.NewTokenBucketLimiter(cfg.TransferSpeedLimit)

	reg := store.NewRegistry()
	reg.Register(store.NewSteamProvider())
	reg.Register(store.NewGOGProvider())
	reg.Register(store.NewRAWGProvider(cfg.RAWGAPIKey))
	reg.Register(store.NewSGDBProvider(cfg.SteamGridDBAPIKey))
	reg.Register(store.NewSteamGridProxyProvider())

	artSvc := store.NewArtworkService(cfg.ArtworkDir, reg)

	taskMgr := tasks.NewManager(tasks.Options{
		MaxConcurrentDownloads:  2,
		MaxConcurrentOperations: 4,
		Store:                   storeAdapter,
		EventSink:               eventSink,
		Limiter:                 limiter,
	})

	httpClient := steam.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	savesSvc := saves.NewService(saves.ServiceConfig{
		BackupDir:      cfg.BackupDir,
		RetentionCount: cfg.BackupRetentionCount,
		ManifestURL:    cfg.SaveManifestURL,
		HTTPClient:     httpClient,
		GlobalPrefix:   cfg.WinePrefix,
	})

	profMgr, err := profiles.NewManager("", cfg)
	if err != nil {
		slog.Warn("Failed to initialize profile manager", "error", err)
	}

	eng := &Engine{
		Config:         cfg,
		HTTPClient:     httpClient,
		EventBus:       bus,
		Tasks:          taskMgr,
		RunnerMgr:      runner.NewManager(bus),
		StoreRegistry:  reg,
		ArtworkService: artSvc,
		Saves:          savesSvc,
		Multiseat:      multiseat.NewSupervisor(cfg, bus),
		Profiles:       profMgr,
		toolManager:    tools.NewManager(cfg),
	}

	// Auto-backup on game process termination if configured
	bus.SubscribeFunc(func(ev events.Event) {
		if ev.Type == events.EventGameExited && eng.Config != nil && eng.Config.AutoBackupSaves && ev.GameDir != "" {
			go func(gDir string) {
				_, _ = eng.Saves.CreateBackup(context.Background(), gDir, "Automatic post-game snapshot", "auto_exit")
			}(ev.GameDir)
		}
	})

	eng.registerDefaultTaskRunners()
	return eng
}

// UpdateConfig dynamically reapplies a modified configuration in-place without tearing down
// the central EventBus, background tasks supervisor, or losing active UI/notification subscriptions.
func (eng *Engine) UpdateConfig(newCfg *config.Config) {
	if eng == nil || newCfg == nil {
		return
	}
	eng.Config = newCfg

	if eng.Tasks != nil {
		eng.Tasks.SetSpeedLimit(newCfg.TransferSpeedLimit)
	}

	reg := store.NewRegistry()
	reg.Register(store.NewSteamProvider())
	reg.Register(store.NewGOGProvider())
	reg.Register(store.NewRAWGProvider(newCfg.RAWGAPIKey))
	reg.Register(store.NewSGDBProvider(newCfg.SteamGridDBAPIKey))
	reg.Register(store.NewSteamGridProxyProvider())
	eng.StoreRegistry = reg
	eng.ArtworkService = store.NewArtworkService(newCfg.ArtworkDir, reg)

	eng.Saves = saves.NewService(saves.ServiceConfig{
		BackupDir:      newCfg.BackupDir,
		RetentionCount: newCfg.BackupRetentionCount,
		ManifestURL:    newCfg.SaveManifestURL,
		HTTPClient:     eng.HTTPClient,
		GlobalPrefix:   newCfg.WinePrefix,
	})

	if eng.Multiseat != nil {
		eng.Multiseat.SetConfig(newCfg)
	} else {
		eng.Multiseat = multiseat.NewSupervisor(newCfg, eng.EventBus)
	}
	eng.toolManager = tools.NewManager(newCfg)
}

