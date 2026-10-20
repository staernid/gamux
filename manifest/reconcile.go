package manifest

import (
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
)

// ReconcileDepots evaluates candidate depots matching the platform filter,
// detects mutually exclusive regional/SKU editions via physical file conflicts,
// selects the optimal compatible set based on requested language and DLC affinity,
// and deduplicates identical shared files.
func ReconcileDepots(candidateDepots map[uint32][]ManifestFileEntry, filter DepotFilter) ([]ManifestFileEntry, []uint32, error) {
	if len(candidateDepots) == 0 {
		return nil, nil, nil
	}

	targetLang := NormalizeLanguageCode(filter.Language)
	if targetLang == "" || targetLang == "auto" {
		targetLang = ResolveSystemLanguage()
	}
	if targetLang == "" || targetLang == "all" {
		targetLang = "english"
	}

	// 1. Index non-directory files by depot
	type depotData struct {
		id      uint32
		files   []ManifestFileEntry
		fileMap map[string]ManifestFileEntry
	}

	depots := make(map[uint32]*depotData)
	var depotIDs []uint32

	for did, files := range candidateDepots {
		fMap := make(map[string]ManifestFileEntry, len(files))
		for _, f := range files {
			if (f.Flags&depotFlagDirectory) == 0 && f.Size > 0 {
				p := NormalizePath(f.Path)
				fMap[p] = f
			}
		}
		depots[did] = &depotData{
			id:      did,
			files:   files,
			fileMap: fMap,
		}
		depotIDs = append(depotIDs, did)
	}

	sort.Slice(depotIDs, func(i, j int) bool {
		return depotIDs[i] < depotIDs[j]
	})

	// 2. Build conflict matrix between depots
	// Two depots conflict if they contain the same relative path with differing hashes or sizes
	hasConflict := func(d1, d2 uint32) bool {
		m1 := depots[d1].fileMap
		m2 := depots[d2].fileMap
		for p, f1 := range m1 {
			if f2, ok := m2[p]; ok {
				if f1.SHA1 != f2.SHA1 || f1.Size != f2.Size {
					return true
				}
			}
		}
		return false
	}

	// Active candidate depots
	active := make(map[uint32]bool, len(depotIDs))
	for _, did := range depotIDs {
		active[did] = true
	}

	// 3. Resolve conflicts iteratively
	for {
		// Find any pair of active depots that conflict
		var conflictPairs [][2]uint32
		for i := 0; i < len(depotIDs); i++ {
			d1 := depotIDs[i]
			if !active[d1] {
				continue
			}
			for j := i + 1; j < len(depotIDs); j++ {
				d2 := depotIDs[j]
				if !active[d2] {
					continue
				}
				if hasConflict(d1, d2) {
					conflictPairs = append(conflictPairs, [2]uint32{d1, d2})
				}
			}
		}

		if len(conflictPairs) == 0 {
			break
		}

		// Pick the first conflicting pair and resolve it using language and affinity scoring
		pair := conflictPairs[0]
		d1, d2 := pair[0], pair[1]

		score1 := scoreDepotAffinity(d1, depots[d1].files, targetLang, filter.DepotMetadata)
		score2 := scoreDepotAffinity(d2, depots[d2].files, targetLang, filter.DepotMetadata)

		var winner, loser uint32
		if score1 > score2 {
			winner, loser = d1, d2
		} else if score2 > score1 {
			winner, loser = d2, d1
		} else {
			// Tie-breaker 1: Check which depot has fewer conflicts with other active depots
			c1 := countConflicts(d1, depotIDs, active, hasConflict)
			c2 := countConflicts(d2, depotIDs, active, hasConflict)
			if c1 < c2 {
				winner, loser = d1, d2
			} else if c2 < c1 {
				winner, loser = d2, d1
			} else {
				// Tie-breaker 2: Deterministic depot ID (lower depot ID)
				if d1 < d2 {
					winner, loser = d1, d2
				} else {
					winner, loser = d2, d1
				}
			}
		}

		slog.Info("Resolved mutually exclusive depot SKU conflict",
			"selectedDepot", winner,
			"excludedDepot", loser,
			"language", targetLang,
			"winnerScore", score1,
			"loserScore", score2,
		)

		delete(active, loser)
	}

	// 4. Merge remaining non-conflicting depots and deduplicate shared identical files
	var includedDepots []uint32
	for _, did := range depotIDs {
		if active[did] {
			includedDepots = append(includedDepots, did)
		}
	}

	var finalFiles []ManifestFileEntry
	seenPaths := make(map[string]ManifestFileEntry)

	for _, did := range includedDepots {
		for _, f := range depots[did].files {
			p := NormalizePath(f.Path)
			if p == "" && (f.Flags&depotFlagDirectory) != 0 {
				continue
			}

			if (f.Flags & depotFlagDirectory) != 0 {
				// For directories, preserve if not yet seen
				if _, ok := seenPaths[p]; !ok {
					seenPaths[p] = f
					finalFiles = append(finalFiles, f)
				}
				continue
			}

			if existing, ok := seenPaths[p]; ok {
				if existing.SHA1 == f.SHA1 {
					// Identical shared file in multiple depots (e.g. shared redistributables)
					continue
				}
				// Conflict safety net (should not happen after reconciliation)
				slog.Warn("Deduplicating residual file collision, preserving primary",
					"path", p,
					"keptDepot", existing.DepotID,
					"droppedDepot", f.DepotID,
				)
				continue
			}

			seenPaths[p] = f
			finalFiles = append(finalFiles, f)
		}
	}

	return finalFiles, includedDepots, nil
}

