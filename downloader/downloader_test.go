package downloader

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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

func TestFetchCDNServers_LoadRanking(t *testing.T) {
	jsonResponse := `{
		"response": {
			"servers": [
				{"host": "cache-pri5-load10.steamcontent.com", "priority_class": 5, "weighted_load": 10, "https_support": "mandatory"},
				{"host": "cache-pri6-load50.steamcontent.com", "priority_class": 6, "weighted_load": 50, "https_support": "mandatory"},
				{"host": "cache-pri6-load20.steamcontent.com", "priority_class": 6, "weighted_load": 20, "https_support": "mandatory"},
				{"host": "cache-unsupported.steamcontent.com", "priority_class": 10, "weighted_load": 1, "https_support": "unsupported"},
				{"host": "", "vhost": "", "priority_class": 10, "weighted_load": 1, "https_support": "mandatory"},
				{"host": "cache-pri4-load10.steamcontent.com", "priority_class": 4, "weighted_load": 10, "https_support": "mandatory"},
				{"host": "cache-pri3-load10.steamcontent.com", "priority_class": 3, "weighted_load": 10, "https_support": "mandatory"},
				{"host": "cache-pri2-load10.steamcontent.com", "priority_class": 2, "weighted_load": 10, "https_support": "mandatory"}
			]
		}
	}`

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	rankedHosts, err := fetchCDNServers(context.Background(), mockClient)
	if err != nil {
		t.Fatalf("fetchCDNServers failed: %v", err)
	}

	expected := []string{
		"cache-pri6-load20.steamcontent.com",
		"cache-pri6-load50.steamcontent.com",
		"cache-pri5-load10.steamcontent.com",
		"cache-pri4-load10.steamcontent.com",
		"cache-pri3-load10.steamcontent.com",
	}

	if len(rankedHosts) != len(expected) {
		t.Fatalf("expected %d hosts, got %d: %v", len(expected), len(rankedHosts), rankedHosts)
	}

	for i, host := range expected {
		if rankedHosts[i] != host {
			t.Errorf("at index %d: expected %s, got %s", i, host, rankedHosts[i])
		}
	}
}

