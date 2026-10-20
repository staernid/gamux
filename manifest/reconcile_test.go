package manifest

import (
	"testing"
)

func TestReconcileDepots_NoConflicts(t *testing.T) {
	candidates := map[uint32][]ManifestFileEntry{
		100: {
			{Path: "bin/game.exe", Size: 1000, SHA1: "aaaa1111", DepotID: 100},
			{Path: "data/common.dat", Size: 2000, SHA1: "bbbb2222", DepotID: 100},
		},
		200: {
			{Path: "data/extra.dat", Size: 500, SHA1: "cccc3333", DepotID: 200},
		},
	}

	filter := DepotFilter{Language: "english", Platform: "windows"}
	files, included, err := ReconcileDepots(candidates, filter)
	if err != nil {
		t.Fatalf("ReconcileDepots failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d", len(files))
	}
	if len(included) != 2 {
		t.Errorf("expected 2 included depots, got %d", len(included))
	}
}

func TestReconcileDepots_DeduplicateIdenticalFiles(t *testing.T) {
	// Two depots contain the exact same shared redistributable file
	candidates := map[uint32][]ManifestFileEntry{
		100: {
			{Path: "bin/game.exe", Size: 1000, SHA1: "aaaa1111", DepotID: 100},
			{Path: "_CommonRedist/vcredist.exe", Size: 5000, SHA1: "dddd4444", DepotID: 100},
		},
		200: {
			{Path: "_CommonRedist/vcredist.exe", Size: 5000, SHA1: "dddd4444", DepotID: 200},
			{Path: "data/extra.dat", Size: 500, SHA1: "cccc3333", DepotID: 200},
		},
	}

	filter := DepotFilter{Language: "english", Platform: "windows"}
	files, included, err := ReconcileDepots(candidates, filter)
	if err != nil {
		t.Fatalf("ReconcileDepots failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("expected 3 deduplicated files, got %d", len(files))
	}
	if len(included) != 2 {
		t.Errorf("expected both depots to be included, got %d", len(included))
	}

	// Verify only one vcredist exists
	vcredistCount := 0
	for _, f := range files {
		if f.Path == "_CommonRedist/vcredist.exe" {
			vcredistCount++
		}
	}
	if vcredistCount != 1 {
		t.Errorf("expected exactly 1 vcredist.exe entry, got %d", vcredistCount)
	}
}

func TestReconcileDepots_ConflictingRegionalSKUs(t *testing.T) {
	// Depot 101: Worldwide SKU (has WW.vdf and English files)
	// Depot 102: Japanese SKU (has JAP.vdf and Japanese files)
	// Both share bin/game.exe and data/main.dat with different SHA-1
	candidates := map[uint32][]ManifestFileEntry{
		101: {
			{Path: "bin/game.exe", Size: 1000, SHA1: "1111ww", DepotID: 101},
			{Path: "data/main.dat", Size: 5000, SHA1: "2222ww", DepotID: 101},
			{Path: "WW.vdf", Size: 50, SHA1: "3333ww", DepotID: 101},
			{Path: "data/english/voice.dat", Size: 2000, SHA1: "4444ww", DepotID: 101},
		},
		102: {
			{Path: "bin/game.exe", Size: 950, SHA1: "1111jp", DepotID: 102},
			{Path: "data/main.dat", Size: 5200, SHA1: "2222jp", DepotID: 102},
			{Path: "JAP.vdf", Size: 50, SHA1: "3333jp", DepotID: 102},
			{Path: "data/japanese/voice.dat", Size: 2100, SHA1: "4444jp", DepotID: 102},
		},
	}

	// Test 1: Request English -> Depot 101 must win, Depot 102 must be excluded
	filterEng := DepotFilter{Language: "english", Platform: "windows"}
	filesEng, includedEng, err := ReconcileDepots(candidates, filterEng)
	if err != nil {
		t.Fatalf("ReconcileDepots english failed: %v", err)
	}

	if len(includedEng) != 1 || includedEng[0] != 101 {
		t.Fatalf("expected depot 101 to win for english, got %v", includedEng)
	}
	if len(filesEng) != 4 {
		t.Errorf("expected 4 files from depot 101, got %d", len(filesEng))
	}
	for _, f := range filesEng {
		if f.DepotID != 101 {
			t.Errorf("expected file from depot 101, got %d (%s)", f.DepotID, f.Path)
		}
	}

	// Test 2: Request Japanese -> Depot 102 must win, Depot 101 must be excluded
	filterJp := DepotFilter{Language: "japanese", Platform: "windows"}
	filesJp, includedJp, err := ReconcileDepots(candidates, filterJp)
	if err != nil {
		t.Fatalf("ReconcileDepots japanese failed: %v", err)
	}

	if len(includedJp) != 1 || includedJp[0] != 102 {
		t.Fatalf("expected depot 102 to win for japanese, got %v", includedJp)
	}
	if len(filesJp) != 4 {
		t.Errorf("expected 4 files from depot 102, got %d", len(filesJp))
	}
	for _, f := range filesJp {
		if f.DepotID != 102 {
			t.Errorf("expected file from depot 102, got %d (%s)", f.DepotID, f.Path)
		}
	}
}

func TestReconcileDepots_DLCCompatibilityTieBreaker(t *testing.T) {
	// Base Depot 101 is compatible with DLC Depot 300
	// Base Depot 102 conflicts with DLC Depot 300
	candidates := map[uint32][]ManifestFileEntry{
		101: {
			{Path: "bin/game.exe", Size: 1000, SHA1: "sha_base1", DepotID: 101},
		},
		102: {
			{Path: "bin/game.exe", Size: 1100, SHA1: "sha_base2", DepotID: 102},
			{Path: "dlc/content.dat", Size: 2000, SHA1: "sha_conflict", DepotID: 102},
		},
		300: {
			{Path: "dlc/content.dat", Size: 2000, SHA1: "sha_dlc", DepotID: 300},
		},
	}

	meta := map[uint32]DepotMeta{
		300: {IsDLC: true, DLCAppID: 300},
	}
	filter := DepotFilter{Language: "english", Platform: "windows", DepotMetadata: meta}

	files, included, err := ReconcileDepots(candidates, filter)
	if err != nil {
		t.Fatalf("ReconcileDepots failed: %v", err)
	}

	// Depot 101 + DLC 300 should be included, 102 excluded
	has101 := false
	has300 := false
	has102 := false
	for _, id := range included {
		if id == 101 {
			has101 = true
		}
		if id == 300 {
			has300 = true
		}
		if id == 102 {
			has102 = true
		}
	}

	if !has101 || !has300 || has102 {
		t.Fatalf("expected depots [101, 300] included and 102 excluded, got %v", included)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 reconciled files, got %d", len(files))
	}
}
