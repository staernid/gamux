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
