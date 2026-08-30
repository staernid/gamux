package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/util"
)

const (
	defaultRAWGKey = "c542e67aec3a4340908f9de9e86038af" // Public default integration key for open queries
	rawgBaseURL    = "https://api.rawg.io/api"
)

// RAWGProvider integrates the RAWG.io open video games database for game discovery,
// rich metadata, and artwork resolution.
type RAWGProvider struct {
	apiKey     string
	httpClient *http.Client
}

// NewRAWGProvider instantiates a new RAWG database provider.
func NewRAWGProvider(apiKey string) *RAWGProvider {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		key = defaultRAWGKey
	}
	return &RAWGProvider{
		apiKey: key,
		httpClient: &http.Client{
			Timeout: 6 * time.Second,
		},
	}
}

func (p *RAWGProvider) ID() string {
	return "rawg"
}

func (p *RAWGProvider) DisplayName() string {
	return "RAWG Video Games Database"
}

func (p *RAWGProvider) Description() string {
	return "Open multi-platform video game database with rich metadata, descriptions, and artwork."
}

func (p *RAWGProvider) IsGameStore() bool {
	return false
}

func (p *RAWGProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		CanSearch:         true,
		CanFetchMetadata:  true,
		CanProvideArtwork: true,
		IsGameStore:       false,
	}
}

type rawgSearchResponse struct {
	Count   int        `json:"count"`
	Results []rawgGame `json:"results"`
}

type rawgGame struct {
	ID               int              `json:"id"`
	Slug             string           `json:"slug"`
	Name             string           `json:"name"`
	Released         string           `json:"released"`
	BackgroundImage  string           `json:"background_image"`
	Rating           float64          `json:"rating"`
	Metacritic       int              `json:"metacritic"`
	DescriptionRaw   string           `json:"description_raw"`
	Description      string           `json:"description"`
	Genres           []rawgNamedItem  `json:"genres"`
	Developers       []rawgNamedItem  `json:"developers"`
	Publishers       []rawgNamedItem  `json:"publishers"`
	Platforms        []rawgPlatform   `json:"platforms"`
	Stores           []rawgStoreEntry `json:"stores"`
	ShortScreenshots []rawgScreenshot `json:"short_screenshots"`
}

type rawgStoreEntry struct {
	ID    int           `json:"id"`
	URL   string        `json:"url"`
	Store rawgNamedItem `json:"store"`
}

type rawgNamedItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type rawgPlatform struct {
	Platform rawgNamedItem `json:"platform"`
}

type rawgScreenshot struct {
	ID    int    `json:"id"`
	Image string `json:"image"`
}

type rawgScreenshotsResponse struct {
	Count   int              `json:"count"`
	Results []rawgScreenshot `json:"results"`
}

func (p *RAWGProvider) buildURL(endpoint string, params url.Values) string {
	if params == nil {
		params = url.Values{}
	}
	params.Set("key", p.apiKey)
	return fmt.Sprintf("%s%s?%s", rawgBaseURL, endpoint, params.Encode())
}

// Search queries the RAWG database for game titles matching the query string.
func (p *RAWGProvider) Search(ctx context.Context, query string) ([]SearchResult, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}

	params := url.Values{}
	params.Set("search", q)
	params.Set("page_size", "20")

	reqURL := p.buildURL("/games", params)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create RAWG search request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute RAWG search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RAWG search HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read RAWG search body: %w", err)
	}

	var data rawgSearchResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("unmarshal RAWG search response: %w", err)
	}

	var results []SearchResult

	for _, g := range data.Results {
		if g.Name == "" {
			continue
		}

		if !util.IsTitleMatch(g.Name, q) && !util.IsTitleMatch(g.Slug, q) {
			// Check if any significant query words match
			qWords := strings.Fields(strings.ToLower(q))
			nameLower := strings.ToLower(g.Name)
			slugLower := strings.ToLower(g.Slug)
			matchedTokens := 0
			for _, w := range qWords {
				if strings.Contains(nameLower, w) || strings.Contains(slugLower, w) {
					matchedTokens++
				}
			}
			if len(qWords) > 0 && matchedTokens == 0 {
				continue
			}
		}

		results = append(results, SearchResult{
			ID:          strconv.Itoa(g.ID),
			Store:       "rawg",
			Name:        g.Name,
			PosterURL:   g.BackgroundImage,
			ReleaseDate: g.Released,
		})
	}

	return results, nil
}

