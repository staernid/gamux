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

func (p *RAWGProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		CanSearch:         true,
		CanFetchMetadata:  true,
		CanProvideArtwork: true,
		IsGameStore:       false,
	}
}

type rawgSearchResponse struct {
	Count   int         `json:"count"`
	Results []rawgGame  `json:"results"`
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
	ShortScreenshots []rawgScreenshot `json:"short_screenshots"`
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
	params.Set("search_precise", "true")

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

	qWords := strings.Fields(strings.ToLower(q))
	var results []SearchResult

	for _, g := range data.Results {
		if g.Name == "" {
			continue
		}

		// Enforce keyword token validation to prevent unrelated catalog recommendations
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

// GetMetadata fetches full game details and descriptions for a RAWG ID or slug.
func (p *RAWGProvider) GetMetadata(ctx context.Context, gameID string) (*Metadata, error) {
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

	return meta, nil
}

// GetArtwork resolves background and hero art for a RAWG ID.
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
	for _, r := range results {
		if util.IsTitleMatch(r.Name, q) {
			targetID = r.ID
			targetName = r.Name
			break
		}
	}
	if targetID == "" && len(results) > 0 {
		targetID = results[0].ID
		targetName = results[0].Name
	}

	if targetID == "" {
		return nil, nil
	}

	var options []ArtworkOption

	// 2. Fetch game details and screenshots
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
							ID:       fmt.Sprintf("rawg_%s_screen_%d", targetID, s.ID),
							Provider: "rawg",
							Type:     domain.AssetHero,
							URL:      s.Image,
							ThumbURL: s.Image,
							Width:    1920,
							Height:   1080,
							Style:    "official",
							Author:   targetName,
							Score:    100 - idx,
						})
						options = append(options, ArtworkOption{
							ID:       fmt.Sprintf("rawg_%s_banner_%d", targetID, s.ID),
							Provider: "rawg",
							Type:     domain.AssetBanner,
							URL:      s.Image,
							ThumbURL: s.Image,
							Width:    1920,
							Height:   1080,
							Style:    "official",
							Author:   targetName,
							Score:    100 - idx,
						})
					}
				}
			}
		}
	}

	slog.Debug("Resolved RAWG artwork candidates", "title", q, "count", len(options))
	return options, nil
}
