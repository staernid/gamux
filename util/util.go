package util

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bodgit/sevenzip"
	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz/lzma"
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

// GetSHA1Hash returns the SHA1 hash of a file.
func GetSHA1Hash(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	h := sha1.New()
	if _, err := h.Write(data); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// GetHash returns the SHA1 hash of a file.
func GetHash(filePath string) (string, error) {
	return GetSHA1Hash(filePath)
}

// BackupAndReplace backs up a file and replaces it.
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

	if err := CopyFile(src, dest); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", src, dest, err)
	}
	slog.Info("Replaced file", "src", src, "dest", dest)

	return nil
}

// CopyFile is a helper function to copy a file.
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

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// FileExists checks if a file exists on disk.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ExpandPath resolves tilde (~) and relative paths against home directory.
func ExpandPath(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	if !filepath.IsAbs(path) {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path)
		}
	}
	return path
}


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

// SanitizeFilename replaces illegal filesystem characters with dashes to return a clean folder name.
func SanitizeFilename(name string) string {
	invalidChars := []string{":", "/", "\\", "?", "*", "<", ">", "|", "\""}
	res := name
	for _, c := range invalidChars {
		res = strings.ReplaceAll(res, c, " - ")
	}
	fields := strings.Fields(res)
	clean := strings.Join(fields, " ")
	if clean == "" {
		return "Game"
	}
	return clean
}

// SanitizeInstallDir converts a game title into a clean, 1:1 Steam-style installdir folder name.
// It removes trademark/registered symbols (™, ®, ©), removes illegal OS path characters,
// and normalizes whitespace while preserving standard folder readability.
func SanitizeInstallDir(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Game"
	}
	s := title
	s = strings.ReplaceAll(s, "™", "")
	s = strings.ReplaceAll(s, "®", "")
	s = strings.ReplaceAll(s, "©", "")

	invalidChars := []string{":", "/", "\\", "?", "*", "<", ">", "|", "\""}
	for _, c := range invalidChars {
		s = strings.ReplaceAll(s, c, "")
	}

	fields := strings.Fields(s)
	clean := strings.Join(fields, " ")
	clean = strings.Trim(clean, ". ")
	if clean == "" {
		return "Game"
	}
	return clean
}

// IsEncryptedBase64Path checks if a filename or path contains raw un-decrypted AES base64 output.
func IsEncryptedBase64Path(p string) bool {
	base := filepath.Base(p)
	cleanBase := strings.TrimSpace(strings.ReplaceAll(base, "\n", ""))
	if strings.HasSuffix(cleanBase, "==") || strings.HasSuffix(cleanBase, "=") {
		return true
	}
	if strings.HasPrefix(cleanBase, "+") && len(cleanBase) > 10 {
		return true
	}
	if len(cleanBase) > 30 && !strings.Contains(cleanBase, " ") && !strings.Contains(cleanBase, ".") && strings.ContainsAny(cleanBase, "+/") {
		return true
	}
	return false
}

// IsVSZaCompressed checks if data starts with Valve's VSZa zstd container header.
// Valve's VSZa chunk container format uses:
// - 4 bytes magic: "VSZa" (0x56 0x53 0x5A 0x61)
// - 4 bytes header metadata: 32-bit CRC32 checksum / chunk metadata
// - Payload: Zstandard (zstd) compressed frame starting at offset 8
func IsVSZaCompressed(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	return bytes.Equal(data[:4], []byte("VSZa"))
}

// IsSteamChunkCompressed checks if data starts with any known Steam Pipe chunk container header.
// Supported stream formats:
// 1. "VSZa": Valve Zstandard chunk container (magic 0x56 0x53 0x5A 0x61)
// 2. "VZ" / "VZa": Valve LZMA chunk container (magic 0x56 0x5A)
// 3. "PK\x03\x04": Zip archive chunk (magic 0x50 0x4B 0x03 0x04)
// 4. 0x28B52FFD: Raw Zstandard frame
func IsSteamChunkCompressed(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	if IsVSZaCompressed(data) {
		return true
	}
	if data[0] == 'V' && data[1] == 'Z' {
		return true
	}
	if len(data) >= 4 {
		if bytes.Equal(data[:4], []byte{'P', 'K', 0x03, 0x04}) || binary.LittleEndian.Uint32(data[:4]) == 0x28B52FFD {
			return true
		}
	}
	return false
}

