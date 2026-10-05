package manifest

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/util"
)


// ChunkInfo represents a single file chunk in a Steam manifest.
type ChunkInfo struct {
	ChunkID      string `json:"chunk_id"`
	Offset       uint64 `json:"offset"`
	CompressedSize uint32 `json:"compressed_size"`
	UncompressedSize uint32 `json:"uncompressed_size"`
	Checksum     string `json:"checksum"`
}

// ManifestFileEntry represents a file entry in a Steam manifest.
type ManifestFileEntry struct {
	Path        string      `json:"path"`
	Size        uint64      `json:"size"`
	Flags       uint32      `json:"flags"`
	SHA1        string      `json:"sha1"`
	DepotID     uint32      `json:"depot_id,omitempty"`
	Chunks      []ChunkInfo `json:"chunks"`
}

// Manifest represents parsed manifest file details for a Steam depot.
type Manifest struct {
	AppID         uint32              `json:"app_id"`
	DepotID       uint32              `json:"depot_id"`
	ManifestID    uint64              `json:"manifest_id"`
	CreationTime  uint32              `json:"creation_time"`
	DecryptionKey string              `json:"decryption_key,omitempty"`
	DepotKeys     map[uint32]string   `json:"depot_keys,omitempty"`
	Files         []ManifestFileEntry `json:"files"`
}

// GetDecryptionKey returns the decryption key for a specific depot ID or falls back to the default key.
func (m *Manifest) GetDecryptionKey(depotID uint32) string {
	if m.DepotKeys != nil {
		if key, ok := m.DepotKeys[depotID]; ok && key != "" {
			return key
		}
	}
	return m.DecryptionKey
}

// DepotCategory classifies a Steam content depot's platform or payload type.
type DepotCategory string

const (
	DepotCategoryCommon     DepotCategory = "common"
	DepotCategoryWindows    DepotCategory = "windows"
	DepotCategoryLinux      DepotCategory = "linux"
	DepotCategoryMacOS      DepotCategory = "macos"
	DepotCategorySoundtrack DepotCategory = "soundtrack"
)

// Content category identifiers for multi-depot and optional content filtering.
const (
	ContentCategoryCore         = "core"
	ContentCategorySoundtrack   = "soundtrack"
	ContentCategoryOptionalHD   = "optional_hd"
	ContentCategoryDLC          = "dlc"
	ContentCategoryLocalePrefix = "locale:"
)

// DepotMeta captures metadata about a Steam depot for classification and filtering.
type DepotMeta struct {
	DepotID      uint32 `json:"depot_id"`
	Name         string `json:"name,omitempty"`
	Language     string `json:"language,omitempty"`
	OSList       string `json:"oslist,omitempty"`
	OSArch       string `json:"osarch,omitempty"`
	Optional     bool   `json:"optional,omitempty"`
	IsDLC        bool   `json:"is_dlc,omitempty"`
	DLCAppID     uint32 `json:"dlc_app_id,omitempty"`
	DLCName      string `json:"dlc_name,omitempty"`
	Size         uint64 `json:"size,omitempty"`
	DownloadSize uint64 `json:"download_size,omitempty"`
}

var languageAliases = map[string]string{
	"english":                 "english",
	"en":                      "english",
	"french":                  "french",
	"fr":                      "french",
	"fr-fr":                   "french",
	"german":                  "german",
	"de":                      "german",
	"de-de":                   "german",
	"italian":                 "italian",
	"it":                      "italian",
	"it-it":                   "italian",
	"spanish":                 "spanish",
	"es":                      "spanish",
	"es-es":                   "spanish",
	"spanish - spain":         "spanish",
	"spanish - latin america": "latam",
	"latam":                   "latam",
	"japanese":                "japanese",
	"ja":                      "japanese",
	"jp":                      "japanese",
	"ja-jp":                   "japanese",
	"korean":                  "koreana",
	"koreana":                 "koreana",
	"ko":                      "koreana",
	"kr":                      "koreana",
	"ko-kr":                   "koreana",
	"polish":                  "polish",
	"pl":                      "polish",
	"pl-pl":                   "polish",
	"russian":                 "russian",
	"ru":                      "russian",
	"ru-ru":                   "russian",
	"simplified chinese":      "schinese",
	"schinese":                "schinese",
	"zh-cn":                   "schinese",
	"zh-hans":                 "schinese",
	"traditional chinese":     "tchinese",
	"tchinese":                "tchinese",
	"zh-tw":                   "tchinese",
	"zh-hant":                 "tchinese",
	"chinese":                 "schinese",
	"zh":                      "schinese",
	"portuguese":              "portuguese",
	"pt":                      "portuguese",
	"pt-pt":                   "portuguese",
	"portuguese - brazil":     "brazilian",
	"brazilian":               "brazilian",
	"pt-br":                   "brazilian",
	"arabic":                  "arabic",
	"ar":                      "arabic",
	"czech":                   "czech",
	"cs":                      "czech",
	"danish":                  "danish",
	"da":                      "danish",
	"dutch":                   "dutch",
	"nl":                      "dutch",
	"finnish":                 "finnish",
	"fi":                      "finnish",
	"greek":                   "greek",
	"el":                      "greek",
	"hungarian":               "hungarian",
	"hu":                      "hungarian",
	"norwegian":               "norwegian",
	"no":                      "norwegian",
	"swedish":                 "swedish",
	"sv":                      "swedish",
	"thai":                    "thai",
	"th":                      "thai",
	"turkish":                 "turkish",
	"tr":                      "turkish",
	"ukrainian":               "ukrainian",
	"uk":                      "ukrainian",
	"ua":                      "ukrainian",
	"vietnamese":              "vietnamese",
	"vi":                      "vietnamese",
}

