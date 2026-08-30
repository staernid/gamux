package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/downloader"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/steamless"
	"github.com/staernid/gamux/util"
)

// ScanStage inspects game directory, decompresses VSZa if needed, and resolves game metadata.
type ScanStage struct{}

func (s *ScanStage) Name() string {
	return "Scan & Detect Game"
}

func (s *ScanStage) Execute(ctx context.Context, pctx *Context) error {
	targetPath := pctx.Game.GameDir
	if targetPath == "" {
		targetPath = pctx.Options.Path
	}
	if targetPath == "" {
		targetPath = "."
	}

	if !pctx.Options.DryRun {
		if count, decErr := util.DecompressVSZaInDir(targetPath); decErr == nil && count > 0 {
			slog.Info("Decompressed raw VSZa Steam depot files", "target", targetPath, "count", count)
		}
	}

	info, err := detector.Inspect(ctx, targetPath)
	if err != nil {
		return fmt.Errorf("auto-detect failed for %s: %w", targetPath, err)
	}

	*pctx.Game = *info

	if pctx.Game.NumericAppID > 0 && (pctx.Game.Name == "" || pctx.Game.Name == pctx.Game.AppID || pctx.Game.Name == filepath.Base(targetPath)) {
		if name, err := steam.FetchAppName(ctx, pctx.Game.AppID); err == nil && name != "" {
			pctx.Game.Name = name
		}
	}

	if pctx.Options.ExePath != "" {
		if filepath.IsAbs(pctx.Options.ExePath) {
			pctx.Game.ExePath = pctx.Options.ExePath
		} else {
			pctx.Game.ExePath = filepath.Join(pctx.Game.GameDir, pctx.Options.ExePath)
		}
	}
	if pctx.Options.ExeArgs != "" {
		pctx.Game.ExeArgs = pctx.Options.ExeArgs
	}

	return nil
}

// ConsolidateManifestsInDir moves/copies external appmanifest and .manifest files into <gameDir>/[Manifests]/.
func ConsolidateManifestsInDir(gameDir, manifestPath string, rawDepotLayout bool) error {
	if gameDir == "" {
		return nil
	}

	manifestsDir := filepath.Join(gameDir, "[Manifests]")

	if manifestPath != "" {
		if strings.HasPrefix(manifestPath, manifestsDir) {
			return nil
		}
		_ = os.MkdirAll(manifestsDir, 0755)

		srcDir := filepath.Dir(manifestPath)
		destACF := filepath.Join(manifestsDir, filepath.Base(manifestPath))
		if manifestPath != destACF && util.FileExists(manifestPath) {
			if err := util.CopyFile(manifestPath, destACF); err == nil {
				slog.Info("Consolidated ACF manifest", "dest", destACF)
			}
		}

		// Also look for matching .manifest files in srcDir or depotcache
		scanDirs := []string{srcDir}
		if rawDepotLayout {
			rootParent := filepath.Dir(filepath.Dir(srcDir))
			scanDirs = append(scanDirs, filepath.Join(rootParent, "depotcache"))
		}

		for _, d := range scanDirs {
			entries, err := os.ReadDir(d)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".manifest") {
					srcMan := filepath.Join(d, entry.Name())
					destMan := filepath.Join(manifestsDir, entry.Name())
					if srcMan != destMan {
						if err := os.Rename(srcMan, destMan); err != nil {
							_ = util.CopyFile(srcMan, destMan)
							_ = os.Remove(srcMan)
						}
					}
				}
			}
		}
	}

	return nil
}

// ManifestStage consolidates and synthesizes 1:1 ACF manifests in [Manifests].
type ManifestStage struct {
	synthesizedACF string
}

func (s *ManifestStage) Name() string {
	return "Consolidate & Synthesize Manifests"
}

