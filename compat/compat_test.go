package compat_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/util"
)

func TestEnsurePrefix(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	cfg := config.DefaultConfig()
	centralUserDir := filepath.Join(tmpDir, "central_user")
	_ = os.MkdirAll(centralUserDir, 0755)
	cfg.SteamUserdata = centralUserDir

	game := domain.NewGame("Test Game", tmpDir, "12345", "win64")

	prefix, err := compat.EnsurePrefix(context.Background(), cfg, game, "")
	if err != nil {
		t.Fatalf("EnsurePrefix failed: %v", err)
	}

	if prefix == "" {
		t.Fatalf("Expected non-empty prefix path")
	}

	steamuserSymlink := filepath.Join(prefix, "drive_c", "users", "steamuser")
	fi, err := os.Lstat(steamuserSymlink)
	if err != nil {
		t.Fatalf("steamuser symlink does not exist: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("Expected %s to be a symlink", steamuserSymlink)
	}
}

func TestProfilePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "MyGame")
	_ = os.MkdirAll(gameDir, 0755)

	p := &domain.LaunchProfile{
		GameDir:       gameDir,
		Name:          "My Game",
		AppID:         "12345",
		OperatingMode: "loader",
		Runner:        "umu",
		Language:      "english",
	}

	if err := compat.SaveProfile(p); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}

	loaded, err := compat.LoadProfile(gameDir)
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}
	if loaded.Name != "My Game" || loaded.AppID != "12345" {
		t.Fatalf("Loaded profile mismatch: %+v", loaded)
	}

	if err := compat.ResetProfile(gameDir); err != nil {
		t.Fatalf("ResetProfile failed: %v", err)
	}

	profileFile := filepath.Join(gameDir, "steam_settings", "launch_profile.json")
	if util.FileExists(profileFile) {
		t.Fatalf("Profile file should have been removed")
	}
}

func TestProfilePreservationOnSave(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "PreserveGame")
	_ = os.MkdirAll(gameDir, 0755)

	initial := &domain.LaunchProfile{
		GameDir:         gameDir,
		Name:            "Preserve Game",
		AppID:           "777",
		InstalledAt:     1650000000,
		PlaytimeSeconds: 7200,
		LastPlayed:      1700000000,
		CustomCoverURL:  "https://example.com/cover.png",
		LockedArtwork:   true,
		MetadataProvider: "steam",
		MetadataID:      "777",
	}

	if err := compat.SaveProfileDirect(initial); err != nil {
		t.Fatalf("SaveProfileDirect failed: %v", err)
	}

	// Now simulate drawer edit saving only settings without playtime or artwork
	partialEdit := &domain.LaunchProfile{
		GameDir:       gameDir,
		Name:          "Preserve Game",
		Runner:        "proton",
		OperatingMode: "standalone",
	}

	if err := compat.SaveProfile(partialEdit); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}

	loaded, err := compat.LoadProfile(gameDir)
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}

	if loaded.InstalledAt != 1650000000 {
		t.Errorf("expected InstalledAt 1650000000 to be preserved, got %d", loaded.InstalledAt)
	}
	if loaded.PlaytimeSeconds != 7200 {
		t.Errorf("expected PlaytimeSeconds 7200 to be preserved, got %d", loaded.PlaytimeSeconds)
	}
	if loaded.LastPlayed != 1700000000 {
		t.Errorf("expected LastPlayed 1700000000 to be preserved, got %d", loaded.LastPlayed)
	}
	if loaded.CustomCoverURL != "https://example.com/cover.png" {
		t.Errorf("expected CustomCoverURL to be preserved, got %s", loaded.CustomCoverURL)
	}
	if !loaded.LockedArtwork {
		t.Errorf("expected LockedArtwork to remain true")
	}
	if loaded.Runner != "proton" {
		t.Errorf("expected updated Runner proton, got %s", loaded.Runner)
	}
}

