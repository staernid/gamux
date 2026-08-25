package domain

import (
	"testing"
)

func TestGameInvariants(t *testing.T) {
	g := NewGame("Portal 2", "/games/portal2", "620", "win64")

	if g.Name != "Portal 2" {
		t.Errorf("expected name 'Portal 2', got %q", g.Name)
	}
	if g.NumericAppID != 620 {
		t.Errorf("expected numeric AppID 620, got %d", g.NumericAppID)
	}
	if !g.IsSteam() {
		t.Errorf("expected IsSteam() to be true for AppID 620")
	}
	if !g.IsClean() {
		t.Errorf("expected brand new game to be clean")
	}

	appID, ok := g.SteamAppID()
	if !ok || appID != 620 {
		t.Errorf("expected SteamAppID 620, got (%d, %v)", appID, ok)
	}

	// Test patch state mutation
	g.OriginalBackups = append(g.OriginalBackups, "/games/portal2/steam_api64.dll.ORIGINAL")
	g.RecomputeState()

	if g.IsClean() {
		t.Errorf("expected game with backups to not be clean")
	}
	if !g.IsPortablePatched() {
		t.Errorf("expected IsPortablePatched to be true")
	}
	if !g.IsPortable {
		t.Errorf("expected IsPortable to be true")
	}
}

func TestNonSteamGameInvariants(t *testing.T) {
	g := NewGame("Cyberpunk DRM-Free", "/games/cp2077", "", "win64")
	g.Store = StoreGOG

	if g.IsSteam() {
		t.Errorf("expected GOG game with empty AppID to not be Steam")
	}
	if g.RequiresSteamEmulator() {
		t.Errorf("expected GOG game without Steam AppID to not require steam emulator")
	}

	g.ExePath = "/games/cp2077/bin/x64/Cyberpunk2077.exe"
	g.OperatingMode = string(ModeStandalone)
	g.RecomputeState()

	if !g.IsReady {
		t.Errorf("expected standalone game with ExePath to be ready")
	}
}
