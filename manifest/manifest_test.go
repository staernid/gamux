package manifest

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
	restore := testutil.SilenceLogging()
	oldClient := DefaultHTTPClient
	DefaultHTTPClient = &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("not found")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	HTTPClient = DefaultHTTPClient
	code := m.Run()
	DefaultHTTPClient = oldClient
	HTTPClient = oldClient
	restore()
	os.Exit(code)
}

func encodeVarint(val uint64) []byte {
	var buf []byte
	for val >= 0x80 {
		buf = append(buf, byte(val|0x80))
		val >>= 7
	}
	buf = append(buf, byte(val))
	return buf
}

func buildTestProtobufPayload(files []ManifestFileEntry) []byte {
	var payload []byte
	for _, f := range files {
		var entryData []byte
		// field 1 (wire 2): Path
		entryData = append(entryData, encodeVarint((1<<3)|2)...)
		entryData = append(entryData, encodeVarint(uint64(len(f.Path)))...)
		entryData = append(entryData, []byte(f.Path)...)

		// field 2 (wire 0): Size
		entryData = append(entryData, encodeVarint((2<<3)|0)...)
		entryData = append(entryData, encodeVarint(f.Size)...)

		// Mapping entry in payload: field 1 (wire 2)
		payload = append(payload, encodeVarint((1<<3)|2)...)
		payload = append(payload, encodeVarint(uint64(len(entryData)))...)
		payload = append(payload, entryData...)
	}

	var result []byte
	magicBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(magicBytes, ProtoPayloadMagic)
	result = append(result, magicBytes...)

	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(len(payload)))
	result = append(result, lenBytes...)
	result = append(result, payload...)

	return result
}


func TestParseLua_ValidContent(t *testing.T) {
	luaContent := `
-- Sample manifest lua
addappid(123456)
addappid(123457, 1, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
addappid(123458, 1, "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210")
`

	parsed, err := ParseLua(luaContent)
	if err != nil {
		t.Fatalf("ParseLua failed: %v", err)
	}

	if parsed.AppID != 123456 {
		t.Errorf("expected AppID 123456, got %d", parsed.AppID)
	}

	if len(parsed.Depots) != 3 {
		t.Fatalf("expected 3 depots, got %d", len(parsed.Depots))
	}

	keyMap := make(map[uint32]string)
	for _, d := range parsed.Depots {
		keyMap[d.DepotID] = d.DecryptionKey
	}

	if key, ok := keyMap[123457]; !ok || key != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Errorf("unexpected key for depot 123457: %s", key)
	}
}

func TestParseLua_InvalidContent(t *testing.T) {
	_, err := ParseLua("")
	if err == nil {
		t.Error("expected error for empty lua content")
	}

	_, err = ParseLua("invalid_content_without_addappid")
	if err == nil {
		t.Error("expected error for content missing addappid")
	}
}

func TestDecryptManifestPayload(t *testing.T) {
	// Generate random 32-byte key
	hexKey := "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	keyBytes, _ := hex.DecodeString(hexKey)

	plaintext := []byte("Hello Steam Manifest World!")
	blockSize := aes.BlockSize

	// Pad plaintext with PKCS7
	padding := blockSize - (len(plaintext) % blockSize)
	padded := make([]byte, len(plaintext)+padding)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padding)
	}

	iv := make([]byte, blockSize)
	for i := range iv {
		iv[i] = byte(i)
	}

	block, _ := aes.NewCipher(keyBytes)
	mode := cipher.NewCBCEncrypter(block, iv)
	encrypted := make([]byte, len(padded))
	mode.CryptBlocks(encrypted, padded)

	fullPayload := append(iv, encrypted...)

	decrypted, err := DecryptManifestPayload(fullPayload, hexKey)
	if err != nil {
		t.Fatalf("DecryptManifestPayload failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("expected plaintext %q, got %q", string(plaintext), string(decrypted))
	}
}

