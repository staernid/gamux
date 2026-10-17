package manifest

import (
	"path/filepath"
	"strings"
)

// isBinaryOrExecutable returns true if the relative file path represents an executable binary or shared library.
func isBinaryOrExecutable(normPath string) bool {
	p := strings.ToLower(normPath)
	ext := filepath.Ext(p)
	if ext == ".exe" || ext == ".dll" || ext == ".so" || ext == ".dylib" || ext == ".app" ||
		ext == ".x86_64" || ext == ".x86" || ext == ".bin.x86" || ext == ".bin.x86_64" || ext == ".elf" {
		return true
	}
	if strings.Contains(p, ".so.") {
		return true
	}
	if strings.HasPrefix(p, "bin/") || strings.Contains(p, "/bin/") ||
		strings.HasPrefix(p, "bin32/") || strings.HasPrefix(p, "bin64/") ||
		strings.Contains(p, "/bin32/") || strings.Contains(p, "/bin64/") ||
		strings.HasPrefix(p, "lib/") || strings.Contains(p, "/lib/") ||
		strings.HasPrefix(p, "lib64/") || strings.Contains(p, "/lib64/") {
		nonBinaryExts := map[string]bool{
			".txt": true, ".json": true, ".xml": true, ".cfg": true, ".ini": true,
			".png": true, ".jpg": true, ".jpeg": true, ".log": true, ".dat": true,
			".pak": true, ".pdb": true, ".html": true, ".htm": true, ".css": true,
			".js": true, ".md": true, ".csv": true, ".yml": true, ".yaml": true,
			".lua": true, ".manifest": true,
		}
		if !nonBinaryExts[ext] {
			return true
		}
	}
	return false
}

