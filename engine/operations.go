package engine

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/downloader"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/tasks"
	"github.com/staernid/gamux/tools"
	"github.com/staernid/gamux/util"
)

// ApplyOperation configures and executes game setup (GBE emulation, launch profile, ACF synthesis).
type ApplyOperation struct {
	Engine   *Engine
	Options  ProcessOptions
	GameName string
	Result   *ProcessResult
}

func (op *ApplyOperation) TaskSpec() tasks.TaskSpec {
	title := fmt.Sprintf("Apply Setup (%s)", op.GameName)
	if op.GameName == "" {
		title = fmt.Sprintf("Apply Setup (%s)", op.Options.Path)
	}
	return tasks.TaskSpec{
		Type:         tasks.TypeApply,
		Title:        title,
		GameName:     op.GameName,
		GameDir:      op.Options.Path,
		Capabilities: tasks.CapCancellable,
	}
}

func (op *ApplyOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	ctrl.ReportStage("Configuring Game", fmt.Sprintf("Applying GBE and launcher profile for %s...", op.GameName))
	op.Options.Reporter = ctrl
	res, err := op.Engine.ProcessGame(ctx, op.Options)
	op.Result = res
	if err == nil && res != nil {
		if res.Info != nil {
			ctrl.SetResultGame(res.Info)
		}
		ctrl.ReportStage("Completed", fmt.Sprintf("Setup applied successfully for %s", op.GameName))
		ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
	}
	return err
}

// UpdateOperation queries and downloads Steam depot updates for an installed game.
type UpdateOperation struct {
	Engine   *Engine
	GamePath string
	GameName string
}

func (op *UpdateOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         tasks.TypeUpdate,
		Title:        fmt.Sprintf("Update %s", op.GameName),
		GameName:     op.GameName,
		GameDir:      op.GamePath,
		Capabilities: tasks.CapCancellable,
	}
}

func (op *UpdateOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	ctrl.ReportStage("Checking Updates", fmt.Sprintf("Checking Steam depot changes for %s...", op.GameName))
	res, err := op.Engine.UpdateGame(ctx, op.GamePath)
	if err == nil && res != nil {
		ctrl.SetResultGame(res)
		ctrl.ReportStage("Completed", fmt.Sprintf("Update completed for %s", op.GameName))
		ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
	}
	return err
}

// VerifyOperation verifies game file hashes against official ACF manifests.
type VerifyOperation struct {
	Engine   *Engine
	GamePath string
	GameName string
}

func (op *VerifyOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         tasks.TypeVerify,
		Title:        fmt.Sprintf("Verify Integrity (%s)", op.GameName),
		GameName:     op.GameName,
		GameDir:      op.GamePath,
		Capabilities: tasks.CapCancellable,
	}
}

func (op *VerifyOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	ctrl.ReportStage("Verifying Integrity", fmt.Sprintf("Scanning files in %s...", op.GameName))
	rep, err := op.Engine.VerifyIntegrity(ctx, op.GamePath)
	if err == nil && rep != nil {
		ctrl.ReportProgress(int64(rep.IntactCount), int64(rep.OfficialCount), rep.IntactCount, rep.OfficialCount, 0, 0, 100)
		ctrl.ReportStage("Completed", fmt.Sprintf("%d/%d files intact", rep.IntactCount, rep.OfficialCount))
	}
	return err
}

// NormalizeOperation standardizes a game folder name to match official Steam InstallDir.
type NormalizeOperation struct {
	Engine   *Engine
	GamePath string
	GameName string
}

func (op *NormalizeOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         tasks.TypeNormalize,
		Title:        fmt.Sprintf("Normalize Directory (%s)", op.GameName),
		GameName:     op.GameName,
		GameDir:      op.GamePath,
		Capabilities: tasks.CapCancellable,
	}
}

func (op *NormalizeOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	ctrl.ReportStage("Normalizing Directory", fmt.Sprintf("Standardizing directory naming for %s...", op.GameName))
	opts := ProcessOptions{
		Path:         op.GamePath,
		NormalizeDir: true,
		ApplyGBE:     false,
		NoSteamless:  true,
		AutoYes:      true,
		Reporter:     ctrl,
	}
	res, err := op.Engine.ProcessGame(ctx, opts)
	if err == nil && res != nil && res.Info != nil && res.Info.GameDir != "" {
		ctrl.SetResultGame(res.Info)
		ctrl.ReportStage("Completed", fmt.Sprintf("Directory normalized to %s", res.Info.GameDir))
		ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
	}
	return err
}

