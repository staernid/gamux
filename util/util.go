package util

import (
	"archive/tar"
	"bytes"
	"compress/bzip2"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/bodgit/sevenzip"
	"golang.org/x/crypto/md4"
)

// runCmd executes a command and returns its output or an error.
func RunCmd(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("command failed: %s %v\nstdout: %s\nstderr: %s", name, args, stdout.String(), stderr.String())
	}
	return stdout.Bytes(), nil
}

// getHash returns the MD5 hash of a file.
func GetHash(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	h := md4.New()
	if _, err := h.Write(data); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// backupAndReplace backs up a file and replaces it.
func BackupAndReplace(src, dest string) error {
	timestamp := time.Now().Format("20060102-150405")
	// Check if the destination file exists before attempting to backup
	if _, err := os.Stat(dest); err == nil {
		backupPath := fmt.Sprintf("%s.%s.ORIGINAL", dest, timestamp)
		if err := os.Rename(dest, backupPath); err != nil {
			return fmt.Errorf("failed to backup %s: %w", dest, err)
		}
		slog.Info("Backed up file", "from", dest, "to", backupPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat destination file %s: %w", dest, err)
	}

	if err := os.Link(src, dest); err != nil {
		// Fallback to copy if hard link fails
		if err := CopyFile(src, dest); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", src, dest, err)
		}
	}
	slog.Info("Replaced file", "src", src, "dest", dest)

	return nil
}

// copyFile is a helper function to copy a file.
func CopyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// downloadAndExtract downloads a file and extracts it.
func DownloadAndExtract(url, destDir, format string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := os.RemoveAll(destDir); err != nil {
		return err
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

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
			targetPath := filepath.Join(destDir, header.Name)
			if header.FileInfo().IsDir() {
				if err := os.MkdirAll(targetPath, header.FileInfo().Mode()); err != nil {
					return err
				}
				continue
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
		entries, err := os.ReadDir(destDir)
		if err != nil {
			return fmt.Errorf("failed to read destination directory after tar.bz2 extraction: %w", err)
		}

		if len(entries) == 1 && entries[0].IsDir() {
			nestedDirPath := filepath.Join(destDir, entries[0].Name())
			slog.Info("Found single nested directory, moving contents up", "nestedDir", nestedDirPath)

			nestedEntries, err := os.ReadDir(nestedDirPath)
			if err != nil {
				return fmt.Errorf("failed to read nested directory '%s': %w", nestedDirPath, err)
			}

			for _, entry := range nestedEntries {
				oldPath := filepath.Join(nestedDirPath, entry.Name())
				newPath := filepath.Join(destDir, entry.Name())
				if err := os.Rename(oldPath, newPath); err != nil {
					return fmt.Errorf("failed to move '%s' to '%s': %w", oldPath, newPath, err)
				}
			}
			if err := os.Remove(nestedDirPath); err != nil {
				return fmt.Errorf("failed to remove empty nested directory '%s': %w", nestedDirPath, err)
			}
		}

	case "7z":
		tempFile := filepath.Join(os.TempDir(), "temp.7z")
		outFile, err := os.Create(tempFile)
		if err != nil {
			return err
		}
		if _, err := io.Copy(outFile, resp.Body); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
		defer os.Remove(tempFile)

		r, err := sevenzip.OpenReader(tempFile)
		if err != nil {
			return fmt.Errorf("failed to open 7z archive: %w", err)
		}
		defer r.Close()

		for _, f := range r.File {
			targetPath := filepath.Join(destDir, f.Name)
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
		releasePath := filepath.Join(destDir, "release")
		if _, err := os.Stat(releasePath); err == nil {
			entries, err := os.ReadDir(releasePath)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if err := os.Rename(filepath.Join(releasePath, entry.Name()), filepath.Join(destDir, entry.Name())); err != nil {
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
