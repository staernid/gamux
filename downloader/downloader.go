package downloader

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/util"
	"github.com/ulikunitz/xz/lzma"
	"golang.org/x/sync/errgroup"
)

// HTTPClient allows injecting custom HTTP clients for testing.
var HTTPClient = http.DefaultClient

// DownloadOptions holds options for downloading or updating game depots directly.
type DownloadOptions struct {
	TargetDir   string
	AppID       uint32
	LuaPath     string
	Platform    string // "win64", "linux", "all"
	DryRun      bool
	WorkerLimit int
}

// Result holds the outcome summary of a download/update operation.
type Result struct {
	TotalFiles     int
	UpdatedFiles   int
	DownloadedBytes uint64
}

type cdnServerEntry struct {
	Host  string `json:"host"`
	VHost string `json:"vhost"`
}

type cdnServerResponse struct {
	Response struct {
		Servers []cdnServerEntry `json:"servers"`
	} `json:"response"`
}

// fetchCDNServers discovers active SteamPipe CDN edge servers via Valve's directory Web API.
func fetchCDNServers(ctx context.Context) ([]string, error) {
	url := "https://api.steampowered.com/IContentServerDirectoryService/GetServersForSteamPipe/v1/?cell_id=0"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create cdn query request: %w", err)
	}
	req.Header.Set("User-Agent", "gamux/1.0")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cdn query request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cdn directory service HTTP status %d", resp.StatusCode)
	}

	var payload cdnServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode cdn response json: %w", err)
	}

	hosts := make([]string, 0, len(payload.Response.Servers))
	for _, s := range payload.Response.Servers {
		h := s.Host
		if h == "" {
			h = s.VHost
		}
		if h != "" {
			hosts = append(hosts, h)
		}
	}

	if len(hosts) == 0 {
		return nil, fmt.Errorf("no cdn hosts returned by SteamPipe service")
	}

	return hosts, nil
}