func TestDownload_DeltaPatching(t *testing.T) {
	tmpDir := t.TempDir()
	chunkSize := 1024 * 1024 // 1 MB per chunk

	// Generate 4 chunks
	c0 := bytes.Repeat([]byte("A"), chunkSize)
	c1 := bytes.Repeat([]byte("B"), chunkSize)
	c2Old := bytes.Repeat([]byte("C"), chunkSize)
	c2New := bytes.Repeat([]byte("X"), chunkSize)
	c3 := bytes.Repeat([]byte("D"), chunkSize)

	hash := func(b []byte) string {
		s := sha1.Sum(b)
		return hex.EncodeToString(s[:])
	}

	id0 := hash(c0)
	id1 := hash(c1)
	id2Old := hash(c2Old)
	id2New := hash(c2New)
	id3 := hash(c3)

	var oldFileBuf bytes.Buffer
	oldFileBuf.Write(c0)
	oldFileBuf.Write(c1)
	oldFileBuf.Write(c2Old)
	oldFileBuf.Write(c3)
	oldFileBytes := oldFileBuf.Bytes()
	oldFileSHA1 := hash(oldFileBytes)

	var newFileBuf bytes.Buffer
	newFileBuf.Write(c0)
	newFileBuf.Write(c1)
	newFileBuf.Write(c2New)
	newFileBuf.Write(c3)
	newFileBytes := newFileBuf.Bytes()
	newFileSHA1 := hash(newFileBytes)

	// Pre-create old 4MB file on disk
	gameBinPath := filepath.Join(tmpDir, "game.bin")
	if err := os.WriteFile(gameBinPath, oldFileBytes, 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	oldM := &manifest.Manifest{
		AppID:   5000,
		DepotID: 5001,
		Files: []manifest.ManifestFileEntry{
			{
				Path: "game.bin",
				Size: uint64(len(oldFileBytes)),
				SHA1: oldFileSHA1,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: id0, Offset: 0, UncompressedSize: uint32(chunkSize)},
					{ChunkID: id1, Offset: uint64(chunkSize), UncompressedSize: uint32(chunkSize)},
					{ChunkID: id2Old, Offset: uint64(2 * chunkSize), UncompressedSize: uint32(chunkSize)},
					{ChunkID: id3, Offset: uint64(3 * chunkSize), UncompressedSize: uint32(chunkSize)},
				},
			},
		},
	}

	newM := &manifest.Manifest{
		AppID:   5000,
		DepotID: 5001,
		Files: []manifest.ManifestFileEntry{
			{
				Path: "game.bin",
				Size: uint64(len(newFileBytes)),
				SHA1: newFileSHA1,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: id0, Offset: 0, UncompressedSize: uint32(chunkSize)},
					{ChunkID: id1, Offset: uint64(chunkSize), UncompressedSize: uint32(chunkSize)},
					{ChunkID: id2New, Offset: uint64(2 * chunkSize), UncompressedSize: uint32(chunkSize)},
					{ChunkID: id3, Offset: uint64(3 * chunkSize), UncompressedSize: uint32(chunkSize)},
				},
			},
		},
	}

	// Mock HTTP transport that serves chunk payloads and records requested chunks
	var reqMu sync.Mutex
	requestedChunks := make(map[string]int)
	chunkPayloads := map[string][]byte{
		id0:    c0,
		id1:    c1,
		id2New: c2New,
		id3:    c3,
	}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			for cid, data := range chunkPayloads {
				if strings.Contains(req.URL.Path, cid) {
					reqMu.Lock()
					requestedChunks[cid]++
					reqMu.Unlock()
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(data)),
						Header:     make(http.Header),
					}, nil
				}
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("chunk not found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	opts := DownloadOptions{
		TargetDir:        tmpDir,
		AppID:            5000,
		DryRun:           false,
		WorkerLimit:      1,
		PreviousManifest: oldM,
		HTTPClient:       mockClient,
	}

	res, err := DownloadOrUpdateGame(context.Background(), newM, opts)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame delta patch failed: %v", err)
	}

	if res.UpdatedFiles != 1 {
		t.Errorf("expected 1 updated file, got %d", res.UpdatedFiles)
	}

	// Assert that ONLY the 1 modified chunk was requested over the network!
	if requestedChunks[id2New] != 1 {
		t.Errorf("expected 1 request for modified chunk %s, got %d", id2New, requestedChunks[id2New])
	}
	if requestedChunks[id0] != 0 || requestedChunks[id1] != 0 || requestedChunks[id3] != 0 {
		t.Errorf("unchanged chunks were fetched over the network! requests: %v", requestedChunks)
	}

	// Assert exactly 1MB was transferred over the network
	if res.NetworkBytes != uint64(chunkSize) {
		t.Errorf("expected NetworkBytes == %d (1MB), got %d", chunkSize, res.NetworkBytes)
	}

	// Assert final file content matches new content exactly
	finalBytes, err := os.ReadFile(gameBinPath)
	if err != nil {
		t.Fatalf("failed to read patched file: %v", err)
	}
	if !bytes.Equal(finalBytes, newFileBytes) {
		t.Errorf("final patched file bytes do not match expected newFileBytes")
	}
	if hash(finalBytes) != newFileSHA1 {
		t.Errorf("final file SHA1 mismatch: got %s, want %s", hash(finalBytes), newFileSHA1)
	}
}

