package engine

import (
	"context"
	"net/http"
	"path/filepath"
	"time"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/store"
	"github.com/staernid/gamux/tasks"
	"github.com/staernid/gamux/tools"
)

// ServiceContext holds the shared infrastructure injected into all domain services.
// It centralises cross-cutting concerns: HTTP client injection, event publishing,
// task management, and configuration — without being a God object (it has no
// domain-specific methods beyond infrastructure wiring).
type ServiceContext struct {
	Config         *config.Config
	HTTPClient     *http.Client
	EventBus       *events.Bus
	Tasks          *tasks.Manager
	RunnerMgr      *runner.Manager
	StoreRegistry  *store.Registry
	ArtworkService *store.ArtworkService
	Saves          *SaveService
	Notifier       func(ctx context.Context, title, message string) error
	StepReporter   func(ctx context.Context, prog StepProgress)
	toolManager    *tools.Manager
}

// NewServiceContext creates a new ServiceContext with all shared infrastructure initialised.
func NewServiceContext(cfg *config.Config) *ServiceContext {
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
	reg.Register(store.NewSGDBProvider(cfg.SteamGridDBAPIKey))

	artSvc := store.NewArtworkService(cfg.ArtworkDir, reg)

	taskMgr := tasks.NewManager(tasks.Options{
		MaxConcurrentDownloads:  2,
		MaxConcurrentOperations: 4,
		Store:                   storeAdapter,
		EventSink:               eventSink,
		Limiter:                 limiter,
	})

	sc := &ServiceContext{
		Config:         cfg,
		HTTPClient:     &http.Client{Timeout: 30 * time.Second},
		EventBus:       bus,
		Tasks:          taskMgr,
		RunnerMgr:      runner.NewManager(bus),
		StoreRegistry:  reg,
		ArtworkService: artSvc,
		toolManager:    tools.NewManager(cfg),
	}
	sc.Saves = NewSaveService(sc)
	return sc
}

// WrapContext decorates a context with the shared HTTP client for downstream
// Steam, GitHub, and Manifest API calls.
func (sc *ServiceContext) WrapContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if sc.HTTPClient != nil {
		ctx = steam.WithHTTPClient(ctx, sc.HTTPClient)
		ctx = github.WithHTTPClient(ctx, sc.HTTPClient)
		ctx = manifest.WithHTTPClient(ctx, sc.HTTPClient)
	}
	return ctx
}

// Notify publishes a system notification event to the EventBus and calls
// the Notifier callback if configured.
func (sc *ServiceContext) Notify(ctx context.Context, title, message string) error {
	if sc.EventBus != nil {
		sc.EventBus.Publish(events.NewNotification(title, message))
	}
	if sc.Notifier != nil {
		return sc.Notifier(ctx, title, message)
	}
	return nil
}

// PublishEvent sends a typed event to the event broker.
func (sc *ServiceContext) PublishEvent(ev events.Event) {
	if sc.EventBus != nil {
		sc.EventBus.Publish(ev)
	}
}

// Tools returns the Tool Manager instance.
func (sc *ServiceContext) Tools() *tools.Manager {
	if sc.toolManager == nil {
		sc.toolManager = tools.NewManager(sc.Config)
	}
	return sc.toolManager
}

// GetToolsStatus retrieves current statuses for all registered post-processing tools.
func (sc *ServiceContext) GetToolsStatus(ctx context.Context) ([]tools.ToolStatus, error) {
	return sc.Tools().GetToolsStatus(ctx)
}

// reportStep emits a step progress event for UI/CLI consumers.
func (sc *ServiceContext) reportStep(ctx context.Context, step, total int, title, details string) {
	if sc.EventBus != nil {
		sc.EventBus.Publish(events.NewStepProgress(step, total, title, details))
	}
	if sc.StepReporter != nil {
		sc.StepReporter(ctx, StepProgress{
			Step:    step,
			Total:   total,
			Title:   title,
			Details: details,
		})
	}
}