// RollbackOperation restores original game backups and removes Goldberg emulator artifacts.
type RollbackOperation struct {
	Engine   *Engine
	GamePath string
	GameName string
	DryRun   bool
}

func (op *RollbackOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         tasks.TypeRollback,
		Title:        fmt.Sprintf("Rollback %s", op.GameName),
		GameName:     op.GameName,
		GameDir:      op.GamePath,
		Capabilities: tasks.CapCancellable,
	}
}

func (op *RollbackOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	ctrl.ReportStage("Restoring Backups", fmt.Sprintf("Reverting patched binaries for %s...", op.GameName))
	err := op.Engine.Rollback(ctx, op.GamePath, op.DryRun)
	if err == nil {
		ctrl.ReportStage("Completed", fmt.Sprintf("Rollback completed for %s", op.GameName))
		ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
	}
	return err
}

// DownloadOperation downloads game depots with progress streaming, throttling, pause/resume, and auto-setup.
type DownloadOperation struct {
	Engine   *Engine
	Options  DownloadGameOptions
	GameName string
	Result   *downloader.Result
}

func (op *DownloadOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         tasks.TypeDownload,
		Title:        fmt.Sprintf("Download %s", op.GameName),
		GameName:     op.GameName,
		GameDir:      op.Options.TargetDir,
		AppID:        fmt.Sprintf("%d", op.Options.AppID),
		Capabilities: tasks.CapCancellable | tasks.CapPausable | tasks.CapPersist | tasks.CapBandwidthLimit,
	}
}

func (op *DownloadOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	wrappedCtx := op.Engine.WrapContext(ctx)
	ctrl.ReportStage("Resolving Keys & Manifests", "Contacting depot database...")

	var lastCurrItems, lastTotalItems int
	origCallback := op.Options.ProgressCallback
	op.Options.ProgressCallback = func(curr, total int, item string) {
		lastCurrItems = curr
		lastTotalItems = total
		if origCallback != nil {
			origCallback(curr, total, item)
		}
	}

	op.Options.ByteProgressCallback = func(currBytes, totalBytes int64, speed float64, eta int, pct float64, item string) {
		if err := ctrl.CheckPauseOrCancel(); err != nil {
			return
		}
		_ = ctrl.Throttle(int(speed / 10))
		ctrl.ReportProgress(currBytes, totalBytes, lastCurrItems, lastTotalItems, speed, eta, pct)
		ctrl.ReportStage("Downloading Depots", item)
	}

	res, err := op.Engine.DownloadGame(wrappedCtx, op.Options)
	op.Result = res
	if err == nil && op.Options.AutoApply && op.Engine != nil {
		if status, inspectErr := op.Engine.InspectStatus(ctx, op.Options.TargetDir); inspectErr == nil && status != nil {
			ctrl.SetResultGame(status)
		}
	}
	return err
}

// ToolUpdateOperation updates auxiliary utilities (GBE, Steamless, UMU).
type ToolUpdateOperation struct {
	Engine      *Engine
	ToolKey     string
	DisplayName string
	TargetDir   string
	Tool        tools.Tool
}

func (op *ToolUpdateOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         tasks.TypeToolUpdate,
		Title:        fmt.Sprintf("Update %s", op.DisplayName),
		GameName:     op.DisplayName,
		GameDir:      op.TargetDir,
		Capabilities: tasks.CapCancellable | tasks.CapPausable | tasks.CapBandwidthLimit,
		Metadata:     map[string]string{"tool_key": op.ToolKey},
	}
}

func (op *ToolUpdateOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	wrappedCtx := op.Engine.WrapContext(ctx)
	ctrl.ReportStage("Checking GitHub Releases", fmt.Sprintf("Querying latest release for %s...", op.Tool.Name()))

	if err := ctrl.CheckPauseOrCancel(); err != nil {
		return err
	}

	ctrl.ReportStage("Downloading Tool Release", fmt.Sprintf("Fetching %s assets to %s", op.Tool.Name(), op.TargetDir))
	if err := op.Tool.Update(wrappedCtx); err != nil {
		return fmt.Errorf("update %s: %w", op.Tool.Name(), err)
	}

	ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
	ctrl.ReportStage("Completed", fmt.Sprintf("%s updated successfully", op.Tool.Name()))
	return nil
}