func TestDownload_DeltaPatching_CorruptionFallback(t *testing.T) {
	tmpDir := t.TempDir()
	chunkSize := 1024 * 1024 // 1 MB per chunk

	c0 := bytes.Repeat([]byte("A"), chunkSize)
	c1 := bytes.Repeat([]byte("B"), chunkSize)
	c2Old := bytes.Repeat([]byte("C"), chunkSize)
	c2New := bytes.Repeat([]byte("X"), chunkSize)
	c3 := bytes.Repeat([]byte("D"), chunkSize)

	hash := func(b []byte) string {
		s := sha1.Sum(b)
		return hex.EncodeToString(s[:])
	}

	id0 := hash(c0)
	id1 := hash(c1)
	id2Old := hash(c2Old)
	id2New := hash(c2New)
	id3 := hash(c3)

	var corruptedOldBuf bytes.Buffer
	corruptedOldBuf.Write(bytes.Repeat([]byte("Z"), chunkSize)) // Corrupted chunk 0!
	corruptedOldBuf.Write(c1)
	corruptedOldBuf.Write(c2Old)
	corruptedOldBuf.Write(c3)

	var newFileBuf bytes.Buffer
	newFileBuf.Write(c0)
	newFileBuf.Write(c1)
	newFileBuf.Write(c2New)
	newFileBuf.Write(c3)
	newFileBytes := newFileBuf.Bytes()
	newFileSHA1 := hash(newFileBytes)

	// Write corrupted file on disk
	gameBinPath := filepath.Join(tmpDir, "game.bin")
	if err := os.WriteFile(gameBinPath, corruptedOldBuf.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write initial corrupted file: %v", err)
	}

	oldM := &manifest.Manifest{
		AppID:   5000,
		DepotID: 5001,
		Files: []manifest.ManifestFileEntry{
			{
				Path: "game.bin",
				Size: uint64(len(newFileBytes)),
				SHA1: "original_unmodified_sha1",
				Chunks: []manifest.ChunkInfo{
					{ChunkID: id0, Offset: 0, UncompressedSize: uint32(chunkSize)},
					{ChunkID: id1, Offset: uint64(chunkSize), UncompressedSize: uint32(chunkSize)},
					{ChunkID: id2Old, Offset: uint64(2 * chunkSize), UncompressedSize: uint32(chunkSize)},
					{ChunkID: id3, Offset: uint64(3 * chunkSize), UncompressedSize: uint32(chunkSize)},
				},
			},
		},
	}

	newM := &manifest.Manifest{
		AppID:   5000,
		DepotID: 5001,
		Files: []manifest.ManifestFileEntry{
			{
				Path: "game.bin",
				Size: uint64(len(newFileBytes)),
				SHA1: newFileSHA1,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: id0, Offset: 0, UncompressedSize: uint32(chunkSize)},
					{ChunkID: id1, Offset: uint64(chunkSize), UncompressedSize: uint32(chunkSize)},
					{ChunkID: id2New, Offset: uint64(2 * chunkSize), UncompressedSize: uint32(chunkSize)},
					{ChunkID: id3, Offset: uint64(3 * chunkSize), UncompressedSize: uint32(chunkSize)},
				},
			},
		},
	}

	var reqMu sync.Mutex
	requestedChunks := make(map[string]int)
	chunkPayloads := map[string][]byte{
		id0:    c0,
		id1:    c1,
		id2New: c2New,
		id3:    c3,
	}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			for cid, data := range chunkPayloads {
				if strings.Contains(req.URL.Path, cid) {
					reqMu.Lock()
					requestedChunks[cid]++
					reqMu.Unlock()
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(data)),
						Header:     make(http.Header),
					}, nil
				}
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("chunk not found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	opts := DownloadOptions{
		TargetDir:        tmpDir,
		AppID:            5000,
		DryRun:           false,
		WorkerLimit:      1,
		PreviousManifest: oldM,
		HTTPClient:       mockClient,
	}

	_, err := DownloadOrUpdateGame(context.Background(), newM, opts)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame failed during fallback: %v", err)
	}

	// Because chunk 0 on disk was corrupted, SHA1 verification failed after delta patch,
	// triggering the full redownload fallback!
	if requestedChunks[id0] == 0 {
		t.Errorf("expected chunk 0 to be downloaded during fallback")
	}

	finalBytes, err := os.ReadFile(gameBinPath)
	if err != nil {
		t.Fatalf("failed to read recovered file: %v", err)
	}
	if !bytes.Equal(finalBytes, newFileBytes) {
		t.Errorf("recovered file bytes do not match expected newFileBytes")
	}
}

