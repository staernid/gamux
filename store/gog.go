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
)

// GOGProvider implements Provider for the GOG ecosystem.
type GOGProvider struct {
	httpClient *http.Client
}

// NewGOGProvider creates a new GOG store provider instance.
func NewGOGProvider() *GOGProvider {
	return &GOGProvider{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (g *GOGProvider) ID() string {
	return "gog"
}

func (g *GOGProvider) DisplayName() string {
	return "GOG.com"
}

func (g *GOGProvider) IsGameStore() bool {
	return true
}

type gogProduct struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Image     string `json:"image"`
	Slug      string `json:"slug"`
	WorksOn   map[string]bool `json:"worksOn"`
}

type gogCatalogResponse struct {
	Products []gogProduct `json:"products"`
	TotalProducts int   `json:"totalProducts"`
}

func (g *GOGProvider) Search(ctx context.Context, query string) ([]SearchResult, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}

	searchURL := fmt.Sprintf("https://embed.gog.com/games/ajax/filtered?mediaType=game&search=%s", url.QueryEscape(q))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GOG search request: %w", err)
	}
	req.Header.Set("User-Agent", "Gamux/1.0 (Linux; GOG Game Manager)")

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

	results := make([]SearchResult, 0, len(res.Products))
	for _, p := range res.Products {
		img := p.Image
		if img != "" && !strings.HasPrefix(img, "http") {
			img = "https:" + img + "_product_quartet_250.jpg"
		}
		results = append(results, SearchResult{
			ID:        strconv.Itoa(p.ID),
			Store:     "gog",
			Name:      p.Title,
			PosterURL: img,
		})
	}

	return results, nil
}

func (g *GOGProvider) GetMetadata(ctx context.Context, gameID string) (*Metadata, error) {
	apiURL := fmt.Sprintf("https://api.gog.com/products/%s?expand=description", gameID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GOG metadata request: %w", err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GOG metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GOG metadata API returned HTTP %d", resp.StatusCode)
	}

	var raw struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Slug  string `json:"slug"`
		Images struct {
			Icon     string `json:"icon"`
			Logo     string `json:"logo"`
			Background string `json:"background"`
		} `json:"images"`
		Description struct {
			Short string `json:"short"`
		} `json:"description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode GOG metadata: %w", err)
	}

	return &Metadata{
		ID:          strconv.Itoa(raw.ID),
		Store:       "gog",
		Name:        raw.Title,
		Summary:     raw.Description.Short,
		HeaderImage: raw.Images.Background,
		PosterImage: raw.Images.Logo,
	}, nil
}

func (g *GOGProvider) GetArtwork(ctx context.Context, gameID string) (*Artwork, error) {
	meta, err := g.GetMetadata(ctx, gameID)
	if err != nil {
		return nil, err
	}

	return &Artwork{
		CoverURL:  meta.PosterImage,
		BannerURL: meta.HeaderImage,
		HeroURL:   meta.HeaderImage,
	}, nil
}
