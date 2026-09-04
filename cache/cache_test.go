package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/staernid/gamux/domain"
)

func TestCache_AppNames(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gamux_cache_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	SetCacheDir(tmpDir)

	if _, ok := GetAppName("1091500"); ok {
		t.Errorf("expected cache miss for unadded appid")
	}

	SaveAppName("1091500", "Cyberpunk 2077")

	name, ok := GetAppName("1091500")
	if !ok || name != "Cyberpunk 2077" {
		t.Errorf("expected Cyberpunk 2077, got %q (ok=%v)", name, ok)
	}

	// Test persistence across memory clear
	nameMu.Lock()
	nameMemoryCache = nil
	namesLoaded = false
	nameMu.Unlock()

	name, ok = GetAppName("1091500")
	if !ok || name != "Cyberpunk 2077" {
		t.Errorf("expected reload from disk Cyberpunk 2077, got %q (ok=%v)", name, ok)
	}
}

func TestCache_AppDetails(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gamux_details_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	SetCacheDir(tmpDir)

	payload := []byte(`{"name":"Portal 2","steam_appid":620}`)
	if err := SaveAppDetails("620", payload); err != nil {
		t.Fatalf("failed to save app details: %v", err)
	}

	data, ok := GetAppDetails("620")
	if !ok || string(data) != string(payload) {
		t.Errorf("expected %s, got %s (ok=%v)", string(payload), string(data), ok)
	}
}

func TestCache_GitHubRelease(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gamux_gh_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	SetCacheDir(tmpDir)

	payload := []byte(`{"tag_name":"v1.2.3"}`)
	if err := SaveGitHubRelease("gbe", payload); err != nil {
		t.Fatalf("failed to save github release: %v", err)
	}

	data, ok := GetGitHubRelease("gbe")
	if !ok || string(data) != string(payload) {
		t.Errorf("expected %s, got %s (ok=%v)", string(payload), string(data), ok)
	}
}

func TestCache_RevoBDAndSteamDepotDB(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gamux_manifest_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	SetCacheDir(tmpDir)

	revoData := []byte("fake-revo-zip-content")
	if err := SaveRevoBDZIP(1091500, revoData); err != nil {
		t.Fatalf("failed to save revobd zip: %v", err)
	}
	data, ok := GetRevoBDZIP(1091500)
	if !ok || string(data) != string(revoData) {
		t.Errorf("revobd zip mismatch")
	}

	depotKeys := []byte(`{"1091501":"key"}`)
	if err := SaveSteamDepotDB(depotKeys); err != nil {
		t.Fatalf("failed to save steamdepotdb: %v", err)
	}
	ddata, ok := GetSteamDepotDB()
	if !ok || string(ddata) != string(depotKeys) {
		t.Errorf("steamdepotdb mismatch")
	}
}

func TestCache_SGDBArtwork(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gamux_sgdb_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	SetCacheDir(tmpDir)

	artPayload := []byte(`{"cover_url":"https://example.com/cover.jpg"}`)
	if err := SaveSGDBArtwork("1091500", artPayload); err != nil {
		t.Fatalf("failed to save sgdb artwork: %v", err)
	}

	data, ok := GetSGDBArtwork("1091500")
	if !ok || string(data) != string(artPayload) {
		t.Errorf("sgdb artwork mismatch")
	}
}

func TestCache_WarmLibraryCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gamux_games_cache_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	SetCacheDir(tmpDir)

	gameDir := "/tmp/TestGameDir"
	if _, ok := GetCachedGame(gameDir, 1000, 2000); ok {
		t.Errorf("expected cache miss for unadded game")
	}

	game := &domain.Game{
		GameRecord: domain.GameRecord{
			Name:    "Cached Test Game",
			GameDir: gameDir,
			AppID:   "12345",
		},
	}

	if err := SaveCachedGame(gameDir, 1000, 2000, game); err != nil {
		t.Fatalf("failed to save cached game: %v", err)
	}

	// 1. Exact timestamp match
	cached, ok := GetCachedGame(gameDir, 1000, 2000)
	if !ok || cached == nil || cached.Name != "Cached Test Game" {
		t.Fatalf("expected to retrieve cached game, got %+v (ok=%v)", cached, ok)
	}

	// 2. Modified directory mtime -> cache miss
	if _, ok := GetCachedGame(gameDir, 1001, 2000); ok {
		t.Errorf("expected cache miss on changed dirMtime")
	}

	// 3. Modified profile mtime -> cache miss
	if _, ok := GetCachedGame(gameDir, 1000, 2001); ok {
		t.Errorf("expected cache miss on changed profileMtime")
	}

	// 4. Invalidate cache
	InvalidateGameCache(gameDir)
	if _, ok := GetCachedGame(gameDir, 1000, 2000); ok {
		t.Errorf("expected cache miss after InvalidateGameCache")
	}
}

