package compat_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/util"
	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
	restoreEnv := testutil.SetupTestEnvironment()
	restoreLogs := testutil.SilenceLogging()
	code := m.Run()
	restoreLogs()
	restoreEnv()
	os.Exit(code)
}

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

func TestEnsureSeatPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, ".local", "share"))

	cfg := config.DefaultConfig()
	game := domain.NewGame("Half-Life", filepath.Join(tmpDir, "Half-Life"), "70", "win64")

	// 1. Seat 1 (Player 1) default prefix -> ~/Games/umu/70
	p1Prefix, err := compat.EnsureSeatPrefix(context.Background(), cfg, game, "p1", "")
	if err != nil {
		t.Fatalf("EnsureSeatPrefix(p1) failed: %v", err)
	}
	expectedP1 := filepath.Join(tmpDir, "Games", "umu", "70")
	if p1Prefix != expectedP1 {
		t.Errorf("expected p1 prefix %s, got %s", expectedP1, p1Prefix)
	}

	// 2. Seat 2 (Player 2) default prefix -> ~/.local/share/gamux/profiles/p2/home/Games/umu/70
	p2Prefix, err := compat.EnsureSeatPrefix(context.Background(), cfg, game, "p2", "")
	if err != nil {
		t.Fatalf("EnsureSeatPrefix(p2) failed: %v", err)
	}
	expectedP2 := filepath.Join(tmpDir, ".local", "share", "gamux", "profiles", "p2", "home", "Games", "umu", "70")
	if p2Prefix != expectedP2 {
		t.Errorf("expected p2 prefix %s, got %s", expectedP2, p2Prefix)
	}

	// 3. Seat 2 with custom configured prefix
	customP2 := filepath.Join(tmpDir, "custom-p2-prefix")
	p2Custom, err := compat.EnsureSeatPrefix(context.Background(), cfg, game, "p2", customP2)
	if err != nil {
		t.Fatalf("EnsureSeatPrefix(p2 custom) failed: %v", err)
	}
	if p2Custom != customP2 {
		t.Errorf("expected p2 custom prefix %s, got %s", customP2, p2Custom)
	}

	// 4. Seat 2 with prefix base directory -> should append prefix ID (70)
	p2Base := filepath.Join(tmpDir, ".local", "share", "gamux", "profiles", "p2", "home", "Games", "umu")
	p2ResolvedFromBase, err := compat.EnsureSeatPrefix(context.Background(), cfg, game, "p2", p2Base)
	if err != nil {
		t.Fatalf("EnsureSeatPrefix(p2 base) failed: %v", err)
	}
	if p2ResolvedFromBase != expectedP2 {
		t.Errorf("expected %s when passing base prefix dir, got %s", expectedP2, p2ResolvedFromBase)
	}

	// 5. SeatIDFromProfileID mapping
	if compat.SeatIDFromProfileID("main") != "p1" {
		t.Errorf("expected main -> p1, got %s", compat.SeatIDFromProfileID("main"))
	}
	if compat.SeatIDFromProfileID("p2") != "p2" {
		t.Errorf("expected p2 -> p2, got %s", compat.SeatIDFromProfileID("p2"))
	}
	if compat.SeatIDFromProfileID("") != "p1" {
		t.Errorf("expected empty -> p1, got %s", compat.SeatIDFromProfileID(""))
	}

	// 6. Pre-created AppData/Roaming in central user data directory
	centralRoamingP2 := filepath.Join(tmpDir, ".local", "share", "gamux", "profiles", "p2", "home", "Games", "umu", "umu-default", "drive_c", "users", "steamuser", "AppData", "Roaming")
	if _, err := os.Stat(centralRoamingP2); os.IsNotExist(err) {
		t.Errorf("expected central AppData/Roaming to exist at %s", centralRoamingP2)
	}

	// 7. GSE Saves symlink exists in prefix
	gseLink := filepath.Join(p2Prefix, "drive_c", "users", "steamuser", "AppData", "Roaming", "GSE Saves")
	fi, err := os.Lstat(gseLink)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected GSE Saves symlink to exist at %s, err: %v", gseLink, err)
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

	profileFile := filepath.Join(gameDir, ".gamux", "launch_profile.json")
	if util.FileExists(profileFile) {
		t.Fatalf("Profile file should have been removed")
	}
}

