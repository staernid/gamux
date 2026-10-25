package store

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/util/testutil"
)

func newMockRAWGServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "blocked-key" {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"detail": "Request was throttled. Expected available in 3600 seconds."}`))
			return
		}

		path := r.URL.Path

		// 1. Search: /api/games?search=...
		if path == "/api/games" {
			query := r.URL.Query().Get("search")
			if strings.Contains(strings.ToLower(query), "hollow") {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"count": 1,
					"results": [{
						"id": 9767,
						"slug": "hollow-knight",
						"name": "Hollow Knight",
						"released": "2017-02-24",
						"background_image": "https://media.rawg.io/media/games/4cf/4cfc6b7f1850590a4634b08bfab308ab.jpg",
						"rating": 4.41,
						"metacritic": 90
					}]
				}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"count": 0, "results": []}`))
			return
		}

		// 2. Game Screenshots: /api/games/{id}/screenshots
		if strings.HasSuffix(path, "/screenshots") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"count": 2,
				"results": [
					{"id": 101, "image": "https://media.rawg.io/screenshots/shot1.jpg"},
					{"id": 102, "image": "https://media.rawg.io/screenshots/shot2.jpg"}
				]
			}`))
			return
		}

		// 3. Single Game Lookup: /api/games/{id}
		if strings.HasPrefix(path, "/api/games/") {
			id := strings.TrimPrefix(path, "/api/games/")
			if id == "9767" || id == "hollow-knight" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"id": 9767,
					"slug": "hollow-knight",
					"name": "Hollow Knight",
					"released": "2017-02-24",
					"background_image": "https://media.rawg.io/media/games/hollow_knight_bg.jpg",
					"description_raw": "Forged with the finest steel, explore Hallownest.",
					"developers": [{"id": 1, "name": "Team Cherry", "slug": "team-cherry"}],
					"publishers": [{"id": 1, "name": "Team Cherry", "slug": "team-cherry"}],
					"genres": [{"id": 1, "name": "Action", "slug": "action"}],
					"stores": [
						{"id": 1, "url": "https://store.steampowered.com/app/367520/"},
						{"id": 2, "url": "https://www.gog.com/game/hollow_knight"}
					]
				}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"detail": "Not found."}`))
			return
		}

		http.NotFound(w, r)
	}))
}

func roundTripRAWGRewrite(targetURL string) http.RoundTripper {
	return testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		newURL := *req.URL
		parsed, _ := req.URL.Parse(targetURL)
		newURL.Scheme = parsed.Scheme
		newURL.Host = parsed.Host
		req.URL = &newURL
		return http.DefaultTransport.RoundTrip(req)
	})
}

func TestRAWGProvider_LiveMockFlow(t *testing.T) {
	server := newMockRAWGServer(t)
	defer server.Close()

	client := &http.Client{
		Transport: roundTripRAWGRewrite(server.URL),
	}
	ctx := WithHTTPClient(context.Background(), client)
	prov := NewRAWGProvider("test-rawg-key")

	// 1. Search
	results, err := prov.Search(ctx, "Hollow Knight")
	if err != nil {
		t.Fatalf("RAWG Search failed: %v", err)
	}
	if len(results) != 1 || results[0].Name != "Hollow Knight" || results[0].ID != "9767" {
		t.Fatalf("unexpected search results: %+v", results)
	}

	// 2. GetMetadata
	meta, err := prov.GetMetadata(ctx, "9767")
	if err != nil {
		t.Fatalf("RAWG GetMetadata failed: %v", err)
	}
	if meta.Name != "Hollow Knight" || len(meta.Developers) != 1 || meta.Developers[0] != "Team Cherry" {
		t.Errorf("unexpected metadata: %+v", meta)
	}

	// 3. GetArtwork - Cover is intentionally empty for 2:3 vertical resolution; Hero & Banner populated
	art, err := prov.GetArtwork(ctx, "9767")
	if err != nil {
		t.Fatalf("RAWG GetArtwork failed: %v", err)
	}
	if art.CoverURL != "" {
		t.Errorf("expected RAWG CoverURL to be empty (wallpapers should not masquerade as vertical covers), got %s", art.CoverURL)
	}
	if art.HeroURL == "" || art.BannerURL == "" {
		t.Errorf("expected non-empty HeroURL and BannerURL from RAWG background, got hero=%s banner=%s", art.HeroURL, art.BannerURL)
	}

	// 4. QueryArtworkCandidates - Cover assetType should discover linked Steam and GOG covers with high score
	candidates, err := prov.QueryArtworkCandidates(ctx, ArtworkQuery{
		Title:     "Hollow Knight",
		AssetType: domain.AssetCover,
	})
	if err != nil {
		t.Fatalf("RAWG QueryArtworkCandidates cover failed: %v", err)
	}
	if len(candidates) == 0 {
		t.Fatalf("expected resolved cover candidates via linked stores")
	}

	var foundSteamCover, foundWallpaperFallback bool
	for _, c := range candidates {
		if c.Provider == "steam" && c.Width == 600 && c.Height == 900 {
			foundSteamCover = true
		}
		if c.Provider == "rawg" && c.Style == "wallpaper" {
			foundWallpaperFallback = true
		}
	}
	if !foundSteamCover {
		t.Errorf("expected steam vertical cover discovered through RAWG store links")
	}
	if !foundWallpaperFallback {
		t.Errorf("expected RAWG background wallpaper candidate offered with 'wallpaper' style")
	}

	// 5. QueryArtworkCandidates - Hero assetType includes background and screenshots
	heroCandidates, err := prov.QueryArtworkCandidates(ctx, ArtworkQuery{
		Title:     "Hollow Knight",
		AssetType: domain.AssetHero,
	})
	if err != nil {
		t.Fatalf("RAWG QueryArtworkCandidates hero failed: %v", err)
	}
	if len(heroCandidates) < 2 {
		t.Fatalf("expected at least 2 hero options (main + screenshots), got %d", len(heroCandidates))
	}
}

func TestRAWGProvider_RateLimiting(t *testing.T) {
	server := newMockRAWGServer(t)
	defer server.Close()

	client := &http.Client{
		Transport: roundTripRAWGRewrite(server.URL),
	}
	ctx := WithHTTPClient(context.Background(), client)
	prov := NewRAWGProvider("blocked-key")

	_, err := prov.Search(ctx, "Hollow Knight")
	if err == nil {
		t.Fatalf("expected error on HTTP 429 rate limit, got nil")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("expected error message to mention 429, got %v", err)
	}
}
