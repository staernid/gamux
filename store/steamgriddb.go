package store

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/staernid/gamux/cache"
	"golang.org/x/sync/errgroup"
)

// SGDBProvider implements Provider for the SteamGridDB API v2.
type SGDBProvider struct {
	apiKey     string
	httpClient *http.Client
}

// NewSGDBProvider creates a new SteamGridDB provider instance.
func NewSGDBProvider(apiKey string) *SGDBProvider {
	return &SGDBProvider{
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: &http.Client{Timeout: 6 * time.Second},
	}
}

// SetAPIKey dynamically updates the provider's API key.
func (s *SGDBProvider) SetAPIKey(key string) {
	s.apiKey = strings.TrimSpace(key)
}

// APIKey returns the current active API key.
func (s *SGDBProvider) APIKey() string {
	return s.apiKey
}

func (s *SGDBProvider) ID() string {
	return "steamgriddb"
}

func (s *SGDBProvider) DisplayName() string {
	return "SteamGridDB"
}

func (s *SGDBProvider) Description() string {
	return "Community-driven high-resolution grid, hero, logo, and cover art platform"
}

func (s *SGDBProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		CanSearch:         true,
		CanFetchMetadata:  false,
		CanProvideArtwork: true,
		IsGameStore:       false,
	}
}

func (s *SGDBProvider) IsGameStore() bool {
	return false
}

type sgdbSearchResponse struct {
	Success bool `json:"success"`
	Data    []struct {
		ID          int      `json:"id"`
		Name        string   `json:"name"`
		ReleaseDate int64    `json:"release_date,omitempty"`
		Types       []string `json:"types,omitempty"`
	} `json:"data"`
}

