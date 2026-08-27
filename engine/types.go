package engine

import (
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/downloader"
	"github.com/staernid/gamux/pipeline"
)

// GameStatus describes the current patch & integration state of a game.
type GameStatus = domain.Game

// ProcessOptions is an alias for pipeline.Options — the single source of truth
// for game processing execution flags shared across CLI, GUI, and pipeline stages.
type ProcessOptions = pipeline.Options

// StepProgress defines a progress stage update for UI/CLI consumers.
type StepProgress struct {
	Step    int    `json:"step"`
	Total   int    `json:"total"`
	Title   string `json:"title"`
	Details string `json:"details"`
}

// ProcessResult summarizes the outcome of processing a game.
type ProcessResult struct {
	Info                *detector.GameInfo `json:"info,omitempty"`
	Patched             bool               `json:"patched"`
	SteamlessUnpacked   bool               `json:"steamless_unpacked"`
	ManifestsMoved      bool               `json:"manifests_moved"`
	DirectoryNormalized bool               `json:"directory_normalized"`
	ACFSynthesized      bool               `json:"acf_synthesized"`
	Errors              []string           `json:"errors,omitempty"`
}

// LibraryUpdateStatus represents update availability status for a game in the library.
type LibraryUpdateStatus struct {
	GameTitle       string
	AppID           string
	GameDir         string
	LocalManifestID string
	HasUpdate       bool
}

// DownloadGameOptions defines parameters for downloading or updating game depots.
type DownloadGameOptions struct {
	AppID                uint32                                                                             `json:"app_id"`
	TargetDir            string                                                                             `json:"target_dir"`
	Platform             string                                                                             `json:"platform"`
	LuaPath              string                                                                             `json:"lua_path"`
	DryRun               bool                                                                               `json:"dry_run"`
	WorkerLimit          int                                                                                `json:"worker_limit,omitempty"`
	AutoApply            bool                                                                               `json:"auto_apply"`
	ApplyOptions         *ProcessOptions                                                                    `json:"apply_options,omitempty"`
	ProgressCallback     func(current, total int, item string)                                              `json:"-"`
	ByteProgressCallback func(currBytes, totalBytes int64, speed float64, eta int, pct float64, item string) `json:"-"`
	OnProgress           func(update downloader.ProgressUpdate)                                             `json:"-"`
	CheckCancel          func() error                                                                       `json:"-"`
	Throttle             func(bytes int) error                                                              `json:"-"`
}