// ResolveSystemLanguage inspects environment variables (LC_ALL, LC_MESSAGES, LANG)
// to determine the system's preferred language, defaulting to "english".
func ResolveSystemLanguage() string {
	for _, env := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		val := strings.TrimSpace(os.Getenv(env))
		if val != "" && val != "C" && val != "POSIX" {
			if idx := strings.Index(val, "."); idx != -1 {
				val = val[:idx]
			}
			val = strings.ReplaceAll(val, "_", "-")
			norm := NormalizeLanguageCode(val)
			if norm != "" && norm != "auto" && norm != "all" {
				return norm
			}
		}
	}
	return "english"
}

// NormalizeLanguageCode normalizes common language names and ISO codes to canonical Steam language names.
func NormalizeLanguageCode(lang string) string {
	clean := strings.ToLower(strings.TrimSpace(lang))
	if clean == "" || clean == "auto" || clean == "all" {
		return clean
	}
	if canonical, ok := languageAliases[clean]; ok {
		return canonical
	}
	for alias, canonical := range languageAliases {
		if strings.EqualFold(alias, clean) || strings.HasPrefix(clean, alias) {
			return canonical
		}
	}
	return clean
}

// LanguagesMatch returns true if two language descriptors refer to the same language.
func LanguagesMatch(lang1, lang2 string) bool {
	l1 := NormalizeLanguageCode(lang1)
	l2 := NormalizeLanguageCode(lang2)
	if l1 == "" || l2 == "" || l1 == "all" || l2 == "all" {
		return true
	}
	if l1 == "auto" {
		l1 = ResolveSystemLanguage()
	}
	if l2 == "auto" {
		l2 = ResolveSystemLanguage()
	}
	if l1 == "" || l2 == "" || l1 == "all" || l2 == "all" {
		return true
	}
	if l1 == l2 {
		return true
	}
	// Broad compatibility for Chinese variants and Portuguese variants
	if (l1 == "schinese" || l1 == "tchinese") && (l2 == "schinese" || l2 == "tchinese") {
		return true
	}
	if (l1 == "portuguese" || l1 == "brazilian") && (l2 == "portuguese" || l2 == "brazilian") {
		return true
	}
	return false
}

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

// NormalizePath cleans and normalizes relative file paths for Linux/Windows cross-platform safety.
func NormalizePath(p string) string {
	cleaned := filepath.Clean(strings.ReplaceAll(p, "\\", "/"))
	return strings.TrimPrefix(cleaned, "/")
}


// TotalSize returns the aggregate uncompressed size of all files in the manifest.
func (m *Manifest) TotalSize() uint64 {
	var total uint64
	for _, f := range m.Files {
		total += f.Size
	}
	return total
}

// FindFile returns the file entry for a given relative path.
func (m *Manifest) FindFile(relativePath string) (*ManifestFileEntry, error) {
	norm := NormalizePath(relativePath)
	for i := range m.Files {
		if NormalizePath(m.Files[i].Path) == norm {
			return &m.Files[i], nil
		}
	}
	return nil, fmt.Errorf("file %q not found in manifest", relativePath)
}


// ScanUntrackedFiles walks gameDir and compares every file on disk against official manifest entries.
// It returns official file count and a slice of relative paths for untracked/mod files.
// IntegrityReport details file matching, modifications, missing files, and untracked mods.
type IntegrityReport struct {
	OfficialCount  int      `json:"official_count"`
	IntactCount    int      `json:"intact_count"`
	ModifiedFiles  []string `json:"modified_files,omitempty"`
	MissingFiles   []string `json:"missing_files,omitempty"`
	UntrackedFiles []string `json:"untracked_files,omitempty"`
	RedistModified []string `json:"redist_modified,omitempty"`
	RedistMissing  []string `json:"redist_missing,omitempty"`
}

// IsRedistFile checks if a relative path belongs to shared DirectX/VC++ runtime installers.
func IsRedistFile(relPath string) bool {
	norm := strings.ToLower(NormalizePath(relPath))
	return strings.HasPrefix(norm, "_commonredist") ||
		strings.HasPrefix(norm, "directx") ||
		strings.HasPrefix(norm, "vcredist") ||
		strings.HasPrefix(norm, "redist") ||
		strings.HasPrefix(norm, "installers")
}

