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
	Path              string                 `json:"path"`
	OperatingMode     string                 `json:"operating_mode,omitempty"` // "loader", "portable", "standalone", "native", "none"
	ApplyGBE          bool                   `json:"apply_gbe,omitempty"`      // Convenience flag for backward compatibility
	Portable          bool                   `json:"portable,omitempty"`       // Convenience flag for backward compatibility
	NormalizeDir      bool                   `json:"normalize_dir"`
	DryRun            bool                   `json:"dry_run"`
	AutoYes           bool                   `json:"auto_yes"`
	NoSteamless       bool                   `json:"no_steamless"`
	FetchAchievements bool                   `json:"fetch_achievements"`
	Language          string                 `json:"language,omitempty"`
	WinePrefix        string                 `json:"wine_prefix,omitempty"`
	ExePath           string                 `json:"exe_path,omitempty"`
	ExeArgs           string                 `json:"exe_args,omitempty"`
	Reporter          tasks.ProgressReporter `json:"-"`
}

// EffectiveOperatingMode resolves the operating mode requested by options,
// taking into account legacy flags (Portable, ApplyGBE) if OperatingMode is empty.
func (o *Options) EffectiveOperatingMode() string {
	if o.OperatingMode != "" {
		return o.OperatingMode
	}
	if o.Portable {
		return domain.ModePortable
	}
	if o.ApplyGBE {
		return domain.ModeLoader
	}
	return ""
}

// Results tracks mutations, syntheses, unpacks, and errors across executed stages.
type Results struct {
	DirectoryNormalized bool     `json:"directory_normalized"`
	ACFSynthesized      bool     `json:"acf_synthesized"`
	SteamlessUnpacked   bool     `json:"steamless_unpacked"`
	Patched             bool     `json:"patched"`
	Errors              []string `json:"errors,omitempty"`
	Mutations           []string `json:"mutations,omitempty"`
}

// Context encapsulates execution state, options, and results passed across pipeline stages.
type Context struct {
	Options Options
	Game    *domain.Game
	Config  *config.Config
	Results Results

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