func (s *ManifestStage) Execute(ctx context.Context, pctx *Context) error {
	gameDir := pctx.Game.GameDir
	manifestsDir := filepath.Join(gameDir, "[Manifests]")

	// Skip if non-Steam game with no numeric AppID
	if pctx.Game.NumericAppID == 0 && pctx.Game.Store != "" && !strings.EqualFold(pctx.Game.Store, "steam") {
		return nil
	}

	// Consolidate manifests into [Manifests] if source manifest is tracked
	if pctx.Game.ManifestID != "" {
		_ = ConsolidateManifestsInDir(gameDir, pctx.Game.ManifestID, false)
	}

	// Auto-fetch manifests if missing
	if pctx.Game.NumericAppID > 0 {
		if entries, readErr := os.ReadDir(manifestsDir); readErr != nil || len(entries) == 0 {
			slog.Info("No local [Manifests] found, resolving keys and manifests", "appID", pctx.Game.AppID)
			parsedLua, keyErr := manifest.ResolveKeys(ctx, pctx.Game.NumericAppID, "", "")
			if keyErr == nil && len(parsedLua.ManifestFiles) > 0 && !pctx.Options.DryRun {
				_ = os.MkdirAll(manifestsDir, 0755)
				for fname, fcontent := range parsedLua.ManifestFiles {
					outPath := filepath.Join(manifestsDir, fname)
					_ = os.WriteFile(outPath, fcontent, 0644)
					slog.Info("Auto-fetched and saved manifest file", "path", outPath)
				}
			}
		}
	}

	// Synthesize 1:1 ACF manifest if missing
	if pctx.Game.NumericAppID > 0 {
		_ = os.MkdirAll(manifestsDir, 0755)
		acfName := fmt.Sprintf("appmanifest_%s.acf", pctx.Game.AppID)
		targetACF := filepath.Join(manifestsDir, acfName)

		if !util.FileExists(targetACF) {
			installDir := util.SanitizeInstallDir(pctx.Game.Name)
			var launchOpts []detector.LaunchCandidate
			if pctx.Game.ExePath != "" {
				relExe, err := filepath.Rel(pctx.Game.GameDir, pctx.Game.ExePath)
				if err == nil && !strings.HasPrefix(relExe, "..") {
					launchOpts = append(launchOpts, detector.LaunchCandidate{
						Executable:  relExe,
						Arguments:   pctx.Game.ExeArgs,
						Description: pctx.Game.Name,
					})
				} else {
					launchOpts = append(launchOpts, detector.LaunchCandidate{
						Executable:  pctx.Game.ExePath,
						Arguments:   pctx.Game.ExeArgs,
						Description: pctx.Game.Name,
					})
				}
			}

			acfData := &detector.ACFData{
				AppID:         pctx.Game.AppID,
				Name:          pctx.Game.Name,
				InstallDir:    installDir,
				BuildID:       "0",
				LaunchOptions: launchOpts,
			}

			content := detector.GenerateACFContent(acfData)
			if pctx.Options.DryRun {
				slog.Info("[DRY RUN] Would synthesize 1:1 ACF manifest", "path", targetACF, "appID", pctx.Game.AppID)
			} else {
				if err := os.WriteFile(targetACF, []byte(content), 0644); err != nil {
					return fmt.Errorf("write ACF manifest: %w", err)
				}
				s.synthesizedACF = targetACF
				slog.Info("Synthesized 1:1 ACF manifest", "path", targetACF, "appID", pctx.Game.AppID)
			}
			pctx.Results.ACFSynthesized = true
		}
	}

	return nil
}

func (s *ManifestStage) Rollback(ctx context.Context, pctx *Context) error {
	if s.synthesizedACF != "" && util.FileExists(s.synthesizedACF) {
		slog.Info("Removing synthesized ACF manifest during rollback", "path", s.synthesizedACF)
		_ = os.Remove(s.synthesizedACF)
		pctx.Results.ACFSynthesized = false
	}
	return nil
}

// NormalizeStage normalizes the game directory name to match official Steam InstallDir.
type NormalizeStage struct {
	originalDir string
	renamedDir  string
}

