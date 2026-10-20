package pipeline

import (
	"context"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/tasks"
)

// Options holds user-facing execution flags for pipeline stages.
// This is the single source of truth — CLI, GUI, and engine all construct this directly.
type Options struct {
	Path              string                 `json:"path"`
	OperatingMode     string                 `json:"operating_mode,omitempty"` // "loader", "portable", "standalone", "native", "none"
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

// Results tracks mutations, syntheses, unpacks, reports, and errors across executed stages.
type Results struct {
	DirectoryNormalized bool                      `json:"directory_normalized"`
	OriginalDir         string                    `json:"original_dir,omitempty"`
	RenamedDir          string                    `json:"renamed_dir,omitempty"`
	ACFSynthesized      bool                      `json:"acf_synthesized"`
	SteamlessUnpacked   bool                      `json:"steamless_unpacked"`
	Patched             bool                      `json:"patched"`
	IntegrityReport     *manifest.IntegrityReport `json:"integrity_report,omitempty"`
	DownloadedBytes     int64                     `json:"downloaded_bytes,omitempty"`
	RestoredCount       int                       `json:"restored_count,omitempty"`
	Errors              []string                  `json:"errors,omitempty"`
	Mutations           []string                  `json:"mutations,omitempty"`
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
