package downloader

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
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
	"sync/atomic"
	"time"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/ui"
	"github.com/staernid/gamux/util"
	"golang.org/x/sync/errgroup"
)

// HTTPClient allows injecting custom HTTP clients for testing.
var HTTPClient = &http.Client{Timeout: 30 * time.Second}

// DownloadOptions holds options for downloading or updating game depots directly.
type DownloadOptions struct {
	TargetDir        string
	AppID            uint32
	LuaPath          string
	Platform         string // "win64", "linux", "all"
	DryRun           bool
	WorkerLimit      int
	AuditTrace       []string
	ProgressCallback func(current, total int, item string)
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
		loadedManifest, err := manifest.LoadManifestFromDirWithKeys(absTargetDir, opts.AppID, m.DepotKeys)
		if err == nil && loadedManifest != nil && len(loadedManifest.Files) > 0 {
			m.Files = loadedManifest.Files
			if m.DepotID == 0 {
				m.DepotID = loadedManifest.DepotID
			}
			if len(m.DepotKeys) == 0 && len(loadedManifest.DepotKeys) > 0 {
				m.DepotKeys = loadedManifest.DepotKeys
			}
			slog.Info("Loaded file list from binary manifest", "count", len(m.Files), "depotID", m.DepotID)
		}
	}

	if len(m.Files) == 0 {
		trace := opts.AuditTrace
		manifestsDir := filepath.Join(absTargetDir, "[Manifests]")
		if entries, err := os.ReadDir(manifestsDir); err == nil {
			trace = append(trace, fmt.Sprintf("Local [Manifests] Dir ('%s'): %d files found, 0 valid .manifest file entries", manifestsDir, len(entries)))
		} else {
			trace = append(trace, fmt.Sprintf("Local [Manifests] Dir ('%s'): Directory missing", manifestsDir))
		}

		traceStr := ""
		if len(trace) > 0 {
			traceStr = "\n\n🔍 Key & Manifest Provider Resolution Audit Trace:\n  • " + strings.Join(trace, "\n  • ")
		}

		return nil, fmt.Errorf("no depot binary manifest (.manifest) files found in '[Manifests]/' or resolved for AppID %d%s", opts.AppID, traceStr)
	}

	// Validate that paths in manifest are decrypted
	for _, f := range m.Files {
		if util.IsEncryptedBase64Path(f.Path) {
			return nil, fmt.Errorf("manifest contains encrypted filenames for depot %d and no valid depot decryption key was found to decrypt them. Aborting download to prevent corrupting target directory", f.DepotID)
		}
	}



	workerLimit := opts.WorkerLimit
	if workerLimit <= 0 {
		workerLimit = 10
	}

	cdnHosts, err := fetchCDNServers(ctx)
	if err != nil {
		slog.Warn("Failed to fetch CDN servers from SteamPipe API, using fallbacks", "error", err)
		cdnHosts = []string{"cache10-fra1.steamcontent.com", "cache4-fra1.steamcontent.com", "cache1-fra1.steamcontent.com"}
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

	var processedCount int32
	var processedUpdated int32
	totalFiles := len(m.Files)

	for _, fileEntry := range m.Files {
		entry := fileEntry
		filePath := filepath.Join(absTargetDir, manifest.NormalizePath(entry.Path))

		g.Go(func() error {
			curr := atomic.AddInt32(&processedCount, 1)
			if !opts.DryRun {
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(int(curr), totalFiles, entry.Path)
				} else {
					ui.RenderProgress(int(curr), totalFiles, entry.Path)
				}
			}

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
					slog.Debug("File size mismatch, needs update", "path", filePath, "localSize", info.Size(), "expectedSize", entry.Size)
					needsDownload = true
				} else if entry.SHA1 != "" {
					localHash, err := util.GetSHA1Hash(filePath)
					if err != nil || !strings.EqualFold(localHash, entry.SHA1) {
						slog.Debug("File hash mismatch, needs update", "path", filePath)
						needsDownload = true
					}
				}
			} else {
				slog.Debug("New file, needs download", "path", filePath)
				needsDownload = true
			}

			if !needsDownload {
				slog.Debug("File is up-to-date", "path", filePath)
				return nil
			}

			if opts.DryRun {
				slog.Debug("[DRY RUN] Would download/update file", "path", filePath, "size", entry.Size)
				atomic.AddInt32(&processedUpdated, 1)
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

			slog.Debug("Successfully updated file", "path", filePath)
			atomic.AddInt32(&processedUpdated, 1)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		ui.ClearProgress()
		return nil, fmt.Errorf("download execution failed: %w", err)
	}

	ui.ClearProgress()
	result.UpdatedFiles = int(atomic.LoadInt32(&processedUpdated))

	// Remove checkpoint file and sweep for any remaining VSZa compressed files on 100% successful completion
	if !opts.DryRun {
		_ = os.Remove(checkpointFile)
		if count, err := util.DecompressVSZaInDir(absTargetDir); err == nil && count > 0 {
			slog.Info("Post-download VSZa file decompression completed", "count", count)
		}
	}

	slog.Debug("Direct game download/update completed",
		"totalFiles", result.TotalFiles,
		"updatedFiles", result.UpdatedFiles,
	)

	return result, nil

}

