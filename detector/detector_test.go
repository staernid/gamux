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

func TestDetect_EngineTopologies(t *testing.T) {
	t.Run("Unity Win64 Topology", func(t *testing.T) {
		tmpDir := t.TempDir()
		gameDir := filepath.Join(tmpDir, "SuperGame")
		pluginsDir := filepath.Join(gameDir, "SuperGame_Data", "Plugins", "x86_64")
		manifestsDir := filepath.Join(gameDir, "[Manifests]")
		_ = os.MkdirAll(pluginsDir, 0755)
		_ = os.MkdirAll(manifestsDir, 0755)

		_ = os.WriteFile(filepath.Join(gameDir, "SuperGame.exe"), []byte("exe"), 0755)
		_ = os.WriteFile(filepath.Join(gameDir, "UnityCrashHandler64.exe"), []byte("crash handler"), 0755)
		_ = os.WriteFile(filepath.Join(pluginsDir, "steam_api64.dll"), []byte("dll"), 0644)
		_ = os.WriteFile(filepath.Join(manifestsDir, "appmanifest_11111.acf"), []byte("\"AppState\"\n{\n\t\"appid\"\t\"11111\"\n\t\"name\"\t\"SuperGame\"\n}"), 0644)

		info, err := Detect(context.Background(), gameDir)
		if err != nil {
			t.Fatalf("Detect failed: %v", err)
		}
		if info.Platform != "win64" {
			t.Errorf("expected platform win64, got %s", info.Platform)
		}
		if info.AppID != "11111" {
			t.Errorf("expected appid 11111, got %s", info.AppID)
		}
		if filepath.Base(info.ExePath) != "SuperGame.exe" {
			t.Errorf("expected main executable SuperGame.exe, got %s", filepath.Base(info.ExePath))
		}
		if !strings.HasSuffix(info.TargetLibPath, "steam_api64.dll") {
			t.Errorf("expected TargetLibPath to end with steam_api64.dll, got %s", info.TargetLibPath)
		}
	})

	t.Run("Unreal Engine Win64 Topology", func(t *testing.T) {
		tmpDir := t.TempDir()
		gameDir := filepath.Join(tmpDir, "UEGame")
		binDir := filepath.Join(gameDir, "UEGame", "Binaries", "Win64")
		steamDir := filepath.Join(gameDir, "Engine", "Binaries", "ThirdParty", "Steamworks", "Steamv153", "Win64")
		_ = os.MkdirAll(binDir, 0755)
		_ = os.MkdirAll(steamDir, 0755)

		_ = os.WriteFile(filepath.Join(gameDir, "UEGame.exe"), []byte("launcher"), 0755)
		_ = os.WriteFile(filepath.Join(binDir, "UEGame-Win64-Shipping.exe"), []byte("shipping exe"), 0755)
		_ = os.WriteFile(filepath.Join(steamDir, "steam_api64.dll"), []byte("dll"), 0644)
		_ = os.WriteFile(filepath.Join(gameDir, "appmanifest_22222.acf"), []byte("\"AppState\"\n{\n\t\"appid\"\t\"22222\"\n\t\"name\"\t\"UEGame\"\n}"), 0644)

		info, err := Detect(context.Background(), gameDir)
		if err != nil {
			t.Fatalf("Detect failed: %v", err)
		}
		if info.Platform != "win64" {
			t.Errorf("expected platform win64, got %s", info.Platform)
		}
		if info.AppID != "22222" {
			t.Errorf("expected appid 22222, got %s", info.AppID)
		}
		if !strings.HasSuffix(info.TargetLibPath, "steam_api64.dll") {
			t.Errorf("expected target lib steam_api64.dll, got %s", info.TargetLibPath)
		}
	})

	t.Run("Linux Native Topology", func(t *testing.T) {
		tmpDir := t.TempDir()
		gameDir := filepath.Join(tmpDir, "LinuxGame")
		libDir := filepath.Join(gameDir, "lib64")
		_ = os.MkdirAll(libDir, 0755)

		_ = os.WriteFile(filepath.Join(gameDir, "LinuxGame.x86_64"), []byte("elf"), 0755)
		_ = os.WriteFile(filepath.Join(libDir, "libsteam_api.so"), []byte("so"), 0755)
		_ = os.WriteFile(filepath.Join(gameDir, "steam_appid.txt"), []byte("33333"), 0644)

		info, err := Detect(context.Background(), gameDir)
		if err != nil {
			t.Fatalf("Detect failed: %v", err)
		}
		if info.Platform != "linux" {
			t.Errorf("expected platform linux, got %s", info.Platform)
		}
		if info.AppID != "33333" {
			t.Errorf("expected appid 33333, got %s", info.AppID)
		}
		if filepath.Base(info.ExePath) != "LinuxGame.x86_64" {
			t.Errorf("expected main executable LinuxGame.x86_64, got %s", filepath.Base(info.ExePath))
		}
		if !strings.HasSuffix(info.TargetLibPath, "libsteam_api.so") {
			t.Errorf("expected target lib libsteam_api.so, got %s", info.TargetLibPath)
		}
	})

	t.Run("LaunchProfile Custom Target Exe Priority", func(t *testing.T) {
		tmpDir := t.TempDir()
		gameDir := filepath.Join(tmpDir, "MultiExeGame")
		_ = os.MkdirAll(filepath.Join(gameDir, "steam_settings"), 0755)

		_ = os.WriteFile(filepath.Join(gameDir, "MultiExeGame.exe"), []byte("exe1"), 0755)
		_ = os.WriteFile(filepath.Join(gameDir, "MultiExeGame_DX11.exe"), []byte("exe2"), 0755)
		_ = os.WriteFile(filepath.Join(gameDir, "steam_appid.txt"), []byte("44444"), 0644)

		lpJSON := `{"exe_path": "MultiExeGame_DX11.exe", "exe_args": "-dx11"}`
		_ = os.WriteFile(filepath.Join(gameDir, "steam_settings", "launch_profile.json"), []byte(lpJSON), 0644)

		info, err := Detect(context.Background(), gameDir)
		if err != nil {
			t.Fatalf("Detect failed: %v", err)
		}
		if filepath.Base(info.EffectiveExePath()) != "MultiExeGame_DX11.exe" {
			t.Errorf("expected target executable MultiExeGame_DX11.exe from launch profile, got %s", filepath.Base(info.EffectiveExePath()))
		}
		if info.EffectiveExeArgs() != "-dx11" {
			t.Errorf("expected exe_args '-dx11', got %s", info.EffectiveExeArgs())
		}
	})

	t.Run("GOG Game Detection", func(t *testing.T) {
		tmpDir := t.TempDir()
		gameDir := filepath.Join(tmpDir, "Witcher3")
		_ = os.MkdirAll(gameDir, 0755)

		gogJSON := `{"gameId": "1207664643", "name": "The Witcher 3: Wild Hunt"}`
		_ = os.WriteFile(filepath.Join(gameDir, "goggame-1207664643.info"), []byte(gogJSON), 0644)
		_ = os.WriteFile(filepath.Join(gameDir, "witcher3.exe"), []byte("exe"), 0755)

		info, err := Detect(context.Background(), gameDir)
		if err != nil {
			t.Fatalf("Detect failed: %v", err)
		}
		if info.Source != "GOG" || info.Store != "GOG" {
			t.Errorf("expected GOG store/source, got %s / %s", info.Store, info.Source)
		}
		if info.GOGID != "1207664643" {
			t.Errorf("expected GOGID 1207664643, got %s", info.GOGID)
		}
		if info.Name != "The Witcher 3: Wild Hunt" {
			t.Errorf("expected name 'The Witcher 3: Wild Hunt', got %s", info.Name)
		}
	})

	t.Run("GOG Game with Steam Metadata Override", func(t *testing.T) {
		tmpDir := t.TempDir()
		gameDir := filepath.Join(tmpDir, "Witcher3_SteamMeta")
		_ = os.MkdirAll(filepath.Join(gameDir, "steam_settings"), 0755)

		gogJSON := `{"gameId": "1207664643", "name": "The Witcher 3: Wild Hunt"}`
		_ = os.WriteFile(filepath.Join(gameDir, "goggame-1207664643.info"), []byte(gogJSON), 0644)
		_ = os.WriteFile(filepath.Join(gameDir, "witcher3.exe"), []byte("exe"), 0755)

		lpJSON := `{"metadata_provider": "steam", "metadata_id": "292030", "store_app_id": "292030"}`
		_ = os.WriteFile(filepath.Join(gameDir, "steam_settings", "launch_profile.json"), []byte(lpJSON), 0644)

		info, err := Detect(context.Background(), gameDir)
		if err != nil {
			t.Fatalf("Detect failed: %v", err)
		}
		if info.Source != "GOG" {
			t.Errorf("expected physical Source to remain 'GOG', got %s", info.Source)
		}
		if info.EffectiveMetadataProvider() != "steam" {
			t.Errorf("expected EffectiveMetadataProvider 'steam', got %s", info.EffectiveMetadataProvider())
		}
		if info.ExternalIDs.SteamAppID != 292030 {
			t.Errorf("expected ExternalIDs.SteamAppID 292030, got %d", info.ExternalIDs.SteamAppID)
		}
	})

	t.Run("Standalone Game with GOG Metadata Override", func(t *testing.T) {
		tmpDir := t.TempDir()
		gameDir := filepath.Join(tmpDir, "Warcraft3")
		_ = os.MkdirAll(filepath.Join(gameDir, "steam_settings"), 0755)

		_ = os.WriteFile(filepath.Join(gameDir, "war3.exe"), []byte("exe"), 0755)

		lpJSON := `{"metadata_provider": "gog", "metadata_id": "99999", "gog_id": "99999"}`
		_ = os.WriteFile(filepath.Join(gameDir, "steam_settings", "launch_profile.json"), []byte(lpJSON), 0644)

		info, err := Detect(context.Background(), gameDir)
		if err != nil {
			t.Fatalf("Detect failed: %v", err)
		}
		if info.Source != "Standalone" {
			t.Errorf("expected physical Source 'Standalone', got %s", info.Source)
		}
		if info.EffectiveMetadataProvider() != "gog" {
			t.Errorf("expected EffectiveMetadataProvider 'gog', got %s", info.EffectiveMetadataProvider())
		}
		if info.ExternalIDs.GOGID != "99999" {
			t.Errorf("expected ExternalIDs.GOGID '99999', got %s", info.ExternalIDs.GOGID)
		}
	})

	t.Run("Epic Game Detection", func(t *testing.T) {
		tmpDir := t.TempDir()
		gameDir := filepath.Join(tmpDir, "EpicGame")
		egDir := filepath.Join(gameDir, ".egstore")
		_ = os.MkdirAll(egDir, 0755)

		_ = os.WriteFile(filepath.Join(egDir, "Sugar.mancpn"), []byte("manifest"), 0644)
		_ = os.WriteFile(filepath.Join(gameDir, "game.exe"), []byte("exe"), 0755)

		info, err := Detect(context.Background(), gameDir)
		if err != nil {
			t.Fatalf("Detect failed: %v", err)
		}
		if info.Source != "Epic" || info.Store != "Epic" {
			t.Errorf("expected Epic store/source, got %s / %s", info.Store, info.Source)
		}
		if info.EpicApp != "Sugar" {
			t.Errorf("expected EpicApp Sugar, got %s", info.EpicApp)
		}
	})

	t.Run("Runner Prefix Directory Rejection", func(t *testing.T) {
		tmpDir := t.TempDir()
		umuDir := filepath.Join(tmpDir, "umu")
		_ = os.MkdirAll(filepath.Join(umuDir, "drive_c", "windows", "system32"), 0755)
		_ = os.WriteFile(filepath.Join(umuDir, "system.reg"), []byte("reg"), 0644)
		_ = os.WriteFile(filepath.Join(umuDir, "drive_c", "windows", "explorer.exe"), []byte("exe"), 0755)

		_, err := Detect(context.Background(), umuDir)
		if err == nil {
			t.Fatalf("expected Detect to reject runner prefix directory 'umu', but got nil error")
		}
	})

	t.Run("Skip Drive C Subfolder During Game Inspection", func(t *testing.T) {
		tmpDir := t.TempDir()
		gameDir := filepath.Join(tmpDir, "MySporeGame")
		driveCDir := filepath.Join(gameDir, "drive_c", "users", "steamuser", "AppData")
		_ = os.MkdirAll(driveCDir, 0755)
		_ = os.WriteFile(filepath.Join(gameDir, "Spore.exe"), []byte("exe"), 0755)
		// Put an unins000 or other exe deep inside drive_c
		_ = os.WriteFile(filepath.Join(driveCDir, "unrelated.exe"), []byte("exe"), 0755)

		info, err := Detect(context.Background(), gameDir)
		if err != nil {
			t.Fatalf("Detect failed: %v", err)
		}
		for _, cand := range info.LaunchCandidates {
			if strings.Contains(cand.Executable, "unrelated.exe") {
				t.Errorf("expected drive_c to be skipped, but found candidate: %s", cand.Executable)
			}
		}
	})

	t.Run("SinglePass Complete Hydration", func(t *testing.T) {
		tmpDir := t.TempDir()
		gameDir := filepath.Join(tmpDir, "CompleteHydrationGame")
		manifestsDir := filepath.Join(gameDir, "[Manifests]")
		settingsDir := filepath.Join(gameDir, "steam_settings")
		_ = os.MkdirAll(manifestsDir, 0755)
		_ = os.MkdirAll(settingsDir, 0755)

		_ = os.WriteFile(filepath.Join(gameDir, "Game.exe"), []byte("MZGame"), 0755)
		_ = os.WriteFile(filepath.Join(gameDir, "steam_api64.dll"), []byte("MZDLL"), 0644)
		_ = os.WriteFile(filepath.Join(gameDir, "steam_api64.dll.ORIGINAL"), []byte("MZOrig"), 0644)
		_ = os.WriteFile(filepath.Join(manifestsDir, "appmanifest_98765.acf"), []byte("\"AppState\"\n{\n\t\"appid\"\t\"98765\"\n\t\"name\"\t\"HydrationGame\"\n}"), 0644)
		_ = os.WriteFile(filepath.Join(manifestsDir, "depot_12345.manifest"), []byte("manifest binary data"), 0644)
		_ = os.WriteFile(filepath.Join(settingsDir, "force_language.txt"), []byte("japanese\n"), 0644)
		_ = os.WriteFile(filepath.Join(settingsDir, "launch_profile.json"), []byte("{\"custom_cover_url\": \"http://test/cover.png\", \"dxvk_hud\": \"fps\"}"), 0644)

		info, err := Inspect(context.Background(), gameDir)
		if err != nil {
			t.Fatalf("Inspect failed: %v", err)
		}

		if info.AppID != "98765" || info.NumericAppID != 98765 {
			t.Errorf("expected AppID 98765, got %s (numeric %d)", info.AppID, info.NumericAppID)
		}
		if info.Name != "HydrationGame" {
			t.Errorf("expected Name 'HydrationGame', got %s", info.Name)
		}
		if info.Platform != "win64" {
			t.Errorf("expected Platform 'win64', got %s", info.Platform)
		}
		if info.FileCount < 5 {
			t.Errorf("expected at least 5 files, got %d", info.FileCount)
		}
		if info.DiskSizeBytes <= 0 {
			t.Errorf("expected positive disk size, got %d", info.DiskSizeBytes)
		}
		if len(info.OriginalBackups) != 1 {
			t.Errorf("expected 1 original backup, got %d", len(info.OriginalBackups))
		}
		if !info.SettingsDirExists {
			t.Errorf("expected SettingsDirExists to be true")
		}
		if info.CurrentLanguage != "japanese" {
			t.Errorf("expected CurrentLanguage 'japanese', got %s", info.CurrentLanguage)
		}
		if info.ManifestID != "12345" {
			t.Errorf("expected ManifestID '12345', got %s", info.ManifestID)
		}
		if info.CustomCoverURL != "http://test/cover.png" {
			t.Errorf("expected CustomCoverURL 'http://test/cover.png', got %s", info.CustomCoverURL)
		}
		if info.State != "Portable-Patched" || !info.IsPortable || !info.IsPatched {
			t.Errorf("expected Portable-Patched state, got State=%s, IsPortable=%v, IsPatched=%v", info.State, info.IsPortable, info.IsPatched)
		}
		if !info.IsReady {
			t.Errorf("expected IsReady to be true")
		}
	})
}

func TestDetect_UnrelatedDirectoryNameRejectsSteamSearch(t *testing.T) {
	tmpDir := t.TempDir()
	customDir := filepath.Join(tmpDir, "UnrelatedCustomGame")
	if err := os.MkdirAll(filepath.Join(customDir, "steam_settings"), 0755); err != nil {
		t.Fatal(err)
	}

	info, err := Detect(context.Background(), customDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if info.Name != "UnrelatedCustomGame" {
		t.Errorf("expected game name to remain 'UnrelatedCustomGame', got %q", info.Name)
	}
	if info.AppID != "" && info.AppID != "0" {
		t.Errorf("expected empty AppID, got %q", info.AppID)
	}
}



