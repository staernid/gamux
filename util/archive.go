package util

import (
	"archive/tar"
	"compress/bzip2"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bodgit/sevenzip"
)

// DownloadAndExtract downloads a file and extracts it.
func DownloadAndExtract(ctx context.Context, url, destDir, format string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download release: HTTP %s", resp.Status)
	}

	if err := os.RemoveAll(destDir); err != nil {
		return err
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	destDirClean := filepath.Clean(destDir)

	switch format {
	case "tar.bz2":
		bzip2Reader := bzip2.NewReader(resp.Body)
		tarReader := tar.NewReader(bzip2Reader)
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			cleanName := filepath.Clean(header.Name)
			if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
				continue
			}

			targetPath := filepath.Join(destDirClean, cleanName)
			rel, err := filepath.Rel(destDirClean, targetPath)
			if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
				continue
			}

			if header.FileInfo().IsDir() {
				if err := os.MkdirAll(targetPath, header.FileInfo().Mode()); err != nil {
					return err
				}
				continue
			}

			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}

			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return err
			}
			file.Close()
		}

		// After extraction, check if there's a single top-level directory and move its contents up
		entries, err := os.ReadDir(destDirClean)
		if err != nil {
			return fmt.Errorf("failed to read destination directory after tar.bz2 extraction: %w", err)
		}

		if len(entries) == 1 && entries[0].IsDir() {
			nestedDirPath := filepath.Join(destDirClean, entries[0].Name())
			slog.Info("Found single nested directory, moving contents up", "nestedDir", nestedDirPath)

			nestedEntries, err := os.ReadDir(nestedDirPath)
			if err != nil {
				return fmt.Errorf("failed to read nested directory '%s': %w", nestedDirPath, err)
			}

			for _, entry := range nestedEntries {
				oldPath := filepath.Join(nestedDirPath, entry.Name())
				newPath := filepath.Join(destDirClean, entry.Name())
				if err := os.Rename(oldPath, newPath); err != nil {
					return fmt.Errorf("failed to move '%s' to '%s': %w", oldPath, newPath, err)
				}
			}
			if err := os.Remove(nestedDirPath); err != nil {
				return fmt.Errorf("failed to remove empty nested directory '%s': %w", nestedDirPath, err)
			}
		}

	case "7z":
		tempFile, err := os.CreateTemp("", "gamux-extract-*.7z")
		if err != nil {
			return fmt.Errorf("create temp 7z file: %w", err)
		}
		tempFilePath := tempFile.Name()
		defer os.Remove(tempFilePath)

		if _, err := io.Copy(tempFile, resp.Body); err != nil {
			tempFile.Close()
			return err
		}
		tempFile.Close()

		r, err := sevenzip.OpenReader(tempFilePath)
		if err != nil {
			return fmt.Errorf("failed to open 7z archive: %w", err)
		}
		defer r.Close()

		for _, f := range r.File {
			cleanName := filepath.Clean(f.Name)
			if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
				continue
			}

			targetPath := filepath.Join(destDirClean, cleanName)
			rel, err := filepath.Rel(destDirClean, targetPath)
			if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
				continue
			}

			if f.FileInfo().IsDir() {
				if err := os.MkdirAll(targetPath, f.Mode()); err != nil {
					return err
				}
				continue
			}

			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}

			rc, err := f.Open()
			if err != nil {
				return err
			}

			dst, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				rc.Close()
				return err
			}

			if _, err := io.Copy(dst, rc); err != nil {
				dst.Close()
				rc.Close()
				return err
			}
			dst.Close()
			rc.Close()
		}

		// Move contents of 'release' subdirectory up
		releasePath := filepath.Join(destDirClean, "release")
		if _, err := os.Stat(releasePath); err == nil {
			entries, err := os.ReadDir(releasePath)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if err := os.Rename(filepath.Join(releasePath, entry.Name()), filepath.Join(destDirClean, entry.Name())); err != nil {
					return err
				}
			}
			if err := os.Remove(releasePath); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported archive format: %s", format)
	}

	return nil
}