func downloadChunk(ctx context.Context, cdnHosts []string, depotID uint32, decKey string, chunk manifest.ChunkInfo, dst io.WriterAt) error {
	if len(cdnHosts) == 0 {
		cdnHosts = []string{"cache10-fra1.steamcontent.com", "cache4-fra1.steamcontent.com", "cache1-fra1.steamcontent.com"}
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
	return util.DecompressChunkSlice(data, expectedSize)
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

// DownloadGame orchestrates key resolution, manifest persistence, and depot chunk downloading for an AppID.
func DownloadGame(ctx context.Context, cfg *config.Config, opts DownloadOptions) (*Result, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	hubcapKey := cfg.HubcapAPIKey

	parsedLua, err := manifest.ResolveKeys(ctx, opts.AppID, opts.LuaPath, hubcapKey)
	if err != nil {
		return nil, fmt.Errorf("resolve keys and manifests for AppID %d: %w", opts.AppID, err)
	}

	opts.AppID = parsedLua.AppID
	if opts.TargetDir == "" {
		opts.TargetDir = "."
	}

	if !opts.DryRun && len(parsedLua.ManifestFiles) > 0 {
		manifestsDir := filepath.Join(opts.TargetDir, "[Manifests]")
		_ = os.MkdirAll(manifestsDir, 0755)
		for fname, fcontent := range parsedLua.ManifestFiles {
			parts := strings.Split(fname, "_")
			if len(parts) >= 2 {
				prefix := parts[0] + "_"
				if existingEntries, err := os.ReadDir(manifestsDir); err == nil {
					for _, ee := range existingEntries {
						if !ee.IsDir() && strings.HasPrefix(ee.Name(), prefix) && strings.HasSuffix(ee.Name(), ".manifest") && ee.Name() != fname {
							_ = os.Remove(filepath.Join(manifestsDir, ee.Name()))
						}
					}
				}
			}
			outPath := filepath.Join(manifestsDir, fname)
			_ = os.WriteFile(outPath, fcontent, 0644)
			slog.Info("Saved manifest file", "path", outPath)
		}
	}

	dummyManifest := &manifest.Manifest{
		AppID:     opts.AppID,
		DepotKeys: make(map[uint32]string),
	}
	for _, d := range parsedLua.Depots {
		dummyManifest.DepotKeys[d.DepotID] = d.DecryptionKey
	}
	if len(parsedLua.Depots) > 0 {
		dummyManifest.DepotID = parsedLua.Depots[0].DepotID
		dummyManifest.DecryptionKey = parsedLua.Depots[0].DecryptionKey
	}

	opts.AuditTrace = parsedLua.AuditTrace
	return DownloadOrUpdateGame(ctx, dummyManifest, opts)
}
