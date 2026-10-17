package util

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestVSZaDecompression(t *testing.T) {
	sampleText := "Hello World Steam VSZa Chunk Decompression Test 1234567890!"

	// Create compressed zstd frame
	var zstdBuf bytes.Buffer
	zw, err := zstd.NewWriter(&zstdBuf)
	if err != nil {
		t.Fatalf("Failed to create zstd writer: %v", err)
	}
	if _, err := zw.Write([]byte(sampleText)); err != nil {
		t.Fatalf("Failed to write to zstd writer: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Failed to close zstd writer: %v", err)
	}

	// Construct VSZa chunk header (4 bytes VSZa + 4 bytes CRC/header + zstd payload)
	var vszaChunk bytes.Buffer
	vszaChunk.Write([]byte("VSZa"))
	vszaChunk.Write([]byte{0x12, 0x34, 0x56, 0x78})
	vszaChunk.Write(zstdBuf.Bytes())

	if !IsVSZaCompressed(vszaChunk.Bytes()) {
		t.Errorf("Expected IsVSZaCompressed to return true for VSZa header")
	}

	if IsVSZaCompressed([]byte("MZ\x90\x00")) {
		t.Errorf("Expected IsVSZaCompressed to return false for PE header")
	}

	// Test DecompressVSZaData
	decomp, err := DecompressVSZaData(vszaChunk.Bytes())
	if err != nil {
		t.Fatalf("DecompressVSZaData failed: %v", err)
	}
	if string(decomp) != sampleText {
		t.Errorf("DecompressVSZaData expected %q, got %q", sampleText, string(decomp))
	}

	// Test DecompressVSZaFile & DecompressVSZaInDir
	tmpDir := t.TempDir()
	vszaFilePath := filepath.Join(tmpDir, "game.exe")
	if err := os.WriteFile(vszaFilePath, vszaChunk.Bytes(), 0755); err != nil {
		t.Fatalf("Failed to write test VSZa file: %v", err)
	}

	normalFilePath := filepath.Join(tmpDir, "readme.txt")
	if err := os.WriteFile(normalFilePath, []byte("plain text"), 0644); err != nil {
		t.Fatalf("Failed to write normal file: %v", err)
	}

	count, err := DecompressVSZaInDir(tmpDir)
	if err != nil {
		t.Fatalf("DecompressVSZaInDir failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 file decompressed, got %d", count)
	}

	decompContent, err := os.ReadFile(vszaFilePath)
	if err != nil {
		t.Fatalf("Failed to read decompressed file: %v", err)
	}
	if string(decompContent) != sampleText {
		t.Errorf("File decompression expected %q, got %q", sampleText, string(decompContent))
	}
}
