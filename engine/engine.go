package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/lutris"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/steamless"
	"github.com/staernid/gamux/util"
	"golang.org/x/sync/errgroup"
)

// ProcessOptions holds configuration flags for processing a game directory.
type ProcessOptions struct {
	Path              string
	ApplyGBE          bool
	AddLutris         bool
	Portable          bool
	NormalizeDir      bool
	DryRun            bool
	AutoYes           bool
	NoSteamless       bool
	FetchAchievements bool
	WinePrefix        string
	Runner            string
	ExePath           string `json:"exe_path,omitempty"`
	ExeArgs           string `json:"exe_args,omitempty"`
}

// ProcessResult summarizes the outcome of processing a game.
type ProcessResult struct {
	Info                *detector.GameInfo
	LutrisAdded         bool
	Patched             bool
	SteamlessUnpacked   bool
	ManifestsMoved      bool
	DirectoryNormalized bool
	ACFSynthesized      bool
	Errors              []string
}

// GameStatus describes the current patch & integration state of a game.
type GameStatus struct {
	Name               string                     `json:"name"`
	AppID              string                     `json:"app_id"`
	Store              string                     `json:"store"` // "Steam", "GOG", "Epic", "Itch", "Custom"
	GameDir            string                     `json:"game_dir"`
	Platform           string                     `json:"platform"`
	ExePath            string                     `json:"exe_path"`
	ExeArgs            string                     `json:"exe_args,omitempty"`
	LaunchCandidates   []detector.LaunchCandidate `json:"launch_candidates,omitempty"`
	State              string                     `json:"state"` // "Original", "Loader-Configured", "Portable-Patched"
	OriginalBackups    []string                   `json:"original_backups"`
	SettingsDirExists  bool                       `json:"settings_dir_exists"`
	SteamAppIDTxtFound bool                       `json:"steam_appid_txt_found"`
	ManifestID         string                     `json:"manifest_id"`
	BuildID            string                     `json:"build_id"`
	DiskSizeBytes      int64                      `json:"disk_size_bytes"`
	FileCount          int                        `json:"file_count"`
	OfficialFileCount  int                        `json:"official_file_count"`
	ModifiedFiles      []string                   `json:"modified_files"`
	MissingFiles       []string                   `json:"missing_files"`
	UntrackedFiles     []string                   `json:"untracked_files"`
	RedistModified     []string                   `json:"redist_modified"`
	RedistMissing      []string                   `json:"redist_missing"`
	HasUpdate          bool                       `json:"has_update"`
	RemoteManifestID   string                     `json:"remote_manifest_id"`
	DLCCount           int                        `json:"dlc_count"`
	AchievementCount   int                        `json:"achievement_count"`
	LutrisRegistered   bool                       `json:"lutris_registered"`
	RecentPatchNote    string                     `json:"recent_patch_note"`
	NewsItems          []steam.NewsItem           `json:"news_items"`
}

// StepProgress defines a progress stage update for UI/CLI consumers.
type StepProgress struct {
	Step    int    `json:"step"`
	Total   int    `json:"total"`
	Title   string `json:"title"`
	Details string `json:"details"`
}

// Engine is the core domain service orchestrating game scanning, patching, and launcher integrations.
type Engine struct {
	Config       *config.Config
	Notifier     func(ctx context.Context, title, message string) error
	StepReporter func(ctx context.Context, prog StepProgress)
}

// New creates a new Engine instance.
func New(cfg *config.Config) *Engine {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &Engine{Config: cfg}
}

// ProcessGame executes the end-to-end post-download setup for a single game directory.
func ProcessGame(ctx context.Context, cfg *config.Config, opts ProcessOptions) (*ProcessResult, error) {
	eng := New(cfg)
	return eng.ProcessGame(ctx, opts)
}

func (e *Engine) reportStep(ctx context.Context, step, total int, title, details string) {
	if e.StepReporter != nil {
		e.StepReporter(ctx, StepProgress{
			Step:    step,
			Total:   total,
			Title:   title,
			Details: details,
		})
	}
}

