package store

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/staernid/gamux/cache"
)

// DefaultBottlesProxyURL is the public endpoint operated by the Bottles project
// that proxies SteamGridDB cover searches without requiring user API authentication.
const DefaultBottlesProxyURL = "https://steamgrid.usebottles.com/api/search"

// SteamGridProxyProvider implements Provider and ArtworkCandidateProvider
// using the public community Bottles SteamGridDB proxy.
type SteamGridProxyProvider struct {
	baseURL    string
	httpClient *http.Client
}

// NewSteamGridProxyProvider creates a new proxy provider.
func NewSteamGridProxyProvider() *SteamGridProxyProvider {
	return &SteamGridProxyProvider{
		baseURL:    DefaultBottlesProxyURL,
		httpClient: &http.Client{Timeout: 8 * time.Second},
	}
}

// WithCustomClient overrides the HTTP client (useful for unit testing with httptest).
func (p *SteamGridProxyProvider) WithCustomClient(client *http.Client, baseURL string) *SteamGridProxyProvider {
	if client != nil {
		p.httpClient = client
	}
	if baseURL != "" {
		p.baseURL = baseURL
	}
	return p
}

func (p *SteamGridProxyProvider) ID() string {
	return "steamgrid-proxy"
}

func (p *SteamGridProxyProvider) DisplayName() string {
	return "SteamGrid Proxy"
}

func (p *SteamGridProxyProvider) Description() string {
	return "Public SteamGridDB proxy service for instant vertical covers without API credentials"
}

func (p *SteamGridProxyProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		CanSearch:         true,
		CanFetchMetadata:  false,
		CanProvideArtwork: true,
		IsGameStore:       false,
	}
}

// FetchCoverURL queries the proxy for a game title and returns the CDN image URL.
func (p *SteamGridProxyProvider) FetchCoverURL(ctx context.Context, title string) (string, error) {
	cleanTitle := strings.TrimSpace(title)
	if cleanTitle == "" {
		return "", fmt.Errorf("title cannot be empty")
	}

	cacheKey := "proxy_cover_" + strings.ToLower(cleanTitle)
	if cached, ok := cache.GetSGDBArtwork(cacheKey); ok && len(cached) > 0 {
		return string(cached), nil
	}

	reqURL := fmt.Sprintf("%s/%s", strings.TrimRight(p.baseURL, "/"), url.PathEscape(cleanTitle))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Gamux/1.0 (Linux; Game Manager)")

	resp, err := httpClientFromContext(ctx, p.httpClient).Do(req)
	if err != nil {
		return "", fmt.Errorf("query proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("cover not found for %q", title)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("proxy returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	imageURL := strings.Trim(strings.TrimSpace(string(body)), `"'`)
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		return "", fmt.Errorf("invalid image URL returned: %q", imageURL)
	}

	_ = cache.SaveSGDBArtwork(cacheKey, []byte(imageURL))
	return imageURL, nil
}

func (p *SteamGridProxyProvider) Search(ctx context.Context, query string) ([]SearchResult, error) {
	imageURL, err := p.FetchCoverURL(ctx, query)
	if err != nil || imageURL == "" {
		return nil, nil
	}

	return []SearchResult{
		{
			ID:        query,
			Store:     p.ID(),
			Name:      query,
			PosterURL: imageURL,
		},
	}, nil
}

func (p *SteamGridProxyProvider) GetMetadata(ctx context.Context, gameID string) (*Metadata, error) {
	imageURL, err := p.FetchCoverURL(ctx, gameID)
	if err != nil {
		return nil, err
	}
	return &Metadata{
		ID:          gameID,
		Store:       p.ID(),
		Name:        gameID,
		PosterImage: imageURL,
	}, nil
}

func (p *SteamGridProxyProvider) GetArtwork(ctx context.Context, gameID string) (*Artwork, error) {
	imageURL, err := p.FetchCoverURL(ctx, gameID)
	if err != nil {
		return nil, err
	}
	return &Artwork{
		CoverURL: imageURL,
	}, nil
}

func (p *SteamGridProxyProvider) QueryArtworkCandidates(ctx context.Context, query ArtworkQuery) ([]ArtworkOption, error) {
	// Proxy only provides vertical cover grids
	if query.AssetType != "" && query.AssetType != "cover" {
		return nil, nil
	}

	title := strings.TrimSpace(query.Title)
	if title == "" {
		return nil, nil
	}

	imageURL, err := p.FetchCoverURL(ctx, title)
	if err != nil || imageURL == "" {
		return nil, nil
	}

	return []ArtworkOption{
		{
			ID:       "proxy-cover-" + title,
			Provider: p.ID(),
			Type:     "cover",
			URL:      imageURL,
			ThumbURL: imageURL,
			Width:    600,
			Height:   900,
			Style:    "alternate",
		},
	}, nil
}
