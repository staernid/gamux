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

	g.OperatingMode = ModeLoader
	if !g.IsLoaderConfigured() {
		t.Error("expected IsLoaderConfigured to be true for ModeLoader")
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