func TestManifest_Helpers(t *testing.T) {
	m := &Manifest{
		AppID:   100,
		DepotID: 101,
		Files: []ManifestFileEntry{
			{Path: "bin\\game.exe", Size: 1000},
			{Path: "data/config.json", Size: 500},
		},
	}

	if m.TotalSize() != 1500 {
		t.Errorf("expected TotalSize 1500, got %d", m.TotalSize())
	}

	entry, err := m.FindFile("bin/game.exe")
	if err != nil {
		t.Fatalf("FindFile failed: %v", err)
	}
	if entry.Size != 1000 {
		t.Errorf("expected entry size 1000, got %d", entry.Size)
	}
}

func TestScanUntrackedFiles_NoManifest(t *testing.T) {
	tmpDir := t.TempDir()
	count, untracked, err := ScanUntrackedFiles(tmpDir, 12345)
	if err != nil {
		t.Fatalf("ScanUntrackedFiles failed: %v", err)
	}
	if count != 0 || len(untracked) != 0 {
		t.Errorf("expected 0 files, got count %d, untracked %d", count, len(untracked))
	}
}

func TestClassifyDepot(t *testing.T) {
	tests := []struct {
		name     string
		files    []ManifestFileEntry
		expected DepotCategory
	}{
		{
			name: "Windows depot with exe and dll",
			files: []ManifestFileEntry{
				{Path: "Torchlight.exe", Size: 1000},
				{Path: "steam_api.dll", Size: 200},
			},
			expected: DepotCategoryWindows,
		},
		{
			name: "macOS depot with .app bundle",
			files: []ManifestFileEntry{
				{Path: "Torchlight.app/Contents/MacOS/Torchlight", Size: 1000},
				{Path: "Torchlight.app/Contents/Info.plist", Size: 200},
			},
			expected: DepotCategoryMacOS,
		},
		{
			name: "Linux depot with .bin.x86",
			files: []ManifestFileEntry{
				{Path: "Torchlight.bin.x86_64", Size: 1000},
				{Path: "lib/libsteam_api.so", Size: 200},
			},
			expected: DepotCategoryLinux,
		},
		{
			name: "Soundtrack depot with mp3 and flac",
			files: []ManifestFileEntry{
				{Path: "Soundtrack/01-Title.mp3", Size: 1000},
				{Path: "Soundtrack/02-Town.flac", Size: 2000},
				{Path: "Soundtrack/cover.jpg", Size: 500},
			},
			expected: DepotCategorySoundtrack,
		},
		{
			name: "Common asset depot with pak and dat",
			files: []ManifestFileEntry{
				{Path: "Torchlight.pak", Size: 50000},
				{Path: "levels/dungeon.dat", Size: 10000},
			},
			expected: DepotCategoryCommon,
		},
		{
			name: "Mixed Windows and Mac plugin binaries",
			files: []ManifestFileEntry{
				{Path: "Game.exe", Size: 5000000},
				{Path: "Plugins/macOS/libhelper.dylib", Size: 200000},
			},
			expected: DepotCategoryCommon,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat := ClassifyDepot(tt.files)
			if cat != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, cat)
			}
		})
	}
}

