package downloader

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/util"
	"golang.org/x/sync/errgroup"
)

// DefaultHTTPClient is the default http.Client used for downloader operations.
var DefaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

// ProgressUpdate carries unified quantitative telemetry for active downloads.
type ProgressUpdate struct {
	DownloadedBytes int64   `json:"downloaded_bytes"`
	TotalBytes      int64   `json:"total_bytes"`
	CurrentItems    int     `json:"current_items"`
	TotalItems      int     `json:"total_items"`
	SpeedBps        float64 `json:"speed_bps"`
	ETASeconds      int     `json:"eta_seconds"`
	Percent         float64 `json:"percent"`
	Item            string  `json:"item"`
}

// DownloadRequest encapsulates parameters and telemetry hooks for downloading or updating game depots.
type DownloadRequest struct {
	AppID        uint32
	TargetDir    string
	Platform     string // "win64", "linux", "all"
	LuaPath      string
	HubcapAPIKey string
	WorkerLimit  int
	DryRun       bool
	HTTPClient   *http.Client

	// Telemetry & lifecycle control hooks
	OnProgress  func(update ProgressUpdate)
	OnStage     func(stage, details string)
	CheckCancel func() error
	Throttle    func(bytes int) error
}

// DownloadOptions holds options for downloading or updating game depots directly (for compatibility).
type DownloadOptions struct {
	TargetDir            string
	AppID                uint32
	LuaPath              string
	Platform             string // "win64", "linux", "all"
	DryRun               bool
	WorkerLimit          int
	AuditTrace           []string
	ProgressCallback     func(current, total int, item string)
	ByteProgressCallback func(downloadedBytes, totalBytes int64, speedBps float64, etaSeconds int, pct float64, item string)
	HTTPClient           *http.Client
}