// ExtractLinkedStoreIDs parses linked store links (Steam AppID, GOG slug) from a RAWG game.
func ExtractLinkedStoreIDs(g *rawgGame) (steamAppID uint32, gogSlug string) {
	if g == nil {
		return 0, ""
	}
	for _, s := range g.Stores {
		u := strings.ToLower(s.URL)
		if strings.Contains(u, "store.steampowered.com/app/") {
			parts := strings.Split(u, "store.steampowered.com/app/")
			if len(parts) >= 2 {
				sub := strings.Trim(parts[1], "/")
				seg := strings.Split(sub, "/")[0]
				if id, ok := util.ParseAppID(seg); ok && id > 0 {
					steamAppID = id
				}
			}
		} else if strings.Contains(u, "gog.com/game/") || strings.Contains(u, "gog.com/en/game/") {
			parts := strings.Split(u, "game/")
			if len(parts) >= 2 {
				sub := strings.Trim(parts[1], "/")
				seg := strings.Split(sub, "/")[0]
				if seg != "" {
					gogSlug = seg
				}
			}
		}
	}
	return steamAppID, gogSlug
}

// FetchRawGame retrieves and unmarshals the raw game payload from RAWG.
func (p *RAWGProvider) FetchRawGame(ctx context.Context, gameID string) (*rawgGame, error) {
	id := strings.TrimSpace(gameID)
	if id == "" {
		return nil, fmt.Errorf("game ID cannot be empty")
	}

	reqURL := p.buildURL("/games/"+url.PathEscape(id), nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create RAWG game request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute RAWG game request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RAWG game lookup returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read RAWG game response: %w", err)
	}

	var g rawgGame
	if err := json.Unmarshal(body, &g); err != nil {
		return nil, fmt.Errorf("unmarshal RAWG game JSON: %w", err)
	}
	return &g, nil
}

// GetMetadata fetches full game details and descriptions for a RAWG ID or slug.
func (p *RAWGProvider) GetMetadata(ctx context.Context, gameID string) (*Metadata, error) {
	id := strings.TrimSpace(gameID)
	if id == "" {
		return nil, fmt.Errorf("game ID cannot be empty")
	}

	if cachedData, ok := cache.GetStoreMetadata("rawg", id); ok && len(cachedData) > 0 {
		var meta Metadata
		if err := json.Unmarshal(cachedData, &meta); err == nil && meta.Name != "" {
			return &meta, nil
		}
	}

	g, err := p.FetchRawGame(ctx, id)
	if err != nil {
		return nil, err
	}

	summary := strings.TrimSpace(g.DescriptionRaw)
	if summary == "" {
		summary = strings.TrimSpace(g.Description)
	}

	var devs, pubs, genres []string
	for _, d := range g.Developers {
		if d.Name != "" {
			devs = append(devs, d.Name)
		}
	}
	for _, pb := range g.Publishers {
		if pb.Name != "" {
			pubs = append(pubs, pb.Name)
		}
	}
	for _, gn := range g.Genres {
		if gn.Name != "" {
			genres = append(genres, gn.Name)
		}
	}

	meta := &Metadata{
		ID:          strconv.Itoa(g.ID),
		Store:       "rawg",
		Name:        g.Name,
		Summary:     summary,
		HeaderImage: g.BackgroundImage,
		PosterImage: g.BackgroundImage,
		Developers:  devs,
		Publishers:  pubs,
		Genres:      genres,
		ReleaseDate: g.Released,
	}

	if encoded, err := json.Marshal(meta); err == nil {
		_ = cache.SaveStoreMetadata("rawg", id, encoded)
	}

	return meta, nil
}

// GetArtwork resolves background, hero, and cover art for a RAWG ID.
func (p *RAWGProvider) GetArtwork(ctx context.Context, gameID string) (*Artwork, error) {
	meta, err := p.GetMetadata(ctx, gameID)
	if err != nil {
		return nil, err
	}

	art := &Artwork{
		CoverURL:  meta.PosterImage,
		BannerURL: meta.HeaderImage,
		HeroURL:   meta.HeaderImage,
	}
	return art, nil
}