func TestDownload_Idempotency_NoSHA1(t *testing.T) {
	c1Data := []byte("chunk_one_payload_data_12345")
	c2Data := []byte("chunk_two_payload_data_67890")
	sum1 := sha1.Sum(c1Data)
	id1 := hex.EncodeToString(sum1[:])
	sum2 := sha1.Sum(c2Data)
	id2 := hex.EncodeToString(sum2[:])

	fileSize := uint64(len(c1Data) + len(c2Data))

	m := &manifest.Manifest{
		AppID:   12345,
		DepotID: 12346,
		Files: []manifest.ManifestFileEntry{
			{
				Path: "game_no_sha.bin",
				Size: fileSize,
				SHA1: "", // Intentionally empty whole-file SHA1
				Chunks: []manifest.ChunkInfo{
					{ChunkID: id1, Offset: 0, UncompressedSize: uint32(len(c1Data))},
					{ChunkID: id2, Offset: uint64(len(c1Data)), UncompressedSize: uint32(len(c2Data))},
				},
			},
		},
	}

	var chunkFetchCount int32
	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, id1) {
				atomic.AddInt32(&chunkFetchCount, 1)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(c1Data)),
					Header:     make(http.Header),
				}, nil
			}
			if strings.Contains(req.URL.Path, id2) {
				atomic.AddInt32(&chunkFetchCount, 1)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(c2Data)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBufferString("not found"))}, nil
		}),
	}

	tmpDir := t.TempDir()
	opts := DownloadOptions{
		TargetDir:   tmpDir,
		AppID:       12345,
		DryRun:      false,
		WorkerLimit: 2,
		HTTPClient:  mockClient,
	}

	// First run: Downloads all chunks
	res1, err := DownloadOrUpdateGame(context.Background(), m, opts)
	if err != nil {
		t.Fatalf("first DownloadOrUpdateGame failed: %v", err)
	}
	if res1.UpdatedFiles != 1 {
		t.Errorf("expected 1 updated file on first run, got %d", res1.UpdatedFiles)
	}
	if atomic.LoadInt32(&chunkFetchCount) != 2 {
		t.Errorf("expected 2 chunk fetches on first run, got %d", atomic.LoadInt32(&chunkFetchCount))
	}

	// Verify checkpoint file was removed on success
	cpPath := filepath.Join(tmpDir, "[Manifests]", ".gamux-checkpoint.json")
	if util.FileExists(cpPath) {
		t.Errorf("expected checkpoint file to be deleted on success, but it exists: %s", cpPath)
	}

	// Verify file content on disk
	content, err := os.ReadFile(filepath.Join(tmpDir, "game_no_sha.bin"))
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	expectedContent := append(append([]byte{}, c1Data...), c2Data...)
	if !bytes.Equal(content, expectedContent) {
		t.Errorf("file content mismatch on disk")
	}

	// Reset network fetch counter
	atomic.StoreInt32(&chunkFetchCount, 0)

	// Second run: Must be fully idempotent! 0 updated files, 0 network chunk requests!
	res2, err := DownloadOrUpdateGame(context.Background(), m, opts)
	if err != nil {
		t.Fatalf("second DownloadOrUpdateGame failed: %v", err)
	}
	if res2.UpdatedFiles != 0 {
		t.Errorf("expected 0 updated files on second run, got %d", res2.UpdatedFiles)
	}
	if count := atomic.LoadInt32(&chunkFetchCount); count != 0 {
		t.Errorf("expected 0 chunk fetches on second run, got %d (false positive redownload!)", count)
	}
}

func TestDownload_CrossFileChunkDeduplication(t *testing.T) {
	// Two files share identical chunk data
	sharedData := []byte("identical_shared_chunk_across_two_files_12345")
	sum := sha1.Sum(sharedData)
	sharedID := hex.EncodeToString(sum[:])

	m := &manifest.Manifest{
		AppID:   12345,
		DepotID: 12346,
		Files: []manifest.ManifestFileEntry{
			{
				Path: "file_alpha.bin",
				Size: uint64(len(sharedData)),
				Chunks: []manifest.ChunkInfo{
					{ChunkID: sharedID, Offset: 0, UncompressedSize: uint32(len(sharedData))},
				},
			},
			{
				Path: "file_beta.bin",
				Size: uint64(len(sharedData)),
				Chunks: []manifest.ChunkInfo{
					{ChunkID: sharedID, Offset: 0, UncompressedSize: uint32(len(sharedData))},
				},
			},
		},
	}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, sharedID) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(sharedData)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBufferString("not found"))}, nil
		}),
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
	if res.UpdatedFiles != 2 {
		t.Errorf("expected 2 updated files, got %d", res.UpdatedFiles)
	}

	// Verify file_alpha.bin
	contentA, err := os.ReadFile(filepath.Join(tmpDir, "file_alpha.bin"))
	if err != nil {
		t.Fatalf("failed to read file_alpha.bin: %v", err)
	}
	if !bytes.Equal(contentA, sharedData) {
		t.Errorf("file_alpha.bin content mismatch: got %q, want %q", string(contentA), string(sharedData))
	}

	// Verify file_beta.bin (must NOT be empty or skipped!)
	contentB, err := os.ReadFile(filepath.Join(tmpDir, "file_beta.bin"))
	if err != nil {
		t.Fatalf("failed to read file_beta.bin: %v", err)
	}
	if !bytes.Equal(contentB, sharedData) {
		t.Errorf("file_beta.bin corrupted! Cross-file chunk deduplication caused chunk write to be skipped! got %q, want %q", string(contentB), string(sharedData))
	}
}

