package store

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
	"github.com/staernid/gamux/util/testutil"
)

func TestArtworkService_MultiProviderResolutionCascade(t *testing.T) {
	// Setup unified mock server that handles CDN downloads, proxy calls, and store APIs
	var mockServer *httptest.Server
	mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Steam AppDetails for modern release 4098030
		if strings.Contains(path, "appdetails") && strings.Contains(r.URL.Query().Get("appids"), "4098030") {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `{
				"4098030": {
					"success": true,
					"data": {
						"name": "Shipping Store Simulator 2037",
						"steam_appid": 4098030,
						"header_image": "%s/modern/header.jpg",
						"background_raw": "%s/modern/hero.jpg"
					}
				}
			}`, mockServer.URL, mockServer.URL)
			return
		}

		if path == "/modern/header.jpg" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fake-modern-header-bytes"))
			return
		}
		if path == "/modern/hero.jpg" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fake-modern-hero-bytes"))
			return
		}

		// Legacy Steam CDN assets (for 4098030, static CDN 404s like on live Steam)
		if strings.Contains(path, "4098030") {
			http.NotFound(w, r)
			return
		}

		// Steam CDN assets
		if strings.Contains(path, "library_600x900_2x.jpg") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fake-steam-vertical-cover"))
			return
		}
		if strings.Contains(path, "library_hero.jpg") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fake-steam-hero"))
			return
		}
		if strings.Contains(path, "header.jpg") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fake-steam-banner"))
			return
		}

		// Bottles Proxy
		if strings.HasPrefix(path, "/bottles/IndieGame") {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, "%q", mockServer.URL+"/cdn/indie_cover.jpg")
			return
		}
		if path == "/cdn/indie_cover.jpg" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fake-proxy-cover"))
			return
		}

		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	// Redirect Steam CDN and HTTP client to mock server
	client := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			newURL := *req.URL
			parsed, _ := req.URL.Parse(mockServer.URL)
			newURL.Scheme = parsed.Scheme
			newURL.Host = parsed.Host
			req.URL = &newURL
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	ctx := WithHTTPClient(context.Background(), client)
	ctx = steam.WithHTTPClient(ctx, client)

	tmpDir := t.TempDir()
	reg := NewRegistry()
	reg.Register(NewSteamProvider())
	reg.Register(NewGOGProvider())
	reg.Register(NewRAWGProvider(""))
	reg.Register(NewSGDBProvider(""))
	proxy := NewSteamGridProxyProvider().WithCustomClient(client, mockServer.URL+"/bottles")
	reg.Register(proxy)

	svc := NewArtworkService(tmpDir, reg)
	svc.httpClient = client

	// 1. Steam game with AppID: resolves real 2:3 vertical cover from Steam CDN
	t.Run("SteamAppIDResolution", func(t *testing.T) {
		gameDir := t.TempDir()
		g := domain.NewGame("Portal 2", gameDir, "620", "linux")
		art, err := svc.Resolve(ctx, g)
		if err != nil {
			t.Fatalf("resolve artwork: %v", err)
		}
		if art.LocalCover == "" || !util.FileExists(art.LocalCover) {
			t.Errorf("expected local cover to exist, got %s", art.LocalCover)
		}
		if art.LocalHero == "" || !util.FileExists(art.LocalHero) {
			t.Errorf("expected local hero to exist, got %s", art.LocalHero)
		}
		if art.LocalBanner == "" || !util.FileExists(art.LocalBanner) {
			t.Errorf("expected local banner to exist, got %s", art.LocalBanner)
		}
	})

	// 2. Standalone Indie game: resolves cover via Bottles proxy fallback
	t.Run("BottlesProxyFallback", func(t *testing.T) {
		gameDir := t.TempDir()
		g := domain.NewGame("IndieGame", gameDir, "", "win64")
		art, err := svc.Resolve(ctx, g)
		if err != nil {
			t.Fatalf("resolve artwork: %v", err)
		}
		if art.LocalCover == "" || !util.FileExists(art.LocalCover) {
			t.Errorf("expected proxy-resolved cover, got %s", art.LocalCover)
		}
	})

	// 3. Fallback when only local hero exists: copies hero to cover
	t.Run("CrossAssetFallback", func(t *testing.T) {
		gameDir := t.TempDir()
		heroFile := filepath.Join(gameDir, "hero.png")
		_ = os.WriteFile(heroFile, []byte("local-hero-bytes"), 0644)

		g := domain.NewGame("CustomArtGame", gameDir, "", "linux")
		art, err := svc.Resolve(ctx, g)
		if err != nil {
			t.Fatalf("resolve artwork: %v", err)
		}
		if art.LocalHero != heroFile {
			t.Errorf("expected local hero %s, got %s", heroFile, art.LocalHero)
		}
		if art.LocalCover == "" || !util.FileExists(art.LocalCover) {
			t.Errorf("expected cross-asset cover to be populated from hero")
		}
	})

	// 4. Modern Steam game where static CDN 404s (e.g. AppID 4098030): resolves via appdetails hashed URLs
	t.Run("ModernSteamReleaseFallback", func(t *testing.T) {
		gameDir := t.TempDir()
		g := domain.NewGame("Shipping Store Simulator 2037", gameDir, "4098030", "win64")
		art, err := svc.Resolve(ctx, g)
		if err != nil {
			t.Fatalf("resolve artwork: %v", err)
		}
		if art.LocalBanner == "" || !util.FileExists(art.LocalBanner) {
			t.Errorf("expected local banner to be downloaded from modern header, got %s", art.LocalBanner)
		}
		if art.LocalHero == "" || !util.FileExists(art.LocalHero) {
			t.Errorf("expected local hero to be downloaded from modern background_raw, got %s", art.LocalHero)
		}
		if art.LocalCover == "" || !util.FileExists(art.LocalCover) {
			t.Errorf("expected local cover fallback to be populated, got empty")
		}
	})
}
