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
)

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

	// Run ApplyGBE targeting gameDir in portable mode with dry-run
	err := ApplyGBE(context.Background(), cfg, gameDir, "linux", "12345", true, true, "", "")


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

type mockRoundTripper func(req *http.Request) (*http.Response, error)

func (f mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGenerateAchievementsJSON_DryRun(t *testing.T) {
	oldTransport := steam.HTTPClient.Transport
	defer func() {
		steam.HTTPClient.Transport = oldTransport
	}()

	steam.HTTPClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
		jsonResp := `{"game": {"availableGameStats": {"achievements": [{"name": "ACH1", "displayName": "Ach 1", "icon": "https://cdn.com/ach1.png"}]}}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(jsonResp)),
			Header:     make(http.Header),
		}, nil
	})

	tmpDir := t.TempDir()
	err := GenerateAchievementsJSON(context.Background(), 12345, "MOCKKEY", tmpDir, true)
	if err != nil {
		t.Fatalf("GenerateAchievementsJSON dry-run failed: %v", err)
	}
}
