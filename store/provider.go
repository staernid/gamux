package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/staernid/gamux/domain"
)

// SearchResult represents a game candidate discovered via a store search query.
type SearchResult struct {
	ID          string `json:"id"`
	Store       string `json:"store"` // "steam", "gog", etc.
	Name        string `json:"name"`
	PosterURL   string `json:"poster_url,omitempty"`
	ReleaseDate string `json:"release_date,omitempty"`
}

// Metadata holds detailed information about a store title.
type Metadata struct {
	ID                 string   `json:"id"`
	Store              string   `json:"store"`
	Name               string   `json:"name"`
	Summary            string   `json:"summary,omitempty"`
	HeaderImage        string   `json:"header_image,omitempty"`
	PosterImage        string   `json:"poster_image,omitempty"`
	Developers         []string `json:"developers,omitempty"`
	Publishers         []string `json:"publishers,omitempty"`
	Genres             []string `json:"genres,omitempty"`
	SupportedLanguages []string `json:"supported_languages,omitempty"`
	ReleaseDate        string   `json:"release_date,omitempty"`
	DLCs               []string `json:"dlcs,omitempty"`
}

// Artwork contains URLs for graphical assets associated with a game.
type Artwork struct {
	CoverURL  string `json:"cover_url,omitempty"`
	BannerURL string `json:"banner_url,omitempty"`
	HeroURL   string `json:"hero_url,omitempty"`
	LogoURL   string `json:"logo_url,omitempty"`
	IconURL   string `json:"icon_url,omitempty"`
}

// ArtworkOption aliases the canonical domain model.
type ArtworkOption = domain.ArtworkOption

// ArtworkQuery aliases the canonical domain model.
type ArtworkQuery = domain.ArtworkQuery

// ProviderCapabilities defines the functional features supported by a metadata/asset provider.
type ProviderCapabilities struct {
	CanSearch        bool `json:"can_search"`
	CanFetchMetadata bool `json:"can_fetch_metadata"`
	CanProvideArtwork bool `json:"can_provide_artwork"`
	IsGameStore      bool `json:"is_game_store"`
}

// ProviderDescriptor provides human-friendly and programmatic metadata about a registered provider.
type ProviderDescriptor struct {
	ID           string               `json:"id"`
	DisplayName  string               `json:"display_name"`
	Description  string               `json:"description,omitempty"`
	Capabilities ProviderCapabilities `json:"capabilities"`
}

// ArtworkCandidateProvider is an optional capability interface implemented by providers
// that can return collections of artwork candidates for user selection (e.g. covers, banners, heroes, logos).
type ArtworkCandidateProvider interface {
	QueryArtworkCandidates(ctx context.Context, query ArtworkQuery) ([]ArtworkOption, error)
}

// StoreProvider is an optional interface implemented by game catalog providers (Steam, GOG) that support game purchases/downloads.
type StoreProvider interface {
	Provider
	IsGameStore() bool
}

// Provider represents a store backend seam providing game discovery, metadata, and asset queries.
type Provider interface {
	// ID returns the canonical identifier for the store (e.g. "steam", "gog", "rawg").
	ID() string
	// DisplayName returns a human-friendly name (e.g. "Steam", "GOG.com", "RAWG Video Games Database").
	DisplayName() string
	// Search queries the store catalogue for game candidates matching a query string.
	Search(ctx context.Context, query string) ([]SearchResult, error)
	// GetMetadata retrieves full metadata for a specific store title by its ID.
	GetMetadata(ctx context.Context, gameID string) (*Metadata, error)
	// GetArtwork resolves high-quality grid and box art URLs for a game.
	GetArtwork(ctx context.Context, gameID string) (*Artwork, error)
}

// Registry coordinates multiple metadata providers.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry instantiates a thread-safe provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.ID()] = p
}

// Get returns a provider by its unique ID.
func (r *Registry) Get(id string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	return p, ok
}

// List returns all registered providers.
func (r *Registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		list = append(list, p)
	}
	return list
}

// ListDescriptors returns structured descriptor information for all registered providers.
func (r *Registry) ListDescriptors() []ProviderDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]ProviderDescriptor, 0, len(r.providers))
	for _, p := range r.providers {
		caps := ProviderCapabilities{
			CanSearch:        true,
			CanFetchMetadata: true,
			CanProvideArtwork: true,
		}
		if cp, ok := p.(interface{ Capabilities() ProviderCapabilities }); ok {
			caps = cp.Capabilities()
		} else {
			if sp, ok := p.(StoreProvider); ok {
				caps.IsGameStore = sp.IsGameStore()
			}
			if _, ok := p.(ArtworkCandidateProvider); ok {
				caps.CanProvideArtwork = true
			}
		}
		desc := ""
		if dp, ok := p.(interface{ Description() string }); ok {
			desc = dp.Description()
		}
		list = append(list, ProviderDescriptor{
			ID:           p.ID(),
			DisplayName:  p.DisplayName(),
			Description:  desc,
			Capabilities: caps,
		})
	}
	return list
}

// SearchStores queries only registered downloadable game catalog store providers (e.g. Steam, GOG), excluding artwork-only databases.
func (r *Registry) SearchStores(ctx context.Context, query string) ([]SearchResult, error) {
	r.mu.RLock()
	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		if sp, ok := p.(StoreProvider); ok {
			if sp.IsGameStore() {
				providers = append(providers, p)
			}
		} else if p.ID() != "steamgriddb" {
			providers = append(providers, p)
		}
	}
	r.mu.RUnlock()

	return r.searchProviders(ctx, query, providers)
}

// SearchAll queries all registered providers concurrently for game matches.
func (r *Registry) SearchAll(ctx context.Context, query string) ([]SearchResult, error) {
	r.mu.RLock()
	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	r.mu.RUnlock()

	return r.searchProviders(ctx, query, providers)
}

func (r *Registry) searchProviders(ctx context.Context, query string, providers []Provider) ([]SearchResult, error) {
	if len(providers) == 0 {
		return nil, nil
	}

	type searchOutcome struct {
		results []SearchResult
		err     error
	}

	ch := make(chan searchOutcome, len(providers))
	var wg sync.WaitGroup

	for _, p := range providers {
		prov := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := prov.Search(ctx, query)
			ch <- searchOutcome{results: res, err: err}
		}()
	}

	wg.Wait()
	close(ch)

	var allResults []SearchResult
	var errs []error
	for out := range ch {
		if out.err != nil {
			errs = append(errs, out.err)
			continue
		}
		allResults = append(allResults, out.results...)
	}

	if len(allResults) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("search failed across providers: %v", errs[0])
	}

	return allResults, nil
}

// QueryArtworkOptions queries all registered candidate providers for artwork matching a structured query.
func (r *Registry) QueryArtworkOptions(ctx context.Context, query ArtworkQuery) ([]ArtworkOption, error) {
	if query.AssetType == "" {
		query.AssetType = "cover"
	}

	queryCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	r.mu.RLock()
	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	r.mu.RUnlock()

	var allOptions []ArtworkOption
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, p := range providers {
		prov := p
		wg.Add(1)
		go func() {
			defer wg.Done()

			if cp, ok := prov.(ArtworkCandidateProvider); ok {
				opts, err := cp.QueryArtworkCandidates(queryCtx, query)
				if err == nil && len(opts) > 0 {
					mu.Lock()
					allOptions = append(allOptions, opts...)
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	return allOptions, nil
}