// Result holds the outcome summary of a download/update operation.
type Result struct {
	TotalFiles      int    `json:"total_files"`
	UpdatedFiles    int    `json:"updated_files"`
	DownloadedBytes uint64 `json:"downloaded_bytes"`
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

// Download is the primary deep entry point for depot acquisition.
// It resolves keys across the provider chain, persists manifests to [Manifests]/,
// loads the file table, discovers and ranks CDN edge nodes, and streams file chunks.
func Download(ctx context.Context, req DownloadRequest) (*Result, error) {
	if req.AppID == 0 {
		return nil, fmt.Errorf("invalid Steam AppID 0")
	}

	targetDir := req.TargetDir
	if targetDir == "" {
		targetDir = "."
	}

	absTargetDir := util.CanonicalPath(targetDir)
	if absTargetDir == "" {
		return nil, fmt.Errorf("resolve target directory: empty path")
	}

	httpClient := req.HTTPClient
	if httpClient == nil {
		httpClient = DefaultHTTPClient
	}
	ctx = manifest.WithHTTPClient(ctx, httpClient)

	if req.OnStage != nil {
		req.OnStage("Resolving Keys & Manifests", fmt.Sprintf("AppID: %d", req.AppID))
	}

	// 1. Resolve keys and binary manifests across the multi-tier provider chain
	parsedLua, err := manifest.ResolveKeys(ctx, req.AppID, req.LuaPath, req.HubcapAPIKey)
	if err != nil {
		return nil, fmt.Errorf("resolve keys and manifests for AppID %d: %w", req.AppID, err)
	}

	req.AppID = parsedLua.AppID
	manifestsDir := filepath.Join(absTargetDir, "[Manifests]")

	// 2. Persist returned .manifest files to disk
	if !req.DryRun && len(parsedLua.ManifestFiles) > 0 {
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

	// 3. Extract depot keys
	depotKeys := make(map[uint32]string)
	for _, d := range parsedLua.Depots {
		depotKeys[d.DepotID] = d.DecryptionKey
	}

	// 4. Load binary manifest(s) from disk or memory filtered by target platform
	var loadedManifest *manifest.Manifest
	if entries, readErr := os.ReadDir(manifestsDir); readErr == nil && len(entries) > 0 {
		loadedManifest, _ = manifest.LoadManifestFromDirWithPlatform(absTargetDir, req.AppID, depotKeys, req.Platform)
	}

	// Fallback to in-memory parsed manifests if disk manifests are absent (e.g. during DryRun)
	if (loadedManifest == nil || len(loadedManifest.Files) == 0) && len(parsedLua.ManifestFiles) > 0 {
		var allFiles []manifest.ManifestFileEntry
		var firstDepotID uint32
		for filename, data := range parsedLua.ManifestFiles {
			parts := strings.Split(filename, "_")
			var fileDepotID uint32
			if len(parts) >= 1 {
				if did, err := strconv.ParseUint(parts[0], 10, 32); err == nil {
					fileDepotID = uint32(did)
				}
			}
			hexKey := depotKeys[fileDepotID]
			files, err := manifest.ParseBinaryManifestWithKey(data, hexKey)
			if err == nil && len(files) > 0 {
				if req.Platform != "" && req.Platform != "all" {
					cat := manifest.ClassifyDepot(files)
					if !manifest.MatchesPlatform(cat, req.Platform) {
						slog.Debug("Skipping remote depot manifest not matching platform", "depotID", fileDepotID, "category", cat, "targetPlatform", req.Platform)
						continue
					}
				}
				if firstDepotID == 0 {
					firstDepotID = fileDepotID
				}
				for idx := range files {
					files[idx].DepotID = fileDepotID
				}
				allFiles = append(allFiles, files...)
			}
		}
		if len(allFiles) > 0 {
			loadedManifest = &manifest.Manifest{
				AppID:     req.AppID,
				DepotID:   firstDepotID,
				DepotKeys: depotKeys,
				Files:     allFiles,
			}
		}
	}


	if loadedManifest == nil || len(loadedManifest.Files) == 0 {
		trace := parsedLua.AuditTrace
		if entries, err := os.ReadDir(manifestsDir); err == nil {
			trace = append(trace, fmt.Sprintf("Local [Manifests] Dir ('%s'): %d files found, 0 valid .manifest file entries", manifestsDir, len(entries)))
		} else {
			trace = append(trace, fmt.Sprintf("Local [Manifests] Dir ('%s'): Directory missing", manifestsDir))
		}

		traceStr := ""
		if len(trace) > 0 {
			traceStr = "\n\n🔍 Key & Manifest Provider Resolution Audit Trace:\n  • " + strings.Join(trace, "\n  • ")
		}

		return nil, fmt.Errorf("no depot binary manifest (.manifest) files found in '[Manifests]/' or resolved for AppID %d%s", req.AppID, traceStr)
	}

	// 5. Validate that all file paths in manifest are decrypted
	for _, f := range loadedManifest.Files {
		if util.IsEncryptedBase64Path(f.Path) {
			return nil, fmt.Errorf("manifest contains encrypted filenames for depot %d and no valid depot decryption key was found to decrypt them. Aborting download to prevent corrupting target directory", f.DepotID)
		}
	}

	// 6. Discover and rank SteamPipe CDN edge servers
	cdnHosts, err := fetchCDNServers(ctx, httpClient)
	if err != nil {
		slog.Warn("Failed to fetch CDN servers from SteamPipe API, using fallbacks", "error", err)
		cdnHosts = []string{"cache10-fra1.steamcontent.com", "cache4-fra1.steamcontent.com", "cache1-fra1.steamcontent.com"}
	}
	rankedHosts := rankCDNEdges(ctx, httpClient, cdnHosts)

	if req.OnStage != nil {
		req.OnStage("Streaming Depot Chunks", fmt.Sprintf("Downloading %d files to %s", len(loadedManifest.Files), targetDir))
	}

	// 7. Stream file chunks
	return streamDepotFiles(ctx, loadedManifest, absTargetDir, req, httpClient, rankedHosts)
}

// DownloadGame orchestrates key resolution, manifest persistence, and depot chunk downloading for an AppID (convenience wrapper).
func DownloadGame(ctx context.Context, cfg *config.Config, opts DownloadOptions) (*Result, error) {
	hubcapKey := ""
	if cfg != nil {
		hubcapKey = cfg.HubcapAPIKey
	}

	req := DownloadRequest{
		AppID:        opts.AppID,
		TargetDir:    opts.TargetDir,
		Platform:     opts.Platform,
		LuaPath:      opts.LuaPath,
		HubcapAPIKey: hubcapKey,
		WorkerLimit:  opts.WorkerLimit,
		DryRun:       opts.DryRun,
		HTTPClient:   opts.HTTPClient,
	}

	if opts.ByteProgressCallback != nil || opts.ProgressCallback != nil {
		req.OnProgress = func(p ProgressUpdate) {
			if opts.ByteProgressCallback != nil {
				opts.ByteProgressCallback(p.DownloadedBytes, p.TotalBytes, p.SpeedBps, p.ETASeconds, p.Percent, p.Item)
			}
			if opts.ProgressCallback != nil {
				opts.ProgressCallback(p.CurrentItems, p.TotalItems, p.Item)
			}
		}
	}

	return Download(ctx, req)
}

// DownloadOrUpdateGame compares local files in targetDir against the manifest and downloads missing or updated file chunks.
// Retained for low-level manifest testing and direct execution.
func DownloadOrUpdateGame(ctx context.Context, m *manifest.Manifest, opts DownloadOptions) (*Result, error) {
	if m == nil {
		return nil, fmt.Errorf("manifest cannot be nil")
	}

	manifestCopy := *m
	m = &manifestCopy

	targetDir := opts.TargetDir

	if targetDir == "" {
		targetDir = "."
	}

	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("resolve target directory: %w", err)
	}

	if len(m.Files) == 0 {
		loadedManifest, err := manifest.LoadManifestFromDirWithPlatform(absTargetDir, opts.AppID, m.DepotKeys, opts.Platform)
		if err == nil && loadedManifest != nil && len(loadedManifest.Files) > 0 {
			m.Files = loadedManifest.Files
			if m.DepotID == 0 {
				m.DepotID = loadedManifest.DepotID
			}
			if len(m.DepotKeys) == 0 && len(loadedManifest.DepotKeys) > 0 {
				m.DepotKeys = loadedManifest.DepotKeys
			}
			slog.Info("Loaded file list from binary manifest", "count", len(m.Files), "depotID", m.DepotID, "platform", opts.Platform)
		}
	} else if opts.Platform != "" && opts.Platform != "all" {
		depotFiles := make(map[uint32][]manifest.ManifestFileEntry)
		for _, f := range m.Files {
			depotFiles[f.DepotID] = append(depotFiles[f.DepotID], f)
		}
		var filtered []manifest.ManifestFileEntry
		for _, files := range depotFiles {
			cat := manifest.ClassifyDepot(files)
			if manifest.MatchesPlatform(cat, opts.Platform) {
				filtered = append(filtered, files...)
			}
		}
		if len(filtered) > 0 {
			m.Files = filtered
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

	for _, f := range m.Files {
		if util.IsEncryptedBase64Path(f.Path) {
			return nil, fmt.Errorf("manifest contains encrypted filenames for depot %d and no valid depot decryption key was found to decrypt them. Aborting download to prevent corrupting target directory", f.DepotID)
		}
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = DefaultHTTPClient
	}

	cdnHosts, err := fetchCDNServers(ctx, httpClient)
	if err != nil {
		slog.Warn("Failed to fetch CDN servers from SteamPipe API, using fallbacks", "error", err)
		cdnHosts = []string{"cache10-fra1.steamcontent.com", "cache4-fra1.steamcontent.com", "cache1-fra1.steamcontent.com"}
	}
	rankedHosts := rankCDNEdges(ctx, httpClient, cdnHosts)

	req := DownloadRequest{
		AppID:       opts.AppID,
		TargetDir:   absTargetDir,
		Platform:    opts.Platform,
		WorkerLimit: opts.WorkerLimit,
		DryRun:      opts.DryRun,
		HTTPClient:  httpClient,
	}

	if opts.ByteProgressCallback != nil || opts.ProgressCallback != nil {
		req.OnProgress = func(p ProgressUpdate) {
			if opts.ByteProgressCallback != nil {
				opts.ByteProgressCallback(p.DownloadedBytes, p.TotalBytes, p.SpeedBps, p.ETASeconds, p.Percent, p.Item)
			}
			if opts.ProgressCallback != nil {
				opts.ProgressCallback(p.CurrentItems, p.TotalItems, p.Item)
			}
		}
	}

	return streamDepotFiles(ctx, m, absTargetDir, req, httpClient, rankedHosts)
}

func isDirectoryEntry(entry manifest.ManifestFileEntry) bool {
	return (entry.Flags&0x40) != 0 || strings.HasSuffix(entry.Path, "/") || strings.HasSuffix(entry.Path, "\\")
}

func streamDepotFiles(ctx context.Context, m *manifest.Manifest, absTargetDir string, req DownloadRequest, httpClient *http.Client, rankedHosts []string) (*Result, error) {
	workerLimit := req.WorkerLimit
	if workerLimit <= 0 {
		workerLimit = 10
	}

	slog.Info("Starting direct game download/update",
		"appID", m.AppID,
		"depotID", m.DepotID,
		"files", len(m.Files),
		"targetDir", absTargetDir,
		"cdnHosts", len(rankedHosts),
		"dryRun", req.DryRun,
	)

	result := &Result{
		TotalFiles: len(m.Files),
	}

	manifestsDir := filepath.Join(absTargetDir, "[Manifests]")
	checkpointFile := filepath.Join(manifestsDir, ".gamux-checkpoint.json")
	cp := loadCheckpoint(checkpointFile)

	if !req.DryRun {
		if err := os.MkdirAll(absTargetDir, 0755); err != nil {
			return nil, fmt.Errorf("create target directory %s: %w", absTargetDir, err)
		}
		_ = os.MkdirAll(manifestsDir, 0755)

		// Pre-create all directories sequentially on main thread
		for _, entry := range m.Files {
			relPath := manifest.NormalizePath(entry.Path)
			if relPath == "" {
				continue
			}
			p := filepath.Join(absTargetDir, relPath)
			if isDirectoryEntry(entry) {
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

	var totalDepotBytes int64
	for _, f := range m.Files {
		if f.Size > 0 && !isDirectoryEntry(f) {
			totalDepotBytes += int64(f.Size)
		}
	}

	var downloadedBytes int64
	var networkTransferredBytes int64
	var processedCount int32
	var processedUpdated int32
	totalFiles := len(m.Files)
	startTime := time.Now()

	reportTelemetry := func(item string) {
		if req.OnProgress == nil {
			return
		}
		curr := atomic.LoadInt64(&downloadedBytes)
		netBytes := atomic.LoadInt64(&networkTransferredBytes)
		itemsDone := int(atomic.LoadInt32(&processedCount))
		elapsed := time.Since(startTime).Seconds()
		speed := float64(0)
		eta := 0
		if elapsed > 0.5 {
			speed = float64(netBytes) / elapsed
			if speed > 0 && totalDepotBytes > curr {
				eta = int(float64(totalDepotBytes-curr) / speed)
			}
		}
		pct := float64(0)
		if totalDepotBytes > 0 {
			pct = float64(curr) / float64(totalDepotBytes) * 100.0
			if pct > 100 {
				pct = 100
			}
		}

		req.OnProgress(ProgressUpdate{
			DownloadedBytes: curr,
			TotalBytes:      totalDepotBytes,
			CurrentItems:    itemsDone,
			TotalItems:      totalFiles,
			SpeedBps:        speed,
			ETASeconds:      eta,
			Percent:         pct,
			Item:            item,
		})
	}

	for _, fileEntry := range m.Files {
		entry := fileEntry
		relPath := manifest.NormalizePath(entry.Path)
		if relPath == "" && isDirectoryEntry(entry) {
			continue
		}
		filePath := filepath.Join(absTargetDir, relPath)

		g.Go(func() error {
			if req.CheckCancel != nil {
				if err := req.CheckCancel(); err != nil {
					return err
				}
			}

			atomic.AddInt32(&processedCount, 1)
			reportTelemetry(entry.Path)

			// Skip directory entries in download loop
			if isDirectoryEntry(entry) {
				return nil
			}

			needsDownload := false

			// Check if file exists and hash matches
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				if uint64(info.Size()) != entry.Size {
					slog.Debug("File size mismatch, redownloading", "path", filePath, "actual", info.Size(), "expected", entry.Size)
					needsDownload = true
				} else if entry.SHA1 != "" {
					localHash, err := util.GetSHA1Hash(filePath)
					if err != nil || !strings.EqualFold(localHash, entry.SHA1) {
						slog.Debug("File hash mismatch, redownloading", "path", filePath, "actual", localHash, "expected", entry.SHA1)
						needsDownload = true
					}
				} else if len(entry.Chunks) > 0 {
					allChunksDone := true
					for _, ch := range entry.Chunks {
						if !cp.isCompleted(ch.ChunkID) {
							allChunksDone = false
							break
						}
					}
					if !allChunksDone {
						slog.Debug("File chunks incomplete in checkpoint, downloading missing chunks", "path", filePath)
						needsDownload = true
					}
				}
			} else {
				slog.Debug("New file, needs download", "path", filePath)
				needsDownload = true
			}

			if !needsDownload {
				slog.Debug("File is up-to-date", "path", filePath)
				atomic.AddInt64(&downloadedBytes, int64(entry.Size))
				reportTelemetry(entry.Path)
				return nil
			}

			if req.DryRun {
				slog.Debug("[DRY RUN] Would download/update file", "path", filePath, "size", entry.Size)
				atomic.AddInt32(&processedUpdated, 1)
				atomic.AddInt64(&downloadedBytes, int64(entry.Size))
				reportTelemetry(entry.Path)
				return nil
			}

			parentDir := filepath.Dir(filePath)
			if fi, err := os.Stat(parentDir); err == nil && !fi.IsDir() {
				_ = os.Remove(parentDir)
			}
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("create directory for %s: %w", filePath, err)
			}

			// If a directory exists at filePath by mistake, remove it
			if fi, err := os.Stat(filePath); err == nil && fi.IsDir() {
				_ = os.RemoveAll(filePath)
			}

			// Download chunks or file payload
			if len(entry.Chunks) == 0 {
				f, err := os.Create(filePath)
				if err != nil {
					return fmt.Errorf("create file %s: %w", filePath, err)
				}
				if entry.Size > 0 {
					_ = f.Truncate(int64(entry.Size))
				}
				f.Close()
				atomic.AddInt64(&downloadedBytes, int64(entry.Size))
				reportTelemetry(entry.Path)
			} else {
				fileFlags := os.O_CREATE | os.O_RDWR
				f, err := os.OpenFile(filePath, fileFlags, 0644)
				if err != nil {
					return fmt.Errorf("open target file %s: %w", filePath, err)
				}
				_ = f.Truncate(int64(entry.Size))

				for _, chunk := range entry.Chunks {
					if req.CheckCancel != nil {
						if err := req.CheckCancel(); err != nil {
							f.Close()
							return err
						}
					}

					if cp.isCompleted(chunk.ChunkID) {
						slog.Debug("Chunk already completed in checkpoint, skipping", "chunkID", chunk.ChunkID)
						atomic.AddInt64(&downloadedBytes, int64(chunk.UncompressedSize))
						reportTelemetry(entry.Path)
						continue
					}

					depotID := entry.DepotID
					if depotID == 0 {
						depotID = m.DepotID
					}
					decKey := m.GetDecryptionKey(depotID)
					if err := downloadChunk(gCtx, httpClient, rankedHosts, depotID, decKey, chunk, f); err != nil {
						f.Close()
						return fmt.Errorf("download chunk for %s (depot %d): %w", filePath, depotID, err)
					}

					if req.Throttle != nil {
						_ = req.Throttle(int(chunk.UncompressedSize))
					}

					atomic.AddInt64(&downloadedBytes, int64(chunk.UncompressedSize))
					atomic.AddInt64(&networkTransferredBytes, int64(chunk.UncompressedSize))
					reportTelemetry(entry.Path)
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
		return nil, fmt.Errorf("download execution failed: %w", err)
	}

	result.UpdatedFiles = int(atomic.LoadInt32(&processedUpdated))
	result.DownloadedBytes = uint64(atomic.LoadInt64(&downloadedBytes))

	// Remove checkpoint file and sweep for any remaining VSZa compressed files on successful completion
	if !req.DryRun {
		_ = os.Remove(checkpointFile)
		if count, err := util.DecompressVSZaInDir(absTargetDir); err == nil && count > 0 {
			slog.Info("Post-download VSZa file decompression completed", "count", count)
		}
	}

	slog.Debug("Direct game download/update completed",
		"totalFiles", result.TotalFiles,
		"updatedFiles", result.UpdatedFiles,
		"bytes", result.DownloadedBytes,
	)

	return result, nil
}

// rankCDNEdges measures HTTP round-trip latency to candidate edge hosts concurrently and returns hosts ordered by speed.
func rankCDNEdges(ctx context.Context, client *http.Client, hosts []string) []string {
	if client == nil {
		client = DefaultHTTPClient
	}
	if len(hosts) <= 1 {
		return hosts
	}

	type hostLatency struct {
		host string
		lat  time.Duration
	}

	latCh := make(chan hostLatency, len(hosts))
	timeoutCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	for _, h := range hosts {
		host := h
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			req, err := http.NewRequestWithContext(timeoutCtx, http.MethodHead, "https://"+host, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", "gamux/1.0")
			resp, err := client.Do(req)
			if err == nil {
				_ = resp.Body.Close()
				latCh <- hostLatency{host: host, lat: time.Since(start)}
			}
		}()
	}

	wg.Wait()
	close(latCh)

	var responsive []hostLatency
	for hl := range latCh {
		responsive = append(responsive, hl)
	}

	if len(responsive) == 0 {
		return hosts
	}

	sort.Slice(responsive, func(i, j int) bool {
		return responsive[i].lat < responsive[j].lat
	})

	ranked := make([]string, 0, len(hosts))
	seen := make(map[string]bool)
	for _, r := range responsive {
		ranked = append(ranked, r.host)
		seen[r.host] = true
	}
	for _, h := range hosts {
		if !seen[h] {
			ranked = append(ranked, h)
		}
	}
	return ranked
}

// fetchCDNServers discovers active SteamPipe CDN edge servers via Valve's directory Web API.
func fetchCDNServers(ctx context.Context, client *http.Client) ([]string, error) {
	if client == nil {
		client = DefaultHTTPClient
	}
	url := "https://api.steampowered.com/IContentServerDirectoryService/GetServersForSteamPipe/v1/?cell_id=0"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create cdn query request: %w", err)
	}
	req.Header.Set("User-Agent", "gamux/1.0")

	resp, err := client.Do(req)
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

	return rankCDNEdges(ctx, client, hosts), nil
}

func downloadChunk(ctx context.Context, client *http.Client, cdnHosts []string, depotID uint32, decKey string, chunk manifest.ChunkInfo, dst io.WriterAt) error {
	if client == nil {
		client = DefaultHTTPClient
	}
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

		resp, err := client.Do(req)
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
		payloadData, err := manifest.DecryptChunkPayload(rawChunkData, decKey)
		if err != nil {
			lastErr = fmt.Errorf("decrypt chunk %s: %w", chunk.ChunkID, err)
			continue
		}

		// 2. Decompress chunk payload
		decompressed, err := decompressChunkData(payloadData, chunk.UncompressedSize)
		if err != nil {
			lastErr = fmt.Errorf("decompress chunk %s: %w", chunk.ChunkID, err)
			continue
		}

		// 3. Verify chunk integrity against chunk.ChunkID SHA-1 (when 40-hex SHA1 hash is provided)
		if len(chunk.ChunkID) == 40 {
			chunkSum := sha1.Sum(decompressed)
			chunkHashHex := hex.EncodeToString(chunkSum[:])
			if !strings.EqualFold(chunkHashHex, chunk.ChunkID) {
				lastErr = fmt.Errorf("chunk %s checksum mismatch: got %s", chunk.ChunkID, chunkHashHex)
				continue
			}
		}

		// 4. Write chunk payload at expected offset
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
		tmpPath := path + ".tmp"
		if err := os.WriteFile(tmpPath, data, 0644); err == nil {
			_ = os.Rename(tmpPath, path)
		}
	}
}
