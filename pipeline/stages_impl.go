package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/gamecfg"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/lutris"
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
		targetPath = "."
	}

	if !pctx.DryRun {
		if count, decErr := util.DecompressVSZaInDir(targetPath); decErr == nil && count > 0 {
			slog.Info("Decompressed raw VSZa Steam depot files", "target", targetPath, "count", count)
		}
	}

	info, err := detector.Detect(ctx, targetPath)
	if err != nil {
		return fmt.Errorf("auto-detect failed for %s: %w", targetPath, err)
	}

	pctx.Game.Name = info.Name
	pctx.Game.AppID = info.AppID
	numID, _ := util.ParseAppID(info.AppID)
	pctx.Game.NumericAppID = numID
	pctx.Game.GameDir = info.GameDir
	pctx.Game.Platform = info.Platform
	pctx.Game.ExePath = info.ExePath
	pctx.Game.ExeArgs = info.ExeArgs
	pctx.Game.Store = info.Store

	if pctx.ExePath != "" {
		if filepath.IsAbs(pctx.ExePath) {
			pctx.Game.ExePath = pctx.ExePath
		} else {
			pctx.Game.ExePath = filepath.Join(pctx.Game.GameDir, pctx.ExePath)
		}
	}
	if pctx.ExeArgs != "" {
		pctx.Game.ExeArgs = pctx.ExeArgs
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
type ManifestStage struct{}

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
			if keyErr == nil && len(parsedLua.ManifestFiles) > 0 && !pctx.DryRun {
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
			if pctx.DryRun {
				slog.Info("[DRY RUN] Would synthesize 1:1 ACF manifest", "path", targetACF, "appID", pctx.Game.AppID)
			} else {
				if err := os.WriteFile(targetACF, []byte(content), 0644); err != nil {
					return fmt.Errorf("write ACF manifest: %w", err)
				}
				slog.Info("Synthesized 1:1 ACF manifest", "path", targetACF, "appID", pctx.Game.AppID)
			}
			pctx.ACFSynthesized = true
		}
	}

	return nil
}

// NormalizeStage normalizes the game directory name to match official Steam InstallDir.
type NormalizeStage struct{}

func (s *NormalizeStage) Name() string {
	return "Normalize Game Directory"
}

func (s *NormalizeStage) Execute(ctx context.Context, pctx *Context) error {
	if !pctx.NormalizeDir {
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

	if pctx.DryRun {
		slog.Info("[DRY RUN] Would normalize game directory", "from", currentDir, "to", targetDir)
		pctx.DirectoryNormalized = true
		return nil
	}

	if err := os.Rename(currentDir, targetDir); err != nil {
		return fmt.Errorf("rename directory from %s to %s: %w", currentDir, targetDir, err)
	}

	slog.Info("Successfully normalized game directory name", "from", currentDir, "to", targetDir)
	pctx.DirectoryNormalized = true
	pctx.Game.GameDir = targetDir
	if pctx.Game.ExePath != "" {
		relExe, err := filepath.Rel(currentDir, pctx.Game.ExePath)
		if err == nil && !strings.HasPrefix(relExe, "..") {
			pctx.Game.ExePath = filepath.Join(targetDir, relExe)
		}
	}

	// Re-base persisted launch profile paths if present
	if prof, err := gamecfg.Load(targetDir); err == nil && prof != nil {
		prof.GameDir = targetDir
		if prof.ExePath != "" {
			if relProfExe, err := filepath.Rel(currentDir, prof.ExePath); err == nil && !strings.HasPrefix(relProfExe, "..") {
				prof.ExePath = filepath.Join(targetDir, relProfExe)
			}
		}
		_ = gamecfg.Save(prof)
	}

	return nil
}

// SteamlessStage unpacks SteamStub DRM from PE executables.
type SteamlessStage struct{}

func (s *SteamlessStage) Name() string {
	return "Steamless DRM Unpacking"
}

func (s *SteamlessStage) Execute(ctx context.Context, pctx *Context) error {
	if pctx.NoSteamless || pctx.Game.Platform != "windows" {
		return nil
	}

	// Skip Steamless for non-Steam DRM-free games or games without SteamAppID
	if !pctx.Game.IsSteam() && !pctx.Game.RequiresSteamEmulator() {
		return nil
	}

	unpacked, err := steamless.UnpackExecutable(ctx, pctx.Config, pctx.Game.GameDir, pctx.DryRun)
	if err != nil {
		slog.Warn("Steamless unpacking warning", "error", err)
		pctx.Errors = append(pctx.Errors, fmt.Sprintf("steamless: %v", err))
	}
	pctx.SteamlessUnpacked = unpacked
	return nil
}

// GBEStage injects Goldberg Steam Emulator libraries and configurations.
type GBEStage struct{}

func (s *GBEStage) Name() string {
	return "Goldberg Steam Emulator"
}

func (s *GBEStage) Execute(ctx context.Context, pctx *Context) error {
	if !pctx.ApplyGBE {
		return nil
	}

	// Skip GBE if game does not require Steamworks emulation
	if !pctx.Game.RequiresSteamEmulator() {
		return nil
	}

	gbeCfg := pctx.Config
	if pctx.Language != "" {
		cloned := *pctx.Config
		cloned.Language = pctx.Language
		gbeCfg = &cloned
	}

	err := gbe.ApplyGBE(ctx, gbeCfg, pctx.Game.GameDir, pctx.Game.Platform, pctx.Game.AppID, pctx.DryRun, pctx.Portable, pctx.WinePrefix, pctx.Game.ExePath)
	if err != nil {
		slog.Warn("GBE application warning/error", "error", err)
		pctx.Errors = append(pctx.Errors, fmt.Sprintf("gbe patch: %v", err))
	} else {
		pctx.Patched = true
	}

	if pctx.FetchAchievements && pctx.Game.NumericAppID > 0 {
		apiKey := ""
		if pctx.Config != nil {
			apiKey = pctx.Config.SteamWebAPIKey
		}
		if achErr := gbe.GenerateAchievementsJSON(ctx, pctx.Game.NumericAppID, apiKey, pctx.Game.GameDir, pctx.DryRun); achErr != nil {
			slog.Warn("Achievement schema generation warning", "error", achErr)
		}
	}

	return nil
}

func (s *GBEStage) Rollback(ctx context.Context, pctx *Context) error {
	// Restore any .ORIGINAL backups
	return filepath.Walk(pctx.Game.GameDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".ORIGINAL") {
			target := strings.TrimSuffix(path, ".ORIGINAL")
			slog.Info("Restoring original file from backup during rollback", "target", target)
			_ = os.Rename(path, target)
		}
		return nil
	})
}

