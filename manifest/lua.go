package manifest

import (
	"context"
	"fmt"
	"regexp"
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
	addAppIDKeyRegex = regexp.MustCompile(`(?m)^\s*addappid\s*\(\s*(\d+)\s*,\s*\d\s*,\s*(?:"|')([a-fA-F0-9]+)(?:"|')\s*\)`)
	// Matches addappid(12345)
	addAppIDNoKeyRegex = regexp.MustCompile(`(?m)^\s*addappid\s*\(\s*(\d+)\s*\)`)
	// General addappid match to detect AppID
	generalAddAppIDRegex = regexp.MustCompile(`(?m)^\s*addappid\s*\(\s*(\d+)`)
)

// ParseLua parses the raw text contents of a .lua file and returns structured depot keys.
func ParseLua(content string) (*ParsedLua, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("empty lua content")
	}

	genMatch := generalAddAppIDRegex.FindStringSubmatch(content)
	if len(genMatch) < 2 {
		return nil, fmt.Errorf("could not find valid addappid(...) entry in lua content")
	}

	appIDVal, err := strconv.ParseUint(genMatch[1], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid appid %s: %w", genMatch[1], err)
	}

	depotMap := make(map[uint32]string)

	keyMatches := addAppIDKeyRegex.FindAllStringSubmatch(content, -1)
	for _, m := range keyMatches {
		depotID, err := strconv.ParseUint(m[1], 10, 32)
		if err == nil {
			depotMap[uint32(depotID)] = strings.TrimSpace(m[2])
		}
	}

	noKeyMatches := addAppIDNoKeyRegex.FindAllStringSubmatch(content, -1)
	for _, m := range noKeyMatches {
		depotID, err := strconv.ParseUint(m[1], 10, 32)
		if err == nil {
			if _, exists := depotMap[uint32(depotID)]; !exists {
				depotMap[uint32(depotID)] = ""
			}
		}
	}

	depots := make([]DepotKeyPair, 0, len(depotMap))
	for depotID, key := range depotMap {
		depots = append(depots, DepotKeyPair{
			DepotID:       depotID,
			DecryptionKey: key,
		})
	}

	return &ParsedLua{
		AppID:  uint32(appIDVal),
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
