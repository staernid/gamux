package profiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/staernid/gamux/config"
)

func TestProfileManager_SeedAndList(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-create isolated configs.user.ini for P2
	p2GseDir := filepath.Join(tmpDir, "p2", "home", ".local", "share", "GSE Saves", "settings")
	_ = os.MkdirAll(p2GseDir, 0755)
	_ = os.WriteFile(filepath.Join(p2GseDir, "configs.user.ini"), []byte("[user::general]\naccount_name=Buddy\naccount_steamid=76561198000000002\nlanguage=english\nip_country=US\n"), 0644)

	cfg := config.DefaultConfig()

	mgr, err := NewManager(tmpDir, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	list := mgr.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(list))
	}

	mainProf := list[0]
	if !mainProf.IsMain || mainProf.ID != "main" {
		t.Errorf("expected first profile to be main, got %+v", mainProf)
	}

	p2Prof := list[1]
	if p2Prof.IsMain || p2Prof.ID != "p2" || p2Prof.Name != "Buddy" {
		t.Errorf("expected p2 profile with Buddy name, got %+v", p2Prof)
	}

	active := mgr.GetActive()
	if active == nil || active.ID != "main" {
		t.Errorf("expected active profile to be main, got %+v", active)
	}
}

func TestProfileManager_CreateAndUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	p3, err := mgr.Create("PlayerThree", "76561198000000003", "german", "DE", "")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if p3.ID != "p3" || p3.Index != 3 {
		t.Errorf("expected p3 with index 3, got %+v", p3)
	}

	prefix := mgr.GetPrefixDir(p3.ID)
	if !strings.HasSuffix(prefix, filepath.Join("Games", "umu-p3")) {
		t.Errorf("expected prefix ending in Games/umu-p3, got %s", prefix)
	}

	// Update p3
	p3.Name = "RenamedThree"
	if err := mgr.Update(p3); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	updated, err := mgr.Get("p3")
	if err != nil {
		t.Fatalf("Get p3 failed: %v", err)
	}
	if updated.Name != "RenamedThree" {
		t.Errorf("expected updated name RenamedThree, got %s", updated.Name)
	}
}

func TestProfileManager_DeleteAndSafety(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Trying to delete main must fail
	if err := mgr.Delete("main"); err == nil {
		t.Errorf("expected error when deleting main profile, got nil")
	}

	// Create p3 and delete it
	p3, _ := mgr.Create("PlayerThree", "", "", "", "")
	if err := mgr.SetActive(p3.ID); err != nil {
		t.Fatalf("SetActive failed: %v", err)
	}
	if mgr.GetActive().ID != "p3" {
		t.Fatalf("expected active to be p3")
	}

	if err := mgr.Delete("p3"); err != nil {
		t.Fatalf("Delete p3 failed: %v", err)
	}

	// Active should reset to main
	if mgr.GetActive().ID != "main" {
		t.Errorf("expected active to reset to main after deleting active profile, got %s", mgr.GetActive().ID)
	}
}

func TestProfileManager_Materialize(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	p2, err := mgr.Get("p2")
	if err != nil {
		t.Fatalf("Get p2 failed: %v", err)
	}

	if err := mgr.Materialize(p2.ID); err != nil {
		t.Fatalf("Materialize p2 failed: %v", err)
	}

	homeDir := mgr.GetHomeDir(p2.ID)
	iniFile := filepath.Join(homeDir, ".local", "share", "GSE Saves", "settings", "configs.user.ini")
	if _, err := os.Stat(iniFile); os.IsNotExist(err) {
		t.Fatalf("expected configs.user.ini to exist at %s", iniFile)
	}

	data, _ := os.ReadFile(iniFile)
	if !strings.Contains(string(data), p2.SteamID) {
		t.Errorf("expected ini to contain steam ID %s, got: %s", p2.SteamID, string(data))
	}
}

func TestResolveProfileHome_And_Env(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, ".local", "share"))

	// Main profile should return host home and empty env overrides
	mainHome := ResolveProfileHome("main")
	if mainHome != tmpDir {
		t.Errorf("expected mainHome %s, got %s", tmpDir, mainHome)
	}
	mainEnv := ResolveProfileEnv("main")
	if len(mainEnv) != 0 {
		t.Errorf("expected empty env for main, got %+v", mainEnv)
	}

	// P2 before creation of directory
	p2Home := ResolveProfileHome("p2")
	expectedP2Home := filepath.Join(tmpDir, ".local", "share", "gamux", "profiles", "p2", "home")
	if p2Home != expectedP2Home {
		t.Errorf("expected p2Home %s, got %s", expectedP2Home, p2Home)
	}

	// Before directory exists, env should be empty
	if env := ResolveProfileEnv("p2"); len(env) != 0 {
		t.Errorf("expected empty env when profileHome does not exist, got %+v", env)
	}

	// After directory exists, env should contain HOME and XDG_*
	_ = os.MkdirAll(expectedP2Home, 0755)
	env := ResolveProfileEnv("p2")
	if env["HOME"] != expectedP2Home {
		t.Errorf("expected HOME %s, got %s", expectedP2Home, env["HOME"])
	}
	if env["XDG_DATA_HOME"] != filepath.Join(expectedP2Home, ".local", "share") {
		t.Errorf("expected XDG_DATA_HOME %s, got %s", filepath.Join(expectedP2Home, ".local", "share"), env["XDG_DATA_HOME"])
	}
}
