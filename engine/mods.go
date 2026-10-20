package engine

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
)

// GetGameMods retrieves the engine topology and list of installed mods for a given game.
func (eng *Engine) GetGameMods(ctx context.Context, gamePath string) (*domain.GameModStatus, error) {
	cleanPath := util.CanonicalPath(gamePath)
	if cleanPath == "" {
		return nil, fmt.Errorf("invalid game directory: %s", gamePath)
	}

	topo := detector.DetectModTopology(cleanPath)
	mods, outdatedCount := detector.ScanInstalledMods(cleanPath, topo)

	return &domain.GameModStatus{
		Topology:      topo,
		Mods:          mods,
		OutdatedCount: outdatedCount,
	}, nil
}

// ToggleGameMod enables or disables an installed mod by renaming with/without the .disabled suffix.
func (eng *Engine) ToggleGameMod(gamePath, modID string, enable bool) (*domain.InstalledMod, error) {
	cleanPath := util.CanonicalPath(gamePath)
	if cleanPath == "" {
		return nil, fmt.Errorf("invalid game directory: %s", gamePath)
	}

	topo := detector.DetectModTopology(cleanPath)
	mods, _ := detector.ScanInstalledMods(cleanPath, topo)

	var target *domain.InstalledMod
	for i := range mods {
		if mods[i].ID == modID || strings.EqualFold(mods[i].Name, modID) {
			target = &mods[i]
			break
		}
	}

	if target == nil {
		return nil, fmt.Errorf("mod %q not found in game directory", modID)
	}

	currentPath := target.Path
	if currentPath == "" {
		return nil, fmt.Errorf("mod path for %q is empty", modID)
	}

	var newPath string
	if enable {
		if strings.HasSuffix(strings.ToLower(currentPath), ".disabled") {
			newPath = strings.TrimSuffix(currentPath, ".disabled")
			newPath = strings.TrimSuffix(newPath, ".DISABLED")
		} else {
			// Already enabled
			return target, nil
		}
	} else {
		if !strings.HasSuffix(strings.ToLower(currentPath), ".disabled") {
			newPath = currentPath + ".disabled"
		} else {
			// Already disabled
			return target, nil
		}
	}

	if err := os.Rename(currentPath, newPath); err != nil {
		return nil, fmt.Errorf("toggle mod rename failed from %s to %s: %w", currentPath, newPath, err)
	}

	target.Path = newPath
	target.Enabled = enable
	rel, _ := filepath.Rel(cleanPath, newPath)
	target.RelativePath = rel

	// Update registry if recorded
	reg, _ := detector.LoadModRegistry(cleanPath)
	if reg != nil {
		for i := range reg.Mods {
			if reg.Mods[i].ID == target.ID || reg.Mods[i].WorkshopID == target.WorkshopID {
				reg.Mods[i].Enabled = enable
				reg.Mods[i].Path = newPath
				reg.Mods[i].RelativePath = rel
				_ = detector.SaveModRegistry(cleanPath, reg)
				break
			}
		}
	}

	cache.InvalidateGameCache(cleanPath)
	return target, nil
}

// DeleteGameMod deletes an installed mod from disk and removes its registry entry.
func (eng *Engine) DeleteGameMod(gamePath, modID string) error {
	cleanPath := util.CanonicalPath(gamePath)
	if cleanPath == "" {
		return fmt.Errorf("invalid game directory: %s", gamePath)
	}

	topo := detector.DetectModTopology(cleanPath)
	mods, _ := detector.ScanInstalledMods(cleanPath, topo)

	var target *domain.InstalledMod
	for i := range mods {
		if mods[i].ID == modID || strings.EqualFold(mods[i].Name, modID) {
			target = &mods[i]
			break
		}
	}

	if target == nil {
		return fmt.Errorf("mod %q not found", modID)
	}

	if target.Path != "" {
		_ = os.RemoveAll(target.Path)
		// Also remove .disabled variant if present
		if strings.HasSuffix(target.Path, ".disabled") {
			_ = os.RemoveAll(strings.TrimSuffix(target.Path, ".disabled"))
		} else {
			_ = os.RemoveAll(target.Path + ".disabled")
		}
	}

	// Remove from registry
	reg, _ := detector.LoadModRegistry(cleanPath)
	if reg != nil {
		filtered := make([]domain.InstalledMod, 0, len(reg.Mods))
		for _, m := range reg.Mods {
			if m.ID != target.ID && (target.WorkshopID == 0 || m.WorkshopID != target.WorkshopID) {
				filtered = append(filtered, m)
			}
		}
		reg.Mods = filtered
		_ = detector.SaveModRegistry(cleanPath, reg)
	}

	cache.InvalidateGameCache(cleanPath)
	return nil
}