// ProcessGame processes a game directory according to the supplied options.
func (e *Engine) ProcessGame(ctx context.Context, opts ProcessOptions) (*ProcessResult, error) {
	targetPath := opts.Path
	if targetPath == "" {
		targetPath = "."
	}

	e.reportStep(ctx, 1, 5, "Scanning Game Directory & Metadata", targetPath)

	if !opts.DryRun {
		if count, decErr := util.DecompressVSZaInDir(targetPath); decErr == nil && count > 0 {
			slog.Info("Decompressed raw VSZa Steam depot files", "target", targetPath, "count", count)
		}
	}

	info, err := detector.Detect(ctx, targetPath)
	if err != nil {
		return nil, fmt.Errorf("auto-detect failed for %s: %w", targetPath, err)
	}

	e.reportStep(ctx, 2, 5, "Synthesizing 1:1 ACF Manifest & Consolidating Depots", info.Name)

	// Manifest Auto-Fetch & Repair Offer if [Manifests] is missing
	if info.AppID != "" && info.AppID != "0" {
		manifestsDir := filepath.Join(info.GameDir, "[Manifests]")
		if entries, readErr := os.ReadDir(manifestsDir); readErr != nil || len(entries) == 0 {
			if appIDVal, parseErr := strconv.ParseUint(info.AppID, 10, 32); parseErr == nil {
				slog.Info("No local [Manifests] found, resolving keys and manifests", "appID", info.AppID)
				parsedLua, keyErr := manifest.ResolveKeys(ctx, uint32(appIDVal), "", "")
				if keyErr == nil && len(parsedLua.ManifestFiles) > 0 && !opts.DryRun {
					_ = os.MkdirAll(manifestsDir, 0755)
					for fname, fcontent := range parsedLua.ManifestFiles {
						outPath := filepath.Join(manifestsDir, fname)
						_ = os.WriteFile(outPath, fcontent, 0644)
						slog.Info("Auto-fetched and saved manifest file", "path", outPath)
					}
				}
			}
		}
	}

	res := &ProcessResult{
		Info: info,
	}

	if err := detector.ConsolidateManifests(info); err != nil {
		slog.Warn("Failed to consolidate manifests", "error", err)
		res.Errors = append(res.Errors, fmt.Sprintf("manifest consolidation: %v", err))
	} else {
		res.ManifestsMoved = true
	}

	if opts.NormalizeDir {
		if err := detector.NormalizeDirectory(info, opts.DryRun); err != nil {
			slog.Warn("Failed to normalize directory name", "error", err)
			res.Errors = append(res.Errors, fmt.Sprintf("directory normalization: %v", err))
		} else {
			res.DirectoryNormalized = true
		}
	}

	if err := detector.EnsureACFManifest(info, opts.DryRun); err != nil {
		slog.Warn("Failed to ensure 1:1 ACF manifest", "error", err)
		res.Errors = append(res.Errors, fmt.Sprintf("acf manifest synthesis: %v", err))
	} else {
		res.ACFSynthesized = true
	}

	if opts.ExePath != "" {
		if filepath.IsAbs(opts.ExePath) {
			info.ExePath = opts.ExePath
		} else {
			info.ExePath = filepath.Join(info.GameDir, opts.ExePath)
		}
	}
	if opts.ExeArgs != "" {
		info.ExeArgs = opts.ExeArgs
	}

	slog.Info("Auto-detected game", "title", info.Name, "appID", info.AppID, "platform", info.Platform, "exe", info.ExePath)


	// If WinePrefix is not specified, attempt to auto-detect from existing Lutris configuration
	if opts.WinePrefix == "" {
		slug := lutris.Slugify(info.Name)
		targetDir := util.ExpandPath(e.Config.LutrisDir)
		if existingPrefix := lutris.GetExistingWinePrefix(slug, targetDir); existingPrefix != "" {
			opts.WinePrefix = existingPrefix
			slog.Info("Auto-detected existing Lutris Wine prefix", "prefix", existingPrefix)
		}
	}

	// 1. Apply Goldberg Emulator & Steamless DRM Unpacking if enabled
	if opts.ApplyGBE {
		e.reportStep(ctx, 3, 5, "Unpacking DRM & Patching Goldberg Emulator", info.Name)
		if opts.NoSteamless {
			slog.Info("Steamless DRM unpacking disabled via --no-steamless flag", "title", info.Name)
		} else {
			unpacked, err := steamless.UnpackExecutable(ctx, e.Config, info.GameDir, opts.DryRun)
			if err != nil {
				slog.Warn("Steamless unpacking error", "error", err)
				res.Errors = append(res.Errors, fmt.Sprintf("steamless unpack: %v", err))
			}
			res.SteamlessUnpacked = unpacked
		}

		if err := gbe.ApplyGBE(ctx, e.Config, info.GameDir, info.Platform, info.AppID, opts.DryRun, opts.Portable, opts.WinePrefix, info.ExePath); err != nil {
			slog.Warn("GBE application warning/error", "error", err)
			res.Errors = append(res.Errors, fmt.Sprintf("gbe patch: %v", err))
		} else {
			res.Patched = true
		}

		if opts.FetchAchievements {
			e.reportStep(ctx, 4, 5, "Generating Achievements & DLC Schemas", "AppID: "+info.AppID)
			var appIDUint uint32
			fmt.Sscanf(info.AppID, "%d", &appIDUint)
			if appIDUint > 0 {
				if err := gbe.GenerateAchievementsJSON(ctx, appIDUint, e.Config.SteamWebAPIKey, info.GameDir, opts.DryRun); err != nil {
					slog.Warn("Achievement schema generation warning/error", "error", err)
				}
			}
		}
	} else {
		slog.Info("GBE & DRM removal skipped as requested", "title", info.Name)
	}


	// 2. Add to Lutris
	if opts.AddLutris {
		e.reportStep(ctx, 5, 5, "Writing Lutris Configuration & Artwork", info.Name)
		if strings.TrimSpace(info.ExePath) == "" {
			err := fmt.Errorf("%w: cannot register in Lutris: no executable detected in %s", detector.ErrNoExecutableFound, info.GameDir)
			slog.Error(err.Error())
			res.Errors = append(res.Errors, err.Error())
			return res, err
		}

		runner := opts.Runner
		if runner == "" {
			if info.Platform == "linux" {
				runner = "linux"
			} else {
				runner = "wine"
			}
		}

		gamePath := info.ExePath
		var env map[string]string
		if !opts.Portable {
			gbeBase := util.ExpandPath(e.Config.GbeDir)
			env = make(map[string]string)
			if runner == "linux" {
				env["LD_PRELOAD"] = filepath.Join(gbeBase, "linux_release", "experimental", "x64", "steamclient.so")
			} else {
				loaderExeName := "steamclient_loader_x64.exe"
				if info.Platform == "win32" {
					loaderExeName = "steamclient_loader_x86.exe"
				}
				gamePath = filepath.Join(info.GameDir, loaderExeName)
				env["PROTON_DISABLE_LSTEAMCLIENT"] = "1"
			}
		}

		var sysConfig *lutris.SystemConfig
		if e.Config.EnableLaunchNotify {
			if gamuxBin, err := os.Executable(); err == nil {
				sysConfig = &lutris.SystemConfig{
					PreLaunchScript: gamuxBin,
					PreLaunchArg:    fmt.Sprintf("notify-launch --path %q", info.GameDir),
					PreLaunchWait:   true,
				}
			}
		}

		lcfg := lutris.Config{
			Name:                  info.Name,
			GamePath:              gamePath,
			Args:                  info.ExeArgs,
			Runner:                runner,
			PrefixPath:            opts.WinePrefix,
			Env:                   env,
			System:                sysConfig,
			CreateMenuShortcut:    false,
			CreateDesktopShortcut: false,
		}

		if opts.DryRun {
			slog.Info("[DRY RUN] Would write Lutris config & register in database", "name", info.Name)
			res.LutrisAdded = true
		} else {
			targetDir := util.ExpandPath(e.Config.LutrisDir)
			if err := lutris.Write(lcfg, targetDir); err == nil {
				slog.Info("Successfully wrote Lutris game config & updated database", "name", info.Name, "dir", targetDir)
				_, _ = steam.FetchLutrisArtwork(ctx, info.AppID, lutris.Slugify(info.Name), false)
				res.LutrisAdded = true
			} else {

				slog.Error("Failed to write Lutris game config", "error", err)
				res.Errors = append(res.Errors, fmt.Sprintf("lutris register: %v", err))
				return res, fmt.Errorf("failed to register in Lutris: %w", err)
			}
		}
	}

	return res, nil
}

