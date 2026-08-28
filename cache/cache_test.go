package cache

import (
	"os"
	"testing"
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
