package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/util"

	"github.com/charmbracelet/glamour"
	"golang.org/x/sync/errgroup"
)

// HTTPClient is the http.Client used for GitHub API operations, allowing mock overrides in tests.
var HTTPClient = &http.Client{Timeout: 30 * time.Second}

// ToolInfo holds version, release metadata, and changelog notes for external tools.
type ToolInfo struct {
	Name          string `json:"name"`
	Key           string `json:"key"` // "gbe" | "steamless"
	InstalledPath string `json:"installed_path"`
	InstalledTag  string `json:"installed_tag"`
	LatestTag     string `json:"latest_tag"`
	ReleaseName   string `json:"release_name"`
	ReleaseNotes  string `json:"release_notes"`
	ReleaseURL    string `json:"release_url"`
	PublishedAt   string `json:"published_at"`
	HasUpdate     bool   `json:"has_update"`
}

// FetchGBERelease fetches the latest release metadata and release notes for GBE fork.
func FetchGBERelease(ctx context.Context, cfg *config.Config) (*ToolInfo, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	gbeHome := util.ExpandPath(cfg.GbeDir)
	timestampFile := filepath.Join(gbeHome, ".gbe_timestamp")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.GithubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch GBE release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GBE release HTTP %s", resp.Status)
	}

	var release config.Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode GBE release JSON: %w", err)
	}

	installedTag := "(not installed)"
	hasUpdate := true
	if _, err := os.Stat(timestampFile); err == nil {
		if timestamp, err := os.ReadFile(timestampFile); err == nil {
			tsStr := strings.TrimSpace(string(timestamp))
			if tsStr == release.UpdatedAt.Format(time.RFC3339) || tsStr == release.TagName {
				hasUpdate = false
				installedTag = release.TagName
				if installedTag == "" {
					installedTag = "Up to Date"
				}
			} else if tsStr != "" {
				installedTag = tsStr
			}
		}
	}

	pubDate := ""
	if !release.PublishedAt.IsZero() {
		pubDate = release.PublishedAt.Format("Jan 02, 2006")
	} else if !release.UpdatedAt.IsZero() {
		pubDate = release.UpdatedAt.Format("Jan 02, 2006")
	}

	return &ToolInfo{
		Name:          "gbe_fork",
		Key:           "gbe",
		InstalledPath: gbeHome,
		InstalledTag:  installedTag,
		LatestTag:     release.TagName,
		ReleaseName:   release.Name,
		ReleaseNotes:  release.Body,
		ReleaseURL:    release.HTMLURL,
		PublishedAt:   pubDate,
		HasUpdate:     hasUpdate,
	}, nil
}

// FetchSteamlessRelease fetches the latest release metadata and release notes for steamless-rs.
func FetchSteamlessRelease(ctx context.Context, cfg *config.Config) (*ToolInfo, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	steamlessHome := util.ExpandPath(cfg.SteamlessDir)
	timestampFile := filepath.Join(steamlessHome, ".steamless_timestamp")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.SteamlessGithubAPI, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch steamless release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steamless release HTTP %s", resp.Status)
	}

	var release config.Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode steamless release JSON: %w", err)
	}

	installedTag := "(not installed)"
	hasUpdate := true
	if _, err := os.Stat(timestampFile); err == nil {
		if timestamp, err := os.ReadFile(timestampFile); err == nil {
			tsStr := strings.TrimSpace(string(timestamp))
			if tsStr == release.UpdatedAt.Format(time.RFC3339) || tsStr == release.TagName {
				hasUpdate = false
				installedTag = release.TagName
				if installedTag == "" {
					installedTag = "Up to Date"
				}
			} else if tsStr != "" {
				installedTag = tsStr
			}
		}
	} else if fi, err := os.Stat(filepath.Join(steamlessHome, "steamless")); err == nil && fi.Size() > 0 {
		installedTag = "Installed (Local)"
	}

	pubDate := ""
	if !release.PublishedAt.IsZero() {
		pubDate = release.PublishedAt.Format("Jan 02, 2006")
	} else if !release.UpdatedAt.IsZero() {
		pubDate = release.UpdatedAt.Format("Jan 02, 2006")
	}

	return &ToolInfo{
		Name:          "steamless-rs",
		Key:           "steamless",
		InstalledPath: steamlessHome,
		InstalledTag:  installedTag,
		LatestTag:     release.TagName,
		ReleaseName:   release.Name,
		ReleaseNotes:  release.Body,
		ReleaseURL:    release.HTMLURL,
		PublishedAt:   pubDate,
		HasUpdate:     hasUpdate,
	}, nil
}

