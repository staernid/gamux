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

// depotFlagDirectory marks a manifest entry as a directory.
const depotFlagDirectory = 0x40

// ClassifyDepotContent inspects depot metadata, depot names, and manifest file entries
// to classify a depot into "core", "locale:<language>", "soundtrack", or "optional_hd".
//
// Rules are evaluated in a fixed precedence order; the first rule that matches wins:
// executables, metadata, depot name, then file entry analysis.
func ClassifyDepotContent(depotID uint32, depotName string, files []ManifestFileEntry, meta *DepotMeta) (string, string) {
	// Any depot containing game executables is definitively CORE content and must
	// never be classified as an optional locale or soundtrack.
	if containsExecutable(files) {
		return ContentCategoryCore, ""
	}
	if cat, lang, ok := classifyByMeta(meta, len(files)); ok {
		return cat, lang
	}
	if cat, lang, ok := classifyByName(depotName, meta); ok {
		return cat, lang
	}
	if cat, lang, ok := classifyByFiles(files); ok {
		return cat, lang
	}
	return ContentCategoryCore, ""
}

// containsExecutable reports whether any non-directory entry is an executable binary or library.
func containsExecutable(files []ManifestFileEntry) bool {
	for _, f := range files {
		if (f.Flags & depotFlagDirectory) != 0 {
			continue
		}
		if isBinaryOrExecutable(strings.ToLower(NormalizePath(f.Path))) {
			return true
		}
	}
	return false
}

func isSoundtrackName(lowerName string) bool {
	return strings.Contains(lowerName, "soundtrack") || strings.Contains(lowerName, " ost") || strings.HasSuffix(lowerName, "ost")
}

func isOptionalHDName(lowerName string) bool {
	for _, kw := range []string{"high resolution", "4k", "hd texture", "ultra texture", "artbook", "bonus content"} {
		if strings.Contains(lowerName, kw) {
			return true
		}
	}
	return false
}

// classifyByKeywords applies the soundtrack and optional-HD name rules shared by
// metadata names and depot names.
func classifyByKeywords(lowerName string) (string, bool) {
	if isSoundtrackName(lowerName) {
		return ContentCategorySoundtrack, true
	}
	if isOptionalHDName(lowerName) {
		return ContentCategoryOptionalHD, true
	}
	return "", false
}

// classifyByMeta inspects external depot metadata.
// Steamworks often sets config.language: "english" on the base game depot, so a language
// tag only marks an optional language pack when the depot is flagged optional, has language
// pack indicators in its name, or has no files to inspect.
func classifyByMeta(meta *DepotMeta, fileCount int) (cat, lang string, ok bool) {
	if meta == nil {
		return "", "", false
	}
	lowerName := strings.ToLower(meta.Name)
	if cat, ok := classifyByKeywords(lowerName); ok {
		return cat, "", true
	}
	if meta.Language != "" {
		isOptionalLocale := meta.Optional || fileCount == 0
		for _, kw := range []string{"voice", "audio", "speech", "language", "locale", "pack"} {
			if strings.Contains(lowerName, kw) {
				isOptionalLocale = true
			}
		}
		if normLang := NormalizeLanguageCode(meta.Language); normLang != "" && isOptionalLocale {
			return ContentCategoryLocalePrefix + normLang, normLang, true
		}
	}
	if meta.IsDLC || meta.DLCAppID > 0 {
		return ContentCategoryDLC, "", true
	}
	return "", "", false
}

var depotNameLanguageKeywords = []string{
	"french", "german", "italian", "spanish", "japanese", "korean", "koreana",
	"polish", "russian", "chinese", "schinese", "tchinese", "brazilian",
	"portuguese", "arabic", "turkish", "ukrainian", "czech", "dutch",
}