// BatchProcess scans a parent directory containing multiple game folders and processes each discovered game.
func (e *Engine) BatchProcess(ctx context.Context, parentDir string, opts ProcessOptions) ([]*ProcessResult, error) {
	absParent, err := filepath.Abs(parentDir)
	if err != nil {
		return nil, fmt.Errorf("resolve batch directory: %w", err)
	}

	entries, err := os.ReadDir(absParent)
	if err != nil {
		return nil, fmt.Errorf("read batch directory %s: %w", absParent, err)
	}

	var results []*ProcessResult
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		gamePath := filepath.Join(absParent, entry.Name())

		// Skip hidden folders
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		slog.Info("Batch scanning candidate game directory", "path", gamePath)
		subOpts := opts
		subOpts.Path = gamePath

		res, err := e.ProcessGame(ctx, subOpts)
		if err != nil {
			slog.Warn("Failed to batch process game directory", "path", gamePath, "error", err)
			continue
		}
		results = append(results, res)
	}

	return results, nil
}

// BatchInspect scans all candidate game subdirectories in parentDir and returns their GameStatus.
func (e *Engine) BatchInspect(ctx context.Context, parentDir string) ([]*GameStatus, error) {
	absParent, err := filepath.Abs(parentDir)
	if err != nil {
		return nil, fmt.Errorf("resolve parent directory path: %w", err)
	}

	entries, err := os.ReadDir(absParent)
	if err != nil {
		return nil, fmt.Errorf("read parent directory %s: %w", absParent, err)
	}

	var validPaths []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		validPaths = append(validPaths, filepath.Join(absParent, entry.Name()))
	}

	results := make([]*GameStatus, len(validPaths))
	var g errgroup.Group
	g.SetLimit(16)

	for i, p := range validPaths {
		idx := i
		path := p
		g.Go(func() error {
			status, err := e.InspectStatusEx(ctx, path, 0)
			if err == nil && status != nil {
				results[idx] = status
			}
			return nil
		})
	}

	_ = g.Wait()

	var finalResults []*GameStatus
	for _, res := range results {
		if res != nil {
			finalResults = append(finalResults, res)
		}
	}
	return finalResults, nil
}



