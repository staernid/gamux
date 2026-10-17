package util

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPEIcon_InvalidInput(t *testing.T) {
	err := ExtractPEIcon("", "")
	if err == nil {
		t.Errorf("expected error for empty paths, got nil")
	}

	tempDir := t.TempDir()
	nonExistent := filepath.Join(tempDir, "nonexistent.exe")
	dest := filepath.Join(tempDir, "out.png")
	err = ExtractPEIcon(nonExistent, dest)
	if err == nil {
		t.Errorf("expected error for nonexistent file, got nil")
	}

	// Test with a non-PE file
	dummyFile := filepath.Join(tempDir, "dummy.exe")
	_ = os.WriteFile(dummyFile, []byte("this is not a valid PE file"), 0644)
	err = ExtractPEIcon(dummyFile, dest)
	if err == nil {
		t.Errorf("expected error for invalid PE file, got nil")
	}
}

func TestConvertRawBmpPayloadToPNG(t *testing.T) {
	width, height := 16, 16
	// Construct 40-byte BITMAPINFOHEADER + 16x16 32-bit RGBA pixel data
	var buf bytes.Buffer
	var header [40]byte
	binary.LittleEndian.PutUint32(header[0:4], 40)                // biSize
	binary.LittleEndian.PutUint32(header[4:8], uint32(width))     // biWidth
	binary.LittleEndian.PutUint32(header[8:12], uint32(height*2)) // biHeight (doubled for icon mask)
	binary.LittleEndian.PutUint16(header[12:14], 1)               // biPlanes
	binary.LittleEndian.PutUint16(header[14:16], 32)              // biBitCount (32 bpp)
	buf.Write(header[:])

	// 16x16 pixels * 4 bytes/pixel (BGRA)
	pixels := make([]byte, width*height*4)
	for i := 0; i < len(pixels); i += 4 {
		pixels[i] = 255   // B
		pixels[i+1] = 0   // G
		pixels[i+2] = 0   // R
		pixels[i+3] = 255 // A
	}
	buf.Write(pixels)

	pngBytes, err := convertRawBmpPayloadToPNG(buf.Bytes(), width, height)
	if err != nil {
		t.Fatalf("convertRawBmpPayloadToPNG failed: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("failed to decode generated PNG: %v", err)
	}

	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		t.Errorf("expected PNG dimensions %dx%d, got %dx%d", width, height, img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestExtractPEIcon_LiveHostExecutable(t *testing.T) {
	// If SteamService.exe is present on this machine, test extraction against real binary
	hostExe := "/home/staernid/.local/share/Steam/legacycompat/SteamService.exe"
	if !FileExists(hostExe) {
		t.Skip("SteamService.exe not available on host, skipping live test")
	}

	tempDir := t.TempDir()
	outPNG := filepath.Join(tempDir, "icon.png")

	err := ExtractPEIcon(hostExe, outPNG)
	if err != nil {
		t.Fatalf("ExtractPEIcon failed on live host executable: %v", err)
	}

	if !FileExists(outPNG) {
		t.Fatalf("expected output file %s to exist", outPNG)
	}

	info, err := os.Stat(outPNG)
	if err != nil || info.Size() == 0 {
		t.Fatalf("output PNG is empty or unreadable: %v", err)
	}

	data, err := os.ReadFile(outPNG)
	if err != nil {
		t.Fatal(err)
	}
	// Verify PNG header or valid ICO
	isPNG := bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n"))
	isICO := bytes.HasPrefix(data, []byte{0, 0, 1, 0})
	if !isPNG && !isICO {
		t.Errorf("output file is neither valid PNG nor ICO header")
	}
}