// ClassifyDepotContent inspects depot metadata, depot names, and manifest file entries
// to classify a depot into "core", "locale:<language>", "soundtrack", or "optional_hd".
func ClassifyDepotContent(depotID uint32, depotName string, files []ManifestFileEntry, meta *DepotMeta) (string, string) {
	// 0. Executable check: Any depot containing game executables (.exe, .so, .dylib, etc.)
	// is definitively CORE content and must never be classified as an optional locale or soundtrack.
	for _, f := range files {
		if (f.Flags & 0x40) != 0 {
			continue // directory entry
		}
		p := strings.ToLower(NormalizePath(f.Path))
		if isBinaryOrExecutable(p) {
			return ContentCategoryCore, ""
		}
	}

	// 1. External depot metadata inspection
	// Note: Steamworks often sets config.language: "english" on the base game depot.
	// Only treat it as an optional language pack if it is marked optional or has language pack indicators in its name,
	// or if files are not available and it has an explicit language tag.
	if meta != nil {
		lowerMetaName := strings.ToLower(meta.Name)
		if strings.Contains(lowerMetaName, "soundtrack") || strings.Contains(lowerMetaName, " ost") || strings.HasSuffix(lowerMetaName, "ost") {
			return ContentCategorySoundtrack, ""
		}
		if strings.Contains(lowerMetaName, "high resolution") || strings.Contains(lowerMetaName, "4k") ||
			strings.Contains(lowerMetaName, "hd texture") || strings.Contains(lowerMetaName, "ultra texture") ||
			strings.Contains(lowerMetaName, "artbook") || strings.Contains(lowerMetaName, "bonus content") {
			return ContentCategoryOptionalHD, ""
		}
		if meta.Language != "" {
			isOptionalLocale := meta.Optional ||
				strings.Contains(lowerMetaName, "voice") ||
				strings.Contains(lowerMetaName, "audio") ||
				strings.Contains(lowerMetaName, "speech") ||
				strings.Contains(lowerMetaName, "language") ||
				strings.Contains(lowerMetaName, "locale") ||
				strings.Contains(lowerMetaName, "pack") ||
				len(files) == 0

			normLang := NormalizeLanguageCode(meta.Language)
			if normLang != "" && isOptionalLocale {
				return ContentCategoryLocalePrefix + normLang, normLang
			}
		}
		if meta.IsDLC || meta.DLCAppID > 0 {
			return ContentCategoryDLC, ""
		}
	}

	// 2. Depot name string inspection
	nameToCheck := strings.ToLower(depotName)
	if meta != nil && meta.Name != "" {
		nameToCheck = strings.ToLower(meta.Name)
	}

	if nameToCheck != "" {
		if strings.Contains(nameToCheck, "soundtrack") || strings.Contains(nameToCheck, " ost") || strings.HasSuffix(nameToCheck, "ost") {
			return ContentCategorySoundtrack, ""
		}
		if strings.Contains(nameToCheck, "high resolution") || strings.Contains(nameToCheck, "4k") ||
			strings.Contains(nameToCheck, "hd texture") || strings.Contains(nameToCheck, "ultra texture") ||
			strings.Contains(nameToCheck, "artbook") || strings.Contains(nameToCheck, "bonus content") {
			return ContentCategoryOptionalHD, ""
		}
		if strings.Contains(nameToCheck, "dlc #") || strings.Contains(nameToCheck, "dlc -") || strings.Contains(nameToCheck, "dlc content") {
			return ContentCategoryDLC, ""
		}

		// Check for language keywords in depot name
		langKeywords := []string{
			"french", "german", "italian", "spanish", "japanese", "korean", "koreana",
			"polish", "russian", "chinese", "schinese", "tchinese", "brazilian",
			"portuguese", "arabic", "turkish", "ukrainian", "czech", "dutch",
		}
		for _, kw := range langKeywords {
			if strings.Contains(nameToCheck, kw) {
				normLang := NormalizeLanguageCode(kw)
				return ContentCategoryLocalePrefix + normLang, normLang
			}
		}
	}

	// 3. Manifest file entries analysis
	if len(files) > 0 {
		platCat := ClassifyDepot(files)
		if platCat == DepotCategorySoundtrack {
			return ContentCategorySoundtrack, ""
		}

		var hasExecutable bool
		hdFileCount := 0
		nonDirCount := 0
		langCounts := make(map[string]int)

		languages := []string{
			"french", "german", "italian", "spanish", "japanese", "koreana",
			"polish", "russian", "schinese", "tchinese", "brazilian", "portuguese",
		}

		for _, f := range files {
			if (f.Flags & 0x40) != 0 {
				continue
			}
			nonDirCount++
			p := strings.ToLower(NormalizePath(f.Path))

			if strings.HasSuffix(p, ".exe") || strings.HasSuffix(p, ".so") || strings.HasSuffix(p, ".dylib") ||
				strings.HasSuffix(p, ".bin.x86_64") || strings.HasSuffix(p, ".x86_64") {
				hasExecutable = true
			}

			if strings.Contains(p, "4k") || strings.Contains(p, "highres") || strings.Contains(p, "hd_textures") || strings.Contains(p, "ultrahd") {
				hdFileCount++
			}

			// Language path checks
			for _, lang := range languages {
				if strings.Contains(p, "/"+lang+"/") ||
					strings.Contains(p, "_"+lang) ||
					strings.Contains(p, lang+"_") ||
					strings.Contains(p, "/"+lang+".") ||
					strings.HasPrefix(p, lang+"/") {
					langCounts[lang]++
				}
			}

			// Short code checks
			if strings.Contains(p, "/fr-fr/") || strings.Contains(p, "_fr.pck") || strings.Contains(p, "_fr.pak") {
				langCounts["french"]++
			}
			if strings.Contains(p, "/de-de/") || strings.Contains(p, "_de.pck") || strings.Contains(p, "_de.pak") {
				langCounts["german"]++
			}
			if strings.Contains(p, "/ja-jp/") || strings.Contains(p, "_ja.pck") || strings.Contains(p, "_ja.pak") {
				langCounts["japanese"]++
			}
			if strings.Contains(p, "/pl-pl/") || strings.Contains(p, "_pl.pck") || strings.Contains(p, "_pl.pak") {
				langCounts["polish"]++
			}
			if strings.Contains(p, "/ru-ru/") || strings.Contains(p, "_ru.pck") || strings.Contains(p, "_ru.pak") {
				langCounts["russian"]++
			}
			if strings.Contains(p, "/zh-cn/") || strings.Contains(p, "_zh.pck") || strings.Contains(p, "_zh.pak") {
				langCounts["schinese"]++
			}
		}

		if nonDirCount > 0 && !hasExecutable {
			if float64(hdFileCount)/float64(nonDirCount) >= 0.5 {
				return ContentCategoryOptionalHD, ""
			}

			// Check if any language dominates the files
			var maxLang string
			var maxCount int
			for l, c := range langCounts {
				if c > maxCount {
					maxCount = c
					maxLang = l
				}
			}

			if maxCount >= 2 && (float64(maxCount)/float64(nonDirCount) >= 0.3 || maxCount >= 10) {
				normLang := NormalizeLanguageCode(maxLang)
				return ContentCategoryLocalePrefix + normLang, normLang
			}
		}
	}

	return ContentCategoryCore, ""
}

