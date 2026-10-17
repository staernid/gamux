package manifest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// DepotKeyPair represents a Steam Depot ID paired with its hex decryption key.
type DepotKeyPair struct {
	DepotID       uint32
	DecryptionKey string
}

// ParsedLua holds the parsed AppID, depot decryption keys, and optional .manifest binary files.
type ParsedLua struct {
	AppID         uint32
	Depots        []DepotKeyPair
	ManifestFiles map[string][]byte
	AuditTrace    []string
}

var (
	// Matches addappid(12345, 1, "key") or addappid(12345, 1, 'key')
	addAppIDKeyRegex = regexp.MustCompile(`(?m)^\s*addappid\s*\(\s*(\d+)\s*,\s*\d+\s*,\s*(?:"|')([^"']+)(?:"|')\s*\)`)
	// Matches addappid(12345)
	addAppIDNoKeyRegex = regexp.MustCompile(`(?m)^\s*addappid\s*\(\s*(\d+)\s*\)`)
)

// ParseLua parses the raw text contents of a .lua file and returns structured depot keys.
func ParseLua(content string) (*ParsedLua, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("empty lua content")
	}

	depotMap := make(map[uint32]string)
	var standaloneAppIDs []uint32

	noKeyMatches := addAppIDNoKeyRegex.FindAllStringSubmatch(content, -1)
	for _, m := range noKeyMatches {
		id, err := strconv.ParseUint(m[1], 10, 32)
		if err == nil {
			standaloneAppIDs = append(standaloneAppIDs, uint32(id))
			if _, exists := depotMap[uint32(id)]; !exists {
				depotMap[uint32(id)] = ""
			}
		}
	}

	keyMatches := addAppIDKeyRegex.FindAllStringSubmatch(content, -1)
	for _, m := range keyMatches {
		depotID, err := strconv.ParseUint(m[1], 10, 32)
		if err == nil {
			depotMap[uint32(depotID)] = strings.TrimSpace(m[2])
		}
	}

	if len(depotMap) == 0 && len(standaloneAppIDs) == 0 {
		return nil, fmt.Errorf("could not find valid addappid(...) entry in lua content")
	}

	var appIDVal uint32
	if len(standaloneAppIDs) > 0 {
		appIDVal = standaloneAppIDs[0]
	} else if len(keyMatches) > 0 {
		// Fallback to first key match if no standalone addappid(id) line was found
		firstKeyID, _ := strconv.ParseUint(keyMatches[0][1], 10, 32)
		appIDVal = uint32(firstKeyID)
	}

	depots := make([]DepotKeyPair, 0, len(depotMap))
	for depotID, key := range depotMap {
		depots = append(depots, DepotKeyPair{
			DepotID:       depotID,
			DecryptionKey: key,
		})
	}

	return &ParsedLua{
		AppID:  appIDVal,
		Depots: depots,
	}, nil
}

// ResolveKeys resolves manifest depot keys through a multi-provider fallback chain:
// 1. Local .lua file (if luaPath is provided)
// 2. RevoBD API (https://api.luagen.revobd.club/{appID}.zip)
// 3. Steam-Depot GitHub DB (https://raw.githubusercontent.com/KoriaPolis/Steam-Depot/main/fallback_depotkeys.json)
// 4. Hubcap API (if hubcapAPIKey is provided)
func ResolveKeys(ctx context.Context, appID uint32, luaPath string, hubcapAPIKey string) (*ParsedLua, error) {
	chain := NewDefaultProviderChain(luaPath, hubcapAPIKey)
	return chain.Resolve(ctx, appID)
}

// FormatLua serializes an AppID and depot keys into standard .lua format.
func FormatLua(appID uint32, depots []DepotKeyPair) string {
	var sb strings.Builder
	if appID > 0 {
		sb.WriteString(fmt.Sprintf("addappid(%d)\n", appID))
	}
	sorted := make([]DepotKeyPair, len(depots))
	copy(sorted, depots)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].DepotID < sorted[j].DepotID
	})
	for _, d := range sorted {
		if d.DepotID == appID && d.DecryptionKey == "" {
			continue
		}
		if d.DecryptionKey != "" {
			sb.WriteString(fmt.Sprintf("addappid(%d, 1, %q)\n", d.DepotID, d.DecryptionKey))
		} else {
			sb.WriteString(fmt.Sprintf("addappid(%d)\n", d.DepotID))
		}
	}
	return sb.String()
}

// SaveLuaFile writes an AppID and its depot keys to a .lua file on disk.
func SaveLuaFile(path string, appID uint32, depots []DepotKeyPair) error {
	content := FormatLua(appID, depots)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
