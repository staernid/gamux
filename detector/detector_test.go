package detector

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseACF(t *testing.T) {
	rawACF := `"AppState"
{
	"appid"		"2088570"
	"name"		"Tiny Rogues"
	"installdir"	"Tiny Rogues"
	"buildid"	"20241007"
}`

	data, err := ParseACF(strings.NewReader(rawACF))
	if err != nil {
		t.Fatalf("ParseACF failed: %v", err)
	}

	if data.AppID != "2088570" {
		t.Errorf("expected AppID 2088570, got %s", data.AppID)
	}
	if data.Name != "Tiny Rogues" {
		t.Errorf("expected Name 'Tiny Rogues', got %s", data.Name)
	}
	if data.InstallDir != "Tiny Rogues" {
		t.Errorf("expected InstallDir 'Tiny Rogues', got %s", data.InstallDir)
	}
}

func TestDetect_ManifestSubfolder(t *testing.T) {
	tmpDir := t.TempDir()
	manifestDir := filepath.Join(tmpDir, "[Manifests]")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatal(err)
	}

	acfPath := filepath.Join(manifestDir, "appmanifest_12345.acf")
	acfContent := `"AppState"
{
	"appid" "12345"
	"name" "Test Game"
	"installdir" "Test Game"
}`
	if err := os.WriteFile(acfPath, []byte(acfContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create fake target library
	libPath := filepath.Join(tmpDir, "libsteam_api.so")
	if err := os.WriteFile(libPath, []byte("fake lib"), 0755); err != nil {
		t.Fatal(err)
	}

	info, err := Detect(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.AppID != "12345" {
		t.Errorf("expected AppID 12345, got %s", info.AppID)
	}
	if info.Name != "Test Game" {
		t.Errorf("expected Name 'Test Game', got %s", info.Name)
	}
	if info.Platform != "linux" {
		t.Errorf("expected Platform 'linux', got %s", info.Platform)
	}
}

func TestDetect_SteamSubfolder(t *testing.T) {
	tmpDir := t.TempDir()
	steamDir := filepath.Join(tmpDir, "[Steam]")
	if err := os.MkdirAll(steamDir, 0755); err != nil {
		t.Fatal(err)
	}

	acfPath := filepath.Join(steamDir, "appmanifest_67890.acf")
	acfContent := `"AppState"
{
	"appid" "67890"
	"name" "Hades Game"
}`
	if err := os.WriteFile(acfPath, []byte(acfContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create fake win64 dll
	x64Dir := filepath.Join(tmpDir, "x64")
	_ = os.MkdirAll(x64Dir, 0755)
	libPath := filepath.Join(x64Dir, "steam_api64.dll")
	_ = os.WriteFile(libPath, []byte("fake dll"), 0755)

	info, err := Detect(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.AppID != "67890" {
		t.Errorf("expected AppID 67890, got %s", info.AppID)
	}
	if info.Platform != "win64" {
		t.Errorf("expected Platform 'win64', got %s", info.Platform)
	}
}

func TestConsolidateManifests(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate Pattern C: steamapps/appmanifest_111.acf + steamapps/common/MyGame
	steamappsDir := filepath.Join(tmpDir, "steamapps")
	commonDir := filepath.Join(steamappsDir, "common", "MyGame")
	_ = os.MkdirAll(commonDir, 0755)

	acfPath := filepath.Join(steamappsDir, "appmanifest_111.acf")
	acfContent := `"AppState"
{
	"appid" "111"
	"name" "MyGame"
	"installdir" "MyGame"
}`
	_ = os.WriteFile(acfPath, []byte(acfContent), 0644)

	info, err := Detect(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if !info.RawDepotLayout {
		t.Error("expected RawDepotLayout to be true")
	}

	if err := ConsolidateManifests(info, false); err != nil {
		t.Fatalf("ConsolidateManifests failed: %v", err)
	}

	// Verify [Steam]/appmanifest_111.acf exists inside MyGame
	consolidatedACF := filepath.Join(info.GameDir, "[Steam]", "appmanifest_111.acf")
	if _, err := os.Stat(consolidatedACF); os.IsNotExist(err) {
		t.Errorf("expected consolidated ACF at %s", consolidatedACF)
	}

	t.Run("no manifest path does not create [Steam] directory", func(t *testing.T) {
		gameDir := t.TempDir()
		emptyInfo := &GameInfo{GameDir: gameDir}
		if err := ConsolidateManifests(emptyInfo, false); err != nil {
			t.Fatalf("ConsolidateManifests failed: %v", err)
		}
		steamDir := filepath.Join(gameDir, "[Steam]")
		if _, err := os.Stat(steamDir); !os.IsNotExist(err) {
			t.Errorf("expected [Steam] directory to not exist, but found at %s", steamDir)
		}
	})
}
