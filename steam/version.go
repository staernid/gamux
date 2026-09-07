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
	"github.com/staernid/gamux/domain"
)

type upToDateResponse struct {
	Response struct {
		Success           bool   `json:"success"`
		UpToDate          bool   `json:"up_to_date"`
		VersionIsListable bool   `json:"version_is_listable"`
		RequiredVersion   uint32 `json:"required_version"`
		Message           string `json:"message"`
		Error             string `json:"error"`
	} `json:"response"`
}

// CheckGameVersion performs a multi-tiered version verification for a Steam AppID.
// Tier 1: Queries official Steam API ISteamApps/UpToDateCheck/v1 when a numeric build ID is available.
// Tier 3: Falls back to analyzing recent store patch notes & community announcements timestamps
// against the local game installation date (with a 24-hour grace buffer).
func CheckGameVersion(ctx context.Context, appID uint32, localBuildID string, installedTime int64) (*domain.VersionStatus, error) {
	if appID == 0 {
		return nil, fmt.Errorf("invalid appID 0")
	}

	cacheKey := fmt.Sprintf("%d_%s_%d", appID, localBuildID, installedTime)

	// 0. Check cache
	if cachedBytes, ok := cache.GetVersionStatus(cacheKey); ok && len(cachedBytes) > 0 {
		var status domain.VersionStatus
		if err := json.Unmarshal(cachedBytes, &status); err == nil {
			return &status, nil
		}
	}

	saveAndReturn := func(status *domain.VersionStatus) (*domain.VersionStatus, error) {
		if raw, err := json.Marshal(status); err == nil {
			cache.SaveVersionStatus(cacheKey, raw)
		}
		return status, nil
	}

	// Tier 1: Try official ISteamApps/UpToDateCheck/v1
	cleanBuildID := strings.TrimSpace(localBuildID)
	if cleanBuildID != "" && cleanBuildID != "0" {
		if versionNum, err := strconv.ParseUint(cleanBuildID, 10, 32); err == nil && versionNum > 0 {
			apiURL := fmt.Sprintf("%s/ISteamApps/UpToDateCheck/v1/?appid=%d&version=%d", DefaultSteamWebAPI, appID, versionNum)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
			if err == nil {
				req.Header.Set("User-Agent", "Gamux/1.0 (Linux; Steam Game Manager)")
				if resp, err := HTTPClientFromContext(ctx).Do(req); err == nil {
					defer resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						var uRes upToDateResponse
						if err := json.NewDecoder(resp.Body).Decode(&uRes); err == nil && uRes.Response.Success {
							reqVerStr := ""
							if uRes.Response.RequiredVersion > 0 {
								reqVerStr = fmt.Sprintf("%d", uRes.Response.RequiredVersion)
							}
							msg := uRes.Response.Message
							if !uRes.Response.UpToDate && msg == "" && reqVerStr != "" {
								msg = fmt.Sprintf("New version available: %s", reqVerStr)
							} else if uRes.Response.UpToDate && msg == "" {
								msg = "Game is up to date"
							}
							status := &domain.VersionStatus{
								Checked:        true,
								IsUpToDate:     uRes.Response.UpToDate,
								CurrentBuildID: cleanBuildID,
								LatestBuildID:  reqVerStr,
								Message:        msg,
							}
							slog.Info("Verified game version via ISteamApps/UpToDateCheck", "appID", appID, "upToDate", status.IsUpToDate)
							return saveAndReturn(status)
						}
					}
				}
			}
		}
	}

	// Tier 3: Heuristic Patch Notes & Announcements Timestamp Check
	items, err := FetchAppNewsItems(ctx, fmt.Sprintf("%d", appID), 5)
	if err == nil && len(items) > 0 {
		var candidatePatch *domain.NewsItem
		for i := range items {
			titleLower := strings.ToLower(items[i].Title)
			feedLower := strings.ToLower(items[i].FeedLabel)
			if strings.Contains(titleLower, "patch") ||
				strings.Contains(titleLower, "update") ||
				strings.Contains(titleLower, "hotfix") ||
				strings.Contains(titleLower, "version") ||
				strings.Contains(titleLower, "release") ||
				strings.Contains(titleLower, "notes") ||
				strings.Contains(titleLower, "changelog") ||
				strings.Contains(feedLower, "community announcements") {
				candidatePatch = &items[i]
				break
			}
		}
		if candidatePatch == nil {
			candidatePatch = &items[0]
		}

		isUpToDate := true
		msg := "Game is up to date"
		// If installed timestamp exists and the latest patch was published > 24 hours after install
		const graceBufferSeconds = 86400 // 24 hours
		if installedTime > 0 && candidatePatch.Date > (installedTime+graceBufferSeconds) {
			isUpToDate = false
			msg = fmt.Sprintf("Update released: %s", candidatePatch.Title)
		}

		status := &domain.VersionStatus{
			Checked:          true,
			IsUpToDate:       isUpToDate,
			CurrentBuildID:   cleanBuildID,
			LatestPatchDate:  candidatePatch.Date,
			LatestPatchTitle: candidatePatch.Title,
			Message:          msg,
		}
		slog.Info("Verified game version via patch timestamp heuristic", "appID", appID, "upToDate", status.IsUpToDate, "patchDate", candidatePatch.Date)
		return saveAndReturn(status)
	}

	// Fallback when both tiers cannot determine version
	status := &domain.VersionStatus{
		Checked:        false,
		IsUpToDate:     true,
		CurrentBuildID: cleanBuildID,
		Message:        "Version check unavailable",
	}
	return status, nil
}
