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
	"strings"

	"github.com/staernid/gamux/util"
)

// ArtworkResult holds local file paths for fetched artwork.
type ArtworkResult struct {
	CoverPath  string
	BannerPath string
}

// DownloadFile downloads a URL to a local target file path.
func DownloadFile(ctx context.Context, downloadURL, targetPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", downloadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", downloadURL, resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("create dir %s: %w", filepath.Dir(targetPath), err)
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("create file %s: %w", targetPath, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// FetchLutrisArtwork downloads cover art and banner images from Steam CDN into Lutris asset directories.
// It uses ~/.cache/lutris/ (and ~/.local/share/lutris/ as secondary), matching lutris-art-fetch conventions.
// If a SteamGridDB API key exists at ~/.config/lutris_art_apikey.txt or $STEAMGRIDDB_API_KEY, it also supports SteamGridDB lookup.
func FetchLutrisArtwork(ctx context.Context, appID, slug string, dryRun bool) (*ArtworkResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	// Primary cache dir used by Lutris (and lutris-art-fetch)
	cacheCoverDir := filepath.Join(home, ".cache", "lutris", "coverart")
	cacheBannerDir := filepath.Join(home, ".cache", "lutris", "banners")

	// Secondary local share dir used by some Lutris builds
	shareCoverDir := filepath.Join(home, ".local", "share", "lutris", "coverart")
	shareBannerDir := filepath.Join(home, ".local", "share", "lutris", "banners")

	coverPath := filepath.Join(cacheCoverDir, slug+".jpg")
	bannerPath := filepath.Join(cacheBannerDir, slug+".jpg")

	if dryRun {
		slog.Info("[DRY RUN] Would fetch Lutris cover art", "slug", slug, "target", coverPath)
		slog.Info("[DRY RUN] Would fetch Lutris banner image", "slug", slug, "target", bannerPath)
		return &ArtworkResult{CoverPath: coverPath, BannerPath: bannerPath}, nil
	}

	res := &ArtworkResult{}

	// Strategy 1: Steam CDN (if AppID is valid)
	if appID != "" && appID != "0" {
		coverURL := fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_600x900.jpg", appID)
		bannerURL := fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/header.jpg", appID)

		if err := DownloadFile(ctx, coverURL, coverPath); err == nil {
			slog.Info("Fetched Lutris cover art (Steam CDN)", "path", coverPath)
			res.CoverPath = coverPath
			_ = util.CopyFile(coverPath, filepath.Join(shareCoverDir, slug+".jpg"))
		} else {
			if err := DownloadFile(ctx, bannerURL, coverPath); err == nil {
				res.CoverPath = coverPath
				_ = util.CopyFile(coverPath, filepath.Join(shareCoverDir, slug+".jpg"))
			}
		}

		if err := DownloadFile(ctx, bannerURL, bannerPath); err == nil {
			slog.Info("Fetched Lutris banner image (Steam CDN)", "path", bannerPath)
			res.BannerPath = bannerPath
			_ = util.CopyFile(bannerPath, filepath.Join(shareBannerDir, slug+".jpg"))
		}
	}

	// Strategy 2: SteamGridDB (if API key available and artwork missing)
	apiKey := getSteamGridDBAPIKey(home)
	if apiKey != "" && (res.CoverPath == "" || res.BannerPath == "") {
		if sgdbID, err := searchSteamGridDB(ctx, apiKey, slug); err == nil && sgdbID != "" {
			if res.CoverPath == "" {
				if coverURL, err := fetchSGDBImageURL(ctx, apiKey, sgdbID, "600x900"); err == nil && coverURL != "" {
					if err := DownloadFile(ctx, coverURL, coverPath); err == nil {
						slog.Info("Fetched Lutris cover art (SteamGridDB)", "path", coverPath)
						res.CoverPath = coverPath
						_ = util.CopyFile(coverPath, filepath.Join(shareCoverDir, slug+".jpg"))
					}
				}
			}
			if res.BannerPath == "" {
				if bannerURL, err := fetchSGDBImageURL(ctx, apiKey, sgdbID, "460x215"); err == nil && bannerURL != "" {
					if err := DownloadFile(ctx, bannerURL, bannerPath); err == nil {
						slog.Info("Fetched Lutris banner image (SteamGridDB)", "path", bannerPath)
						res.BannerPath = bannerPath
						_ = util.CopyFile(bannerPath, filepath.Join(shareBannerDir, slug+".jpg"))
					}
				}
			}
		}
	}

	return res, nil
}

// ── SteamGridDB Helpers ─────────────────────────────────────────────

func getSteamGridDBAPIKey(homeDir string) string {
	if key := os.Getenv("STEAMGRIDDB_API_KEY"); key != "" {
		return strings.TrimSpace(key)
	}

	keyFile := filepath.Join(homeDir, ".config", "lutris_art_apikey.txt")
	if data, err := os.ReadFile(keyFile); err == nil {
		return strings.TrimSpace(string(data))
	}

	return ""
}

func searchSteamGridDB(ctx context.Context, apiKey, gameName string) (string, error) {
	reqURL := fmt.Sprintf("https://www.steamgriddb.com/api/v2/search/autocomplete/%s", url.PathEscape(gameName))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SGDB search HTTP %d", resp.StatusCode)
	}

	var res struct {
		Success bool `json:"success"`
		Data    []struct {
			ID int `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if !res.Success || len(res.Data) == 0 {
		return "", fmt.Errorf("no SGDB result found")
	}

	return fmt.Sprintf("%d", res.Data[0].ID), nil
}

func fetchSGDBImageURL(ctx context.Context, apiKey, gameID, dimensions string) (string, error) {
	reqURL := fmt.Sprintf("https://www.steamgriddb.com/api/v2/grids/game/%s?dimensions=%s", gameID, dimensions)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SGDB grid HTTP %d", resp.StatusCode)
	}

	var res struct {
		Success bool `json:"success"`
		Data    []struct {
			URL string `json:"url"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if !res.Success || len(res.Data) == 0 {
		return "", fmt.Errorf("no grid image found")
	}

	return res.Data[0].URL, nil
}