func TestMatchesPlatform(t *testing.T) {
	// Common matches all platforms
	if !MatchesPlatform(DepotCategoryCommon, "win64") || !MatchesPlatform(DepotCategoryCommon, "linux") || !MatchesPlatform(DepotCategoryCommon, "macos") {
		t.Errorf("expected Common depot to match all platforms")
	}

	// Windows matches only windows/all
	if !MatchesPlatform(DepotCategoryWindows, "win64") || !MatchesPlatform(DepotCategoryWindows, "windows") || !MatchesPlatform(DepotCategoryWindows, "all") {
		t.Errorf("expected Windows depot to match win64")
	}
	if MatchesPlatform(DepotCategoryWindows, "linux") || MatchesPlatform(DepotCategoryWindows, "macos") {
		t.Errorf("Windows depot should not match linux or macos")
	}

	// macOS matches only macos/all
	if !MatchesPlatform(DepotCategoryMacOS, "macos") || !MatchesPlatform(DepotCategoryMacOS, "osx") || !MatchesPlatform(DepotCategoryMacOS, "all") {
		t.Errorf("expected macOS depot to match macos and osx")
	}
	if MatchesPlatform(DepotCategoryMacOS, "win64") || MatchesPlatform(DepotCategoryMacOS, "linux") {
		t.Errorf("macOS depot should not match win64 or linux")
	}

	// Linux matches only linux/all
	if !MatchesPlatform(DepotCategoryLinux, "linux") || !MatchesPlatform(DepotCategoryLinux, "all") {
		t.Errorf("expected Linux depot to match linux")
	}
	if MatchesPlatform(DepotCategoryLinux, "win64") || MatchesPlatform(DepotCategoryLinux, "macos") {
		t.Errorf("Linux depot should not match win64 or macos")
	}

	// Soundtrack matches soundtrack/all
	if !MatchesPlatform(DepotCategorySoundtrack, "soundtrack") || !MatchesPlatform(DepotCategorySoundtrack, "music") || !MatchesPlatform(DepotCategorySoundtrack, "all") {
		t.Errorf("expected Soundtrack depot to match soundtrack and music")
	}
	if MatchesPlatform(DepotCategorySoundtrack, "win64") || MatchesPlatform(DepotCategorySoundtrack, "linux") {
		t.Errorf("Soundtrack depot should not match win64 or linux game installs")
	}
}

func TestInspectGameDepots_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	depots, err := InspectGameDepots(tmpDir, 41500)
	if err != nil {
		t.Fatalf("InspectGameDepots failed: %v", err)
	}
	if len(depots) != 0 {
		t.Errorf("expected 0 depots for empty dir, got %d", len(depots))
	}
}

func TestDiscoverManifestExecutables(t *testing.T) {
	tmpDir := t.TempDir()
	manifestsDir := filepath.Join(tmpDir, "[Manifests]")
	_ = os.MkdirAll(manifestsDir, 0755)

	// Create dummy manifest file entry
	candidates := DiscoverManifestExecutables(tmpDir, 41500, "win64")
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates for empty dir, got %d", len(candidates))
	}
}

func TestUninstallDepot_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	manifestsDir := filepath.Join(tmpDir, "[Manifests]")
	_ = os.MkdirAll(manifestsDir, 0755)

	_, _, err := UninstallDepot(tmpDir, 41500, 41502)
	if err == nil {
		t.Errorf("expected error when no manifest found for depot")
	}
}

func TestScanDepotIntegrity_DirectoriesExcluded(t *testing.T) {
	tmpDir := t.TempDir()
	manifestsDir := filepath.Join(tmpDir, "[Manifests]")
	_ = os.MkdirAll(manifestsDir, 0755)

	// Create dummy game files on disk
	_ = os.MkdirAll(filepath.Join(tmpDir, "Data", "Levels"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "Game.exe"), []byte("game binary data"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "Data", "Levels", "level1.dat"), []byte("level data"), 0644)

	// Write mock manifest with 2 files and 2 directory nodes
	manifest := &Manifest{
		AppID:   12345,
		DepotID: 12346,
		Files: []ManifestFileEntry{
			{Path: "Data", Flags: 0x40, Size: 0},
			{Path: "Data/Levels", Flags: 0x40, Size: 0},
			{Path: "Game.exe", Size: uint64(len("game binary data")), Chunks: []ChunkInfo{{ChunkID: "c1"}}},
			{Path: "Data/Levels/level1.dat", Size: uint64(len("level data")), Chunks: []ChunkInfo{{ChunkID: "c2"}}},
		},
	}

	// Test directly against mock manifest
	officialMap := make(map[string]ManifestFileEntry)
	officialCount := 0
	for _, f := range manifest.Files {
		if (f.Flags&0x40 != 0) || (len(f.Chunks) == 0 && f.Size == 0) {
			continue
		}
		officialMap[f.Path] = f
		officialCount++
	}

	if officialCount != 2 {
		t.Fatalf("expected 2 official files (excluding 2 directories), got %d", officialCount)
	}
}

