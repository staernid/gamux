package util

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz/lzma"
)

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

var zstdDecoderPool = sync.Pool{
	New: func() any {
		dec, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderLowmem(true))
		if err != nil {
			return nil
		}
		return dec
	},
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
		dec, ok := zstdDecoderPool.Get().(*zstd.Decoder)
		if !ok || dec == nil {
			var err error
			dec, err = zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderLowmem(true))
			if err != nil {
				return nil, fmt.Errorf("zstd reader init for VSZa chunk: %w", err)
			}
		}
		defer zstdDecoderPool.Put(dec)

		if err := dec.Reset(bytes.NewReader(chunk[8:])); err != nil {
			return nil, fmt.Errorf("zstd reset for VSZa chunk: %w", err)
		}

		var uncomp []byte
		var err error
		if expectedSize > 0 {
			uncomp, err = io.ReadAll(io.LimitReader(dec, int64(expectedSize)))
		} else {
			uncomp, err = io.ReadAll(dec)
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
		dec, ok := zstdDecoderPool.Get().(*zstd.Decoder)
		if !ok || dec == nil {
			var err error
			dec, err = zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderLowmem(true))
			if err != nil {
				return nil, fmt.Errorf("raw zstd reader init: %w", err)
			}
		}
		defer zstdDecoderPool.Put(dec)

		if err := dec.Reset(bytes.NewReader(chunk)); err != nil {
			return nil, fmt.Errorf("raw zstd reset: %w", err)
		}
		uncomp, err := io.ReadAll(dec)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
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
	if err != nil || fi.IsDir() || fi.Size() < 4 {
		return false, nil
	}
	if fi.Size() > 512*1024*1024 {
		return false, nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("open file %s: %w", filePath, err)
	}

	header := make([]byte, 4)
	n, readErr := f.Read(header)
	f.Close()

	if readErr != nil || n < 4 || !IsVSZaCompressed(header[:n]) {
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