// QueryArtworkCandidates implements ArtworkCandidateProvider for RAWG.
func (p *RAWGProvider) QueryArtworkCandidates(ctx context.Context, query ArtworkQuery) ([]ArtworkOption, error) {
	q := strings.TrimSpace(query.Title)
	if q == "" {
		return nil, nil
	}

	// 1. Search for matching game
	results, err := p.Search(ctx, q)
	if err != nil || len(results) == 0 {
		return nil, nil
	}

	var targetID string
	var targetName string
	var targetPoster string
	for _, r := range results {
		if util.IsTitleMatch(r.Name, q) {
			targetID = r.ID
			targetName = r.Name
			targetPoster = r.PosterURL
			break
		}
	}
	if targetID == "" && len(results) > 0 {
		targetID = results[0].ID
		targetName = results[0].Name
		targetPoster = results[0].PosterURL
	}

	if targetID == "" {
		return nil, nil
	}

	var options []ArtworkOption

	// Add primary artwork from main background image if present
	if targetPoster != "" {
		options = append(options, ArtworkOption{
			ID:       fmt.Sprintf("rawg_%s_cover_main", targetID),
			Provider: "rawg",
			Type:     domain.AssetCover,
			URL:      targetPoster,
			ThumbURL: targetPoster,
			Style:    "official",
			Author:   targetName,
			Score:    100,
		})
		options = append(options, ArtworkOption{
			ID:       fmt.Sprintf("rawg_%s_hero_main", targetID),
			Provider: "rawg",
			Type:     domain.AssetHero,
			URL:      targetPoster,
			ThumbURL: targetPoster,
			Width:    1920,
			Height:   1080,
			Style:    "official",
			Author:   targetName,
			Score:    100,
		})
		options = append(options, ArtworkOption{
			ID:       fmt.Sprintf("rawg_%s_banner_main", targetID),
			Provider: "rawg",
			Type:     domain.AssetBanner,
			URL:      targetPoster,
			ThumbURL: targetPoster,
			Width:    1920,
			Height:   1080,
			Style:    "official",
			Author:   targetName,
			Score:    100,
		})
	}

	// 2. Fetch raw game to discover linked Steam and GOG options
	if rg, err := p.FetchRawGame(ctx, targetID); err == nil && rg != nil {
		steamID, _ := ExtractLinkedStoreIDs(rg)
		if steamID > 0 {
			options = append(options,
				ArtworkOption{
					ID:       fmt.Sprintf("steam_%d_cover", steamID),
					Provider: "steam",
					Type:     domain.AssetCover,
					URL:      fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%d/library_600x900_2x.jpg", steamID),
					ThumbURL: fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%d/library_600x900.jpg", steamID),
					Width:    600,
					Height:   900,
					Style:    "official",
					Score:    100,
				},
				ArtworkOption{
					ID:       fmt.Sprintf("steam_%d_hero", steamID),
					Provider: "steam",
					Type:     domain.AssetHero,
					URL:      fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%d/library_hero.jpg", steamID),
					ThumbURL: fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%d/library_hero.jpg", steamID),
					Width:    1920,
					Height:   620,
					Style:    "official",
					Score:    100,
				},
				ArtworkOption{
					ID:       fmt.Sprintf("steam_%d_banner", steamID),
					Provider: "steam",
					Type:     domain.AssetBanner,
					URL:      fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%d/header.jpg", steamID),
					ThumbURL: fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%d/header.jpg", steamID),
					Width:    460,
					Height:   215,
					Style:    "official",
					Score:    100,
				},
				ArtworkOption{
					ID:       fmt.Sprintf("steam_%d_logo", steamID),
					Provider: "steam",
					Type:     domain.AssetLogo,
					URL:      fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%d/logo.png", steamID),
					ThumbURL: fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%d/logo.png", steamID),
					Style:    "official",
					Score:    100,
				},
			)
		}
	}

	// 3. Fetch game screenshots
	reqURL := p.buildURL("/games/"+url.PathEscape(targetID)+"/screenshots", nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err == nil {
		resp, err := p.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			if body, err := io.ReadAll(resp.Body); err == nil {
				var screenData rawgScreenshotsResponse
				if err := json.Unmarshal(body, &screenData); err == nil {
					for idx, s := range screenData.Results {
						if s.Image == "" {
							continue
						}
						options = append(options, ArtworkOption{
							ID:       fmt.Sprintf("rawg_%s_cover_%d", targetID, s.ID),
							Provider: "rawg",
							Type:     domain.AssetCover,
							URL:      s.Image,
							ThumbURL: s.Image,
							Style:    "screenshot",
							Author:   targetName,
							Score:    90 - idx,
						})
						options = append(options, ArtworkOption{
							ID:       fmt.Sprintf("rawg_%s_hero_%d", targetID, s.ID),
							Provider: "rawg",
							Type:     domain.AssetHero,
							URL:      s.Image,
							ThumbURL: s.Image,
							Width:    1920,
							Height:   1080,
							Style:    "screenshot",
							Author:   targetName,
							Score:    90 - idx,
						})
						options = append(options, ArtworkOption{
							ID:       fmt.Sprintf("rawg_%s_banner_%d", targetID, s.ID),
							Provider: "rawg",
							Type:     domain.AssetBanner,
							URL:      s.Image,
							ThumbURL: s.Image,
							Width:    1920,
							Height:   1080,
							Style:    "screenshot",
							Author:   targetName,
							Score:    90 - idx,
						})
					}
				}
			}
		}
	}

	// Filter options to match requested AssetType if specified
	targetType := strings.ToLower(query.AssetType)
	if targetType != "" {
		var filtered []ArtworkOption
		for _, opt := range options {
			if strings.EqualFold(opt.Type, targetType) {
				filtered = append(filtered, opt)
			}
		}
		return filtered, nil
	}

	slog.Debug("Resolved RAWG artwork candidates", "title", q, "count", len(options))
	return options, nil
}
