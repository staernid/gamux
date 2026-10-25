package store

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util/testutil"
)

func newMockSteamHTTPClient() *http.Client {
	return &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			u := req.URL.String()

			// 1. Search storefront candidates
			if strings.Contains(u, "storesearch") || strings.Contains(u, "store.steampowered.com/api/storesearch") {
				jsonResp := `{"total": 1, "items": [{"id": 1091500, "name": "Cyberpunk 2077"}]}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
					Header:     make(http.Header),
				}, nil
			}

			// 2. App Details
			if strings.Contains(u, "appdetails") {
				jsonResp := `{
					"1091500": {
						"success": true,
						"data": {
							"type": "game",
							"name": "Cyberpunk 2077",
							"steam_appid": 1091500,
							"short_description": "An open-world action-adventure RPG.",
							"header_image": "https://cdn.akamai.steamstatic.com/steam/apps/1091500/header.jpg",
							"developers": ["CD PROJEKT RED"],
							"publishers": ["CD PROJEKT RED"],
							"genres": [{"id": "1", "description": "RPG"}],
							"release_date": {"coming_soon": false, "date": "Dec 9, 2020"}
						}
					}
				}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
					Header:     make(http.Header),
				}, nil
			}

			// 3. News
			if strings.Contains(u, "GetNewsForApp") {
				jsonResp := `{
					"appnews": {
						"appid": 1091500,
						"newsitems": [
							{"gid": "1", "title": "Patch 2.1 Released", "url": "https://steam.com/news/1", "contents": "Notes"}
						]
					}
				}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
					Header:     make(http.Header),
				}, nil
			}

			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
				Header:     make(http.Header),
			}, nil
		}),
	}
}

func TestSteamProvider_LiveMockFlow(t *testing.T) {
	mockClient := newMockSteamHTTPClient()
	ctx := steam.WithHTTPClient(context.Background(), mockClient)
	ctx = WithHTTPClient(ctx, mockClient)

	prov := NewSteamProvider()

	// 1. Search by title
	results, err := prov.Search(ctx, "Cyberpunk 2077")
	if err != nil {
		t.Fatalf("Steam Search failed: %v", err)
	}
	if len(results) == 0 || results[0].ID != "1091500" {
		t.Fatalf("unexpected search results: %+v", results)
	}

	// 2. GetMetadata
	meta, err := prov.GetMetadata(ctx, "1091500")
	if err != nil {
		t.Fatalf("Steam GetMetadata failed: %v", err)
	}
	if meta.Name != "Cyberpunk 2077" || meta.Store != "steam" || len(meta.Developers) == 0 {
		t.Errorf("unexpected metadata: %+v", meta)
	}

	// 3. GetArtwork
	art, err := prov.GetArtwork(ctx, "1091500")
	if err != nil {
		t.Fatalf("Steam GetArtwork failed: %v", err)
	}
	if !strings.Contains(art.CoverURL, "1091500") || !strings.Contains(art.BannerURL, "1091500") {
		t.Errorf("unexpected artwork URLs: %+v", art)
	}

	// 4. GetNews
	news, err := prov.GetNews(ctx, "1091500", 5)
	if err != nil {
		t.Fatalf("Steam GetNews failed: %v", err)
	}
	if len(news) != 1 || news[0].Title != "Patch 2.1 Released" {
		t.Errorf("unexpected news items: %+v", news)
	}

	// 5. QueryArtworkCandidates - Cover
	coverOpts, err := prov.QueryArtworkCandidates(ctx, ArtworkQuery{
		SteamAppID: 1091500,
		AssetType:  "cover",
	})
	if err != nil {
		t.Fatalf("QueryArtworkCandidates cover failed: %v", err)
	}
	if len(coverOpts) < 2 {
		t.Fatalf("expected at least 2 cover candidates (2x and 1x), got %d", len(coverOpts))
	}
	if coverOpts[0].Width != 600 || coverOpts[0].Height != 900 {
		t.Errorf("unexpected cover dimensions: %+v", coverOpts[0])
	}

	// 6. QueryArtworkCandidates - Banner
	bannerOpts, err := prov.QueryArtworkCandidates(ctx, ArtworkQuery{
		SteamAppID: 1091500,
		AssetType:  "banner",
	})
	if err != nil {
		t.Fatalf("QueryArtworkCandidates banner failed: %v", err)
	}
	if len(bannerOpts) == 0 || bannerOpts[0].Width != 460 {
		t.Errorf("unexpected banner options: %+v", bannerOpts)
	}

	// 7. QueryArtworkCandidates - Hero
	heroOpts, err := prov.QueryArtworkCandidates(ctx, ArtworkQuery{
		SteamAppID: 1091500,
		AssetType:  "hero",
	})
	if err != nil {
		t.Fatalf("QueryArtworkCandidates hero failed: %v", err)
	}
	if len(heroOpts) == 0 || heroOpts[0].Width != 1920 {
		t.Errorf("unexpected hero options: %+v", heroOpts)
	}

	// 8. QueryArtworkCandidates - Logo
	logoOpts, err := prov.QueryArtworkCandidates(ctx, ArtworkQuery{
		SteamAppID: 1091500,
		AssetType:  "logo",
	})
	if err != nil {
		t.Fatalf("QueryArtworkCandidates logo failed: %v", err)
	}
	if len(logoOpts) == 0 || !strings.Contains(logoOpts[0].URL, "logo.png") {
		t.Errorf("unexpected logo options: %+v", logoOpts)
	}

	// 9. QueryArtworkCandidates - Title Search
	titleOpts, err := prov.QueryArtworkCandidates(ctx, ArtworkQuery{
		Title:     "Cyberpunk 2077",
		AssetType: "cover",
	})
	if err != nil {
		t.Fatalf("QueryArtworkCandidates with title search failed: %v", err)
	}
	if len(titleOpts) == 0 || !strings.Contains(titleOpts[0].URL, "1091500") {
		t.Errorf("unexpected title options: %+v", titleOpts)
	}
}

func TestSteamProvider_ModernHashedAssetResolution(t *testing.T) {
	// Mock appdetails for modern release (e.g. AppID 4098031)
	modernDetailsJSON := `{
		"4098031": {
			"success": true,
			"data": {
				"name": "Shipping Store Simulator 2037",
				"steam_appid": 4098031,
				"header_image": "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/4098031/5779065f314165697c01a5382faee517b39e1436/header.jpg?t=1792433418",
				"capsule_image": "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/4098031/3485e9062fd28c4abcf6ba6d5f60e82119ade026/capsule_231x87.jpg?t=1792433418",
				"background_raw": "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/4098031/015503202645429ffdadd404e7a9ae14e0147489/page_bg_raw.jpg?t=1792433418"
			}
		}
	}`

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "appdetails") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(modernDetailsJSON)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := steam.WithHTTPClient(context.Background(), mockClient)
	ctx = WithHTTPClient(ctx, mockClient)

	prov := NewSteamProvider()

	// 1. GetArtwork resolves modern hashed header and background
	art, err := prov.GetArtwork(ctx, "4098031")
	if err != nil {
		t.Fatalf("GetArtwork failed: %v", err)
	}
	expectedHeader := "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/4098031/5779065f314165697c01a5382faee517b39e1436/header.jpg?t=1792433418"
	expectedHero := "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/4098031/015503202645429ffdadd404e7a9ae14e0147489/page_bg_raw.jpg?t=1792433418"

	if art.BannerURL != expectedHeader {
		t.Errorf("expected modern BannerURL %s, got %s", expectedHeader, art.BannerURL)
	}
	if art.HeroURL != expectedHero {
		t.Errorf("expected modern HeroURL %s, got %s", expectedHero, art.HeroURL)
	}

	// 2. QueryArtworkCandidates - Banner includes modern live header
	banners, err := prov.QueryArtworkCandidates(ctx, ArtworkQuery{
		SteamAppID: 4098031,
		AssetType:  "banner",
	})
	if err != nil {
		t.Fatalf("banner query failed: %v", err)
	}
	var foundLiveBanner bool
	for _, b := range banners {
		if b.URL == expectedHeader {
			foundLiveBanner = true
			if b.Score != 100 {
				t.Errorf("expected score 100 for live modern banner, got %d", b.Score)
			}
		}
	}
	if !foundLiveBanner {
		t.Errorf("expected live modern banner option in candidates")
	}

	// 3. QueryArtworkCandidates - Cover includes live capsule/header fallback
	covers, err := prov.QueryArtworkCandidates(ctx, ArtworkQuery{
		SteamAppID: 4098031,
		AssetType:  "cover",
	})
	if err != nil {
		t.Fatalf("cover query failed: %v", err)
	}
	var foundCoverFallback bool
	for _, c := range covers {
		if c.URL == expectedHeader || strings.Contains(c.URL, "capsule_231x87") {
			foundCoverFallback = true
		}
	}
	if !foundCoverFallback {
		t.Errorf("expected live store fallback candidate in cover options")
	}
}