// InspectStatus inspects a game directory to determine its patch and configuration state.
func (e *Engine) InspectStatus(ctx context.Context, targetPath string) (*GameStatus, error) {

	info, err := detector.Detect(ctx, targetPath)
	if err != nil {
		return nil, fmt.Errorf("detect game: %w", err)
	}

	status := &GameStatus{
		Name:             info.Name,
		AppID:            info.AppID,
		Store:            info.Store,
		GameDir:          info.GameDir,
		Platform:         info.Platform,
		ExePath:          info.ExePath,
		ExeArgs:          info.ExeArgs,
		LaunchCandidates: info.LaunchCandidates,
		State:            "Original",
	}


	// Walk game directory to check for backup files, size, and file count
	_ = filepath.WalkDir(info.GameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			status.FileCount++
			if fi, err := d.Info(); err == nil {
				status.DiskSizeBytes += fi.Size()
			}
		}
		name := d.Name()
		if strings.Contains(name, ".ORIGINAL") {
			status.OriginalBackups = append(status.OriginalBackups, path)
		}
		if name == "steam_appid.txt" {
			status.SteamAppIDTxtFound = true
		}
		return nil
	})

	// Check manifests
	manifestsDir := filepath.Join(info.GameDir, "[Manifests]")
	if entries, err := os.ReadDir(manifestsDir); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".manifest") {
				parts := strings.Split(e.Name(), "_")
				if len(parts) >= 2 {
					status.ManifestID = strings.TrimSuffix(parts[1], ".manifest")
				} else {
					status.ManifestID = e.Name()
				}
				break
			}
		}
	}

	// Check Lutris
	slug := lutris.Slugify(info.Name)
	lutrisTargetDir := e.Config.LutrisDir
	if !filepath.IsAbs(lutrisTargetDir) {
		if home, err := os.UserHomeDir(); err == nil {
			lutrisTargetDir = filepath.Join(home, lutrisTargetDir)
		}
	}
	if lutris.GetExistingWinePrefix(slug, lutrisTargetDir) != "" || util.FileExists(filepath.Join(lutrisTargetDir, slug+".yml")) {
		status.LutrisRegistered = true
	}

	// Check recent news
	if info.AppID != "" && info.AppID != "0" {
		if news, err := steam.FetchAppNews(ctx, info.AppID); err == nil {
			status.RecentPatchNote = news
		}
	}

	// Scan depot integrity & untracked/mod files via manifest file list comparison
	if appIDVal, err := strconv.ParseUint(info.AppID, 10, 32); err == nil && appIDVal > 0 {
		if rep, err := manifest.ScanDepotIntegrity(info.GameDir, uint32(appIDVal)); err == nil && rep != nil {
			status.OfficialFileCount = rep.OfficialCount
			status.ModifiedFiles = rep.ModifiedFiles
			status.MissingFiles = rep.MissingFiles
			status.UntrackedFiles = rep.UntrackedFiles
			status.RedistModified = rep.RedistModified
			status.RedistMissing = rep.RedistMissing
			if len(rep.ModifiedFiles) > 0 || len(rep.MissingFiles) > 0 {
				status.HasUpdate = true
			}
		}
	}


	steamSettingsDir := filepath.Join(info.GameDir, "steam_settings")
	if _, err := os.Stat(steamSettingsDir); err == nil {
		status.SettingsDirExists = true
	}

	coldClientIni := filepath.Join(info.GameDir, "ColdClientLoader.ini")
	if len(status.OriginalBackups) > 0 {
		status.State = "Portable-Patched"
	} else if util.FileExists(coldClientIni) || status.SettingsDirExists || status.SteamAppIDTxtFound {
		status.State = "Loader-Configured"
	}

	return status, nil
}