// IsChunkValidOnDisk checks if the chunk at the specified offset in f matches the expected ChunkID.
func IsChunkValidOnDisk(f io.ReaderAt, chunk ChunkInfo, buf []byte) bool {
	if chunk.UncompressedSize == 0 {
		return true
	}
	if len(chunk.ChunkID) != 40 {
		return false
	}
	slice := buf
	if len(slice) < int(chunk.UncompressedSize) {
		slice = make([]byte, chunk.UncompressedSize)
	} else {
		slice = slice[:chunk.UncompressedSize]
	}
	if _, err := f.ReadAt(slice, int64(chunk.Offset)); err != nil {
		return false
	}
	sum := sha1.Sum(slice)
	return strings.EqualFold(hex.EncodeToString(sum[:]), chunk.ChunkID)
}

// VerifyFileChunksOnDisk checks if all chunks of a file on disk match their expected chunk hashes.
func VerifyFileChunksOnDisk(filePath string, chunks []ChunkInfo) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 1024*1024)
	for _, ch := range chunks {
		if !IsChunkValidOnDisk(f, ch, buf) {
			return false
		}
	}
	return true
}

// IntegrityProgress reports quantitative telemetry during depot integrity verification.
type IntegrityProgress struct {
	CurrentFile  string  `json:"current_file"`
	CheckedFiles int     `json:"checked_files"`
	TotalFiles   int     `json:"total_files"`
	CheckedBytes int64   `json:"checked_bytes"`
	TotalBytes   int64   `json:"total_bytes"`
	SpeedBps     float64 `json:"speed_bps"`
	ETASeconds   int     `json:"eta_seconds"`
	Percent      float64 `json:"percent"`
}

// IntegrityProgressFunc is a callback invoked with progress updates during integrity scanning.
type IntegrityProgressFunc func(IntegrityProgress)

// ScanDepotIntegrity compares files on disk against official manifest entries for size/SHA1/chunk mismatches.
func ScanDepotIntegrity(gameDir string, appID uint32) (*IntegrityReport, error) {
	return ScanDepotIntegrityWithPlatform(gameDir, appID, "")
}

// ScanDepotIntegrityWithPlatform compares files on disk against official manifest entries matching a target platform.
func ScanDepotIntegrityWithPlatform(gameDir string, appID uint32, platform string) (*IntegrityReport, error) {
	return ScanDepotIntegrityWithProgress(context.Background(), gameDir, appID, platform, nil)
}