// CheckGameModUpdates queries Steam Web API for newer timestamps on installed workshop items.
func (eng *Engine) CheckGameModUpdates(ctx context.Context, gamePath string) (*domain.GameModStatus, error) {
	cleanPath := util.CanonicalPath(gamePath)
	if cleanPath == "" {
		return nil, fmt.Errorf("invalid game directory: %s", gamePath)
	}

	topo := detector.DetectModTopology(cleanPath)
	mods, _ := detector.ScanInstalledMods(cleanPath, topo)

	var workshopIDs []uint64
	for _, m := range mods {
		if m.WorkshopID > 0 {
			workshopIDs = append(workshopIDs, m.WorkshopID)
		}
	}

	if len(workshopIDs) == 0 {
		return eng.GetGameMods(ctx, cleanPath)
	}

	detailsList, err := steam.FetchWorkshopItemsDetails(ctx, workshopIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch workshop details for updates: %w", err)
	}

	detailMap := make(map[uint64]steam.WorkshopItemDetails)
	for _, d := range detailsList {
		id, pErr := steam.ExtractWorkshopID(d.PublishedFileID)
		if pErr == nil {
			detailMap[id] = d
		}
	}

	reg, _ := detector.LoadModRegistry(cleanPath)
	if reg == nil {
		reg = &detector.ModRegistry{Mods: []domain.InstalledMod{}}
	}

	regMap := make(map[uint64]*domain.InstalledMod)
	for i := range reg.Mods {
		if reg.Mods[i].WorkshopID > 0 {
			regMap[reg.Mods[i].WorkshopID] = &reg.Mods[i]
		}
	}

	for _, m := range mods {
		if m.WorkshopID == 0 {
			continue
		}
		detail, found := detailMap[m.WorkshopID]
		if !found {
			continue
		}

		regMod, inReg := regMap[m.WorkshopID]
		if !inReg {
			newEntry := m
			reg.Mods = append(reg.Mods, newEntry)
			regMod = &reg.Mods[len(reg.Mods)-1]
			regMap[m.WorkshopID] = regMod
		}

		regMod.LatestTimeUpdated = detail.TimeUpdated
		if detail.TimeUpdated > regMod.TimeUpdated && regMod.TimeUpdated > 0 {
			regMod.HasUpdate = true
		} else {
			regMod.HasUpdate = false
		}
		if detail.Title != "" {
			regMod.Name = detail.Title
		}
		if detail.Description != "" {
			regMod.Description = detail.Description
		}
		if detail.PreviewURL != "" {
			regMod.PreviewURL = detail.PreviewURL
		}
	}

	_ = detector.SaveModRegistry(cleanPath, reg)
	cache.InvalidateGameCache(cleanPath)

	return eng.GetGameMods(ctx, cleanPath)
}