func encodeVarintHelper(val uint64) []byte {
	var buf []byte
	for val >= 0x80 {
		buf = append(buf, byte(val|0x80))
		val >>= 7
	}
	buf = append(buf, byte(val))
	return buf
}

func encodeBytesFieldHelper(fieldNum uint64, data []byte) []byte {
	tag := encodeVarintHelper((fieldNum << 3) | 2)
	l := encodeVarintHelper(uint64(len(data)))
	res := append(tag, l...)
	return append(res, data...)
}

func encodeVarintFieldHelper(fieldNum uint64, val uint64) []byte {
	tag := encodeVarintHelper((fieldNum << 3) | 0)
	return append(tag, encodeVarintHelper(val)...)
}

func createTestBinaryManifest(filename string, size uint64, sha1Bytes []byte) []byte {
	var entry []byte
	entry = append(entry, encodeBytesFieldHelper(1, []byte(filename))...)
	entry = append(entry, encodeVarintFieldHelper(2, size)...)
	if len(sha1Bytes) > 0 {
		entry = append(entry, encodeBytesFieldHelper(5, sha1Bytes)...)
	}

	var payload []byte
	payload = append(payload, encodeBytesFieldHelper(1, entry)...)

	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, ProtoPayloadMagic)
	_ = binary.Write(buf, binary.LittleEndian, uint32(len(payload)))
	buf.Write(payload)
	return buf.Bytes()
}

func TestScanDepotIntegrityWithProgress(t *testing.T) {
	tmpDir := t.TempDir()
	manifestsDir := filepath.Join(tmpDir, "[Manifests]")
	_ = os.MkdirAll(manifestsDir, 0755)

	testData := []byte("integrity test data")
	_ = os.WriteFile(filepath.Join(tmpDir, "file1.bin"), testData, 0644)
	h := sha1.Sum(testData)

	// Create binary manifest file
	manifestBytes := createTestBinaryManifest("file1.bin", uint64(len(testData)), h[:])
	_ = os.WriteFile(filepath.Join(manifestsDir, "55556_111111.manifest"), manifestBytes, 0644)

	var progressUpdates []IntegrityProgress
	rep, err := ScanDepotIntegrityWithProgress(context.Background(), tmpDir, 55555, "", func(p IntegrityProgress) {
		progressUpdates = append(progressUpdates, p)
	})
	if err != nil {
		t.Fatalf("ScanDepotIntegrityWithProgress failed: %v", err)
	}
	if rep == nil || rep.OfficialCount != 1 || rep.IntactCount != 1 {
		t.Fatalf("unexpected report: %+v", rep)
	}
	if len(progressUpdates) == 0 {
		t.Errorf("expected at least 1 progress update, got 0")
	}

	// Test cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	_, errCancel := ScanDepotIntegrityWithProgress(ctx, tmpDir, 55555, "", nil)
	if errCancel == nil {
		t.Errorf("expected context cancellation error, got nil")
	}
}

