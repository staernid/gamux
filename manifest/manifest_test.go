package manifest

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"testing"
)

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

