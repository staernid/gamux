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

// CheckGameVersion performs authoritative version verification for a Steam AppID.
// Tier 1: Queries official Steam API ISteamApps/UpToDateCheck/v1 when a numeric build ID is available.
// Tier 2: Queries SteamCMD AppInfo branches for the authoritative public release buildid.
// Tier 3: When local build ID is unpopulated ("0", ""), absence is treated as non-failure (AGENTS.md Core Tenet 4),
// and news/announcements are kept informational without triggering false update alerts.
func CheckGameVersion(ctx context.Context, appID uint32, localBuildID string, installedTime int64, branchOptional ...string) (*domain.VersionStatus, error) {
	if appID == 0 {
		return nil, fmt.Errorf("invalid appID 0")
	}

	targetBranchName := "public"
	if len(branchOptional) > 0 && strings.TrimSpace(branchOptional[0]) != "" {
		targetBranchName = strings.TrimSpace(branchOptional[0])
	}

	cleanManifestID := ""
	if len(branchOptional) > 1 && strings.TrimSpace(branchOptional[1]) != "" {
		cleanManifestID = strings.TrimSpace(branchOptional[1])
	}

	cleanBuildID := strings.TrimSpace(localBuildID)
	cacheKey := fmt.Sprintf("v3_%d_%s_%s_%s", appID, cleanBuildID, targetBranchName, cleanManifestID)

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

	// Case A: Unpopulated sentinel build ID ("" or "0").
	// If a local depot manifest GID is present, perform authoritative depot manifest comparison with Steam.
	// Otherwise, per AGENTS.md Core Tenet 4 ("Absence is Not an Error"), unpopulated sentinels must never
	// trigger false update alerts. We attempt to fetch the latest branch build ID and patch info
	// for informational display, but keep IsUpToDate = true and Checked = false.
	if cleanBuildID == "" || cleanBuildID == "0" {
		latestBuildID := ""
		if branches, err := FetchAppBranches(ctx, appID); err == nil && len(branches) > 0 {
			for _, b := range branches {
				if strings.EqualFold(b.Name, targetBranchName) && b.BuildID != "" {
					latestBuildID = b.BuildID
					break
				}
			}
			if latestBuildID == "" {
				for _, b := range branches {
					if b.Name == "public" && b.BuildID != "" {
						latestBuildID = b.BuildID
						break
					}
				}
			}
			if latestBuildID == "" && branches[0].BuildID != "" {
				latestBuildID = branches[0].BuildID
			}
		}

		// If a local depot manifest GID is present, perform authoritative depot manifest comparison with Steam
		if cleanManifestID != "" && cleanManifestID != "0" {
			if branchManifests, err := FetchAppBranchManifestGIDs(ctx, appID, targetBranchName); err == nil && len(branchManifests) > 0 {
				manifestMatches := false
				remoteManifestID := ""
				for _, remoteGID := range branchManifests {
					if remoteGID == cleanManifestID {
						manifestMatches = true
						remoteManifestID = remoteGID
						break
					}
				}
				if !manifestMatches {
					for _, remoteGID := range branchManifests {
						if remoteGID != "" {
							remoteManifestID = remoteGID
							break
						}
					}
				}

				status := &domain.VersionStatus{
					Checked:          true,
					IsUpToDate:       manifestMatches,
					CurrentBuildID:   cleanBuildID,
					LatestBuildID:    latestBuildID,
					RemoteManifestID: remoteManifestID,
				}
				if items, err := FetchAppNewsItems(ctx, fmt.Sprintf("%d", appID), 5); err == nil && len(items) > 0 {
					status.LatestPatchDate = items[0].Date
					status.LatestPatchTitle = items[0].Title
				}
				if manifestMatches {
					status.Message = fmt.Sprintf("Depot manifest %s is up to date", cleanManifestID)
				} else {
					status.Message = fmt.Sprintf("Depot manifest %s is outdated (latest build #%s on %s)", cleanManifestID, latestBuildID, targetBranchName)
				}
				slog.Info("Verified game version via depot manifest comparison", "appID", appID, "localManifest", cleanManifestID, "remoteManifest", remoteManifestID, "matches", manifestMatches, "latestBuild", latestBuildID)
				return saveAndReturn(status)
			}
		}

		status := &domain.VersionStatus{
			Checked:        false,
			IsUpToDate:     true,
			CurrentBuildID: cleanBuildID,
			LatestBuildID:  latestBuildID,
			Message:        "Version check unavailable (unpopulated BuildID)",
		}

		// Attach recent patch note metadata for UI display if available
		if items, err := FetchAppNewsItems(ctx, fmt.Sprintf("%d", appID), 5); err == nil && len(items) > 0 {
			status.LatestPatchDate = items[0].Date
			status.LatestPatchTitle = items[0].Title
		}

		slog.Info("Handled unpopulated build ID sentinel without false alarm", "appID", appID, "latestBuildID", latestBuildID, "branch", targetBranchName)
		return saveAndReturn(status)
	}

	// Case B: Valid local build ID is present.
	// Tier 1: Try official ISteamApps/UpToDateCheck/v1 (only applicable for default public branch)
	if targetBranchName == "public" || targetBranchName == "" {
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

	// Tier 2: Check SteamCMD AppInfo branches for target/authoritative branch buildid
	if branches, err := FetchAppBranches(ctx, appID); err == nil && len(branches) > 0 {
		var targetBranch *domain.SteamBranch
		for i := range branches {
			if strings.EqualFold(branches[i].Name, targetBranchName) && branches[i].BuildID != "" {
				targetBranch = &branches[i]
				break
			}
		}
		if targetBranch == nil && targetBranchName != "public" {
			for i := range branches {
				if branches[i].Name == "public" && branches[i].BuildID != "" {
					targetBranch = &branches[i]
					break
				}
			}
		}
		if targetBranch == nil {
			for i := range branches {
				if branches[i].BuildID != "" {
					targetBranch = &branches[i]
					break
				}
			}
		}

		if targetBranch != nil && targetBranch.BuildID != "" {
			localNum, lErr := strconv.ParseUint(cleanBuildID, 10, 64)
			remoteNum, rErr := strconv.ParseUint(targetBranch.BuildID, 10, 64)

			isUpToDate := true
			msg := "Game is up to date"

			if lErr == nil && rErr == nil {
				if localNum < remoteNum {
					isUpToDate = false
					msg = fmt.Sprintf("New build available: %s", targetBranch.BuildID)
				} else {
					isUpToDate = true
					msg = "Game is up to date"
				}
			} else {
				if cleanBuildID != targetBranch.BuildID {
					isUpToDate = false
					msg = fmt.Sprintf("New build available: %s", targetBranch.BuildID)
				} else {
					isUpToDate = true
					msg = "Game is up to date"
				}
			}

			status := &domain.VersionStatus{
				Checked:        true,
				IsUpToDate:     isUpToDate,
				CurrentBuildID: cleanBuildID,
				LatestBuildID:  targetBranch.BuildID,
				Message:        msg,
			}
			if items, err := FetchAppNewsItems(ctx, fmt.Sprintf("%d", appID), 1); err == nil && len(items) > 0 {
				status.LatestPatchDate = items[0].Date
				status.LatestPatchTitle = items[0].Title
			}
			slog.Info("Verified game version via SteamCMD branches", "appID", appID, "localBuildID", cleanBuildID, "remoteBuildID", targetBranch.BuildID, "branch", targetBranch.Name, "upToDate", isUpToDate)
			return saveAndReturn(status)
		}
	}

	// Fallback when authoritative build IDs cannot be resolved
	status := &domain.VersionStatus{
		Checked:        false,
		IsUpToDate:     true,
		CurrentBuildID: cleanBuildID,
		Message:        "Version check unavailable",
	}
	if items, err := FetchAppNewsItems(ctx, fmt.Sprintf("%d", appID), 1); err == nil && len(items) > 0 {
		status.LatestPatchDate = items[0].Date
		status.LatestPatchTitle = items[0].Title
	}
	return saveAndReturn(status)
}
