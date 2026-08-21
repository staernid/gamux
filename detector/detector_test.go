package detector

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
	restore := testutil.SilenceLogging()
	code := m.Run()
	restore()
	os.Exit(code)
}

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

	if err := ConsolidateManifests(info); err != nil {
		t.Fatalf("ConsolidateManifests failed: %v", err)
	}

	// Verify [Manifests]/appmanifest_111.acf exists inside MyGame
	consolidatedACF := filepath.Join(info.GameDir, "[Manifests]", "appmanifest_111.acf")
	if _, err := os.Stat(consolidatedACF); os.IsNotExist(err) {
		t.Errorf("expected consolidated ACF at %s", consolidatedACF)
	}

	t.Run("no manifest path does not create [Manifests] directory", func(t *testing.T) {
		gameDir := t.TempDir()
		emptyInfo := &GameInfo{GameDir: gameDir}
		if err := ConsolidateManifests(emptyInfo); err != nil {
			t.Fatalf("ConsolidateManifests failed: %v", err)
		}
		manifestsDir := filepath.Join(gameDir, "[Manifests]")
		if _, err := os.Stat(manifestsDir); !os.IsNotExist(err) {
			t.Errorf("expected [Manifests] directory to not exist, but found at %s", manifestsDir)
		}
	})

	t.Run("consolidates manifest into [Manifests]", func(t *testing.T) {
		gameDir := t.TempDir()
		sourceDir := filepath.Join(gameDir, "source_manifests")
		_ = os.MkdirAll(sourceDir, 0755)
		_ = os.WriteFile(filepath.Join(sourceDir, "appmanifest_2088570.acf"), []byte("test"), 0644)
		_ = os.WriteFile(filepath.Join(sourceDir, "2088571_1234.manifest"), []byte("test"), 0644)

		info := &GameInfo{
			GameDir:      gameDir,
			ManifestPath: filepath.Join(sourceDir, "appmanifest_2088570.acf"),
		}

		if err := ConsolidateManifests(info); err != nil {
			t.Fatalf("ConsolidateManifests failed: %v", err)
		}

		// Verify [Manifests] has both files
		manifestsDir := filepath.Join(gameDir, "[Manifests]")
		if _, err := os.Stat(filepath.Join(manifestsDir, "appmanifest_2088570.acf")); os.IsNotExist(err) {
			t.Error("expected appmanifest in [Manifests]")
		}
		if _, err := os.Stat(filepath.Join(manifestsDir, "2088571_1234.manifest")); os.IsNotExist(err) {
			t.Error("expected .manifest in [Manifests]")
		}
	})
}

func TestGenerateACFContent(t *testing.T) {
	data := &ACFData{
		AppID:      "1091500",
		Name:       "Cyberpunk 2077",
		InstallDir: "Cyberpunk 2077",
		BuildID:    "12345678",
	}

	content := GenerateACFContent(data)
	if !strings.Contains(content, `"appid"`+"\t\t"+`"1091500"`) {
		t.Errorf("expected appid in generated ACF, got:\n%s", content)
	}
	if !strings.Contains(content, `"installdir"`+"\t\t"+`"Cyberpunk 2077"`) {
		t.Errorf("expected installdir in generated ACF, got:\n%s", content)
	}
}

func TestEnsureACFManifest(t *testing.T) {
	tmpDir := t.TempDir()
	info := &GameInfo{
		AppID:      "1091500",
		Name:       "Cyberpunk 2077",
		InstallDir: "Cyberpunk 2077",
		GameDir:    tmpDir,
	}

	if err := EnsureACFManifest(info, false); err != nil {
		t.Fatalf("EnsureACFManifest failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "[Manifests]", "appmanifest_1091500.acf")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected ACF file at %s", expectedPath)
	}

	data, err := ParseACFFile(expectedPath)
	if err != nil {
		t.Fatalf("ParseACFFile failed on synthesized manifest: %v", err)
	}
	if data.AppID != "1091500" || data.InstallDir != "Cyberpunk 2077" {
		t.Errorf("synthesized ACF content mismatch: %+v", data)
	}
}

func TestNormalizeDirectory(t *testing.T) {
	tmpParent := t.TempDir()
	rawDir := filepath.Join(tmpParent, "Cyberpunk.2077.v2.12-P2P")
	if err := os.MkdirAll(rawDir, 0755); err != nil {
		t.Fatal(err)
	}

	info := &GameInfo{
		AppID:      "1091500",
		Name:       "Cyberpunk 2077",
		InstallDir: "Cyberpunk 2077",
		GameDir:    rawDir,
	}

	if err := NormalizeDirectory(info, false); err != nil {
		t.Fatalf("NormalizeDirectory failed: %v", err)
	}

	expectedDir := filepath.Join(tmpParent, "Cyberpunk 2077")
	if info.GameDir != expectedDir {
		t.Errorf("expected GameDir %s, got %s", expectedDir, info.GameDir)
	}
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("expected directory at %s", expectedDir)
	}
}

