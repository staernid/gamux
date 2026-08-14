package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"golang.org/x/sync/errgroup"
)

// AchievementSchema represents a single achievement entry in Goldberg Emulator format.
type AchievementSchema struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Hidden      int    `json:"hidden"`
	Icon        string `json:"icon"`
	IconGray    string `json:"icongray"`
}

// DownloadAchievementImages downloads achievement images from Steam CDN.
func DownloadAchievementImages(ctx context.Context, appID int, imageNames []string, outputFolder string) error {
	slog.Info("Downloading achievement images", "appID", appID, "count", len(imageNames), "output", outputFolder)

	if len(imageNames) == 0 {
		return nil
	}

	if err := os.MkdirAll(outputFolder, 0755); err != nil {
		return fmt.Errorf("failed to create output folder: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10) // Limit concurrency

	urls := []string{
		"https://cdn.akamai.steamstatic.com/steamcommunity/public/images/apps/",
		"https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps/",
	}

	for _, name := range imageNames {
		imgName := name
		g.Go(func() error {
			succeeded := false
			for _, baseURL := range urls {
				url := fmt.Sprintf("%s%d/%s", baseURL, appID, imgName)
				err := func() error {
					req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
					if err != nil {
						return err
					}

					resp, err := HTTPClient.Do(req)
					if err != nil {
						return err
					}
					defer resp.Body.Close()

					if resp.StatusCode != http.StatusOK {
						return fmt.Errorf("HTTP status %d", resp.StatusCode)
					}

					imgPath := filepath.Join(outputFolder, imgName)
					out, err := os.Create(imgPath)
					if err != nil {
						return fmt.Errorf("failed to create image file: %w", err)
					}
					defer out.Close()

					if _, err := io.Copy(out, resp.Body); err != nil {
						return fmt.Errorf("failed to write image data: %w", err)
					}
					if err := out.Close(); err != nil {
						return err
					}
					succeeded = true
					return nil
				}()
				if err == nil {
					break
				}
			}

			if !succeeded {
				slog.Warn("Failed to download achievement image", "name", imgName)
			}
			return nil
		})
	}

	return g.Wait()
}

// FetchAchievementSchema queries Steam Web API ISteamUserStats/GetSchemaForGame/v2 for a game's achievements.
func FetchAchievementSchema(ctx context.Context, appID uint32, apiKey string, lang string) ([]AchievementSchema, error) {
	if lang == "" {
		lang = "english"
	}
	apiURL := fmt.Sprintf("https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v0002/?appid=%d&l=%s", appID, lang)
	if apiKey != "" {
		apiURL += "&key=" + url.QueryEscape(apiKey)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create achievement schema request: %w", err)
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch achievement schema: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("achievement schema HTTP status %d", resp.StatusCode)
	}

	var raw struct {
		Game struct {
			AvailableGameStats struct {
				Achievements []struct {
					Name        string `json:"name"`
					DisplayName string `json:"displayName"`
					Description string `json:"description"`
					Hidden      int    `json:"hidden"`
					Icon        string `json:"icon"`
					IconGray    string `json:"icongray"`
				} `json:"achievements"`
			} `json:"availableGameStats"`
		} `json:"game"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode achievement schema JSON: %w", err)
	}

	achievements := raw.Game.AvailableGameStats.Achievements
	if len(achievements) == 0 {
		return nil, nil
	}

	result := make([]AchievementSchema, 0, len(achievements))
	for _, ach := range achievements {
		iconName := filepath.Base(ach.Icon)
		iconGrayName := filepath.Base(ach.IconGray)

		iconRelPath := ach.Icon
		if iconName != "." && iconName != "/" && iconName != "" {
			iconRelPath = "img/" + iconName
		}
		iconGrayRelPath := ach.IconGray
		if iconGrayName != "." && iconGrayName != "/" && iconGrayName != "" {
			iconGrayRelPath = "img/" + iconGrayName
		}

		dispName := ach.DisplayName
		if dispName == "" {
			dispName = ach.Name
		}

		hiddenVal := 0
		if ach.Hidden != 0 {
			hiddenVal = 1
		}

		result = append(result, AchievementSchema{
			Name:        ach.Name,
			DisplayName: dispName,
			Description: ach.Description,
			Hidden:      hiddenVal,
			Icon:        iconRelPath,
			IconGray:    iconGrayRelPath,
		})
	}

	return result, nil
}