// ScanDepotIntegrityWithProgress compares files on disk against official manifest entries with streaming progress telemetry.
func ScanDepotIntegrityWithProgress(ctx context.Context, gameDir string, appID uint32, platform string, onProgress IntegrityProgressFunc) (*IntegrityReport, error) {
	manifest, _ := LoadManifestFromDirWithPlatform(gameDir, appID, nil, platform)

	// Check if manifest files are missing or encrypted
	needsKeyOrManifest := manifest == nil || len(manifest.Files) == 0
	if !needsKeyOrManifest {
		for _, f := range manifest.Files {
			if util.IsEncryptedBase64Path(f.Path) {
				needsKeyOrManifest = true
				break
			}
		}
	}

	if needsKeyOrManifest {
		chain := NewDefaultProviderChain("", "")
		res, err := chain.Resolve(ctx, appID)
		if err == nil && res != nil && len(res.Depots) > 0 {
			depotKeys := make(map[uint32]string)
			for _, pair := range res.Depots {
				depotKeys[pair.DepotID] = pair.DecryptionKey
			}

			// Try loading local files with resolved keys and platform filtering
			if loaded, err := LoadManifestFromDirWithPlatform(gameDir, appID, depotKeys, platform); err == nil && loaded != nil && len(loaded.Files) > 0 {
				manifest = loaded
			} else if len(res.ManifestFiles) > 0 {
				// Parse remote/provider manifests directly with platform filtering
				var allFiles []ManifestFileEntry
				var resolvedDepotID uint32
				for filename, data := range res.ManifestFiles {
					parts := strings.Split(filename, "_")
					var fileDepotID uint32
					if len(parts) >= 1 {
						if did, err := strconv.ParseUint(parts[0], 10, 32); err == nil {
							fileDepotID = uint32(did)
						}
					}
					hexKey := depotKeys[fileDepotID]
					files, err := ParseBinaryManifestWithKey(data, hexKey)
					if err == nil && len(files) > 0 {
						if platform != "" && platform != "all" {
							cat := ClassifyDepot(files)
							if !MatchesPlatform(cat, platform) {
								continue
							}
						}
						if resolvedDepotID == 0 {
							resolvedDepotID = fileDepotID
						}
						for idx := range files {
							files[idx].DepotID = fileDepotID
						}
						allFiles = append(allFiles, files...)
					}
				}
				if len(allFiles) > 0 {
					manifest = &Manifest{
						AppID:     appID,
						DepotID:   resolvedDepotID,
						DepotKeys: depotKeys,
						Files:     allFiles,
					}
				}
			}
		}
	}


	if manifest == nil || len(manifest.Files) == 0 {
		return nil, fmt.Errorf("no SteamPipe depot manifests or keys found for AppID %d", appID)
	}

	// Verify decrypted paths
	for _, f := range manifest.Files {
		if util.IsEncryptedBase64Path(f.Path) {
			return nil, fmt.Errorf("depot manifest files for AppID %d could not be decrypted (missing decryption key)", appID)
		}
	}

	officialMap := make(map[string]ManifestFileEntry, len(manifest.Files))
	foundMap := make(map[string]bool, len(manifest.Files))
	officialCount := 0
	var totalBytes int64

	for _, f := range manifest.Files {
		// Ignore directory manifest nodes from official file count & verification
		if (f.Flags&0x40 != 0) || (len(f.Chunks) == 0 && f.Size == 0) {
			continue
		}
		norm := strings.ToLower(NormalizePath(f.Path))
		officialMap[norm] = f
		officialCount++
		totalBytes += int64(f.Size)
	}

	report := &IntegrityReport{
		OfficialCount: officialCount,
	}

	var checkedFiles int
	var checkedBytes int64
	startTime := time.Now()
	var lastReport time.Time

	reportProgress := func(currentFile string, force bool) {
		if onProgress == nil {
			return
		}
		now := time.Now()
		if !force && now.Sub(lastReport) < 100*time.Millisecond {
			return
		}
		lastReport = now

		elapsed := now.Sub(startTime).Seconds()
		speed := float64(0)
		eta := 0
		if elapsed > 0.5 {
			speed = float64(checkedBytes) / elapsed
			if speed > 0 && totalBytes > checkedBytes {
				eta = int(float64(totalBytes-checkedBytes) / speed)
			}
		}
		pct := float64(0)
		if totalBytes > 0 {
			pct = float64(checkedBytes) / float64(totalBytes) * 100.0
			if pct > 100 {
				pct = 100
			}
		} else if officialCount > 0 {
			pct = float64(checkedFiles) / float64(officialCount) * 100.0
			if pct > 100 {
				pct = 100
			}
		}

		onProgress(IntegrityProgress{
			CurrentFile:  currentFile,
			CheckedFiles: checkedFiles,
			TotalFiles:   officialCount,
			CheckedBytes: checkedBytes,
			TotalBytes:   totalBytes,
			SpeedBps:     speed,
			ETASeconds:   eta,
			Percent:      pct,
		})
	}

	walkErr := filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		rel, relErr := filepath.Rel(gameDir, path)
		if relErr != nil {
			return nil
		}

		normRel := strings.ToLower(NormalizePath(rel))
		baseName := d.Name()

		// Skip internal gamux folders, GPU caches, and Wine log files from untracked reporting
		if strings.HasPrefix(rel, "[Manifests]") ||
			strings.HasPrefix(rel, "[Steam]") ||
			strings.HasPrefix(rel, "steam_settings") ||
			strings.HasPrefix(rel, "GLCache") ||
			strings.HasPrefix(rel, "ShaderCache") ||
			strings.HasPrefix(rel, ".dxvk-cache") ||
			strings.HasSuffix(strings.ToLower(baseName), ".log") ||
			strings.Contains(baseName, ".ORIGINAL") {
			return nil
		}

		entry, isOfficial := officialMap[normRel]
		if isOfficial {
			foundMap[normRel] = true
			if !d.IsDir() {
				checkedFiles++
				checkedBytes += int64(entry.Size)
				reportProgress(rel, false)

				if fi, err := d.Info(); err == nil {
					lowerRel := strings.ToLower(rel)
					isDrmWrapped := (strings.HasSuffix(lowerRel, "steam_api64.dll") || strings.HasSuffix(lowerRel, "steam_api.dll") || strings.HasSuffix(lowerRel, "libsteam_api.so")) && hasOriginalBackup(gameDir, rel)

					corrupted := false
					if uint64(fi.Size()) != entry.Size {
						corrupted = true
					} else if !isDrmWrapped {
						if entry.SHA1 != "" {
							localHash, err := util.GetSHA1Hash(path)
							if err != nil || !strings.EqualFold(localHash, entry.SHA1) {
								corrupted = true
							}
						} else if len(entry.Chunks) > 0 {
							if !VerifyFileChunksOnDisk(path, entry.Chunks) {
								corrupted = true
							}
						}
					}

					if corrupted {
						if isDrmWrapped {
							report.IntactCount++
						} else if IsRedistFile(rel) {
							report.RedistModified = append(report.RedistModified, rel)
						} else {
							report.ModifiedFiles = append(report.ModifiedFiles, rel)
						}
					} else {
						report.IntactCount++
					}
				} else {
					report.IntactCount++
				}
			}
		} else if !d.IsDir() {
			// Skip emulator/gamux generated files from untracked reporting
			if baseName != "steam_appid.txt" &&
				baseName != "ColdClientLoader.ini" &&
				baseName != "steamclient_loader_x64.exe" &&
				baseName != "steamclient_loader_x86.exe" &&
				baseName != "steamclient64.dll" &&
				baseName != "steamclient.dll" &&
				baseName != "GameOverlayRenderer64.dll" &&
				baseName != "GameOverlayRenderer.dll" &&
				baseName != "steam_interfaces.txt" {
				report.UntrackedFiles = append(report.UntrackedFiles, rel)
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	for normRel, entry := range officialMap {
		if !foundMap[normRel] {
			baseName := filepath.Base(entry.Path)
			if baseName == "steam_appid.txt" ||
				baseName == "steam_interfaces.txt" ||
				baseName == "ColdClientLoader.ini" ||
				baseName == "steamclient_loader_x64.exe" ||
				baseName == "steamclient_loader_x86.exe" ||
				baseName == "steamclient64.dll" ||
				baseName == "steamclient.dll" ||
				baseName == "GameOverlayRenderer64.dll" ||
				baseName == "GameOverlayRenderer.dll" {
				continue
			}
			if IsRedistFile(entry.Path) {
				report.RedistMissing = append(report.RedistMissing, entry.Path)
			} else {
				// Ignore empty directory manifest nodes or platform-mismatched files (e.g. .so/.dylib for Windows games)
				if (entry.Flags&0x40 != 0) || (len(entry.Chunks) == 0 && entry.Size == 0) {
					continue
				}
				lowerPath := strings.ToLower(entry.Path)
				if strings.HasSuffix(lowerPath, ".so") || strings.HasSuffix(lowerPath, ".dylib") || strings.Contains(lowerPath, ".app/contents/") {
					continue
				}
				report.MissingFiles = append(report.MissingFiles, entry.Path)
			}
		}
	}

	reportProgress("Completed", true)
	return report, nil
}


// ScanUntrackedFiles walks gameDir and compares every file on disk against official manifest entries.
func ScanUntrackedFiles(gameDir string, appID uint32) (int, []string, error) {
	rep, err := ScanDepotIntegrity(gameDir, appID)
	if err != nil || rep == nil {
		return 0, nil, nil
	}
	return rep.OfficialCount, rep.UntrackedFiles, nil
}

func hasOriginalBackup(gameDir, rel string) bool {
	absPath := filepath.Join(gameDir, rel)
	if matches, err := filepath.Glob(absPath + "*ORIGINAL*"); err == nil && len(matches) > 0 {
		return true
	}
	return false
}

// InspectGameDepots inspects local manifests to produce a structured inventory of all game depots.
func InspectGameDepots(gameDir string, appID uint32) ([]domain.DepotSummary, error) {
	return InspectGameDepotsWithMetadata(gameDir, appID, nil)
}

// InspectGameDepotsWithMetadata inspects local manifests enriched with optional remote metadata to produce a structured inventory of game depots.
func InspectGameDepotsWithMetadata(gameDir string, appID uint32, metaMap map[uint32]DepotMeta) ([]domain.DepotSummary, error) {
	manifestsDir := filepath.Join(gameDir, "[Manifests]")
	var summaries []domain.DepotSummary

	if entries, err := os.ReadDir(manifestsDir); err == nil && len(entries) > 0 {
		depotLatest := make(map[uint32]string)
		depotLatestManifestID := make(map[uint32]uint64)

		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".manifest") {
				parts := strings.Split(e.Name(), "_")
				if len(parts) >= 2 {
					var did uint32
					var mid uint64
					if d, err := strconv.ParseUint(parts[0], 10, 32); err == nil {
						did = uint32(d)
					}
					midStr := strings.TrimSuffix(parts[1], ".manifest")
					if m, err := strconv.ParseUint(midStr, 10, 64); err == nil {
						mid = m
					}
					if mid > depotLatestManifestID[did] {
						depotLatestManifestID[did] = mid
						depotLatest[did] = filepath.Join(manifestsDir, e.Name())
					}
				}
			}
		}

		// Read depot keys from lua if present
		depotKeys := make(map[uint32]string)
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".lua") {
				if content, err := os.ReadFile(filepath.Join(manifestsDir, e.Name())); err == nil {
					if parsed, err := ParseLua(string(content)); err == nil {
						for _, pair := range parsed.Depots {
							depotKeys[pair.DepotID] = pair.DecryptionKey
						}
					}
				}
			}
		}

		// If no depot keys were found locally and appID is known, resolve from default providers
		if len(depotKeys) == 0 && appID > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			chain := NewDefaultProviderChain("", "")
			if res, err := chain.Resolve(ctx, appID); err == nil && res != nil && len(res.Depots) > 0 {
				for _, pair := range res.Depots {
					depotKeys[pair.DepotID] = pair.DecryptionKey
				}
				luaPath := filepath.Join(manifestsDir, fmt.Sprintf("%d.lua", appID))
				_ = SaveLuaFile(luaPath, appID, res.Depots)
			}
			cancel()
		}

		for dID, manifestPath := range depotLatest {
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				continue
			}
			files, err := ParseBinaryManifestWithKey(data, depotKeys[dID])
			if err != nil || len(files) == 0 {
				continue
			}

			cat := ClassifyDepot(files)
			var meta *DepotMeta
			if metaMap != nil {
				if m, ok := metaMap[dID]; ok {
					meta = &m
				}
			}

			// Authoritative platform override from remote metadata
			if meta != nil && meta.OSList != "" {
				switch strings.ToLower(meta.OSList) {
				case "windows", "win":
					cat = DepotCategoryWindows
				case "linux":
					cat = DepotCategoryLinux
				case "macos", "osx", "darwin":
					cat = DepotCategoryMacOS
				}
			}

			contentCat, lang := ClassifyDepotContent(dID, "", files, meta)

			var totalBytes uint64
			fileCount := 0
			installedCount := 0
			execFileCount := 0
			installedExecCount := 0

			for _, f := range files {
				if (f.Flags & 0x40) != 0 {
					continue
				}
				totalBytes += f.Size
				fileCount++

				norm := NormalizePath(f.Path)
				isExec := isBinaryOrExecutable(norm)
				if isExec {
					execFileCount++
				}

				if norm != "" {
					if fi, statErr := os.Stat(filepath.Join(gameDir, norm)); statErr == nil && !fi.IsDir() {
						installedCount++
						if isExec {
							installedExecCount++
						}
					}
				}
			}

			var isInstalled bool
			if fileCount > 0 {
				if execFileCount > 0 {
					// Depots with platform binaries MUST have at least one binary present on disk
					isInstalled = installedExecCount > 0 && float64(installedCount)/float64(fileCount) >= 0.5
				} else {
					// Content / asset depots without binaries require high matching ratio
					isInstalled = installedCount > 0 && float64(installedCount)/float64(fileCount) >= 0.8
				}
			}
			depotName := fmt.Sprintf("Depot %d", dID)
			if meta != nil && meta.Name != "" {
				depotName = meta.Name
			} else if strings.HasPrefix(contentCat, ContentCategoryLocalePrefix) {
				depotName = fmt.Sprintf("%s Language Pack (%d)", strings.Title(lang), dID)
			} else if contentCat == ContentCategoryOptionalHD {
				depotName = fmt.Sprintf("High-Resolution Assets (%d)", dID)
			} else if contentCat == ContentCategorySoundtrack {
				depotName = fmt.Sprintf("Official Soundtrack (%d)", dID)
			} else {
				switch cat {
				case DepotCategoryCommon:
					depotName = fmt.Sprintf("Base Content (%d)", dID)
				case DepotCategoryWindows:
					depotName = fmt.Sprintf("Windows Binaries (%d)", dID)
				case DepotCategoryLinux:
					depotName = fmt.Sprintf("Linux Native Binaries (%d)", dID)
				case DepotCategoryMacOS:
					depotName = fmt.Sprintf("macOS Bundle (%d)", dID)
				case DepotCategorySoundtrack:
					depotName = fmt.Sprintf("Official Soundtrack (%d)", dID)
				}
			}

			var downloadSize uint64
			var osArch string
			var dlcAppID uint32
			var isDLC bool
			var dlcName string
			if meta != nil {
				downloadSize = meta.DownloadSize
				osArch = meta.OSArch
				dlcAppID = meta.DLCAppID
				isDLC = meta.IsDLC
				dlcName = meta.DLCName
				if meta.Size > 0 && totalBytes == 0 {
					totalBytes = meta.Size
				}
			}

			summaries = append(summaries, domain.DepotSummary{
				DepotID:         dID,
				Name:            depotName,
				Category:        string(cat),
				ContentCategory: contentCat,
				Language:        lang,
				Installed:       isInstalled,
				Size:            totalBytes,
				DownloadSize:    downloadSize,
				FileCount:       fileCount,
				ManifestID:      depotLatestManifestID[dID],
				OSArch:          osArch,
				DLCAppID:        dlcAppID,
				IsDLC:           isDLC,
				DLCName:         dlcName,
			})
		}
	}

	// If no local manifests found on disk, but remote metadata exists (e.g. preview or uninstalled stub)
	if len(summaries) == 0 && len(metaMap) > 0 {
		for dID, meta := range metaMap {
			m := meta
			contentCat, lang := ClassifyDepotContent(dID, m.Name, nil, &m)
			cat := DepotCategoryCommon
			if m.OSList == "windows" {
				cat = DepotCategoryWindows
			} else if m.OSList == "linux" {
				cat = DepotCategoryLinux
			} else if m.OSList == "macos" {
				cat = DepotCategoryMacOS
			}

			depotName := m.Name
			if depotName == "" {
				if strings.HasPrefix(contentCat, ContentCategoryLocalePrefix) {
					depotName = fmt.Sprintf("%s Language Pack (%d)", strings.Title(lang), dID)
				} else if contentCat == ContentCategoryOptionalHD {
					depotName = fmt.Sprintf("High-Resolution Assets (%d)", dID)
				} else if contentCat == ContentCategorySoundtrack {
					depotName = fmt.Sprintf("Official Soundtrack (%d)", dID)
				} else {
					archStr := ""
					if m.OSArch == "64" {
						archStr = " (64-bit)"
					} else if m.OSArch == "32" {
						archStr = " (32-bit)"
					}
					prefix := ""
					if m.IsDLC && m.DLCAppID > 0 {
						prefix = fmt.Sprintf("DLC #%d ", m.DLCAppID)
					}
					switch cat {
					case DepotCategoryCommon:
						if m.IsDLC {
							depotName = fmt.Sprintf("%sContent (%d)", prefix, dID)
						} else {
							depotName = fmt.Sprintf("Base Content (%d)", dID)
						}
					case DepotCategoryWindows:
						depotName = fmt.Sprintf("%sWindows Binaries%s (%d)", prefix, archStr, dID)
					case DepotCategoryLinux:
						depotName = fmt.Sprintf("%sLinux Native Binaries%s (%d)", prefix, archStr, dID)
					case DepotCategoryMacOS:
						depotName = fmt.Sprintf("%smacOS Bundle%s (%d)", prefix, archStr, dID)
					default:
						depotName = fmt.Sprintf("Depot %d", dID)
					}
				}
			}

			summaries = append(summaries, domain.DepotSummary{
				DepotID:         dID,
				Name:            depotName,
				Category:        string(cat),
				ContentCategory: contentCat,
				Language:        lang,
				Installed:       false,
				Size:            m.Size,
				DownloadSize:    m.DownloadSize,
				FileCount:       0,
				ManifestID:      0,
				OSArch:          m.OSArch,
				DLCAppID:        m.DLCAppID,
				IsDLC:           m.IsDLC,
				DLCName:         m.DLCName,
			})
		}
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].DepotID < summaries[j].DepotID
	})

	return summaries, nil
}

