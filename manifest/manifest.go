package manifest

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ChunkInfo represents a single file chunk in a Steam manifest.
type ChunkInfo struct {
	ChunkID      string `json:"chunk_id"`
	Offset       uint64 `json:"offset"`
	CompressedSize uint32 `json:"compressed_size"`
	UncompressedSize uint32 `json:"uncompressed_size"`
	Checksum     string `json:"checksum"`
}

// ManifestFileEntry represents a file entry in a Steam manifest.
type ManifestFileEntry struct {
	Path        string      `json:"path"`
	Size        uint64      `json:"size"`
	Flags       uint32      `json:"flags"`
	SHA1        string      `json:"sha1"`
	DepotID     uint32      `json:"depot_id,omitempty"`
	Chunks      []ChunkInfo `json:"chunks"`
}

// Manifest represents parsed manifest file details for a Steam depot.
type Manifest struct {
	AppID         uint32              `json:"app_id"`
	DepotID       uint32              `json:"depot_id"`
	ManifestID    uint64              `json:"manifest_id"`
	CreationTime  uint32              `json:"creation_time"`
	DecryptionKey string              `json:"decryption_key,omitempty"`
	DepotKeys     map[uint32]string   `json:"depot_keys,omitempty"`
	Files         []ManifestFileEntry `json:"files"`
}

// GetDecryptionKey returns the decryption key for a specific depot ID or falls back to the default key.
func (m *Manifest) GetDecryptionKey(depotID uint32) string {
	if m.DepotKeys != nil {
		if key, ok := m.DepotKeys[depotID]; ok && key != "" {
			return key
		}
	}
	return m.DecryptionKey
}

// NormalizePath cleans and normalizes relative file paths for Linux/Windows cross-platform safety.
func NormalizePath(p string) string {
	cleaned := filepath.Clean(strings.ReplaceAll(p, "\\", "/"))
	return strings.TrimPrefix(cleaned, "/")
}

// TotalSize returns the aggregate uncompressed size of all files in the manifest.
func (m *Manifest) TotalSize() uint64 {
	var total uint64
	for _, f := range m.Files {
		total += f.Size
	}
	return total
}

// FindFile returns the file entry for a given relative path.
func (m *Manifest) FindFile(relativePath string) (*ManifestFileEntry, error) {
	norm := NormalizePath(relativePath)
	for i := range m.Files {
		if NormalizePath(m.Files[i].Path) == norm {
			return &m.Files[i], nil
		}
	}
	return nil, fmt.Errorf("file %q not found in manifest", relativePath)
}
