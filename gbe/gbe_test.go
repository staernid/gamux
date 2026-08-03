package gbe

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/staernid/gamux/config"
)

func TestApplyGBE_InvalidPlatform(t *testing.T) {
	err := ApplyGBE(context.Background(), config.DefaultConfig(), ".", "invalid_platform", "123", true, false)
	if err == nil {
		t.Fatal("expected error for invalid platform")
	}
}

func TestApplyGBE_ExplicitTargetDirectory(t *testing.T) {
	// Create mock GBE directory structure
	tmpDir := t.TempDir()
	gbeDir := filepath.Join(tmpDir, "gbe_fork")
	linuxGbe := filepath.Join(gbeDir, "linux_release", "experimental", "x64")
	if err := os.MkdirAll(linuxGbe, 0755); err != nil {
		t.Fatal(err)
	}

	mockSourceLib := filepath.Join(linuxGbe, "libsteam_api.so")
	if err := os.WriteFile(mockSourceLib, []byte("emulated_gbe_lib"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create mock game directory in a completely separate folder
	gameDir := filepath.Join(tmpDir, "MyGame")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatal(err)
	}
	targetLib := filepath.Join(gameDir, "libsteam_api.so")
	if err := os.WriteFile(targetLib, []byte("original_steam_lib"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		GbeDir: gbeDir,
	}

	// Run ApplyGBE targeting gameDir in portable mode with dry-run
	err := ApplyGBE(context.Background(), cfg, gameDir, "linux", "12345", true, true)
	if err != nil {
		t.Fatalf("ApplyGBE failed: %v", err)
	}

	// Verify original file remained intact due to dry-run
	data, err := os.ReadFile(targetLib)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original_steam_lib" {
		t.Errorf("expected original lib content, got %s", string(data))
	}
}
