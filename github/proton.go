package github

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/util"
	"github.com/ulikunitz/xz"
)

// Proton release API endpoints
const (
	DefaultProtonGEAPI      = "https://api.github.com/repos/GloriousEggroll/proton-ge-custom/releases/latest"
	DefaultProtonCachyOSAPI = "https://api.github.com/repos/CachyOS/proton-cachyos/releases/latest"
)

// ProtonFlavorSpec defines configuration for a supported Proton distribution.
type ProtonFlavorSpec struct {
	Key         string
	Name        string
	Alias       string
	DirName     string
	APIURL      string
	AssetSuffix string
}

// GetProtonFlavor returns the ProtonFlavorSpec for the given key or alias.
func GetProtonFlavor(key string) (*ProtonFlavorSpec, error) {
	clean := strings.ToLower(strings.TrimSpace(key))
	switch clean {
	case "proton-ge", "ge", "proton-ge latest", "ge-proton":
		return &ProtonFlavorSpec{
			Key:         "proton-ge",
			Name:        "GE-Proton (Latest)",
			Alias:       "Proton-GE Latest",
			DirName:     "Proton-GE-Latest",
			APIURL:      DefaultProtonGEAPI,
			AssetSuffix: ".tar.gz",
		}, nil
	case "proton-cachyos", "cachyos", "proton-cachyos latest":
		return &ProtonFlavorSpec{
			Key:         "proton-cachyos",
			Name:        "Proton-CachyOS (Latest)",
			Alias:       "Proton-CachyOS Latest",
			DirName:     "Proton-CachyOS-Latest",
			APIURL:      DefaultProtonCachyOSAPI,
			AssetSuffix: ".tar.xz",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported proton flavor: %s", key)
	}
}

// GetProtonCompatDir returns the Steam compatibilitytools.d directory.
func GetProtonCompatDir() string {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "Steam", "compatibilitytools.d")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "Steam", "compatibilitytools.d")
}

func normalizeProtonVersion(v string) string {
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "GE-Proton")
	v = strings.TrimPrefix(v, "cachyos-wineland-")
	v = strings.TrimPrefix(v, "wineland-")
	v = strings.TrimPrefix(v, "cachyos-")
	v = strings.TrimPrefix(v, "proton-cachyos-")
	v = strings.TrimSuffix(v, "+")
	return strings.TrimSpace(v)
}

// GetInstalledProtonVersion reads the version of an installed Proton directory.
func GetInstalledProtonVersion(protonDir string) string {
	versionFile := filepath.Join(protonDir, "version")
	if data, err := os.ReadFile(versionFile); err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) > 0 {
			fields := strings.Fields(lines[0])
			if len(fields) >= 2 {
				return fields[1]
			} else if len(fields) == 1 {
				return fields[0]
			}
		}
	}
	tsFile := filepath.Join(protonDir, ".gamux_proton_timestamp")
	if data, err := os.ReadFile(tsFile); err == nil {
		return strings.TrimSpace(string(data))
	}
	return ""
}

