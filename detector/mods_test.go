package detector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/staernid/gamux/domain"
)

func TestDetectModTopology(t *testing.T) {
	t.Run("Unity with BepInEx", func(t *testing.T) {
		tmp := t.TempDir()
		_ = os.MkdirAll(filepath.Join(tmp, "Game_Data"), 0755)
		_ = os.WriteFile(filepath.Join(tmp, "UnityPlayer.dll"), []byte("mock"), 0644)

		topo := DetectModTopology(tmp)
		if !topo.Detected {
			t.Errorf("expected detected=true for Unity")
		}
		if topo.Engine != "unity" {
			t.Errorf("expected engine=unity, got %s", topo.Engine)
		}
		if topo.ModDir != filepath.Join("BepInEx", "plugins") {
			t.Errorf("expected mod dir BepInEx/plugins, got %s", topo.ModDir)
		}
	})

	t.Run("Unreal Engine with Content/Paks", func(t *testing.T) {
		tmp := t.TempDir()
		paksDir := filepath.Join(tmp, "MyGame", "Content", "Paks")
		_ = os.MkdirAll(paksDir, 0755)

		topo := DetectModTopology(tmp)
		if !topo.Detected {
			t.Errorf("expected detected=true for Unreal")
		}
		if topo.Engine != "unreal" {
			t.Errorf("expected engine=unreal, got %s", topo.Engine)
		}
		expectedRel := filepath.Join("MyGame", "Content", "Paks", "~mods")
		if topo.ModDir != expectedRel {
			t.Errorf("expected mod dir %s, got %s", expectedRel, topo.ModDir)
		}
	})

	t.Run("Generic mods directory", func(t *testing.T) {
		tmp := t.TempDir()
		_ = os.MkdirAll(filepath.Join(tmp, "mods"), 0755)

		topo := DetectModTopology(tmp)
		if !topo.Detected {
			t.Errorf("expected detected=true for generic mods")
		}
		if topo.Engine != "generic" {
			t.Errorf("expected engine=generic, got %s", topo.Engine)
		}
		if topo.ModDir != "mods" {
			t.Errorf("expected mod dir mods, got %s", topo.ModDir)
		}
	})
}

func TestScanInstalledMods(t *testing.T) {
	tmp := t.TempDir()
	modsDir := filepath.Join(tmp, "mods")
	_ = os.MkdirAll(modsDir, 0755)

	// Enabled mod file
	_ = os.WriteFile(filepath.Join(modsDir, "SuperMod.pak"), []byte("active"), 0644)
	// Disabled mod file
	_ = os.WriteFile(filepath.Join(modsDir, "OldMod.pak.disabled"), []byte("inactive"), 0644)

	// Save registry with update available
	reg := &ModRegistry{
		Mods: []domain.InstalledMod{
			{
				ID:                "SuperMod.pak",
				Name:              "Super Texture Pack",
				WorkshopID:        123456,
				Enabled:           true,
				RelativePath:      filepath.Join("mods", "SuperMod.pak"),
				TimeUpdated:       1000,
				LatestTimeUpdated: 2000,
				HasUpdate:         true,
			},
		},
	}
	if err := SaveModRegistry(tmp, reg); err != nil {
		t.Fatalf("failed to save mod registry: %v", err)
	}

	topo := domain.ModTopology{
		Engine:         "generic",
		ModDir:         "mods",
		AbsoluteModDir: modsDir,
		Detected:       true,
	}

	mods, outdatedCount := ScanInstalledMods(tmp, topo)
	if len(mods) != 2 {
		t.Fatalf("expected 2 mods, got %d", len(mods))
	}
	if outdatedCount != 1 {
		t.Errorf("expected 1 outdated mod, got %d", outdatedCount)
	}

	var superMod, oldMod *domain.InstalledMod
	for i := range mods {
		if mods[i].ID == "SuperMod.pak" {
			superMod = &mods[i]
		} else if mods[i].ID == "OldMod.pak" {
			oldMod = &mods[i]
		}
	}

	if superMod == nil || !superMod.Enabled || !superMod.HasUpdate || superMod.WorkshopID != 123456 {
		t.Errorf("superMod not properly loaded: %+v", superMod)
	}
	if oldMod == nil || oldMod.Enabled {
		t.Errorf("oldMod should be disabled: %+v", oldMod)
	}
}
