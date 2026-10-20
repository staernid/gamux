package downloader

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/util"
	"golang.org/x/sync/errgroup"
)

const (
	defaultWorkerLimit = 10
	chunkWorkerLimit   = 4
)

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

// depotDownload carries the shared state of one streamDepotFiles run.
type depotDownload struct {
	m            *manifest.Manifest
	req          DownloadRequest
	absTargetDir string
	client       *http.Client
	hosts        []string

	cp       *Checkpoint
	cpFile   string
	diffMap  map[string]manifest.FileChunkDiff
	progress *progressTracker
}

func streamDepotFiles(ctx context.Context, m *manifest.Manifest, absTargetDir string, req DownloadRequest, httpClient *http.Client, rankedHosts []string) (*Result, error) {
	workerLimit := req.WorkerLimit
	if workerLimit <= 0 {
		workerLimit = defaultWorkerLimit
	}

	slog.Info("Starting direct game download/update",
		"appID", m.AppID,
		"depotID", m.DepotID,
		"files", len(m.Files),
		"targetDir", absTargetDir,
		"cdnHosts", len(rankedHosts),
		"dryRun", req.DryRun,
	)

	result := &Result{TotalFiles: len(m.Files)}

	manifestsDir := filepath.Join(absTargetDir, "[Manifests]")
	checkpointFile := filepath.Join(manifestsDir, ".gamux-checkpoint.json")

	d := &depotDownload{
		m:            m,
		req:          req,
		absTargetDir: absTargetDir,
		client:       httpClient,
		hosts:        rankedHosts,
		cp:           loadCheckpoint(checkpointFile),
		cpFile:       checkpointFile,
	}

	var downloadSucceeded bool
	if !req.DryRun {
		defer func() {
			if !downloadSucceeded {
				d.cp.Flush(checkpointFile)
			}
		}()
		if err := os.MkdirAll(absTargetDir, 0755); err != nil {
			return nil, fmt.Errorf("create target directory %s: %w", absTargetDir, err)
		}
		_ = os.MkdirAll(manifestsDir, 0755)
		d.precreateDirs()
	}

	var deletedFiles []string
	d.diffMap, deletedFiles = d.computeDelta()
	if len(d.diffMap) > 0 {
		slog.Info("Discovered previous depot manifests for delta patching",
			"deltaFiles", len(d.diffMap),
			"deletedFiles", len(deletedFiles),
		)
	}
	if !req.DryRun && len(deletedFiles) > 0 {
		d.purgeDeleted(deletedFiles)
	}

	// Defense-in-depth: Deduplicate file entries by normalized relative path.
	// If two entries have the same relative path:
	// - If identical SHA-1/Size: keep one, ignore duplicate.
	// - If conflicting SHA-1/Size: abort early with a clear error to prevent corruption.
	filesToDownload := make([]manifest.ManifestFileEntry, 0, len(m.Files))
	seenPaths := make(map[string]manifest.ManifestFileEntry, len(m.Files))
	for _, f := range m.Files {
		relPath := manifest.NormalizePath(f.Path)
		if relPath == "" && isDirectoryEntry(f) {
			continue
		}
		if existing, ok := seenPaths[relPath]; ok {
			if existing.SHA1 == f.SHA1 && existing.Size == f.Size {
				continue
			}
			return nil, fmt.Errorf("conflicting file versions detected for path %q (depot %d vs depot %d)", relPath, existing.DepotID, f.DepotID)
		}
		seenPaths[relPath] = f
		filesToDownload = append(filesToDownload, f)
	}

	var totalDepotBytes int64
	for _, f := range filesToDownload {
		if f.Size > 0 && !isDirectoryEntry(f) {
			totalDepotBytes += int64(f.Size)
		}
	}
	d.progress = newProgressTracker(req.OnProgress, totalDepotBytes, len(filesToDownload))

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(workerLimit)
	for _, fileEntry := range filesToDownload {
		entry := fileEntry
		relPath := manifest.NormalizePath(entry.Path)
		if relPath == "" && isDirectoryEntry(entry) {
			continue
		}
		filePath := filepath.Join(absTargetDir, relPath)
		g.Go(func() error { return d.processFile(gCtx, entry, relPath, filePath) })
	}

	if err := g.Wait(); err != nil {
		slog.Error("Direct game download aborted due to worker error",
			"appID", m.AppID,
			"depotID", m.DepotID,
			"error", err,
		)
		return nil, fmt.Errorf("download execution failed: %w", err)
	}

	result.UpdatedFiles = int(d.progress.updatedCount.Load())
	result.DownloadedBytes = uint64(d.progress.downloadedBytes.Load())
	result.NetworkBytes = uint64(d.progress.networkBytes.Load())
	d.progress.flush("Completed")

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

// precreateDirs sequentially creates every directory the manifest needs,
// replacing regular files that squat on a directory path.
func (d *depotDownload) precreateDirs() {
	for _, entry := range d.m.Files {
		relPath := manifest.NormalizePath(entry.Path)
		if relPath == "" {
			continue
		}
		p := filepath.Join(d.absTargetDir, relPath)
		dir := filepath.Dir(p)
		if isDirectoryEntry(entry) {
			dir = p
		}
		if fi, err := os.Stat(dir); err == nil && !fi.IsDir() {
			_ = os.Remove(dir)
		}
		_ = os.MkdirAll(dir, 0755)
	}
}

// computeDelta diffs the target manifest against previously installed depot
// manifests, returning per-file chunk diffs and the files deleted upstream.
func (d *depotDownload) computeDelta() (map[string]manifest.FileChunkDiff, []string) {
	m := d.m
	diffMap := make(map[string]manifest.FileChunkDiff)
	var deleted []string

	collect := func(diff *manifest.DepotDiff) {
		if diff == nil {
			return
		}
		for path, fd := range diff.ModifiedFiles {
			diffMap[manifest.NormalizePath(path)] = fd
		}
		deleted = append(deleted, diff.DeletedFiles...)
	}

	if d.req.PreviousManifest != nil {
		collect(manifest.DiffDepotManifests(d.req.PreviousManifest, m))
		return diffMap, deleted
	}

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
		prevM, err := manifest.LoadPreviousManifestFromDir(d.absTargetDir, did, m.DepotKeys)
		if err != nil || prevM == nil {
			continue
		}
		collect(manifest.DiffDepotManifests(prevM, &manifest.Manifest{
			AppID:         m.AppID,
			DepotID:       did,
			DecryptionKey: m.GetDecryptionKey(did),
			DepotKeys:     m.DepotKeys,
			Files:         dFiles,
		}))
	}
	return diffMap, deleted
}

