package manifest

import (
	"sort"
	"strings"
)

// FileChunkDiff represents the chunk-level differences for a single file between manifests.
type FileChunkDiff struct {
	Path            string      `json:"path"`
	OldSize         uint64      `json:"old_size"`
	NewSize         uint64      `json:"new_size"`
	TotalChunks     int         `json:"total_chunks"`
	UnchangedChunks []ChunkInfo `json:"unchanged_chunks"`
	MissingChunks   []ChunkInfo `json:"missing_chunks"`
}

// DepotDiff holds the aggregated differences between two depot manifests.
type DepotDiff struct {
	DepotID       uint32                   `json:"depot_id"`
	ModifiedFiles map[string]FileChunkDiff `json:"modified_files"`
	NewFiles      []ManifestFileEntry      `json:"new_files"`
	DeletedFiles  []string                 `json:"deleted_files"`
}

// isDirEntry checks if a manifest entry represents a directory node.
func isDirEntry(f ManifestFileEntry) bool {
	return (f.Flags&0x40) != 0 || strings.HasSuffix(f.Path, "/") || strings.HasSuffix(f.Path, "\\")
}

// DiffDepotManifests compares an existing local manifest against a newly acquired target manifest.
// It returns a DepotDiff identifying unchanged chunks, missing/modified chunks, new files, and deleted files.
func DiffDepotManifests(oldM, newM *Manifest) *DepotDiff {
	diff := &DepotDiff{
		ModifiedFiles: make(map[string]FileChunkDiff),
	}

	if newM != nil {
		diff.DepotID = newM.DepotID
	} else if oldM != nil {
		diff.DepotID = oldM.DepotID
	}

	if oldM == nil && newM == nil {
		return diff
	}

	if oldM == nil {
		for _, f := range newM.Files {
			if !isDirEntry(f) {
				diff.NewFiles = append(diff.NewFiles, f)
			}
		}
		return diff
	}

	if newM == nil {
		for _, f := range oldM.Files {
			if !isDirEntry(f) {
				diff.DeletedFiles = append(diff.DeletedFiles, f.Path)
			}
		}
		sort.Strings(diff.DeletedFiles)
		return diff
	}

	// Index old files by normalized relative path
	oldMap := make(map[string]ManifestFileEntry, len(oldM.Files))
	for _, f := range oldM.Files {
		if isDirEntry(f) {
			continue
		}
		oldMap[NormalizePath(f.Path)] = f
	}

	seenOld := make(map[string]bool, len(oldMap))

	for _, newEntry := range newM.Files {
		if isDirEntry(newEntry) {
			continue
		}

		normPath := NormalizePath(newEntry.Path)
		oldEntry, exists := oldMap[normPath]
		if !exists {
			diff.NewFiles = append(diff.NewFiles, newEntry)
			continue
		}

		seenOld[normPath] = true

		// Check if file is 100% identical by SHA-1 hash and size
		if newEntry.SHA1 != "" && oldEntry.SHA1 != "" && strings.EqualFold(newEntry.SHA1, oldEntry.SHA1) && newEntry.Size == oldEntry.Size {
			continue
		}

		// If SHA-1 is not present, compare all chunks directly
		if newEntry.SHA1 == "" || oldEntry.SHA1 == "" {
			if newEntry.Size == oldEntry.Size && len(newEntry.Chunks) == len(oldEntry.Chunks) {
				allMatch := true
				for i := range newEntry.Chunks {
					if !strings.EqualFold(newEntry.Chunks[i].ChunkID, oldEntry.Chunks[i].ChunkID) || newEntry.Chunks[i].Offset != oldEntry.Chunks[i].Offset {
						allMatch = false
						break
					}
				}
				if allMatch {
					continue
				}
			}
		}

		// File is modified. Diff chunks by matching (Offset, ChunkID)
		oldChunksByOffset := make(map[uint64]string, len(oldEntry.Chunks))
		for _, ch := range oldEntry.Chunks {
			oldChunksByOffset[ch.Offset] = ch.ChunkID
		}

		fileDiff := FileChunkDiff{
			Path:        newEntry.Path,
			OldSize:     oldEntry.Size,
			NewSize:     newEntry.Size,
			TotalChunks: len(newEntry.Chunks),
		}

		for _, ch := range newEntry.Chunks {
			if oldID, ok := oldChunksByOffset[ch.Offset]; ok && strings.EqualFold(oldID, ch.ChunkID) {
				fileDiff.UnchangedChunks = append(fileDiff.UnchangedChunks, ch)
			} else {
				fileDiff.MissingChunks = append(fileDiff.MissingChunks, ch)
			}
		}

		diff.ModifiedFiles[normPath] = fileDiff
	}

	// Identify deleted files
	for normPath, oldEntry := range oldMap {
		if !seenOld[normPath] {
			diff.DeletedFiles = append(diff.DeletedFiles, oldEntry.Path)
		}
	}
	sort.Strings(diff.DeletedFiles)

	return diff
}