func (s *NormalizeStage) Name() string {
	return "Normalize Game Directory"
}

func (s *NormalizeStage) Execute(ctx context.Context, pctx *Context) error {
	if !pctx.Options.NormalizeDir {
		return nil
	}

	// Non-Steam games (GOG, Standalone, Epic, etc.) must not have their directories renamed
	if pctx.Game.Store != domain.StoreSteam && pctx.Game.Source != domain.SourceSteam {
		return nil
	}

	currentDir := pctx.Game.GameDir
	baseName := filepath.Base(currentDir)
	cleanName := util.SanitizeInstallDir(pctx.Game.Name)

	if cleanName == "" || strings.EqualFold(baseName, cleanName) {
		return nil
	}

	parentDir := filepath.Dir(currentDir)
	targetDir := filepath.Join(parentDir, cleanName)

	if util.FileExists(targetDir) {
		slog.Warn("Target normalized directory already exists; skipping rename", "target", targetDir)
		return nil
	}

	if pctx.Options.DryRun {
		slog.Info("[DRY RUN] Would normalize game directory", "from", currentDir, "to", targetDir)
		pctx.Results.DirectoryNormalized = true
		return nil
	}

	if err := os.Rename(currentDir, targetDir); err != nil {
		return fmt.Errorf("rename directory from %s to %s: %w", currentDir, targetDir, err)
	}

	s.originalDir = currentDir
	s.renamedDir = targetDir

	slog.Info("Successfully normalized game directory name", "from", currentDir, "to", targetDir)
	pctx.Results.DirectoryNormalized = true
	pctx.Game.GameDir = targetDir
	if pctx.Game.ExePath != "" {
		relExe, err := filepath.Rel(currentDir, pctx.Game.ExePath)
		if err == nil && !strings.HasPrefix(relExe, "..") {
			pctx.Game.ExePath = filepath.Join(targetDir, relExe)
		}
	}

	// Re-base persisted launch profile paths if present
	if prof, err := compat.LoadProfile(targetDir); err == nil && prof != nil {
		prof.GameDir = targetDir
		if prof.ExePath != "" {
			if relProfExe, err := filepath.Rel(currentDir, prof.ExePath); err == nil && !strings.HasPrefix(relProfExe, "..") {
				prof.ExePath = filepath.Join(targetDir, relProfExe)
			}
		}
		_ = compat.SaveProfile(prof)
	}

	return nil
}

func (s *NormalizeStage) Rollback(ctx context.Context, pctx *Context) error {
	if s.originalDir == "" || s.renamedDir == "" {
		return nil
	}
	if !util.FileExists(s.renamedDir) {
		return nil
	}
	slog.Info("Rolling back game directory normalization", "from", s.renamedDir, "to", s.originalDir)
	if err := os.Rename(s.renamedDir, s.originalDir); err != nil {
		return fmt.Errorf("rollback rename directory from %s to %s: %w", s.renamedDir, s.originalDir, err)
	}
	pctx.Game.GameDir = s.originalDir
	if pctx.Game.ExePath != "" {
		relExe, err := filepath.Rel(s.renamedDir, pctx.Game.ExePath)
		if err == nil && !strings.HasPrefix(relExe, "..") {
			pctx.Game.ExePath = filepath.Join(s.originalDir, relExe)
		}
	}
	if prof, err := compat.LoadProfile(s.originalDir); err == nil && prof != nil {
		prof.GameDir = s.originalDir
		if prof.ExePath != "" {
			if relProfExe, err := filepath.Rel(s.renamedDir, prof.ExePath); err == nil && !strings.HasPrefix(relProfExe, "..") {
				prof.ExePath = filepath.Join(s.originalDir, relProfExe)
			}
		}
		_ = compat.SaveProfile(prof)
	}
	pctx.Results.DirectoryNormalized = false
	return nil
}