// purgeDeleted removes files dropped upstream, unless the current manifest still lists them.
func (d *depotDownload) purgeDeleted(deletedFiles []string) {
	activePaths := make(map[string]bool, len(d.m.Files))
	for _, f := range d.m.Files {
		activePaths[manifest.NormalizePath(f.Path)] = true
	}
	for _, delPath := range deletedFiles {
		normDel := manifest.NormalizePath(delPath)
		if normDel == "" || activePaths[normDel] {
			continue
		}
		absDel := filepath.Join(d.absTargetDir, normDel)
		if fi, err := os.Stat(absDel); err == nil && !fi.IsDir() {
			if err := os.Remove(absDel); err == nil {
				slog.Info("Removed deleted depot file", "path", normDel)
			}
		}
	}
}

func (d *depotDownload) checkCancel() error {
	if d.req.CheckCancel != nil {
		return d.req.CheckCancel()
	}
	return nil
}

func (d *depotDownload) depotIDFor(entry manifest.ManifestFileEntry) uint32 {
	if entry.DepotID != 0 {
		return entry.DepotID
	}
	return d.m.DepotID
}

// processFile brings a single manifest entry up to date on disk.
func (d *depotDownload) processFile(ctx context.Context, entry manifest.ManifestFileEntry, relPath, filePath string) error {
	if err := d.checkCancel(); err != nil {
		return err
	}

	d.progress.processedCount.Add(1)
	d.progress.report(entry.Path)

	// Skip directory entries in download loop
	if isDirectoryEntry(entry) {
		return nil
	}

	info, statErr := os.Stat(filePath)
	fileExists := statErr == nil && !info.IsDir()

	if !needsDownload(filePath, entry, info, fileExists) {
		slog.Debug("File is up-to-date", "path", filePath)
		d.progress.downloadedBytes.Add(int64(entry.Size))
		d.progress.report(entry.Path)
		return nil
	}

	if d.req.DryRun {
		if len(entry.Chunks) > 0 && fileExists && uint64(info.Size()) == entry.Size && verifyFileChunksOnDisk(filePath, entry.Chunks) {
			slog.Debug("File is up-to-date", "path", filePath)
			d.progress.downloadedBytes.Add(int64(entry.Size))
			d.progress.report(entry.Path)
			return nil
		}
		slog.Debug("[DRY RUN] Would download/update file", "path", filePath, "size", entry.Size)
		d.progress.updatedCount.Add(1)
		d.progress.downloadedBytes.Add(int64(entry.Size))
		d.progress.report(entry.Path)
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

	var err error
	var updated bool
	if len(entry.Chunks) == 0 {
		err = d.writeEmptyFile(entry, filePath)
		updated = true
	} else {
		updated, err = d.downloadChunked(ctx, entry, relPath, filePath, info, fileExists)
	}
	if err != nil {
		return err
	}

	if updated {
		slog.Debug("Successfully updated file", "path", filePath)
		d.progress.updatedCount.Add(1)
	} else {
		slog.Debug("File is up-to-date", "path", filePath)
	}
	return nil
}

// needsDownload reports whether the on-disk file differs from the manifest entry.
func needsDownload(filePath string, entry manifest.ManifestFileEntry, info os.FileInfo, fileExists bool) bool {
	if !fileExists {
		slog.Debug("New file, needs download", "path", filePath)
		return true
	}
	if uint64(info.Size()) != entry.Size {
		slog.Debug("File size mismatch, redownloading", "path", filePath, "actual", info.Size(), "expected", entry.Size)
		return true
	}
	// For chunked files, pendingChunks in downloadChunked handles chunk-level verification
	// in a single pass without pre-hashing the full file.
	if len(entry.Chunks) > 0 {
		return true
	}
	if entry.SHA1 != "" {
		localHash, err := util.GetSHA1Hash(filePath)
		if err != nil || !strings.EqualFold(localHash, entry.SHA1) {
			slog.Debug("File hash mismatch, redownloading", "path", filePath, "actual", localHash, "expected", entry.SHA1)
			return true
		}
		return false
	}
	return false
}

// writeEmptyFile creates a chunkless entry (empty or preallocated-only file).
func (d *depotDownload) writeEmptyFile(entry manifest.ManifestFileEntry, filePath string) error {
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
	d.progress.downloadedBytes.Add(int64(entry.Size))
	d.progress.report(entry.Path)
	return nil
}

// downloadChunked fetches the chunks of an entry that are missing or stale,
// finalizes the file, and verifies its SHA-1 (falling back to a full
// redownload on mismatch). It returns whether any chunks were fetched/updated.
func (d *depotDownload) downloadChunked(ctx context.Context, entry manifest.ManifestFileEntry, relPath, filePath string, info os.FileInfo, fileExists bool) (bool, error) {
	fileDiff, isDelta := d.diffMap[relPath]
	canDelta := isDelta && fileExists && info.Size() > 0

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return false, fmt.Errorf("open target file %s: %w", filePath, err)
	}

	if !canDelta && (!fileExists || info.Size() < int64(entry.Size)) {
		if err := util.PreallocateFile(f, int64(entry.Size)); err != nil {
			f.Close()
			return false, fmt.Errorf("preallocate target file %s: %w", filePath, err)
		}
	}

	unchanged := make(map[string]bool)
	if canDelta {
		for _, ch := range fileDiff.UnchangedChunks {
			if info.Size() >= int64(ch.Offset+uint64(ch.UncompressedSize)) {
				unchanged[ch.ChunkID] = true
			}
		}
	}

	pending, err := d.pendingChunks(f, entry, relPath, filePath, info, fileExists, unchanged)
	if err != nil {
		f.Close()
		return false, err
	}

	if len(pending) > 0 {
		if err := d.fetchChunks(ctx, f, entry, relPath, filePath, pending, false); err != nil {
			f.Close()
			return false, err
		}
	}

	if err := f.Truncate(int64(entry.Size)); err != nil {
		f.Close()
		return false, fmt.Errorf("truncate final file %s: %w", filePath, err)
	}
	f.Close()

	if isExecutableEntry(entry) {
		_ = os.Chmod(filePath, 0755)
	}

	// Post-download integrity verification against SHA-1 if present and chunks were updated.
	// If zero chunks were fetched (len(pending) == 0), the chunks on disk were already verified.
	if entry.SHA1 != "" && len(pending) > 0 {
		localHash, err := util.GetSHA1Hash(filePath)
		if err != nil || !strings.EqualFold(localHash, entry.SHA1) {
			slog.Warn("Downloaded file hash mismatch, falling back to full file download",
				"path", filePath, "actual", localHash, "expected", entry.SHA1)
			return true, d.redownloadFull(ctx, entry, relPath, filePath)
		}
	}
	return len(pending) > 0, nil
}

