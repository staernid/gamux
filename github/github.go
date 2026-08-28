package github

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
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

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/util"

	"github.com/charmbracelet/glamour"
	"golang.org/x/sync/errgroup"
)

// DefaultHTTPClient is the default http.Client used for GitHub API operations.
var DefaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

// HTTPClient allows injecting custom HTTP clients for testing.
var HTTPClient = DefaultHTTPClient

type contextKey string

const httpClientKey contextKey = "github_http_client"

// WithHTTPClient returns a new context with the provided http.Client attached.
func WithHTTPClient(ctx context.Context, client *http.Client) context.Context {
	if client == nil {
		return ctx
	}
	return context.WithValue(ctx, httpClientKey, client)
}

// HTTPClientFromContext extracts the http.Client from context or returns DefaultHTTPClient.
func HTTPClientFromContext(ctx context.Context) *http.Client {
	if ctx != nil {
		if client, ok := ctx.Value(httpClientKey).(*http.Client); ok && client != nil {
			return client
		}
	}
	return DefaultHTTPClient
}

// Default GitHub release endpoints
const (
	DefaultGithubAPIURL       = "https://api.github.com/repos/Detanup01/gbe_fork/releases/latest"
	DefaultSteamlessGithubAPI = "https://api.github.com/repos/staernid/steamless-rs/releases/latest"
	DefaultUMUGithubAPI       = "https://api.github.com/repos/Open-Wine-Components/umu-launcher/releases/latest"
)

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

