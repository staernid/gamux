package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/ini"
	"github.com/staernid/gamux/util"
	"golang.org/x/sync/errgroup"
)

// Default Steam API endpoints
const (
	DefaultSteamStoreAPI = "https://store.steampowered.com/api"
	DefaultSteamWebAPI   = "https://api.steampowered.com"
)

// DefaultHTTPClient is the default http.Client used for Steam API operations.
var DefaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

// HTTPClient allows injecting custom HTTP clients or RoundTrippers for testing.
var HTTPClient = DefaultHTTPClient

type contextKey string

const httpClientKey contextKey = "steam_http_client"

// WithHTTPClient returns a new context with the provided http.Client attached.
func WithHTTPClient(ctx context.Context, client *http.Client) context.Context {
	if client == nil {
		return ctx
	}
	return context.WithValue(ctx, httpClientKey, client)
}

// HTTPClientFromContext extracts the http.Client from context or returns DefaultHTTPClient.
func HTTPClientFromContext(ctx context.Context) *http.Client {
	if ctx != nil {
		if client, ok := ctx.Value(httpClientKey).(*http.Client); ok && client != nil {
			return client
		}
	}
	return DefaultHTTPClient
}

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

// AppGenre holds Steam store genre tags.
type AppGenre struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// AppDetails holds metadata returned from the Steam Store API.
type AppDetails struct {
	SteamAppID         uint32         `json:"steam_appid,omitempty"`
	Name               string         `json:"name"`
	Type               string         `json:"type"`
	IsFree             bool           `json:"is_free"`
	HeaderImage        string         `json:"header_image"`
	RequiredAge        int            `json:"required_age"`
	About              string         `json:"about_the_game"`
	ShortDesc          string         `json:"short_description"`
	DetailedDesc       string         `json:"detailed_description"`
	Developers         []string       `json:"developers"`
	Publishers         []string       `json:"publishers"`
	Genres             []AppGenre     `json:"genres"`
	SupportedLanguages string         `json:"supported_languages"`
	ReleaseDate        ReleaseDate    `json:"release_date"`
	Platforms          AppPlatforms   `json:"platforms"`
	PriceOverview      *PriceOverview `json:"price_overview,omitempty"`
	DLC                []uint32       `json:"dlc,omitempty"`
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

var languageStringToCode = map[string]string{
	"english":                 "english",
	"french":                  "french",
	"italian":                 "italian",
	"german":                  "german",
	"spanish":                 "spanish",
	"spanish - spain":         "spanish",
	"spanish - latin america": "latam",
	"arabic":                  "arabic",
	"czech":                   "czech",
	"danish":                  "danish",
	"dutch":                   "dutch",
	"finnish":                 "finnish",
	"greek":                   "greek",
	"hungarian":               "hungarian",
	"japanese":                "japanese",
	"korean":                  "koreana",
	"norwegian":               "norwegian",
	"polish":                  "polish",
	"portuguese":              "portuguese",
	"portuguese - brazil":     "brazilian",
	"portuguese - portugal":   "portuguese",
	"russian":                 "russian",
	"simplified chinese":      "schinese",
	"traditional chinese":     "tchinese",
	"swedish":                 "swedish",
	"thai":                    "thai",
	"turkish":                 "turkish",
	"ukrainian":               "ukrainian",
	"vietnamese":              "vietnamese",
}

// GetSupportedLanguageCodes parses Steam's HTML-formatted SupportedLanguages string into canonical Steam language identifiers.
func (a *AppDetails) GetSupportedLanguageCodes() []string {
	if a == nil || strings.TrimSpace(a.SupportedLanguages) == "" {
		return []string{"english"}
	}

	clean := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(a.SupportedLanguages, "")
	clean = strings.ReplaceAll(clean, "*", "")
	clean = strings.Split(clean, "languages with full audio support")[0]

	parts := strings.Split(clean, ",")
	var result []string
	seen := make(map[string]bool)

	for _, p := range parts {
		normalized := strings.ToLower(strings.TrimSpace(p))
		if normalized == "" {
			continue
		}
		code, ok := languageStringToCode[normalized]
		if !ok {
			code = normalized
		}
		if !seen[code] {
			seen[code] = true
			result = append(result, code)
		}
	}

	if len(result) == 0 {
		return []string{"english"}
	}
	return result
}

// FetchAppDetails fetches full app details for a given AppID using a tiered fallback strategy with deep caching.
// Tier 1: Query with explicit US region, English language, and mature content cookies (bypasses age-gate & NSFW blocks).
// Tier 2: Fallback query without region/language parameters for geo-IP / region-exclusive store listings.
func FetchAppDetails(ctx context.Context, appID string) (*AppDetails, error) {
	if appID == "" || appID == "0" {
		return nil, fmt.Errorf("invalid AppID %q", appID)
	}

	// 0. Check local disk & memory cache first
	if cachedData, ok := cache.GetAppDetails(appID); ok && len(cachedData) > 0 {
		var details AppDetails
		if err := json.Unmarshal(cachedData, &details); err == nil && details.Name != "" {
			return &details, nil
		}
	}

	doQuery := func(apiURL string) (*AppDetails, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		// Include User-Agent and mature content cookies so Steam doesn't block age-gated/NSFW games with success: false
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		req.Header.Set("Cookie", "wants_mature_content=1; birthtime=0; lastagecheckage=1-January-1990")

		resp, err := HTTPClientFromContext(ctx).Do(req)
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

		var appData struct {
			Success bool        `json:"success"`
			Data    *AppDetails `json:"data"`
		}
		found := false
		if entry, ok := result[appID]; ok && entry.Success && entry.Data != nil {
			appData = entry
			found = true
		} else {
			// Steam store API sometimes keys the response under a redirected AppID (e.g. demo, package, or parent app)
			targetNum, _ := strconv.ParseUint(appID, 10, 32)
			for _, entry := range result {
				if entry.Success && entry.Data != nil {
					// Match numeric AppID if present in Data, or fallback to first successful entry if single-item response
					if (targetNum > 0 && entry.Data.SteamAppID == uint32(targetNum)) || len(result) == 1 {
						appData = entry
						found = true
						break
					}
				}
			}
		}

		if !found || !appData.Success || appData.Data == nil {
			return nil, fmt.Errorf("app details unavailable for AppID %s", appID)
		}

		return appData.Data, nil
	}

	var lastErr error

	// Tier 1: Query with explicit US region, English language, and mature content cookies
	primaryURL := fmt.Sprintf("%s/appdetails?appids=%s&cc=US&l=english", DefaultSteamStoreAPI, appID)
	details, err := doQuery(primaryURL)
	if err == nil && details != nil && details.Name != "" {
		if raw, mErr := json.Marshal(details); mErr == nil {
			_ = cache.SaveAppDetails(appID, raw)
		}
		cache.SaveAppName(appID, details.Name)
		return details, nil
	}
	if err != nil {
		lastErr = err
	}

	// Tier 2: Fallback query without region query parameters for region-exclusive store listings
	fallbackURL := fmt.Sprintf("%s/appdetails?appids=%s", DefaultSteamStoreAPI, appID)
	details, err = doQuery(fallbackURL)
	if err == nil && details != nil && details.Name != "" {
		if raw, mErr := json.Marshal(details); mErr == nil {
			_ = cache.SaveAppDetails(appID, raw)
		}
		cache.SaveAppName(appID, details.Name)
		return details, nil
	}
	if err != nil {
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("app details not found or unavailable for AppID %s across all store tiers", appID)
}

// FetchAppName gets the app name for a Steam AppID with offline cache fallback.
func FetchAppName(ctx context.Context, appID string) (string, error) {
	if name, ok := cache.GetAppName(appID); ok && name != "" {
		return name, nil
	}
	details, err := FetchAppDetails(ctx, appID)
	if err != nil {
		if cachedName, ok := cache.GetAppName(appID); ok && cachedName != "" {
			return cachedName, nil
		}
		return "", err
	}
	return details.Name, nil
}

// FetchSupportedLanguages fetches canonical language codes supported by a Steam game.
func FetchSupportedLanguages(ctx context.Context, appID string) ([]string, error) {
	if appID == "" || appID == "0" {
		return []string{"english"}, nil
	}
	details, err := FetchAppDetails(ctx, appID)
	if err != nil {
		return []string{"english"}, nil
	}
	return details.GetSupportedLanguageCodes(), nil
}

// IsSoundtrack returns true if the Steam app type represents music or a soundtrack.
func IsSoundtrack(appType string) bool {
	t := strings.ToLower(strings.TrimSpace(appType))
	return t == "music" || t == "soundtrack"
}

// FetchAppInfo gets the canonical app title, store type (game, music, dlc, etc.), and sanitized installdir.
func FetchAppInfo(ctx context.Context, appID string) (name, appType, installDir string, err error) {
	details, err := FetchAppDetails(ctx, appID)
	if err != nil {
		if cachedName, ok := cache.GetAppName(appID); ok && cachedName != "" {
			return cachedName, "game", util.SanitizeInstallDir(cachedName), nil
		}
		return "", "", "", err
	}
	t := details.Type
	if t == "" {
		t = "game"
	}
	cleanInstall := util.SanitizeInstallDir(details.Name)
	return details.Name, t, cleanInstall, nil
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

	searchURL := fmt.Sprintf("%s/storesearch/?term=%s&l=english&cc=US", DefaultSteamStoreAPI, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("create search request: %w", err)
	}

	resp, err := HTTPClientFromContext(ctx).Do(req)
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

// AppSearchResult represents a single candidate result returned from Steam Store search.
type AppSearchResult struct {
	AppID uint32
	Name  string
}

// SearchAppIDCandidates queries Steam Store API and returns up to 5 matching game candidates.
func SearchAppIDCandidates(ctx context.Context, query string) ([]AppSearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	searchURL := fmt.Sprintf("%s/storesearch/?term=%s&l=english&cc=US", DefaultSteamStoreAPI, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create search request: %w", err)
	}

	resp, err := HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch store search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("store search HTTP %d", resp.StatusCode)
	}

	var res SearchStoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode store search JSON: %w", err)
	}

	if res.Total == 0 || len(res.Items) == 0 {
		return nil, fmt.Errorf("no Steam AppID found for query %q", query)
	}

	results := make([]AppSearchResult, 0, len(res.Items))
	for _, item := range res.Items {
		results = append(results, AppSearchResult{
			AppID: uint32(item.ID),
			Name:  item.Name,
		})
		if len(results) >= 5 {
			break
		}
	}

	return results, nil
}

// DiscoverDLCIDs returns a deduplicated list of DLC AppIDs for a given Steam AppID.
// It prioritizes official structured JSON from the Steam Store API (Tier 1),
// falling back to AJAX recommendations scraping (Tier 3).
func DiscoverDLCIDs(ctx context.Context, appID string) ([]string, error) {
	if appID == "" || appID == "0" {
		return nil, fmt.Errorf("invalid appID %q", appID)
	}

	// Tier 1: Query official Store API AppDetails
	if details, err := FetchAppDetails(ctx, appID); err == nil && details != nil && len(details.DLC) > 0 {
		seen := make(map[string]struct{})
		var ids []string
		for _, dlcID := range details.DLC {
			idStr := fmt.Sprintf("%d", dlcID)
			if _, exists := seen[idStr]; !exists {
				seen[idStr] = struct{}{}
				ids = append(ids, idStr)
			}
		}
		if len(ids) > 0 {
			slog.Info("Discovered DLCs via official Steam Store API", "appID", appID, "count", len(ids))
			return ids, nil
		}
	}

	// Tier 3 (Fallback): Query Steam Store AJAX recommendations endpoint
	dlcURL := fmt.Sprintf("https://store.steampowered.com/dlc/%s/random/ajaxgetfilteredrecommendations/?query&count=10000", appID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dlcURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create dlc request: %w", err)
	}
	resp, err := HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch DLCs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch DLCs: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	re := regexp.MustCompile(`data-ds-appid=\\?"(\d+)`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	if len(matches) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{})
	var ids []string
	for _, m := range matches {
		id := m[1]
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	slog.Info("Discovered DLCs via AJAX fallback scraper", "appID", appID, "count", len(ids))
	return ids, nil
}

// DiscoverDLCs resolves both IDs and names for all DLCs belonging to a Steam AppID.
func DiscoverDLCs(ctx context.Context, appID string) ([]domain.DLCItem, error) {
	ids, err := DiscoverDLCIDs(ctx, appID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}

	var mu sync.Mutex
	items := make([]domain.DLCItem, 0, len(ids))
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(10) // Limit concurrency to 10 workers

	for _, idStr := range ids {
		id := idStr
		g.Go(func() error {
			name, nErr := FetchAppName(gCtx, id)
			if nErr != nil {
				slog.Warn("Failed to get name for DLC", "dlcID", id, "error", nErr)
				name = fmt.Sprintf("DLC %s", id)
			}
			mu.Lock()
			items = append(items, domain.DLCItem{AppID: id, Name: name})
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("failed to fetch DLC names: %w", err)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].AppID < items[j].AppID
	})
	return items, nil
}

// FetchDLCs fetches DLCs for a given AppID and configures them in steam_settings/configs.app.ini.
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

	dlcItems, err := DiscoverDLCs(ctx, appID)
	if err != nil {
		return err
	}
	if len(dlcItems) == 0 {
		slog.Warn("No DLCs found", "appID", appID)
		return nil
	}

	if dryRun {
		slog.Info("[DRY RUN] Would write DLC configuration", "path", configsAppIniPath, "count", len(dlcItems))
	} else {
		f, err := ini.LoadFile(configsAppIniPath)
		if err != nil {
			f = ini.NewFile()
		}

		sec := f.Section("app::dlcs")
		sec.Set("unlock_all", "0")

		for _, item := range dlcItems {
			sec.Set(item.AppID, item.Name)
		}

		if err := f.SaveFile(configsAppIniPath); err != nil {
			return fmt.Errorf("failed to save configs.app.ini: %w", err)
		}

		slog.Info("Wrote DLC configuration", "path", configsAppIniPath, "count", len(dlcItems))
	}

	return nil
}

// NewsItem aliases canonical domain.NewsItem.
type NewsItem = domain.NewsItem

// AppNewsResponse models Steam API ISteamNews/GetNewsForApp response.
type AppNewsResponse struct {
	AppNews struct {
		NewsItems []struct {
			Title     string `json:"title"`
			URL       string `json:"url"`
			Author    string `json:"author"`
			Contents  string `json:"contents"`
			FeedLabel string `json:"feedlabel"`
			Date      int64  `json:"date"`
		} `json:"newsitems"`
	} `json:"appnews"`
}

// FetchAppNews fetches the latest news/patch note item for a given Steam AppID.
func FetchAppNews(ctx context.Context, appID string) (string, error) {
	if news, ok := cache.GetNews(appID); ok {
		return news, nil
	}
	items, err := FetchAppNewsItems(ctx, appID, 1)
	if err != nil || len(items) == 0 {
		return "", err
	}
	res := fmt.Sprintf("%s (%s)", items[0].Title, items[0].FeedLabel)
	cache.SaveNews(appID, res)
	return res, nil
}

// FetchAppNewsItems fetches up to count news/patch note items for a given Steam AppID.
func FetchAppNewsItems(ctx context.Context, appID string, count int) ([]NewsItem, error) {
	if appID == "" || appID == "0" {
		return nil, fmt.Errorf("invalid appID")
	}
	if count <= 0 {
		count = 5
	}

	// 1. Check disk cache first to prevent rate limiting and provide instant loading
	var cachedItems []NewsItem
	if cachedBytes, ok := cache.GetNewsItems(appID); ok {
		if err := json.Unmarshal(cachedBytes, &cachedItems); err == nil && len(cachedItems) > 0 {
			if len(cachedItems) >= count {
				return cachedItems[:count], nil
			}
			// Cache has fewer items than requested; proceed to fetch from Steam API
		}
	}

	// 2. Fetch from Steam Web API - always request at least 15 items to populate a rich cache for all views
	fetchCount := count
	if fetchCount < 15 {
		fetchCount = 15
	}
	newsURL := fmt.Sprintf("%s/ISteamNews/GetNewsForApp/v2/?appid=%s&count=%d&maxlength=0", DefaultSteamWebAPI, appID, fetchCount)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, newsURL, nil)
	if err != nil {
		if len(cachedItems) > 0 {
			return cachedItems, nil
		}
		return nil, fmt.Errorf("create news request: %w", err)
	}
	req.Header.Set("User-Agent", "Gamux/1.0 (Linux; Steam Game Manager)")

	resp, err := HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		if len(cachedItems) > 0 {
			return cachedItems, nil
		}
		return nil, fmt.Errorf("fetch app news: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if len(cachedItems) > 0 {
			return cachedItems, nil
		}
		return nil, fmt.Errorf("app news HTTP %d", resp.StatusCode)
	}

	var res AppNewsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		if len(cachedItems) > 0 {
			return cachedItems, nil
		}
		return nil, fmt.Errorf("decode news JSON: %w", err)
	}

	if len(res.AppNews.NewsItems) == 0 {
		if len(cachedItems) > 0 {
			return cachedItems, nil
		}
		return nil, fmt.Errorf("no news items found")
	}

	items := make([]NewsItem, 0, len(res.AppNews.NewsItems))
	for _, item := range res.AppNews.NewsItems {
		items = append(items, NewsItem{
			Title:     item.Title,
			URL:       item.URL,
			Author:    item.Author,
			Contents:  item.Contents,
			FeedLabel: item.FeedLabel,
			Date:      item.Date,
		})
	}

	// 3. Cache fetched items to disk
	if itemsData, err := json.Marshal(items); err == nil {
		cache.SaveNewsItems(appID, itemsData)
	}

	if len(items) > count {
		return items[:count], nil
	}
	return items, nil
}

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// StripHTML converts HTML formatted patch notes into clean, readable terminal text.
func StripHTML(htmlStr string) string {
	if htmlStr == "" {
		return ""
	}
	s := strings.ReplaceAll(htmlStr, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = strings.ReplaceAll(s, "</p>", "\n\n")
	s = strings.ReplaceAll(s, "</li>", "\n")
	s = strings.ReplaceAll(s, "<li>", "  • ")
	s = htmlTagRegex.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return strings.TrimSpace(s)
}

type cachedPlayerCount struct {
	count     int
	fetchedAt time.Time
}

var (
	playerCountMu    sync.RWMutex
	playerCountCache = make(map[uint32]cachedPlayerCount)
)

// FetchCurrentPlayers queries Steam Web API ISteamUserStats/GetNumberOfCurrentPlayers/v1.
// Returns real-time concurrent player count cached for 5 minutes in-memory.
func FetchCurrentPlayers(ctx context.Context, appID uint32) (int, error) {
	if appID == 0 {
		return 0, fmt.Errorf("invalid appID 0")
	}

	playerCountMu.RLock()
	if entry, ok := playerCountCache[appID]; ok {
		if time.Since(entry.fetchedAt) < 5*time.Minute {
			playerCountMu.RUnlock()
			return entry.count, nil
		}
	}
	playerCountMu.RUnlock()

	apiURL := fmt.Sprintf("%s/ISteamUserStats/GetNumberOfCurrentPlayers/v1/?appid=%d", DefaultSteamWebAPI, appID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("create current players request: %w", err)
	}

	resp, err := HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch current players: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("current players HTTP status %d", resp.StatusCode)
	}

	var payload struct {
		Response struct {
			PlayerCount int `json:"player_count"`
			Result      int `json:"result"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, fmt.Errorf("decode current players JSON: %w", err)
	}

	count := payload.Response.PlayerCount
	playerCountMu.Lock()
	playerCountCache[appID] = cachedPlayerCount{
		count:     count,
		fetchedAt: time.Now(),
	}
	playerCountMu.Unlock()

	return count, nil
}