// WorkshopOperation downloads a Steam Workshop item or mod.
type WorkshopOperation struct {
	ItemURLOrID string
	TargetDir   string
	DryRun      bool
}

func (op *WorkshopOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         tasks.TypeWorkshop,
		Title:        fmt.Sprintf("Download Workshop Item (%s)", op.ItemURLOrID),
		GameDir:      op.TargetDir,
		Capabilities: tasks.CapCancellable,
	}
}

func (op *WorkshopOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	ctrl.ReportStage("Fetching Workshop Details", op.ItemURLOrID)
	return steam.DownloadWorkshopItem(ctx, op.ItemURLOrID, op.TargetDir, op.DryRun)
}

// BatchAchievementsOperation scans library games and generates missing achievement schemas and icons.
type BatchAchievementsOperation struct {
	Engine      *Engine
	LibraryDirs []string
}

func (op *BatchAchievementsOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         tasks.TypeCustom,
		Title:        "Batch Download Missing Achievements",
		Capabilities: tasks.CapCancellable,
	}
}

func (op *BatchAchievementsOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	ctrl.ReportStage("Scanning Library", "Scanning games for missing achievement data...")
	games, err := op.Engine.BatchInspectLibraries(ctx, op.LibraryDirs)
	if err != nil {
		return err
	}

	apiKey := ""
	if op.Engine.Config != nil {
		apiKey = op.Engine.Config.SteamWebAPIKey
	}

	var candidates []*domain.Game
	for _, g := range games {
		if g == nil || g.AppID == "" || g.AppID == "0" || strings.EqualFold(g.AppID, "custom") {
			continue
		}
		if appIDNum, ok := util.ParseAppID(g.AppID); ok && appIDNum > 0 {
			candidates = append(candidates, g)
		}
	}

	total := len(candidates)
	if total == 0 {
		ctrl.ReportStage("Completed", "No Steam games found in library")
		ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
		return nil
	}

	processedCount := 0
	successCount := 0

	for i, g := range candidates {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		appIDNum, _ := util.ParseAppID(g.AppID)
		achFile := filepath.Join(g.GameDir, "steam_settings", "achievements.json")
		imgDir := filepath.Join(g.GameDir, "steam_settings", "img")

		needsFetch := !util.FileExists(achFile) || !util.FileExists(imgDir)

		ctrl.ReportStage(
			fmt.Sprintf("[%d/%d] Checking %s", i+1, total, g.Name),
			fmt.Sprintf("Checking achievements for AppID %d (%s)...", appIDNum, g.Name),
		)
		pct := float64(i) / float64(total) * 100
		ctrl.ReportProgress(int64(i), int64(total), i, total, 0, 0, pct)

		if needsFetch {
			ctrl.ReportStage(
				fmt.Sprintf("[%d/%d] Fetching %s", i+1, total, g.Name),
				fmt.Sprintf("Downloading achievements schema and artwork for %s (AppID %d)...", g.Name, appIDNum),
			)
			if err := gbe.GenerateAchievementsJSON(ctx, appIDNum, apiKey, g.GameDir, false); err == nil {
				successCount++
			} else {
				slog.Warn("Failed to generate achievements for game", "game", g.Name, "appID", appIDNum, "error", err)
			}
		}

		processedCount++
	}

	ctrl.ReportProgress(int64(total), int64(total), total, total, 0, 0, 100)
	ctrl.ReportStage("Completed", fmt.Sprintf("Processed %d games (downloaded data for %d)", processedCount, successCount))
	return nil
}

// BatchArtworkOperation queries and caches missing covers, heroes, and banners across all library games.
type BatchArtworkOperation struct {
	Engine      *Engine
	LibraryDirs []string
}

func (op *BatchArtworkOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         tasks.TypeCustom,
		Title:        "Batch Refresh Missing Artwork",
		Capabilities: tasks.CapCancellable,
	}
}

