package gbe

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
)

// LoadAchievements discovers and parses achievement definitions and unlocked statuses
// for a game directory across local steam_settings and standard Goldberg GSE emulator save paths.
func LoadAchievements(ctx context.Context, gameDir string, appID uint32, settingsDirs []string, apiKey string) ([]domain.AchievementItem, error) {
	if gameDir == "" {
		return nil, fmt.Errorf("game directory cannot be empty")
	}

	gameDir = util.CanonicalPath(gameDir)

	if len(settingsDirs) == 0 {
		settingsDirs = []string{filepath.Join(gameDir, "steam_settings")}
		for _, d := range FindSteamSettingsDirs(gameDir) {
			if d != settingsDirs[0] {
				settingsDirs = append(settingsDirs, d)
			}
		}
	}

	// Fallback to steam_appid.txt if appID wasn't parsed from ACF/status
	if appID == 0 {
		checkFiles := []string{filepath.Join(gameDir, "steam_appid.txt")}
		for _, sDir := range settingsDirs {
			checkFiles = append(checkFiles, filepath.Join(sDir, "steam_appid.txt"))
		}
		for _, p := range checkFiles {
			if data, err := os.ReadFile(p); err == nil {
				txt := strings.TrimSpace(string(data))
				if parsed, ok := util.ParseAppID(txt); ok && parsed > 0 {
					appID = parsed
					break
				}
			}
		}
	}

	// Fallback to launch_profile.json
	if appID == 0 {
		profPath := filepath.Join(gameDir, "steam_settings", "launch_profile.json")
		if data, err := os.ReadFile(profPath); err == nil {
			var prof struct {
				AppID string `json:"app_id"`
			}
			if err := json.Unmarshal(data, &prof); err == nil && prof.AppID != "" {
				if parsed, ok := util.ParseAppID(prof.AppID); ok && parsed > 0 {
					appID = parsed
				}
			}
		}
	}

	var schemas []steam.AchievementSchema
	for _, sDir := range settingsDirs {
		achievementsFile := filepath.Join(sDir, "achievements.json")
		if util.FileExists(achievementsFile) {
			if data, err := os.ReadFile(achievementsFile); err == nil {
				if err := json.Unmarshal(data, &schemas); err == nil && len(schemas) > 0 {
					break
				}
				// Try map format {"ACH_NAME": {displayName: "...", ...}}
				var schemaMap map[string]steam.AchievementSchema
				if err2 := json.Unmarshal(data, &schemaMap); err2 == nil && len(schemaMap) > 0 {
					for k, v := range schemaMap {
						if v.Name == "" {
							v.Name = k
						}
						schemas = append(schemas, v)
					}
					break
				}
			}
		}
	}

	if len(schemas) == 0 && appID > 0 {
		if fetched, err := steam.FetchAchievementSchema(ctx, appID, apiKey, "english"); err == nil && len(fetched) > 0 {
			schemas = fetched
		}
	}

	if len(schemas) == 0 {
		return []domain.AchievementItem{}, nil
	}

	// Discover unlocked achievements from standard GSE emulator save paths
	unlockedMap := make(map[string]int64)

	var candidatePaths []string
	for _, sDir := range settingsDirs {
		candidatePaths = append(candidatePaths, filepath.Join(sDir, "achievements_unlocked.json"))
	}

	home, _ := os.UserHomeDir()
	if appID > 0 && home != "" {
		appIDStr := fmt.Sprintf("%d", appID)
		candidatePaths = append(candidatePaths,
			filepath.Join(home, ".local", "share", "GSE Saves", appIDStr, "achievements.json"),
			filepath.Join(home, ".local", "share", "GSE Saves", appIDStr, "remote", "achievements.json"),
			filepath.Join(home, ".local", "share", "GSE Saves", "settings", "achievements.json"),
			filepath.Join(home, "Games", "umu", "umu-default", "drive_c", "users", "steamuser", "AppData", "Roaming", "GSE Saves", appIDStr, "achievements.json"),
		)

		// Expand nested remote folder patterns (e.g. GSE Saves/<appID>/remote/*/achievements.json)
		if matches, err := filepath.Glob(filepath.Join(home, ".local", "share", "GSE Saves", appIDStr, "remote", "*", "achievements.json")); err == nil {
			candidatePaths = append(candidatePaths, matches...)
		}
	}

	for _, p := range candidatePaths {
		if !util.FileExists(p) {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		var simpleMap map[string]any
		if err := json.Unmarshal(data, &simpleMap); err == nil {
			for k, v := range simpleMap {
				switch val := v.(type) {
				case float64:
					if val > 0 {
						unlockedMap[strings.ToUpper(k)] = int64(val)
					}
				case bool:
					if val {
						unlockedMap[strings.ToUpper(k)] = 1
					}
				case map[string]any:
					earned := false
					if e, ok := val["earned"].(bool); ok && e {
						earned = true
					} else if u, ok := val["unlocked"].(bool); ok && u {
						earned = true
					}

					if earned {
						var t int64 = 1
						if et, ok := val["earned_time"].(float64); ok && et > 0 {
							t = int64(et)
						} else if ut, ok := val["unlock_time"].(float64); ok && ut > 0 {
							t = int64(ut)
						}
						unlockedMap[strings.ToUpper(k)] = t
					}
				default:
					unlockedMap[strings.ToUpper(k)] = 1
				}
			}
		} else {
			var arr []string
			if err := json.Unmarshal(data, &arr); err == nil {
				for _, k := range arr {
					unlockedMap[strings.ToUpper(k)] = 1
				}
			}
		}
	}

	items := make([]domain.AchievementItem, 0, len(schemas))
	for _, s := range schemas {
		unlockedTime, isUnlocked := unlockedMap[strings.ToUpper(s.Name)]

		iconURL := s.Icon
		iconGrayURL := s.IconGray

		cleanIcon := strings.TrimPrefix(s.Icon, "img/")
		cleanGray := strings.TrimPrefix(s.IconGray, "img/")

		// Check if local image files exist across all candidate steam_settings dirs
		var localImgPath string
		for _, sDir := range settingsDirs {
			cand := filepath.Join(sDir, "img", cleanIcon)
			if util.FileExists(cand) {
				localImgPath = cand
				break
			}
			cand2 := filepath.Join(sDir, s.Icon)
			if util.FileExists(cand2) {
				localImgPath = cand2
				break
			}
		}
		if localImgPath != "" {
			iconURL = "/local-img" + localImgPath
		} else if appID > 0 && !strings.HasPrefix(iconURL, "http") {
			iconURL = fmt.Sprintf("https://shared.fastly.steamstatic.com/community_assets/images/apps/%d/%s", appID, cleanIcon)
		}

		var localGrayPath string
		for _, sDir := range settingsDirs {
			cand := filepath.Join(sDir, "img", cleanGray)
			if util.FileExists(cand) {
				localGrayPath = cand
				break
			}
			cand2 := filepath.Join(sDir, s.IconGray)
			if util.FileExists(cand2) {
				localGrayPath = cand2
				break
			}
		}
		if localGrayPath != "" {
			iconGrayURL = "/local-img" + localGrayPath
		} else if appID > 0 && !strings.HasPrefix(iconGrayURL, "http") {
			iconGrayURL = fmt.Sprintf("https://shared.fastly.steamstatic.com/community_assets/images/apps/%d/%s", appID, cleanGray)
		}

		items = append(items, domain.AchievementItem{
			ID:          s.Name,
			Name:        s.DisplayName,
			Description: s.Description,
			IconURL:     iconURL,
			IconGrayURL: iconGrayURL,
			Hidden:      s.Hidden == 1,
			Unlocked:    isUnlocked,
			UnlockTime:  unlockedTime,
		})
	}

	// Sort achievements: completed/unlocked first (sorted by most recent unlock time desc), then locked
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Unlocked != items[j].Unlocked {
			return items[i].Unlocked // true comes before false
		}
		if items[i].Unlocked {
			return items[i].UnlockTime > items[j].UnlockTime // most recently unlocked first
		}
		return false // preserve schema order for locked items
	})

	return items, nil
}