func TestDownload_NoVSZaCorruption(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-create a legitimate ZIP archive in the target directory
	zipPath := filepath.Join(tmpDir, "archive.zip")
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	fWriter, err := zw.Create("hello.txt")
	if err != nil {
		t.Fatalf("create zip file writer: %v", err)
	}
	_, _ = fWriter.Write([]byte("Hello from inside zip archive!"))
	_ = zw.Close()
	originalZipBytes := zipBuf.Bytes()

	if err := os.WriteFile(zipPath, originalZipBytes, 0644); err != nil {
		t.Fatalf("write test zip archive: %v", err)
	}

	// Download manifest with a completely different file
	chunkPayload := []byte("new_game_file_content_12345")
	sum := sha1.Sum(chunkPayload)
	chunkID := hex.EncodeToString(sum[:])

	m := &manifest.Manifest{
		AppID:   12345,
		DepotID: 12346,
		Files: []manifest.ManifestFileEntry{
			{
				Path: "game.exe",
				Size: uint64(len(chunkPayload)),
				Chunks: []manifest.ChunkInfo{
					{ChunkID: chunkID, Offset: 0, UncompressedSize: uint32(len(chunkPayload))},
				},
			},
		},
	}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(chunkPayload)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	opts := DownloadOptions{
		TargetDir:   tmpDir,
		AppID:       12345,
		DryRun:      false,
		WorkerLimit: 1,
		HTTPClient:  mockClient,
	}

	_, err = DownloadOrUpdateGame(context.Background(), m, opts)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame failed: %v", err)
	}

	// Check that archive.zip was NOT modified, decompressed, or overwritten!
	afterBytes, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatalf("read zip after download: %v", err)
	}
	if !bytes.Equal(afterBytes, originalZipBytes) {
		t.Errorf("archive.zip was corrupted by post-download sweep! length before=%d, length after=%d", len(originalZipBytes), len(afterBytes))
	}
}

func TestDownload_ExecutablePermissions(t *testing.T) {
	tmpDir := t.TempDir()

	c1 := []byte("executable_binary_content")
	c2 := []byte("#!/bin/bash\necho hello\n")
	c3 := []byte("plain text documentation")

	s1 := hex.EncodeToString(sha1Digest(c1))
	s2 := hex.EncodeToString(sha1Digest(c2))
	s3 := hex.EncodeToString(sha1Digest(c3))

	m := &manifest.Manifest{
		AppID:   12345,
		DepotID: 12346,
		Files: []manifest.ManifestFileEntry{
			{
				Path:  "bin/game.x86_64",
				Size:  uint64(len(c1)),
				Flags: 0x01, // k_EManifestFileFlagExecutable
				Chunks: []manifest.ChunkInfo{
					{ChunkID: s1, Offset: 0, UncompressedSize: uint32(len(c1))},
				},
			},
			{
				Path:  "launch.sh",
				Size:  uint64(len(c2)),
				Flags: 0x00,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: s2, Offset: 0, UncompressedSize: uint32(len(c2))},
				},
			},
			{
				Path:  "doc/readme.txt",
				Size:  uint64(len(c3)),
				Flags: 0x00,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: s3, Offset: 0, UncompressedSize: uint32(len(c3))},
				},
			},
		},
	}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			for cid, data := range map[string][]byte{s1: c1, s2: c2, s3: c3} {
				if strings.Contains(req.URL.Path, cid) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(data)),
						Header:     make(http.Header),
					}, nil
				}
			}
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBufferString("not found"))}, nil
		}),
	}

	opts := DownloadOptions{
		TargetDir:   tmpDir,
		AppID:       12345,
		DryRun:      false,
		WorkerLimit: 2,
		HTTPClient:  mockClient,
	}

	_, err := DownloadOrUpdateGame(context.Background(), m, opts)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame failed: %v", err)
	}

	fi1, err := os.Stat(filepath.Join(tmpDir, "bin/game.x86_64"))
	if err != nil {
		t.Fatalf("stat game.x86_64: %v", err)
	}
	if fi1.Mode()&0111 == 0 {
		t.Errorf("expected bin/game.x86_64 to have executable mode, got %v", fi1.Mode())
	}

	fi2, err := os.Stat(filepath.Join(tmpDir, "launch.sh"))
	if err != nil {
		t.Fatalf("stat launch.sh: %v", err)
	}
	if fi2.Mode()&0111 == 0 {
		t.Errorf("expected launch.sh to have executable mode, got %v", fi2.Mode())
	}

	fi3, err := os.Stat(filepath.Join(tmpDir, "doc/readme.txt"))
	if err != nil {
		t.Fatalf("stat readme.txt: %v", err)
	}
	if fi3.Mode()&0111 != 0 {
		t.Errorf("expected readme.txt to NOT have executable mode, got %v", fi3.Mode())
	}
}

