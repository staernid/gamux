package domain

import (
	"testing"
)

func TestGame_StateHelpers(t *testing.T) {
	g := NewGame("Cyberpunk 2077", "/games/cyberpunk", "1091500", "windows")

	if g.NumericAppID != 1091500 {
		t.Errorf("expected numeric AppID 1091500, got %d", g.NumericAppID)
	}
	if !g.IsClean() {
		t.Error("newly created game should be clean")
	}

	g.OriginalBackups = append(g.OriginalBackups, "/games/cyberpunk/steam_api64.dll.ORIGINAL")
	if !g.HasBackups() {
		t.Error("expected HasBackups to be true")
	}
	if !g.IsPortablePatched() {
		t.Error("expected IsPortablePatched to be true when backups exist")
	}
	if g.IsClean() {
		t.Error("game with backups should not be clean")
	}

	g.LaunchProfile = &LaunchProfile{
		OperatingMode: ModeLoader,
	}
	if !g.IsLoaderConfigured() {
		t.Error("expected IsLoaderConfigured to be true for ModeLoader in LaunchProfile")
	}
	if g.EffectiveOperatingMode() != ModeLoader {
		t.Errorf("expected EffectiveOperatingMode to be ModeLoader, got %s", g.EffectiveOperatingMode())
	}
}

func TestGame_MultiStoreAndArtwork(t *testing.T) {
	g := NewGame("The Witcher 3", "/games/witcher3", "", "windows")
	g.Source = SourceGOG
	g.GOGID = "1207664643"
	g.ExternalIDs.SteamAppID = 292030
	g.CoverURL = "https://example.com/cover.jpg"
	g.LocalCover = "/local/cover.jpg"

	if g.Source != SourceGOG {
		t.Errorf("expected SourceGOG, got %s", g.Source)
	}
	if g.GOGID != "1207664643" {
		t.Errorf("expected GOGID 1207664643, got %s", g.GOGID)
	}
	if id, ok := g.SteamAppID(); !ok || id != 292030 {
		t.Errorf("expected SteamAppID 292030, got %d (ok: %v)", id, ok)
	}
	if g.CoverURL != "https://example.com/cover.jpg" || g.LocalCover != "/local/cover.jpg" {
		t.Errorf("unexpected artwork fields: %+v", g.GameArtwork)
	}
}

func TestGame_EffectiveAccessors(t *testing.T) {
	g := NewGame("Test Game", "/games/test", "123", "linux")
	g.GameRecord.ExePath = "/games/test/start.sh"
	g.GameRecord.ExeArgs = "--default"

	if g.EffectiveExePath() != "/games/test/start.sh" {
		t.Errorf("expected default ExePath from GameRecord, got %s", g.EffectiveExePath())
	}
	if g.EffectiveExeArgs() != "--default" {
		t.Errorf("expected default ExeArgs from GameRecord, got %s", g.EffectiveExeArgs())
	}
	if g.EffectiveOperatingMode() != "" {
		t.Errorf("expected unset EffectiveOperatingMode (empty string), got %s", g.EffectiveOperatingMode())
	}
	if g.EffectiveRunner() != RunnerAuto {
		t.Errorf("expected default RunnerAuto, got %s", g.EffectiveRunner())
	}

	// Override with LaunchProfile
	g.LaunchProfile = &LaunchProfile{
		ExePath:         "/games/test/custom.sh",
		ExeArgs:         "--custom",
		OperatingMode:   ModePortable,
		Runner:          RunnerLinux,
		WinePrefix:      "/prefixes/custom",
		PlaytimeSeconds: 3600,
		LastPlayed:      1700000000,
		InstalledAt:     1680000000,
	}

	if g.EffectiveExePath() != "/games/test/custom.sh" {
		t.Errorf("expected profile override ExePath, got %s", g.EffectiveExePath())
	}
	if g.EffectiveExeArgs() != "--custom" {
		t.Errorf("expected profile override ExeArgs, got %s", g.EffectiveExeArgs())
	}
	if g.EffectiveOperatingMode() != ModePortable {
		t.Errorf("expected profile override ModePortable, got %s", g.EffectiveOperatingMode())
	}
	if g.EffectiveRunner() != RunnerLinux {
		t.Errorf("expected profile override RunnerLinux, got %s", g.EffectiveRunner())
	}
	if g.EffectiveWinePrefix() != "/prefixes/custom" {
		t.Errorf("expected profile override WinePrefix, got %s", g.EffectiveWinePrefix())
	}
	playtime, lastPlayed := g.EffectivePlaytime()
	if playtime != 3600 || lastPlayed != 1700000000 {
		t.Errorf("expected playtime 3600 / 1700000000, got %d / %d", playtime, lastPlayed)
	}
	if g.EffectiveInstalledAt() != 1680000000 {
		t.Errorf("expected installed at 1680000000, got %d", g.EffectiveInstalledAt())
	}
}

