package manifest

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	ProtoPayloadMagic       uint32 = 0x71F617D0
	ProtoMetadataMagic      uint32 = 0x1F4812BE
	ProtoSignatureMagic     uint32 = 0x1B81B817
	ProtoEndOfManifestMagic uint32 = 0x32C415AB
)

// decodeVarint decodes a Protobuf varint from data bytes.
func decodeVarint(data []byte) (uint64, int) {
	var val uint64
	var shift uint
	for i, b := range data {
		val |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			return val, i + 1
		}
		shift += 7
		if shift >= 64 {
			return 0, 0
		}
	}
	return 0, 0
}

// parseChunkInfo decodes a single Protobuf chunk payload.
func parseChunkInfo(data []byte) *ChunkInfo {
	chunk := &ChunkInfo{}
	offset := 0
	for offset < len(data) {
		numWire, n := decodeVarint(data[offset:])
		if n == 0 {
			break
		}
		offset += n

		fieldNum := numWire >> 3
		wireType := numWire & 0x07

		switch wireType {
		case 0: // Varint
			val, n := decodeVarint(data[offset:])
			if n == 0 {
				break
			}
			offset += n
			switch fieldNum {
			case 3:
				chunk.Offset = val
			case 4:
				chunk.UncompressedSize = uint32(val)
			case 5:
				chunk.CompressedSize = uint32(val)
			}
		case 2: // Length-delimited (Field 1: SHA1 / ChunkID)
			length, n := decodeVarint(data[offset:])
			if n == 0 {
				break
			}
			offset += n
			if offset+int(length) > len(data) {
				break
			}
			valBytes := data[offset : offset+int(length)]
			offset += int(length)

			if fieldNum == 1 {
				chunk.ChunkID = hex.EncodeToString(valBytes)
			}
		case 5: // 32-bit fixed (Field 2: CRC Checksum)
			if offset+4 > len(data) {
				break
			}
			val := binary.LittleEndian.Uint32(data[offset : offset+4])
			offset += 4
			if fieldNum == 2 {
				chunk.Checksum = fmt.Sprintf("%08x", val)
			}
		case 1: // 64-bit fixed
			offset += 8
		default:
			return chunk
		}
	}
	return chunk
}

// parseFileEntry decodes a single Protobuf manifest file entry payload.
func parseFileEntry(data []byte, hexKey string) (*ManifestFileEntry, error) {
	entry := &ManifestFileEntry{}
	offset := 0
	for offset < len(data) {
		numWire, n := decodeVarint(data[offset:])
		if n == 0 {
			break
		}
		offset += n

		fieldNum := numWire >> 3
		wireType := numWire & 0x07

		switch wireType {
		case 0: // Varint
			val, n := decodeVarint(data[offset:])
			if n == 0 {
				break
			}
			offset += n
			if fieldNum == 2 {
				entry.Size = val
			} else if fieldNum == 3 {
				entry.Flags = uint32(val)
			}
		case 2: // Length-delimited
			length, n := decodeVarint(data[offset:])
			if n == 0 {
				break
			}
			offset += n
			if offset+int(length) > len(data) {
				break
			}
			valBytes := data[offset : offset+int(length)]
			offset += int(length)

			if fieldNum == 1 {
				pathStr := string(valBytes)
				if hexKey != "" {
					if decPath, err := DecryptFilename(pathStr, hexKey); err == nil && decPath != "" {
						pathStr = decPath
					}
				}
				entry.Path = NormalizePath(pathStr)
			} else if fieldNum == 5 {
				entry.SHA1 = hex.EncodeToString(valBytes)
			} else if fieldNum == 6 {
				chunk := parseChunkInfo(valBytes)
				if chunk != nil {
					entry.Chunks = append(entry.Chunks, *chunk)
				}
			}
		case 5: // 32-bit fixed
			offset += 4
		case 1: // 64-bit fixed
			offset += 8
		default:
			return entry, nil
		}
	}
	return entry, nil
}

// ParseBinaryManifest parses a Steam binary .manifest container payload into structured file entries.
func ParseBinaryManifest(data []byte) ([]ManifestFileEntry, error) {
	return ParseBinaryManifestWithKey(data, "")
}