type sgdbImageData struct {
	ID     int    `json:"id"`
	URL    string `json:"url"`
	Thumb  string `json:"thumb"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type sgdbImageResponse struct {
	Success bool            `json:"success"`
	Data    []sgdbImageData `json:"data"`
}

type sgdbGameResponse struct {
	Success bool `json:"success"`
	Data    struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		ReleaseDate int64  `json:"release_date,omitempty"`
	} `json:"data"`
}

func (s *SGDBProvider) doRequest(ctx context.Context, reqURL string, target any) error {
	if s.apiKey == "" {
		return fmt.Errorf("steamgriddb API key not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("User-Agent", "Gamux/1.0 (Linux; Game Manager)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("steamgriddb API returned HTTP %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

// Search searches SteamGridDB for games matching a query string or AppID.
func (s *SGDBProvider) Search(ctx context.Context, query string) ([]SearchResult, error) {
	q := strings.TrimSpace(query)
	if q == "" || s.apiKey == "" {
		return nil, nil
	}

	// If numeric, look up directly by Steam AppID first
	if numID, err := strconv.ParseUint(q, 10, 32); err == nil && numID > 0 {
		if sgdbID, name, err := s.GetGameBySteamAppID(ctx, uint32(numID)); err == nil && sgdbID != "" {
			return []SearchResult{
				{
					ID:    sgdbID,
					Store: "steamgriddb",
					Name:  name,
				},
			}, nil
		}
	}

	reqURL := fmt.Sprintf("https://www.steamgriddb.com/api/v2/search/autocomplete/%s", url.PathEscape(q))
	var res sgdbSearchResponse
	if err := s.doRequest(ctx, reqURL, &res); err != nil {
		return nil, err
	}

	if !res.Success || len(res.Data) == 0 {
		return nil, nil
	}

	results := make([]SearchResult, 0, len(res.Data))
	for _, item := range res.Data {
		results = append(results, SearchResult{
			ID:    strconv.Itoa(item.ID),
			Store: "steamgriddb",
			Name:  item.Name,
		})
	}

	return results, nil
}

// GetGameBySteamAppID looks up a game on SteamGridDB directly by its Steam AppID.
func (s *SGDBProvider) GetGameBySteamAppID(ctx context.Context, appID uint32) (string, string, error) {
	if appID == 0 || s.apiKey == "" {
		return "", "", fmt.Errorf("invalid appid or missing api key")
	}

	reqURL := fmt.Sprintf("https://www.steamgriddb.com/api/v2/games/steam/%d", appID)
	var res sgdbGameResponse
	if err := s.doRequest(ctx, reqURL, &res); err != nil {
		return "", "", err
	}

	if !res.Success || res.Data.ID == 0 {
		return "", "", fmt.Errorf("no game found for steam appid %d", appID)
	}

	return strconv.Itoa(res.Data.ID), res.Data.Name, nil
}

// GetMetadata retrieves game metadata from SteamGridDB by SGDB game ID or Steam AppID.
func (s *SGDBProvider) GetMetadata(ctx context.Context, gameID string) (*Metadata, error) {
	art, _ := s.GetArtwork(ctx, gameID)
	cover := ""
	header := ""
	if art != nil {
		cover = art.CoverURL
		header = art.BannerURL
	}

	return &Metadata{
		ID:          gameID,
		Store:       "steamgriddb",
		HeaderImage: header,
		PosterImage: cover,
	}, nil
}

// GetArtwork fetches the highest rated grid (cover), hero, logo, and icon from SteamGridDB concurrently with caching.
func (s *SGDBProvider) GetArtwork(ctx context.Context, gameID string) (*Artwork, error) {
	if gameID == "" || s.apiKey == "" {
		return nil, fmt.Errorf("missing game id or api key")
	}

	lookupType := "game"
	cleanID := strings.TrimSpace(gameID)
	if strings.HasPrefix(cleanID, "steam:") {
		lookupType = "steam"
		cleanID = strings.TrimPrefix(cleanID, "steam:")
	}

	cacheKey := fmt.Sprintf("%s_%s", lookupType, cleanID)
	if cachedData, ok := cache.GetSGDBArtwork(cacheKey); ok && len(cachedData) > 0 {
		var cached Artwork
		if err := json.Unmarshal(cachedData, &cached); err == nil && (cached.CoverURL != "" || cached.HeroURL != "" || cached.LogoURL != "" || cached.IconURL != "") {
			return &cached, nil
		}
	}

	art := &Artwork{}
	var mu sync.Mutex
	g, gCtx := errgroup.WithContext(ctx)

	// 1. Fetch Vertical Cover Art (600x900)
	g.Go(func() error {
		gridURL := fmt.Sprintf("https://www.steamgriddb.com/api/v2/grids/%s/%s?dimensions=600x900", lookupType, cleanID)
		var gridRes sgdbImageResponse
		if err := s.doRequest(gCtx, gridURL, &gridRes); err == nil && gridRes.Success && len(gridRes.Data) > 0 {
			mu.Lock()
			art.CoverURL = gridRes.Data[0].URL
			mu.Unlock()
		}
		return nil
	})

	// 2. Fetch Horizontal Banner Header (460x215 / 920x430)
	g.Go(func() error {
		bannerURL := fmt.Sprintf("https://www.steamgriddb.com/api/v2/grids/%s/%s?dimensions=460x215,920x430", lookupType, cleanID)
		var bannerRes sgdbImageResponse
		if err := s.doRequest(gCtx, bannerURL, &bannerRes); err == nil && bannerRes.Success && len(bannerRes.Data) > 0 {
			mu.Lock()
			art.BannerURL = bannerRes.Data[0].URL
			mu.Unlock()
		}
		return nil
	})

	// 3. Fetch Hero Wallpaper / Backdrop (1920x620)
	g.Go(func() error {
		heroURL := fmt.Sprintf("https://www.steamgriddb.com/api/v2/heroes/%s/%s", lookupType, cleanID)
		var heroRes sgdbImageResponse
		if err := s.doRequest(gCtx, heroURL, &heroRes); err == nil && heroRes.Success && len(heroRes.Data) > 0 {
			mu.Lock()
			art.HeroURL = heroRes.Data[0].URL
			mu.Unlock()
		}
		return nil
	})

	// 4. Fetch Logo (transparent title PNG)
	g.Go(func() error {
		logoURL := fmt.Sprintf("https://www.steamgriddb.com/api/v2/logos/%s/%s", lookupType, cleanID)
		var logoRes sgdbImageResponse
		if err := s.doRequest(gCtx, logoURL, &logoRes); err == nil && logoRes.Success && len(logoRes.Data) > 0 {
			mu.Lock()
			art.LogoURL = logoRes.Data[0].URL
			mu.Unlock()
		}
		return nil
	})

	// 5. Fetch Icon
	g.Go(func() error {
		iconURL := fmt.Sprintf("https://www.steamgriddb.com/api/v2/icons/%s/%s", lookupType, cleanID)
		var iconRes sgdbImageResponse
		if err := s.doRequest(gCtx, iconURL, &iconRes); err == nil && iconRes.Success && len(iconRes.Data) > 0 {
			mu.Lock()
			art.IconURL = iconRes.Data[0].URL
			mu.Unlock()
		}
		return nil
	})

	_ = g.Wait()

	if art.CoverURL != "" || art.BannerURL != "" || art.HeroURL != "" || art.LogoURL != "" || art.IconURL != "" {
		if raw, err := json.Marshal(art); err == nil {
			_ = cache.SaveSGDBArtwork(cacheKey, raw)
		}
	}

	return art, nil
}

func (s *SGDBProvider) parseImageOptions(items []sgdbImageData, assetType string) []ArtworkOption {
	options := make([]ArtworkOption, 0, len(items))
	for _, item := range items {
		thumb := item.Thumb
		if thumb == "" {
			thumb = item.URL
		}
		options = append(options, ArtworkOption{
			ID:       strconv.Itoa(item.ID),
			Provider: "steamgriddb",
			Type:     assetType,
			URL:      item.URL,
			ThumbURL: thumb,
			Width:    item.Width,
			Height:   item.Height,
			Style:    "alternate",
		})
	}
	return options
}

func normalizeTitle(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isTitleMatch(candidateName, targetTitle string) bool {
	c := normalizeTitle(candidateName)
	t := normalizeTitle(targetTitle)
	if c == "" || t == "" {
		return false
	}
	if c == t {
		return true
	}
	if strings.Contains(c, t) || strings.Contains(t, c) {
		diff := len(c) - len(t)
		if diff < 0 {
			diff = -diff
		}
		if diff <= 8 || (len(c) > 0 && float64(diff)/float64(len(c)) < 0.35) {
			return true
		}
	}
	return false
}

// QueryArtworkCandidates fetches a list of candidate assets using a typed ArtworkQuery.
func (s *SGDBProvider) QueryArtworkCandidates(ctx context.Context, query ArtworkQuery) ([]ArtworkOption, error) {
	if s.apiKey == "" {
		if (query.AssetType == "cover" || query.AssetType == "") && query.Title != "" {
			proxy := NewSteamGridProxyProvider().WithCustomClient(s.httpClient, "")
			return proxy.QueryArtworkCandidates(ctx, query)
		}
		return nil, fmt.Errorf("missing steamgriddb api key")
	}

	endpoint := "grids"
	dimQuery := "?dimensions=600x900"
	switch query.AssetType {
	case "cover":
		endpoint = "grids"
		dimQuery = "?dimensions=600x900"
	case "banner":
		endpoint = "grids"
		dimQuery = "?dimensions=460x215,920x430"
	case "hero":
		endpoint = "heroes"
		dimQuery = ""
	case "logo":
		endpoint = "logos"
		dimQuery = ""
	case "icon":
		endpoint = "icons"
		dimQuery = ""
	default:
		endpoint = "grids"
		dimQuery = "?dimensions=600x900"
	}

	// 1. Direct Steam AppID query (1:1 direct endpoint)
	if query.SteamAppID > 0 {
		reqURL := fmt.Sprintf("https://www.steamgriddb.com/api/v2/%s/steam/%d%s", endpoint, query.SteamAppID, dimQuery)
		var imgRes sgdbImageResponse
		if err := s.doRequest(ctx, reqURL, &imgRes); err == nil && imgRes.Success && len(imgRes.Data) > 0 {
			return s.parseImageOptions(imgRes.Data, query.AssetType), nil
		}
	}

	// 2. Direct SGDB Game ID query
	if query.SGDBID != "" {
		reqURL := fmt.Sprintf("https://www.steamgriddb.com/api/v2/%s/game/%s%s", endpoint, query.SGDBID, dimQuery)
		var imgRes sgdbImageResponse
		if err := s.doRequest(ctx, reqURL, &imgRes); err == nil && imgRes.Success && len(imgRes.Data) > 0 {
			return s.parseImageOptions(imgRes.Data, query.AssetType), nil
		}
	}

	// 3. Strict title search
	if query.Title != "" {
		results, err := s.Search(ctx, query.Title)
		if err != nil || len(results) == 0 {
			return nil, nil
		}
		var allOpts []ArtworkOption
		matchedCount := 0
		for _, res := range results {
			if !isTitleMatch(res.Name, query.Title) {
				continue
			}
			matchedCount++
			reqURL := fmt.Sprintf("https://www.steamgriddb.com/api/v2/%s/game/%s%s", endpoint, res.ID, dimQuery)
			var imgRes sgdbImageResponse
			if err := s.doRequest(ctx, reqURL, &imgRes); err == nil && imgRes.Success && len(imgRes.Data) > 0 {
				allOpts = append(allOpts, s.parseImageOptions(imgRes.Data, query.AssetType)...)
			}
			if matchedCount >= 1 {
				break
			}
		}
		return allOpts, nil
	}

	return nil, nil
}
