package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
)

func TestEngineModsManagement(t *testing.T) {
	tmpDir := t.TempDir()
	modsDir := filepath.Join(tmpDir, "mods")
	if err := os.MkdirAll(modsDir, 0755); err != nil {
		t.Fatalf("failed to create mods dir: %v", err)
	}

	modFile := filepath.Join(modsDir, "TestMod.pak")
	if err := os.WriteFile(modFile, []byte("mod content"), 0644); err != nil {
		t.Fatalf("failed to write test mod: %v", err)
	}

	reg := &detector.ModRegistry{
		Mods: []domain.InstalledMod{
			{
				ID:           "TestMod.pak",
				Name:         "Test Mod",
				WorkshopID:   99999,
				Enabled:      true,
				RelativePath: filepath.Join("mods", "TestMod.pak"),
			},
		},
	}
	if err := detector.SaveModRegistry(tmpDir, reg); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	eng := New(nil)
	ctx := context.Background()

	// 1. GetGameMods & warm cache via InspectLocal
	_ = os.WriteFile(filepath.Join(tmpDir, "game.exe"), []byte("MZ"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "steam_appid.txt"), []byte("400"), 0644)
	inspected, err := eng.InspectLocal(ctx, tmpDir)
	if err != nil {
		t.Fatalf("InspectLocal failed: %v", err)
	}
	if len(inspected.Mods) != 1 || !inspected.Mods[0].Enabled {
		t.Fatalf("expected 1 enabled mod in InspectLocal, got %+v", inspected.Mods)
	}

	status, err := eng.GetGameMods(ctx, tmpDir)
	if err != nil {
		t.Fatalf("GetGameMods failed: %v", err)
	}
	if len(status.Mods) != 1 {
		t.Fatalf("expected 1 mod, got %d", len(status.Mods))
	}
	if !status.Mods[0].Enabled {
		t.Errorf("expected mod to be enabled")
	}

	// 2. Toggle disable
	disabledMod, err := eng.ToggleGameMod(tmpDir, "TestMod.pak", false)
	if err != nil {
		t.Fatalf("ToggleGameMod(disable) failed: %v", err)
	}
	if disabledMod.Enabled {
		t.Errorf("expected mod to be disabled")
	}
	if _, err := os.Stat(filepath.Join(modsDir, "TestMod.pak.disabled")); err != nil {
		t.Errorf("expected .disabled file on disk: %v", err)
	}

	// Verify warm cache was invalidated so InspectLocal reflects disabled mod
	inspectedAfterDisable, err := eng.InspectLocal(ctx, tmpDir)
	if err != nil {
		t.Fatalf("InspectLocal after disable failed: %v", err)
	}
	if len(inspectedAfterDisable.Mods) != 1 || inspectedAfterDisable.Mods[0].Enabled {
		t.Errorf("expected InspectLocal to reflect disabled mod, got %+v", inspectedAfterDisable.Mods)
	}

	// Verify status after toggle
	status, _ = eng.GetGameMods(ctx, tmpDir)
	if len(status.Mods) != 1 || status.Mods[0].Enabled {
		t.Errorf("expected mod to be detected as disabled in GetGameMods: %+v", status.Mods)
	}

	// 3. Toggle enable
	enabledMod, err := eng.ToggleGameMod(tmpDir, "TestMod.pak", true)
	if err != nil {
		t.Fatalf("ToggleGameMod(enable) failed: %v", err)
	}
	if !enabledMod.Enabled {
		t.Errorf("expected mod to be enabled again")
	}
	if _, err := os.Stat(modFile); err != nil {
		t.Errorf("expected original file back on disk: %v", err)
	}

	// 4. DeleteGameMod
	if err := eng.DeleteGameMod(tmpDir, "TestMod.pak"); err != nil {
		t.Fatalf("DeleteGameMod failed: %v", err)
	}
	if _, err := os.Stat(modFile); !os.IsNotExist(err) {
		t.Errorf("expected mod file to be deleted")
	}

	status, _ = eng.GetGameMods(ctx, tmpDir)
	if len(status.Mods) != 0 {
		t.Errorf("expected 0 mods after deletion, got %d", len(status.Mods))
	}
}