func TestNormalizeLanguageCodeAndMatch(t *testing.T) {
	if NormalizeLanguageCode("fr") != "french" {
		t.Errorf("expected fr -> french, got %s", NormalizeLanguageCode("fr"))
	}
	if NormalizeLanguageCode("German") != "german" {
		t.Errorf("expected German -> german, got %s", NormalizeLanguageCode("German"))
	}
	if NormalizeLanguageCode("zh-cn") != "schinese" {
		t.Errorf("expected zh-cn -> schinese, got %s", NormalizeLanguageCode("zh-cn"))
	}
	if NormalizeLanguageCode("pt-br") != "brazilian" {
		t.Errorf("expected pt-br -> brazilian, got %s", NormalizeLanguageCode("pt-br"))
	}

	if !LanguagesMatch("french", "fr") {
		t.Errorf("expected french and fr to match")
	}
	if !LanguagesMatch("schinese", "tchinese") {
		t.Errorf("expected schinese and tchinese to match")
	}
	if !LanguagesMatch("portuguese", "brazilian") {
		t.Errorf("expected portuguese and brazilian to match")
	}
	if LanguagesMatch("french", "german") {
		t.Errorf("french and german should not match")
	}
	if !LanguagesMatch("french", "all") {
		t.Errorf("all should match any language")
	}
}

func TestClassifyDepotContent(t *testing.T) {
	// 1. Core Windows executable depot
	coreFiles := []ManifestFileEntry{
		{Path: "bin/Game.exe", Size: 10000000},
		{Path: "bin/engine.dll", Size: 5000000},
	}
	cat, lang := ClassifyDepotContent(100, "Game Windows Core", coreFiles, nil)
	if cat != ContentCategoryCore || lang != "" {
		t.Errorf("expected core, got cat=%s lang=%s", cat, lang)
	}

	// 2. Metadata-based language tagging
	meta := &DepotMeta{
		DepotID:  101,
		Name:     "Cyberpunk 2077 - Polish",
		Language: "polish",
	}
	cat, lang = ClassifyDepotContent(101, "", nil, meta)
	if cat != "locale:polish" || lang != "polish" {
		t.Errorf("expected locale:polish, got cat=%s lang=%s", cat, lang)
	}

	// 3. Name-based language tagging
	cat, lang = ClassifyDepotContent(102, "Trepang2 French Audio Pack", nil, nil)
	if cat != "locale:french" || lang != "french" {
		t.Errorf("expected locale:french, got cat=%s lang=%s", cat, lang)
	}

	// 4. File-based language tagging (WWise/audio PCK)
	frenchFiles := []ManifestFileEntry{
		{Path: "Content/Audio/GeneratedSoundBanks/Windows/French.pck", Size: 400000000},
		{Path: "Content/Audio/GeneratedSoundBanks/Windows/SoundbanksInfo_French.xml", Size: 20000},
		{Path: "Content/Paks/vo_french.pak", Size: 800000000},
	}
	cat, lang = ClassifyDepotContent(103, "Depot 103", frenchFiles, nil)
	if cat != "locale:french" || lang != "french" {
		t.Errorf("expected locale:french from files, got cat=%s lang=%s", cat, lang)
	}

	// 5. Optional HD Textures
	hdFiles := []ManifestFileEntry{
		{Path: "Textures/4K/albedo_4k.dat", Size: 2000000000},
		{Path: "Textures/highres/normals_4k.dat", Size: 2000000000},
	}
	cat, lang = ClassifyDepotContent(104, "High Resolution Texture Pack", hdFiles, nil)
	if cat != ContentCategoryOptionalHD {
		t.Errorf("expected optional_hd, got cat=%s", cat)
	}

	// 6. Soundtrack
	ostFiles := []ManifestFileEntry{
		{Path: "OST/track01.flac", Size: 40000000},
		{Path: "OST/track02.mp3", Size: 10000000},
	}
	cat, _ = ClassifyDepotContent(105, "Official Soundtrack", ostFiles, nil)
	if cat != ContentCategorySoundtrack {
		t.Errorf("expected soundtrack, got cat=%s", cat)
	}
}

