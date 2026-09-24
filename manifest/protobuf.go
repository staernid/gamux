package manifest

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
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
	return LoadManifestFromDirWithPlatform(targetDir, appID, nil, "")
}

// DepotFilter defines criteria for including or excluding depots during manifest loading and chunk streaming.
type DepotFilter struct {
	Platform        string               `json:"platform,omitempty"`
	Language        string               `json:"language,omitempty"`
	IncludedDepots  []uint32             `json:"included_depots,omitempty"`
	ExcludedDepots  []uint32             `json:"excluded_depots,omitempty"`
	IncludeOptional bool                 `json:"include_optional,omitempty"`
	DepotMetadata   map[uint32]DepotMeta `json:"-"`
}

// ShouldIncludeDepot tests whether a content depot matches the filtering criteria.
func ShouldIncludeDepot(depotID uint32, files []ManifestFileEntry, filter DepotFilter) bool {
	// 1. Explicit exclusions
	for _, id := range filter.ExcludedDepots {
		if id == depotID {
			return false
		}
	}

	// 2. Explicit inclusions override all category/language heuristics
	for _, id := range filter.IncludedDepots {
		if id == depotID {
			return true
		}
	}
	if len(filter.IncludedDepots) > 0 {
		return false
	}

	// 3. Platform matching
	if filter.Platform != "" && filter.Platform != "all" {
		cat := ClassifyDepot(files)
		if !MatchesPlatform(cat, filter.Platform) {
			return false
		}
	}

	// 4. Content Category & Language filtering
	var meta *DepotMeta
	if filter.DepotMetadata != nil {
		if m, ok := filter.DepotMetadata[depotID]; ok {
			meta = &m
		}
	}

	contentCat, lang := ClassifyDepotContent(depotID, "", files, meta)

	// Soundtracks are skipped unless explicitly requested via platform or include_optional
	if contentCat == ContentCategorySoundtrack {
		targetPlat := strings.ToLower(strings.TrimSpace(filter.Platform))
		if targetPlat == "soundtrack" || targetPlat == "music" || filter.IncludeOptional {
			return true
		}
		return false
	}

	// Optional HD assets
	if contentCat == ContentCategoryOptionalHD {
		return filter.IncludeOptional
	}

	// Localized voice / video language packs
	if strings.HasPrefix(contentCat, ContentCategoryLocalePrefix) {
		prefLang := filter.Language
		if prefLang == "" {
			prefLang = "english"
		}
		return LanguagesMatch(lang, prefLang)
	}

	// Core depots are always included
	return true
}

func LoadManifestFromDirWithKeys(targetDir string, appID uint32, depotKeys map[uint32]string) (*Manifest, error) {
	return LoadManifestFromDirWithPlatform(targetDir, appID, depotKeys, "")
}

// LoadManifestFromDirWithPlatform loads depot manifests matching a specific target platform from <targetDir>/[Manifests].
func LoadManifestFromDirWithPlatform(targetDir string, appID uint32, depotKeys map[uint32]string, platform string) (*Manifest, error) {
	return LoadManifestFromDirWithFilter(targetDir, appID, depotKeys, DepotFilter{Platform: platform})
}