func TestDownload_ResumeInterrupted(t *testing.T) {
	tmpDir := t.TempDir()

	c1 := []byte("chunk_part_1_11111")
	c2 := []byte("chunk_part_2_22222")
	c3 := []byte("chunk_part_3_33333")

	s1 := hex.EncodeToString(sha1Digest(c1))
	s2 := hex.EncodeToString(sha1Digest(c2))
	s3 := hex.EncodeToString(sha1Digest(c3))

	totalSize := uint64(len(c1) + len(c2) + len(c3))

	m := &manifest.Manifest{
		AppID:   12345,
		DepotID: 12346,
		Files: []manifest.ManifestFileEntry{
			{
				Path: "resumable.bin",
				Size: totalSize,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: s1, Offset: 0, UncompressedSize: uint32(len(c1))},
					{ChunkID: s2, Offset: uint64(len(c1)), UncompressedSize: uint32(len(c2))},
					{ChunkID: s3, Offset: uint64(len(c1) + len(c2)), UncompressedSize: uint32(len(c3))},
				},
			},
		},
	}

	var failChunk2 atomic.Bool
	failChunk2.Store(true)
	var s1FetchedOnResume atomic.Int32

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, s1) {
				if !failChunk2.Load() {
					s1FetchedOnResume.Add(1)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(c1)),
					Header:     make(http.Header),
				}, nil
			}
			if strings.Contains(req.URL.Path, s2) {
				if failChunk2.Load() {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(bytes.NewBufferString("simulated network drop")),
						Header:     make(http.Header),
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(c2)),
					Header:     make(http.Header),
				}, nil
			}
			if strings.Contains(req.URL.Path, s3) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(c3)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBufferString("not found"))}, nil
		}),
	}

	opts := DownloadOptions{
		TargetDir:   tmpDir,
		AppID:       12345,
		DryRun:      false,
		WorkerLimit: 1,
		HTTPClient:  mockClient,
	}

	// Run 1: Must fail on chunk 2
	_, err := DownloadOrUpdateGame(context.Background(), m, opts)
	if err == nil {
		t.Fatalf("expected first download run to fail, but it succeeded")
	}

	// Checkpoint file should exist because download failed/interrupted
	cpPath := filepath.Join(tmpDir, "[Manifests]", ".gamux-checkpoint.json")
	if !util.FileExists(cpPath) {
		t.Fatalf("expected checkpoint file to be preserved on interrupted download, but not found")
	}

	// Run 2: Resume
	failChunk2.Store(false)
	res, err := DownloadOrUpdateGame(context.Background(), m, opts)
	if err != nil {
		t.Fatalf("resumed download failed: %v", err)
	}
	if res.UpdatedFiles != 1 {
		t.Errorf("expected 1 updated file on resume, got %d", res.UpdatedFiles)
	}
	if s1FetchedOnResume.Load() > 0 {
		t.Errorf("chunk 1 was re-fetched on resume, expected it to be skipped via checkpoint")
	}

	// Verify final file on disk is complete
	content, err := os.ReadFile(filepath.Join(tmpDir, "resumable.bin"))
	if err != nil {
		t.Fatalf("read resumable.bin: %v", err)
	}
	expected := append(append(append([]byte{}, c1...), c2...), c3...)
	if !bytes.Equal(content, expected) {
		t.Errorf("resumed file content mismatch: got %d bytes, want %d bytes", len(content), len(expected))
	}
}

