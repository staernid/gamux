package steamshortcut

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/staernid/gamux/config"
)

func mustTempDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("", "steamshortcut-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

// ── SteamUserdataDir ────────────────────────────────────────────────

func TestSteamUserdataDir_NoNumericDirs(t *testing.T) {
	// Override the SteamUserdata config var for this test
	oldSteamUserdata := config.SteamUserdata
	defer func() { config.SteamUserdata = oldSteamUserdata }()

	// Point to a temp dir with no numeric subdirs
	tmpDir := mustTempDir(t)
	config.SteamUserdata = filepath.Join(tmpDir, "nonexistent")

	_, err := SteamUserdataDir()
	if err == nil {
		t.Fatal("expected error for non-existent Steam userdata dir")
	}
}

func TestSteamUserdataDir_FindsNumericDir(t *testing.T) {
	oldSteamUserdata := config.SteamUserdata
	defer func() { config.SteamUserdata = oldSteamUserdata }()

	// SteamUserdataDir joins home + config.SteamUserdata, so we need the
	// expected path to exist relative to home.
	home, _ := os.UserHomeDir()
	relPath := filepath.Join(home, "tmp", "steamshortcut-test-steam")
	os.MkdirAll(relPath, 0755)
	config.SteamUserdata = "tmp/steamshortcut-test-steam"

	// Create a fake numeric user dir
	userDir := filepath.Join(relPath, "12345")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		t.Fatal(err)
	}

	dir, err := SteamUserdataDir()
	if err != nil {
		t.Fatalf("SteamUserdataDir failed: %v", err)
	}
	if dir != userDir {
		t.Errorf("SteamUserdataDir() = %q, want %q", dir, userDir)
	}
}

func TestSteamUserdataDir_PicksMostRecent(t *testing.T) {
	oldSteamUserdata := config.SteamUserdata
	defer func() { config.SteamUserdata = oldSteamUserdata }()

	// SteamUserdataDir joins home + config.SteamUserdata
	home, _ := os.UserHomeDir()
	relPath := filepath.Join(home, "tmp", "steamshortcut-test-mostrecent")
	os.MkdirAll(relPath, 0755)
	config.SteamUserdata = "tmp/steamshortcut-test-mostrecent"

	// Create two numeric dirs, modify one to be more recent
	userA := filepath.Join(relPath, "11111")
	userB := filepath.Join(relPath, "22222")
	os.MkdirAll(userA, 0755)
	os.MkdirAll(userB, 0755)

	// Set userB to be more recent
	os.WriteFile(filepath.Join(userB, ".dummy"), []byte("test"), 0644)

	dir, err := SteamUserdataDir()
	if err != nil {
		t.Fatalf("SteamUserdataDir failed: %v", err)
	}
	if dir != userB {
		t.Errorf("SteamUserdataDir() = %q, want %q (most recent)", dir, userB)
	}
}

// ── DesktopEntry ────────────────────────────────────────────────────

func TestDesktopEntry_WritesFile(t *testing.T) {
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", mustTempDir(t))
	defer os.Setenv("HOME", oldHome)

	cfg := ShortcutConfig{
		Name:    "Test Game",
		ExePath: "/usr/games/test",
		AppID:   "12345",
	}

	path, err := DesktopEntry(cfg, false)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !contains(content, "Name=Test Game") {
		t.Error("desktop file should contain Name=Test Game")
	}
	if !contains(content, "steam://rungameid/12345") {
		t.Error("desktop file should contain steam://rungameid/12345")
	}
	if !contains(content, "[Desktop Entry]") {
		t.Error("desktop file should contain [Desktop Entry]")
	}
	if !contains(content, "Categories=Game;") {
		t.Error("desktop file should contain Categories=Game")
	}
}

func TestDesktopEntry_DryRun(t *testing.T) {
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", mustTempDir(t))
	defer os.Setenv("HOME", oldHome)

	cfg := ShortcutConfig{
		Name:    "Dry Run Game",
		ExePath: "/usr/games/dryrun",
	}

	path, err := DesktopEntry(cfg, true)
	if err != nil {
		t.Fatal(err)
	}

	// File should NOT exist (dry run)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("dry run should not create file")
	}
}