// pendingChunks returns the chunks that still need a network fetch, crediting
// progress for chunks already present (checkpoint, delta, or valid on disk).
func (d *depotDownload) pendingChunks(f *os.File, entry manifest.ManifestFileEntry, relPath, filePath string, info os.FileInfo, fileExists bool, unchanged map[string]bool) ([]manifest.ChunkInfo, error) {
	var pending []manifest.ChunkInfo
	chunkBuf := make([]byte, 1024*1024)
	for _, chunk := range entry.Chunks {
		if err := d.checkCancel(); err != nil {
			return nil, err
		}

		key := chunkKey(relPath, chunk)
		if d.cp.isCompleted(key) || unchanged[chunk.ChunkID] {
			slog.Debug("Chunk already present or completed in checkpoint, skipping network fetch", "chunkID", chunk.ChunkID, "path", filePath)
			d.creditChunk(entry, key, chunk)
			continue
		}

		// In-place chunk repair: if file exists and chunk on disk already matches expected ChunkID, skip download
		if fileExists && info.Size() >= int64(chunk.Offset+uint64(chunk.UncompressedSize)) {
			if manifest.IsChunkValidOnDisk(f, chunk, chunkBuf) {
				slog.Debug("Chunk already valid on disk, skipping network fetch", "chunkID", chunk.ChunkID, "offset", chunk.Offset, "path", filePath)
				d.creditChunk(entry, key, chunk)
				continue
			}
		}

		pending = append(pending, chunk)
	}
	return pending, nil
}

