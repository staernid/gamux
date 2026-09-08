package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/domain"
)

// DefaultSteamCMDBaseURL is the base URL for community steamcmd appinfo queries.
const DefaultSteamCMDBaseURL = "https://api.steamcmd.net/v1/info"

type steamcmdBranchEntry struct {
	BuildID          any    `json:"buildid"`
	Description      string `json:"description"`
	TimeUpdated      any    `json:"timeupdated"`
	TimeBuildUpdated any    `json:"timebuildupdated"`
	PwdRequired      any    `json:"pwdrequired"`
}

type steamcmdResponse struct {
	Status string `json:"status"`
	Data   map[string]struct {
		Depots struct {
			Branches map[string]steamcmdBranchEntry `json:"branches"`
		} `json:"depots"`
	} `json:"data"`
}

func parseBuildID(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return strconv.Itoa(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func parseTimestamp(val any) int64 {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case string:
		trimmed := strings.TrimSpace(v)
		if n, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return n
		}
	}
	return 0
}

func parsePwdRequired(val any) bool {
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case float64:
		return v == 1
	case int:
		return v == 1
	case string:
		trimmed := strings.ToLower(strings.TrimSpace(v))
		return trimmed == "1" || trimmed == "true" || trimmed == "yes"
	}
	return false
}

// FetchAppBranches retrieves official and beta branches for a Steam AppID.
// It checks local memory and disk caches first, falling back to querying the
// public steamcmd appinfo service.
func FetchAppBranches(ctx context.Context, appID uint32) ([]domain.SteamBranch, error) {
	if appID == 0 {
		return nil, fmt.Errorf("invalid Steam AppID 0")
	}

	// 0. Cache check
	if cachedBytes, ok := cache.GetSteamBranches(appID); ok && len(cachedBytes) > 0 {
		var branches []domain.SteamBranch
		if err := json.Unmarshal(cachedBytes, &branches); err == nil && len(branches) > 0 {
			return branches, nil
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
		return nil, fmt.Errorf("fetch steamcmd branches: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steamcmd appinfo HTTP %d", resp.StatusCode)
	}

	var res steamcmdResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode steamcmd appinfo JSON: %w", err)
	}

	appKey := fmt.Sprintf("%d", appID)
	appData, ok := res.Data[appKey]
	if !ok || len(appData.Depots.Branches) == 0 {
		// If app data or branches aren't returned, fallback to standard public branch
		slog.Warn("No branch data found in steamcmd response; using default public branch", "appID", appID)
		defaultBranches := []domain.SteamBranch{
			{
				Name:        "public",
				Description: "Default / Public release",
			},
		}
		if raw, mErr := json.Marshal(defaultBranches); mErr == nil {
			cache.SaveSteamBranches(appID, raw)
		}
		return defaultBranches, nil
	}

	var branches []domain.SteamBranch
	var hasPublic bool

	for name, b := range appData.Depots.Branches {
		branchName := strings.TrimSpace(name)
		if branchName == "" {
			continue
		}
		if branchName == "public" {
			hasPublic = true
		}

		timeUpdated := parseTimestamp(b.TimeUpdated)
		if timeUpdated == 0 {
			timeUpdated = parseTimestamp(b.TimeBuildUpdated)
		}

		branches = append(branches, domain.SteamBranch{
			Name:        branchName,
			Description: strings.TrimSpace(b.Description),
			BuildID:     parseBuildID(b.BuildID),
			TimeUpdated: timeUpdated,
			PwdRequired: parsePwdRequired(b.PwdRequired),
		})
	}

	if !hasPublic {
		branches = append(branches, domain.SteamBranch{
			Name:        "public",
			Description: "Default / Public release",
		})
	}

	// Sort branches: "public" always first, then alphabetical by Name
	sort.Slice(branches, func(i, j int) bool {
		if branches[i].Name == "public" {
			return true
		}
		if branches[j].Name == "public" {
			return false
		}
		return strings.ToLower(branches[i].Name) < strings.ToLower(branches[j].Name)
	})

	if raw, mErr := json.Marshal(branches); mErr == nil {
		cache.SaveSteamBranches(appID, raw)
	}

	slog.Info("Resolved Steam branches", "appID", appID, "count", len(branches))
	return branches, nil
}
