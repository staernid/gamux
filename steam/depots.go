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
	DepotID       uint32            `json:"depot_id"`
	Name          string            `json:"name,omitempty"`
	Language      string            `json:"language,omitempty"`
	OSList        string            `json:"oslist,omitempty"`
	OSArch        string            `json:"osarch,omitempty"`
	IsDLC         bool              `json:"is_dlc,omitempty"`
	DLCAppID      uint32            `json:"dlc_app_id,omitempty"`
	Optional      bool              `json:"optional,omitempty"`
	Realm         string            `json:"realm,omitempty"`
	Size          uint64            `json:"size,omitempty"`
	DownloadSize  uint64            `json:"download_size,omitempty"`
	SharedInstall bool              `json:"shared_install,omitempty"`
	DepotFromApp  uint32            `json:"depot_from_app,omitempty"`
	ManifestGIDs  map[string]string `json:"manifest_gids,omitempty"` // branchName -> manifestGID
}

type steamcmdDepotEntry struct {
	Name          string `json:"name"`
	DLCAppID      any    `json:"dlcappid"`
	DepotFromApp  any    `json:"depotfromapp"`
	SharedInstall any    `json:"sharedinstall"`
	Config        struct {
		Language    string `json:"language"`
		OSList      string `json:"oslist"`
		OSArch      string `json:"osarch"`
		OptionalDLC any    `json:"optionaldlc"`
		Realm       string `json:"realm"`
	} `json:"config"`
	Manifests map[string]any `json:"manifests"`
}

func parseManifestGID(val any) string {
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		if gidVal, ok := v["gid"]; ok {
			return parseManifestGID(gidVal)
		}
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case json.Number:
		return v.String()
	}
	return ""
}

func parseUint64(val any) uint64 {
	switch v := val.(type) {
	case uint64:
		return v
	case int64:
		if v > 0 {
			return uint64(v)
		}
	case float64:
		if v > 0 {
			return uint64(v)
		}
	case string:
		if n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64); err == nil {
			return n
		}
	case json.Number:
		if n, err := v.Int64(); err == nil && n > 0 {
			return uint64(n)
		}
	}
	return 0
}

func extractDepotSizes(manifests map[string]any) (size uint64, download uint64) {
	if len(manifests) == 0 {
		return 0, 0
	}
	// 1. Prefer "public" branch first
	if pubVal, ok := manifests["public"]; ok {
		if pubMap, ok := pubVal.(map[string]any); ok {
			size = parseUint64(pubMap["size"])
			download = parseUint64(pubMap["download"])
			if size > 0 || download > 0 {
				return size, download
			}
		}
	}
	// 2. Fallback to largest branch manifest size
	for _, bVal := range manifests {
		if bMap, ok := bVal.(map[string]any); ok {
			s := parseUint64(bMap["size"])
			d := parseUint64(bMap["download"])
			if s > size {
				size = s
			}
			if d > download {
				download = d
			}
		}
	}
	return size, download
}

// SynthesizeDepotName produces a clean, human-readable depot description based on its metadata.
func SynthesizeDepotName(depotID uint32, osList, osArch, lang string, isDLC bool, dlcAppID uint32, sharedInstall bool, depotFromApp uint32) string {
	if sharedInstall || depotFromApp > 0 {
		if depotFromApp > 0 {
			return fmt.Sprintf("Shared Redistributables (App %d)", depotFromApp)
		}
		return "Steam Common Redistributables"
	}

	if lang != "" {
		normLang := strings.Title(strings.ToLower(lang))
		return fmt.Sprintf("%s Language Pack (%d)", normLang, depotID)
	}

	archStr := ""
	if osArch == "64" {
		archStr = " (64-bit)"
	} else if osArch == "32" {
		archStr = " (32-bit)"
	}

	prefix := ""
	if isDLC && dlcAppID > 0 {
		prefix = fmt.Sprintf("DLC #%d ", dlcAppID)
	}

	switch strings.ToLower(osList) {
	case "windows", "win":
		return fmt.Sprintf("%sWindows Binaries%s (%d)", prefix, archStr, depotID)
	case "linux":
		return fmt.Sprintf("%sLinux Native%s (%d)", prefix, archStr, depotID)
	case "macos", "osx":
		return fmt.Sprintf("%smacOS Bundle%s (%d)", prefix, archStr, depotID)
	case "android":
		return fmt.Sprintf("%sAndroid Content%s (%d)", prefix, archStr, depotID)
	default:
		if isDLC {
			return fmt.Sprintf("%sContent (%d)", prefix, depotID)
		}
		return fmt.Sprintf("Base Content (%d)", depotID)
	}
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

	return fetchAppDepotMetadataRemote(ctx, appID)
}

