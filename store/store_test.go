package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

type mockProvider struct {
	id          string
	displayName string
	results     []SearchResult
}

func (m *mockProvider) ID() string          { return m.id }
func (m *mockProvider) DisplayName() string { return m.displayName }
func (m *mockProvider) Description() string { return "Mock provider for testing" }
func (m *mockProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		CanSearch:         true,
		CanFetchMetadata:  true,
		CanProvideArtwork:  true,
		IsGameStore:       m.id == "steam" || m.id == "gog",
	}
}
func (m *mockProvider) Search(ctx context.Context, query string) ([]SearchResult, error) {
	return m.results, nil
}
func (m *mockProvider) GetMetadata(ctx context.Context, gameID string) (*Metadata, error) {
	return &Metadata{ID: gameID, Store: m.id, Name: "Mock Game"}, nil
}
func (m *mockProvider) GetArtwork(ctx context.Context, gameID string) (*Artwork, error) {
	return &Artwork{CoverURL: "https://example.com/cover.jpg"}, nil
}

func TestRegistry(t *testing.T) {
	reg := NewRegistry()

	steamMock := &mockProvider{
		id:          "steam",
		displayName: "Steam",
		results: []SearchResult{
			{ID: "10", Store: "steam", Name: "Counter-Strike"},
		},
	}
	gogMock := &mockProvider{
		id:          "gog",
		displayName: "GOG.com",
		results: []SearchResult{
			{ID: "12345", Store: "gog", Name: "The Witcher 3"},
		},
	}

	reg.Register(steamMock)
	reg.Register(gogMock)

	if len(reg.List()) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(reg.List()))
	}

	p, ok := reg.Get("steam")
	if !ok || p.DisplayName() != "Steam" {
		t.Fatalf("failed to get steam provider")
	}

	results, err := reg.SearchAll(context.Background(), "game")
	if err != nil {
		t.Fatalf("SearchAll failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results across all providers, got %d", len(results))
	}
}

func TestArtworkService_LocalWaterfall(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry()
	svc := NewArtworkService(tmpDir, reg)

	gameDir := t.TempDir()
	coverFile := filepath.Join(gameDir, "cover.jpg")
	_ = os.WriteFile(coverFile, []byte("fake-jpeg"), 0644)

	g := domain.NewGame("Local Game", gameDir, "", "linux")
	art, err := svc.Resolve(context.Background(), g)
	if err != nil {
		t.Fatalf("resolve artwork: %v", err)
	}

	if art.LocalCover != coverFile {
		t.Errorf("expected local cover to be %s, got %s", coverFile, art.LocalCover)
	}
}

func TestSGDBProvider_APIKey(t *testing.T) {
	p := NewSGDBProvider("test-key")
	if p.APIKey() != "test-key" {
		t.Errorf("expected test-key, got %s", p.APIKey())
	}
	p.SetAPIKey("updated-key")
	if p.APIKey() != "updated-key" {
		t.Errorf("expected updated-key, got %s", p.APIKey())
	}
	if p.ID() != "steamgriddb" || p.DisplayName() != "SteamGridDB" {
		t.Errorf("unexpected ID or DisplayName")
	}
}

func TestArtworkService_ApplyCustomArtwork(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry()
	svc := NewArtworkService(tmpDir, reg)

	gameDir := t.TempDir()
	customImg := filepath.Join(gameDir, "my_custom_cover.png")
	_ = os.WriteFile(customImg, []byte("custom-image-bytes"), 0644)

	g := domain.NewGame("Custom Game", gameDir, "9999", "linux")
	art, err := svc.ApplyCustomArtwork(context.Background(), g, "cover", customImg)
	if err != nil {
		t.Fatalf("apply custom artwork: %v", err)
	}

	if art == nil || !g.LockedArtwork {
		t.Fatalf("expected locked artwork flag to be true")
	}
	if g.CustomCoverURL != customImg {
		t.Errorf("expected CustomCoverURL %s, got %s", customImg, g.CustomCoverURL)
	}
	if !util.FileExists(art.LocalCover) {
		t.Errorf("expected cached cover file to exist at %s", art.LocalCover)
	}
}

