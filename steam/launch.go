package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/util"
)

type steamcmdLaunchEntry struct {
	Executable     string            `json:"executable"`
	Arguments      string            `json:"arguments"`
	Description    string            `json:"description"`
	DescriptionLoc map[string]string `json:"description_loc"`
	Type           string            `json:"type"`
	Config         struct {
		OSList  string `json:"oslist"`
		OSArch  string `json:"osarch"`
		BetaKey string `json:"betakey"`
	} `json:"config"`
}

// FetchAppLaunchOptions retrieves official Steam launch targets for an AppID from SteamCMD AppInfo or cache.
func FetchAppLaunchOptions(ctx context.Context, appID uint32, platform string, branch string) ([]domain.LaunchCandidate, error) {
	if appID == 0 {
		return nil, fmt.Errorf("invalid Steam AppID 0")
	}

	// 0. Cache check
	if cachedBytes, ok := cache.GetSteamLaunch(appID); ok && len(cachedBytes) > 0 {
		var cachedAll []steamcmdLaunchEntry
		if err := json.Unmarshal(cachedBytes, &cachedAll); err == nil && len(cachedAll) > 0 {
			return filterAndBuildLaunchCandidates(cachedAll, platform, branch), nil
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
		return nil, fmt.Errorf("fetch steamcmd launch options: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steamcmd appinfo HTTP %d", resp.StatusCode)
	}

	var rawResponse struct {
		Status string `json:"status"`
		Data   map[string]struct {
			Config struct {
				Launch map[string]steamcmdLaunchEntry `json:"launch"`
			} `json:"config"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResponse); err != nil {
		return nil, fmt.Errorf("decode steamcmd appinfo JSON: %w", err)
	}

	appKey := fmt.Sprintf("%d", appID)
	appData, ok := rawResponse.Data[appKey]
	if !ok || len(appData.Config.Launch) == 0 {
		return nil, nil
	}

	// Sort raw entries by their key (e.g. "0", "1", "2") to preserve Steam's natural indexing
	keys := make([]string, 0, len(appData.Config.Launch))
	for k := range appData.Config.Launch {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		idxI, errI := strconv.Atoi(keys[i])
		idxJ, errJ := strconv.Atoi(keys[j])
		if errI == nil && errJ == nil {
			return idxI < idxJ
		}
		return keys[i] < keys[j]
	})

	var allEntries []steamcmdLaunchEntry
	for _, k := range keys {
		entry := appData.Config.Launch[k]
		if strings.TrimSpace(entry.Executable) != "" {
			allEntries = append(allEntries, entry)
		}
	}

	if raw, mErr := json.Marshal(allEntries); mErr == nil {
		cache.SaveSteamLaunch(appID, raw)
	}

	candidates := filterAndBuildLaunchCandidates(allEntries, platform, branch)
	slog.Info("Resolved Steam launch candidates", "appID", appID, "platform", platform, "count", len(candidates))
	return candidates, nil
}

// filterAndBuildLaunchCandidates filters and formats raw Steam launch entries into domain.LaunchCandidate items.
func filterAndBuildLaunchCandidates(entries []steamcmdLaunchEntry, platform string, branch string) []domain.LaunchCandidate {
	if len(entries) == 0 {
		return nil
	}

	platLower := strings.ToLower(strings.TrimSpace(platform))
	var targetOS string
	switch {
	case strings.Contains(platLower, "win"):
		targetOS = "windows"
	case strings.Contains(platLower, "linux"):
		targetOS = "linux"
	case strings.Contains(platLower, "mac") || strings.Contains(platLower, "darwin"):
		targetOS = "macos"
	default:
		targetOS = "windows"
	}

	branchClean := strings.ToLower(strings.TrimSpace(branch))
	if branchClean == "" {
		branchClean = "public"
	}

	type indexedCand struct {
		candidate domain.LaunchCandidate
		isDefault bool
		rawIndex  int
	}

	var matched []indexedCand

	for idx, entry := range entries {
		exe := util.NormalizeRelativePath(entry.Executable)
		if exe == "" {
			continue
		}

		// Filter out crash report clients, unity handlers, loaders
		lowerExe := strings.ToLower(filepath.Base(exe))
		if strings.Contains(lowerExe, "crashreport") ||
			strings.Contains(lowerExe, "unitycrashhandler") ||
			strings.Contains(lowerExe, "steamclient_loader") ||
			strings.Contains(lowerExe, "coldclientloader") ||
			strings.Contains(lowerExe, "steamless") {
			continue
		}

		// Check OS list
		if entry.Config.OSList != "" {
			osList := strings.ToLower(entry.Config.OSList)
			// oslist can be comma/space-separated e.g. "windows,linux"
			if !strings.Contains(osList, targetOS) && !strings.Contains(osList, "all") {
				continue
			}
		}

		// Check BetaKey:
		// If entry specifies a betakey, only include it when matching branchClean.
		// If branchClean != "public", prefer betakey-specific options; if entry has no betakey, only include for "public".
		entryBranch := strings.ToLower(strings.TrimSpace(entry.Config.BetaKey))
		if entryBranch == "" {
			entryBranch = "public"
		}
		if entryBranch != branchClean {
			continue
		}

		// Resolve Description
		desc := strings.TrimSpace(entry.Description)
		if desc == "" && len(entry.DescriptionLoc) > 0 {
			if eng, ok := entry.DescriptionLoc["english"]; ok && strings.TrimSpace(eng) != "" {
				desc = strings.TrimSpace(eng)
			}
		}

		isDefault := strings.EqualFold(entry.Type, "default") || idx == 0

		// Generate human-friendly Name
		var name string
		if desc != "" {
			if strings.HasPrefix(strings.ToLower(desc), "play ") {
				name = desc
			} else {
				name = fmt.Sprintf("Play (%s)", desc)
			}
		} else {
			if isDefault {
				name = "Play"
			} else {
				base := filepath.Base(exe)
				clean := strings.TrimSuffix(base, filepath.Ext(base))
				name = fmt.Sprintf("Play (%s)", clean)
			}
		}

		matched = append(matched, indexedCand{
			candidate: domain.LaunchCandidate{
				Name:        name,
				Executable:  exe,
				Arguments:   strings.TrimSpace(entry.Arguments),
				Description: desc,
			},
			isDefault: isDefault,
			rawIndex:  idx,
		})
	}

	// Sort default option first, then preserve raw indexing order
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].isDefault != matched[j].isDefault {
			return matched[i].isDefault
		}
		return matched[i].rawIndex < matched[j].rawIndex
	})

	result := make([]domain.LaunchCandidate, 0, len(matched))
	for _, m := range matched {
		result = append(result, m.candidate)
	}

	return result
}
