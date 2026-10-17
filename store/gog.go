package store

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/util"
)

// GOGProvider implements Provider and ArtworkCandidateProvider for the GOG ecosystem.
type GOGProvider struct {
	httpClient *http.Client
}

// NewGOGProvider creates a new GOG store provider instance.
func NewGOGProvider() *GOGProvider {
	return &GOGProvider{
		httpClient: &http.Client{Timeout: 6 * time.Second},
	}
}

func (g *GOGProvider) ID() string {
	return "gog"
}

func (g *GOGProvider) DisplayName() string {
	return "GOG.com"
}

func (g *GOGProvider) Description() string {
	return "GOG.com DRM-free digital store & catalog"
}

func (g *GOGProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		CanSearch:         true,
		CanFetchMetadata:  true,
		CanProvideArtwork: true,
		IsGameStore:       true,
	}
}

func (g *GOGProvider) IsGameStore() bool {
	return true
}

type gogCatalogProduct struct {
	ID                    string   `json:"id"`
	Title                 string   `json:"title"`
	Slug                  string   `json:"slug"`
	CoverVertical         string   `json:"coverVertical"`
	CoverHorizontal       string   `json:"coverHorizontal"`
	Logo                  string   `json:"logo"`
	GalaxyBackgroundImage string   `json:"galaxyBackgroundImage"`
	Developers            []string `json:"developers"`
	Publishers            []string `json:"publishers"`
	ReleaseDate           string   `json:"releaseDate"`
	Genres                []struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"genres"`
}

type gogCatalogResponse struct {
	Products []gogCatalogProduct `json:"products"`
}

func (g *GOGProvider) Search(ctx context.Context, query string) ([]SearchResult, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}

	searchURL := fmt.Sprintf("https://catalog.gog.com/v1/catalog?query=like:%s&order=desc:score&limit=20", url.QueryEscape(q))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GOG search request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) Gamux/1.0")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query GOG catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GOG catalog search returned HTTP %d", resp.StatusCode)
	}

	var res gogCatalogResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode GOG catalog response: %w", err)
	}

	qWords := strings.Fields(strings.ToLower(q))
	results := make([]SearchResult, 0, len(res.Products))
	for _, p := range res.Products {
		titleLower := strings.ToLower(p.Title)
		slugLower := strings.ToLower(p.Slug)

		// Filter out GOG default recommendation results when query didn't match
		matches := false
		if strings.Contains(titleLower, strings.ToLower(q)) || strings.Contains(slugLower, strings.ToLower(q)) {
			matches = true
		} else {
			wordMatches := 0
			for _, w := range qWords {
				if strings.Contains(titleLower, w) || strings.Contains(slugLower, w) {
					wordMatches++
				}
			}
			if len(qWords) > 0 && wordMatches >= (len(qWords)+1)/2 {
				matches = true
			}
		}

		if !matches {
			continue
		}

		poster := p.CoverVertical
		if poster == "" {
			poster = p.CoverHorizontal
		}
		results = append(results, SearchResult{
			ID:          p.ID,
			Store:       "gog",
			Name:        p.Title,
			PosterURL:   poster,
			ReleaseDate: p.ReleaseDate,
		})
	}

	return results, nil
}

func (g *GOGProvider) fetchCatalogProduct(ctx context.Context, gameID, title, slug string) (*gogCatalogProduct, error) {
	var searchQueries []string
	if title != "" {
		searchQueries = append(searchQueries, title)
	}
	if slug != "" && slug != title {
		searchQueries = append(searchQueries, slug)
	}
	if gameID != "" && gameID != title && gameID != slug {
		searchQueries = append(searchQueries, gameID)
	}

	for _, sq := range searchQueries {
		sq = strings.TrimSpace(sq)
		if sq == "" {
			continue
		}

		apiURL := fmt.Sprintf("https://catalog.gog.com/v1/catalog?query=like:%s&limit=10", url.QueryEscape(sq))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) Gamux/1.0")

		resp, err := g.httpClient.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var res gogCatalogResponse
			if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
				for _, p := range res.Products {
					if (gameID != "" && p.ID == gameID) ||
						(slug != "" && strings.EqualFold(p.Slug, slug)) ||
						(title != "" && (strings.EqualFold(p.Title, title) || strings.EqualFold(p.Slug, strings.ReplaceAll(strings.ToLower(title), " ", "_")))) {
						resp.Body.Close()
						return &p, nil
					}
				}
			}
		}
		resp.Body.Close()
	}

	return nil, fmt.Errorf("gog catalog product not found")
}

func (g *GOGProvider) GetMetadata(ctx context.Context, gameID string) (*Metadata, error) {
	gameID = strings.TrimSpace(gameID)
	if gameID == "" {
		return nil, fmt.Errorf("empty GOG game ID")
	}

	if cachedData, ok := cache.GetStoreMetadata("gog", gameID); ok && len(cachedData) > 0 {
		var meta Metadata
		if err := json.Unmarshal(cachedData, &meta); err == nil && meta.Name != "" {
			return &meta, nil
		}
	}

	meta := &Metadata{
		ID:    gameID,
		Store: "gog",
	}

	var productSlug string
	var productTitle string

	// 1. If numeric ID, query api.gog.com/products/{id} directly (100% exact match, returns 404 if not found)
	if _, err := strconv.Atoi(gameID); err == nil {
		apiURL := fmt.Sprintf("https://api.gog.com/products/%s?expand=description", gameID)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create GOG request: %w", err)
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) Gamux/1.0")

		resp, err := g.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to query GOG product: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var raw struct {
				ID     int    `json:"id"`
				Title  string `json:"title"`
				Slug   string `json:"slug"`
				Images struct {
					Icon       string `json:"icon"`
					Logo       string `json:"logo"`
					Logo2x     string `json:"logo2x"`
					Background string `json:"background"`
				} `json:"images"`
				Description struct {
					Short string `json:"short"`
					Full  string `json:"full"`
				} `json:"description"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&raw); err == nil {
				meta.ID = strconv.Itoa(raw.ID)
				meta.Name = raw.Title
				productSlug = raw.Slug
				productTitle = raw.Title
				summary := raw.Description.Short
				if summary == "" {
					summary = raw.Description.Full
				}
				meta.Summary = summary
				if raw.Images.Background != "" {
					meta.HeaderImage = normalizeGOGURL(raw.Images.Background)
				}
				logo := raw.Images.Logo2x
				if logo == "" {
					logo = raw.Images.Logo
				}
				if logo != "" {
					meta.PosterImage = normalizeGOGURL(logo)
				}
			}
		}
	}

	// 2. Query catalog.gog.com for high-res vertical cover & galaxy background if matching
	catProduct, _ := g.fetchCatalogProduct(ctx, meta.ID, productTitle, productSlug)
	if catProduct != nil {
		if meta.Name == "" {
			meta.Name = catProduct.Title
		}
		if catProduct.CoverVertical != "" {
			meta.PosterImage = catProduct.CoverVertical
		}
		if catProduct.GalaxyBackgroundImage != "" {
			meta.HeaderImage = catProduct.GalaxyBackgroundImage
		}
		meta.Developers = catProduct.Developers
		meta.Publishers = catProduct.Publishers
		meta.ReleaseDate = catProduct.ReleaseDate
		for _, gn := range catProduct.Genres {
			if gn.Name != "" {
				meta.Genres = append(meta.Genres, gn.Name)
			}
		}
	}

	if meta.Name == "" {
		return nil, fmt.Errorf("GOG metadata not found for %s", gameID)
	}

	if encoded, err := json.Marshal(meta); err == nil {
		_ = cache.SaveStoreMetadata("gog", gameID, encoded)
	}

	return meta, nil
}