func TestDesktopEntry_SlugFromName(t *testing.T) {
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", mustTempDir(t))
	defer os.Setenv("HOME", oldHome)

	cfg := ShortcutConfig{
		Name:    "My Super Cool Game!",
		ExePath: "/usr/games/my-super-cool-game",
	}

	path, err := DesktopEntry(cfg, false)
	if err != nil {
		t.Fatal(err)
	}

	expectedFile := "steam-my-super-cool-game.desktop"
	expectedPath := filepath.Join(os.Getenv("HOME"), ".local", "share", "applications", expectedFile)
	if path != expectedPath {
		t.Errorf("DesktopEntry() path = %q, want %q", path, expectedPath)
	}
}

// ── AddShortcut ─────────────────────────────────────────────────────

func TestAddShortcut_Validation(t *testing.T) {
	_, err := SteamUserdataDir() // Will fail if no Steam installed, but that's OK

	// Empty name
	err = AddShortcut(ShortcutConfig{ExePath: "/tmp/game"}, false)
	if err == nil || !contains(err.Error(), "Name") {
		t.Error("expected Name error")
	}

	// Empty path
	err = AddShortcut(ShortcutConfig{Name: "Test"}, false)
	if err == nil || !contains(err.Error(), "ExePath") {
		t.Error("expected ExePath error")
	}
}

func TestAddShortcut_DryRun(t *testing.T) {
	cfg := ShortcutConfig{
		Name:    "Dry Run Test",
		ExePath: "/usr/games/drytest",
		AppID:   "54321",
	}

	err := AddShortcut(cfg, true)
	if err != nil {
		t.Fatalf("dry run should not error: %v", err)
	}
}

// ── Slug helper (used internally) ───────────────────────────────────

func TestSlugFromName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Test Game", "test-game"},
		{"My Game!", "my-game"},
		{"Special @#$%^& Characters", "special--------characters"},
		{"", "game"},
	}

	for _, tt := range tests {
		slug := strings.ToLower(tt.name)
		slug = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
				return r
			}
			return '-'
		}, slug)
		slug = strings.Trim(slug, "-")
		if slug == "" {
			slug = "game"
		}
		if slug != tt.want {
			t.Errorf("slug(%q) = %q, want %q", tt.name, slug, tt.want)
		}
	}
}

// ── Binary VDF write ───────────────────────────────────────────────

func TestVDFWrite_CreatesValidFile(t *testing.T) {
	// Create a minimal binary VDF in memory
	root := &vdfNode{
		Key:      "shortcuts",
		NodeType: 1,
		Children: []vdfChild{
			{
				Key: "0",
				Child: &vdfNode{
					NodeType: 1,
					Children: []vdfChild{
						{Key: "Name", Value: vdfValue{Type: vdfTypeString, Value: "RoundTrip Game"}},
						{Key: "Exe", Value: vdfValue{Type: vdfTypeString, Value: "/tmp/test.exe"}},
						{Key: "AppID", Value: vdfValue{Type: vdfTypeInt32, Int64: 12345}},
						{Key: "IsHidden", Value: vdfValue{Type: vdfTypeInt32, Int64: 0}},
					},
				},
			},
		},
	}

	// Write to temp file
	tmpDir := mustTempDir(t)
	vdfPath := filepath.Join(tmpDir, "shortcuts.vdf")
	if err := writeVDF(vdfPath, root); err != nil {
		t.Fatal(err)
	}

	// Verify the file starts with the magic header
	data, err := os.ReadFile(vdfPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(data[:4]) != string(magicHeader) {
		t.Errorf("VDF file should start with magic header, got %x", data[:4])
	}

	// Verify file is non-empty
	if len(data) < 20 {
		t.Errorf("VDF file too small: %d bytes", len(data))
	}
}

// ── Helpers ─────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
