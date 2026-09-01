package engine

import (
	"context"
	"net/http"
	"path/filepath"
	"time"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/events"
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
	Notifier       func(ctx context.Context, title, message string) error
	StepReporter   func(ctx context.Context, prog StepProgress)
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

	eng := &Engine{
		Config:         cfg,
		HTTPClient:     httpClient,
		EventBus:       bus,
		Tasks:          taskMgr,
		RunnerMgr:      runner.NewManager(bus),
		StoreRegistry:  reg,
		ArtworkService: artSvc,
		Saves:          savesSvc,
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