// FetchProtonRelease queries release information for a Proton flavor.
func FetchProtonRelease(ctx context.Context, flavorKey string) (*ToolInfo, error) {
	spec, err := GetProtonFlavor(flavorKey)
	if err != nil {
		return nil, err
	}

	compatDir := GetProtonCompatDir()
	protonDir := filepath.Join(compatDir, spec.DirName)

	release, err := fetchGithubReleaseWithCache(ctx, spec.Key, spec.APIURL)
	if err != nil {
		return nil, err
	}

	installedVer := GetInstalledProtonVersion(protonDir)
	isInstalled := util.DirExists(protonDir)
	hasUpdate := true
	installedTag := ""

	if isInstalled {
		installedTag = installedVer
		if installedTag == "" {
			installedTag = "Installed"
		}
		if installedVer != "" && release.TagName != "" {
			if installedVer == release.TagName || normalizeProtonVersion(installedVer) == normalizeProtonVersion(release.TagName) {
				hasUpdate = false
				installedTag = release.TagName
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
		Name:          spec.Name,
		Key:           spec.Key,
		InstalledPath: protonDir,
		InstalledTag:  installedTag,
		LatestTag:     release.TagName,
		ReleaseName:   release.Name,
		ReleaseNotes:  release.Body,
		ReleaseURL:    release.HTMLURL,
		PublishedAt:   pubDate,
		HasUpdate:     hasUpdate,
		IsInstalled:   isInstalled,
	}, nil
}

// UpdateProton downloads, extracts, and configures the latest release of a Proton flavor.
func UpdateProton(ctx context.Context, flavorKey string) error {
	spec, err := GetProtonFlavor(flavorKey)
	if err != nil {
		return err
	}

	slog.Info("Fetching latest Proton release metadata", "flavor", spec.Key)
	release, err := fetchGithubReleaseWithCache(ctx, spec.Key, spec.APIURL)
	if err != nil {
		return fmt.Errorf("fetch %s release: %w", spec.Key, err)
	}

	// Select matching asset
	var downloadURL, assetName string
	for _, asset := range release.Assets {
		name := asset.Name
		if !strings.HasSuffix(name, spec.AssetSuffix) {
			continue
		}
		if strings.HasSuffix(name, ".sha512sum") || strings.HasSuffix(name, ".sha256sum") {
			continue
		}
		// Prefer x86_64, avoid arm/aarch64
		if strings.Contains(name, "aarch64") || strings.Contains(name, "arm64") {
			continue
		}
		downloadURL = asset.BrowserDownloadURL
		assetName = name
		break
	}

	if downloadURL == "" {
		return fmt.Errorf("no suitable asset found for %s (looked for %s)", spec.Key, spec.AssetSuffix)
	}

	compatDir := GetProtonCompatDir()
	if err := os.MkdirAll(compatDir, 0755); err != nil {
		return fmt.Errorf("create compat dir %s: %w", compatDir, err)
	}

	tmpDir, err := os.MkdirTemp(compatDir, fmt.Sprintf(".tmp_%s_*", spec.Key))
	if err != nil {
		return fmt.Errorf("create temp extraction dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	slog.Info("Downloading Proton release asset", "asset", assetName, "url", downloadURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Gamux/1.0 (Linux; Game Manager)")

	resp, err := HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return fmt.Errorf("download %s asset: %w", spec.Key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s asset: HTTP %s", spec.Key, resp.Status)
	}

	// Decompress and untar
	var tarReader *tar.Reader
	if strings.HasSuffix(assetName, ".tar.gz") || strings.HasSuffix(assetName, ".tgz") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("open gzip reader: %w", err)
		}
		defer gz.Close()
		tarReader = tar.NewReader(gz)
	} else if strings.HasSuffix(assetName, ".tar.xz") || strings.HasSuffix(assetName, ".txz") {
		xzr, err := xz.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("open xz reader: %w", err)
		}
		tarReader = tar.NewReader(xzr)
	} else {
		return fmt.Errorf("unsupported archive format for %s: %s", spec.Key, assetName)
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		cleanName := filepath.Clean(header.Name)
		if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			continue
		}

		targetPath := filepath.Join(tmpDir, cleanName)
		rel, err := filepath.Rel(tmpDir, targetPath)
		if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
			continue
		}

		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		if header.Typeflag == tar.TypeSymlink {
			_ = os.Remove(targetPath)
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				slog.Warn("Failed creating symlink during Proton extraction", "target", targetPath, "err", err)
			}
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

	// Find the extracted top-level root directory
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return fmt.Errorf("read extracted dir: %w", err)
	}

	var extractedRoot string
	for _, entry := range entries {
		if entry.IsDir() {
			extractedRoot = filepath.Join(tmpDir, entry.Name())
			break
		}
	}
	if extractedRoot == "" {
		extractedRoot = tmpDir
	}

	targetDir := filepath.Join(compatDir, spec.DirName)
	if util.DirExists(targetDir) {
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("remove old proton dir %s: %w", targetDir, err)
		}
	}

	if err := os.Rename(extractedRoot, targetDir); err != nil {
		return fmt.Errorf("move extracted proton to %s: %w", targetDir, err)
	}

	// Write compatibilitytool.vdf with the alias name so Steam/UMU recognize it
	vdfContent := fmt.Sprintf(`"compatibilitytools"
{
  "compat_tools"
  {
    "%s"
    {
      "install_path" "."
      "display_name" "%s"
      "from_oslist"  "windows"
      "to_oslist"    "linux"
    }
  }
}
`, spec.Alias, spec.Alias)

	vdfPath := filepath.Join(targetDir, "compatibilitytool.vdf")
	if err := os.WriteFile(vdfPath, []byte(vdfContent), 0644); err != nil {
		slog.Warn("Failed to write compatibilitytool.vdf", "path", vdfPath, "err", err)
	}

	// Create symlink for the spaced alias if different from directory name
	if spec.Alias != spec.DirName {
		aliasPath := filepath.Join(compatDir, spec.Alias)
		_ = os.Remove(aliasPath)
		if err := os.Symlink(spec.DirName, aliasPath); err != nil {
			slog.Warn("Failed to create Proton alias symlink", "alias", aliasPath, "err", err)
		}
	}

	// Record installed timestamp / tag
	tsFile := filepath.Join(targetDir, ".gamux_proton_timestamp")
	_ = os.WriteFile(tsFile, []byte(release.TagName), 0644)

	slog.Info("Proton installed successfully", "flavor", spec.Key, "dir", targetDir, "tag", release.TagName)
	return nil
}