func TestIncrementPlaytime(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "IncrementGame")
	_ = os.MkdirAll(gameDir, 0755)

	tot1, lp1, err := compat.IncrementPlaytime(gameDir, 120)
	if err != nil {
		t.Fatalf("IncrementPlaytime failed: %v", err)
	}
	if tot1 != 120 || lp1 <= 0 {
		t.Errorf("expected 120s playtime, got %d (lastPlayed: %d)", tot1, lp1)
	}

	tot2, lp2, err := compat.IncrementPlaytime(gameDir, 60)
	if err != nil {
		t.Fatalf("second IncrementPlaytime failed: %v", err)
	}
	if tot2 != 180 || lp2 < lp1 {
		t.Errorf("expected 180s playtime, got %d (lastPlayed: %d)", tot2, lp2)
	}
}

func TestSyncSettings_DoesNotMutateBinaries(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "NonMutatingGame")
	_ = os.MkdirAll(gameDir, 0755)

	// Create dummy binary file
	dummyBin := filepath.Join(gameDir, "steam_api64.dll")
	_ = os.WriteFile(dummyBin, []byte("ORIGINAL_BINARY_BYTES"), 0755)

	p := &domain.LaunchProfile{
		GameDir:       gameDir,
		Name:          "Non-Mutating Game",
		AppID:         "480",
		Language:      "german",
		ExeArgs:       "--fullscreen --novid",
		OperatingMode: "standalone",
	}

	cfg := config.DefaultConfig()
	if err := compat.SyncSettings(context.Background(), p, cfg); err != nil {
		t.Fatalf("SyncSettings failed: %v", err)
	}

	// Verify that dummy binary is 100% untouched
	binData, err := os.ReadFile(dummyBin)
	if err != nil {
		t.Fatalf("Failed to read binary: %v", err)
	}
	if string(binData) != "ORIGINAL_BINARY_BYTES" {
		t.Errorf("Binary was mutated during SyncSettings!")
	}

	// Verify no .ORIGINAL backups were created
	matches, _ := filepath.Glob(filepath.Join(gameDir, "*.ORIGINAL"))
	if len(matches) > 0 {
		t.Errorf("Expected 0 backup files, found: %v", matches)
	}

	// Verify settings files were written correctly
	appIDFile := filepath.Join(gameDir, "steam_settings", "steam_appid.txt")
	if data, err := os.ReadFile(appIDFile); err != nil || string(data) != "480" {
		t.Errorf("steam_appid.txt not synced properly, got: %s (err: %v)", string(data), err)
	}

	langFile := filepath.Join(gameDir, "steam_settings", "force_language.txt")
	if data, err := os.ReadFile(langFile); err != nil || string(data) != "german\n" {
		t.Errorf("force_language.txt not synced properly, got: %s (err: %v)", string(data), err)
	}
}

func TestApply_InjectedOperatingMode(t *testing.T) {
	tmpDir := t.TempDir()
	gbeDir := filepath.Join(tmpDir, "gbe_fork")
	winGbe := filepath.Join(gbeDir, "win_release", "experimental", "x64")
	_ = os.MkdirAll(winGbe, 0755)
	_ = os.WriteFile(filepath.Join(winGbe, "steam_api64.dll"), []byte("emulated_dll_bytes"), 0644)

	gameDir := filepath.Join(tmpDir, "InjectedGame")
	_ = os.MkdirAll(gameDir, 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "steam_api64.dll"), []byte("original_dll_bytes"), 0644)

	cfg := &config.Config{GbeDir: gbeDir}
	profile := &domain.LaunchProfile{
		GameDir:       gameDir,
		AppID:         "1091500",
		Platform:      "win64",
		OperatingMode: "injected",
	}
	game := domain.NewGame("InjectedGame", gameDir, "1091500", "win64")

	err := compat.Apply(context.Background(), cfg, game, profile, false)
	if err != nil {
		t.Fatalf("compat.Apply failed with injected mode: %v", err)
	}

	// Verify original file was backed up and replaced with emulator dll
	backups, _ := filepath.Glob(filepath.Join(gameDir, "*.ORIGINAL"))
	if len(backups) == 0 {
		t.Errorf("expected .ORIGINAL backup to be created for injected mode")
	}

	currentBytes, _ := os.ReadFile(filepath.Join(gameDir, "steam_api64.dll"))
	if string(currentBytes) != "emulated_dll_bytes" {
		t.Errorf("expected steam_api64.dll to be replaced with emulator bytes, got %s", string(currentBytes))
	}
}
