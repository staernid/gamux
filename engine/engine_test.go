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
	"github.com/staernid/gamux/util"
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

func snapshotDir(t *testing.T, root string) map[string]string {
	t.Helper()
	hashes := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		hash, err := util.GetHash(path)
		if err != nil {
			return err
		}
		hashes[rel] = hash
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotDir failed: %v", err)
	}
	return hashes
}

func TestEngine_DryRunZeroMutationInvariant(t *testing.T) {
	oldTransport := steam.HTTPClient.Transport
	defer func() { steam.HTTPClient.Transport = oldTransport }()

	steam.HTTPClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
			Header:     make(http.Header),
		}, nil
	})

	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "DryRunGame")
	_ = os.MkdirAll(filepath.Join(gameDir, "[Manifests]"), 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "[Manifests]", "appmanifest_12345.acf"), []byte("\"AppState\"\n{\n\t\"appid\"\t\"12345\"\n\t\"name\"\t\"DryRunGame\"\n}"), 0644)
	_ = os.WriteFile(filepath.Join(gameDir, "game.exe"), []byte("executable binary payload"), 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "steam_api64.dll"), []byte("original dll payload"), 0644)

	baseline := snapshotDir(t, gameDir)

	eng := New(config.DefaultConfig())
	opts := ProcessOptions{
		Path:         gameDir,
		DryRun:       true,
		ApplyGBE:     true,
		Portable:     true,
		NormalizeDir: false,
		NoSteamless:  true,
	}

	res, err := eng.ProcessGame(context.Background(), opts)
	if err != nil {
		t.Fatalf("ProcessGame dry-run failed: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ProcessResult")
	}

	postRun := snapshotDir(t, gameDir)

	for k, v := range baseline {
		if postRun[k] != v {
			t.Errorf("Dry-run mutation detected on file %s: expected hash %s, got %s", k, v, postRun[k])
		}
	}
	for k := range postRun {
		if _, ok := baseline[k]; !ok {
			t.Errorf("Dry-run created unexpected file: %s", k)
		}
	}
}

func TestEngine_SyncAndRollbackInvariant(t *testing.T) {
	oldTransport := steam.HTTPClient.Transport
	defer func() { steam.HTTPClient.Transport = oldTransport }()

	steam.HTTPClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
			Header:     make(http.Header),
		}, nil
	})

	tmpDir := t.TempDir()
	gbeDir := filepath.Join(tmpDir, "gbe_mock")
	gbeWin64 := filepath.Join(gbeDir, "win_release", "experimental", "x64")
	_ = os.MkdirAll(gbeWin64, 0755)
	_ = os.WriteFile(filepath.Join(gbeWin64, "steam_api64.dll"), []byte("gbe_emulator_patched_binary"), 0644)

	cfg := config.DefaultConfig()
	cfg.GbeDir = gbeDir

	gameDir := filepath.Join(tmpDir, "TestRollbackGame")
	_ = os.MkdirAll(filepath.Join(gameDir, "[Manifests]"), 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "[Manifests]", "appmanifest_99999.acf"), []byte("\"AppState\"\n{\n\t\"appid\"\t\"99999\"\n\t\"name\"\t\"TestRollbackGame\"\n}"), 0644)
	_ = os.WriteFile(filepath.Join(gameDir, "game.exe"), []byte("original game exe"), 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "steam_api64.dll"), []byte("original steam api dll"), 0644)

	baseline := snapshotDir(t, gameDir)

	eng := New(cfg)
	opts := ProcessOptions{
		Path:         gameDir,
		DryRun:       false,
		ApplyGBE:     true,
		Portable:     true,
		NormalizeDir: false,
		NoSteamless:  true,
	}

	res, err := eng.ProcessGame(context.Background(), opts)
	if err != nil {
		t.Fatalf("ProcessGame failed: %v", err)
	}
	if !res.Patched {
		t.Fatalf("expected game to be patched: errors = %v", res.Errors)
	}

	// Verify steam_api64.dll was patched
	patchedData, err := os.ReadFile(filepath.Join(gameDir, "steam_api64.dll"))
	if err != nil {
		t.Fatal(err)
	}
	if string(patchedData) != "gbe_emulator_patched_binary" {
		t.Fatalf("expected patched dll content, got %s", string(patchedData))
	}

	// Perform rollback
	if err := eng.Rollback(context.Background(), gameDir, false); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	postRollback := snapshotDir(t, gameDir)

	for k, v := range baseline {
		if postRollback[k] != v {
			t.Errorf("Rollback state mismatch on file %s: expected baseline hash %s, got %s", k, v, postRollback[k])
		}
	}
	for k := range postRollback {
		if _, ok := baseline[k]; !ok {
			t.Errorf("Rollback left unexpected leftover file: %s", k)
		}
	}
}




