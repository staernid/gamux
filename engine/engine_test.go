package engine

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
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



func TestInspectStatus_Original(t *testing.T) {
	oldTransport := steam.HTTPClient.Transport
	defer func() { steam.HTTPClient.Transport = oldTransport }()

	steam.HTTPClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"appnews": {"newsitems": []}}`)),
			Header:     make(http.Header),
		}, nil
	})

	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "TestGame")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatal(err)
	}

	acfContent := "\"AppState\"\n{\n\t\"appid\"\t\"999\"\n\t\"name\"\t\"TestGame\"\n}"
	if err := os.WriteFile(filepath.Join(gameDir, "appmanifest_999.acf"), []byte(acfContent), 0644); err != nil {
		t.Fatal(err)
	}

	eng := New(config.DefaultConfig())
	status, err := eng.InspectStatus(context.Background(), gameDir)
	if err != nil {
		t.Fatalf("InspectStatus failed: %v", err)
	}

	if status.Name != "TestGame" {
		t.Errorf("expected Name 'TestGame', got %s", status.Name)
	}
	if status.State != "Original" {
		t.Errorf("expected State 'Original', got %s", status.State)
	}
}

func TestInspectStatus_PortablePatchedAndRollback(t *testing.T) {
	oldTransport := steam.HTTPClient.Transport
	defer func() { steam.HTTPClient.Transport = oldTransport }()

	steam.HTTPClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"appnews": {"newsitems": []}}`)),
			Header:     make(http.Header),
		}, nil
	})

	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "PatchedGame")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatal(err)
	}

	acfContent := "\"AppState\"\n{\n\t\"appid\"\t\"888\"\n\t\"name\"\t\"PatchedGame\"\n}"
	_ = os.WriteFile(filepath.Join(gameDir, "appmanifest_888.acf"), []byte(acfContent), 0644)

	// Create fake library and backup file
	libPath := filepath.Join(gameDir, "libsteam_api.so")
	_ = os.WriteFile(libPath, []byte("patched_lib"), 0755)

	backupPath := filepath.Join(gameDir, "libsteam_api.so.20260802-120000.ORIGINAL")
	_ = os.WriteFile(backupPath, []byte("original_lib"), 0755)

	// Create steam_settings directory
	settingsDir := filepath.Join(gameDir, "steam_settings")
	_ = os.MkdirAll(settingsDir, 0755)
	_ = os.WriteFile(filepath.Join(settingsDir, "configs.app.ini"), []byte("[app]"), 0644)

	eng := New(config.DefaultConfig())
	status, err := eng.InspectStatus(context.Background(), gameDir)
	if err != nil {
		t.Fatalf("InspectStatus failed: %v", err)
	}

	if status.State != "Portable-Patched" {
		t.Errorf("expected State 'Portable-Patched', got %s", status.State)
	}
	if len(status.OriginalBackups) != 1 {
		t.Errorf("expected 1 backup, got %d", len(status.OriginalBackups))
	}

	// Test Rollback
	err = eng.Rollback(context.Background(), gameDir, false)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Check restored content
	restoredData, err := os.ReadFile(libPath)
	if err != nil {
		t.Fatalf("failed to read restored lib: %v", err)
	}
	if string(restoredData) != "original_lib" {
		t.Errorf("expected restored content 'original_lib', got %s", string(restoredData))
	}

	// Check settings dir removed
	if _, err := os.Stat(settingsDir); !os.IsNotExist(err) {
		t.Errorf("expected steam_settings to be removed")
	}
}

type mockRoundTripper func(req *http.Request) (*http.Response, error)

func (f mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestProcessGame_NormalizeAndSynthesizeACF(t *testing.T) {
	oldTransport := steam.HTTPClient.Transport
	defer func() { steam.HTTPClient.Transport = oldTransport }()

	steam.HTTPClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
		jsonResp := `{"1091500": {"success": true, "data": {"name": "Cyberpunk 2077"}}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
			Header:     make(http.Header),
		}, nil
	})

	tmpDir := t.TempDir()
	rawGameDir := filepath.Join(tmpDir, "Cyberpunk.2077.v2.12-P2P")
	if err := os.MkdirAll(rawGameDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create steam_appid.txt so AppID is recognized as 1091500
	_ = os.WriteFile(filepath.Join(rawGameDir, "steam_appid.txt"), []byte("1091500"), 0644)
	_ = os.WriteFile(filepath.Join(rawGameDir, "Cyberpunk2077.exe"), []byte("fake exe"), 0755)

	eng := New(config.DefaultConfig())
	opts := ProcessOptions{
		Path:         rawGameDir,
		NormalizeDir: true,
		NoSteamless:  true,
	}

	res, err := eng.ProcessGame(context.Background(), opts)
	if err != nil {
		t.Fatalf("ProcessGame failed: %v", err)
	}

	expectedNormalizedDir := filepath.Join(tmpDir, "Cyberpunk 2077")
	if res.Info.GameDir != expectedNormalizedDir {
		t.Errorf("expected normalized game dir %s, got %s", expectedNormalizedDir, res.Info.GameDir)
	}

	// Verify synthesized ACF manifest exists inside [Manifests]
	synthesizedACF := filepath.Join(expectedNormalizedDir, "[Manifests]", "appmanifest_1091500.acf")
	if _, err := os.Stat(synthesizedACF); os.IsNotExist(err) {
		t.Errorf("expected synthesized ACF manifest at %s", synthesizedACF)
	}
}

func TestNotifyLaunch(t *testing.T) {
	oldTransport := steam.HTTPClient.Transport
	defer func() { steam.HTTPClient.Transport = oldTransport }()

	steam.HTTPClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"appnews": {"newsitems": []}}`)),
			Header:     make(http.Header),
		}, nil
	})

	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "CleanGame")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatal(err)
	}

	_ = os.WriteFile(filepath.Join(gameDir, "appmanifest_123.acf"), []byte("\"AppState\"\n{\n\t\"appid\"\t\"123\"\n\t\"name\"\t\"CleanGame\"\n}"), 0644)

	eng := New(config.DefaultConfig())
	err := eng.NotifyLaunch(context.Background(), gameDir)
	if err != nil {
		t.Fatalf("NotifyLaunch failed: %v", err)
	}
}