// DownloadOrUpdateGame compares local files in targetDir against the manifest and downloads missing or updated file chunks.
func DownloadOrUpdateGame(ctx context.Context, m *manifest.Manifest, opts DownloadOptions) (*Result, error) {
	if m == nil {
		return nil, fmt.Errorf("manifest cannot be nil")
	}

	targetDir := opts.TargetDir
	if targetDir == "" {
		targetDir = "."
	}

	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("resolve target directory: %w", err)
	}

	if len(m.Files) == 0 {
		loadedManifest, err := manifest.LoadManifestFromDir(absTargetDir, opts.AppID)
		if err == nil && loadedManifest != nil && len(loadedManifest.Files) > 0 {
			m.Files = loadedManifest.Files
			if m.DepotID == 0 {
				m.DepotID = loadedManifest.DepotID
			}
			slog.Info("Loaded file list from binary manifest", "count", len(m.Files), "depotID", m.DepotID)
		}
	}

	workerLimit := opts.WorkerLimit
	if workerLimit <= 0 {
		workerLimit = 10
	}

	cdnHosts, err := fetchCDNServers(ctx)
	if err != nil {
		slog.Warn("Failed to fetch CDN servers from SteamPipe API, using fallbacks", "error", err)
		cdnHosts = []string{"cache10-fra1.steamcontent.com", "steampipe.steamcontent.com"}
	}

	slog.Info("Starting direct game download/update",
		"appID", m.AppID,
		"depotID", m.DepotID,
		"files", len(m.Files),
		"targetDir", absTargetDir,
		"cdnHosts", len(cdnHosts),
		"dryRun", opts.DryRun,
	)

	result := &Result{
		TotalFiles: len(m.Files),
	}

	manifestsDir := filepath.Join(absTargetDir, "[Manifests]")
	checkpointFile := filepath.Join(manifestsDir, ".gamux-checkpoint.json")
	cp := loadCheckpoint(checkpointFile)

	if !opts.DryRun {
		if err := os.MkdirAll(absTargetDir, 0755); err != nil {
			return nil, fmt.Errorf("create target directory %s: %w", absTargetDir, err)
		}
		_ = os.MkdirAll(manifestsDir, 0755)

		// Pre-create all directories sequentially on main thread
		for _, entry := range m.Files {
			p := filepath.Join(absTargetDir, manifest.NormalizePath(entry.Path))
			if entry.Size == 0 || (entry.Flags&0x40) != 0 {
				if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
					_ = os.Remove(p)
				}
				_ = os.MkdirAll(p, 0755)
			} else {
				parent := filepath.Dir(p)
				if fi, err := os.Stat(parent); err == nil && !fi.IsDir() {
					_ = os.Remove(parent)
				}
				_ = os.MkdirAll(parent, 0755)
			}
		}
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(workerLimit)

	for _, fileEntry := range m.Files {
		entry := fileEntry
		filePath := filepath.Join(absTargetDir, manifest.NormalizePath(entry.Path))

		g.Go(func() error {
			// Skip directory entries in download loop
			if entry.Size == 0 || (entry.Flags&0x40) != 0 {
				return nil
			}

			// Filter out mismatched platform files if specific platform selected
			normPath := strings.ToLower(entry.Path)
			plat := strings.ToLower(opts.Platform)
			if plat == "linux" {
				if strings.HasSuffix(normPath, ".exe") || strings.HasSuffix(normPath, ".dll") || strings.HasSuffix(normPath, ".dylib") || strings.HasSuffix(normPath, ".app") {
					slog.Debug("Filtering out non-Linux file for Linux target", "path", entry.Path)
					return nil
				}
			} else if plat == "win64" || plat == "win32" {
				if strings.HasSuffix(normPath, ".so") || strings.HasSuffix(normPath, ".dylib") {
					slog.Debug("Filtering out non-Windows library for Windows PE target", "path", entry.Path)
					return nil
				}
			} else if plat == "osx" || plat == "macos" {
				if strings.HasSuffix(normPath, ".exe") || strings.HasSuffix(normPath, ".dll") || strings.HasSuffix(normPath, ".so") {
					slog.Debug("Filtering out non-macOS file for macOS target", "path", entry.Path)
					return nil
				}
			}

			needsDownload := false

			// Check if file exists and hash matches
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				if uint64(info.Size()) != entry.Size {
					slog.Info("File size mismatch, needs update", "path", filePath, "localSize", info.Size(), "expectedSize", entry.Size)
					needsDownload = true
				} else if entry.SHA1 != "" {
					localHash, err := util.GetSHA1Hash(filePath)
					if err != nil || !strings.EqualFold(localHash, entry.SHA1) {
						slog.Info("File hash mismatch, needs update", "path", filePath)
						needsDownload = true
					}
				}
			} else {
				slog.Info("New file, needs download", "path", filePath)
				needsDownload = true
			}

			if !needsDownload {
				slog.Info("File is up-to-date", "path", filePath)
				return nil
			}

			if opts.DryRun {
				slog.Info("[DRY RUN] Would download/update file", "path", filePath, "size", entry.Size)
				result.UpdatedFiles++
				return nil
			}

			parentDir := filepath.Dir(filePath)
			if fi, err := os.Stat(parentDir); err == nil && !fi.IsDir() {
				_ = os.Remove(parentDir)
			}
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("create directory for %s: %w", filePath, err)
			}

			// Download chunks or file payload
			if len(entry.Chunks) == 0 {
				// Empty file
				f, err := os.Create(filePath)
				if err != nil {
					return fmt.Errorf("create file %s: %w", filePath, err)
				}
				_ = f.Truncate(int64(entry.Size))
				f.Close()
			} else {
				// Open file for chunk assembly
				fileFlags := os.O_CREATE | os.O_RDWR
				f, err := os.OpenFile(filePath, fileFlags, 0644)
				if err != nil {
					return fmt.Errorf("open target file %s: %w", filePath, err)
				}
				_ = f.Truncate(int64(entry.Size))

				for _, chunk := range entry.Chunks {
					if cp.isCompleted(chunk.ChunkID) {
						slog.Debug("Chunk already completed in checkpoint, skipping", "chunkID", chunk.ChunkID)
						continue
					}

					depotID := entry.DepotID
					if depotID == 0 {
						depotID = m.DepotID
					}
					decKey := m.GetDecryptionKey(depotID)
					if err := downloadChunk(gCtx, cdnHosts, depotID, decKey, chunk, f); err != nil {
						f.Close()
						return fmt.Errorf("download chunk for %s (depot %d): %w", filePath, depotID, err)
					}

					cp.markCompleted(checkpointFile, chunk.ChunkID)
				}
				f.Close()
			}

			slog.Info("Successfully updated file", "path", filePath)
			result.UpdatedFiles++
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("download execution failed: %w", err)
	}

	// Remove checkpoint file on 100% successful completion
	if !opts.DryRun {
		_ = os.Remove(checkpointFile)
	}

	slog.Info("Direct game download/update completed",
		"totalFiles", result.TotalFiles,
		"updatedFiles", result.UpdatedFiles,
	)

	return result, nil
}

