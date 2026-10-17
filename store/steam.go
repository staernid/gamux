package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/steam"
)

// SteamProvider implements Provider for the Steam Store & Web API ecosystem.
type SteamProvider struct{}

// NewSteamProvider creates a new Steam store provider instance.
func NewSteamProvider() *SteamProvider {
	return &SteamProvider{}
}

func (s *SteamProvider) ID() string {
	return "steam"
}

func (s *SteamProvider) DisplayName() string {
	return "Steam"
}

func (s *SteamProvider) Description() string {
	return "Valve Steam Store, Community CDN, and Web API"
}

func (s *SteamProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		CanSearch:         true,
		CanFetchMetadata:  true,
		CanProvideArtwork: true,
		IsGameStore:       true,
	}
}

func (s *SteamProvider) IsGameStore() bool {
	return true
}

func (s *SteamProvider) Search(ctx context.Context, query string) ([]SearchResult, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}

	// Handle direct numeric AppID search
	if _, err := strconv.ParseUint(q, 10, 32); err == nil {
		name, err := steam.FetchAppName(ctx, q)
		if err == nil && name != "" {
			return []SearchResult{
				{
					ID:        q,
					Store:     "steam",
					Name:      name,
					PosterURL: fmt.Sprintf("https://cdn.akamai.steamstatic.com/steam/apps/%s/library_600x900_2x.jpg", q),
				},
			}, nil
		}
	}

	candidates, err := steam.SearchAppIDCandidates(ctx, q)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(candidates))
	for _, c := range candidates {
		idStr := strconv.FormatUint(uint64(c.AppID), 10)
		results = append(results, SearchResult{
			ID:        idStr,
			Store:     "steam",
			Name:      c.Name,
			PosterURL: fmt.Sprintf("https://cdn.akamai.steamstatic.com/steam/apps/%s/library_600x900_2x.jpg", idStr),
		})
	}

	return results, nil
}

func (s *SteamProvider) GetMetadata(ctx context.Context, gameID string) (*Metadata, error) {
	details, err := steam.FetchAppDetails(ctx, gameID)
	if err != nil {
		return nil, err
	}

	var genres []string
	for _, g := range details.Genres {
		if g.Description != "" {
			genres = append(genres, g.Description)
		}
	}

	summary := details.ShortDesc
	if summary == "" {
		summary = details.About
	}
	if summary == "" {
		summary = details.DetailedDesc
	}

	return &Metadata{
		ID:                 gameID,
		Store:              "steam",
		Name:               details.Name,
		Summary:            summary,
		HeaderImage:        details.HeaderImage,
		PosterImage:        fmt.Sprintf("https://cdn.akamai.steamstatic.com/steam/apps/%s/library_600x900_2x.jpg", gameID),
		Developers:         details.Developers,
		Publishers:         details.Publishers,
		Genres:             genres,
		SupportedLanguages: details.GetSupportedLanguageCodes(),
		ReleaseDate:        details.ReleaseDate.Date,
	}, nil
}

func (s *SteamProvider) GetArtwork(ctx context.Context, gameID string) (*Artwork, error) {
	return &Artwork{
		CoverURL:  fmt.Sprintf("https://cdn.akamai.steamstatic.com/steam/apps/%s/library_600x900_2x.jpg", gameID),
		BannerURL: fmt.Sprintf("https://cdn.akamai.steamstatic.com/steam/apps/%s/header.jpg", gameID),
		HeroURL:   fmt.Sprintf("https://cdn.akamai.steamstatic.com/steam/apps/%s/library_hero.jpg", gameID),
		LogoURL:   fmt.Sprintf("https://cdn.akamai.steamstatic.com/steam/apps/%s/logo.png", gameID),
		IconURL:   fmt.Sprintf("https://cdn.akamai.steamstatic.com/steam/apps/%s/capsule_sm_120.jpg", gameID),
	}, nil
}

// GetNews retrieves news and patch notes for a Steam AppID.
func (s *SteamProvider) GetNews(ctx context.Context, gameID string, count int) ([]domain.NewsItem, error) {
	return steam.FetchAppNewsItems(ctx, gameID, count)
}

