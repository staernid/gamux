package cache

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	DisableCache bool
	baseCacheDir string
	dirMutex     sync.RWMutex
	nameMutex    sync.Mutex
)

// SetCacheDir overrides the cache root directory (useful for test isolation).
func SetCacheDir(dir string) {
	dirMutex.Lock()
	defer dirMutex.Unlock()
	baseCacheDir = dir
	if dir != "" {
		_ = os.MkdirAll(filepath.Join(baseCacheDir, "hubcap"), 0755)
		_ = os.MkdirAll(filepath.Join(baseCacheDir, "news"), 0755)
	}
}

func getCacheDir() string {
	if DisableCache {
		return ""
	}
	dirMutex.RLock()
	if baseCacheDir != "" {
		d := baseCacheDir
		dirMutex.RUnlock()
		return d
	}
	dirMutex.RUnlock()

	dirMutex.Lock()
	defer dirMutex.Unlock()
	if baseCacheDir != "" {
		return baseCacheDir
	}
	userCache, err := os.UserCacheDir()
	if err != nil || userCache == "" {
		home, _ := os.UserHomeDir()
		userCache = filepath.Join(home, ".cache")
	}
	baseCacheDir = filepath.Join(userCache, "gamux")
	_ = os.MkdirAll(filepath.Join(baseCacheDir, "hubcap"), 0755)
	_ = os.MkdirAll(filepath.Join(baseCacheDir, "news"), 0755)
	return baseCacheDir
}

// GetHubcapZIP retrieves cached Hubcap ZIP archive bytes if available on disk.
func GetHubcapZIP(appID uint32) ([]byte, bool) {
	if DisableCache {
		return nil, false
	}
	dir := getCacheDir()

	zipPath := filepath.Join(dir, "hubcap", fmt.Sprintf("%d.zip", appID))
	data, err := os.ReadFile(zipPath)
	if err == nil && len(data) > 0 {
		slog.Info("Using cached Hubcap ZIP archive from disk (0 API requests consumed)", "appID", appID)
		return data, true
	}
	return nil, false
}

// SaveHubcapZIP writes Hubcap ZIP archive bytes to local disk cache.
func SaveHubcapZIP(appID uint32, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	dir := getCacheDir()
	zipPath := filepath.Join(dir, "hubcap", fmt.Sprintf("%d.zip", appID))
	slog.Debug("Caching Hubcap ZIP archive to disk", "appID", appID, "path", zipPath)
	return os.WriteFile(zipPath, data, 0644)
}

type AppNameCache struct {
	Names map[string]string `json:"names"`
}

// GetAppName looks up a cached Steam AppID title.
func GetAppName(appID string) (string, bool) {
	if DisableCache {
		return "", false
	}
	nameMutex.Lock()
	defer nameMutex.Unlock()


	dir := getCacheDir()
	path := filepath.Join(dir, "app_names.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var cache AppNameCache
	if err := json.Unmarshal(data, &cache); err == nil && cache.Names != nil {
		if name, ok := cache.Names[appID]; ok && name != "" {
			return name, true
		}
	}
	return "", false
}

// SaveAppName saves a Steam AppID title to local disk cache.
func SaveAppName(appID, name string) {
	if appID == "" || name == "" {
		return
	}
	nameMutex.Lock()
	defer nameMutex.Unlock()

	dir := getCacheDir()
	path := filepath.Join(dir, "app_names.json")

	var cache AppNameCache
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &cache)
	}
	if cache.Names == nil {
		cache.Names = make(map[string]string)
	}
	cache.Names[appID] = name

	if bytes, err := json.MarshalIndent(cache, "", "  "); err == nil {
		_ = os.WriteFile(path, bytes, 0644)
	}
}

type CachedNews struct {
	Timestamp time.Time `json:"timestamp"`
	News      string    `json:"news"`
}

// GetNews retrieves cached Steam patch notes if less than 24 hours old.
func GetNews(appID string) (string, bool) {
	dir := getCacheDir()
	path := filepath.Join(dir, "news", fmt.Sprintf("%s.json", appID))
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var item CachedNews
	if err := json.Unmarshal(data, &item); err == nil {
		if time.Since(item.Timestamp) < 24*time.Hour {
			return item.News, true
		}
	}
	return "", false
}

// SaveNews caches Steam patch notes to disk for 24 hours.
func SaveNews(appID, news string) {
	if appID == "" || news == "" {
		return
	}
	dir := getCacheDir()
	path := filepath.Join(dir, "news", fmt.Sprintf("%s.json", appID))
	item := CachedNews{
		Timestamp: time.Now(),
		News:      news,
	}
	if bytes, err := json.MarshalIndent(item, "", "  "); err == nil {
		_ = os.WriteFile(path, bytes, 0644)
	}
}
