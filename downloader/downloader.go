package downloader

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
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

// NewDownloaderHTTPClient returns an http.Client configured with dedicated transport pooling
// optimized for high-concurrency depot chunk downloads.
func NewDownloaderHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Transport: transport,
	}
}

// DefaultHTTPClient is the default http.Client used for downloader operations.
var DefaultHTTPClient = NewDownloaderHTTPClient()

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
	AppID            uint32
	TargetDir        string
	Platform         string // "win64", "linux", "all"
	Language         string // Preferred language code (default "english")
	IncludedDepots   []uint32
	ExcludedDepots   []uint32
	IncludeOptional  bool
	DepotMetadata    map[uint32]manifest.DepotMeta
	LuaPath          string
	HubcapAPIKey     string
	ManifestGIDs     map[uint32]string // Optional map of DepotID -> ManifestGID to acquire via GMRC
	GMRCEndpoint     string            // Optional custom GMRC endpoint URL
	WorkerLimit      int
	DryRun           bool
	HTTPClient       *http.Client
	PreviousManifest *manifest.Manifest // Optional previous manifest for delta patching

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
	Language             string // Preferred language code (default "english")
	IncludedDepots       []uint32
	ExcludedDepots       []uint32
	IncludeOptional      bool
	DepotMetadata        map[uint32]manifest.DepotMeta
	ManifestGIDs         map[uint32]string
	GMRCEndpoint         string
	DryRun               bool
	WorkerLimit          int
	AuditTrace           []string
	PreviousManifest     *manifest.Manifest
	ProgressCallback     func(current, total int, item string)
	ByteProgressCallback func(downloadedBytes, totalBytes int64, speedBps float64, etaSeconds int, pct float64, item string)
	HTTPClient           *http.Client
}

// Result holds the outcome summary of a download/update operation.
type Result struct {
	TotalFiles      int    `json:"total_files"`
	UpdatedFiles    int    `json:"updated_files"`
	DownloadedBytes uint64 `json:"downloaded_bytes"`
	NetworkBytes    uint64 `json:"network_bytes,omitempty"`
}