// ClassifyDepot inspects manifest file entries to determine the target OS or payload category.
func ClassifyDepot(files []ManifestFileEntry) DepotCategory {
	if len(files) == 0 {
		return DepotCategoryCommon
	}

	var hasWin, hasMac, hasLinux, hasExecutables, hasMediaOnly bool
	mediaExts := map[string]bool{
		".mp3": true, ".flac": true, ".wav": true, ".ogg": true, ".m4a": true,
		".aac": true, ".opus": true, ".cue": true, ".log": true, ".jpg": true,
		".jpeg": true, ".png": true, ".pdf": true,
	}

	mediaCount := 0
	nonDirCount := 0

	for _, f := range files {
		if (f.Flags & 0x40) != 0 {
			continue // directory
		}
		nonDirCount++
		p := strings.ToLower(NormalizePath(f.Path))

		// 1. macOS checks
		if strings.Contains(p, ".app/") || strings.Contains(p, "contents/macos/") || strings.HasPrefix(p, "macos/") || strings.HasPrefix(p, "osx/") || strings.HasSuffix(p, ".dylib") {
			hasMac = true
		}

		// 2. Linux checks
		if strings.HasSuffix(p, ".x86_64") || strings.HasSuffix(p, ".x86") || strings.HasSuffix(p, ".bin.x86") || strings.HasSuffix(p, ".bin.x86_64") || strings.Contains(p, "linux_client") || strings.HasPrefix(p, "linux/") ||
			strings.Contains(p, "bin/x64/") || strings.Contains(p, "bin/i386/") || strings.Contains(p, "bin/x86/") || strings.Contains(p, "bin/linux/") || strings.HasPrefix(p, "linux64/") {
			if !strings.HasSuffix(p, ".exe") && !strings.HasSuffix(p, ".dll") && !strings.HasSuffix(p, ".pdb") {
				hasLinux = true
			}
		}

		// 3. Windows checks
		if strings.HasSuffix(p, ".exe") || strings.HasSuffix(p, ".dll") || strings.HasSuffix(p, ".bat") || strings.HasSuffix(p, ".cmd") || strings.HasPrefix(p, "win32/") || strings.HasPrefix(p, "win64/") || strings.HasPrefix(p, "windows/") || strings.Contains(p, "_commonredist") || strings.Contains(p, "bin/win32/") || strings.Contains(p, "bin/win64/") {
			hasWin = true
		}

		// 4. Executable / binary check
		if isBinaryOrExecutable(p) {
			hasExecutables = true
		}

		ext := filepath.Ext(p)
		if mediaExts[ext] {
			mediaCount++
		}
	}

	if nonDirCount > 0 && mediaCount == nonDirCount && !hasExecutables {
		hasMediaOnly = true
	}

	if hasMediaOnly {
		return DepotCategorySoundtrack
	}
	if hasWin && !hasMac && !hasLinux {
		return DepotCategoryWindows
	}
	if hasLinux && !hasWin && !hasMac {
		return DepotCategoryLinux
	}
	if hasMac && !hasWin && !hasLinux {
		return DepotCategoryMacOS
	}

	return DepotCategoryCommon
}

// MatchesPlatform tests whether a depot category matches the requested platform target.
func MatchesPlatform(cat DepotCategory, targetPlat string) bool {
	target := strings.ToLower(strings.TrimSpace(targetPlat))
	if target == "" || target == "all" {
		return true
	}

	switch cat {
	case DepotCategoryCommon:
		return true
	case DepotCategoryWindows:
		return target == "win64" || target == "win32" || target == "windows" || target == "win"
	case DepotCategoryLinux:
		return target == "linux" || target == "linuxnative"
	case DepotCategoryMacOS:
		return target == "macos" || target == "osx" || target == "darwin" || target == "mac"
	case DepotCategorySoundtrack:
		return target == "soundtrack" || target == "music"
	default:
		return true
	}
}

// MatchesOSList tests whether an official Steam depot oslist matches the requested platform target.
// oslist may be empty (common to all platforms), comma/space separated, or a single OS name.
func MatchesOSList(osList, targetPlat string) bool {
	cleaned := strings.ToLower(strings.TrimSpace(osList))
	if cleaned == "" || cleaned == "all" {
		return true
	}
	target := strings.ToLower(strings.TrimSpace(targetPlat))
	if target == "" || target == "all" {
		return true
	}

	normTarget := ""
	switch target {
	case "win64", "win32", "windows", "win":
		normTarget = "windows"
	case "linux", "linuxnative":
		normTarget = "linux"
	case "macos", "osx", "darwin", "mac":
		normTarget = "macos"
	default:
		normTarget = target
	}

	parts := strings.FieldsFunc(cleaned, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	})
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "all" || p == normTarget {
			return true
		}
		if (normTarget == "macos" && (p == "osx" || p == "darwin" || p == "mac")) ||
			(normTarget == "windows" && (p == "win" || p == "win32" || p == "win64")) {
			return true
		}
	}
	return false
}
