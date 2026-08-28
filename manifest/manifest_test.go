package manifest

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"


	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
	restore := testutil.SilenceLogging()
	code := m.Run()
	restore()
	os.Exit(code)
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




