package downloader

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/util"
	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
	restore := testutil.SilenceLogging()
	code := m.Run()
	restore()
	os.Exit(code)
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
	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("chunk_data_content")),
				Header:     make(http.Header),
			}, nil
		}),
	}

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
		HTTPClient:  mockClient,
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

	payloadData, err := manifest.DecryptChunkPayload(rawChunkData, "1d68c2af57678e17e48935daee51d4fa41f10d47b531e6129d1d890d0436f2a5")
	if err != nil {
		t.Fatalf("manifest.DecryptChunkPayload failed: %v", err)
	}

	decomp, err := decompressChunkData(payloadData, 12480)
	if err != nil {
		t.Fatalf("decompressChunkData failed for 4eb3a382f57e6aaf160b6e72dcb1bdbd632713c8: %v", err)
	}
	if len(decomp) != 12480 {
		t.Fatalf("expected decompressed length 12480, got %d", len(decomp))
	}
}

func TestRankCDNEdges(t *testing.T) {
	hosts := []string{"cache1.steamcontent.com", "cache2.steamcontent.com"}
	ranked := rankCDNEdges(context.Background(), nil, hosts)
	if len(ranked) != 2 {
		t.Errorf("expected 2 ranked hosts, got %d", len(ranked))
	}
}

func TestCheckpoint_AtomicSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cpPath := filepath.Join(tmpDir, ".gamux-checkpoint.json")

	cp := loadCheckpoint(cpPath)
	if cp.isCompleted("chunk_abc") {
		t.Error("expected chunk_abc to not be completed")
	}

	cp.markCompleted(cpPath, "chunk_abc")
	if !cp.isCompleted("chunk_abc") {
		t.Error("expected chunk_abc to be completed in memory")
	}

	// Reload from disk to verify atomic save
	reloaded := loadCheckpoint(cpPath)
	if !reloaded.isCompleted("chunk_abc") {
		t.Error("expected chunk_abc to be completed in reloaded checkpoint")
	}
}

func TestDownload_InvalidAppID(t *testing.T) {
	_, err := Download(context.Background(), DownloadRequest{AppID: 0})
	if err == nil {
		t.Fatal("expected error for AppID 0, got nil")
	}
}

func TestDownload_DryRunWithLua(t *testing.T) {
	tmpDir := t.TempDir()
	luaPath := filepath.Join(tmpDir, "test.lua")
	luaContent := `
addappid(12345, 1, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
setDepotKey(12346, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
`
	if err := os.WriteFile(luaPath, []byte(luaContent), 0644); err != nil {
		t.Fatalf("failed to write test lua: %v", err)
	}

	// Create dummy [Manifests] with a manifest file
	manifestsDir := filepath.Join(tmpDir, "[Manifests]")
	_ = os.MkdirAll(manifestsDir, 0755)

	var stageReported string
	var progressReported bool

	req := DownloadRequest{
		AppID:     12345,
		TargetDir: tmpDir,
		LuaPath:   luaPath,
		DryRun:    true,
		OnStage: func(stage, details string) {
			stageReported = stage
		},
		OnProgress: func(p ProgressUpdate) {
			progressReported = true
		},
	}

	// Will fail manifest loading because no .manifest file in manifestsDir, which tests the audit trace
	_, err := Download(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing .manifest file, got nil")
	}

	if stageReported == "" {
		t.Error("expected stage to be reported")
	}
	_ = progressReported
}

func TestDownload_ZeroByteFilesAndDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	m := &manifest.Manifest{
		AppID:   70, // Half-Life
		DepotID: 71,
		Files: []manifest.ManifestFileEntry{
			{
				Path:  "valve",
				Size:  0,
				Flags: 0x40, // Directory flag
			},
			{
				Path:  "valve/violence.cfg",
				Size:  0,
				Flags: 0, // 0-byte regular file
			},
			{
				Path:  "valve/autoexec.cfg",
				Size:  0,
				Flags: 0, // 0-byte regular file
			},
			{
				Path: "valve/game.cfg",
				Size: 12,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: "chk_game_cfg", UncompressedSize: 12},
				},
			},
		},
	}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("bind w +up\n")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	opts := DownloadOptions{
		TargetDir:   tmpDir,
		AppID:       70,
		DryRun:      false,
		WorkerLimit: 2,
		HTTPClient:  mockClient,
	}

	res, err := DownloadOrUpdateGame(context.Background(), m, opts)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame failed: %v", err)
	}

	if res.TotalFiles != 4 {
		t.Errorf("expected 4 total files, got %d", res.TotalFiles)
	}

	// Verify violence.cfg is a regular file and NOT a directory
	violenceFile := filepath.Join(tmpDir, "valve", "violence.cfg")
	fi, err := os.Stat(violenceFile)
	if err != nil {
		t.Fatalf("failed to stat violence.cfg: %v", err)
	}
	if fi.IsDir() {
		t.Errorf("expected violence.cfg to be a regular file, but it is a directory!")
	}
	if fi.Size() != 0 {
		t.Errorf("expected violence.cfg to be 0 bytes, got %d", fi.Size())
	}

	// Verify valve/game.cfg
	gameCfgFile := filepath.Join(tmpDir, "valve", "game.cfg")
	gameFi, err := os.Stat(gameCfgFile)
	if err != nil {
		t.Fatalf("failed to stat game.cfg: %v", err)
	}
	if gameFi.IsDir() {
		t.Errorf("expected game.cfg to be a regular file")
	}
}

func TestDownload_CommonRootPrefixStripping(t *testing.T) {
	tmpDir := t.TempDir()

	m := &manifest.Manifest{
		AppID:   4001890,
		DepotID: 4001891,
		Files: []manifest.ManifestFileEntry{
			{
				Path:  "How to Fish",
				Size:  0,
				Flags: 0x40, // Root Directory entry
			},
			{
				Path: "How to Fish/How to Fish.exe",
				Size: 15,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: "chk_exe", UncompressedSize: 15},
				},
			},
			{
				Path:  "How to Fish/How to Fish_Data",
				Size:  0,
				Flags: 0x40, // Subdirectory
			},
			{
				Path: "How to Fish/How to Fish_Data/app.info",
				Size: 8,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: "chk_info", UncompressedSize: 8},
				},
			},
		},
	}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("testdata")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	gameDir := filepath.Join(tmpDir, "How to Fish")
	opts := DownloadOptions{
		TargetDir:   gameDir,
		AppID:       4001890,
		DryRun:      false,
		WorkerLimit: 2,
		HTTPClient:  mockClient,
	}

	_, err := DownloadOrUpdateGame(context.Background(), m, opts)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame failed: %v", err)
	}

	// Verify files were normalized to gameDir directly without redundant "How to Fish/How to Fish" parent folder
	exePath := filepath.Join(gameDir, "How to Fish.exe")
	if !util.FileExists(exePath) {
		t.Errorf("expected normalized exe at %s, but file does not exist", exePath)
	}

	dataInfoPath := filepath.Join(gameDir, "How to Fish_Data", "app.info")
	if !util.FileExists(dataInfoPath) {
		t.Errorf("expected normalized data at %s, but file does not exist", dataInfoPath)
	}
}