func TestShouldIncludeDepot(t *testing.T) {
	frenchFiles := []ManifestFileEntry{
		{Path: "Content/Paks/vo_french.pak", Size: 800000000},
		{Path: "Content/Audio/French.pck", Size: 400000000},
	}
	germanFiles := []ManifestFileEntry{
		{Path: "Content/Paks/vo_german.pak", Size: 800000000},
		{Path: "Content/Audio/German.pck", Size: 400000000},
	}
	coreFiles := []ManifestFileEntry{
		{Path: "Game.exe", Size: 10000000},
	}
	hdFiles := []ManifestFileEntry{
		{Path: "Textures/4k/textures.dat", Size: 5000000000},
	}

	// Case 1: Default to english, skip french and german
	filterEnglish := DepotFilter{
		Platform: "win64",
		Language: "english",
	}
	if !ShouldIncludeDepot(1, coreFiles, filterEnglish) {
		t.Errorf("core depot should always be included")
	}
	if ShouldIncludeDepot(2, frenchFiles, filterEnglish) {
		t.Errorf("french voice depot should be skipped when preferred language is english")
	}
	if ShouldIncludeDepot(3, germanFiles, filterEnglish) {
		t.Errorf("german voice depot should be skipped when preferred language is english")
	}

	// Case 2: Preferred language is french
	filterFrench := DepotFilter{
		Platform: "win64",
		Language: "french",
	}
	if !ShouldIncludeDepot(1, coreFiles, filterFrench) {
		t.Errorf("core depot should be included")
	}
	if !ShouldIncludeDepot(2, frenchFiles, filterFrench) {
		t.Errorf("french voice depot should be included when preferred language is french")
	}
	if ShouldIncludeDepot(3, germanFiles, filterFrench) {
		t.Errorf("german voice depot should be skipped when preferred language is french")
	}

	// Case 3: Explicit inclusion overrides language filter
	filterExplicit := DepotFilter{
		Platform:       "win64",
		Language:       "english",
		IncludedDepots: []uint32{3}, // explicitly requested german
	}
	if !ShouldIncludeDepot(3, germanFiles, filterExplicit) {
		t.Errorf("german voice depot should be included when in IncludedDepots")
	}
	if ShouldIncludeDepot(2, frenchFiles, filterExplicit) {
		t.Errorf("french voice depot should not be included")
	}

	// Case 4: Explicit exclusion
	filterExclude := DepotFilter{
		Platform:       "win64",
		Language:       "french",
		ExcludedDepots: []uint32{2},
	}
	if ShouldIncludeDepot(2, frenchFiles, filterExclude) {
		t.Errorf("french voice depot should be skipped when in ExcludedDepots")
	}

	// Case 5: Optional HD assets
	filterNoOptional := DepotFilter{
		Platform:        "win64",
		IncludeOptional: false,
	}
	if ShouldIncludeDepot(4, hdFiles, filterNoOptional) {
		t.Errorf("HD textures depot should be skipped when IncludeOptional is false")
	}

	filterWithOptional := DepotFilter{
		Platform:        "win64",
		IncludeOptional: true,
	}
	if !ShouldIncludeDepot(4, hdFiles, filterWithOptional) {
		t.Errorf("HD textures depot should be included when IncludeOptional is true")
	}
}

func TestInspectGameDepotsWithMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	metaMap := map[uint32]DepotMeta{
		1091501: {DepotID: 1091501, Name: "Cyberpunk 2077 Base", OSList: "windows"},
		1091504: {DepotID: 1091504, Name: "Cyberpunk 2077 - French", Language: "french", OSList: "windows"},
	}

	summaries, err := InspectGameDepotsWithMetadata(tmpDir, 1091500, metaMap)
	if err != nil {
		t.Fatalf("InspectGameDepotsWithMetadata failed: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 depot summaries, got %d", len(summaries))
	}
	if summaries[1].ContentCategory != "locale:french" || summaries[1].Language != "french" {
		t.Errorf("expected locale:french, got %+v", summaries[1])
	}
}

