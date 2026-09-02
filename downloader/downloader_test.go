package downloader

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
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
	code := m.Run()
	DefaultHTTPClient = oldClient
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
		HTTPClient: &http.Client{
			Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"response":{"servers":[]}}`)),
					Header:     make(http.Header),
				}, nil
			}),
		},
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

func TestDownload_ExactPathPreservation(t *testing.T) {
	tmpDir := t.TempDir()

	m := &manifest.Manifest{
		AppID:   400,
		DepotID: 401,
		Files: []manifest.ManifestFileEntry{
			{
				Path:  "portal",
				Size:  0,
				Flags: 0x40, // Subdirectory entry
			},
			{
				Path: "portal/maps/testchmb_a_00.bsp",
				Size: 15,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: "chk_map", UncompressedSize: 15},
				},
			},
			{
				Path: "bin/engine.dll",
				Size: 10,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: "chk_dll", UncompressedSize: 10},
				},
			},
			{
				Path: "bin/libsteam_api.so",
				Size: 10,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: "chk_so", UncompressedSize: 10},
				},
			},
			{
				Path: "hl2.exe",
				Size: 8,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: "chk_exe", UncompressedSize: 8},
				},
			},
		},
	}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("testdata12345678")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	gameDir := filepath.Join(tmpDir, "Portal")
	opts := DownloadOptions{
		TargetDir:   gameDir,
		AppID:       400,
		Platform:    "win64",
		DryRun:      false,
		WorkerLimit: 2,
		HTTPClient:  mockClient,
	}

	_, err := DownloadOrUpdateGame(context.Background(), m, opts)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame failed: %v", err)
	}

	// Verify nested directory structure portal/maps/testchmb_a_00.bsp was preserved 1:1
	mapPath := filepath.Join(gameDir, "portal", "maps", "testchmb_a_00.bsp")
	if !util.FileExists(mapPath) {
		t.Errorf("expected map file at %s, but file does not exist (path was corrupted)", mapPath)
	}

	// Verify all depot files are downloaded verbatim (including .so and .dll)
	dllPath := filepath.Join(gameDir, "bin", "engine.dll")
	if !util.FileExists(dllPath) {
		t.Errorf("expected dll file at %s", dllPath)
	}

	soPath := filepath.Join(gameDir, "bin", "libsteam_api.so")
	if !util.FileExists(soPath) {
		t.Errorf("expected so file at %s (should not be filtered out by platform)", soPath)
	}

	exePath := filepath.Join(gameDir, "hl2.exe")
	if !util.FileExists(exePath) {
		t.Errorf("expected exe file at %s", exePath)
	}
}

func TestDownload_MultiPlatformDepotFiltering(t *testing.T) {
	tmpDir := t.TempDir()

	// Multi-depot game simulating Torchlight
	m := &manifest.Manifest{
		AppID: 41500,
		Files: []manifest.ManifestFileEntry{
			// Depot 41501: Common Game Assets
			{
				DepotID: 41501,
				Path:    "Torchlight.pak",
				Size:    16,
				Chunks:  []manifest.ChunkInfo{{ChunkID: "chk_pak", UncompressedSize: 16}},
			},
			// Depot 41502: Windows Binaries
			{
				DepotID: 41502,
				Path:    "Torchlight.exe",
				Size:    16,
				Chunks:  []manifest.ChunkInfo{{ChunkID: "chk_winexe", UncompressedSize: 16}},
			},
			{
				DepotID: 41502,
				Path:    "steam_api.dll",
				Size:    16,
				Chunks:  []manifest.ChunkInfo{{ChunkID: "chk_windll", UncompressedSize: 16}},
			},
			// Depot 41503: macOS Bundle
			{
				DepotID: 41503,
				Path:    "Torchlight.app/Contents/MacOS/Torchlight",
				Size:    16,
				Chunks:  []manifest.ChunkInfo{{ChunkID: "chk_macapp", UncompressedSize: 16}},
			},
			// Depot 41504: Linux Native
			{
				DepotID: 41504,
				Path:    "Torchlight.bin.x86_64",
				Size:    16,
				Chunks:  []manifest.ChunkInfo{{ChunkID: "chk_linbin", UncompressedSize: 16}},
			},
			// Depot 41505: Soundtrack
			{
				DepotID: 41505,
				Path:    "Soundtrack/01-Title.mp3",
				Size:    16,
				Chunks:  []manifest.ChunkInfo{{ChunkID: "chk_mp3", UncompressedSize: 16}},
			},
		},
	}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("testdata12345678")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	// 1. Download for Windows (win64) -> Should download Common (41501) + Windows (41502) ONLY
	winGameDir := filepath.Join(tmpDir, "Torchlight_Win")
	_, err := DownloadOrUpdateGame(context.Background(), m, DownloadOptions{
		TargetDir:   winGameDir,
		AppID:       41500,
		Platform:    "win64",
		DryRun:      false,
		WorkerLimit: 2,
		HTTPClient:  mockClient,
	})
	if err != nil {
		t.Fatalf("Download (win64) failed: %v", err)
	}

	if !util.FileExists(filepath.Join(winGameDir, "Torchlight.pak")) {
		t.Errorf("expected Common asset Torchlight.pak in win64 install")
	}
	if !util.FileExists(filepath.Join(winGameDir, "Torchlight.exe")) {
		t.Errorf("expected Windows binary Torchlight.exe in win64 install")
	}
	if util.FileExists(filepath.Join(winGameDir, "Torchlight.app")) || util.FileExists(filepath.Join(winGameDir, "Torchlight.app", "Contents", "MacOS", "Torchlight")) {
		t.Errorf("macOS .app should NOT be downloaded in win64 install")
	}
	if util.FileExists(filepath.Join(winGameDir, "Torchlight.bin.x86_64")) {
		t.Errorf("Linux binary should NOT be downloaded in win64 install")
	}
	if util.FileExists(filepath.Join(winGameDir, "Soundtrack", "01-Title.mp3")) {
		t.Errorf("Soundtrack should NOT be downloaded into game folder in win64 install")
	}

	// 2. Download for macOS (macos) -> Should download Common (41501) + macOS (41503) ONLY
	macGameDir := filepath.Join(tmpDir, "Torchlight_Mac")
	_, err = DownloadOrUpdateGame(context.Background(), m, DownloadOptions{
		TargetDir:   macGameDir,
		AppID:       41500,
		Platform:    "macos",
		DryRun:      false,
		WorkerLimit: 2,
		HTTPClient:  mockClient,
	})
	if err != nil {
		t.Fatalf("Download (macos) failed: %v", err)
	}

	if !util.FileExists(filepath.Join(macGameDir, "Torchlight.pak")) {
		t.Errorf("expected Common asset Torchlight.pak in macos install")
	}
	if !util.FileExists(filepath.Join(macGameDir, "Torchlight.app", "Contents", "MacOS", "Torchlight")) {
		t.Errorf("expected macOS app bundle in macos install")
	}
	if util.FileExists(filepath.Join(macGameDir, "Torchlight.exe")) {
		t.Errorf("Windows exe should NOT be downloaded in macos install")
	}
	if util.FileExists(filepath.Join(macGameDir, "Torchlight.bin.x86_64")) {
		t.Errorf("Linux binary should NOT be downloaded in macos install")
	}
}

func TestDownload_ChunkChecksumValidation(t *testing.T) {
	chunkPayload := "exact_expected_payload"
	sum := sha1.Sum([]byte(chunkPayload))
	validSHA1 := hex.EncodeToString(sum[:])

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(chunkPayload)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	mValid := &manifest.Manifest{
		AppID:   12345,
		DepotID: 12346,
		Files: []manifest.ManifestFileEntry{
			{
				Path: "valid.bin",
				Size: uint64(len(chunkPayload)),
				Chunks: []manifest.ChunkInfo{
					{ChunkID: validSHA1, UncompressedSize: uint32(len(chunkPayload))},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	res, err := DownloadOrUpdateGame(context.Background(), mValid, DownloadOptions{
		TargetDir:   tmpDir,
		AppID:       12345,
		DryRun:      false,
		WorkerLimit: 1,
		HTTPClient:  mockClient,
	})
	if err != nil {
		t.Fatalf("Download with valid chunk SHA1 failed: %v", err)
	}
	if res.UpdatedFiles != 1 {
		t.Errorf("expected 1 updated file, got %d", res.UpdatedFiles)
	}

	// Test with invalid SHA1 mismatch
	mInvalid := &manifest.Manifest{
		AppID:   12345,
		DepotID: 12346,
		Files: []manifest.ManifestFileEntry{
			{
				Path: "invalid.bin",
				Size: uint64(len(chunkPayload)),
				Chunks: []manifest.ChunkInfo{
					{ChunkID: "0000000000000000000000000000000000000000", UncompressedSize: uint32(len(chunkPayload))},
				},
			},
		},
	}

	_, err = DownloadOrUpdateGame(context.Background(), mInvalid, DownloadOptions{
		TargetDir:   tmpDir,
		AppID:       12345,
		DryRun:      false,
		WorkerLimit: 1,
		HTTPClient:  mockClient,
	})
	if err == nil {
		t.Errorf("expected checksum mismatch error, got nil")
	}
}