func fetchAppDepotMetadataRemote(ctx context.Context, appID uint32) (map[uint32]SteamDepotInfo, error) {
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
			Common struct {
				Name   string `json:"name"`
				Parent string `json:"parent"`
				Type   string `json:"type"`
			} `json:"common"`
			Extended struct {
				DLCForAppID string `json:"dlcforappid"`
				ListOfDLC   string `json:"listofdlc"`
			} `json:"extended"`
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

	var parentAppID uint32
	if pStr := strings.TrimSpace(appData.Common.Parent); pStr != "" {
		if p, err := strconv.ParseUint(pStr, 10, 32); err == nil && uint32(p) != appID {
			parentAppID = uint32(p)
		}
	}
	if parentAppID == 0 {
		if pStr := strings.TrimSpace(appData.Extended.DLCForAppID); pStr != "" {
			if p, err := strconv.ParseUint(pStr, 10, 32); err == nil && uint32(p) != appID {
				parentAppID = uint32(p)
			}
		}
	}

	// If this app is a DLC with a parent app, query parent app depots to find DLC depots
	// with full manifests and sizes, because Steam PICS stores DLC manifests in the parent game's depots list.
	if parentAppID > 0 {
		if parentDepots, pErr := FetchAppDepotMetadata(ctx, parentAppID); pErr == nil && len(parentDepots) > 0 {
			appName := strings.TrimSpace(appData.Common.Name)
			for dID, dInfo := range parentDepots {
				if dInfo.DLCAppID == appID {
					if appName != "" && (dInfo.Name == "" || strings.HasPrefix(dInfo.Name, "Depot ") || strings.HasPrefix(dInfo.Name, "DLC #")) {
						archStr := ""
						if dInfo.OSArch == "64" {
							archStr = " (64-bit)"
						} else if dInfo.OSArch == "32" {
							archStr = " (32-bit)"
						}
						switch strings.ToLower(dInfo.OSList) {
						case "windows", "win":
							dInfo.Name = fmt.Sprintf("%s - Windows%s (%d)", appName, archStr, dID)
						case "linux":
							dInfo.Name = fmt.Sprintf("%s - Linux%s (%d)", appName, archStr, dID)
						case "macos", "osx":
							dInfo.Name = fmt.Sprintf("%s - macOS%s (%d)", appName, archStr, dID)
						default:
							dInfo.Name = fmt.Sprintf("%s - Content (%d)", appName, dID)
						}
					}
					result[dID] = dInfo
				}
			}
		}
	}

	for key, rawMsg := range appData.Depots {
		if key == "branches" || key == "baselanguages" || key == "hasdepotsindlc" || key == "privatebranches" {
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

		realm := strings.ToLower(strings.TrimSpace(entry.Config.Realm))
		if realm == "steamchina" {
			slog.Debug("Skipping Steam China exclusive depot", "appID", appID, "depotID", dID)
			continue
		}

		lang := strings.ToLower(strings.TrimSpace(entry.Config.Language))
		osList := strings.ToLower(strings.TrimSpace(entry.Config.OSList))
		osArch := strings.TrimSpace(entry.Config.OSArch)

		dlcID := uint32(parseUint64(entry.DLCAppID))
		if dlcID == 0 && entry.Config.OptionalDLC != nil {
			dlcID = uint32(parseUint64(entry.Config.OptionalDLC))
		}
		isDLC := dlcID > 0 || entry.Config.OptionalDLC != nil || strings.EqualFold(appData.Common.Type, "dlc")
		if isDLC && dlcID == 0 && parentAppID > 0 {
			dlcID = appID
		}

		depotFromApp := uint32(parseUint64(entry.DepotFromApp))
		sharedInstall := parseUint64(entry.SharedInstall) > 0

		name := strings.TrimSpace(entry.Name)
		isOptional := isDLC || strings.Contains(strings.ToLower(name), "soundtrack") ||
			strings.Contains(strings.ToLower(name), "artbook") ||
			strings.Contains(strings.ToLower(name), "bonus") ||
			strings.Contains(strings.ToLower(name), "high resolution") ||
			strings.Contains(strings.ToLower(name), "4k")

		size, download := extractDepotSizes(entry.Manifests)

		manifestGIDs := make(map[string]string)
		for bName, bVal := range entry.Manifests {
			if gid := parseManifestGID(bVal); gid != "" {
				manifestGIDs[bName] = gid
			}
		}

		if name == "" {
			name = SynthesizeDepotName(dID, osList, osArch, lang, isDLC, dlcID, sharedInstall, depotFromApp)
		}

		// Don't overwrite higher-fidelity depot info if already populated from parent
		if existing, exists := result[dID]; exists {
			if existing.Size > 0 && size == 0 {
				continue
			}
		}

		result[dID] = SteamDepotInfo{
			DepotID:       dID,
			Name:          name,
			Language:      lang,
			OSList:        osList,
			OSArch:        osArch,
			IsDLC:         isDLC,
			DLCAppID:      dlcID,
			Optional:      isOptional,
			Realm:         realm,
			Size:          size,
			DownloadSize:  download,
			SharedInstall: sharedInstall,
			DepotFromApp:  depotFromApp,
			ManifestGIDs:  manifestGIDs,
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

// FetchAppBranchManifestGIDs queries Steam depot metadata and extracts the manifest GIDs
// corresponding to the target branch. If a depot lacks an entry for the specified branch,
// it gracefully falls back to "public" if available.
func FetchAppBranchManifestGIDs(ctx context.Context, appID uint32, branch string) (map[uint32]string, error) {
	if branch == "" {
		branch = "public"
	}
	depots, err := FetchAppDepotMetadata(ctx, appID)
	if err != nil {
		return nil, err
	}

	// Check if any depot in cached/returned depots has ManifestGIDs
	hasGIDs := false
	for _, d := range depots {
		if len(d.ManifestGIDs) > 0 {
			hasGIDs = true
			break
		}
	}

	// If no depot had ManifestGIDs (e.g. from an older cache format), re-fetch from remote
	if !hasGIDs && len(depots) > 0 {
		if freshDepots, fErr := fetchAppDepotMetadataRemote(ctx, appID); fErr == nil && len(freshDepots) > 0 {
			depots = freshDepots
		}
	}

	res := make(map[uint32]string)
	for dID, dInfo := range depots {
		if strings.EqualFold(dInfo.Realm, "steamchina") {
			continue
		}
		if gid, ok := dInfo.ManifestGIDs[branch]; ok && gid != "" {
			res[dID] = gid
		} else if branch != "public" {
			if publicGID, ok := dInfo.ManifestGIDs["public"]; ok && publicGID != "" {
				res[dID] = publicGID
			}
		}
	}
	return res, nil
}
