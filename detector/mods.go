package detector

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/util"
)

// ModRegistry holds persisted workshop mod metadata in <gameDir>/.gamux/mods.json.
type ModRegistry struct {
	Mods []domain.InstalledMod `json:"mods"`
}

// DetectModTopology identifies the game engine and resolves the appropriate mod layout directory.
func DetectModTopology(gameDir string) domain.ModTopology {
	topo := domain.ModTopology{
		Engine:           "generic",
		ModDir:           "mods",
		AbsoluteModDir:   filepath.Join(gameDir, "mods"),
		Detected:         false,
		SupportedFormats: []string{".zip", ".tar.gz", ".pak", ".dll"},
	}

	if gameDir == "" {
		return topo
	}

	// 1. Check for Unity Engine
	// Look for *_Data directories, UnityPlayer.dll, UnityCrashHandler, or BepInEx
	entries, err := os.ReadDir(gameDir)
	if err == nil {
		isUnity := false
		hasBepInEx := false
		for _, e := range entries {
			nameLower := strings.ToLower(e.Name())
			if e.IsDir() {
				if strings.HasSuffix(nameLower, "_data") {
					isUnity = true
				} else if nameLower == "bepinex" {
					hasBepInEx = true
					isUnity = true
				}
			} else {
				if strings.HasPrefix(nameLower, "unityplayer") || strings.Contains(nameLower, "unitycrashhandler") {
					isUnity = true
				}
			}
		}

		if isUnity || hasBepInEx {
			topo.Engine = "unity"
			topo.ModDir = filepath.Join("BepInEx", "plugins")
			topo.AbsoluteModDir = filepath.Join(gameDir, "BepInEx", "plugins")
			topo.Detected = true
			topo.SupportedFormats = []string{".dll", ".zip"}
			return topo
		}
	}

	// 2. Check for Unreal Engine
	// Look for Engine/ or */Content/Paks
	unrealPaksDir := findUnrealPaksDir(gameDir)
	if unrealPaksDir != "" {
		rel, relErr := filepath.Rel(gameDir, unrealPaksDir)
		if relErr == nil && !strings.HasPrefix(rel, "..") {
			// Unreal Engine standard mod directory is Content/Paks/~mods
			modSubdir := filepath.Join(rel, "~mods")
			absModSubdir := filepath.Join(unrealPaksDir, "~mods")

			topo.Engine = "unreal"
			topo.ModDir = modSubdir
			topo.AbsoluteModDir = absModSubdir
			topo.Detected = true
			topo.SupportedFormats = []string{".pak", ".ucas", ".utoc", ".zip"}
			return topo
		}
	}

	// 3. Check for standard / generic mods directory
	for _, cand := range []string{"mods", "Mods", "MODS"} {
		candPath := filepath.Join(gameDir, cand)
		if fi, sErr := os.Stat(candPath); sErr == nil && fi.IsDir() {
			topo.Engine = "generic"
			topo.ModDir = cand
			topo.AbsoluteModDir = candPath
			topo.Detected = true
			topo.SupportedFormats = []string{".zip", ".tar.gz", ".pak", ".dll"}
			return topo
		}
	}

	return topo
}

// findUnrealPaksDir traverses subdirectories to locate an Unreal Content/Paks folder.
func findUnrealPaksDir(gameDir string) string {
	var foundDir string
	_ = filepath.WalkDir(gameDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		// Don't descend too deep or into steam_settings / .gamux / wine prefixes
		rel, err := filepath.Rel(gameDir, path)
		if err == nil {
			depth := strings.Count(rel, string(os.PathSeparator))
			if depth > 4 {
				return filepath.SkipDir
			}
			base := strings.ToLower(d.Name())
			if base == ".gamux" || base == "steam_settings" || base == "drive_c" || base == "pfx" {
				return filepath.SkipDir
			}
		}

		cleanPath := strings.ReplaceAll(path, "\\", "/")
		if strings.HasSuffix(strings.ToLower(cleanPath), "/content/paks") {
			foundDir = path
			return filepath.SkipAll
		}
		return nil
	})
	return foundDir
}

// LoadModRegistry reads registered workshop mod metadata from <gameDir>/.gamux/mods.json.
func LoadModRegistry(gameDir string) (*ModRegistry, error) {
	regPath := filepath.Join(gameDir, ".gamux", "mods.json")
	data, err := os.ReadFile(regPath)
	if err != nil {
		// Fallback check in steam_settings
		altPath := filepath.Join(gameDir, "steam_settings", "mods.json")
		data, err = os.ReadFile(altPath)
		if err != nil {
			return &ModRegistry{Mods: []domain.InstalledMod{}}, nil
		}
	}

	var reg ModRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return &ModRegistry{Mods: []domain.InstalledMod{}}, err
	}
	return &reg, nil
}