func TestDownload_DifferentCompressionFormats(t *testing.T) {
	tmpDir := t.TempDir()

	raw1 := []byte("decompressed_chunk_data_vsza_zstd_frame_12345")
	raw2 := []byte("decompressed_chunk_data_zip_container_frame_67890")
	raw3 := []byte("decompressed_chunk_data_plain_raw_uncompressed_abcde")

	s1 := hex.EncodeToString(sha1Digest(raw1))
	s2 := hex.EncodeToString(sha1Digest(raw2))
	s3 := hex.EncodeToString(sha1Digest(raw3))

	// Construct VSZa chunk (4 bytes VSZa + 4 bytes header + zstd payload)
	var zstdBuf bytes.Buffer
	zw, _ := zstd.NewWriter(&zstdBuf)
	_, _ = zw.Write(raw1)
	_ = zw.Close()
	var vszaChunk bytes.Buffer
	vszaChunk.Write([]byte("VSZa\x00\x00\x00\x00"))
	vszaChunk.Write(zstdBuf.Bytes())

	// Construct Zip chunk
	var zipBuf bytes.Buffer
	zwZip := zip.NewWriter(&zipBuf)
	zf, _ := zwZip.Create("chunk")
	_, _ = zf.Write(raw2)
	_ = zwZip.Close()

	totalSize := uint64(len(raw1) + len(raw2) + len(raw3))

	m := &manifest.Manifest{
		AppID:   12345,
		DepotID: 12346,
		Files: []manifest.ManifestFileEntry{
			{
				Path: "multi_compression.bin",
				Size: totalSize,
				Chunks: []manifest.ChunkInfo{
					{ChunkID: s1, Offset: 0, UncompressedSize: uint32(len(raw1))},
					{ChunkID: s2, Offset: uint64(len(raw1)), UncompressedSize: uint32(len(raw2))},
					{ChunkID: s3, Offset: uint64(len(raw1) + len(raw2)), UncompressedSize: uint32(len(raw3))},
				},
			},
		},
	}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, s1) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(vszaChunk.Bytes())),
					Header:     make(http.Header),
				}, nil
			}
			if strings.Contains(req.URL.Path, s2) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(zipBuf.Bytes())),
					Header:     make(http.Header),
				}, nil
			}
			if strings.Contains(req.URL.Path, s3) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(raw3)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBufferString("not found"))}, nil
		}),
	}

	opts := DownloadOptions{
		TargetDir:   tmpDir,
		AppID:       12345,
		DryRun:      false,
		WorkerLimit: 1,
		HTTPClient:  mockClient,
	}

	_, err := DownloadOrUpdateGame(context.Background(), m, opts)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "multi_compression.bin"))
	if err != nil {
		t.Fatalf("read multi_compression.bin: %v", err)
	}
	expected := append(append(append([]byte{}, raw1...), raw2...), raw3...)
	if !bytes.Equal(content, expected) {
		t.Errorf("content mismatch for multi-compression chunk download")
	}
}

func sha1Digest(b []byte) []byte {
	h := sha1.Sum(b)
	return h[:]
}

func TestDownload_InPlaceChunkRepair(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gamux_inplace_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	chunk0Data := []byte("VALID_CHUNK_0_CONTENT_DATA!")
	chunk1Data := []byte("CORRUPTED_CHUNK_1_TO_BE_REPLACED!")
	chunk2Data := []byte("VALID_CHUNK_2_CONTENT_DATA!")

	c0Hash := hex.EncodeToString(sha1Digest(chunk0Data))
	c1Hash := hex.EncodeToString(sha1Digest(chunk1Data))
	c2Hash := hex.EncodeToString(sha1Digest(chunk2Data))

	// On disk: chunk0 is valid, chunk1 is corrupted, chunk2 is valid
	diskFile := filepath.Join(tmpDir, "game.bin")
	corruptChunk1 := bytes.Repeat([]byte{0x00}, len(chunk1Data))
	initialOnDisk := append(append(append([]byte{}, chunk0Data...), corruptChunk1...), chunk2Data...)
	if err := os.WriteFile(diskFile, initialOnDisk, 0644); err != nil {
		t.Fatal(err)
	}

	m := &manifest.Manifest{
		AppID:   999,
		DepotID: 100,
		Files: []manifest.ManifestFileEntry{
			{
				Path: "game.bin",
				Size: uint64(len(initialOnDisk)),
				Chunks: []manifest.ChunkInfo{
					{
						ChunkID:          c0Hash,
						Offset:           0,
						UncompressedSize: uint32(len(chunk0Data)),
					},
					{
						ChunkID:          c1Hash,
						Offset:           uint64(len(chunk0Data)),
						UncompressedSize: uint32(len(chunk1Data)),
					},
					{
						ChunkID:          c2Hash,
						Offset:           uint64(len(chunk0Data) + len(chunk1Data)),
						UncompressedSize: uint32(len(chunk2Data)),
					},
				},
			},
		},
	}

	networkFetchCount := 0
	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, c1Hash) {
				networkFetchCount++
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(chunk1Data)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBufferString("not found"))}, nil
		}),
	}

	opts := DownloadOptions{
		TargetDir:   tmpDir,
		AppID:       999,
		WorkerLimit: 1,
		HTTPClient:  mockClient,
	}

	res, err := DownloadOrUpdateGame(context.Background(), m, opts)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame failed: %v", err)
	}

	if networkFetchCount != 1 {
		t.Errorf("expected exactly 1 chunk network fetch, got %d", networkFetchCount)
	}

	if res.NetworkBytes != uint64(len(chunk1Data)) {
		t.Errorf("expected NetworkBytes %d, got %d", len(chunk1Data), res.NetworkBytes)
	}

	// Verify repaired file on disk
	repaired, err := os.ReadFile(diskFile)
	if err != nil {
		t.Fatalf("read repaired file: %v", err)
	}
	expectedRepaired := append(append(append([]byte{}, chunk0Data...), chunk1Data...), chunk2Data...)
	if !bytes.Equal(repaired, expectedRepaired) {
		t.Errorf("repaired file content mismatch")
	}

	// Verify that a second run does 0 network fetches
	res2, err := DownloadOrUpdateGame(context.Background(), m, opts)
	if err != nil {
		t.Fatalf("second DownloadOrUpdateGame failed: %v", err)
	}
	if res2.NetworkBytes != 0 {
		t.Errorf("expected 0 NetworkBytes on second run, got %d", res2.NetworkBytes)
	}
}

