package downloader

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
	restore := testutil.SilenceLogging()
	code := m.Run()
	restore()
	os.Exit(code)
}


type mockRoundTripper func(req *http.Request) (*http.Response, error)

func (f mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDownloadOrUpdateGame_DryRun(t *testing.T) {
	m := &manifest.Manifest{
		AppID:   12345,
		DepotID: 12346,
		Files: []manifest.ManifestFileEntry{
			{Path: "bin/game.exe", Size: 100},
			{Path: "data/config.json", Size: 50},
		},
	}

	tmpDir := t.TempDir()
	opts := DownloadOptions{
		TargetDir:   tmpDir,
		AppID:       12345,
		DryRun:      true,
		WorkerLimit: 2,
	}

	res, err := DownloadOrUpdateGame(context.Background(), m, opts)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame dry-run failed: %v", err)
	}

	if res.TotalFiles != 2 {
		t.Errorf("expected TotalFiles 2, got %d", res.TotalFiles)
	}
	if res.UpdatedFiles != 2 {
		t.Errorf("expected UpdatedFiles 2, got %d", res.UpdatedFiles)
	}
}

func TestDownloadOrUpdateGame_Execution(t *testing.T) {
	oldTransport := HTTPClient.Transport
	defer func() {
		HTTPClient.Transport = oldTransport
	}()

	HTTPClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("chunk_data_content")),
			Header:     make(http.Header),
		}, nil
	})

	m := &manifest.Manifest{
		AppID:   12345,
		DepotID: 12346,
		Files: []manifest.ManifestFileEntry{
			{
				Path: "game.bin",
				Size: 18,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: "chunk1", UncompressedSize: 18},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	opts := DownloadOptions{
		TargetDir:   tmpDir,
		AppID:       12345,
		DryRun:      false,
		WorkerLimit: 2,
	}

	res, err := DownloadOrUpdateGame(context.Background(), m, opts)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame failed: %v", err)
	}

	if res.UpdatedFiles != 1 {
		t.Errorf("expected UpdatedFiles 1, got %d", res.UpdatedFiles)
	}

	downloadedFile := filepath.Join(tmpDir, "game.bin")
	content, err := os.ReadFile(downloadedFile)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}

	if string(content) != "chunk_data_content" {
		t.Errorf("expected content 'chunk_data_content', got %q", string(content))
	}
}

func TestDecompressChunkData_VSZa(t *testing.T) {
	rawPayload := "This is decompressed chunk payload content for VSZa test."

	var zstdBuf bytes.Buffer
	zw, err := zstd.NewWriter(&zstdBuf)
	if err != nil {
		t.Fatalf("Failed to create zstd writer: %v", err)
	}
	if _, err := zw.Write([]byte(rawPayload)); err != nil {
		t.Fatalf("Failed to write to zstd writer: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Failed to close zstd writer: %v", err)
	}

	var chunkData bytes.Buffer
	chunkData.Write([]byte("VSZa"))
	chunkData.Write([]byte{0x00, 0x00, 0x00, 0x00})
	chunkData.Write(zstdBuf.Bytes())

	decompressed, err := decompressChunkData(chunkData.Bytes(), uint32(len(rawPayload)))
	if err != nil {
		t.Fatalf("decompressChunkData failed for VSZa chunk: %v", err)
	}

	if string(decompressed) != rawPayload {
		t.Errorf("expected decompressed payload %q, got %q", rawPayload, string(decompressed))
	}
}

func TestLevel0ChunkDecompression(t *testing.T) {
	resp, err := http.Get("https://cache10-fra1.steamcontent.com/depot/2477271/chunk/4eb3a382f57e6aaf160b6e72dcb1bdbd632713c8")
	if err != nil {
		t.Skip("Network error fetching test chunk:", err)
	}
	defer resp.Body.Close()
	rawChunkData, _ := io.ReadAll(resp.Body)

	keyBytes, _ := hex.DecodeString("1d68c2af57678e17e48935daee51d4fa41f10d47b531e6129d1d890d0436f2a5")
	block, _ := aes.NewCipher(keyBytes)
	iv := make([]byte, 16)
	block.Decrypt(iv, rawChunkData[:16])

	ciphertext := rawChunkData[16:]
	payloadData := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(payloadData, ciphertext)

	decomp, err := decompressChunkData(payloadData, 12480)
	if err != nil {
		t.Fatalf("decompressChunkData failed for 4eb3a382f57e6aaf160b6e72dcb1bdbd632713c8: %v", err)
	}
	if len(decomp) != 12480 {
		t.Fatalf("expected decompressed length 12480, got %d", len(decomp))
	}
}