func downloadChunk(ctx context.Context, cdnHosts []string, depotID uint32, decKey string, chunk manifest.ChunkInfo, dst io.WriterAt) error {
	if len(cdnHosts) == 0 {
		cdnHosts = []string{"cache10-fra1.steamcontent.com", "steampipe.steamcontent.com", "content1.steampowered.com"}
	} else {
		// Append global fallback hosts if not present
		hasGlobal := false
		for _, h := range cdnHosts {
			if strings.Contains(h, "steampipe.steamcontent.com") {
				hasGlobal = true
				break
			}
		}
		if !hasGlobal {
			cdnHosts = append(cdnHosts, "steampipe.steamcontent.com", "content1.steampowered.com")
		}
	}

	chunkID := strings.ToLower(chunk.ChunkID)
	var lastErr error
	for _, host := range cdnHosts {
		chunkURL := fmt.Sprintf("https://%s/depot/%d/chunk/%s", host, depotID, chunkID)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, chunkURL, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "Valve/Steam HTTP Client 1.0")

		resp, err := HTTPClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP status %d from %s: %s", resp.StatusCode, host, string(body))
			continue
		}

		rawChunkData, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		// 1. Decrypt chunk payload if encrypted with depot key
		payloadData := rawChunkData
		if decKey != "" && len(rawChunkData) >= 16 {
			keyBytes, err := hex.DecodeString(decKey)
			if err == nil && len(keyBytes) == 32 {
				block, err := aes.NewCipher(keyBytes)
				if err == nil {
					// First 16 bytes: ECB-encrypted IV
					encryptedIV := rawChunkData[:16]
					iv := make([]byte, 16)
					block.Decrypt(iv, encryptedIV)

					ciphertext := rawChunkData[16:]
					if len(ciphertext)%16 == 0 && len(ciphertext) > 0 {
						decrypted := make([]byte, len(ciphertext))
						mode := cipher.NewCBCDecrypter(block, iv)
						mode.CryptBlocks(decrypted, ciphertext)
						payloadData = decrypted
					}
				}
			}
		}

		// 2. Decompress chunk payload
		decompressed, err := decompressChunkData(payloadData, chunk.UncompressedSize)
		if err != nil {
			lastErr = fmt.Errorf("decompress chunk %s: %w", chunk.ChunkID, err)
			continue
		}

		// 3. Write chunk payload at expected offset
		if _, err := dst.WriteAt(decompressed, int64(chunk.Offset)); err != nil {
			return fmt.Errorf("write chunk %s at offset %d: %w", chunk.ChunkID, chunk.Offset, err)
		}

		return nil
	}

	return fmt.Errorf("download chunk %s (depot %d) failed across all CDN hosts: %w", chunk.ChunkID, depotID, lastErr)
}

func decompressChunkData(data []byte, expectedSize uint32) ([]byte, error) {
	if len(data) < 2 {
		return data, nil
	}

	// 1. Steam VZ container header ("VZ" / 0x56 0x5A)
	if data[0] == 'V' && data[1] == 'Z' {
		if len(data) < 12 {
			return nil, fmt.Errorf("invalid VZ payload length %d", len(data))
		}
		// Construct standard 13-byte LZMA header:
		// props (1 byte at offset 7) + dict size (4 bytes at 8:12) + uncompressed size (8 bytes uint64) + compressed stream
		var lzmaBuf bytes.Buffer
		lzmaBuf.WriteByte(data[7])
		lzmaBuf.Write(data[8:12])
		_ = binary.Write(&lzmaBuf, binary.LittleEndian, uint64(expectedSize))
		lzmaBuf.Write(data[12:])

		lzr, err := lzma.NewReader(&lzmaBuf)
		if err != nil {
			return nil, fmt.Errorf("lzma reader init: %w", err)
		}
		uncomp, err := io.ReadAll(lzr)
		if err != nil {
			return nil, fmt.Errorf("lzma decode: %w", err)
		}
		return uncomp, nil
	}

	// 2. Zip archive payload ("PK\x03\x04")
	if bytes.Equal(data[:2], []byte{'P', 'K'}) {
		r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err == nil && len(r.File) > 0 {
			rc, err := r.File[0].Open()
			if err == nil {
				uncomp, err := io.ReadAll(rc)
				rc.Close()
				if err == nil {
					return uncomp, nil
				}
			}
		}
	}

	// 3. ZSTD payload (0x28B52FFD)
	if len(data) >= 4 && binary.LittleEndian.Uint32(data[:4]) == 0x28B52FFD {
		zr, err := zstd.NewReader(bytes.NewReader(data))
		if err == nil {
			uncomp, err := io.ReadAll(zr)
			zr.Close()
			if err == nil {
				return uncomp, nil
			}
		}
	}

	// 4. Raw uncompressed fallback
	return data, nil
}

// Checkpoint tracks completed chunk IDs for resume support.
type Checkpoint struct {
	AppID           uint32          `json:"app_id"`
	DepotID         uint32          `json:"depot_id"`
	CompletedChunks map[string]bool `json:"completed_chunks"`
	mu              sync.Mutex
}

func loadCheckpoint(path string) *Checkpoint {
	data, err := os.ReadFile(path)
	if err != nil {
		return &Checkpoint{CompletedChunks: make(map[string]bool)}
	}
	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil || cp.CompletedChunks == nil {
		return &Checkpoint{CompletedChunks: make(map[string]bool)}
	}
	return &cp
}

func (cp *Checkpoint) isCompleted(chunkID string) bool {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.CompletedChunks[chunkID]
}

func (cp *Checkpoint) markCompleted(path string, chunkID string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.CompletedChunks[chunkID] = true
	data, err := json.Marshal(cp)
	if err == nil {
		_ = os.WriteFile(path, data, 0644)
	}
}
