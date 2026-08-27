package pipeline

import (
	"context"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/tasks"
)

// Options holds user-facing execution flags for pipeline stages.
// This is the single source of truth — CLI, GUI, and engine all construct this directly.
type Options struct {
	Path              string `json:"path"`
	ApplyGBE          bool   `json:"apply_gbe"`
	Portable          bool   `json:"portable"`
	NormalizeDir      bool   `json:"normalize_dir"`
	DryRun            bool   `json:"dry_run"`
	AutoYes           bool   `json:"auto_yes"`
	NoSteamless       bool   `json:"no_steamless"`
	FetchAchievements bool   `json:"fetch_achievements"`
	Language          string                 `json:"language,omitempty"`
	WinePrefix        string                 `json:"wine_prefix,omitempty"`
	ExePath           string                 `json:"exe_path,omitempty"`
	ExeArgs           string                 `json:"exe_args,omitempty"`
	Reporter          tasks.ProgressReporter `json:"-"`
}

// Context encapsulates execution state, options, and results passed across pipeline stages.
type Context struct {
	Options // Embedded — all user-facing execution flags

	Game   *domain.Game
	Config *config.Config

	// Results tracking
	DirectoryNormalized bool
	ACFSynthesized      bool
	SteamlessUnpacked   bool
	Patched             bool
	Errors              []string
	Mutations           []string

	// Telemetry reporting
	Reporter tasks.ProgressReporter
	EventBus *events.Bus
}

// Stage is an isolated, single-responsibility step in a game processing pipeline.
type Stage interface {
	Name() string
	Execute(ctx context.Context, pctx *Context) error
}

// RollbackableStage is a stage capable of undoing its filesystem changes upon pipeline failure.
type RollbackableStage interface {
	Stage
	Rollback(ctx context.Context, pctx *Context) error
}
