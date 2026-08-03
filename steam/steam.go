package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/staernid/gamux/config"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

// HTTPClient allows injecting custom HTTP clients or RoundTrippers for testing.
var HTTPClient = http.DefaultClient

// ReleaseDate holds release date metadata returned from the Steam Store API.
type ReleaseDate struct {
	ComingSoon bool   `json:"coming_soon"`
	Date       string `json:"date"`
}

func (r *ReleaseDate) UnmarshalJSON(b []byte) error {
	if string(b) == "null" || string(b) == `""` {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		r.Date = s
		return nil
	}
	type Alias ReleaseDate
	var raw Alias
	if err := json.Unmarshal(b, &raw); err == nil {
		*r = ReleaseDate(raw)
		return nil
	}
	return nil
}

// AppDetails holds metadata returned from the Steam Store API.
type AppDetails struct {
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	IsFree        bool           `json:"is_free"`
	HeaderImage   string         `json:"header_image"`
	RequiredAge   int            `json:"required_age"`
	About         string         `json:"about_the_game"`
	ShortDesc     string         `json:"short_description"`
	ReleaseDate   ReleaseDate    `json:"release_date"`
	Platforms     AppPlatforms   `json:"platforms"`
	PriceOverview *PriceOverview `json:"price_overview,omitempty"`
}

// AppPlatforms holds platform availability flags.
type AppPlatforms struct {
	Windows bool `json:"windows"`
	Mac     bool `json:"mac"`
	Linux   bool `json:"linux"`
}

// PriceOverview holds pricing information.
type PriceOverview struct {
	Currency        string `json:"currency"`
	Initial         int    `json:"initial"`
	Final           int    `json:"final"`
	DiscountPercent int    `json:"discount_percent"`
	FormattedPrices struct {
		Initial string `json:"initial"`
		Final   string `json:"final"`
	} `json:"formatted_prices,omitempty"`
}

// FetchAppDetails fetches full app details for a given AppID.
func FetchAppDetails(ctx context.Context, appID string) (*AppDetails, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/appdetails?appids=%s", config.SteamStoreAPI, appID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch app details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch app details: HTTP %d", resp.StatusCode)
	}

	var result map[string]struct {
		Success bool        `json:"success"`
		Data    *AppDetails `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	appData, ok := result[appID]
	if !ok || !appData.Success || appData.Data == nil {
		return nil, fmt.Errorf("app details not found or unavailable for AppID %s", appID)
	}

	return appData.Data, nil
}

// FetchAppName gets the app name for a Steam AppID.
// It wraps FetchAppDetails for backward compatibility.
func FetchAppName(ctx context.Context, appID string) (string, error) {
	details, err := FetchAppDetails(ctx, appID)
	if err != nil {
		return "", err
	}
	return details.Name, nil
}

// SearchResultItem represents an item in storesearch results.
type SearchResultItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// SearchStoreResponse represents the Steam Store search API response.
type SearchStoreResponse struct {
	Total int                `json:"total"`
	Items []SearchResultItem `json:"items"`
}

// SearchAppID searches the Steam Store API for a game title and returns the best matching AppID and canonical Name.
func SearchAppID(ctx context.Context, query string) (string, string, error) {
	if strings.TrimSpace(query) == "" {
		return "", "", fmt.Errorf("search query cannot be empty")
	}

	searchURL := fmt.Sprintf("%s/storesearch/?term=%s&l=english&cc=US", config.SteamStoreAPI, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("create search request: %w", err)
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("fetch store search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("store search HTTP %d", resp.StatusCode)
	}

	var res SearchStoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", fmt.Errorf("decode store search JSON: %w", err)
	}

	if res.Total == 0 || len(res.Items) == 0 {
		return "", "", fmt.Errorf("no Steam AppID found for query %q", query)
	}

	best := res.Items[0]
	return fmt.Sprintf("%d", best.ID), best.Name, nil
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
