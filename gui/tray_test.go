package gui

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
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

func TestTrayMenuItemsAndEvents(t *testing.T) {
	var activated atomic.Bool
	var quitted atomic.Bool

	tray := &TrayManager{
		onActivate: func() {
			activated.Store(true)
		},
		onQuit: func() {
			quitted.Store(true)
		},
	}

	items := tray.menuItems()
	if len(items) != 3 {
		t.Fatalf("expected 3 menu items without recent games, got %d", len(items))
	}
	if items[0].ID != 1 || items[1].ID != 2 || items[2].ID != 3 {
		t.Errorf("unexpected item IDs: %v", items)
	}

	menuRecv := &dbusMenuReceiver{tray: tray}
	sniRecv := &sniReceiver{tray: tray}

	// 1. Test Activate via SNI
	if err := sniRecv.Activate(0, 0); err != nil {
		t.Errorf("sniRecv.Activate returned error: %v", err)
	}
	if !activated.Load() {
		t.Errorf("expected onActivate to be called on sniRecv.Activate")
	}

	// Reset
	activated.Store(false)

	// 2. Test Activate via DBusMenu (item ID 1)
	if err := menuRecv.Event(1, "clicked", dbus.MakeVariant(0), 0); err != nil {
		t.Errorf("menuRecv.Event(1) returned error: %v", err)
	}
	if !activated.Load() {
		t.Errorf("expected onActivate to be called on menuRecv.Event for item 1")
	}

	// 3. Test Quit via DBusMenu (item ID 3)
	if err := menuRecv.Event(3, "clicked", dbus.MakeVariant(0), 0); err != nil {
		t.Errorf("menuRecv.Event(3) returned error: %v", err)
	}
	for i := 0; i < 50; i++ {
		if quitted.Load() {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !quitted.Load() {
		t.Errorf("expected onQuit to be called on menuRecv.Event for item 3")
	}

	// 4. Test GetLayout
	rev, layout, err := menuRecv.GetLayout(0, -1, nil)
	if err != nil {
		t.Fatalf("menuRecv.GetLayout returned error: %v", err)
	}
	if rev != 1 {
		t.Errorf("expected revision 1, got %d", rev)
	}
	if layout.ID != 0 || len(layout.Children) != 3 {
		t.Errorf("unexpected layout: %+v", layout)
	}

	// 5. Test GetProperty
	val, pErr := menuRecv.GetProperty(1, "label")
	if pErr != nil {
		t.Fatalf("menuRecv.GetProperty returned error: %v", pErr)
	}
	if val.Value() != "Show Gamux" {
		t.Errorf("expected label 'Show Gamux', got %v", val.Value())
	}
}

func TestTrayRecentGamesMenuAndLaunch(t *testing.T) {
	var launchedDir atomic.Value

	tray := &TrayManager{
		onLaunchGame: func(gameDir string) {
			launchedDir.Store(gameDir)
		},
	}

	// Add 3 recent games
	updated := tray.SetRecentGames([]TrayGame{
		{Name: "Half_Life_2", GameDir: "/games/hl2", LastPlayed: 300},
		{Name: "Cyberpunk 2077", GameDir: "/games/cp2077", LastPlayed: 200},
		{Name: "Portal", GameDir: "/games/portal", LastPlayed: 100},
	})
	if !updated {
		t.Errorf("expected SetRecentGames to return true when updated")
	}

	items := tray.menuItems()
	// 3 recent games + 1 separator + 1 Show Gamux + 1 separator + 1 Quit = 7 items
	if len(items) != 7 {
		t.Fatalf("expected 7 menu items, got %d", len(items))
	}

	// Check topmost items are the 3 games
	if items[0].ID != 10 || items[0].Props["label"].Value() != "Half__Life__2" {
		t.Errorf("item 0 mismatch: %+v", items[0])
	}
	if items[1].ID != 11 || items[1].Props["label"].Value() != "Cyberpunk 2077" {
		t.Errorf("item 1 mismatch: %+v", items[1])
	}
	if items[2].ID != 12 || items[2].Props["label"].Value() != "Portal" {
		t.Errorf("item 2 mismatch: %+v", items[2])
	}
	if items[3].ID != 4 || items[3].Props["type"].Value() != "separator" {
		t.Errorf("item 3 separator mismatch: %+v", items[3])
	}
	if items[4].ID != 1 || items[4].Props["label"].Value() != "Show Gamux" {
		t.Errorf("item 4 Show Gamux mismatch: %+v", items[4])
	}

	menuRecv := &dbusMenuReceiver{tray: tray}

	// Test clicking recent game (ID 10)
	if err := menuRecv.Event(10, "clicked", dbus.MakeVariant(0), 0); err != nil {
		t.Errorf("Event(10) returned error: %v", err)
	}

	// Allow goroutine to run
	for i := 0; i < 20; i++ {
		if launchedDir.Load() != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if launchedDir.Load() != "/games/hl2" {
		t.Errorf("expected launched game to be /games/hl2, got %v", launchedDir.Load())
	}

	// Test AboutToShow dynamic refresh
	dynamicGames := []TrayGame{
		{Name: "New Game", GameDir: "/games/new", LastPlayed: 9999},
	}
	tray.getRecentGames = func() []TrayGame {
		return dynamicGames
	}

	needUpdate, err := menuRecv.AboutToShow(0)
	if err != nil {
		t.Errorf("AboutToShow(0) returned error: %v", err)
	}
	if !needUpdate {
		t.Errorf("expected AboutToShow to report needUpdate=true when games changed")
	}

	// Calling AboutToShow again with no change should return needUpdate=false
	needUpdate2, _ := menuRecv.AboutToShow(0)
	if needUpdate2 {
		t.Errorf("expected AboutToShow to report needUpdate=false when games did not change")
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