// DiscoverManifestExecutables discovers candidate launch targets directly from developer depot manifests.
func DiscoverManifestExecutables(gameDir string, appID uint32, platform string) []domain.LaunchCandidate {
	m, err := LoadManifestFromDirWithPlatform(gameDir, appID, nil, platform)
	if err != nil || m == nil || len(m.Files) == 0 {
		return nil
	}

	var candidates []domain.LaunchCandidate
	seen := make(map[string]bool)

	for _, f := range m.Files {
		if (f.Flags & 0x40) != 0 {
			continue
		}
		p := NormalizePath(f.Path)
		pLower := strings.ToLower(p)

		isExe := false
		if strings.HasSuffix(pLower, ".exe") || strings.HasSuffix(pLower, ".bin.x86_64") || strings.HasSuffix(pLower, ".bin.x86") || strings.HasSuffix(pLower, ".x86_64") || strings.HasSuffix(pLower, ".x86") || strings.HasSuffix(pLower, ".app") || strings.Contains(pLower, ".app/contents/macos/") {
			isExe = true
		}

		if !isExe {
			continue
		}

		// Filter out obvious installers, crash dumpers, and redistributables
		if strings.Contains(pLower, "_commonredist") || strings.Contains(pLower, "directx") || strings.Contains(pLower, "vcredist") || strings.Contains(pLower, "crashreport") || strings.Contains(pLower, "unins000") || strings.Contains(pLower, "uninstall") || strings.Contains(pLower, "cleanup") {
			continue
		}

		if seen[p] {
			continue
		}
		seen[p] = true

		baseName := filepath.Base(p)
		cleanName := strings.TrimSuffix(baseName, filepath.Ext(baseName))

		desc := "Discovered Executable"
		if strings.Contains(pLower, "shipping") {
			desc = "Main Shipping Binary"
		} else if strings.Contains(pLower, "dx11") || strings.Contains(pLower, "d3d11") {
			desc = "DirectX 11 Mode"
		} else if strings.Contains(pLower, "dx12") || strings.Contains(pLower, "d3d12") {
			desc = "DirectX 12 Mode"
		} else if strings.Contains(pLower, "vulkan") || strings.Contains(pLower, "vk") {
			desc = "Vulkan Mode"
		} else if strings.Contains(pLower, "server") {
			desc = "Dedicated Server"
		} else if strings.Contains(pLower, "config") || strings.Contains(pLower, "setup") || strings.Contains(pLower, "launcher") {
			desc = "Launcher / Config Utility"
		}

		candidates = append(candidates, domain.LaunchCandidate{
			Name:        cleanName,
			Executable:  p,
			Description: desc,
		})
	}

	// Sort candidates: main game executables first, launcher/config utilities second
	sort.SliceStable(candidates, func(i, j int) bool {
		iLower := strings.ToLower(candidates[i].Executable)
		jLower := strings.ToLower(candidates[j].Executable)
		iIsHelper := strings.Contains(iLower, "config") || strings.Contains(iLower, "setup") || strings.Contains(iLower, "server")
		jIsHelper := strings.Contains(jLower, "config") || strings.Contains(jLower, "setup") || strings.Contains(jLower, "server")
		if iIsHelper != jIsHelper {
			return !iIsHelper
		}
		return len(candidates[i].Executable) < len(candidates[j].Executable)
	})

	return candidates
}