type cdnServerEntry struct {
	Host          string `json:"host"`
	VHost         string `json:"vhost"`
	Load          int    `json:"load"`
	WeightedLoad  int    `json:"weighted_load"`
	PriorityClass int    `json:"priority_class"`
	HTTPSSupport  string `json:"https_support"`
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

	// 1.5 Attempt GMRC direct SteamPipe CDN manifest acquisition if manifest GIDs were provided and manifests are missing
	if (parsedLua.ManifestFiles == nil || len(parsedLua.ManifestFiles) == 0) && len(req.ManifestGIDs) > 0 {
		for depotID, gid := range req.ManifestGIDs {
			if gid == "" {
				continue
			}
			reqCode, err := manifest.FetchGMRCRequestCode(ctx, gid, req.GMRCEndpoint)
			if err != nil {
				slog.Warn("Failed to resolve GMRC request code", "depotID", depotID, "manifestGID", gid, "error", err)
				parsedLua.AuditTrace = append(parsedLua.AuditTrace, fmt.Sprintf("GMRC Request Code (%d_%s): %v", depotID, gid, err))
				continue
			}
			mBytes, err := manifest.DownloadManifestFromSteamCDN(ctx, depotID, gid, reqCode, "")
			if err != nil {
				slog.Warn("Failed to download manifest from Steam CDN", "depotID", depotID, "manifestGID", gid, "error", err)
				parsedLua.AuditTrace = append(parsedLua.AuditTrace, fmt.Sprintf("Steam CDN Manifest Download (%d_%s): %v", depotID, gid, err))
				continue
			}
			if parsedLua.ManifestFiles == nil {
				parsedLua.ManifestFiles = make(map[string][]byte)
			}
			fname := fmt.Sprintf("%d_%s.manifest", depotID, gid)
			parsedLua.ManifestFiles[fname] = mBytes
			parsedLua.AuditTrace = append(parsedLua.AuditTrace, fmt.Sprintf("GMRC + Steam CDN: Successfully downloaded manifest %s (%d bytes)", fname, len(mBytes)))
		}
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
							oldPath := filepath.Join(manifestsDir, ee.Name())
							prevPath := filepath.Join(manifestsDir, strings.TrimSuffix(ee.Name(), ".manifest")+".manifest.previous")
							_ = os.Remove(prevPath)
							_ = os.Rename(oldPath, prevPath)
							slog.Info("Preserved previous manifest for delta patching", "previous", prevPath)
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

	prefLang := req.Language
	if prefLang == "" {
		prefLang = "english"
	}

	depotFilter := manifest.DepotFilter{
		Platform:        req.Platform,
		Language:        prefLang,
		IncludedDepots:  req.IncludedDepots,
		ExcludedDepots:  req.ExcludedDepots,
		IncludeOptional: req.IncludeOptional,
		DepotMetadata:   req.DepotMetadata,
	}

	// 4. Load binary manifest(s) from disk or memory filtered by target platform and language options
	var loadedManifest *manifest.Manifest
	var localLoadErr error
	if entries, readErr := os.ReadDir(manifestsDir); readErr == nil && len(entries) > 0 {
		loadedManifest, localLoadErr = manifest.LoadManifestFromDirWithFilter(absTargetDir, req.AppID, depotKeys, depotFilter)
		if localLoadErr != nil {
			slog.Debug("Local manifest filter/load result", "dir", manifestsDir, "error", localLoadErr)
		}
	}

	// Fallback to in-memory parsed manifests if disk manifests are absent or could not be loaded
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
				if !manifest.ShouldIncludeDepot(fileDepotID, files, depotFilter) {
					slog.Debug("Skipping remote depot manifest filtered out by options", "depotID", fileDepotID, "platform", req.Platform, "language", prefLang)
					continue
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

		// If strict language filter produced 0 files, retry with relaxed language filtering (Core Tenet 2)
		if len(allFiles) == 0 && depotFilter.Language != "" && depotFilter.Language != "all" {
			relaxedFilter := depotFilter
			relaxedFilter.Language = "all"
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
					if !manifest.ShouldIncludeDepot(fileDepotID, files, relaxedFilter) {
						continue
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
			var manifestFiles []string
			var otherFiles []string
			for _, e := range entries {
				if !e.IsDir() {
					if strings.HasSuffix(strings.ToLower(e.Name()), ".manifest") {
						manifestFiles = append(manifestFiles, e.Name())
					} else {
						otherFiles = append(otherFiles, e.Name())
					}
				}
			}
			if len(manifestFiles) > 0 {
				if localLoadErr != nil {
					trace = append(trace, fmt.Sprintf("Local [Manifests] Dir ('%s'): %d .manifest file(s) found (%s), but load failed: %v", manifestsDir, len(manifestFiles), strings.Join(manifestFiles, ", "), localLoadErr))
				} else {
					trace = append(trace, fmt.Sprintf("Local [Manifests] Dir ('%s'): %d .manifest file(s) found (%s)", manifestsDir, len(manifestFiles), strings.Join(manifestFiles, ", ")))
				}
			} else if len(otherFiles) > 0 {
				trace = append(trace, fmt.Sprintf("Local [Manifests] Dir ('%s'): 0 .manifest files found (%d non-manifest files present: %s)", manifestsDir, len(otherFiles), strings.Join(otherFiles, ", ")))
			} else {
				trace = append(trace, fmt.Sprintf("Local [Manifests] Dir ('%s'): directory is empty", manifestsDir))
			}
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
		cdnHosts = rankCDNEdges(ctx, httpClient, []string{"cache10-fra1.steamcontent.com", "cache4-fra1.steamcontent.com", "cache1-fra1.steamcontent.com"})
	}
	rankedHosts := cdnHosts

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
		AppID:            opts.AppID,
		TargetDir:        opts.TargetDir,
		Platform:         opts.Platform,
		Language:         opts.Language,
		IncludedDepots:   opts.IncludedDepots,
		ExcludedDepots:   opts.ExcludedDepots,
		IncludeOptional:  opts.IncludeOptional,
		DepotMetadata:    opts.DepotMetadata,
		LuaPath:          opts.LuaPath,
		HubcapAPIKey:     hubcapKey,
		ManifestGIDs:     opts.ManifestGIDs,
		GMRCEndpoint:     opts.GMRCEndpoint,
		WorkerLimit:      opts.WorkerLimit,
		DryRun:           opts.DryRun,
		HTTPClient:       opts.HTTPClient,
		PreviousManifest: opts.PreviousManifest,
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

	prefLang := opts.Language
	if prefLang == "" {
		prefLang = "english"
	}
	depotFilter := manifest.DepotFilter{
		Platform:        opts.Platform,
		Language:        prefLang,
		IncludedDepots:  opts.IncludedDepots,
		ExcludedDepots:  opts.ExcludedDepots,
		IncludeOptional: opts.IncludeOptional,
		DepotMetadata:   opts.DepotMetadata,
	}

	if len(m.Files) == 0 {
		loadedManifest, err := manifest.LoadManifestFromDirWithFilter(absTargetDir, opts.AppID, m.DepotKeys, depotFilter)
		if err == nil && loadedManifest != nil && len(loadedManifest.Files) > 0 {
			m.Files = loadedManifest.Files
			if m.DepotID == 0 {
				m.DepotID = loadedManifest.DepotID
			}
			if len(m.DepotKeys) == 0 && len(loadedManifest.DepotKeys) > 0 {
				m.DepotKeys = loadedManifest.DepotKeys
			}
			slog.Info("Loaded file list from binary manifest", "count", len(m.Files), "depotID", m.DepotID, "platform", opts.Platform, "language", prefLang)
		}
	} else {
		depotFiles := make(map[uint32][]manifest.ManifestFileEntry)
		for _, f := range m.Files {
			depotFiles[f.DepotID] = append(depotFiles[f.DepotID], f)
		}
		var filtered []manifest.ManifestFileEntry
		for dID, files := range depotFiles {
			if manifest.ShouldIncludeDepot(dID, files, depotFilter) {
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
		cdnHosts = rankCDNEdges(ctx, httpClient, []string{"cache10-fra1.steamcontent.com", "cache4-fra1.steamcontent.com", "cache1-fra1.steamcontent.com"})
	}
	rankedHosts := cdnHosts

	req := DownloadRequest{
		AppID:            opts.AppID,
		TargetDir:        absTargetDir,
		Platform:         opts.Platform,
		Language:         opts.Language,
		IncludedDepots:   opts.IncludedDepots,
		ExcludedDepots:   opts.ExcludedDepots,
		IncludeOptional:  opts.IncludeOptional,
		DepotMetadata:    opts.DepotMetadata,
		WorkerLimit:      opts.WorkerLimit,
		DryRun:           opts.DryRun,
		HTTPClient:       httpClient,
		PreviousManifest: opts.PreviousManifest,
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

func chunkKey(relPath string, chunk manifest.ChunkInfo) string {
	return fmt.Sprintf("%s:%s@%d", manifest.NormalizePath(relPath), strings.ToLower(chunk.ChunkID), chunk.Offset)
}

func verifyFileChunksOnDisk(filePath string, chunks []manifest.ChunkInfo) bool {
	return manifest.VerifyFileChunksOnDisk(filePath, chunks)
}

func isExecutableEntry(entry manifest.ManifestFileEntry) bool {
	if (entry.Flags & 0x01) != 0 {
		return true
	}
	p := strings.ToLower(manifest.NormalizePath(entry.Path))
	return strings.HasSuffix(p, ".sh") || strings.HasSuffix(p, ".x86_64") || strings.HasSuffix(p, ".x86") || strings.HasSuffix(p, ".bin")
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

	var downloadSucceeded bool
	if !req.DryRun {
		defer func() {
			if !downloadSucceeded {
				cp.Flush(checkpointFile)
			}
		}()
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

	// Calculate delta depot differences against previous manifests
	diffMap := make(map[string]manifest.FileChunkDiff)
	var deletedFiles []string

	if req.PreviousManifest != nil {
		diff := manifest.DiffDepotManifests(req.PreviousManifest, m)
		if diff != nil {
			for path, fd := range diff.ModifiedFiles {
				diffMap[manifest.NormalizePath(path)] = fd
			}
			deletedFiles = append(deletedFiles, diff.DeletedFiles...)
		}
	} else {
		// Group target files by depot ID to diff each depot against its previous manifest
		depotFiles := make(map[uint32][]manifest.ManifestFileEntry)
		for _, f := range m.Files {
			did := f.DepotID
			if did == 0 {
				did = m.DepotID
			}
			depotFiles[did] = append(depotFiles[did], f)
		}

		for did, dFiles := range depotFiles {
			prevM, err := manifest.LoadPreviousManifestFromDir(absTargetDir, did, m.DepotKeys)
			if err == nil && prevM != nil {
				targetDepotM := &manifest.Manifest{
					AppID:         m.AppID,
					DepotID:       did,
					DecryptionKey: m.GetDecryptionKey(did),
					DepotKeys:     m.DepotKeys,
					Files:         dFiles,
				}
				diff := manifest.DiffDepotManifests(prevM, targetDepotM)
				if diff != nil {
					for path, fd := range diff.ModifiedFiles {
						diffMap[manifest.NormalizePath(path)] = fd
					}
					deletedFiles = append(deletedFiles, diff.DeletedFiles...)
				}
			}
		}
	}

	if len(diffMap) > 0 {
		slog.Info("Discovered previous depot manifests for delta patching",
			"deltaFiles", len(diffMap),
			"deletedFiles", len(deletedFiles),
		)
	}

	// Purge deleted files if not in dry-run
	if !req.DryRun && len(deletedFiles) > 0 {
		activePaths := make(map[string]bool, len(m.Files))
		for _, f := range m.Files {
			activePaths[manifest.NormalizePath(f.Path)] = true
		}

		for _, delPath := range deletedFiles {
			normDel := manifest.NormalizePath(delPath)
			if normDel == "" || activePaths[normDel] {
				continue
			}
			absDel := filepath.Join(absTargetDir, normDel)
			if fi, err := os.Stat(absDel); err == nil && !fi.IsDir() {
				if err := os.Remove(absDel); err == nil {
					slog.Info("Removed deleted depot file", "path", normDel)
				}
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

	var lastTelemetry time.Time
	var telemetryMu sync.Mutex

	reportTelemetry := func(item string, force ...bool) {
		if req.OnProgress == nil {
			return
		}
		telemetryMu.Lock()
		now := time.Now()
		isForced := len(force) > 0 && force[0]
		if !isForced && now.Sub(lastTelemetry) < 100*time.Millisecond {
			telemetryMu.Unlock()
			return
		}
		lastTelemetry = now
		telemetryMu.Unlock()

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
			info, statErr := os.Stat(filePath)
			fileExists := statErr == nil && !info.IsDir()

			// Check if file exists and hash matches
			if fileExists {
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
					if !verifyFileChunksOnDisk(filePath, entry.Chunks) {
						slog.Debug("File chunk verification failed on disk, downloading missing chunks", "path", filePath)
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
					if err := util.PreallocateFile(f, int64(entry.Size)); err != nil {
						f.Close()
						return fmt.Errorf("preallocate empty file %s: %w", filePath, err)
					}
				}
				f.Close()
				if isExecutableEntry(entry) {
					_ = os.Chmod(filePath, 0755)
				}
				atomic.AddInt64(&downloadedBytes, int64(entry.Size))
				reportTelemetry(entry.Path)
			} else {
				fileDiff, isDelta := diffMap[relPath]
				canDelta := isDelta && fileExists && info.Size() > 0

				fileFlags := os.O_CREATE | os.O_RDWR
				f, err := os.OpenFile(filePath, fileFlags, 0644)
				if err != nil {
					return fmt.Errorf("open target file %s: %w", filePath, err)
				}

				if !canDelta && (!fileExists || info.Size() < int64(entry.Size)) {
					if err := util.PreallocateFile(f, int64(entry.Size)); err != nil {
						f.Close()
						return fmt.Errorf("preallocate target file %s: %w", filePath, err)
					}
				}

				unchangedChunkIDs := make(map[string]bool)
				if canDelta {
					for _, ch := range fileDiff.UnchangedChunks {
						if info.Size() >= int64(ch.Offset+uint64(ch.UncompressedSize)) {
							unchangedChunkIDs[ch.ChunkID] = true
						}
					}
				}

				var pendingChunks []manifest.ChunkInfo
				chunkBuf := make([]byte, 1024*1024)
				for _, chunk := range entry.Chunks {
					if req.CheckCancel != nil {
						if err := req.CheckCancel(); err != nil {
							f.Close()
							return err
						}
					}

					key := chunkKey(relPath, chunk)
					if cp.isCompleted(key) || unchangedChunkIDs[chunk.ChunkID] {
						slog.Debug("Chunk already present or completed in checkpoint, skipping network fetch", "chunkID", chunk.ChunkID, "path", filePath)
						atomic.AddInt64(&downloadedBytes, int64(chunk.UncompressedSize))
						reportTelemetry(entry.Path)
						cp.markCompleted(checkpointFile, key)
						continue
					}

					// In-place chunk repair: if file exists and chunk on disk already matches expected ChunkID, skip download
					if fileExists && info.Size() >= int64(chunk.Offset+uint64(chunk.UncompressedSize)) {
						if manifest.IsChunkValidOnDisk(f, chunk, chunkBuf) {
							slog.Debug("Chunk already valid on disk, skipping network fetch", "chunkID", chunk.ChunkID, "offset", chunk.Offset, "path", filePath)
							atomic.AddInt64(&downloadedBytes, int64(chunk.UncompressedSize))
							reportTelemetry(entry.Path)
							cp.markCompleted(checkpointFile, key)
							continue
						}
					}

					pendingChunks = append(pendingChunks, chunk)
				}

				if len(pendingChunks) > 0 {
					chunkWorkers := 4
					if chunkWorkers > len(pendingChunks) {
						chunkWorkers = len(pendingChunks)
					}
					cg, cgCtx := errgroup.WithContext(gCtx)
					cg.SetLimit(chunkWorkers)

					for _, ch := range pendingChunks {
						chunk := ch
						cg.Go(func() error {
							if req.CheckCancel != nil {
								if err := req.CheckCancel(); err != nil {
									return err
								}
							}

							key := chunkKey(relPath, chunk)
							depotID := entry.DepotID
							if depotID == 0 {
								depotID = m.DepotID
							}
							decKey := m.GetDecryptionKey(depotID)
							if err := downloadChunk(cgCtx, httpClient, rankedHosts, depotID, decKey, chunk, f); err != nil {
								return fmt.Errorf("download chunk for %s (depot %d): %w", filePath, depotID, err)
							}

							if req.Throttle != nil {
								_ = req.Throttle(int(chunk.UncompressedSize))
							}

							atomic.AddInt64(&downloadedBytes, int64(chunk.UncompressedSize))
							atomic.AddInt64(&networkTransferredBytes, int64(chunk.UncompressedSize))
							reportTelemetry(entry.Path)
							cp.markCompleted(checkpointFile, key)
							return nil
						})
					}

					if err := cg.Wait(); err != nil {
						f.Close()
						return err
					}
				}

				if err := f.Truncate(int64(entry.Size)); err != nil {
					f.Close()
					return fmt.Errorf("truncate final file %s: %w", filePath, err)
				}
				f.Close()

				if isExecutableEntry(entry) {
					_ = os.Chmod(filePath, 0755)
				}

				// Post-download integrity verification against SHA-1 if present
				if entry.SHA1 != "" {
					localHash, err := util.GetSHA1Hash(filePath)
					if err != nil || !strings.EqualFold(localHash, entry.SHA1) {
						slog.Warn("Downloaded file hash mismatch, falling back to full file download",
							"path", filePath, "actual", localHash, "expected", entry.SHA1)

						fFull, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
						if err != nil {
							return fmt.Errorf("reopen file for full download %s: %w", filePath, err)
						}
						if err := util.PreallocateFile(fFull, int64(entry.Size)); err != nil {
							fFull.Close()
							return fmt.Errorf("preallocate fallback file %s: %w", filePath, err)
						}

						chunkWorkers := 4
						if chunkWorkers > len(entry.Chunks) {
							chunkWorkers = len(entry.Chunks)
						}
						cgFull, cgFullCtx := errgroup.WithContext(gCtx)
						cgFull.SetLimit(chunkWorkers)

						for _, ch := range entry.Chunks {
							chunk := ch
							cgFull.Go(func() error {
								depotID := entry.DepotID
								if depotID == 0 {
									depotID = m.DepotID
								}
								decKey := m.GetDecryptionKey(depotID)
								if err := downloadChunk(cgFullCtx, httpClient, rankedHosts, depotID, decKey, chunk, fFull); err != nil {
									return fmt.Errorf("download chunk during fallback for %s: %w", filePath, err)
								}
								atomic.AddInt64(&networkTransferredBytes, int64(chunk.UncompressedSize))
								cp.markCompleted(checkpointFile, chunkKey(relPath, chunk))
								return nil
							})
						}

						if err := cgFull.Wait(); err != nil {
							fFull.Close()
							return err
						}
						fFull.Close()

						if isExecutableEntry(entry) {
							_ = os.Chmod(filePath, 0755)
						}

						fallbackHash, err := util.GetSHA1Hash(filePath)
						if err != nil || !strings.EqualFold(fallbackHash, entry.SHA1) {
							return fmt.Errorf("full redownload hash mismatch for %s: got %s, want %s", filePath, fallbackHash, entry.SHA1)
						}
					}
				}
			}

			slog.Debug("Successfully updated file", "path", filePath)
			atomic.AddInt32(&processedUpdated, 1)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		slog.Error("Direct game download aborted due to worker error",
			"appID", m.AppID,
			"depotID", m.DepotID,
			"error", err,
		)
		return nil, fmt.Errorf("download execution failed: %w", err)
	}

	result.UpdatedFiles = int(atomic.LoadInt32(&processedUpdated))
	result.DownloadedBytes = uint64(atomic.LoadInt64(&downloadedBytes))
	result.NetworkBytes = uint64(atomic.LoadInt64(&networkTransferredBytes))
	reportTelemetry("Completed", true)

	// Remove checkpoint file on successful completion
	if !req.DryRun {
		downloadSucceeded = true
		_ = os.Remove(checkpointFile)
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

	var validServers []cdnServerEntry
	for _, s := range payload.Response.Servers {
		h := strings.TrimSpace(s.Host)
		if h == "" {
			h = strings.TrimSpace(s.VHost)
		}
		if h == "" {
			continue
		}
		if strings.EqualFold(s.HTTPSSupport, "unsupported") || strings.EqualFold(s.HTTPSSupport, "none") {
			continue
		}
		s.Host = h
		validServers = append(validServers, s)
	}

	if len(validServers) == 0 {
		return nil, fmt.Errorf("no cdn hosts returned by SteamPipe service")
	}

	// Rank servers by Valve's official metrics:
	// 1. Primary sort: PriorityClass DESC (Valve's preferred tier)
	// 2. Secondary sort: WeightedLoad ASC (least congested node)
	sort.SliceStable(validServers, func(i, j int) bool {
		if validServers[i].PriorityClass != validServers[j].PriorityClass {
			return validServers[i].PriorityClass > validServers[j].PriorityClass
		}
		return validServers[i].WeightedLoad < validServers[j].WeightedLoad
	})

	limit := 5
	if len(validServers) < limit {
		limit = len(validServers)
	}

	topHosts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		topHosts = append(topHosts, validServers[i].Host)
	}

	return topHosts, nil
}

func downloadChunk(ctx context.Context, client *http.Client, cdnHosts []string, depotID uint32, decKey string, chunk manifest.ChunkInfo, dst io.WriterAt) error {
	if client == nil {
		client = DefaultHTTPClient
	}
	if len(cdnHosts) == 0 {
		cdnHosts = []string{"cache10-fra1.steamcontent.com", "cache4-fra1.steamcontent.com", "cache1-fra1.steamcontent.com"}
	}

	chunkID := strings.ToLower(chunk.ChunkID)
	startIdx := 0
	if len(cdnHosts) > 1 && len(chunkID) >= 4 {
		if n, err := strconv.ParseUint(chunkID[:4], 16, 32); err == nil {
			startIdx = int(n) % len(cdnHosts)
		}
	}

	const maxAttempts = 3
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if attempt > 0 {
			jitter := time.Duration(10+int(chunkID[0])%50) * time.Millisecond
			backoff := time.Duration(150*(1<<attempt))*time.Millisecond + jitter
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		for i := 0; i < len(cdnHosts); i++ {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			host := cdnHosts[(startIdx+i+attempt)%len(cdnHosts)]
			chunkURL := fmt.Sprintf("https://%s/depot/%d/chunk/%s", host, depotID, chunkID)

			reqCtx, reqCancel := context.WithTimeout(ctx, 45*time.Second)
			req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, chunkURL, nil)
			if err != nil {
				reqCancel()
				lastErr = err
				continue
			}
			req.Header.Set("User-Agent", "Valve/Steam HTTP Client 1.0")

			resp, err := client.Do(req)
			if err != nil {
				reqCancel()
				if ctx.Err() != nil {
					return ctx.Err()
				}
				lastErr = fmt.Errorf("fetch from %s: %w", host, err)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				_, _ = io.CopyN(io.Discard, resp.Body, 4096)
				_ = resp.Body.Close()
				reqCancel()
				lastErr = fmt.Errorf("HTTP status %d from %s", resp.StatusCode, host)
				continue
			}

			rawChunkData, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			reqCancel()
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				lastErr = fmt.Errorf("read chunk body from %s: %w", host, err)
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

		slog.Warn("Retrying chunk download after host round failure",
			"chunkID", chunk.ChunkID,
			"depotID", depotID,
			"attempt", attempt+1,
			"maxAttempts", maxAttempts,
			"lastErr", lastErr,
		)
	}

	slog.Error("Failed to download chunk after all retry attempts",
		"chunkID", chunk.ChunkID,
		"depotID", depotID,
		"hosts", len(cdnHosts),
		"error", lastErr,
	)
	return fmt.Errorf("download chunk %s (depot %d) failed after %d attempts across %d CDN hosts: %w",
		chunk.ChunkID, depotID, maxAttempts, len(cdnHosts), lastErr)
}

func decompressChunkData(data []byte, expectedSize uint32) ([]byte, error) {
	return util.DecompressChunkSlice(data, expectedSize)
}

// Checkpoint tracks completed chunk IDs for resume support.
type Checkpoint struct {
	AppID           uint32          `json:"app_id"`
	DepotID         uint32          `json:"depot_id"`
	CompletedChunks map[string]bool `json:"completed_chunks"`
	lastSaved       time.Time
	dirty           bool
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

func (cp *Checkpoint) isCompleted(key string) bool {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.CompletedChunks[key]
}

func (cp *Checkpoint) markCompleted(path string, key string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.CompletedChunks[key] {
		return
	}
	cp.CompletedChunks[key] = true
	cp.dirty = true

	// Throttle disk writes to at most once per second
	if time.Since(cp.lastSaved) >= 1*time.Second {
		cp.saveLocked(path)
	}
}

func (cp *Checkpoint) Flush(path string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.dirty {
		cp.saveLocked(path)
	}
}

func (cp *Checkpoint) saveLocked(path string) {
	data, err := json.Marshal(cp)
	if err == nil {
		if err := util.WriteFileAtomic(path, data, 0644); err == nil {
			cp.lastSaved = time.Now()
			cp.dirty = false
		}
	}
}
