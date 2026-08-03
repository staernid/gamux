package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/staernid/gamux/config"
)

func TestInspectStatus_Original(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "TestGame")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatal(err)
	}

	acfContent := "\"AppState\"\n{\n\t\"appid\"\t\"999\"\n\t\"name\"\t\"TestGame\"\n}"
	if err := os.WriteFile(filepath.Join(gameDir, "appmanifest_999.acf"), []byte(acfContent), 0644); err != nil {
		t.Fatal(err)
	}

	eng := New(config.DefaultConfig())
	status, err := eng.InspectStatus(context.Background(), gameDir)
	if err != nil {
		t.Fatalf("InspectStatus failed: %v", err)
	}

	if status.Name != "TestGame" {
		t.Errorf("expected Name 'TestGame', got %s", status.Name)
	}
	if status.State != "Original" {
		t.Errorf("expected State 'Original', got %s", status.State)
	}
}

func TestInspectStatus_PortablePatchedAndRollback(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "PatchedGame")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatal(err)
	}

	acfContent := "\"AppState\"\n{\n\t\"appid\"\t\"888\"\n\t\"name\"\t\"PatchedGame\"\n}"
	_ = os.WriteFile(filepath.Join(gameDir, "appmanifest_888.acf"), []byte(acfContent), 0644)

	// Create fake library and backup file
	libPath := filepath.Join(gameDir, "libsteam_api.so")
	_ = os.WriteFile(libPath, []byte("patched_lib"), 0755)

	backupPath := filepath.Join(gameDir, "libsteam_api.so.20260802-120000.ORIGINAL")
	_ = os.WriteFile(backupPath, []byte("original_lib"), 0755)

	// Create steam_settings directory
	settingsDir := filepath.Join(gameDir, "steam_settings")
	_ = os.MkdirAll(settingsDir, 0755)
	_ = os.WriteFile(filepath.Join(settingsDir, "configs.app.ini"), []byte("[app]"), 0644)

	eng := New(config.DefaultConfig())
	status, err := eng.InspectStatus(context.Background(), gameDir)
	if err != nil {
		t.Fatalf("InspectStatus failed: %v", err)
	}

	if status.State != "Portable-Patched" {
		t.Errorf("expected State 'Portable-Patched', got %s", status.State)
	}
	if len(status.OriginalBackups) != 1 {
		t.Errorf("expected 1 backup, got %d", len(status.OriginalBackups))
	}

	// Test Rollback
	err = eng.Rollback(context.Background(), gameDir, false)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Check restored content
	restoredData, err := os.ReadFile(libPath)
	if err != nil {
		t.Fatalf("failed to read restored lib: %v", err)
	}
	if string(restoredData) != "original_lib" {
		t.Errorf("expected restored content 'original_lib', got %s", string(restoredData))
	}

	// Check settings dir removed
	if _, err := os.Stat(settingsDir); !os.IsNotExist(err) {
		t.Errorf("expected steam_settings to be removed")
	}
}