// LoadManifestFromDirWithFilter loads depot manifests matching a filter from <targetDir>/[Manifests].
func LoadManifestFromDirWithFilter(targetDir string, appID uint32, depotKeys map[uint32]string, filter DepotFilter) (*Manifest, error) {
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

	depotManifests := make(map[uint32][]manifestFileInfo)

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
			depotManifests[fileDepotID] = append(depotManifests[fileDepotID], manifestFileInfo{
				name:       name,
				depotID:    fileDepotID,
				manifestID: fileManifestID,
				path:       manifestPath,
			})
		}
	}

	// Sort candidates for each depot by manifestID descending (newest first)
	for dID := range depotManifests {
		sort.Slice(depotManifests[dID], func(i, j int) bool {
			return depotManifests[dID][i].manifestID > depotManifests[dID][j].manifestID
		})
	}

	allFiles := make([]ManifestFileEntry, 0)
	var resolvedDepotID uint32

	for dID, candidates := range depotManifests {
		var validFiles []ManifestFileEntry
		hexKey := ""
		if depotKeys != nil {
			hexKey = depotKeys[dID]
		}
		for _, info := range candidates {
			data, err := os.ReadFile(info.path)
			if err != nil {
				slog.Warn("Failed to read manifest file", "path", info.path, "error", err)
				continue
			}
			files, err := ParseBinaryManifestWithKey(data, hexKey)
			if err != nil {
				slog.Warn("Failed to parse local manifest file, trying fallback candidate if available", "path", info.path, "depotID", dID, "error", err)
				continue
			}
			if len(files) > 0 {
				validFiles = files
				break
			}
		}

		if len(validFiles) > 0 {
			if !ShouldIncludeDepot(dID, validFiles, filter) {
				slog.Debug("Skipping depot manifest filtered out by options", "depotID", dID, "platform", filter.Platform, "language", filter.Language)
				continue
			}
			if resolvedDepotID == 0 {
				resolvedDepotID = dID
			}
			for idx := range validFiles {
				validFiles[idx].DepotID = dID
			}
			allFiles = append(allFiles, validFiles...)
		}
	}

	// Fallback: If strict language filter excluded all depots, retry with relaxed language filtering
	// so the user is never locked out of downloading available depots (Core Tenet 2)
	if len(allFiles) == 0 && filter.Language != "" && filter.Language != "all" {
		relaxedFilter := filter
		relaxedFilter.Language = "all"
		for dID, candidates := range depotManifests {
			var validFiles []ManifestFileEntry
			hexKey := ""
			if depotKeys != nil {
				hexKey = depotKeys[dID]
			}
			for _, info := range candidates {
				data, err := os.ReadFile(info.path)
				if err != nil {
					continue
				}
				files, err := ParseBinaryManifestWithKey(data, hexKey)
				if err == nil && len(files) > 0 {
					validFiles = files
					break
				}
			}
			if len(validFiles) > 0 {
				if !ShouldIncludeDepot(dID, validFiles, relaxedFilter) {
					continue
				}
				if resolvedDepotID == 0 {
					resolvedDepotID = dID
				}
				for idx := range validFiles {
					validFiles[idx].DepotID = dID
				}
				allFiles = append(allFiles, validFiles...)
			}
		}
		if len(allFiles) > 0 {
			slog.Info("Fallback to all language depots after language filter matched 0 depots", "depotsCount", len(depotManifests))
		}
	}

	if len(allFiles) == 0 {
		return nil, fmt.Errorf("no valid manifest file entries matching filter (platform=%q, lang=%q) loaded from %s", filter.Platform, filter.Language, manifestsDir)
	}

	return &Manifest{
		AppID:     appID,
		DepotID:   resolvedDepotID,
		DepotKeys: depotKeys,
		Files:     allFiles,
	}, nil
}

// LoadPreviousManifestFromDir loads the previous (.manifest.previous) depot manifest from <targetDir>/[Manifests].
func LoadPreviousManifestFromDir(targetDir string, depotID uint32, depotKeys map[uint32]string) (*Manifest, error) {
	manifestsDir := filepath.Join(targetDir, "[Manifests]")
	entries, err := os.ReadDir(manifestsDir)
	if err != nil {
		return nil, fmt.Errorf("read manifests dir: %w", err)
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

	prefix := ""
	if depotID != 0 {
		prefix = fmt.Sprintf("%d_", depotID)
	}

	var bestPath string
	var bestManifestID uint64

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".manifest.previous") {
			continue
		}
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}

		parts := strings.Split(name, "_")
		if len(parts) >= 2 {
			midStr := strings.TrimSuffix(parts[1], ".manifest.previous")
			if mid, err := strconv.ParseUint(midStr, 10, 64); err == nil {
				if mid > bestManifestID {
					bestManifestID = mid
					bestPath = filepath.Join(manifestsDir, name)
				}
			}
		} else if bestPath == "" {
			bestPath = filepath.Join(manifestsDir, name)
		}
	}

	if bestPath == "" {
		return nil, fmt.Errorf("no previous manifest found in %s", manifestsDir)
	}

	data, err := os.ReadFile(bestPath)
	if err != nil {
		return nil, fmt.Errorf("read previous manifest %s: %w", bestPath, err)
	}

	hexKey := ""
	if depotKeys != nil {
		hexKey = depotKeys[depotID]
	}

	files, err := ParseBinaryManifestWithKey(data, hexKey)
	if err != nil {
		return nil, fmt.Errorf("parse previous binary manifest: %w", err)
	}

	for i := range files {
		files[i].DepotID = depotID
	}

	return &Manifest{
		DepotID:       depotID,
		ManifestID:    bestManifestID,
		DecryptionKey: hexKey,
		DepotKeys:     depotKeys,
		Files:         files,
	}, nil
}

