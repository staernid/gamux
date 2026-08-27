package saves

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
	"sync"
	"time"

	"github.com/staernid/gamux/util"
	"gopkg.in/yaml.v3"
)

// DefaultLudusaviManifestURL is the official GitHub raw URL for Ludusavi's primary manifest.
const DefaultLudusaviManifestURL = "https://raw.githubusercontent.com/mtkennerly/ludusavi-manifest/master/data/manifest.yaml"

// ManifestManager coordinates loading, caching, and in-memory indexing of the Ludusavi save manifest.
type ManifestManager struct {
	mu                 sync.RWMutex
	cacheDir           string
	customURL          string
	httpClient         *http.Client
	bySteamAppID       map[uint32]*ManifestGame
	byNormalizedTitle  map[string]*ManifestGame
	byGOGID            map[uint64]*ManifestGame
	lastLoaded         time.Time
}

// NewManifestManager creates an initialized ManifestManager.
func NewManifestManager(cacheDir string, customURL string, httpClient *http.Client) *ManifestManager {
	if cacheDir == "" {
		home, _ := os.UserHomeDir()
		cacheDir = filepath.Join(home, ".local", "share", "gamux", "cache")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &ManifestManager{
		cacheDir:          cacheDir,
		customURL:         customURL,
		httpClient:        httpClient,
		bySteamAppID:      make(map[uint32]*ManifestGame),
		byNormalizedTitle: make(map[string]*ManifestGame),
		byGOGID:           make(map[uint64]*ManifestGame),
	}
}

func (m *ManifestManager) getCachePath() string {
	return filepath.Join(m.cacheDir, "ludusavi_manifest.json")
}

func (m *ManifestManager) getManifestURL() string {
	if m.customURL != "" {
		return m.customURL
	}
	return DefaultLudusaviManifestURL
}

// NormalizeTitle prepares a game title for fuzzy/normalized dictionary lookup.
func NormalizeTitle(title string) string {
	lower := strings.ToLower(title)
	var sb strings.Builder
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// LoadOrFetch loads the manifest from local disk cache, or downloads it from GitHub if stale (>7 days) or missing.
func (m *ManifestManager) LoadOrFetch(ctx context.Context, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cachePath := m.getCachePath()
	cacheExists := util.FileExists(cachePath)

	if cacheExists && !force {
		info, err := os.Stat(cachePath)
		if err == nil && time.Since(info.ModTime()) < 7*24*time.Hour {
			if err := m.loadFromJSONFile(cachePath); err == nil {
				slog.Info("Loaded Ludusavi save manifest from disk cache", "games", len(m.byNormalizedTitle))
				return nil
			}
			slog.Warn("Failed parsing local save manifest JSON, falling back to download", "path", cachePath)
		}
	}

	// Download from manifest URL
	url := m.getManifestURL()
	slog.Info("Fetching updated Ludusavi save manifest", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create manifest request: %w", err)
	}
	req.Header.Set("User-Agent", "Gamux/1.0")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		if cacheExists {
			slog.Warn("Manifest network download failed, falling back to existing cache", "error", err)
			return m.loadFromJSONFile(cachePath)
		}
		return fmt.Errorf("download manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if cacheExists {
			slog.Warn("Manifest download returned non-200, falling back to cache", "status", resp.StatusCode)
			return m.loadFromJSONFile(cachePath)
		}
		return fmt.Errorf("download manifest failed with HTTP %d", resp.StatusCode)
	}

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read manifest response: %w", err)
	}

	rawMap := make(map[string]ManifestGame)
	if strings.HasSuffix(url, ".json") {
		if err := json.Unmarshal(bodyData, &rawMap); err != nil {
			return fmt.Errorf("unmarshal manifest JSON: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(bodyData, &rawMap); err != nil {
			return fmt.Errorf("unmarshal manifest YAML: %w", err)
		}
	}

	m.indexMap(rawMap)

	// Persist to JSON disk cache for rapid subsequent startup (<50ms)
	if err := os.MkdirAll(m.cacheDir, 0755); err == nil {
		if jsonData, err := json.Marshal(rawMap); err == nil {
			_ = os.WriteFile(cachePath, jsonData, 0644)
		}
	}

	m.lastLoaded = time.Now()
	slog.Info("Successfully indexed Ludusavi save manifest", "games", len(m.byNormalizedTitle))
	return nil
}

func (m *ManifestManager) loadFromJSONFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	rawMap := make(map[string]ManifestGame)
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}
	m.indexMap(rawMap)
	m.lastLoaded = time.Now()
	return nil
}

func (m *ManifestManager) indexMap(rawMap map[string]ManifestGame) {
	bySteam := make(map[uint32]*ManifestGame, len(rawMap))
	byTitle := make(map[string]*ManifestGame, len(rawMap))
	byGOG := make(map[uint64]*ManifestGame, len(rawMap))

	for title, game := range rawMap {
		g := game // copy
		norm := NormalizeTitle(title)
		if norm != "" {
			byTitle[norm] = &g
		}
		if g.Steam.ID > 0 {
			bySteam[g.Steam.ID] = &g
		}
		if g.GOG.ID > 0 {
			byGOG[g.GOG.ID] = &g
		}
	}

	m.bySteamAppID = bySteam
	m.byNormalizedTitle = byTitle
	m.byGOGID = byGOG
}

// FindGame retrieves save configuration rules for a game by AppID, GOG ID, or Title (resolving aliases recursively).
func (m *ManifestManager) FindGame(steamAppID uint32, title string, gogID uint64) (*ManifestGame, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var candidate *ManifestGame

	if steamAppID > 0 {
		if g, ok := m.bySteamAppID[steamAppID]; ok {
			candidate = g
		}
	}

	if candidate == nil && gogID > 0 {
		if g, ok := m.byGOGID[gogID]; ok {
			candidate = g
		}
	}

	if candidate == nil && title != "" {
		norm := NormalizeTitle(title)
		if g, ok := m.byNormalizedTitle[norm]; ok {
			candidate = g
		}
	}

	if candidate == nil {
		return nil, false
	}

	// Resolve aliases recursively (with depth cap to avoid loops)
	seen := make(map[string]bool)
	for candidate != nil && candidate.Alias != "" {
		if seen[candidate.Alias] {
			break
		}
		seen[candidate.Alias] = true
		aliasNorm := NormalizeTitle(candidate.Alias)
		if target, ok := m.byNormalizedTitle[aliasNorm]; ok {
			candidate = target
		} else {
			break
		}
	}

	return candidate, true
}