// ApplyGame performs the end-to-end One-Shot lifecycle: Inspect -> Normalize -> Synthesize ACF -> GBE Setup -> Lutris Register.
func (e *Engine) ApplyGame(ctx context.Context, opts ProcessOptions) (*ProcessResult, error) {
	status, err := e.InspectStatus(ctx, opts.Path)
	if err != nil {
		slog.Warn("Apply inspection warning", "error", err)
	}

	opts.NormalizeDir = true

	if status != nil && status.HasUpdate {
		slog.Info("Apply detected available Steam depot update", "title", status.Name, "localManifest", status.ManifestID)
	}

	return e.ProcessGame(ctx, opts)
}


// InspectStatusEx extends InspectStatus with customizable news history depth.
func (e *Engine) InspectStatusEx(ctx context.Context, targetPath string, newsCount int) (*GameStatus, error) {
	status, err := e.InspectStatus(ctx, targetPath)
	if err != nil {
		return nil, err
	}
	if newsCount > 1 && status.AppID != "" && status.AppID != "0" {
		if items, err := steam.FetchAppNewsItems(ctx, status.AppID, newsCount); err == nil {
			status.NewsItems = items
		}
	}
	return status, nil
}

// VerifyIntegrity runs an on-demand full depot integrity comparison for a game directory.
func (e *Engine) VerifyIntegrity(ctx context.Context, targetPath string) (*manifest.IntegrityReport, error) {
	info, err := detector.Detect(ctx, targetPath)
	if err != nil {
		return nil, fmt.Errorf("detect game for integrity check: %w", err)
	}

	appIDVal, err := strconv.ParseUint(info.AppID, 10, 32)
	if err != nil || appIDVal == 0 {
		return nil, fmt.Errorf("invalid AppID %q for integrity verification", info.AppID)
	}

	return manifest.ScanDepotIntegrity(info.GameDir, uint32(appIDVal))
}