func TestGame_RecomputeStateAndMetadata(t *testing.T) {
	g := NewGame("Test Game", "/games/test", "480", "win64")
	g.RecomputeState()

	if g.State != "Original" || g.IsPatched || g.IsPortable {
		t.Errorf("expected Original state, got State=%s IsPatched=%v IsPortable=%v", g.State, g.IsPatched, g.IsPortable)
	}

	// Adding settings directory
	g.SettingsDirExists = true
	g.RecomputeState()
	if g.State != "Loader-Configured" || !g.IsPatched {
		t.Errorf("expected Loader-Configured state, got State=%s IsPatched=%v", g.State, g.IsPatched)
	}

	// Adding original backup
	g.OriginalBackups = []string{"/games/test/steam_api64.dll.ORIGINAL"}
	g.RecomputeState()
	if g.State != "Portable-Patched" || !g.IsPortable || !g.IsPatched {
		t.Errorf("expected Portable-Patched state, got State=%s IsPortable=%v IsPatched=%v", g.State, g.IsPortable, g.IsPatched)
	}

	// Metadata provider resolution
	if g.EffectiveMetadataProvider() != "steam" {
		t.Errorf("expected steam metadata provider for AppID 480, got %s", g.EffectiveMetadataProvider())
	}
	if g.EffectiveMetadataID() != "480" {
		t.Errorf("expected metadata ID 480, got %s", g.EffectiveMetadataID())
	}

	// Custom metadata provider override
	g.LaunchProfile = &LaunchProfile{
		MetadataProvider: "steamgriddb",
		MetadataID:       "9999",
	}
	if !g.HasCustomMetadata() {
		t.Error("expected HasCustomMetadata to be true")
	}
	if g.EffectiveMetadataProvider() != "steamgriddb" {
		t.Errorf("expected steamgriddb, got %s", g.EffectiveMetadataProvider())
	}
	if g.EffectiveMetadataID() != "9999" {
		t.Errorf("expected 9999, got %s", g.EffectiveMetadataID())
	}
}

func TestGame_AssetGetSet(t *testing.T) {
	g := NewGame("Test Game", "/games/test", "480", "win64")

	g.SetAsset(AssetCover, AssetLocation{
		RemoteURL: "https://example.com/cover.jpg",
		LocalPath: "/local/cover.jpg",
	})
	coverLoc := g.GetAsset(AssetCover)
	if coverLoc.RemoteURL != "https://example.com/cover.jpg" || coverLoc.LocalPath != "/local/cover.jpg" {
		t.Errorf("unexpected cover location: %+v", coverLoc)
	}

	g.SetAsset(AssetHero, AssetLocation{
		CustomURL: "https://example.com/custom_hero.jpg",
	})
	heroLoc := g.GetAsset(AssetHero)
	if heroLoc.CustomURL != "https://example.com/custom_hero.jpg" {
		t.Errorf("unexpected hero location: %+v", heroLoc)
	}
}

func TestNormalizeOperatingMode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"portable", ModePortable},
		{"injected", ModePortable},
		{"Injected", ModePortable},
		{"patch", ModePortable},
		{"portable-patched", ModePortable},
		{"loader", ModeLoader},
		{"LOADER", ModeLoader},
		{"loader-configured", ModeLoader},
		{"none", ModeNone},
		{"standalone", ModeNone},
		{"clean", ModeNone},
		{"drm-free", ModeNone},
		{"native", ModeNative},
	}

	for _, tt := range tests {
		actual := NormalizeOperatingMode(tt.input)
		if actual != tt.expected {
			t.Errorf("NormalizeOperatingMode(%q) = %q, expected %q", tt.input, actual, tt.expected)
		}
	}
}