func TestProfilePreservationOnSave(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "PreserveGame")
	_ = os.MkdirAll(gameDir, 0755)

	initial := &domain.LaunchProfile{
		GameDir:          gameDir,
		Name:             "Preserve Game",
		AppID:            "777",
		InstalledAt:      1650000000,
		PlaytimeSeconds:  7200,
		LastPlayed:       1700000000,
		CustomCoverURL:   "https://example.com/cover.png",
		LockedArtwork:    true,
		MetadataProvider: "steam",
		MetadataID:       "777",
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

func TestLegacyProfileMigration(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "LegacyGame")
	legacySettingsDir := filepath.Join(gameDir, "steam_settings")
	_ = os.MkdirAll(legacySettingsDir, 0755)

	legacyProfile := filepath.Join(legacySettingsDir, "launch_profile.json")
	_ = os.WriteFile(legacyProfile, []byte(`{
		"game_dir": "`+gameDir+`",
		"name": "Migrated Title",
		"app_id": "99999",
		"runner": "umu"
	}`), 0644)

	// LoadProfile should load, migrate to .gamux, and delete legacy file
	loaded, err := compat.LoadProfile(gameDir)
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}
	if loaded.Name != "Migrated Title" || loaded.AppID != "99999" {
		t.Fatalf("unexpected loaded profile data: %+v", loaded)
	}

	newProfilePath := filepath.Join(gameDir, ".gamux", "launch_profile.json")
	if !util.FileExists(newProfilePath) {
		t.Errorf("expected migrated profile at %s", newProfilePath)
	}
	if util.FileExists(legacyProfile) {
		t.Errorf("expected legacy profile %s to be deleted after migration", legacyProfile)
	}
}