func TestSteamProvider_QueryArtworkCandidates_AppID(t *testing.T) {
	sp := NewSteamProvider()
	opts, err := sp.QueryArtworkCandidates(context.Background(), ArtworkQuery{
		SteamAppID: 1091500,
		AssetType:  "cover",
	})
	if err != nil {
		t.Fatalf("QueryArtworkCandidates failed: %v", err)
	}
	if len(opts) == 0 {
		t.Fatalf("expected at least 1 cover option for steam app 1091500")
	}
	if opts[0].Provider != "steam" || opts[0].Type != "cover" {
		t.Errorf("unexpected artwork option attributes: %+v", opts[0])
	}
}

func TestSteamProvider_QueryArtworkCandidates_TitleSearch(t *testing.T) {
	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			jsonResp := `{"total": 1, "items": [{"id": 70, "name": "Half-Life"}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	ctx := steam.WithHTTPClient(context.Background(), mockClient)

	sp := NewSteamProvider()
	opts, err := sp.QueryArtworkCandidates(ctx, ArtworkQuery{
		Title:     "Half-Life",
		AssetType: "cover",
	})
	if err != nil {
		t.Fatalf("QueryArtworkCandidates with title failed: %v", err)
	}
	if len(opts) == 0 {
		t.Fatalf("expected resolved cover options for Half-Life")
	}
	if !strings.Contains(opts[0].URL, "/apps/70/") {
		t.Errorf("expected numeric AppID 70 in URL, got %s", opts[0].URL)
	}
}

func TestRegistry_QueryArtworkOptions(t *testing.T) {
	reg := NewRegistry()
	sp := NewSteamProvider()
	reg.Register(sp)

	opts, err := reg.QueryArtworkOptions(context.Background(), ArtworkQuery{
		SteamAppID: 70,
		AssetType:  "cover",
	})
	if err != nil {
		t.Fatalf("QueryArtworkOptions failed: %v", err)
	}
	if len(opts) == 0 {
		t.Fatalf("expected at least 1 cover option for Half-Life (AppID 70)")
	}
}

func TestSGDBProvider_IsTitleMatch(t *testing.T) {
	tests := []struct {
		candidate string
		target    string
		expected  bool
	}{
		{"Bang Bang Racing", "Bang Bang Barrage", false},
		{"Bang Bang Barrage", "Bang Bang Barrage", true},
		{"Cyberpunk 2077", "Cyberpunk 2077", true},
		{"The Witcher 3: Wild Hunt", "The Witcher 3", true},
		{"Hades II", "Hades", true},
		{"Grand Theft Auto V", "Need For Speed", false},
	}

	for _, tc := range tests {
		got := isTitleMatch(tc.candidate, tc.target)
		if got != tc.expected {
			t.Errorf("isTitleMatch(%q, %q) = %v; want %v", tc.candidate, tc.target, got, tc.expected)
		}
	}
}

func TestArtworkService_ApplyCustomHeroAndLogo(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry()
	svc := NewArtworkService(tmpDir, reg)

	gameDir := t.TempDir()
	heroImg := filepath.Join(gameDir, "my_custom_hero.jpg")
	_ = os.WriteFile(heroImg, []byte("hero-image-bytes"), 0644)
	logoImg := filepath.Join(gameDir, "my_custom_logo.png")
	_ = os.WriteFile(logoImg, []byte("logo-image-bytes"), 0644)

	g := domain.NewGame("Custom Hero Game", gameDir, "12345", "linux")

	// Apply hero
	art, err := svc.ApplyCustomArtwork(context.Background(), g, "hero", heroImg)
	if err != nil {
		t.Fatalf("apply custom hero: %v", err)
	}
	if art.LocalHero == "" || !util.FileExists(art.LocalHero) {
		t.Fatalf("expected LocalHero to exist at %s", art.LocalHero)
	}
	if g.CustomHeroURL != heroImg {
		t.Errorf("expected CustomHeroURL %s, got %s", heroImg, g.CustomHeroURL)
	}

	// Apply logo
	art, err = svc.ApplyCustomArtwork(context.Background(), g, "logo", logoImg)
	if err != nil {
		t.Fatalf("apply custom logo: %v", err)
	}
	if art.LocalLogo == "" || !util.FileExists(art.LocalLogo) {
		t.Fatalf("expected LocalLogo to exist at %s", art.LocalLogo)
	}
	if g.CustomLogoURL != logoImg {
		t.Errorf("expected CustomLogoURL %s, got %s", logoImg, g.CustomLogoURL)
	}
}

func TestArtworkService_ParameterizedResolution(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry()
	svc := NewArtworkService(tmpDir, reg)

	gameDir := t.TempDir()
	coverFile := filepath.Join(gameDir, "cover.png")
	bannerFile := filepath.Join(gameDir, "banner.png")
	heroFile := filepath.Join(gameDir, "hero.png")
	logoFile := filepath.Join(gameDir, "logo.png")
	iconFile := filepath.Join(gameDir, "icon.png")

	_ = os.WriteFile(coverFile, []byte("fake-cover"), 0644)
	_ = os.WriteFile(bannerFile, []byte("fake-banner"), 0644)
	_ = os.WriteFile(heroFile, []byte("fake-hero"), 0644)
	_ = os.WriteFile(logoFile, []byte("fake-logo"), 0644)
	_ = os.WriteFile(iconFile, []byte("fake-icon"), 0644)

	g := domain.NewGame("Parameterized Game", gameDir, "55555", "linux")
	art, err := svc.Resolve(context.Background(), g)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if art.LocalCover != coverFile {
		t.Errorf("expected cover %s, got %s", coverFile, art.LocalCover)
	}
	if art.LocalBanner != bannerFile {
		t.Errorf("expected banner %s, got %s", bannerFile, art.LocalBanner)
	}
	if art.LocalHero != heroFile {
		t.Errorf("expected hero %s, got %s", heroFile, art.LocalHero)
	}
	if art.LocalLogo != logoFile {
		t.Errorf("expected logo %s, got %s", logoFile, art.LocalLogo)
	}

	// Verify domain accessors
	coverLoc := g.GetAsset(domain.AssetCover)
	if coverLoc.LocalPath != coverFile {
		t.Errorf("expected GetAsset(cover) LocalPath %s, got %s", coverFile, coverLoc.LocalPath)
	}
}

func TestArtworkService_ClearOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry()
	svc := NewArtworkService(tmpDir, reg)

	gameDir := t.TempDir()
	g := domain.NewGame("Clear Game", gameDir, "77777", "linux")

	art, err := svc.ApplyCustomArtwork(context.Background(), g, "cover", "none")
	if err != nil {
		t.Fatalf("ApplyCustomArtwork clear failed: %v", err)
	}
	if art.LocalCover != "" || art.CoverURL != "" || g.CustomCoverURL != "none" {
		t.Errorf("expected cover to be cleared and custom_cover_url='none', got local=%q, custom=%q", art.LocalCover, g.CustomCoverURL)
	}
}

func TestRegistry_SearchStores(t *testing.T) {
	reg := NewRegistry()

	steamMock := &mockProvider{
		id:          "steam",
		displayName: "Steam",
		results: []SearchResult{
			{ID: "1091500", Store: "steam", Name: "Cyberpunk 2077"},
		},
	}
	gogMock := &mockProvider{
		id:          "gog",
		displayName: "GOG.com",
		results: []SearchResult{
			{ID: "1424355577", Store: "gog", Name: "Cyberpunk 2077"},
		},
	}
	sgdbMock := NewSGDBProvider("") // IsGameStore() == false

	reg.Register(steamMock)
	reg.Register(gogMock)
	reg.Register(sgdbMock)

	results, err := reg.SearchStores(context.Background(), "Cyberpunk")
	if err != nil {
		t.Fatalf("SearchStores failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results from store providers, got %d", len(results))
	}

	for _, res := range results {
		if res.Store == "steamgriddb" {
			t.Errorf("unexpected steamgriddb result in SearchStores: %+v", res)
		}
	}
}

func TestSGDBProvider_AssetTypeEndpoints(t *testing.T) {
	p := NewSGDBProvider("mock-key")
	if p.APIKey() != "mock-key" {
		t.Fatalf("expected api key")
	}

	// Verify that querying without api key returns an error
	noKeyProv := NewSGDBProvider("")
	if _, err := noKeyProv.QueryArtworkCandidates(context.Background(), ArtworkQuery{AssetType: "banner"}); err == nil {
		t.Errorf("expected error when API key is missing")
	}
}

func TestGOGProvider_Endpoints(t *testing.T) {
	p := NewGOGProvider()
	if p.ID() != "gog" || p.DisplayName() != "GOG.com" || !p.IsGameStore() {
		t.Fatalf("unexpected GOG provider properties")
	}

	results, err := p.Search(context.Background(), "Witcher")
	if err != nil {
		t.Fatalf("GOG search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected at least one search result for 'Witcher'")
	}
	if results[0].Store != "gog" || results[0].Name == "" || results[0].ID == "" {
		t.Errorf("invalid GOG search result: %+v", results[0])
	}

	meta, err := p.GetMetadata(context.Background(), results[0].ID)
	if err != nil {
		t.Fatalf("GOG GetMetadata failed: %v", err)
	}
	if meta.Name == "" || meta.Store != "gog" {
		t.Errorf("invalid GOG metadata: %+v", meta)
	}

	art, err := p.GetArtwork(context.Background(), results[0].ID)
	if err != nil {
		t.Fatalf("GOG GetArtwork failed: %v", err)
	}
	if art.CoverURL == "" && art.BannerURL == "" {
		t.Errorf("expected GOG artwork to have cover or banner URL, got %+v", art)
	}

	candidates, err := p.QueryArtworkCandidates(context.Background(), ArtworkQuery{
		GOGID:     results[0].ID,
		AssetType: "cover",
	})
	if err != nil {
		t.Fatalf("GOG QueryArtworkCandidates failed: %v", err)
	}
	if len(candidates) == 0 {
		t.Errorf("expected at least one cover candidate from GOG")
	}
}

func TestGOGProvider_NoFalsePositives(t *testing.T) {
	p := NewGOGProvider()

	// 1. Non-existent search query must return 0 results, not Witcher 3 recommendations
	results, err := p.Search(context.Background(), "NonExistentGameXYZ12345")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for non-existent game, got %d (first: %s)", len(results), results[0].Name)
	}

	// 2. Non-existent game ID must return an error, not Witcher 3
	_, err = p.GetMetadata(context.Background(), "999999999")
	if err == nil {
		t.Errorf("expected error for non-existent GOG ID, got nil")
	}

	// 3. Candidate query for unmatched title must return 0 candidates
	candidates, err := p.QueryArtworkCandidates(context.Background(), ArtworkQuery{
		Title:     "NonExistentGameXYZ12345",
		AssetType: "cover",
	})
	if err != nil {
		t.Fatalf("QueryArtworkCandidates error: %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates for non-existent game, got %d", len(candidates))
	}
}

func TestArtworkService_NoGOGDefaultFallback(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry()
	reg.Register(NewGOGProvider())
	svc := NewArtworkService(tmpDir, reg)

	gameDir := t.TempDir()
	// Game without GOG ID and without matching title
	g := domain.NewGame("Unmatched Standalone Game", gameDir, "", "linux")
	art, err := svc.Resolve(context.Background(), g)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Must NOT have populated Witcher 3 artwork!
	if art.CoverURL != "" || art.BannerURL != "" || art.HeroURL != "" {
		t.Errorf("expected empty artwork for unmatched game, got %+v", art)
	}
}

func TestRAWGProvider_Basics(t *testing.T) {
	p := NewRAWGProvider("")
	if p.ID() != "rawg" {
		t.Errorf("expected ID 'rawg', got %s", p.ID())
	}
	if p.DisplayName() != "RAWG Video Games Database" {
		t.Errorf("expected DisplayName 'RAWG Video Games Database', got %s", p.DisplayName())
	}
	caps := p.Capabilities()
	if !caps.CanSearch || !caps.CanFetchMetadata || !caps.CanProvideArtwork || caps.IsGameStore {
		t.Errorf("unexpected capabilities: %+v", caps)
	}
}

func TestRegistry_ListDescriptors(t *testing.T) {
	reg := NewRegistry()
	reg.Register(NewSteamProvider())
	reg.Register(NewGOGProvider())
	reg.Register(NewRAWGProvider(""))
	reg.Register(NewSGDBProvider(""))

	descriptors := reg.ListDescriptors()
	if len(descriptors) != 4 {
		t.Fatalf("expected 4 descriptors, got %d", len(descriptors))
	}

	foundRAWG := false
	for _, d := range descriptors {
		if d.ID == "rawg" {
			foundRAWG = true
			if d.Capabilities.IsGameStore {
				t.Errorf("expected rawg IsGameStore to be false")
			}
		}
	}
	if !foundRAWG {
		t.Errorf("rawg descriptor not found in list")
	}
}

func TestSteamProvider_Basics(t *testing.T) {
	p := NewSteamProvider()
	if p.ID() != "steam" {
		t.Errorf("expected ID 'steam', got %s", p.ID())
	}
	if p.DisplayName() != "Steam" {
		t.Errorf("expected DisplayName 'Steam', got %s", p.DisplayName())
	}
	caps := p.Capabilities()
	if !caps.CanSearch || !caps.CanFetchMetadata || !caps.CanProvideArtwork || !caps.IsGameStore {
		t.Errorf("unexpected capabilities for Steam provider: %+v", caps)
	}

	art, err := p.GetArtwork(context.Background(), "1091500")
	if err != nil {
		t.Fatalf("GetArtwork failed: %v", err)
	}
	if art.CoverURL == "" || art.BannerURL == "" || art.HeroURL == "" {
		t.Errorf("expected non-empty artwork URLs, got: %+v", art)
	}
}

func TestArtworkService_CrossAssetFallback(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry()
	svc := NewArtworkService(tmpDir, reg)

	gameArtDir := filepath.Join(tmpDir, "test-game")
	_ = os.MkdirAll(gameArtDir, 0755)
	heroFile := filepath.Join(gameArtDir, "hero.jpg")
	_ = os.WriteFile(heroFile, []byte("fake-hero-jpeg"), 0644)

	g := domain.NewGame("Test Game", t.TempDir(), "", "linux")
	g.LocalHero = heroFile

	art, err := svc.Resolve(context.Background(), g)
	if err != nil {
		t.Fatalf("resolve artwork: %v", err)
	}

	// Cover should fallback to heroFile
	if art.LocalCover == "" {
		t.Errorf("expected local cover fallback to be populated from hero, got empty")
	}
	if art.LocalCoverURL == "" {
		t.Errorf("expected local cover URL to be populated, got empty")
	}
}

func TestRAWGProvider_ExtractLinkedStoreIDs(t *testing.T) {
	g := &rawgGame{
		ID:   9767,
		Name: "Hollow Knight",
		Stores: []rawgStoreEntry{
			{
				ID:  1,
				URL: "https://store.steampowered.com/app/367520/",
			},
			{
				ID:  2,
				URL: "https://www.gog.com/game/hollow_knight",
			},
		},
	}

	steamID, gogSlug := ExtractLinkedStoreIDs(g)
	if steamID != 367520 {
		t.Errorf("expected steam appid 367520, got %d", steamID)
	}
	if gogSlug != "hollow_knight" {
		t.Errorf("expected gog slug 'hollow_knight', got '%s'", gogSlug)
	}
}

func TestArtworkService_SteamGridProxyFallback(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search/IndieGame" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf("%q", server.URL+"/indie_cover.jpg")))
			return
		}
		if r.URL.Path == "/indie_cover.jpg" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fake-cover-bytes"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	proxy := NewSteamGridProxyProvider().WithCustomClient(server.Client(), server.URL+"/search")
	reg := NewRegistry()
	reg.Register(proxy)

	tmpDir := t.TempDir()
	svc := NewArtworkService(tmpDir, reg)
	svc.httpClient = server.Client()

	gameDir := t.TempDir()
	g := domain.NewGame("IndieGame", gameDir, "", "win64")

	art, err := svc.Resolve(context.Background(), g)
	if err != nil {
		t.Fatalf("resolve artwork: %v", err)
	}

	if art.LocalCover == "" {
		t.Errorf("expected proxy cover to be downloaded and populated, got empty")
	}
	if !util.FileExists(art.LocalCover) {
		t.Errorf("expected downloaded cover file to exist at %s", art.LocalCover)
	}
}

func TestArtworkService_PEIconExtraction(t *testing.T) {
	hostExe := "/home/staernid/.local/share/Steam/legacycompat/SteamService.exe"
	if !util.FileExists(hostExe) {
		t.Skip("SteamService.exe not available on host, skipping live PE icon extraction test")
	}

	reg := NewRegistry()
	tmpDir := t.TempDir()
	svc := NewArtworkService(tmpDir, reg)

	gameDir := t.TempDir()
	g := domain.NewGame("Custom Game", gameDir, "", "win64")
	g.ExePath = hostExe

	art, err := svc.Resolve(context.Background(), g)
	if err != nil {
		t.Fatalf("resolve artwork: %v", err)
	}

	if art.LocalIcon == "" {
		t.Errorf("expected LocalIcon to be extracted from PE executable, got empty")
	}
	if !util.FileExists(art.LocalIcon) {
		t.Errorf("expected extracted icon file to exist at %s", art.LocalIcon)
	}
}