func TestGame_InjectedOperatingMode(t *testing.T) {
	g := NewGame("Injected Game", "/games/injected", "1091500", "win64")
	g.LaunchProfile = &LaunchProfile{
		OperatingMode: "injected",
	}

	if g.EffectiveOperatingMode() != ModePortable {
		t.Errorf("expected EffectiveOperatingMode to normalize 'injected' to ModePortable, got %s", g.EffectiveOperatingMode())
	}
	if !g.IsPortablePatched() {
		t.Error("expected IsPortablePatched to be true for 'injected' mode")
	}

	g.RecomputeState()
	if g.State != "Portable-Patched" || !g.IsPortable || !g.IsPatched {
		t.Errorf("expected Portable-Patched state for injected mode, got State=%s, IsPortable=%v, IsPatched=%v", g.State, g.IsPortable, g.IsPatched)
	}
}

func TestLaunchProfile_MergeFrom(t *testing.T) {
	primary := &LaunchProfile{
		GameDir:       "/games/portal2",
		ExePath:       "portal2.exe",
		OperatingMode: ModeLoader,
		CustomEnv: map[string]string{
			"EXISTING_VAR": "val1",
		},
	}

	fallback := &LaunchProfile{
		Name:            "Portal 2",
		AppID:           "620",
		Platform:        "win64",
		ExePath:         "fallback.exe", // should NOT overwrite primary
		OperatingMode:   ModePortable,   // should NOT overwrite primary
		Runner:          RunnerUMU,
		PlaytimeSeconds: 7200,
		LastPlayed:      1700000000,
		InstalledAt:     1650000000,
		CustomCoverURL:  "https://example.com/cover.jpg",
		LockedArtwork:   true,
		CustomEnv: map[string]string{
			"EXISTING_VAR": "should_not_overwrite",
			"NEW_VAR":      "val2",
		},
	}

	primary.MergeFrom(fallback)

	if primary.Name != "Portal 2" {
		t.Errorf("expected Name 'Portal 2', got %q", primary.Name)
	}
	if primary.AppID != "620" {
		t.Errorf("expected AppID '620', got %q", primary.AppID)
	}
	if primary.Platform != "win64" {
		t.Errorf("expected Platform 'win64', got %q", primary.Platform)
	}
	if primary.ExePath != "portal2.exe" {
		t.Errorf("expected ExePath to remain 'portal2.exe', got %q", primary.ExePath)
	}
	if primary.OperatingMode != ModeLoader {
		t.Errorf("expected OperatingMode to remain ModeLoader, got %q", primary.OperatingMode)
	}
	if primary.Runner != RunnerUMU {
		t.Errorf("expected Runner 'umu', got %q", primary.Runner)
	}
	if primary.PlaytimeSeconds != 7200 || primary.LastPlayed != 1700000000 || primary.InstalledAt != 1650000000 {
		t.Errorf("playtime or timestamp fields not merged properly")
	}
	if primary.CustomCoverURL != "https://example.com/cover.jpg" || !primary.LockedArtwork {
		t.Errorf("artwork fields not merged properly")
	}
	if primary.CustomEnv["EXISTING_VAR"] != "val1" || primary.CustomEnv["NEW_VAR"] != "val2" {
		t.Errorf("custom env not merged properly: %+v", primary.CustomEnv)
	}
}

func TestGame_ApplyProfile(t *testing.T) {
	g := NewGame("Skyrim", "/games/skyrim", "489830", "win64")
	profile := &LaunchProfile{
		Store:            "Steam",
		MetadataProvider: "steam",
		MetadataID:       "489830",
		CustomCoverURL:   "https://example.com/skyrim_cover.jpg",
		LockedArtwork:    true,
		PlaytimeSeconds:  12345,
		LastPlayed:       1710000000,
	}

	g.ApplyProfile(profile, true)

	if g.CustomCoverURL != "https://example.com/skyrim_cover.jpg" {
		t.Errorf("expected CustomCoverURL applied to game, got %q", g.CustomCoverURL)
	}
	if !g.LockedArtwork {
		t.Error("expected LockedArtwork applied to game")
	}
	if g.PlaytimeSeconds != 12345 || g.LastPlayed != 1710000000 {
		t.Errorf("expected playtime 12345 / 1710000000, got %d / %d", g.PlaytimeSeconds, g.LastPlayed)
	}
}