// QueryArtworkCandidates returns official Steam CDN asset options using a typed query.
func (s *SteamProvider) QueryArtworkCandidates(ctx context.Context, query ArtworkQuery) ([]ArtworkOption, error) {
	assetType := query.AssetType
	if assetType == "" {
		assetType = "cover"
	}

	// 1. Direct AppID resolution (0 network calls)
	if query.SteamAppID > 0 {
		return s.generateCDNOptions(query.SteamAppID, assetType), nil
	}

	// 2. Fuzzy title search
	if query.Title != "" {
		results, err := s.Search(ctx, query.Title)
		if err != nil || len(results) == 0 {
			return nil, nil
		}
		var allOpts []ArtworkOption
		for i, res := range results {
			if i >= 2 {
				break
			}
			numID, err := strconv.ParseUint(res.ID, 10, 32)
			if err == nil && numID > 0 {
				allOpts = append(allOpts, s.generateCDNOptions(uint32(numID), assetType)...)
			}
		}
		return allOpts, nil
	}

	return nil, nil
}

func (s *SteamProvider) generateCDNOptions(appID uint32, assetType string) []ArtworkOption {
	idStr := strconv.FormatUint(uint64(appID), 10)
	var opts []ArtworkOption
	switch assetType {
	case "cover":
		opts = append(opts,
			ArtworkOption{
				ID:       idStr + "-cover-2x",
				Provider: "steam",
				Type:     "cover",
				URL:      fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_600x900_2x.jpg", idStr),
				ThumbURL: fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_600x900.jpg", idStr),
				Width:    600,
				Height:   900,
				Style:    "official",
				Score:    100,
			},
			ArtworkOption{
				ID:       idStr + "-cover-1x",
				Provider: "steam",
				Type:     "cover",
				URL:      fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_600x900.jpg", idStr),
				ThumbURL: fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_600x900.jpg", idStr),
				Width:    300,
				Height:   450,
				Style:    "official",
				Score:    95,
			},
			ArtworkOption{
				ID:       idStr + "-capsule-main",
				Provider: "steam",
				Type:     "cover",
				URL:      fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/capsule_616x353.jpg", idStr),
				ThumbURL: fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/capsule_616x353.jpg", idStr),
				Width:    616,
				Height:   353,
				Style:    "capsule",
				Score:    90,
			},
		)
	case "banner":
		opts = append(opts,
			ArtworkOption{
				ID:       idStr + "-header",
				Provider: "steam",
				Type:     "banner",
				URL:      fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/header.jpg", idStr),
				ThumbURL: fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/header.jpg", idStr),
				Width:    460,
				Height:   215,
				Style:    "official",
				Score:    100,
			},
			ArtworkOption{
				ID:       idStr + "-capsule-banner",
				Provider: "steam",
				Type:     "banner",
				URL:      fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/capsule_616x353.jpg", idStr),
				ThumbURL: fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/capsule_616x353.jpg", idStr),
				Width:    616,
				Height:   353,
				Style:    "capsule",
				Score:    90,
			},
		)
	case "hero":
		opts = append(opts,
			ArtworkOption{
				ID:       idStr + "-hero",
				Provider: "steam",
				Type:     "hero",
				URL:      fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_hero.jpg", idStr),
				ThumbURL: fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_hero.jpg", idStr),
				Width:    1920,
				Height:   620,
				Style:    "official",
				Score:    100,
			},
			ArtworkOption{
				ID:       idStr + "-hero-header",
				Provider: "steam",
				Type:     "hero",
				URL:      fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/header.jpg", idStr),
				ThumbURL: fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/header.jpg", idStr),
				Width:    460,
				Height:   215,
				Style:    "banner",
				Score:    80,
			},
		)
	case "logo":
		opts = append(opts, ArtworkOption{
			ID:       idStr + "-logo",
			Provider: "steam",
			Type:     "logo",
			URL:      fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/logo.png", idStr),
			ThumbURL: fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/logo.png", idStr),
			Style:    "official",
			Score:    100,
		})
	case "icon":
		opts = append(opts, ArtworkOption{
			ID:       idStr + "-icon",
			Provider: "steam",
			Type:     "icon",
			URL:      fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/capsule_sm_120.jpg", idStr),
			ThumbURL: fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/capsule_sm_120.jpg", idStr),
			Width:    120,
			Height:   45,
			Style:    "official",
			Score:    100,
		})
	}
	return opts
}