func (op *BatchArtworkOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	ctrl.ReportStage("Scanning Library", "Scanning games for missing poster, hero, and banner artwork...")
	games, err := op.Engine.BatchInspectLibraries(ctx, op.LibraryDirs)
	if err != nil {
		return err
	}

	total := len(games)
	if total == 0 {
		ctrl.ReportStage("Completed", "No games in library")
		ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
		return nil
	}

	for i, g := range games {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if g == nil {
			continue
		}

		pct := float64(i) / float64(total) * 100
		ctrl.ReportProgress(int64(i), int64(total), i, total, 0, 0, pct)
		ctrl.ReportStage(
			fmt.Sprintf("[%d/%d] %s", i+1, total, g.Name),
			fmt.Sprintf("Resolving artwork for %s...", g.Name),
		)

		_, _ = op.Engine.ResolveArtwork(ctx, g)
	}

	ctrl.ReportProgress(int64(total), int64(total), total, total, 0, 0, 100)
	ctrl.ReportStage("Completed", fmt.Sprintf("Artwork refresh completed for %d games", total))
	return nil
}

// BatchNormalizeOperation normalizes folder names across all library games.
type BatchNormalizeOperation struct {
	Engine      *Engine
	LibraryDirs []string
}

func (op *BatchNormalizeOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         tasks.TypeNormalize,
		Title:        "Batch Normalize Directory Names",
		Capabilities: tasks.CapCancellable,
	}
}

func (op *BatchNormalizeOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	ctrl.ReportStage("Scanning Library", "Scanning game directories for standard naming...")
	games, err := op.Engine.BatchInspectLibraries(ctx, op.LibraryDirs)
	if err != nil {
		return err
	}

	total := len(games)
	if total == 0 {
		ctrl.ReportStage("Completed", "No games in library")
		ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
		return nil
	}

	normalizedCount := 0
	for i, g := range games {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if g == nil {
			continue
		}

		pct := float64(i) / float64(total) * 100
		ctrl.ReportProgress(int64(i), int64(total), i, total, 0, 0, pct)
		ctrl.ReportStage(
			fmt.Sprintf("[%d/%d] %s", i+1, total, g.Name),
			fmt.Sprintf("Normalizing %s...", g.Name),
		)

		opts := ProcessOptions{
			Path:         g.GameDir,
			NormalizeDir: true,
			ApplyGBE:     false,
			NoSteamless:  true,
			AutoYes:      true,
			Reporter:     ctrl,
		}
		res, err := op.Engine.ProcessGame(ctx, opts)
		if err == nil && res != nil && res.Info != nil && res.Info.GameDir != g.GameDir {
			normalizedCount++
		}
	}

	ctrl.ReportProgress(int64(total), int64(total), total, total, 0, 0, 100)
	ctrl.ReportStage("Completed", fmt.Sprintf("Normalization finished for %d games (%d directories renamed)", total, normalizedCount))
	return nil
}

// BatchVerifyOperation verifies depot integrity across all library games with ACF manifests.
type BatchVerifyOperation struct {
	Engine      *Engine
	LibraryDirs []string
}

func (op *BatchVerifyOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         tasks.TypeVerify,
		Title:        "Batch Verify Depot Integrity",
		Capabilities: tasks.CapCancellable,
	}
}

func (op *BatchVerifyOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	ctrl.ReportStage("Scanning Library", "Scanning games for depot manifests...")
	games, err := op.Engine.BatchInspectLibraries(ctx, op.LibraryDirs)
	if err != nil {
		return err
	}

	total := len(games)
	if total == 0 {
		ctrl.ReportStage("Completed", "No games in library")
		ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
		return nil
	}

	verifiedCount := 0
	for i, g := range games {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if g == nil {
			continue
		}

		pct := float64(i) / float64(total) * 100
		ctrl.ReportProgress(int64(i), int64(total), i, total, 0, 0, pct)
		ctrl.ReportStage(
			fmt.Sprintf("[%d/%d] %s", i+1, total, g.Name),
			fmt.Sprintf("Verifying integrity of %s...", g.Name),
		)

		rep, err := op.Engine.VerifyIntegrity(ctx, g.GameDir)
		if err == nil && rep != nil && rep.OfficialCount > 0 {
			verifiedCount++
		}
	}

	ctrl.ReportProgress(int64(total), int64(total), total, total, 0, 0, 100)
	ctrl.ReportStage("Completed", fmt.Sprintf("Integrity check finished for %d games (%d verified)", total, verifiedCount))
	return nil
}