func (d *depotDownload) creditChunk(entry manifest.ManifestFileEntry, key string, chunk manifest.ChunkInfo) {
	d.progress.downloadedBytes.Add(int64(chunk.UncompressedSize))
	d.progress.report(entry.Path)
	d.cp.markCompleted(d.cpFile, key)
}

// fetchChunks downloads chunks concurrently into f. In fallback mode (a full
// redownload after a hash mismatch) cancellation checks, throttling and
// downloaded-byte accounting are skipped because those bytes were already
// counted by the first pass.
func (d *depotDownload) fetchChunks(ctx context.Context, f *os.File, entry manifest.ManifestFileEntry, relPath, filePath string, chunks []manifest.ChunkInfo, fallback bool) error {
	workers := chunkWorkerLimit
	if workers > len(chunks) {
		workers = len(chunks)
	}
	cg, cgCtx := errgroup.WithContext(ctx)
	cg.SetLimit(workers)

	depotID := d.depotIDFor(entry)
	decKey := d.m.GetDecryptionKey(depotID)

	for _, ch := range chunks {
		chunk := ch
		cg.Go(func() error {
			if !fallback {
				if err := d.checkCancel(); err != nil {
					return err
				}
			}

			if err := downloadChunk(cgCtx, d.client, d.hosts, depotID, decKey, chunk, f); err != nil {
				if fallback {
					return fmt.Errorf("download chunk during fallback for %s: %w", filePath, err)
				}
				return fmt.Errorf("download chunk for %s (depot %d): %w", filePath, depotID, err)
			}

			if !fallback {
				if d.req.Throttle != nil {
					_ = d.req.Throttle(int(chunk.UncompressedSize))
				}
				d.progress.downloadedBytes.Add(int64(chunk.UncompressedSize))
			}
			d.progress.networkBytes.Add(int64(chunk.UncompressedSize))
			if !fallback {
				d.progress.report(entry.Path)
			}
			d.cp.markCompleted(d.cpFile, chunkKey(relPath, chunk))
			return nil
		})
	}
	return cg.Wait()
}

// redownloadFull truncates the file and refetches every chunk, then verifies SHA-1 again.
func (d *depotDownload) redownloadFull(ctx context.Context, entry manifest.ManifestFileEntry, relPath, filePath string) error {
	fFull, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("reopen file for full download %s: %w", filePath, err)
	}
	if err := util.PreallocateFile(fFull, int64(entry.Size)); err != nil {
		fFull.Close()
		return fmt.Errorf("preallocate fallback file %s: %w", filePath, err)
	}

	if err := d.fetchChunks(ctx, fFull, entry, relPath, filePath, entry.Chunks, true); err != nil {
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
	return nil
}
