package manifest

import (
	"context"
	"fmt"
	"os"
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
	var trace []string

	// 1. Local .lua file override
	if luaPath != "" {
		content, err := os.ReadFile(luaPath)
		if err != nil {
			return nil, fmt.Errorf("read local lua file %s: %w", luaPath, err)
		}
		parsed, err := ParseLua(string(content))
		if err != nil {
			return nil, err
		}
		parsed.AuditTrace = append(parsed.AuditTrace, fmt.Sprintf("Local Lua File override ('%s'): %d depot keys, %d manifest files", luaPath, len(parsed.Depots), len(parsed.ManifestFiles)))
		return parsed, nil
	} else {
		trace = append(trace, "Local Lua File: Skipped (no --lua file specified)")
	}

	if appID == 0 {
		return nil, fmt.Errorf("no AppID provided and no --lua key file specified")
	}

	var bestResult *ParsedLua
	var errs []string

	// 2. RevoBD Public API
	if parsed, err := FetchRevoBDKeys(ctx, appID); err == nil && parsed != nil && len(parsed.Depots) > 0 {
		trace = append(trace, fmt.Sprintf("RevoBD API (luagen.revobd.club): Found %d depot keys, %d .manifest files in ZIP", len(parsed.Depots), len(parsed.ManifestFiles)))
		bestResult = parsed
	} else if err != nil {
		trace = append(trace, fmt.Sprintf("RevoBD API (luagen.revobd.club): %v", err))
		errs = append(errs, fmt.Sprintf("RevoBD: %v", err))
	}

	// 3. Steam-Depot GitHub DB Fallback
	if bestResult == nil {
		if parsed, err := FetchSteamDepotDBKeys(ctx, appID); err == nil && parsed != nil && len(parsed.Depots) > 0 {
			trace = append(trace, fmt.Sprintf("Steam-Depot DB (KoriaPolis/Steam-Depot): Found %d depot keys, %d .manifest files", len(parsed.Depots), len(parsed.ManifestFiles)))
			bestResult = parsed
		} else if err != nil {
			trace = append(trace, fmt.Sprintf("Steam-Depot DB: %v", err))
			errs = append(errs, fmt.Sprintf("Steam-Depot: %v", err))
		}
	} else {
		trace = append(trace, "Steam-Depot DB: Skipped (already resolved keys from RevoBD)")
	}

	// 4. Hubcap API (Always query Hubcap if key is provided and no .manifest files have been obtained yet!)
	if hubcapAPIKey != "" {
		maskedKey := hubcapAPIKey
		if len(maskedKey) > 8 {
			maskedKey = maskedKey[:4] + "..." + maskedKey[len(maskedKey)-4:]
		}
		if bestResult == nil || len(bestResult.ManifestFiles) == 0 {
			if parsed, err := FetchHubcapKeys(ctx, appID, hubcapAPIKey); err == nil && parsed != nil && len(parsed.Depots) > 0 {
				trace = append(trace, fmt.Sprintf("Hubcap API (key '%s'): Found %d depot keys, %d .manifest files", maskedKey, len(parsed.Depots), len(parsed.ManifestFiles)))
				if bestResult == nil {
					bestResult = parsed
				} else if len(parsed.ManifestFiles) > 0 {
					if bestResult.ManifestFiles == nil {
						bestResult.ManifestFiles = make(map[string][]byte)
					}
					for k, v := range parsed.ManifestFiles {
						bestResult.ManifestFiles[k] = v
					}
				}
			} else if err != nil {
				trace = append(trace, fmt.Sprintf("Hubcap API (key '%s'): %v", maskedKey, err))
				errs = append(errs, fmt.Sprintf("Hubcap: %v", err))
			} else {
				trace = append(trace, fmt.Sprintf("Hubcap API (key '%s'): Returned 0 depot keys", maskedKey))
			}
		} else {
			trace = append(trace, fmt.Sprintf("Hubcap API (key '%s'): Skipped (already obtained .manifest files from RevoBD)", maskedKey))
		}
	} else {
		trace = append(trace, "Hubcap API: Skipped (no hubcap_api_key configured in ~/.config/gamux/config.json)")
	}

	if bestResult != nil {
		bestResult.AuditTrace = trace
		return bestResult, nil
	}

	return nil, fmt.Errorf("failed to resolve depot keys for AppID %d via provider chain: %s", appID, strings.Join(errs, " | "))
}