// GetToolsStatus returns release and update status for both GBE and steamless.
func GetToolsStatus(ctx context.Context, cfg *config.Config) ([]ToolInfo, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	var gbeInfo *ToolInfo
	var steamlessInfo *ToolInfo
	var gbeErr, steamlessErr error

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		gbeInfo, gbeErr = FetchGBERelease(gCtx, cfg)
		return nil
	})
	g.Go(func() error {
		steamlessInfo, steamlessErr = FetchSteamlessRelease(gCtx, cfg)
		return nil
	})
	_ = g.Wait()

	var list []ToolInfo
	if gbeErr == nil && gbeInfo != nil {
		list = append(list, *gbeInfo)
	} else {
		list = append(list, ToolInfo{
			Name:          "gbe_fork",
			Key:           "gbe",
			InstalledPath: util.ExpandPath(cfg.GbeDir),
			InstalledTag:  "Installed",
			ReleaseNotes:  "Could not fetch latest release notes from GitHub.",
		})
	}

	if steamlessErr == nil && steamlessInfo != nil {
		list = append(list, *steamlessInfo)
	} else {
		list = append(list, ToolInfo{
			Name:          "steamless-rs",
			Key:           "steamless",
			InstalledPath: util.ExpandPath(cfg.SteamlessDir),
			InstalledTag:  "Installed",
			ReleaseNotes:  "Could not fetch latest release notes from GitHub.",
		})
	}

	return list, nil
}

// UpdateGBE fetches and extracts the latest GBE fork.
func UpdateGBE(ctx context.Context, cfg *config.Config) error {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	slog.Info("Fetching latest GBE fork from GitHub")
	gbeHome := util.ExpandPath(cfg.GbeDir)
	timestampFile := filepath.Join(gbeHome, ".gbe_timestamp")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.GithubAPIURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch release information: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch release information: HTTP %s", resp.Status)
	}

	var release config.Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	if _, err := os.Stat(timestampFile); err == nil {
		timestamp, err := os.ReadFile(timestampFile)
		if err == nil && strings.TrimSpace(string(timestamp)) == release.UpdatedAt.Format(time.RFC3339) {
			slog.Info("GBE fork is already up-to-date")
			return nil
		}
	}
	// Render release notes with glamour if available
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
	)
	if err == nil {
		if renderedText, err := renderer.Render(release.Body); err == nil && renderedText != "" {
			slog.Info("GBE Release Notes", "notes", strings.TrimSpace(renderedText))
		}
	} else {
		slog.Info("GBE Release Notes", "notes", strings.TrimSpace(release.Body))
	}

	if err := os.MkdirAll(gbeHome, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", gbeHome, err)
	}

	// Find download URLs
	linuxURL := ""
	winURL := ""
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, "linux-release.tar.bz2") {
			linuxURL = asset.BrowserDownloadURL
		} else if strings.HasSuffix(asset.Name, "win-release.7z") {
			winURL = asset.BrowserDownloadURL
		}
	}

	if linuxURL == "" {
		return fmt.Errorf("failed to find Linux download URL")
	}
	if winURL == "" {
		return fmt.Errorf("failed to find Windows download URL")
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("Downloading Linux release")
		if err := util.DownloadAndExtract(gCtx, linuxURL, filepath.Join(gbeHome, "linux_release"), "tar.bz2"); err != nil {
			return fmt.Errorf("failed to update Linux release: %w", err)
		}
		slog.Info("Linux release extracted")
		return nil
	})

	g.Go(func() error {
		slog.Info("Downloading Windows release")
		if err := util.DownloadAndExtract(gCtx, winURL, filepath.Join(gbeHome, "win_release"), "7z"); err != nil {
			return fmt.Errorf("failed to update Windows release: %w", err)
		}
		slog.Info("Windows release extracted")
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	if err := os.WriteFile(timestampFile, []byte(release.UpdatedAt.Format(time.RFC3339)), 0644); err != nil {
		return fmt.Errorf("failed to write timestamp file: %w", err)
	}

	slog.Info("GBE fork updated successfully")
	return nil
}

// UpdateSteamlessAssets fetches the latest release assets from steamless-rs and saves them to destDir.
func UpdateSteamlessAssets(ctx context.Context, cfg *config.Config, destDir string) error {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	slog.Info("Fetching latest Steamless release assets from GitHub")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.SteamlessGithubAPI, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch steamless release information: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch steamless release info: HTTP %s", resp.Status)
	}

	var release config.Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to decode release JSON: %w", err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", destDir, err)
	}

	g, gCtx := errgroup.WithContext(ctx)

	for _, asset := range release.Assets {
		asset := asset
		g.Go(func() error {
			destPath := filepath.Join(destDir, asset.Name)
			slog.Info("Downloading steamless release asset", "asset", asset.Name, "url", asset.BrowserDownloadURL)

			req, err := http.NewRequestWithContext(gCtx, http.MethodGet, asset.BrowserDownloadURL, nil)
			if err != nil {
				return err
			}
			resp, err := HTTPClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to download asset %s: HTTP %s", asset.Name, resp.Status)
			}

			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.Copy(out, resp.Body)
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed downloading steamless assets: %w", err)
	}

	timestampFile := filepath.Join(destDir, ".steamless_timestamp")
	_ = os.WriteFile(timestampFile, []byte(release.UpdatedAt.Format(time.RFC3339)), 0644)

	slog.Info("Steamless release assets updated successfully in", "dir", destDir)
	return nil
}
