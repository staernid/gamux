package downloader

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/util"
	"golang.org/x/sync/errgroup"
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
