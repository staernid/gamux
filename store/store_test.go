package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/util"
)

type mockProvider struct {
	id          string
	displayName string
	results     []SearchResult
}

func (m *mockProvider) ID() string          { return m.id }
func (m *mockProvider) DisplayName() string { return m.displayName }
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
	sp := NewSteamProvider()
	opts, err := sp.QueryArtworkCandidates(context.Background(), ArtworkQuery{
		Title:     "Half-Life",
		AssetType: "cover",
	})
	if err != nil {
		t.Fatalf("QueryArtworkCandidates with title failed: %v", err)
	}
	if len(opts) == 0 {
		t.Fatalf("expected resolved cover options for Half-Life")
	}
	if opts[0].URL == "https://cdn.akamai.steamstatic.com/steam/apps/Half-Life/library_600x900_2x.jpg" {
		t.Fatalf("expected resolved numeric AppID in URL, but got raw title string")
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