// SteamlessStage unpacks SteamStub DRM from PE executables.
type SteamlessStage struct{}

func (s *SteamlessStage) Name() string {
	return "Steamless DRM Unpacking"
}

func (s *SteamlessStage) Execute(ctx context.Context, pctx *Context) error {
	if pctx.Options.NoSteamless || !pctx.Game.IsWindows() {
		return nil
	}

	// Skip Steamless for non-Steam DRM-free games or games without SteamAppID
	if !pctx.Game.IsSteam() && !pctx.Game.RequiresSteamEmulator() {
		return nil
	}

	unpacked, err := steamless.UnpackExecutable(ctx, pctx.Config, pctx.Game.GameDir, pctx.Options.DryRun)
	if err != nil {
		slog.Warn("Steamless unpacking warning", "error", err)
		pctx.Results.Errors = append(pctx.Results.Errors, fmt.Sprintf("steamless: %v", err))
	}
	pctx.Results.SteamlessUnpacked = unpacked
	return nil
}

func (s *SteamlessStage) Rollback(ctx context.Context, pctx *Context) error {
	if pctx.Game == nil || pctx.Game.GameDir == "" || !pctx.Results.SteamlessUnpacked {
		return nil
	}
	slog.Info("Rolling back steamless unpacking", "dir", pctx.Game.GameDir)
	_ = filepath.WalkDir(pctx.Game.GameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.Contains(path, ".ORIGINAL") {
			idx := strings.Index(path, ".20")
			if idx != -1 {
				targetFile := path[:idx]
				if strings.HasSuffix(strings.ToLower(targetFile), ".exe") {
					if !pctx.Options.DryRun {
						if err := os.Rename(path, targetFile); err == nil {
							slog.Info("Restored original executable", "file", targetFile)
						}
					}
				}
			}
		}
		return nil
	})
	pctx.Results.SteamlessUnpacked = false
	return nil
}

// GBEStage injects Goldberg Steam Emulator libraries and configurations.
type GBEStage struct{}

func (s *GBEStage) Name() string {
	return "Goldberg Steam Emulator"
}

func (s *GBEStage) Execute(ctx context.Context, pctx *Context) error {
	opMode := pctx.Options.OperatingMode
	if opMode == "" {
		opMode = pctx.Game.EffectiveOperatingMode()
	}
	if opMode == "" {
		if pctx.Config != nil && pctx.Config.OperatingMode != "" {
			opMode = pctx.Config.OperatingMode
		} else {
			opMode = domain.ModeLoader
		}
	}

	// If game is set to ModeNone or does not require emulation, revert any old emulator artifacts and skip
	if !pctx.Game.RequiresSteamEmulator() || opMode == domain.ModeNone || opMode == domain.ModeStandalone {
		if pctx.Game.HasBackups() || pctx.Game.SettingsDirExists {
			_, _ = compat.Revert(ctx, pctx.Game.GameDir, pctx.Options.DryRun)
		}
		return nil
	}

	profile := pctx.Game.LaunchProfile
	if profile == nil {
		profile = &domain.LaunchProfile{
			GameDir:       pctx.Game.GameDir,
			AppID:         pctx.Game.AppID,
			Platform:      pctx.Game.Platform,
			OperatingMode: opMode,
			ExePath:       pctx.Game.EffectiveExePath(),
			WinePrefix:    pctx.Options.WinePrefix,
			Language:      pctx.Options.Language,
		}
	} else {
		profile.OperatingMode = opMode
		if pctx.Options.WinePrefix != "" {
			profile.WinePrefix = pctx.Options.WinePrefix
		}
		if pctx.Options.Language != "" {
			profile.Language = pctx.Options.Language
		}
	}

	err := compat.Apply(ctx, pctx.Config, pctx.Game, profile, pctx.Options.DryRun)
	if err != nil {
		slog.Warn("Compatibility & DRM application warning/error", "error", err)
		pctx.Results.Errors = append(pctx.Results.Errors, fmt.Sprintf("gbe patch: %v", err))
	} else {
		pctx.Results.Patched = true
	}

	if pctx.Options.FetchAchievements && pctx.Game.NumericAppID > 0 {
		apiKey := ""
		if pctx.Config != nil {
			apiKey = pctx.Config.SteamWebAPIKey
		}
		if achErr := gbe.GenerateAchievementsJSON(ctx, pctx.Game.NumericAppID, apiKey, pctx.Game.GameDir, pctx.Options.DryRun); achErr != nil {
			slog.Warn("Achievement schema generation warning", "error", achErr)
		}
	}

	return nil
}

