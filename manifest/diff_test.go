package manifest

import (
	"fmt"
	"testing"
)

func TestDiffDepotManifests(t *testing.T) {
	// Create synthetic 10-chunk file
	var oldChunks []ChunkInfo
	var newChunks []ChunkInfo

	for i := 0; i < 10; i++ {
		oldChunk := ChunkInfo{
			ChunkID:          fmt.Sprintf("chunk_%02d_old", i),
			Offset:           uint64(i * 1024),
			CompressedSize:   512,
			UncompressedSize: 1024,
			Checksum:         fmt.Sprintf("chk_%02d", i),
		}
		oldChunks = append(oldChunks, oldChunk)

		newChunk := oldChunk
		// Mutate 2 chunks in new manifest (chunks at index 3 and 7)
		if i == 3 || i == 7 {
			newChunk.ChunkID = fmt.Sprintf("chunk_%02d_mutated", i)
		}
		newChunks = append(newChunks, newChunk)
	}

	oldM := &Manifest{
		AppID:   12345,
		DepotID: 12346,
		Files: []ManifestFileEntry{
			{
				Path:   "game/data.bin",
				Size:   10 * 1024,
				SHA1:   "oldsha1hash",
				Chunks: oldChunks,
			},
			{
				Path: "game/deleted.txt",
				Size: 100,
				SHA1: "deletedsha1",
			},
			{
				Path: "game/identical.cfg",
				Size: 50,
				SHA1: "identicalsha1",
			},
		},
	}

	newM := &Manifest{
		AppID:   12345,
		DepotID: 12346,
		Files: []ManifestFileEntry{
			{
				Path:   "game/data.bin",
				Size:   10 * 1024,
				SHA1:   "newsha1hash",
				Chunks: newChunks,
			},
			{
				Path: "game/identical.cfg",
				Size: 50,
				SHA1: "identicalsha1",
			},
			{
				Path: "game/brand_new.dat",
				Size: 200,
				SHA1: "newdatasha1",
			},
		},
	}

	diff := DiffDepotManifests(oldM, newM)
	if diff == nil {
		t.Fatalf("expected non-nil DepotDiff")
	}

	if diff.DepotID != 12346 {
		t.Errorf("expected DepotID 12346, got %d", diff.DepotID)
	}

	// 1. Verify modified file
	fileDiff, ok := diff.ModifiedFiles["game/data.bin"]
	if !ok {
		t.Fatalf("expected 'game/data.bin' in ModifiedFiles")
	}

	if fileDiff.TotalChunks != 10 {
		t.Errorf("expected TotalChunks 10, got %d", fileDiff.TotalChunks)
	}

	if len(fileDiff.UnchangedChunks) != 8 {
		t.Errorf("expected 8 UnchangedChunks, got %d", len(fileDiff.UnchangedChunks))
	}

	if len(fileDiff.MissingChunks) != 2 {
		t.Errorf("expected 2 MissingChunks, got %d", len(fileDiff.MissingChunks))
	}

	// 2. Verify new file
	if len(diff.NewFiles) != 1 {
		t.Fatalf("expected 1 NewFiles, got %d", len(diff.NewFiles))
	}
	if diff.NewFiles[0].Path != "game/brand_new.dat" {
		t.Errorf("expected 'game/brand_new.dat' in NewFiles, got %s", diff.NewFiles[0].Path)
	}

	// 3. Verify deleted file
	if len(diff.DeletedFiles) != 1 {
		t.Fatalf("expected 1 DeletedFiles, got %d", len(diff.DeletedFiles))
	}
	if diff.DeletedFiles[0] != "game/deleted.txt" {
		t.Errorf("expected 'game/deleted.txt' in DeletedFiles, got %s", diff.DeletedFiles[0])
	}
}

func TestDiffDepotManifests_EdgeCases(t *testing.T) {
	// Nil manifests
	emptyDiff := DiffDepotManifests(nil, nil)
	if len(emptyDiff.ModifiedFiles) != 0 || len(emptyDiff.NewFiles) != 0 || len(emptyDiff.DeletedFiles) != 0 {
		t.Errorf("expected empty diff for nil manifests")
	}

	newM := &Manifest{
		DepotID: 999,
		Files: []ManifestFileEntry{
			{Path: "file1.txt", Size: 10},
		},
	}

	diffOnlyNew := DiffDepotManifests(nil, newM)
	if len(diffOnlyNew.NewFiles) != 1 || diffOnlyNew.DepotID != 999 {
		t.Errorf("expected 1 new file with DepotID 999")
	}

	diffOnlyOld := DiffDepotManifests(newM, nil)
	if len(diffOnlyOld.DeletedFiles) != 1 || diffOnlyOld.DeletedFiles[0] != "file1.txt" {
		t.Errorf("expected 1 deleted file")
	}
}