// GetPatchNote fetches and retrieves a single patch note at itemIndex (1-indexed) for a game path or AppID.
func (e *Engine) GetPatchNote(ctx context.Context, targetPath string, itemIndex int) (*steam.NewsItem, string, error) {
	appID := targetPath
	gameName := targetPath

	if info, err := detector.Detect(ctx, targetPath); err == nil && info.AppID != "" && info.AppID != "0" {
		appID = info.AppID
		gameName = info.Name
	}

	items, err := steam.FetchAppNewsItems(ctx, appID, itemIndex+5)
	if err != nil {
		return nil, gameName, fmt.Errorf("fetch news: %w", err)
	}

	if itemIndex <= 0 || itemIndex > len(items) {
		return nil, gameName, fmt.Errorf("news item index %d out of range (1-%d available)", itemIndex, len(items))
	}

	return &items[itemIndex-1], gameName, nil
}

// Rollback restores original backup files and cleans up generated configuration files in a game directory.
func (e *Engine) Rollback(ctx context.Context, targetPath string, dryRun bool) error {
	info, err := detector.Detect(ctx, targetPath)
	if err != nil {
		return fmt.Errorf("detect game for rollback: %w", err)
	}

	slog.Info("Starting rollback for game", "name", info.Name, "dir", info.GameDir)

	var restoredCount int
	err = filepath.WalkDir(info.GameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.Contains(d.Name(), ".ORIGINAL") {
			idx := strings.Index(path, ".20")
			origPath := path
			if idx != -1 {
				origPath = path[:idx]
			} else {
				origPath = strings.TrimSuffix(path, ".ORIGINAL")
			}

			if dryRun {
				slog.Info("[DRY RUN] Would restore original file", "from", path, "to", origPath)
			} else {
				if err := os.Rename(path, origPath); err != nil {
					slog.Error("Failed to restore original file", "from", path, "to", origPath, "error", err)
				} else {
					slog.Info("Restored original file", "restored", origPath)
					restoredCount++
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk dir during rollback: %w", err)
	}

	// Clean up loader files and configurations
	loaderFiles := []string{
		"ColdClientLoader.ini",
		"steamclient_loader_x64.exe",
		"steamclient_loader_x86.exe",
		"steamclient64.dll",
		"steamclient.dll",
		"GameOverlayRenderer64.dll",
		"GameOverlayRenderer.dll",
		"steam_interfaces.txt",
		".steam_interfaces.hash",
	}

	for _, fname := range loaderFiles {
		p := filepath.Join(info.GameDir, fname)
		if util.FileExists(p) {
			if dryRun {
				slog.Info("[DRY RUN] Would remove loader file", "path", p)
			} else {
				_ = os.Remove(p)
				slog.Info("Removed loader file", "path", p)
			}
		}
	}

	steamSettingsDir := filepath.Join(info.GameDir, "steam_settings")
	if _, err := os.Stat(steamSettingsDir); err == nil {
		if dryRun {
			slog.Info("[DRY RUN] Would remove steam_settings directory", "path", steamSettingsDir)
		} else {
			if err := os.RemoveAll(steamSettingsDir); err == nil {
				slog.Info("Removed steam_settings directory", "path", steamSettingsDir)
			}
		}
	}


	appIDTxt := filepath.Join(info.GameDir, "steam_appid.txt")
	if util.FileExists(appIDTxt) {
		if dryRun {
			slog.Info("[DRY RUN] Would remove steam_appid.txt", "path", appIDTxt)
		} else {
			_ = os.Remove(appIDTxt)
			slog.Info("Removed steam_appid.txt", "path", appIDTxt)
		}
	}

	// Clean up Lutris integration if present
	slug := lutris.Slugify(info.Name)
	targetDir := e.Config.LutrisDir
	if !filepath.IsAbs(targetDir) {
		home, err := os.UserHomeDir()
		if err == nil {
			targetDir = filepath.Join(home, targetDir)
		}
	}

	if dryRun {
		slog.Info("[DRY RUN] Would unregister game from Lutris and clean up Lutris configs", "slug", slug)
	} else {
		if err := lutris.UnregisterLutris(slug, targetDir); err == nil {
			slog.Info("Unregistered game from Lutris and cleaned up Lutris configs/shortcuts", "slug", slug)
		}
	}

	slog.Info("Rollback completed", "restoredFiles", restoredCount)
	return nil
}

// LibraryUpdateStatus represents update availability status for a game in the library.
type LibraryUpdateStatus struct {
	GameTitle       string
	AppID           string
	GameDir         string
	LocalManifestID string
	HasUpdate       bool
}

// CheckLibraryUpdates scans a directory containing game folders and checks for available Steam updates.
func (e *Engine) CheckLibraryUpdates(ctx context.Context, parentDir string) ([]LibraryUpdateStatus, error) {
	if parentDir == "" {
		parentDir = "."
	}

	absPath, err := filepath.Abs(parentDir)
	if err != nil {
		return nil, fmt.Errorf("resolve abs path: %w", err)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("read library directory %s: %w", absPath, err)
	}

	var results []LibraryUpdateStatus
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		gameDir := filepath.Join(absPath, entry.Name())
		manifestsDir := filepath.Join(gameDir, "[Manifests]")
		if _, err := os.Stat(manifestsDir); err != nil {
			continue
		}

		info, err := detector.Detect(ctx, gameDir)
		if err != nil || info.AppID == "" || info.AppID == "0" {
			continue
		}

		status := LibraryUpdateStatus{
			GameTitle: info.Name,
			AppID:     info.AppID,
			GameDir:   gameDir,
		}

		manifestEntries, _ := os.ReadDir(manifestsDir)
		for _, mFile := range manifestEntries {
			if strings.HasSuffix(mFile.Name(), ".manifest") {
				status.LocalManifestID = mFile.Name()
				break
			}
		}

		results = append(results, status)
	}

	return results, nil
}

// NotifyLaunch inspects a game directory for updates or configuration issues, and triggers a console/desktop notification if found.
func (e *Engine) NotifyLaunch(ctx context.Context, gameDir string) error {
	status, err := e.InspectStatus(ctx, gameDir)
	if err != nil {
		slog.Warn("NotifyLaunch status inspection failed", "dir", gameDir, "error", err)
		return nil // Don't block launch if inspection fails
	}

	var issues []string
	if status.HasUpdate {
		issues = append(issues, "An update is available for this game on Steam.")
	}
	if len(status.MissingFiles) > 0 {
		issues = append(issues, fmt.Sprintf("%d official game files are missing.", len(status.MissingFiles)))
	}
	if len(status.ModifiedFiles) > 0 {
		issues = append(issues, fmt.Sprintf("%d game files have been modified.", len(status.ModifiedFiles)))
	}

	if len(issues) == 0 {
		return nil
	}

	msg := fmt.Sprintf("Game: %s (AppID: %s)\nPath: %s\n\nIssues Detected:\n", status.Name, status.AppID, status.GameDir)
	for _, issue := range issues {
		msg += fmt.Sprintf(" • %s\n", issue)
	}
	msg += fmt.Sprintf("\nSuggested Action: Run 'gamux apply %q' to resolve.", status.GameDir)

	slog.Info("Pre-launch notification sent", "title", status.Name, "issuesCount", len(issues))
	if e.Notifier != nil {
		return e.Notifier(ctx, status.Name, msg)
	}
	return nil
}