func TestDownloadOrUpdateGame_LanguageAndDepotFiltering(t *testing.T) {
	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"response":{"servers":[]}}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	m := &manifest.Manifest{
		AppID:   500,
		DepotID: 501,
		Files: []manifest.ManifestFileEntry{
			// Depot 501: Core game binaries
			{DepotID: 501, Path: "Game.exe", Size: 100},
			{DepotID: 501, Path: "data.pak", Size: 200},
			// Depot 502: French voice files
			{DepotID: 502, Path: "Audio/French.pck", Size: 300},
			{DepotID: 502, Path: "Audio/SoundbanksInfo_French.xml", Size: 40},
			// Depot 503: German voice files
			{DepotID: 503, Path: "Audio/German.pck", Size: 300},
			{DepotID: 503, Path: "Audio/SoundbanksInfo_German.xml", Size: 40},
			// Depot 504: 4K HD textures
			{DepotID: 504, Path: "Textures/4k/textures_4k.dat", Size: 1000},
		},
	}

	tmpDir := t.TempDir()

	// 1. Default english download: Core files only (2 files), voice packs and 4K skipped
	optsDefault := DownloadOptions{
		TargetDir:   tmpDir,
		AppID:       500,
		Platform:    "win64",
		Language:    "english",
		DryRun:      true,
		WorkerLimit: 2,
		HTTPClient:  mockClient,
	}

	res1, err := DownloadOrUpdateGame(context.Background(), m, optsDefault)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame default failed: %v", err)
	}
	if res1.TotalFiles != 2 {
		t.Errorf("expected 2 files for english/core download, got %d", res1.TotalFiles)
	}

	// 2. French language preference: Core files (2) + French files (2) = 4 files
	optsFrench := DownloadOptions{
		TargetDir:   tmpDir,
		AppID:       500,
		Platform:    "win64",
		Language:    "french",
		DryRun:      true,
		WorkerLimit: 2,
		HTTPClient:  mockClient,
	}

	res2, err := DownloadOrUpdateGame(context.Background(), m, optsFrench)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame french failed: %v", err)
	}
	if res2.TotalFiles != 4 {
		t.Errorf("expected 4 files for french download, got %d", res2.TotalFiles)
	}

	// 3. Explicit depot inclusion: English + German depot (503) = 2 (core) + 2 (german) = 4 files
	optsExplicit := DownloadOptions{
		TargetDir:      tmpDir,
		AppID:          500,
		Platform:       "win64",
		Language:       "english",
		IncludedDepots: []uint32{501, 503},
		DryRun:         true,
		WorkerLimit:    2,
		HTTPClient:     mockClient,
	}

	res3, err := DownloadOrUpdateGame(context.Background(), m, optsExplicit)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame explicit failed: %v", err)
	}
	if res3.TotalFiles != 4 {
		t.Errorf("expected 4 files for explicit inclusion, got %d", res3.TotalFiles)
	}

	// 4. Include optional HD: Core (2) + 4K (1) = 3 files
	optsHD := DownloadOptions{
		TargetDir:       tmpDir,
		AppID:           500,
		Platform:        "win64",
		Language:        "english",
		IncludeOptional: true,
		DryRun:          true,
		WorkerLimit:     2,
		HTTPClient:      mockClient,
	}

	res4, err := DownloadOrUpdateGame(context.Background(), m, optsHD)
	if err != nil {
		t.Fatalf("DownloadOrUpdateGame HD failed: %v", err)
	}
	if res4.TotalFiles != 3 {
		t.Errorf("expected 3 files with IncludeOptional, got %d", res4.TotalFiles)
	}
}