// LutrisStage registers game in Lutris database and writes YAML configuration.
type LutrisStage struct{}

func (s *LutrisStage) Name() string {
	return "Lutris Integration"
}

func (s *LutrisStage) Execute(ctx context.Context, pctx *Context) error {
	if !pctx.AddLutris {
		return nil
	}

	if strings.TrimSpace(pctx.Game.ExePath) == "" {
		return fmt.Errorf("%w: no executable detected in %s", detector.ErrNoExecutableFound, pctx.Game.GameDir)
	}

	runnerName := pctx.Runner
	if runnerName == "" {
		if pctx.Game.Platform == "linux" {
			runnerName = "linux"
		} else {
			runnerName = "wine"
		}
	}

	gamePath := pctx.Game.ExePath
	env := make(map[string]string)

	if pctx.Game.Platform != "linux" {
		loaderX64 := filepath.Join(pctx.Game.GameDir, "steamclient_loader_x64.exe")
		loaderX86 := filepath.Join(pctx.Game.GameDir, "steamclient_loader_x86.exe")
		coldIni := filepath.Join(pctx.Game.GameDir, "ColdClientLoader.ini")

		if util.FileExists(coldIni) || (!pctx.Portable && pctx.ApplyGBE) {
			if pctx.Game.Platform == "win32" && util.FileExists(loaderX86) {
				gamePath = loaderX86
				env["PROTON_DISABLE_LSTEAMCLIENT"] = "1"
			} else if util.FileExists(loaderX64) {
				gamePath = loaderX64
				env["PROTON_DISABLE_LSTEAMCLIENT"] = "1"
			}
		}
	}

	var sysConfig *lutris.SystemConfig
	if pctx.Config != nil && pctx.Config.EnableLaunchNotify {
		if gamuxBin, err := os.Executable(); err == nil {
			sysConfig = &lutris.SystemConfig{
				PreLaunchScript: gamuxBin,
				PreLaunchArg:    fmt.Sprintf("notify-launch --path %q", pctx.Game.GameDir),
				PreLaunchWait:   true,
			}
		}
	}

	lcfg := lutris.Config{
		Name:                  pctx.Game.Name,
		GamePath:              gamePath,
		Args:                  pctx.Game.ExeArgs,
		Runner:                runnerName,
		PrefixPath:            pctx.WinePrefix,
		Env:                   env,
		System:                sysConfig,
		CreateMenuShortcut:    false,
		CreateDesktopShortcut: false,
	}

	if pctx.DryRun {
		slog.Info("[DRY RUN] Would write Lutris config & register in database", "name", pctx.Game.Name)
		pctx.LutrisAdded = true
		return nil
	}

	targetDir := ""
	if pctx.Config != nil {
		targetDir = util.ExpandPath(pctx.Config.LutrisDir)
	}

	if err := lutris.Write(lcfg, targetDir); err != nil {
		return fmt.Errorf("write Lutris config: %w", err)
	}

	_, _ = steam.FetchLutrisArtwork(ctx, pctx.Game.AppID, lutris.Slugify(pctx.Game.Name), false)
	pctx.LutrisAdded = true
	return nil
}
