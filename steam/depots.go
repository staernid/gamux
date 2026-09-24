package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/staernid/gamux/cache"
)

// SteamDepotInfo captures structured metadata about a Steam content depot.
type SteamDepotInfo struct {
	DepotID  uint32 `json:"depot_id"`
	Name     string `json:"name,omitempty"`
	Language string `json:"language,omitempty"`
	OSList   string `json:"oslist,omitempty"`
	IsDLC    bool   `json:"is_dlc,omitempty"`
	Optional bool   `json:"optional,omitempty"`
}

type steamcmdDepotEntry struct {
	Name   string `json:"name"`
	Config struct {
		Language    string `json:"language"`
		OSList      string `json:"oslist"`
		OSArch      string `json:"osarch"`
		OptionalDLC string `json:"optionaldlc"`
	} `json:"config"`
	Manifests map[string]any `json:"manifests"`
}

// FetchAppDepotMetadata queries SteamCMD or cache for official Steam depot definitions for an AppID.
func FetchAppDepotMetadata(ctx context.Context, appID uint32) (map[uint32]SteamDepotInfo, error) {
	if appID == 0 {
		return nil, fmt.Errorf("invalid Steam AppID 0")
	}

	// 0. Cache check
	if cachedBytes, ok := cache.GetSteamDepots(appID); ok && len(cachedBytes) > 0 {
		var cachedMap map[uint32]SteamDepotInfo
		if err := json.Unmarshal(cachedBytes, &cachedMap); err == nil && len(cachedMap) > 0 {
			return cachedMap, nil
		}
	}

	apiURL := fmt.Sprintf("%s/%d", DefaultSteamCMDBaseURL, appID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create steamcmd request: %w", err)
	}
	req.Header.Set("User-Agent", "Gamux/1.0 (Linux; Steam Game Manager)")

	resp, err := HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch steamcmd depots: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steamcmd appinfo HTTP %d", resp.StatusCode)
	}

	var rawResponse struct {
		Status string `json:"status"`
		Data   map[string]struct {
			Depots map[string]json.RawMessage `json:"depots"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResponse); err != nil {
		return nil, fmt.Errorf("decode steamcmd appinfo JSON: %w", err)
	}

	appKey := fmt.Sprintf("%d", appID)
	appData, ok := rawResponse.Data[appKey]
	if !ok || len(appData.Depots) == 0 {
		return make(map[uint32]SteamDepotInfo), nil
	}

	result := make(map[uint32]SteamDepotInfo)

	for key, rawMsg := range appData.Depots {
		if key == "branches" {
			continue
		}
		did64, err := strconv.ParseUint(key, 10, 32)
		if err != nil || did64 == 0 {
			continue
		}
		dID := uint32(did64)

		var entry steamcmdDepotEntry
		if err := json.Unmarshal(rawMsg, &entry); err != nil {
			continue
		}

		lang := strings.ToLower(strings.TrimSpace(entry.Config.Language))
		name := strings.TrimSpace(entry.Name)
		isDLC := entry.Config.OptionalDLC != ""
		isOptional := isDLC || strings.Contains(strings.ToLower(name), "soundtrack") ||
			strings.Contains(strings.ToLower(name), "artbook") ||
			strings.Contains(strings.ToLower(name), "bonus") ||
			strings.Contains(strings.ToLower(name), "high resolution") ||
			strings.Contains(strings.ToLower(name), "4k")

		result[dID] = SteamDepotInfo{
			DepotID:  dID,
			Name:     name,
			Language: lang,
			OSList:   strings.ToLower(strings.TrimSpace(entry.Config.OSList)),
			IsDLC:    isDLC,
			Optional: isOptional,
		}
	}

	if len(result) > 0 {
		if raw, mErr := json.Marshal(result); mErr == nil {
			cache.SaveSteamDepots(appID, raw)
		}
	}

	slog.Info("Resolved Steam depot metadata", "appID", appID, "depots", len(result))
	return result, nil
}