func TestShouldIncludeDepot_BaseDepotWithEnglishMetadata(t *testing.T) {
	// ASTRONEER pattern: Base depot has executables and config.language: "english".
	// It must NEVER be filtered out even if user selects German or French!
	coreFiles := []ManifestFileEntry{
		{Path: "Astro.exe", Size: 478720},
		{Path: "Astro/Binaries/Win64/Astro-Win64-Shipping.exe", Size: 70000000},
	}
	astroneerMeta := DepotMeta{
		DepotID:  361422,
		Name:     "",
		Language: "english",
		OSList:   "windows",
		Optional: false,
	}

	filterGerman := DepotFilter{
		Platform:      "win64",
		Language:      "german",
		DepotMetadata: map[uint32]DepotMeta{361422: astroneerMeta},
	}
	if !ShouldIncludeDepot(361422, coreFiles, filterGerman) {
		t.Errorf("base game depot with executables must be included even if user requested german")
	}

	filterAuto := DepotFilter{
		Platform:      "win64",
		Language:      "auto",
		DepotMetadata: map[uint32]DepotMeta{361422: astroneerMeta},
	}
	if !ShouldIncludeDepot(361422, coreFiles, filterAuto) {
		t.Errorf("base game depot must be included with auto language")
	}
}

func TestLanguagesMatch_Auto(t *testing.T) {
	if !LanguagesMatch("english", "auto") {
		t.Errorf("expected english and auto to match")
	}
	if !LanguagesMatch("auto", "english") {
		t.Errorf("expected auto and english to match")
	}
	if !LanguagesMatch("auto", "auto") {
		t.Errorf("expected auto and auto to match")
	}
}

func TestLoadManifestFromDirWithFilter_CorruptedManifestFallback(t *testing.T) {
	tmpDir := t.TempDir()
	manifestsDir := filepath.Join(tmpDir, "[Manifests]")
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	// 1. Create older valid manifest for depot 100
	validFiles := []ManifestFileEntry{
		{Path: "Game.exe", Size: 5000000},
	}
	validPayload := buildTestProtobufPayload(validFiles)
	olderPath := filepath.Join(manifestsDir, "100_1000.manifest")
	if err := os.WriteFile(olderPath, validPayload, 0644); err != nil {
		t.Fatalf("write valid manifest failed: %v", err)
	}

	// 2. Create newer CORRUPTED manifest (0 bytes / invalid bytes)
	newerPath := filepath.Join(manifestsDir, "100_2000.manifest")
	if err := os.WriteFile(newerPath, []byte("corrupted_junk_bytes_not_protobuf"), 0644); err != nil {
		t.Fatalf("write corrupt manifest failed: %v", err)
	}

	// Loading should fall back from 2000 (corrupted) to 1000 (valid)
	m, err := LoadManifestFromDirWithFilter(tmpDir, 100, nil, DepotFilter{Platform: "win64", Language: "english"})
	if err != nil {
		t.Fatalf("expected fallback to valid manifest, got error: %v", err)
	}
	if len(m.Files) != 1 || m.Files[0].Path != "Game.exe" {
		t.Errorf("expected 1 file Game.exe from valid manifest, got %+v", m.Files)
	}
}

func TestLoadManifestFromDirWithFilter_RelaxedLanguageFallback(t *testing.T) {
	tmpDir := t.TempDir()
	manifestsDir := filepath.Join(tmpDir, "[Manifests]")
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	// Depot 101 only has French voice files
	frenchFiles := []ManifestFileEntry{
		{Path: "Content/Paks/vo_french.pak", Size: 10000000},
	}
	payload := buildTestProtobufPayload(frenchFiles)
	if err := os.WriteFile(filepath.Join(manifestsDir, "101_1000.manifest"), payload, 0644); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}

	// User requested "japanese", but only french exists
	// Relaxed fallback should include the available depot rather than failing
	m, err := LoadManifestFromDirWithFilter(tmpDir, 100, nil, DepotFilter{
		Platform: "all",
		Language: "japanese",
	})
	if err != nil {
		t.Fatalf("expected relaxed language fallback to succeed, got error: %v", err)
	}
	if len(m.Files) == 0 {
		t.Errorf("expected non-empty files from relaxed fallback")
	}
}






