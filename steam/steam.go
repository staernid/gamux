package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"gbe_fork_helper/config"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"golang.org/x/sync/errgroup"
)

// fetchAppName gets the app name for a Steam AppID.
func FetchAppName(ctx context.Context, appID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/appdetails?appids=%s&filters=basic", config.SteamStoreAPI, appID), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch app details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch app details: HTTP %d", resp.StatusCode)
	}

	var result map[string]struct {
		Data struct {
			Name string `json:"name"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode JSON: %w", err)
	}
	if appData, ok := result[appID]; ok {
		return appData.Data.Name, nil
	}
	return "", fmt.Errorf("app details not found for AppID %s", appID)
}

// FetchDLCs fetches DLCs for a given AppID.
func FetchDLCs(ctx context.Context, appID, libraryPath string, dryRun bool) error {
	slog.Info("Fetching DLCs", "appID", appID, "libraryPath", libraryPath)

	// Write steam_appid.txt
	appIDFilePath := filepath.Join(libraryPath, "steam_appid.txt")
	if dryRun {
		slog.Info("[DRY RUN] Would write steam_appid.txt", "path", appIDFilePath)
	} else if err := os.WriteFile(appIDFilePath, []byte(appID), 0644); err != nil {
		return fmt.Errorf("failed to write steam_appid.txt: %w", err)
	}
	if !dryRun {
		slog.Info("Wrote steam_appid.txt", "path", appIDFilePath)
	}

	// Prepare for configs.app.ini
	steamSettingsDir := filepath.Join(libraryPath, "steam_settings")
	if dryRun {
		slog.Info("[DRY RUN] Would create steam_settings directory", "path", steamSettingsDir)
	} else if err := os.MkdirAll(steamSettingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create steam_settings directory: %w", err)
	}
	configsAppIniPath := filepath.Join(steamSettingsDir, "configs.app.ini")

	// Fetch DLCs
	dlcURL := fmt.Sprintf("https://store.steampowered.com/dlc/%s/random/ajaxgetfilteredrecommendations/?query&count=10000", appID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dlcURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch DLCs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch DLCs: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	re := regexp.MustCompile(`data-ds-appid=\\"(\d+)`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	if len(matches) == 0 {
		slog.Warn("No DLCs found", "appID", appID)
		return nil
	}

	uniqueDLCs := make(map[string]struct{})
	for _, m := range matches {
		uniqueDLCs[m[1]] = struct{}{}
	}

	var mu sync.Mutex
	dlcNames := make(map[string]string)
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(10) // Limit concurrency to 10 workers

	for dlcID := range uniqueDLCs {
		id := dlcID
		g.Go(func() error {
			name, err := FetchAppName(gCtx, id)
			if err != nil {
				slog.Warn("Failed to get name for DLC", "dlcID", id, "error", err)
				return nil // Don't fail the whole process if one DLC fails
			}
			mu.Lock()
			dlcNames[id] = name
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to fetch DLC names: %w", err)
	}

	if dryRun {
		slog.Info("[DRY RUN] Would write DLC configuration", "path", configsAppIniPath, "count", len(dlcNames))
	} else {
		file, err := os.Create(configsAppIniPath)
		if err != nil {
			return fmt.Errorf("failed to create configs.app.ini: %w", err)
		}
		defer file.Close()

		fmt.Fprintln(file, "[app::dlcs]")
		fmt.Fprintln(file, "unlock_all=0")
		for id, name := range dlcNames {
			fmt.Fprintf(file, "%s=%s\n", id, name)
		}

		slog.Info("Wrote DLC configuration", "path", configsAppIniPath, "count", len(dlcNames))
	}

	return nil
}