func countConflicts(target uint32, allIDs []uint32, active map[uint32]bool, hasConflict func(d1, d2 uint32) bool) int {
	count := 0
	for _, other := range allIDs {
		if other != target && active[other] && hasConflict(target, other) {
			count++
		}
	}
	return count
}

// scoreDepotAffinity evaluates how well a depot matches the requested target language
// and installation environment.
func scoreDepotAffinity(dID uint32, files []ManifestFileEntry, targetLang string, metaMap map[uint32]DepotMeta) int {
	score := 0
	targetIsEnglish := targetLang == "english"

	var meta *DepotMeta
	if metaMap != nil {
		if m, ok := metaMap[dID]; ok {
			meta = &m
		}
	}

	// 1. External metadata evaluation
	if meta != nil {
		if meta.Language != "" {
			if LanguagesMatch(meta.Language, targetLang) {
				score += 60
			} else {
				score -= 60
			}
		}
		if meta.IsDLC || meta.DLCAppID > 0 {
			score += 20
		}
	}

	// 2. Scan depot files for install scripts and regional language markers
	for _, f := range files {
		lp := strings.ToLower(f.Path)
		base := strings.ToLower(filepath.Base(lp))

		// Regional installscript check (Ubisoft, EA, Valve patterns)
		switch base {
		case "ww.vdf", "row.vdf":
			if targetIsEnglish {
				score += 80
			}
		case "jap.vdf", "jp.vdf":
			if LanguagesMatch("japanese", targetLang) {
				score += 80
			} else {
				score -= 80
			}
		case "rus.vdf", "ru.vdf":
			if LanguagesMatch("russian", targetLang) {
				score += 80
			} else {
				score -= 80
			}
		case "de.vdf", "ger.vdf":
			if LanguagesMatch("german", targetLang) {
				score += 80
			} else {
				score -= 60
			}
		}

		// Language path fragments
		if strings.Contains(lp, "/japanese/") || strings.Contains(lp, "_japanese") || strings.Contains(lp, "japanese_") || strings.Contains(lp, "/ja-jp/") {
			if LanguagesMatch("japanese", targetLang) {
				score += 2
			} else {
				score -= 2
			}
		}
		if strings.Contains(lp, "/russian/") || strings.Contains(lp, "_russian") || strings.Contains(lp, "russian_") || strings.Contains(lp, "/ru-ru/") {
			if LanguagesMatch("russian", targetLang) {
				score += 2
			} else {
				score -= 2
			}
		}
		if strings.Contains(lp, "/english/") || strings.Contains(lp, "_english") || strings.Contains(lp, "english_") || strings.Contains(lp, "/en-us/") {
			if targetIsEnglish {
				score += 2
			}
		}
	}

	return score
}
