package downloader

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/util"
)

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
	Branch           string            // Optional target branch name (e.g. "public", "beta")
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
	Branch               string
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

// DownloadError wraps a root cause with the audit trace and AppID for diagnostics.
// Callers can inspect the Cause to determine the specific failure reason (e.g.
// manifest.PlatformMismatchError), while the audit trace provides detailed
// provider resolution context for debugging.
type DownloadError struct {
	AppID      uint32
	AuditTrace []string
	Cause      error
}

func (e *DownloadError) Error() string {
	return e.Cause.Error()
}

func (e *DownloadError) Unwrap() error {
	return e.Cause
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

	// 1.5 Target branch manifest acquisition:
	// If specific ManifestGIDs were provided (e.g. for a selected branch), ensure each depot acquires
	// and uses that exact target manifest GID.
	if len(req.ManifestGIDs) > 0 {
		if parsedLua.ManifestFiles == nil {
			parsedLua.ManifestFiles = make(map[string][]byte)
		}
		for depotID, gid := range req.ManifestGIDs {
			if gid == "" {
				continue
			}
			targetFname := fmt.Sprintf("%d_%s.manifest", depotID, gid)

			// Remove any non-matching manifests for this depot from parsedLua.ManifestFiles
			depotPrefix := fmt.Sprintf("%d_", depotID)
			for existingFname := range parsedLua.ManifestFiles {
				if strings.HasPrefix(existingFname, depotPrefix) && existingFname != targetFname {
					delete(parsedLua.ManifestFiles, existingFname)
				}
			}

			// If target manifest is already in parsedLua, proceed
			if _, exists := parsedLua.ManifestFiles[targetFname]; exists {
				continue
			}

			// Check if target manifest is already present on disk in [Manifests]
			diskPath := filepath.Join(manifestsDir, targetFname)
			if content, err := os.ReadFile(diskPath); err == nil && len(content) > 0 {
				parsedLua.ManifestFiles[targetFname] = content
				parsedLua.AuditTrace = append(parsedLua.AuditTrace, fmt.Sprintf("Local Disk: Found target branch manifest %s (%d bytes)", targetFname, len(content)))
				continue
			}

			// Download via GMRC + Steam CDN
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
			parsedLua.ManifestFiles[targetFname] = mBytes
			parsedLua.AuditTrace = append(parsedLua.AuditTrace, fmt.Sprintf("GMRC + Steam CDN: Successfully downloaded manifest %s (%d bytes)", targetFname, len(mBytes)))
		}
	}

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

	// 3. Extract depot keys and persist .lua file to [Manifests]
	if len(parsedLua.Depots) > 0 {
		luaPath := filepath.Join(manifestsDir, fmt.Sprintf("%d.lua", req.AppID))
		if err := manifest.SaveLuaFile(luaPath, req.AppID, parsedLua.Depots); err == nil {
			slog.Info("Saved depot keys lua file", "path", luaPath, "depots", len(parsedLua.Depots))
		}
	}

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
		ManifestGIDs:    req.ManifestGIDs,
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
		candidateDepots := make(map[uint32][]manifest.ManifestFileEntry)
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
				for idx := range files {
					files[idx].DepotID = fileDepotID
				}
				candidateDepots[fileDepotID] = files
			}
		}

		// If strict language filter produced 0 files, retry with relaxed language filtering (Core Tenet 2)
		if len(candidateDepots) == 0 && depotFilter.Language != "" && depotFilter.Language != "all" {
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
					for idx := range files {
						files[idx].DepotID = fileDepotID
					}
					candidateDepots[fileDepotID] = files
				}
			}
		}

		if len(candidateDepots) > 0 {
			reconciledFiles, includedDepots, recErr := manifest.ReconcileDepots(candidateDepots, depotFilter)
			if recErr == nil && len(reconciledFiles) > 0 {
				firstDepotID := uint32(0)
				if len(includedDepots) > 0 {
					firstDepotID = includedDepots[0]
				}
				loadedManifest = &manifest.Manifest{
					AppID:     req.AppID,
					DepotID:   firstDepotID,
					DepotKeys: depotKeys,
					Files:     reconciledFiles,
				}
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

		// If the manifest module returned a structured error, propagate it directly
		// as the cause instead of generating a misleading generic message.
		var cause error
		if localLoadErr != nil {
			cause = localLoadErr
		} else {
			cause = fmt.Errorf("no depot manifests could be resolved for AppID %d", req.AppID)
		}
		return nil, &DownloadError{
			AppID:      req.AppID,
			AuditTrace: trace,
			Cause:      cause,
		}
	}

	// 5. Validate that all file paths in manifest are decrypted
	for _, f := range loadedManifest.Files {
		if util.IsEncryptedBase64Path(f.Path) {
			return nil, &DownloadError{
				AppID:      req.AppID,
				AuditTrace: parsedLua.AuditTrace,
				Cause:      &manifest.EncryptedDepotError{DepotID: f.DepotID},
			}
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
		Branch:           opts.Branch,
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
		ManifestGIDs:    opts.ManifestGIDs,
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
		candidateDepots := make(map[uint32][]manifest.ManifestFileEntry)
		for dID, files := range depotFiles {
			if manifest.ShouldIncludeDepot(dID, files, depotFilter) {
				candidateDepots[dID] = files
			}
		}
		if len(candidateDepots) > 0 {
			reconciled, _, err := manifest.ReconcileDepots(candidateDepots, depotFilter)
			if err == nil && len(reconciled) > 0 {
				m.Files = reconciled
			}
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

		return nil, &DownloadError{
			AppID:      opts.AppID,
			AuditTrace: trace,
			Cause:      fmt.Errorf("no depot manifests could be resolved for AppID %d", opts.AppID),
		}
	}

	for _, f := range m.Files {
		if util.IsEncryptedBase64Path(f.Path) {
			return nil, &DownloadError{
				AppID:      opts.AppID,
				AuditTrace: opts.AuditTrace,
				Cause:      &manifest.EncryptedDepotError{DepotID: f.DepotID},
			}
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