func TestArbitraryNestedSteamSettingsAndLoaderMode(t *testing.T) {
	tmpDir := t.TempDir()

	// GBE Mock Repo
	gbeDir := filepath.Join(tmpDir, "gbe_fork")
	x64Release := filepath.Join(gbeDir, "win_release", "experimental", "x64")
	_ = os.MkdirAll(x64Release, 0755)
	_ = os.WriteFile(filepath.Join(x64Release, "steam_api64.dll"), []byte("emulated_gbe_dll"), 0755)
	_ = os.WriteFile(filepath.Join(x64Release, "steamclient64.dll"), []byte("emulated_steamclient_dll"), 0755)

	loaderDir := filepath.Join(gbeDir, "win_release", "steamclient_experimental")
	_ = os.MkdirAll(loaderDir, 0755)
	_ = os.WriteFile(filepath.Join(loaderDir, "steamclient64.dll"), []byte("loader_steamclient64"), 0755)
	_ = os.WriteFile(filepath.Join(loaderDir, "steamclient.dll"), []byte("loader_steamclient32"), 0755)
	_ = os.WriteFile(filepath.Join(loaderDir, "GameOverlayRenderer64.dll"), []byte("loader_overlay64"), 0755)
	_ = os.WriteFile(filepath.Join(loaderDir, "GameOverlayRenderer.dll"), []byte("loader_overlay32"), 0755)

	cfg := config.DefaultConfig()
	cfg.GbeDir = gbeDir
	cfg.OperatingMode = domain.ModeLoader

	// Simulate nested Unreal Engine directory structure (like Retro Rewind)
	gameDir := filepath.Join(tmpDir, "UnrealGame")
	shippingExeDir := filepath.Join(gameDir, "SubGame", "Binaries", "Win64")
	pluginLibDir := filepath.Join(gameDir, "Engine", "Binaries", "ThirdParty", "Steamworks", "Steamv153", "Win64")
	_ = os.MkdirAll(shippingExeDir, 0755)
	_ = os.MkdirAll(pluginLibDir, 0755)

	rootExe := filepath.Join(gameDir, "Game.exe")
	shippingExe := filepath.Join(shippingExeDir, "SubGame-Win64-Shipping.exe")
	steamLib := filepath.Join(pluginLibDir, "steam_api64.dll")

	_ = os.WriteFile(rootExe, []byte("MZ_ROOT"), 0755)
	_ = os.WriteFile(shippingExe, []byte("MZ_SHIPPING"), 0755)
	_ = os.WriteFile(steamLib, []byte("original_steam_api64"), 0755)

	// Pre-existing nested steam_settings in shipping binary folder
	preNestedSettings := filepath.Join(shippingExeDir, "steam_settings")
	_ = os.MkdirAll(preNestedSettings, 0755)
	_ = os.WriteFile(filepath.Join(preNestedSettings, "steam_appid.txt"), []byte("2986870"), 0644)
	_ = os.WriteFile(filepath.Join(preNestedSettings, "force_language.txt"), []byte("spanish\n"), 0644)

	// 1. Verify LoadProfile hydrates metadata from nested steam_settings
	prof, err := compat.LoadProfile(gameDir)
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}
	if prof.AppID != "2986870" {
		t.Errorf("expected hydrated AppID 2986870, got: %s", prof.AppID)
	}
	if prof.Language != "spanish" {
		t.Errorf("expected hydrated Language spanish, got: %s", prof.Language)
	}

	game := domain.NewGame("Unreal Game", gameDir, "2986870", "win64")
	game.ExePath = rootExe
	game.LaunchCandidates = []domain.LaunchCandidate{
		{Executable: shippingExe, Name: "Shipping"},
	}

	// 2. Apply Loader Mode
	loaderProfile := &domain.LaunchProfile{
		GameDir:       gameDir,
		AppID:         "2986870",
		Platform:      "win64",
		OperatingMode: domain.ModeLoader,
		ExePath:       shippingExe,
		Language:      "english",
	}
	if err := compat.Apply(context.Background(), cfg, game, loaderProfile, false); err != nil {
		t.Fatalf("compat.Apply(loader) failed: %v", err)
	}

	// In Loader mode: original steam_api64.dll must remain pristine
	libData, _ := os.ReadFile(steamLib)
	if string(libData) != "original_steam_api64" {
		t.Fatalf("original library was modified in loader mode: %s", string(libData))
	}

	// Loader assets must be deployed to root and candidate exe dirs
	for _, dir := range []string{gameDir, shippingExeDir, pluginLibDir} {
		clientDll := filepath.Join(dir, "steamclient64.dll")
		if !util.FileExists(clientDll) {
			t.Errorf("expected loader asset steamclient64.dll at %s", dir)
		}
		appIDTxt := filepath.Join(dir, "steam_appid.txt")
		if !util.FileExists(appIDTxt) {
			t.Errorf("expected steam_appid.txt at %s", dir)
		}
	}

	// Force language must be synchronized across all steam_settings dirs
	for _, dir := range []string{
		filepath.Join(gameDir, "steam_settings"),
		preNestedSettings,
		filepath.Join(pluginLibDir, "steam_settings"),
	} {
		langFile := filepath.Join(dir, "force_language.txt")
		if data, err := os.ReadFile(langFile); err != nil || strings.TrimSpace(string(data)) != "english" {
			t.Errorf("expected synchronized force_language.txt in %s, got: %s (err: %v)", dir, string(data), err)
		}
	}

	// 3. Transition to Portable Mode
	portableProfile := &domain.LaunchProfile{
		GameDir:       gameDir,
		AppID:         "2986870",
		Platform:      "win64",
		OperatingMode: domain.ModePortable,
		ExePath:       shippingExe,
		Language:      "english",
	}
	if err := compat.Apply(context.Background(), cfg, game, portableProfile, false); err != nil {
		t.Fatalf("compat.Apply(portable) failed: %v", err)
	}

	// In Portable mode: loader assets must be cleaned up
	if util.FileExists(filepath.Join(gameDir, "steamclient64.dll")) {
		t.Errorf("expected loader steamclient64.dll to be removed in portable mode")
	}
	if util.FileExists(filepath.Join(shippingExeDir, "steamclient64.dll")) {
		t.Errorf("expected shipping loader steamclient64.dll to be removed in portable mode")
	}

	// Library replaced and backup created
	newLibData, _ := os.ReadFile(steamLib)
	if string(newLibData) != "emulated_gbe_dll" {
		t.Errorf("expected emulated dll in portable mode, got: %s", string(newLibData))
	}
	backups, _ := filepath.Glob(filepath.Join(pluginLibDir, "*.ORIGINAL"))
	if len(backups) == 0 {
		t.Errorf("expected backup file in pluginLibDir")
	}

	// 4. Revert back to clean state
	if _, err := compat.Revert(context.Background(), gameDir, false); err != nil {
		t.Fatalf("compat.Revert failed: %v", err)
	}
	revertedLib, _ := os.ReadFile(steamLib)
	if string(revertedLib) != "original_steam_api64" {
		t.Errorf("expected restored original library after revert, got: %s", string(revertedLib))
	}
	if util.FileExists(filepath.Join(gameDir, "steam_settings")) {
		t.Errorf("expected root steam_settings to be deleted after revert")
	}
	if util.FileExists(preNestedSettings) {
		t.Errorf("expected nested steam_settings to be deleted after revert")
	}
}