// ParseBinaryManifestWithKey parses a binary manifest container, optionally decrypting filenames if a depot key is provided.
func ParseBinaryManifestWithKey(data []byte, hexKey string) ([]ManifestFileEntry, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty manifest data")
	}

	// If zip file, read first uncompressed stream
	if len(data) >= 4 && bytes.Equal(data[:4], []byte{'P', 'K', 0x03, 0x04}) {
		r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err == nil && len(r.File) > 0 {
			rc, err := r.File[0].Open()
			if err == nil {
				unzipped, err := io.ReadAll(rc)
				rc.Close()
				if err == nil {
					data = unzipped
				}
			}
		}
	}

	if len(data) < 8 {
		return nil, fmt.Errorf("manifest data too short")
	}

	magic := binary.LittleEndian.Uint32(data[0:4])
	if magic != ProtoPayloadMagic {
		return nil, fmt.Errorf("invalid manifest payload magic 0x%08X", magic)
	}

	payloadLen := binary.LittleEndian.Uint32(data[4:8])
	if 8+payloadLen > uint32(len(data)) {
		return nil, fmt.Errorf("payload length overflow: %d > %d", payloadLen, len(data)-8)
	}

	payloadData := data[8 : 8+payloadLen]

	var files []ManifestFileEntry
	offset := 0
	for offset < len(payloadData) {
		numWire, n := decodeVarint(payloadData[offset:])
		if n == 0 {
			break
		}
		offset += n

		fieldNum := numWire >> 3
		wireType := numWire & 0x07

		if fieldNum == 1 && wireType == 2 { // mappings
			length, n := decodeVarint(payloadData[offset:])
			if n == 0 {
				break
			}
			offset += n
			if offset+int(length) > len(payloadData) {
				break
			}
			mappingData := payloadData[offset : offset+int(length)]
			offset += int(length)

			entry, err := parseFileEntry(mappingData, hexKey)
			if err == nil && entry != nil && entry.Path != "" {
				files = append(files, *entry)
			}
		} else {
			// Skip unknown payload fields
			switch wireType {
			case 0:
				_, n := decodeVarint(payloadData[offset:])
				if n == 0 {
					break
				}
				offset += n
			case 2:
				length, n := decodeVarint(payloadData[offset:])
				if n == 0 {
					break
				}
				offset += n + int(length)
			case 5:
				offset += 4
			case 1:
				offset += 8
			default:
				offset++
			}
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no valid file entries parsed from binary manifest")
	}

	return files, nil
}

// LoadManifestFromDir scans targetDir/[Manifests]/ for .manifest files and returns a populated Manifest.
func LoadManifestFromDir(targetDir string, appID uint32) (*Manifest, error) {
	return LoadManifestFromDirWithKeys(targetDir, appID, nil)
}

func LoadManifestFromDirWithKeys(targetDir string, appID uint32, depotKeys map[uint32]string) (*Manifest, error) {
	manifestsDir := filepath.Join(targetDir, "[Manifests]")
	if _, err := os.Stat(manifestsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("manifests directory %s does not exist", manifestsDir)
	}

	entries, err := os.ReadDir(manifestsDir)
	if err != nil {
		return nil, fmt.Errorf("read manifests dir %s: %w", manifestsDir, err)
	}

	// Try reading local keys from any .lua files in [Manifests] if depotKeys is empty
	if len(depotKeys) == 0 {
		depotKeys = make(map[uint32]string)
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".lua") {
				luaPath := filepath.Join(manifestsDir, e.Name())
				if content, err := os.ReadFile(luaPath); err == nil {
					if parsed, err := ParseLua(string(content)); err == nil && len(parsed.Depots) > 0 {
						for _, pair := range parsed.Depots {
							depotKeys[pair.DepotID] = pair.DecryptionKey
						}
					}
				}
			}
		}
	}

	type manifestFileInfo struct {
		name       string
		depotID    uint32
		manifestID uint64
		path       string
	}

	depotLatestManifest := make(map[uint32]manifestFileInfo)

	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".manifest") {
			name := e.Name()
			parts := strings.Split(name, "_")
			var fileDepotID uint32
			var fileManifestID uint64

			if len(parts) >= 2 {
				if did, err := strconv.ParseUint(parts[0], 10, 32); err == nil {
					fileDepotID = uint32(did)
				}
				midStr := strings.TrimSuffix(parts[1], ".manifest")
				if mid, err := strconv.ParseUint(midStr, 10, 64); err == nil {
					fileManifestID = mid
				}
			}

			manifestPath := filepath.Join(manifestsDir, name)

			existing, exists := depotLatestManifest[fileDepotID]
			if !exists || fileManifestID > existing.manifestID {
				depotLatestManifest[fileDepotID] = manifestFileInfo{
					name:       name,
					depotID:    fileDepotID,
					manifestID: fileManifestID,
					path:       manifestPath,
				}
			}
		}
	}

	allFiles := make([]ManifestFileEntry, 0)
	var resolvedDepotID uint32

	for dID, info := range depotLatestManifest {
		if resolvedDepotID == 0 {
			resolvedDepotID = dID
		}
		data, err := os.ReadFile(info.path)
		if err != nil {
			continue
		}

		hexKey := ""
		if depotKeys != nil {
			hexKey = depotKeys[dID]
		}

		files, err := ParseBinaryManifestWithKey(data, hexKey)
		if err == nil && len(files) > 0 {
			for idx := range files {
				files[idx].DepotID = dID
			}
			allFiles = append(allFiles, files...)
		}
	}

	if len(allFiles) == 0 {
		return nil, fmt.Errorf("no valid manifest file entries loaded from %s", manifestsDir)
	}

	return &Manifest{
		AppID:     appID,
		DepotID:   resolvedDepotID,
		DepotKeys: depotKeys,
		Files:     allFiles,
	}, nil
}