// DecompressChunkSlice decompresses a single Steam Pipe chunk slice (up to 1MB) matching any Steam Pipe stream format.
func DecompressChunkSlice(chunk []byte, expectedSize uint32) ([]byte, error) {
	if len(chunk) < 2 {
		return chunk, nil
	}

	// 1. VSZa (Zstandard) - 4 bytes "VSZa"
	if IsVSZaCompressed(chunk) {
		if len(chunk) < 8 {
			return nil, fmt.Errorf("invalid VSZa chunk header length %d", len(chunk))
		}
		zr, err := zstd.NewReader(bytes.NewReader(chunk[8:]))
		if err != nil {
			return nil, fmt.Errorf("zstd reader init for VSZa chunk: %w", err)
		}
		defer zr.Close()

		var uncomp []byte
		if expectedSize > 0 {
			uncomp, err = io.ReadAll(io.LimitReader(zr, int64(expectedSize)))
		} else {
			uncomp, err = io.ReadAll(zr)
		}
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("zstd decode for VSZa chunk: %w", err)
		}
		return uncomp, nil
	}

	// 2. VZ / VZa (LZMA) - 2 bytes "VZ" (0x56 0x5A)
	if chunk[0] == 'V' && chunk[1] == 'Z' {
		if len(chunk) < 12 {
			return nil, fmt.Errorf("invalid VZ payload length %d", len(chunk))
		}
		// Construct standard 13-byte LZMA header:
		// props (1 byte at offset 7) + dict size (4 bytes at 8:12) + uncompressed size (8 bytes uint64) + compressed stream
		var lzmaBuf bytes.Buffer
		lzmaBuf.WriteByte(chunk[7])
		lzmaBuf.Write(chunk[8:12])
		uncompSize := uint64(expectedSize)
		if uncompSize == 0 {
			uncompSize = ^uint64(0) // 0xFFFFFFFFFFFFFFFF (unknown size specifier in LZMA format)
		}
		_ = binary.Write(&lzmaBuf, binary.LittleEndian, uncompSize)
		lzmaBuf.Write(chunk[12:])

		lzr, err := lzma.NewReader(&lzmaBuf)
		if err != nil {
			return nil, fmt.Errorf("lzma reader init: %w", err)
		}

		var uncomp []byte
		if expectedSize > 0 {
			uncomp, err = io.ReadAll(io.LimitReader(lzr, int64(expectedSize)))
		} else {
			uncomp, err = io.ReadAll(lzr)
		}
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("lzma decode: %w", err)
		}
		return uncomp, nil
	}

	// 3. Zip payload ("PK\x03\x04")
	if len(chunk) >= 4 && bytes.Equal(chunk[:4], []byte{'P', 'K', 0x03, 0x04}) {
		r, err := zip.NewReader(bytes.NewReader(chunk), int64(len(chunk)))
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

	// 4. Raw Zstandard frame (0x28B52FFD)
	if len(chunk) >= 4 && binary.LittleEndian.Uint32(chunk[:4]) == 0x28B52FFD {
		zr, err := zstd.NewReader(bytes.NewReader(chunk))
		if err != nil {
			return nil, fmt.Errorf("raw zstd reader init: %w", err)
		}
		uncomp, err := io.ReadAll(zr)
		zr.Close()
		if err != nil {
			return nil, fmt.Errorf("raw zstd decode: %w", err)
		}
		return uncomp, nil
	}

	return chunk, nil
}

// DecompressVSZaData decompresses a stream of 1MB (1,048,576 byte) Steam Pipe chunk slices using any supported stream format.
// Files downloaded directly as raw Steam depot chunks are stored as concatenated 1MB container frames.
func DecompressVSZaData(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	const chunkSize = 1048576 // 1MB Steam chunk slice
	var result bytes.Buffer

	for offset := 0; offset < len(data); offset += chunkSize {
		end := offset + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[offset:end]

		if IsSteamChunkCompressed(chunk) {
			decomp, err := DecompressChunkSlice(chunk, 0)
			if err != nil {
				return nil, fmt.Errorf("decompress Steam chunk at offset %d: %w", offset, err)
			}
			result.Write(decomp)
		} else {
			result.Write(chunk)
		}
	}

	return result.Bytes(), nil
}

// DecompressVSZaFile decompresses a compressed Steam depot file on disk in place.
// Returns true if the file was compressed and decompressed successfully.
func DecompressVSZaFile(filePath string) (bool, error) {
	fi, err := os.Stat(filePath)
	if err != nil || fi.IsDir() || fi.Size() < 2 {
		return false, nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("open file %s: %w", filePath, err)
	}

	header := make([]byte, 4)
	n, readErr := f.Read(header)
	f.Close()

	if readErr != nil || n < 2 || !IsSteamChunkCompressed(header[:n]) {
		return false, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("read file %s: %w", filePath, err)
	}

	decompressed, err := DecompressVSZaData(data)
	if err != nil {
		return false, fmt.Errorf("decompress file %s: %w", filePath, err)
	}

	// Write decompressed content atomically
	tmpFile := filePath + ".tmp_vsza"
	if err := os.WriteFile(tmpFile, decompressed, fi.Mode()); err != nil {
		return false, fmt.Errorf("write temp file for %s: %w", filePath, err)
	}

	if err := os.Rename(tmpFile, filePath); err != nil {
		_ = os.Remove(tmpFile)
		return false, fmt.Errorf("replace file %s: %w", filePath, err)
	}

	slog.Info("Decompressed VSZa compressed file", "path", filePath, "origSize", fi.Size(), "newSize", len(decompressed))
	return true, nil
}

// DecompressVSZaInDir walks dirPath recursively and decompresses any VSZa-compressed files in place.
// Returns the total number of files decompressed.
func DecompressVSZaInDir(dirPath string) (int, error) {
	if !FileExists(dirPath) {
		return 0, nil
	}

	var count int
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		rel, relErr := filepath.Rel(dirPath, path)
		if relErr == nil {
			if strings.HasPrefix(rel, "[Manifests]") ||
				strings.HasPrefix(rel, "[Steam]") ||
				strings.HasPrefix(rel, "steam_settings") ||
				strings.HasPrefix(rel, ".git") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if d.IsDir() {
			return nil
		}

		decompressed, decErr := DecompressVSZaFile(path)
		if decErr != nil {
			slog.Warn("Failed to decompress VSZa file", "path", path, "error", decErr)
			return nil
		}
		if decompressed {
			count++
		}
		return nil
	})

	if err != nil {
		return count, fmt.Errorf("walk directory %s for VSZa decompression: %w", dirPath, err)
	}

	if count > 0 {
		slog.Info("Decompressed VSZa files in directory", "dir", dirPath, "count", count)
	}
	return count, nil
}




