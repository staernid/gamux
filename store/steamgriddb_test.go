package store

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/staernid/gamux/util/testutil"
)

func newMockSGDBServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Authorization header check
		auth := r.Header.Get("Authorization")
		if auth != "Bearer valid-sgdb-key" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"success": false, "errors": ["Unauthorized"]}`))
			return
		}

		path := r.URL.Path

		// 1. Search Autocomplete: /api/v2/search/autocomplete/{term}
		if strings.HasPrefix(path, "/api/v2/search/autocomplete/") {
			term := strings.TrimPrefix(path, "/api/v2/search/autocomplete/")
			if strings.Contains(strings.ToLower(term), "cyberpunk") {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"success": true,
					"data": [
						{"id": 48721, "name": "Cyberpunk 2077", "types": ["steam", "gog"]},
						{"id": 99999, "name": "Cyberpunk 2077: Phantom Liberty", "types": ["steam"]}
					]
				}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true, "data": []}`))
			return
		}

		// 2. Game by Steam AppID: /api/v2/games/steam/{appID}
		if strings.HasPrefix(path, "/api/v2/games/steam/") {
			appID := strings.TrimPrefix(path, "/api/v2/games/steam/")
			if appID == "1091500" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"success": true,
					"data": {
						"id": 48721,
						"name": "Cyberpunk 2077",
						"release_date": 1607558400
					}
				}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": false, "data": {"id": 0}}`))
			return
		}

		// 3. Grids (Covers & Banners): /api/v2/grids/{type}/{id}
		if strings.HasPrefix(path, "/api/v2/grids/") {
			dim := r.URL.Query().Get("dimensions")
			if strings.Contains(dim, "600x900") {
				// Vertical cover
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"success": true,
					"data": [
						{"id": 1001, "url": "https://cdn.sgdb.com/cover1.jpg", "thumb": "https://cdn.sgdb.com/cover1_thumb.jpg", "width": 600, "height": 900},
						{"id": 1002, "url": "https://cdn.sgdb.com/cover2.jpg", "thumb": "https://cdn.sgdb.com/cover2_thumb.jpg", "width": 600, "height": 900}
					]
				}`))
				return
			}
			if strings.Contains(dim, "460x215") || strings.Contains(dim, "920x430") {
				// Horizontal banner
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"success": true,
					"data": [
						{"id": 2001, "url": "https://cdn.sgdb.com/banner1.jpg", "thumb": "https://cdn.sgdb.com/banner1_thumb.jpg", "width": 460, "height": 215}
					]
				}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true, "data": []}`))
			return
		}

		// 4. Heroes: /api/v2/heroes/{type}/{id}
		if strings.HasPrefix(path, "/api/v2/heroes/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"success": true,
				"data": [
					{"id": 3001, "url": "https://cdn.sgdb.com/hero1.jpg", "thumb": "https://cdn.sgdb.com/hero1_thumb.jpg", "width": 1920, "height": 620}
				]
			}`))
			return
		}

		// 5. Logos: /api/v2/logos/{type}/{id}
		if strings.HasPrefix(path, "/api/v2/logos/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"success": true,
				"data": [
					{"id": 4001, "url": "https://cdn.sgdb.com/logo1.png", "thumb": "https://cdn.sgdb.com/logo1_thumb.png", "width": 800, "height": 300}
				]
			}`))
			return
		}

		// 6. Icons: /api/v2/icons/{type}/{id}
		if strings.HasPrefix(path, "/api/v2/icons/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"success": true,
				"data": [
					{"id": 5001, "url": "https://cdn.sgdb.com/icon1.ico", "thumb": "https://cdn.sgdb.com/icon1_thumb.ico", "width": 256, "height": 256}
				]
			}`))
			return
		}

		http.NotFound(w, r)
	}))
}

func TestSGDBProvider_LiveMockFlow(t *testing.T) {
	server := newMockSGDBServer(t)
	defer server.Close()

	// Redirect requests to test server by intercepting RoundTripper
	client := &http.Client{
		Transport: roundTripRewrite(server.URL),
	}

	ctx := WithHTTPClient(context.Background(), client)
	prov := NewSGDBProvider("valid-sgdb-key")

	// 1. Search by text
	results, err := prov.Search(ctx, "Cyberpunk")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != "48721" || results[0].Name != "Cyberpunk 2077" {
		t.Errorf("unexpected first result: %+v", results[0])
	}

	// 2. Search by numeric Steam AppID
	steamResults, err := prov.Search(ctx, "1091500")
	if err != nil {
		t.Fatalf("Search numeric AppID failed: %v", err)
	}
	if len(steamResults) != 1 || steamResults[0].ID != "48721" {
		t.Errorf("expected mapped SGDB ID 48721, got: %+v", steamResults)
	}

	// 3. Direct AppID lookup
	sgdbID, name, err := prov.GetGameBySteamAppID(ctx, 1091500)
	if err != nil {
		t.Fatalf("GetGameBySteamAppID failed: %v", err)
	}
	if sgdbID != "48721" || name != "Cyberpunk 2077" {
		t.Errorf("expected 48721 / Cyberpunk 2077, got %s / %s", sgdbID, name)
	}

	// 4. Full Artwork resolution (concurrent grids, heroes, logos, icons)
	art, err := prov.GetArtwork(ctx, "48721")
	if err != nil {
		t.Fatalf("GetArtwork failed: %v", err)
	}
	if art.CoverURL != "https://cdn.sgdb.com/cover1.jpg" {
		t.Errorf("expected CoverURL, got %s", art.CoverURL)
	}
	if art.BannerURL != "https://cdn.sgdb.com/banner1.jpg" {
		t.Errorf("expected BannerURL, got %s", art.BannerURL)
	}
	if art.HeroURL != "https://cdn.sgdb.com/hero1.jpg" {
		t.Errorf("expected HeroURL, got %s", art.HeroURL)
	}
	if art.LogoURL != "https://cdn.sgdb.com/logo1.png" {
		t.Errorf("expected LogoURL, got %s", art.LogoURL)
	}
	if art.IconURL != "https://cdn.sgdb.com/icon1.ico" {
		t.Errorf("expected IconURL, got %s", art.IconURL)
	}

	// 5. QueryArtworkCandidates - Cover
	coverOpts, err := prov.QueryArtworkCandidates(ctx, ArtworkQuery{
		SGDBID:    "48721",
		AssetType: "cover",
	})
	if err != nil {
		t.Fatalf("QueryArtworkCandidates cover failed: %v", err)
	}
	if len(coverOpts) != 2 {
		t.Fatalf("expected 2 cover options, got %d", len(coverOpts))
	}
	if coverOpts[0].Width != 600 || coverOpts[0].Height != 900 || coverOpts[0].Provider != "steamgriddb" {
		t.Errorf("unexpected cover option properties: %+v", coverOpts[0])
	}

	// 6. QueryArtworkCandidates - Hero
	heroOpts, err := prov.QueryArtworkCandidates(ctx, ArtworkQuery{
		SGDBID:    "48721",
		AssetType: "hero",
	})
	if err != nil {
		t.Fatalf("QueryArtworkCandidates hero failed: %v", err)
	}
	if len(heroOpts) != 1 || heroOpts[0].Width != 1920 {
		t.Errorf("unexpected hero options: %+v", heroOpts)
	}

	// 7. QueryArtworkCandidates - Title Search matching
	titleOpts, err := prov.QueryArtworkCandidates(ctx, ArtworkQuery{
		Title:     "Cyberpunk 2077",
		AssetType: "banner",
	})
	if err != nil {
		t.Fatalf("QueryArtworkCandidates title search failed: %v", err)
	}
	if len(titleOpts) != 1 || titleOpts[0].Width != 460 {
		t.Errorf("unexpected title banner option: %+v", titleOpts)
	}
}

func TestSGDBProvider_AuthFailure(t *testing.T) {
	server := newMockSGDBServer(t)
	defer server.Close()

	client := &http.Client{
		Transport: roundTripRewrite(server.URL),
	}
	ctx := WithHTTPClient(context.Background(), client)

	// Provider with invalid key should receive HTTP 401
	prov := NewSGDBProvider("invalid-key")
	_, err := prov.Search(ctx, "Cyberpunk")
	if err == nil {
		t.Fatalf("expected auth error, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to mention 401, got %v", err)
	}
}

// roundTripRewrite rewrites outgoing steamgriddb.com requests to the target test server.
func roundTripRewrite(targetURL string) http.RoundTripper {
	return testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		newURL := *req.URL
		parsed, _ := req.URL.Parse(targetURL)
		newURL.Scheme = parsed.Scheme
		newURL.Host = parsed.Host
		req.URL = &newURL
		return http.DefaultTransport.RoundTrip(req)
	})
}
