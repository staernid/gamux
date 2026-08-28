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
	if g.EffectiveOperatingMode() != ModeStandalone {
		t.Errorf("expected default ModeStandalone, got %s", g.EffectiveOperatingMode())
	}
	if g.EffectiveRunner() != RunnerAuto {
		t.Errorf("expected default RunnerAuto, got %s", g.EffectiveRunner())
	}

	// Override with LaunchProfile
	g.LaunchProfile = &LaunchProfile{
		ExePath:       "/games/test/custom.sh",
		ExeArgs:       "--custom",
		OperatingMode: ModePortable,
		Runner:        RunnerLinux,
		WinePrefix:    "/prefixes/custom",
		PlaytimeSeconds: 3600,
		LastPlayed:    1700000000,
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