// classifyByName inspects the depot name (metadata name takes precedence when present).
func classifyByName(depotName string, meta *DepotMeta) (cat, lang string, ok bool) {
	name := strings.ToLower(depotName)
	if meta != nil && meta.Name != "" {
		name = strings.ToLower(meta.Name)
	}
	if name == "" {
		return "", "", false
	}
	if cat, ok := classifyByKeywords(name); ok {
		return cat, "", true
	}
	if strings.Contains(name, "dlc #") || strings.Contains(name, "dlc -") || strings.Contains(name, "dlc content") {
		return ContentCategoryDLC, "", true
	}
	for _, kw := range depotNameLanguageKeywords {
		if strings.Contains(name, kw) {
			normLang := NormalizeLanguageCode(kw)
			return ContentCategoryLocalePrefix + normLang, normLang, true
		}
	}
	return "", "", false
}

var pathLanguages = []string{
	"french", "german", "italian", "spanish", "japanese", "koreana",
	"polish", "russian", "schinese", "tchinese", "brazilian", "portuguese",
}

// shortLanguageMarkers maps a language to path fragments identifying its short locale codes.
var shortLanguageMarkers = []struct {
	lang    string
	markers []string
}{
	{"french", []string{"/fr-fr/", "_fr.pck", "_fr.pak"}},
	{"german", []string{"/de-de/", "_de.pck", "_de.pak"}},
	{"japanese", []string{"/ja-jp/", "_ja.pck", "_ja.pak"}},
	{"polish", []string{"/pl-pl/", "_pl.pck", "_pl.pak"}},
	{"russian", []string{"/ru-ru/", "_ru.pck", "_ru.pak"}},
	{"schinese", []string{"/zh-cn/", "_zh.pck", "_zh.pak"}},
}

// fileSignals summarizes manifest entries for content classification.
type fileSignals struct {
	nonDirCount int
	hdCount     int
	langCounts  map[string]int
}

func scanFileSignals(files []ManifestFileEntry) fileSignals {
	sig := fileSignals{langCounts: make(map[string]int)}
	for _, f := range files {
		if (f.Flags & depotFlagDirectory) != 0 {
			continue
		}
		sig.nonDirCount++
		p := strings.ToLower(NormalizePath(f.Path))

		if strings.Contains(p, "4k") || strings.Contains(p, "highres") || strings.Contains(p, "hd_textures") || strings.Contains(p, "ultrahd") {
			sig.hdCount++
		}
		for _, lang := range pathLanguages {
			if strings.Contains(p, "/"+lang+"/") ||
				strings.Contains(p, "_"+lang) ||
				strings.Contains(p, lang+"_") ||
				strings.Contains(p, "/"+lang+".") ||
				strings.HasPrefix(p, lang+"/") {
				sig.langCounts[lang]++
			}
		}
		for _, sm := range shortLanguageMarkers {
			for _, marker := range sm.markers {
				if strings.Contains(p, marker) {
					sig.langCounts[sm.lang]++
					break
				}
			}
		}
	}
	return sig
}

// dominantLanguage returns the language with the highest hit count.
func (s fileSignals) dominantLanguage() (string, int) {
	var maxLang string
	var maxCount int
	for l, c := range s.langCounts {
		if c > maxCount {
			maxCount = c
			maxLang = l
		}
	}
	return maxLang, maxCount
}

// classifyByFiles analyzes manifest file entries.
func classifyByFiles(files []ManifestFileEntry) (cat, lang string, ok bool) {
	if len(files) == 0 {
		return "", "", false
	}
	if ClassifyDepot(files) == DepotCategorySoundtrack {
		return ContentCategorySoundtrack, "", true
	}

	sig := scanFileSignals(files)
	if sig.nonDirCount == 0 {
		return "", "", false
	}
	if float64(sig.hdCount)/float64(sig.nonDirCount) >= 0.5 {
		return ContentCategoryOptionalHD, "", true
	}
	maxLang, maxCount := sig.dominantLanguage()
	if maxCount >= 2 && (float64(maxCount)/float64(sig.nonDirCount) >= 0.3 || maxCount >= 10) {
		normLang := NormalizeLanguageCode(maxLang)
		return ContentCategoryLocalePrefix + normLang, normLang, true
	}
	return "", "", false
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