func (s *GBEStage) Rollback(ctx context.Context, pctx *Context) error {
	_, err := compat.Revert(ctx, pctx.Game.GameDir, pctx.Options.DryRun)
	return err
}

// VerifyStage scans game files and verifies SHA1 checksums against official manifests.
type VerifyStage struct{}

func (s *VerifyStage) Name() string {
	return "Verify File Integrity"
}

func (s *VerifyStage) Execute(ctx context.Context, pctx *Context) error {
	if pctx.Game == nil || pctx.Game.NumericAppID == 0 {
		return fmt.Errorf("cannot verify game without numeric Steam AppID")
	}

	report, err := manifest.ScanDepotIntegrityWithPlatform(pctx.Game.GameDir, pctx.Game.NumericAppID, pctx.Game.Platform)
	if err != nil {
		return fmt.Errorf("integrity scan failed: %w", err)
	}

	pctx.Results.IntegrityReport = report
	if report != nil {
		slog.Info("Integrity verification completed", "game", pctx.Game.Name, "intact", report.IntactCount, "official", report.OfficialCount)
	}
	return nil
}

// UpdateStage downloads updated Steam depot chunks and manifests for a game.
type UpdateStage struct{}

func (s *UpdateStage) Name() string {
	return "Download Depot Updates"
}

func (s *UpdateStage) Execute(ctx context.Context, pctx *Context) error {
	if pctx.Game == nil || pctx.Game.NumericAppID == 0 {
		return fmt.Errorf("cannot update game without numeric Steam AppID")
	}

	dlOpts := downloader.DownloadOptions{
		AppID:     pctx.Game.NumericAppID,
		TargetDir: pctx.Game.GameDir,
		Platform:  pctx.Game.Platform,
		DryRun:    pctx.Options.DryRun,
	}

	res, err := downloader.DownloadGame(ctx, pctx.Config, dlOpts)
	if err != nil {
		return fmt.Errorf("depot update failed: %w", err)
	}

	if res != nil {
		pctx.Results.DownloadedBytes = int64(res.DownloadedBytes)
		slog.Info("Depot update download completed", "game", pctx.Game.Name, "bytes", res.DownloadedBytes)
	}
	return nil
}

// RevertDRMStage restores original backup binaries and cleans up emulator artifacts.
type RevertDRMStage struct{}

func (s *RevertDRMStage) Name() string {
	return "Revert DRM & Emulation"
}

func (s *RevertDRMStage) Execute(ctx context.Context, pctx *Context) error {
	targetDir := ""
	if pctx.Game != nil && pctx.Game.GameDir != "" {
		targetDir = pctx.Game.GameDir
	} else if pctx.Options.Path != "" {
		targetDir = pctx.Options.Path
	}
	targetDir = util.CanonicalPath(targetDir)
	if targetDir == "" {
		return fmt.Errorf("target directory cannot be empty")
	}

	restoredCount, err := gbe.Revert(targetDir, pctx.Options.DryRun)
	if err != nil {
		return fmt.Errorf("revert DRM failed: %w", err)
	}

	pctx.Results.RestoredCount = restoredCount
	pctx.Results.Patched = false
	slog.Info("Revert DRM completed", "gameDir", targetDir, "restoredFiles", restoredCount)
	return nil
}