func TestCache_NewsItems_InMemory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gamux_news_mem_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	SetCacheDir(tmpDir)

	items := []byte(`[{"title":"Patch 1.5"},{"title":"Patch 1.4"}]`)
	SaveNewsItems("99999", items)

	// Should hit in-memory cache
	data, ok := GetNewsItems("99999")
	if !ok {
		t.Fatal("expected cache hit from in-memory news items cache")
	}
	if string(data) != string(items) {
		t.Errorf("in-memory news items mismatch: got %s", string(data))
	}

	// Clear in-memory cache, should fall back to disk
	newsItemsMu.Lock()
	newsItemsMemoryCache = make(map[string][]byte)
	newsItemsMu.Unlock()

	data, ok = GetNewsItems("99999")
	if !ok {
		t.Fatal("expected cache hit from disk news items cache after memory clear")
	}
	if string(data) != string(items) {
		t.Errorf("disk news items mismatch: got %s", string(data))
	}
}

func TestCache_Prune(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gamux_prune_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	SetCacheDir(tmpDir)

	// Create a fresh (valid) GitHub release cache entry
	freshPayload := []byte(`{"tag_name":"v2.0.0"}`)
	if err := SaveGitHubRelease("fresh_release", freshPayload); err != nil {
		t.Fatalf("failed to save fresh github release: %v", err)
	}

	// Create an expired GitHub release cache entry (timestamp 3 hours ago, TTL=1h, prune threshold=2h)
	expiredWrapper := TTLWrapper{
		Timestamp: time.Now().Add(-3 * time.Hour),
		Payload:   json.RawMessage(`{"tag_name":"v0.1.0"}`),
	}
	expiredData, _ := json.Marshal(expiredWrapper)
	expiredPath := filepath.Join(tmpDir, "github", "expired_release.json")
	if err := os.WriteFile(expiredPath, expiredData, 0644); err != nil {
		t.Fatalf("failed to write expired cache file: %v", err)
	}

	// Create an expired hubcap ZIP (mtime 61 days ago, TTL=30d, prune threshold=60d)
	expiredZipPath := filepath.Join(tmpDir, "hubcap", "999999.zip")
	if err := os.WriteFile(expiredZipPath, []byte("old-zip"), 0644); err != nil {
		t.Fatalf("failed to write expired zip: %v", err)
	}
	oldTime := time.Now().Add(-61 * 24 * time.Hour)
	_ = os.Chtimes(expiredZipPath, oldTime, oldTime)

	// Create a fresh hubcap ZIP
	freshZipPath := filepath.Join(tmpDir, "hubcap", "111111.zip")
	if err := os.WriteFile(freshZipPath, []byte("new-zip"), 0644); err != nil {
		t.Fatalf("failed to write fresh zip: %v", err)
	}

	removed := Prune()

	if removed != 2 {
		t.Errorf("expected 2 files pruned, got %d", removed)
	}

	// Verify fresh entries survived
	if _, ok := GetGitHubRelease("fresh_release"); !ok {
		t.Error("fresh github release should have survived pruning")
	}
	if _, err := os.Stat(freshZipPath); os.IsNotExist(err) {
		t.Error("fresh hubcap zip should have survived pruning")
	}

	// Verify expired entries were removed
	if _, err := os.Stat(expiredPath); !os.IsNotExist(err) {
		t.Error("expired github release should have been pruned")
	}
	if _, err := os.Stat(expiredZipPath); !os.IsNotExist(err) {
		t.Error("expired hubcap zip should have been pruned")
	}
}

func TestCache_DisableCache_Atomic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gamux_disable_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	SetCacheDir(tmpDir)

	SetDisabled(false)
	if IsDisabled() {
		t.Error("expected cache to be enabled")
	}

	SaveAppName("100", "Test Game")
	if _, ok := GetAppName("100"); !ok {
		t.Error("expected cache hit when enabled")
	}

	SetDisabled(true)
	defer SetDisabled(false)

	if _, ok := GetAppName("100"); ok {
		t.Error("expected cache miss when disabled")
	}
}
