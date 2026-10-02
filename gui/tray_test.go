package gui

import (
	"sync/atomic"
	"testing"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
)

func TestTraySortingAndLimiting(t *testing.T) {
	games := []TrayGame{
		{Name: "Game Low", GameDir: "/games/low", LastPlayed: 100},
		{Name: "Game Unplayed", GameDir: "/games/unplayed", LastPlayed: 0},
		{Name: "Game Top", GameDir: "/games/top", LastPlayed: 500},
		{Name: "Game Mid", GameDir: "/games/mid", LastPlayed: 300},
		{Name: "Game Second", GameDir: "/games/second", LastPlayed: 400},
	}

	sorted := SortTrayGames(games, 3)
	if len(sorted) != 3 {
		t.Fatalf("expected 3 games, got %d", len(sorted))
	}
	if sorted[0].Name != "Game Top" || sorted[0].LastPlayed != 500 {
		t.Errorf("expected sorted[0] to be Game Top (500), got %+v", sorted[0])
	}
	if sorted[1].Name != "Game Second" || sorted[1].LastPlayed != 400 {
		t.Errorf("expected sorted[1] to be Game Second (400), got %+v", sorted[1])
	}
	if sorted[2].Name != "Game Mid" || sorted[2].LastPlayed != 300 {
		t.Errorf("expected sorted[2] to be Game Mid (300), got %+v", sorted[2])
	}

	// Tie-break test
	tieGames := []TrayGame{
		{Name: "Zeta", GameDir: "/games/zeta", LastPlayed: 1000},
		{Name: "Alpha", GameDir: "/games/alpha", LastPlayed: 1000},
	}
	sortedTie := SortTrayGames(tieGames, 3)
	if len(sortedTie) != 2 || sortedTie[0].Name != "Alpha" || sortedTie[1].Name != "Zeta" {
		t.Errorf("expected alphabetical tie-break, got %+v", sortedTie)
	}

	// Domain game test
	dGames := []*domain.Game{
		{
			GameRecord: domain.GameRecord{
				GameDir:    "/games/alpha",
				Name:       "Alpha Game",
				LastPlayed: 100,
			},
		},
		{
			GameRecord: domain.GameRecord{
				GameDir:    "/games/beta",
				Name:       "Beta Base",
				LastPlayed: 50,
			},
			LaunchProfile: &domain.LaunchProfile{
				GameDir:    "/games/beta",
				Name:       "Beta Profile Name",
				LastPlayed: 900,
			},
		},
		{
			GameRecord: domain.GameRecord{
				GameDir:    "/games/never",
				Name:       "Never Played",
				LastPlayed: 0,
			},
		},
	}
	fromDomain := FilterAndSortRecentGames(dGames, 3)
	if len(fromDomain) != 2 {
		t.Fatalf("expected 2 played domain games, got %d", len(fromDomain))
	}
	if fromDomain[0].Name != "Beta Profile Name" || fromDomain[0].LastPlayed != 900 {
		t.Errorf("expected profile name and timestamp override, got %+v", fromDomain[0])
	}
	if fromDomain[1].Name != "Alpha Game" || fromDomain[1].LastPlayed != 100 {
		t.Errorf("expected alpha game second, got %+v", fromDomain[1])
	}
}

func TestTrayManagerCallbacksAndGames(t *testing.T) {
	var activated atomic.Bool
	var quitted atomic.Bool
	var launchedDir atomic.Value

	cb := TrayCallbacks{
		OnActivate: func() {
			activated.Store(true)
		},
		OnQuit: func() {
			quitted.Store(true)
		},
		OnLaunchGame: func(gameDir string) {
			launchedDir.Store(gameDir)
		},
		GetRecentGames: func() []TrayGame {
			return []TrayGame{
				{Name: "Game 1", GameDir: "/games/1", LastPlayed: 100},
			}
		},
	}

	tray, err := StartTray(cb)
	if err != nil {
		t.Fatalf("StartTray failed: %v", err)
	}
	defer tray.Close()

	if len(tray.recentGames) != 1 {
		t.Fatalf("expected 1 initial recent game, got %d", len(tray.recentGames))
	}

	// Update recent games
	updated := tray.SetRecentGames([]TrayGame{
		{Name: "Half-Life 2", GameDir: "/games/hl2", LastPlayed: 300},
		{Name: "Cyberpunk 2077", GameDir: "/games/cp2077", LastPlayed: 200},
		{Name: "Portal", GameDir: "/games/portal", LastPlayed: 100},
		{Name: "Old Game", GameDir: "/games/old", LastPlayed: 10},
	})
	if !updated {
		t.Errorf("expected SetRecentGames to return true")
	}

	if len(tray.recentGames) != 3 {
		t.Fatalf("expected exactly 3 recent games capped, got %d", len(tray.recentGames))
	}
	if tray.recentGames[0].Name != "Half-Life 2" {
		t.Errorf("expected Half-Life 2 first, got %s", tray.recentGames[0].Name)
	}

	// Verify callbacks invocation
	tray.cb.OnActivate()
	if !activated.Load() {
		t.Errorf("expected activated to be true")
	}

	tray.cb.OnQuit()
	if !quitted.Load() {
		t.Errorf("expected quitted to be true")
	}

	tray.cb.OnLaunchGame("/games/hl2")
	if launchedDir.Load() != "/games/hl2" {
		t.Errorf("expected launchedDir to be /games/hl2, got %v", launchedDir.Load())
	}
}

func TestAppCachedGamesAndRecentGames(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		LibraryPaths: []string{tmpDir},
	}
	app := NewApp(cfg)

	// Initially empty in temp directory
	recent := app.GetRecentGames(3)
	if len(recent) != 0 {
		t.Fatalf("expected 0 recent games initially, got %d", len(recent))
	}

	g1 := &domain.Game{
		GameRecord: domain.GameRecord{
			GameDir:    "/library/game1",
			Name:       "Game One",
			LastPlayed: 100,
		},
	}
	g2 := &domain.Game{
		GameRecord: domain.GameRecord{
			GameDir:    "/library/game2",
			Name:       "Game Two",
			LastPlayed: 200,
		},
	}

	app.UpdateCachedGames([]*domain.Game{g1, g2})
	recent = app.GetRecentGames(3)
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent games, got %d", len(recent))
	}
	if recent[0].Name != "Game Two" || recent[1].Name != "Game One" {
		t.Errorf("unexpected sort order: %+v", recent)
	}

	// Now record play session on game1 with newer timestamp
	app.RecordGamePlayed("/library/game1", 500, 3600)
	recent = app.GetRecentGames(3)
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent games after play, got %d", len(recent))
	}
	if recent[0].Name != "Game One" || recent[0].LastPlayed != 500 {
		t.Errorf("expected Game One to be first after play session, got %+v", recent[0])
	}
}

func TestAppQuitAndIsQuitting(t *testing.T) {
	app := NewApp(nil)
	if app.IsQuitting() {
		t.Errorf("expected IsQuitting to be false initially")
	}

	app.SetQuitting(true)
	if !app.IsQuitting() {
		t.Errorf("expected IsQuitting to be true after SetQuitting(true)")
	}

	app.SetQuitting(false)
	if app.IsQuitting() {
		t.Errorf("expected IsQuitting to be false after SetQuitting(false)")
	}

	app.Quit()
	if !app.IsQuitting() {
		t.Errorf("expected Quit() to set IsQuitting to true")
	}
}