func (g *GOGProvider) GetArtwork(ctx context.Context, gameID string) (*Artwork, error) {
	meta, err := g.GetMetadata(ctx, gameID)
	if err != nil {
		return nil, err
	}

	catProduct, _ := g.fetchCatalogProduct(ctx, meta.ID, meta.Name, "")

	art := &Artwork{
		CoverURL:  meta.PosterImage,
		BannerURL: meta.HeaderImage,
		HeroURL:   meta.HeaderImage,
	}

	if catProduct != nil {
		if catProduct.CoverVertical != "" {
			art.CoverURL = catProduct.CoverVertical
		}
		if catProduct.CoverHorizontal != "" {
			art.BannerURL = catProduct.CoverHorizontal
		}
		if catProduct.GalaxyBackgroundImage != "" {
			art.HeroURL = catProduct.GalaxyBackgroundImage
		}
		if catProduct.Logo != "" {
			art.LogoURL = catProduct.Logo
		}
	}

	return art, nil
}

func (g *GOGProvider) QueryArtworkCandidates(ctx context.Context, query ArtworkQuery) ([]ArtworkOption, error) {
	var gogID string

	if query.GOGID != "" {
		gogID = query.GOGID
	} else if query.Title != "" {
		results, err := g.Search(ctx, query.Title)
		if err == nil && len(results) > 0 {
			for _, r := range results {
				if util.IsTitleMatch(r.Name, query.Title) {
					gogID = r.ID
					break
				}
			}
			if gogID == "" && len(results) > 0 {
				gogID = results[0].ID
			}
		}
	}

	if gogID == "" {
		return nil, nil
	}

	meta, err := g.GetMetadata(ctx, gogID)
	if err != nil || meta == nil {
		return nil, nil
	}

	catProduct, _ := g.fetchCatalogProduct(ctx, meta.ID, meta.Name, "")

	var candidates []ArtworkOption
	switch strings.ToLower(query.AssetType) {
	case domain.AssetCover:
		cover := meta.PosterImage
		if catProduct != nil && catProduct.CoverVertical != "" {
			cover = catProduct.CoverVertical
		}
		if cover != "" {
			candidates = append(candidates, ArtworkOption{
				ID:       "gog-cover-vert",
				Provider: "gog",
				Type:     domain.AssetCover,
				URL:      cover,
				ThumbURL: cover,
				Style:    "official",
				Score:    100,
			})
		}
	case domain.AssetBanner:
		banner := meta.HeaderImage
		if catProduct != nil && catProduct.CoverHorizontal != "" {
			banner = catProduct.CoverHorizontal
		}
		if banner != "" {
			candidates = append(candidates, ArtworkOption{
				ID:       "gog-banner-horiz",
				Provider: "gog",
				Type:     domain.AssetBanner,
				URL:      banner,
				ThumbURL: banner,
				Style:    "official",
				Score:    100,
			})
		}
	case domain.AssetHero:
		hero := meta.HeaderImage
		if catProduct != nil && catProduct.GalaxyBackgroundImage != "" {
			hero = catProduct.GalaxyBackgroundImage
		}
		if hero != "" {
			candidates = append(candidates, ArtworkOption{
				ID:       "gog-hero-bg",
				Provider: "gog",
				Type:     domain.AssetHero,
				URL:      hero,
				ThumbURL: hero,
				Style:    "official",
				Score:    100,
			})
		}
	case domain.AssetLogo:
		if catProduct != nil && catProduct.Logo != "" {
			candidates = append(candidates, ArtworkOption{
				ID:       "gog-logo",
				Provider: "gog",
				Type:     domain.AssetLogo,
				URL:      catProduct.Logo,
				ThumbURL: catProduct.Logo,
				Style:    "official",
				Score:    100,
			})
		}
	}

	return candidates, nil
}

func normalizeGOGURL(u string) string {
	if strings.HasPrefix(u, "//") {
		return "https:" + u
	}
	return u
}