func fetchGithubReleaseWithCache(ctx context.Context, key, apiURL string) (*Release, error) {
	if cachedData, ok := cache.GetGitHubRelease(key); ok && len(cachedData) > 0 {
		var release Release
		if err := json.Unmarshal(cachedData, &release); err == nil && release.TagName != "" {
			slog.Debug("Using cached GitHub release metadata", "key", key, "tag", release.TagName)
			return &release, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Gamux/1.0 (Linux; Game Manager)")

	resp, err := HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s release: %w", key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s release HTTP %s", key, resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode %s release JSON: %w", key, err)
	}

	if raw, err := json.Marshal(release); err == nil {
		_ = cache.SaveGitHubRelease(key, raw)
	}

	return &release, nil
}

// FetchGBERelease fetches the latest release metadata and release notes for GBE fork.
func FetchGBERelease(ctx context.Context, cfg *config.Config) (*ToolInfo, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	gbeHome := util.ExpandPath(cfg.GbeDir)
	timestampFile := filepath.Join(gbeHome, ".gbe_timestamp")

	release, err := fetchGithubReleaseWithCache(ctx, "gbe", DefaultGithubAPIURL)
	if err != nil {
		return nil, err
	}

	installedTag := "(not installed)"
	hasUpdate := true
	if _, err := os.Stat(timestampFile); err == nil {
		if timestamp, err := os.ReadFile(timestampFile); err == nil {
			tsStr := strings.TrimSpace(string(timestamp))
			if tsStr != "" {
				if tsStr == release.TagName || tsStr == release.UpdatedAt.Format(time.RFC3339) || tsStr == release.PublishedAt.Format(time.RFC3339) {
					hasUpdate = false
					installedTag = release.TagName
					if installedTag == "" {
						installedTag = "Up to Date"
					}
				} else if t, err := time.Parse(time.RFC3339, tsStr); err == nil {
					installedTag = t.Format("Jan 02, 2006")
				} else {
					installedTag = tsStr
				}
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

	release, err := fetchGithubReleaseWithCache(ctx, "steamless", DefaultSteamlessGithubAPI)
	if err != nil {
		return nil, err
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

// FetchUMURelease fetches the latest release metadata and release notes for umu-launcher.
func FetchUMURelease(ctx context.Context, cfg *config.Config) (*ToolInfo, error) {
	home, _ := os.UserHomeDir()
	umuHome := filepath.Join(home, ".local", "share", "umu")
	timestampFile := filepath.Join(umuHome, ".umu_timestamp")

	release, err := fetchGithubReleaseWithCache(ctx, "umu", DefaultUMUGithubAPI)
	if err != nil {
		return nil, err
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
	} else if fi, err := os.Stat(filepath.Join(umuHome, "umu-run")); err == nil && fi.Size() > 0 {
		installedTag = "Installed"
		hasUpdate = false
	}

	pubDate := ""
	if !release.PublishedAt.IsZero() {
		pubDate = release.PublishedAt.Format("Jan 02, 2006")
	} else if !release.UpdatedAt.IsZero() {
		pubDate = release.UpdatedAt.Format("Jan 02, 2006")
	}

	return &ToolInfo{
		Name:          "umu-launcher",
		Key:           "umu",
		InstalledPath: umuHome,
		InstalledTag:  installedTag,
		LatestTag:     release.TagName,
		ReleaseName:   release.Name,
		ReleaseNotes:  release.Body,
		ReleaseURL:    release.HTMLURL,
		PublishedAt:   pubDate,
		HasUpdate:     hasUpdate,
	}, nil
}

// GetToolsStatus returns release and update status for GBE, steamless, and umu-launcher.
func GetToolsStatus(ctx context.Context, cfg *config.Config) ([]ToolInfo, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	var gbeInfo, steamlessInfo, umuInfo *ToolInfo
	var gbeErr, steamlessErr, umuErr error

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		gbeInfo, gbeErr = FetchGBERelease(gCtx, cfg)
		return nil
	})
	g.Go(func() error {
		steamlessInfo, steamlessErr = FetchSteamlessRelease(gCtx, cfg)
		return nil
	})
	g.Go(func() error {
		umuInfo, umuErr = FetchUMURelease(gCtx, cfg)
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

	if umuErr == nil && umuInfo != nil {
		list = append(list, *umuInfo)
	} else {
		home, _ := os.UserHomeDir()
		list = append(list, ToolInfo{
			Name:          "umu-launcher",
			Key:           "umu",
			InstalledPath: filepath.Join(home, ".local", "share", "umu"),
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, DefaultGithubAPIURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch release information: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch release information: HTTP %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	tagValue := release.TagName
	if tagValue == "" {
		tagValue = release.UpdatedAt.Format(time.RFC3339)
	}

	if _, err := os.Stat(timestampFile); err == nil {
		timestamp, err := os.ReadFile(timestampFile)
		if err == nil {
			ts := strings.TrimSpace(string(timestamp))
			if ts == tagValue || ts == release.UpdatedAt.Format(time.RFC3339) || ts == release.PublishedAt.Format(time.RFC3339) {
				slog.Info("GBE fork is already up-to-date")
				return nil
			}
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

	if err := os.WriteFile(timestampFile, []byte(tagValue), 0644); err != nil {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, DefaultSteamlessGithubAPI, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch steamless release information: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch steamless release info: HTTP %s", resp.Status)
	}

	var release Release
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
			resp, err := HTTPClientFromContext(gCtx).Do(req)
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

// UpdateUMU downloads the latest standalone umu-run executable asset into ~/.local/share/umu/umu-run.
func UpdateUMU(ctx context.Context, cfg *config.Config) error {
	home, _ := os.UserHomeDir()
	destDir := filepath.Join(home, ".local", "share", "umu")
	return UpdateUMUAssets(ctx, cfg, destDir)
}

// UpdateUMUAssets fetches the latest umu-run release asset and saves it to destDir.
func UpdateUMUAssets(ctx context.Context, cfg *config.Config, destDir string) error {
	slog.Info("Fetching latest umu-launcher release assets from GitHub")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, DefaultUMUGithubAPI, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch umu release information: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch umu release info: HTTP %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to decode release JSON: %w", err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", destDir, err)
	}

	var umuAsset *ReleaseAsset

	// 1. Search for standalone zipapp tar or tar.gz (preferred official distribution)
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, "zipapp") && (strings.HasSuffix(name, ".tar") || strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz")) {
			a := asset
			umuAsset = &a
			break
		}
	}

	// 2. Search for direct umu-run executable or zipapp.zip
	if umuAsset == nil {
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			if name == "umu-run" || name == "umu_run" || name == "umu-run.zip" || strings.HasSuffix(name, "-zipapp.zip") {
				a := asset
				umuAsset = &a
				break
			}
		}
	}

	// 3. Search for any tar archive that is NOT a distro package (.deb/.rpm)
	if umuAsset == nil {
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			if !strings.HasSuffix(name, ".deb") && !strings.HasSuffix(name, ".rpm") && (strings.HasSuffix(name, ".tar") || strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip")) {
				a := asset
				umuAsset = &a
				break
			}
		}
	}

	if umuAsset == nil {
		return fmt.Errorf("no suitable standalone umu-run or zipapp.tar found in latest GitHub release")
	}

	destPath := filepath.Join(destDir, "umu-run")
	slog.Info("Downloading umu-run release asset", "asset", umuAsset.Name, "url", umuAsset.BrowserDownloadURL)

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, umuAsset.BrowserDownloadURL, nil)
	if err != nil {
		return err
	}
	resp, err = HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download asset %s: HTTP %s", umuAsset.Name, resp.Status)
	}

	assetName := strings.ToLower(umuAsset.Name)
	if strings.HasSuffix(assetName, ".tar.gz") || strings.HasSuffix(assetName, ".tgz") {
		gzr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("failed creating gzip reader: %w", err)
		}
		defer gzr.Close()
		if err := extractTarArchive(gzr, destDir); err != nil {
			return fmt.Errorf("failed extracting umu tar.gz: %w", err)
		}
	} else if strings.HasSuffix(assetName, ".tar") {
		if err := extractTarArchive(resp.Body, destDir); err != nil {
			return fmt.Errorf("failed extracting umu tar: %w", err)
		}
	} else if strings.HasSuffix(assetName, ".zip") {
		tmpFile, err := os.CreateTemp("", "umu-*.zip")
		if err != nil {
			return err
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		if _, err := io.Copy(tmpFile, resp.Body); err != nil {
			return err
		}
		if err := extractZipArchive(tmpFile.Name(), destDir); err != nil {
			return fmt.Errorf("failed extracting umu-run zip: %w", err)
		}
	} else {
		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		defer out.Close()

		if _, err := io.Copy(out, resp.Body); err != nil {
			return err
		}
	}

	// Ensure destDir/umu-run is executable
	if util.FileExists(destPath) {
		_ = os.Chmod(destPath, 0755)
	}

	timestampFile := filepath.Join(destDir, ".umu_timestamp")
	tag := release.TagName
	if tag == "" {
		tag = release.UpdatedAt.Format(time.RFC3339)
	}
	_ = os.WriteFile(timestampFile, []byte(tag), 0644)

	slog.Info("umu-launcher asset updated successfully in", "dir", destDir)
	return nil
}

func extractTarArchive(r io.Reader, destDir string) error {
	tarReader := tar.NewReader(r)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		cleanName := filepath.Clean(header.Name)
		if strings.HasPrefix(cleanName, "..") {
			continue
		}

		targetPath := filepath.Join(destDir, cleanName)
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, header.FileInfo().Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		if header.Typeflag == tar.TypeSymlink {
			_ = os.Remove(targetPath)
			_ = os.Symlink(header.Linkname, targetPath)
			continue
		}

		outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, header.FileInfo().Mode())
		if err != nil {
			return err
		}
		if _, err := io.Copy(outFile, tarReader); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
	}

	// If umu/umu-run was extracted to destDir/umu/umu-run, copy or symlink it to destDir/umu-run as well
	nestedUmu := filepath.Join(destDir, "umu", "umu-run")
	rootUmu := filepath.Join(destDir, "umu-run")
	if util.FileExists(nestedUmu) {
		_ = os.Chmod(nestedUmu, 0755)
		if !util.FileExists(rootUmu) || isInvalidHeader(rootUmu) {
			_ = util.CopyFile(nestedUmu, rootUmu)
			_ = os.Chmod(rootUmu, 0755)
		}
	}

	return nil
}

func isInvalidHeader(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 7 {
		return true
	}
	// Detect !<arch> (ar archive / deb)
	return strings.HasPrefix(string(data), "!<arch>")
}

func extractZipArchive(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		targetPath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(targetPath, f.Mode())
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		_, copyErr := io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