// UninstallDepot removes all files on disk belonging to a specific depot ID, preserving shared files and manifests.
// Returns the count of removed files and freed bytes.
func UninstallDepot(gameDir string, appID uint32, depotID uint32) (int, uint64, error) {
	manifestsDir := filepath.Join(gameDir, "[Manifests]")
	entries, err := os.ReadDir(manifestsDir)
	if err != nil {
		return 0, 0, fmt.Errorf("read manifests dir: %w", err)
	}

	var targetManifest string
	var targetManifestID uint64

	// Read depot keys from lua if present
	depotKeys := make(map[uint32]string)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".lua") {
			if content, err := os.ReadFile(filepath.Join(manifestsDir, e.Name())); err == nil {
				if parsed, err := ParseLua(string(content)); err == nil {
					for _, pair := range parsed.Depots {
						depotKeys[pair.DepotID] = pair.DecryptionKey
					}
				}
			}
		}
	}

	// If no depot keys were found locally and appID is known, resolve from default providers
	if len(depotKeys) == 0 && appID > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		chain := NewDefaultProviderChain("", "")
		if res, err := chain.Resolve(ctx, appID); err == nil && res != nil && len(res.Depots) > 0 {
			for _, pair := range res.Depots {
				depotKeys[pair.DepotID] = pair.DecryptionKey
			}
			luaPath := filepath.Join(manifestsDir, fmt.Sprintf("%d.lua", appID))
			_ = SaveLuaFile(luaPath, appID, res.Depots)
		}
		cancel()
	}

	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".manifest") {
			parts := strings.Split(e.Name(), "_")
			if len(parts) >= 2 {
				var did uint32
				var mid uint64
				if d, err := strconv.ParseUint(parts[0], 10, 32); err == nil {
					did = uint32(d)
				}
				midStr := strings.TrimSuffix(parts[1], ".manifest")
				if m, err := strconv.ParseUint(midStr, 10, 64); err == nil {
					mid = m
				}
				if did == depotID && mid > targetManifestID {
					targetManifestID = mid
					targetManifest = filepath.Join(manifestsDir, e.Name())
				}
			}
		}
	}

	if targetManifest == "" {
		return 0, 0, fmt.Errorf("no manifest found for depot %d", depotID)
	}

	data, err := os.ReadFile(targetManifest)
	if err != nil {
		return 0, 0, fmt.Errorf("read manifest %s: %w", targetManifest, err)
	}

	targetFiles, err := ParseBinaryManifestWithKey(data, depotKeys[depotID])
	if err != nil {
		return 0, 0, fmt.Errorf("parse manifest: %w", err)
	}

	// Collect files from other installed depots to prevent deleting shared files
	otherDepotFiles := make(map[string]bool)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".manifest") {
			parts := strings.Split(e.Name(), "_")
			if len(parts) >= 2 {
				var did uint32
				if d, err := strconv.ParseUint(parts[0], 10, 32); err == nil {
					did = uint32(d)
				}
				if did != depotID {
					otherData, err := os.ReadFile(filepath.Join(manifestsDir, e.Name()))
					if err == nil {
						if oFiles, err := ParseBinaryManifestWithKey(otherData, depotKeys[did]); err == nil {
							for _, of := range oFiles {
								norm := NormalizePath(of.Path)
								if norm != "" {
									otherDepotFiles[norm] = true
								}
							}
						}
					}
				}
			}
		}
	}

	deletedCount := 0
	var freedBytes uint64
	var parentDirs []string

	for _, f := range targetFiles {
		if (f.Flags & 0x40) != 0 {
			continue
		}
		norm := NormalizePath(f.Path)
		if norm == "" || otherDepotFiles[norm] {
			continue
		}

		absPath := filepath.Join(gameDir, norm)
		fi, statErr := os.Stat(absPath)
		if statErr == nil && !fi.IsDir() {
			freedBytes += uint64(fi.Size())
			if removeErr := os.Remove(absPath); removeErr == nil {
				deletedCount++
				parentDirs = append(parentDirs, filepath.Dir(absPath))
			}
		}
	}

	// Clean up empty directories
	for _, pDir := range parentDirs {
		curr := pDir
		for curr != gameDir && curr != "." && curr != "/" && curr != "" {
			_ = os.Remove(curr) // Only succeeds if directory is empty
			curr = filepath.Dir(curr)
		}
	}

	return deletedCount, freedBytes, nil
}




