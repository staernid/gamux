package gbe

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
	restore := testutil.SilenceLogging()
	code := m.Run()
	restore()
	os.Exit(code)
}


func TestApplyGBE_InvalidPlatform(t *testing.T) {
	err := ApplyGBE(context.Background(), config.DefaultConfig(), ".", "invalid_platform", "123", true, false, "", "")
	if err == nil {
		t.Fatal("expected error for invalid platform")
	}
}

func TestApplyGBE_ExplicitTargetDirectory(t *testing.T) {
	// Create mock GBE directory structure
	tmpDir := t.TempDir()
	gbeDir := filepath.Join(tmpDir, "gbe_fork")
	linuxGbe := filepath.Join(gbeDir, "linux_release", "experimental", "x64")
	if err := os.MkdirAll(linuxGbe, 0755); err != nil {
		t.Fatal(err)
	}

	mockSourceLib := filepath.Join(linuxGbe, "libsteam_api.so")
	if err := os.WriteFile(mockSourceLib, []byte("emulated_gbe_lib"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create mock game directory in a completely separate folder
	gameDir := filepath.Join(tmpDir, "MyGame")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatal(err)
	}
	targetLib := filepath.Join(gameDir, "libsteam_api.so")
	if err := os.WriteFile(targetLib, []byte("original_steam_lib"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		GbeDir: gbeDir,
	}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	ctx := steam.WithHTTPClient(context.Background(), mockClient)

	// Run ApplyGBE targeting gameDir in portable mode with dry-run
	err := ApplyGBE(ctx, cfg, gameDir, "linux", "12345", true, true, "", "")

	if err != nil {
		t.Fatalf("ApplyGBE failed: %v", err)
	}

	// Verify original file remained intact due to dry-run
	data, err := os.ReadFile(targetLib)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original_steam_lib" {
		t.Errorf("expected original lib content, got %s", string(data))
	}
}

func TestGenerateAchievementsJSON_DryRun(t *testing.T) {
	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			jsonResp := `{"game": {"availableGameStats": {"achievements": [{"name": "ACH1", "displayName": "Ach 1", "icon": "https://cdn.com/ach1.png"}]}}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(jsonResp)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	ctx := steam.WithHTTPClient(context.Background(), mockClient)

	tmpDir := t.TempDir()
	err := GenerateAchievementsJSON(ctx, 12345, "MOCKKEY", tmpDir, true)
	if err != nil {
		t.Fatalf("GenerateAchievementsJSON dry-run failed: %v", err)
	}
}

func TestConfigureCustomSettings(t *testing.T) {
	tmpDir := t.TempDir()
	steamSettingsDir := filepath.Join(tmpDir, "steam_settings")

	cfg := &config.Config{
		Language:       "german",
		DisableOverlay: true,
		OfflineMode:    true,
	}

	supportedLanguages := []string{"english", "german", "french"}

	err := ConfigureCustomSettings(steamSettingsDir, cfg, supportedLanguages, false)
	if err != nil {
		t.Fatalf("ConfigureCustomSettings failed: %v", err)
	}

	// Verify force_language.txt
	langData, err := os.ReadFile(filepath.Join(steamSettingsDir, "force_language.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(langData)) != "german" {
		t.Errorf("expected language 'german', got %q", string(langData))
	}

	// Verify force_account_name.txt & force_steamid.txt are NOT generated per-game
	if _, err := os.Stat(filepath.Join(steamSettingsDir, "force_account_name.txt")); !os.IsNotExist(err) {
		t.Errorf("expected force_account_name.txt not to exist")
	}
	if _, err := os.Stat(filepath.Join(steamSettingsDir, "force_steamid.txt")); !os.IsNotExist(err) {
		t.Errorf("expected force_steamid.txt not to exist")
	}

	// Verify disable_overlay.txt & offline.txt
	if _, err := os.Stat(filepath.Join(steamSettingsDir, "disable_overlay.txt")); os.IsNotExist(err) {
		t.Error("expected disable_overlay.txt to exist")
	}
	if _, err := os.Stat(filepath.Join(steamSettingsDir, "offline.txt")); os.IsNotExist(err) {
		t.Error("expected offline.txt to exist")
	}
}

func TestConfigureCustomSettings_FallbackUnsupportedLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	steamSettingsDir := filepath.Join(tmpDir, "steam_settings")

	cfg := &config.Config{
		Language: "japanese",
	}

	supportedLanguages := []string{"english", "german"} // Japanese not supported

	err := ConfigureCustomSettings(steamSettingsDir, cfg, supportedLanguages, false)
	if err != nil {
		t.Fatalf("ConfigureCustomSettings failed: %v", err)
	}

	langData, _ := os.ReadFile(filepath.Join(steamSettingsDir, "force_language.txt"))
	if strings.TrimSpace(string(langData)) != "english" {
		t.Errorf("expected fallback language 'english', got %q", string(langData))
	}
}

func TestConfigureCustomSettings_AutoLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	steamSettingsDir := filepath.Join(tmpDir, "steam_settings")
	_ = os.MkdirAll(steamSettingsDir, 0755)

	// Pre-existing force_language.txt should be removed on "auto"
	_ = os.WriteFile(filepath.Join(steamSettingsDir, "force_language.txt"), []byte("english"), 0644)

	cfg := &config.Config{
		Language: "auto",
	}

	err := ConfigureCustomSettings(steamSettingsDir, cfg, []string{"english", "german"}, false)
	if err != nil {
		t.Fatalf("ConfigureCustomSettings failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(steamSettingsDir, "force_language.txt")); !os.IsNotExist(err) {
		t.Errorf("expected force_language.txt to be removed on auto language")
	}
}