// InstallWorkshopMod downloads a Steam Workshop item and installs it into the game's detected mod directory.
func (eng *Engine) InstallWorkshopMod(ctx context.Context, gamePath, itemURLOrID string) (*domain.InstalledMod, error) {
	cleanPath := util.CanonicalPath(gamePath)
	if cleanPath == "" {
		return nil, fmt.Errorf("invalid game directory: %s", gamePath)
	}

	workshopID, err := steam.ExtractWorkshopID(itemURLOrID)
	if err != nil {
		return nil, err
	}

	slog.Info("Fetching Workshop details for mod installation", "workshopID", workshopID, "gameDir", cleanPath)
	details, err := steam.FetchWorkshopItemDetails(ctx, workshopID)
	if err != nil {
		return nil, fmt.Errorf("fetch workshop details: %w", err)
	}

	topo := detector.DetectModTopology(cleanPath)
	targetModDir := topo.AbsoluteModDir
	if err := os.MkdirAll(targetModDir, 0755); err != nil {
		return nil, fmt.Errorf("create mod target directory %s: %w", targetModDir, err)
	}

	// Destination folder or file for this mod
	modDirName := fmt.Sprintf("%d_%s", workshopID, util.SanitizeInstallDir(details.Title))
	if modDirName == fmt.Sprintf("%d_", workshopID) {
		modDirName = fmt.Sprintf("%d", workshopID)
	}
	destDir := filepath.Join(targetModDir, modDirName)

	var installedFiles []string

	if details.FileURL != "" {
		slog.Info("Downloading Workshop zip file", "fileURL", details.FileURL, "destDir", destDir)
		files, dErr := downloadAndExtractModZip(ctx, details.FileURL, destDir)
		if dErr != nil {
			return nil, dErr
		}
		installedFiles = files
	} else if details.HContentFile != "" {
		slog.Info("SteamPipe UGC manifest mod detected, attempting SteamCMD download", "ugcID", details.HContentFile)
		steamcmdPath, lErr := exec.LookPath("steamcmd")
		if lErr != nil {
			return nil, fmt.Errorf("workshop item %d (%s) uses SteamPipe UGC manifest storage; downloading requires 'steamcmd' to be installed on your system", workshopID, details.Title)
		}

		tmpDir, tErr := os.MkdirTemp("", "gamux-steamcmd-*")
		if tErr != nil {
			return nil, fmt.Errorf("create temp dir for steamcmd: %w", tErr)
		}
		defer os.RemoveAll(tmpDir)

		appID := details.ConsumerAppID
		if appID == 0 {
			if gameInfo, dErr := detector.DetectGame(ctx, cleanPath); dErr == nil && gameInfo.NumericAppID > 0 {
				appID = gameInfo.NumericAppID
			}
		}

		cmd := exec.CommandContext(ctx, steamcmdPath,
			"+login", "anonymous",
			"+workshop_download_item", fmt.Sprintf("%d", appID), fmt.Sprintf("%d", workshopID),
			"+quit",
		)
		cmd.Dir = tmpDir
		out, cErr := cmd.CombinedOutput()
		if cErr != nil {
			return nil, fmt.Errorf("steamcmd workshop download failed: %w (output: %s)", cErr, string(out))
		}

		// Find downloaded content
		sourceDir := filepath.Join(tmpDir, "steamapps", "workshop", "content", fmt.Sprintf("%d", appID), fmt.Sprintf("%d", workshopID))
		if _, sErr := os.Stat(sourceDir); sErr != nil {
			// Check user home steamcmd location
			home, _ := os.UserHomeDir()
			altSource := filepath.Join(home, ".local", "share", "Steam", "steamapps", "workshop", "content", fmt.Sprintf("%d", appID), fmt.Sprintf("%d", workshopID))
			if _, aErr := os.Stat(altSource); aErr == nil {
				sourceDir = altSource
			} else {
				return nil, fmt.Errorf("steamcmd finished but could not locate downloaded files for item %d: %w", workshopID, sErr)
			}
		}

		if err := util.CopyDir(sourceDir, destDir); err != nil {
			return nil, fmt.Errorf("copy downloaded mod files to %s: %w", destDir, err)
		}
	} else {
		return nil, fmt.Errorf("workshop item %d has neither direct download URL nor manifest UGC ID", workshopID)
	}

	relPath, _ := filepath.Rel(cleanPath, destDir)

	modItem := domain.InstalledMod{
		ID:                fmt.Sprintf("%d", workshopID),
		Name:              details.Title,
		WorkshopID:        workshopID,
		Enabled:           true,
		Path:              destDir,
		RelativePath:      relPath,
		TimeUpdated:       details.TimeUpdated,
		LatestTimeUpdated: details.TimeUpdated,
		HasUpdate:         false,
		Description:       details.Description,
		PreviewURL:        details.PreviewURL,
		Files:             installedFiles,
	}

	// Update registry
	reg, _ := detector.LoadModRegistry(cleanPath)
	if reg == nil {
		reg = &detector.ModRegistry{}
	}

	updated := false
	for i := range reg.Mods {
		if reg.Mods[i].WorkshopID == workshopID || reg.Mods[i].ID == modItem.ID {
			reg.Mods[i] = modItem
			updated = true
			break
		}
	}
	if !updated {
		reg.Mods = append(reg.Mods, modItem)
	}
	_ = detector.SaveModRegistry(cleanPath, reg)
	cache.InvalidateGameCache(cleanPath)

	return &modItem, nil
}

func downloadAndExtractModZip(ctx context.Context, downloadURL, outDir string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}

	resp, err := steam.HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return nil, fmt.Errorf("download zip: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download zip HTTP %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read zip body: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("open zip archive: %w", err)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("create output directory %s: %w", outDir, err)
	}

	var extractedFiles []string
	cleanOutDir := filepath.Clean(outDir)

	for _, file := range zipReader.File {
		targetPath := filepath.Join(outDir, filepath.Clean(file.Name))
		if !strings.HasPrefix(targetPath, cleanOutDir+string(os.PathSeparator)) && targetPath != cleanOutDir {
			continue // Prevent ZipSlip
		}

		if file.FileInfo().IsDir() {
			_ = os.MkdirAll(targetPath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return nil, err
		}

		rc, err := file.Open()
		if err != nil {
			return nil, err
		}

		outFile, err := os.Create(targetPath)
		if err != nil {
			rc.Close()
			return nil, err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("extract %s: %w", targetPath, err)
		}

		extractedFiles = append(extractedFiles, file.Name)
	}

	return extractedFiles, nil
}