// SaveModRegistry persists registered mod metadata atomically to <gameDir>/.gamux/mods.json.
func SaveModRegistry(gameDir string, reg *ModRegistry) error {
	dir := filepath.Join(gameDir, ".gamux")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	targetFile := filepath.Join(dir, "mods.json")
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	return util.WriteFileAtomic(targetFile, data, 0644)
}

// ScanInstalledMods inspects the resolved mod directory and pairs disk entries with the mod registry.
func ScanInstalledMods(gameDir string, topo domain.ModTopology) ([]domain.InstalledMod, int) {
	registry, _ := LoadModRegistry(gameDir)
	regMap := make(map[string]domain.InstalledMod)
	regByWorkshopID := make(map[uint64]domain.InstalledMod)
	if registry != nil {
		for _, m := range registry.Mods {
			regMap[m.ID] = m
			if m.WorkshopID > 0 {
				regByWorkshopID[m.WorkshopID] = m
			}
		}
	}

	var mods []domain.InstalledMod
	seenPaths := make(map[string]bool)
	outdatedCount := 0

	// Check main mod directory
	scanDir := topo.AbsoluteModDir
	if scanDir != "" {
		entries, err := os.ReadDir(scanDir)
		if err == nil {
			for _, e := range entries {
				name := e.Name()
				if strings.HasPrefix(name, ".") {
					continue
				}

				fullPath := filepath.Join(scanDir, name)
				relPath, _ := filepath.Rel(gameDir, fullPath)
				seenPaths[fullPath] = true

				enabled := true
				cleanName := name
				if strings.HasSuffix(strings.ToLower(name), ".disabled") {
					enabled = false
					cleanName = strings.TrimSuffix(name, ".disabled")
					cleanName = strings.TrimSuffix(cleanName, ".DISABLED")
				}

				size := computePathSize(fullPath)
				modID := cleanName

				// Match with registered workshop item
				matchedMod, isRegistered := regMap[modID]
				if !isRegistered {
					// Also try matching by relative path
					for _, reg := range regMap {
						if reg.RelativePath == relPath || strings.EqualFold(reg.Name, cleanName) {
							matchedMod = reg
							isRegistered = true
							break
						}
					}
				}

				item := domain.InstalledMod{
					ID:           modID,
					Name:         cleanName,
					Enabled:      enabled,
					Path:         fullPath,
					RelativePath: relPath,
					Size:         size,
				}

				if isRegistered {
					item.WorkshopID = matchedMod.WorkshopID
					item.TimeUpdated = matchedMod.TimeUpdated
					item.LatestTimeUpdated = matchedMod.LatestTimeUpdated
					item.HasUpdate = matchedMod.HasUpdate
					item.Description = matchedMod.Description
					item.PreviewURL = matchedMod.PreviewURL
					if matchedMod.Name != "" {
						item.Name = matchedMod.Name
					}
					if item.HasUpdate {
						outdatedCount++
					}
				}

				mods = append(mods, item)
			}
		}
	}

	// Also check if there are registered workshop items in other subfolders (e.g. workshop/<id>)
	if registry != nil {
		for _, reg := range registry.Mods {
			fullPath := reg.Path
			if fullPath == "" && reg.RelativePath != "" {
				fullPath = filepath.Join(gameDir, reg.RelativePath)
			}
			if fullPath != "" && !seenPaths[fullPath] {
				if _, err := os.Stat(fullPath); err == nil {
					seenPaths[fullPath] = true
					enabled := reg.Enabled
					if strings.HasSuffix(strings.ToLower(fullPath), ".disabled") {
						enabled = false
					}
					reg.Path = fullPath
					reg.Enabled = enabled
					if reg.Size <= 0 {
						reg.Size = computePathSize(fullPath)
					}
					if reg.HasUpdate {
						outdatedCount++
					}
					mods = append(mods, reg)
				}
			}
		}
	}

	sort.Slice(mods, func(i, j int) bool {
		return strings.ToLower(mods[i].Name) < strings.ToLower(mods[j].Name)
	})

	return mods, outdatedCount
}

func computePathSize(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	if !fi.IsDir() {
		return fi.Size()
	}

	var total int64
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, wErr error) error {
		if wErr != nil {
			return nil
		}
		if !d.IsDir() {
			if info, iErr := d.Info(); iErr == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}