func TestResolveStrategy(t *testing.T) {
	loader := compat.ResolveStrategy(domain.ModeLoader)
	if loader.Mode() != domain.ModeLoader {
		t.Errorf("expected ModeLoader, got %s", loader.Mode())
	}

	portable := compat.ResolveStrategy(domain.ModePortable)
	if portable.Mode() != domain.ModePortable {
		t.Errorf("expected ModePortable, got %s", portable.Mode())
	}

	standalone := compat.ResolveStrategy(domain.ModeStandalone)
	if standalone.Mode() != domain.ModeStandalone && standalone.Mode() != domain.ModeNone {
		t.Errorf("expected ModeStandalone or ModeNone, got %s", standalone.Mode())
	}

	none := compat.ResolveStrategy(domain.ModeNone)
	if none.Mode() != domain.ModeNone {
		t.Errorf("expected ModeNone, got %s", none.Mode())
	}

	fallback := compat.ResolveStrategy("unknown_mode")
	if fallback.Mode() != domain.ModeLoader {
		t.Errorf("expected default fallback to ModeLoader, got %s", fallback.Mode())
	}
}

func TestShouldEmulate(t *testing.T) {
	if compat.ShouldEmulate(nil) {
		t.Error("nil game should not emulate")
	}

	gameNoID := domain.NewGame("Unlinked Game", "/tmp/unlinked", "", "win64")
	gameNoID.Store = domain.StoreStandalone
	if compat.ShouldEmulate(gameNoID) {
		t.Error("game with no AppID should not emulate by default")
	}

	gameZeroID := domain.NewGame("Zero ID Game", "/tmp/zero", "0", "win64")
	if compat.ShouldEmulate(gameZeroID) {
		t.Error("game with AppID '0' should not emulate")
	}

	gameSteam := domain.NewGame("Portal 2", "/tmp/portal2", "620", "win64")
	if !compat.ShouldEmulate(gameSteam) {
		t.Error("Steam game with valid AppID should emulate")
	}

	// ForceEmulation = true overrides absence of AppID
	forceTrue := true
	gameNoID.LaunchProfile = &domain.LaunchProfile{ForceEmulation: &forceTrue}
	if !compat.ShouldEmulate(gameNoID) {
		t.Error("ForceEmulation=true should enable emulation for unlinked game")
	}

	// ForceEmulation = false overrides valid Steam AppID
	forceFalse := false
	gameSteam.LaunchProfile = &domain.LaunchProfile{ForceEmulation: &forceFalse}
	if compat.ShouldEmulate(gameSteam) {
		t.Error("ForceEmulation=false should disable emulation even with valid AppID")
	}
}

func TestScrubPrefixEmulation(t *testing.T) {
	tmpDir := t.TempDir()
	steamDir := filepath.Join(tmpDir, "drive_c", "Program Files (x86)", "Steam")
	if err := os.MkdirAll(steamDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Populate emulation files and symlinks
	_ = os.WriteFile(filepath.Join(steamDir, "steamclient64.dll"), []byte("fake dll"), 0644)
	_ = os.WriteFile(filepath.Join(steamDir, "steamclient.dll"), []byte("fake dll"), 0644)
	_ = os.WriteFile(filepath.Join(steamDir, "GameOverlayRenderer64.dll"), []byte("fake dll"), 0644)
	_ = os.WriteFile(filepath.Join(steamDir, "GameOverlayRenderer.dll"), []byte("fake dll"), 0644)
	_ = os.WriteFile(filepath.Join(steamDir, "steam_appid.txt"), []byte("12345"), 0644)
	_ = os.Symlink(tmpDir, filepath.Join(steamDir, "steam_settings"))

	// Genuine user file that should NOT be removed
	userFile := filepath.Join(steamDir, "user_custom_data.txt")
	_ = os.WriteFile(userFile, []byte("user data"), 0644)

	if err := compat.ScrubPrefixEmulation(tmpDir); err != nil {
		t.Fatalf("ScrubPrefixEmulation failed: %v", err)
	}

	// Verify emulation artifacts removed
	for _, f := range []string{"steamclient64.dll", "steamclient.dll", "GameOverlayRenderer64.dll", "GameOverlayRenderer.dll", "steam_appid.txt", "steam_settings"} {
		p := filepath.Join(steamDir, f)
		if util.FileExists(p) {
			t.Errorf("expected %s to be scrubbed, but it still exists", f)
		}
	}

	// Verify genuine user file preserved
	if !util.FileExists(userFile) {
		t.Errorf("expected user file %s to be preserved, but it was removed", userFile)
	}
}
