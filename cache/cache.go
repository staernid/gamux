package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/staernid/gamux/domain"
)

var (
	disableCache atomic.Bool
	baseCacheDir string
	dirMutex     sync.RWMutex

	// In-memory thread-safe cache for Steam app names
	nameMu          sync.RWMutex
	nameMemoryCache map[string]string
	namesLoaded     bool

	// In-memory thread-safe cache for Steam store AppDetails
	detailsMu          sync.RWMutex
	detailsMemoryCache = make(map[string][]byte)

	// In-memory thread-safe cache for inspected game records (Warm Library Cache)
	gamesMu          sync.RWMutex
	gamesMemoryCache = make(map[string]CachedGameRecord)

	// In-memory thread-safe cache for Steam news items
	newsItemsMu          sync.RWMutex
	newsItemsMemoryCache = make(map[string][]byte)
)

// IsDisabled reports whether caching is globally disabled (useful for testing or forced refresh).
func IsDisabled() bool {
	return disableCache.Load()
}

// SetDisabled globally enables or disables all disk and memory caching.
func SetDisabled(v bool) {
	disableCache.Store(v)
}

var cacheSubdirs = []string{
	"hubcap",
	"news",
	"appdetails",
	"github",
	"revobd",
	"steamdepotdb",
	"manifesthub",
	"sgdb",
	"store_meta",
	"games",
}

// subdirTTLs maps each cache subdirectory to its canonical TTL.
// Used by Prune to determine which entries are expired.
var subdirTTLs = map[string]time.Duration{
	"appdetails":  7 * 24 * time.Hour,
	"news":        24 * time.Hour,
	"github":      1 * time.Hour,
	"steamdepotdb": 24 * time.Hour,
	"manifesthub": 24 * time.Hour,
	"sgdb":        7 * 24 * time.Hour,
	"store_meta":  24 * time.Hour,
	"hubcap":      30 * 24 * time.Hour,
	"revobd":      30 * 24 * time.Hour,
}

// SetCacheDir overrides the cache root directory (useful for test isolation).
func SetCacheDir(dir string) {
	dirMutex.Lock()
	defer dirMutex.Unlock()
	baseCacheDir = dir
	if dir != "" {
		for _, sub := range cacheSubdirs {
			_ = os.MkdirAll(filepath.Join(baseCacheDir, sub), 0755)
		}
	}

	nameMu.Lock()
	nameMemoryCache = nil
	namesLoaded = false
	nameMu.Unlock()

	detailsMu.Lock()
	detailsMemoryCache = make(map[string][]byte)
	detailsMu.Unlock()

	gamesMu.Lock()
	gamesMemoryCache = make(map[string]CachedGameRecord)
	gamesMu.Unlock()

	newsItemsMu.Lock()
	newsItemsMemoryCache = make(map[string][]byte)
	newsItemsMu.Unlock()
}

func getCacheDir() string {
	if disableCache.Load() {
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
	for _, sub := range cacheSubdirs {
		_ = os.MkdirAll(filepath.Join(baseCacheDir, sub), 0755)
	}
	return baseCacheDir
}

// TTLWrapper wraps cached byte payloads with an explicit expiration timestamp.
type TTLWrapper struct {
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

func readTTLFile(path string, ttl time.Duration) ([]byte, bool) {
	if disableCache.Load() {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var wrapper TTLWrapper
	if err := json.Unmarshal(data, &wrapper); err == nil && len(wrapper.Payload) > 0 {
		if time.Since(wrapper.Timestamp) < ttl {
			return wrapper.Payload, true
		}
	}
	return nil, false
}

func writeTTLFile(path string, payload []byte) error {
	if disableCache.Load() || len(payload) == 0 {
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	wrapper := TTLWrapper{
		Timestamp: time.Now(),
		Payload:   payload,
	}
	data, err := json.Marshal(wrapper)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ==========================================
// 1. Steam App Names Cache (In-Memory + Disk)
// ==========================================

type appNameDiskCache struct {
	Names map[string]string `json:"names"`
}

func loadAppNamesLocked() {
	if namesLoaded {
		return
	}
	nameMemoryCache = make(map[string]string)
	namesLoaded = true

	dir := getCacheDir()
	if dir == "" {
		return
	}
	path := filepath.Join(dir, "app_names.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var disk appNameDiskCache
	if err := json.Unmarshal(data, &disk); err == nil && disk.Names != nil {
		for k, v := range disk.Names {
			nameMemoryCache[k] = v
		}
	}
}

// GetAppName looks up a cached Steam AppID title (in-memory fast lookup with disk fallback).
func GetAppName(appID string) (string, bool) {
	if disableCache.Load() || appID == "" {
		return "", false
	}
	nameMu.RLock()
	if namesLoaded {
		if name, ok := nameMemoryCache[appID]; ok && name != "" {
			nameMu.RUnlock()
			return name, true
		}
	}
	nameMu.RUnlock()

	nameMu.Lock()
	defer nameMu.Unlock()
	loadAppNamesLocked()
	if name, ok := nameMemoryCache[appID]; ok && name != "" {
		return name, true
	}
	return "", false
}

// SaveAppName saves a Steam AppID title to in-memory and local disk cache.
func SaveAppName(appID, name string) {
	if disableCache.Load() || appID == "" || name == "" {
		return
	}
	nameMu.Lock()
	defer nameMu.Unlock()

	loadAppNamesLocked()
	if nameMemoryCache[appID] == name {
		return
	}
	nameMemoryCache[appID] = name

	dir := getCacheDir()
	if dir == "" {
		return
	}
	path := filepath.Join(dir, "app_names.json")
	disk := appNameDiskCache{Names: nameMemoryCache}
	if bytes, err := json.MarshalIndent(disk, "", "  "); err == nil {
		_ = os.WriteFile(path, bytes, 0644)
	}
}

// ==========================================
// 2. Steam AppDetails Cache (7 Days TTL)
// ==========================================

// GetAppDetails retrieves cached Steam store AppDetails JSON if less than 7 days old.
func GetAppDetails(appID string) ([]byte, bool) {
	if disableCache.Load() || appID == "" {
		return nil, false
	}

	// 1. Fast in-memory cache check
	detailsMu.RLock()
	if cached, ok := detailsMemoryCache[appID]; ok && len(cached) > 0 {
		detailsMu.RUnlock()
		return cached, true
	}
	detailsMu.RUnlock()

	// 2. Disk cache check
	dir := getCacheDir()
	if dir == "" {
		return nil, false
	}
	path := filepath.Join(dir, "appdetails", fmt.Sprintf("%s.json", appID))
	if data, ok := readTTLFile(path, 7*24*time.Hour); ok {
		detailsMu.Lock()
		detailsMemoryCache[appID] = data
		detailsMu.Unlock()
		return data, true
	}
	return nil, false
}

// SaveAppDetails writes Steam store AppDetails JSON to in-memory and disk cache for 7 days.
func SaveAppDetails(appID string, data []byte) error {
	if disableCache.Load() || appID == "" || len(data) == 0 {
		return nil
	}
	detailsMu.Lock()
	detailsMemoryCache[appID] = data
	detailsMu.Unlock()

	dir := getCacheDir()
	if dir == "" {
		return nil
	}
	path := filepath.Join(dir, "appdetails", fmt.Sprintf("%s.json", appID))
	return writeTTLFile(path, data)
}

// ==========================================
// 3. Steam News / Patch Notes Cache (12h/24h)
// ==========================================

// GetNews retrieves cached Steam patch notes if less than 24 hours old.
func GetNews(appID string) (string, bool) {
	if disableCache.Load() || appID == "" {
		return "", false
	}
	dir := getCacheDir()
	if dir == "" {
		return "", false
	}
	path := filepath.Join(dir, "news", fmt.Sprintf("%s.json", appID))
	if data, ok := readTTLFile(path, 24*time.Hour); ok {
		return string(data), true
	}
	return "", false
}

// SaveNews caches Steam patch notes to disk for 24 hours.
func SaveNews(appID, news string) {
	if disableCache.Load() || appID == "" || news == "" {
		return
	}
	dir := getCacheDir()
	if dir == "" {
		return
	}
	path := filepath.Join(dir, "news", fmt.Sprintf("%s.json", appID))
	_ = writeTTLFile(path, []byte(news))
}

// GetNewsItems retrieves cached Steam patch notes list if less than 12 hours old.
func GetNewsItems(appID string) ([]byte, bool) {
	if disableCache.Load() || appID == "" {
		return nil, false
	}

	// 1. Fast in-memory cache check
	newsItemsMu.RLock()
	if cached, ok := newsItemsMemoryCache[appID]; ok && len(cached) > 0 {
		newsItemsMu.RUnlock()
		return cached, true
	}
	newsItemsMu.RUnlock()

	// 2. Disk cache check
	dir := getCacheDir()
	if dir == "" {
		return nil, false
	}
	path := filepath.Join(dir, "news", fmt.Sprintf("%s_items.json", appID))
	if data, ok := readTTLFile(path, 12*time.Hour); ok {
		newsItemsMu.Lock()
		newsItemsMemoryCache[appID] = data
		newsItemsMu.Unlock()
		return data, true
	}
	return nil, false
}

// SaveNewsItems caches Steam patch notes list to in-memory and disk cache for 12 hours.
func SaveNewsItems(appID string, itemsJSON []byte) {
	if disableCache.Load() || appID == "" || len(itemsJSON) == 0 {
		return
	}

	// Never downgrade cache: if existing cache has more items and is valid, keep it
	if existingBytes, ok := GetNewsItems(appID); ok {
		var existingItems []json.RawMessage
		var newItems []json.RawMessage
		if json.Unmarshal(existingBytes, &existingItems) == nil && json.Unmarshal(itemsJSON, &newItems) == nil {
			if len(existingItems) > len(newItems) {
				return
			}
		}
	}

	// Update in-memory cache
	newsItemsMu.Lock()
	newsItemsMemoryCache[appID] = itemsJSON
	newsItemsMu.Unlock()

	dir := getCacheDir()
	if dir == "" {
		return
	}
	path := filepath.Join(dir, "news", fmt.Sprintf("%s_items.json", appID))
	_ = writeTTLFile(path, itemsJSON)
}

// ==========================================
// 4. Hubcap & RevoBD Manifest ZIP Cache (30 Days)
// ==========================================

// GetHubcapZIP retrieves cached Hubcap ZIP archive bytes if available and less than 30 days old.
func GetHubcapZIP(appID uint32) ([]byte, bool) {
	if disableCache.Load() {
		return nil, false
	}
	dir := getCacheDir()
	if dir == "" {
		return nil, false
	}
	zipPath := filepath.Join(dir, "hubcap", fmt.Sprintf("%d.zip", appID))
	info, err := os.Stat(zipPath)
	if err != nil || info.Size() == 0 {
		return nil, false
	}
	if time.Since(info.ModTime()) >= 30*24*time.Hour {
		slog.Debug("Hubcap ZIP cache expired (>30 days), will re-fetch", "appID", appID)
		return nil, false
	}
	data, err := os.ReadFile(zipPath)
	if err == nil && len(data) > 0 {
		slog.Info("Using cached Hubcap ZIP archive from disk (0 API requests consumed)", "appID", appID)
		return data, true
	}
	return nil, false
}

// SaveHubcapZIP writes Hubcap ZIP archive bytes to local disk cache.
func SaveHubcapZIP(appID uint32, data []byte) error {
	if disableCache.Load() || len(data) == 0 {
		return nil
	}
	dir := getCacheDir()
	if dir == "" {
		return nil
	}
	zipPath := filepath.Join(dir, "hubcap", fmt.Sprintf("%d.zip", appID))
	slog.Debug("Caching Hubcap ZIP archive to disk", "appID", appID, "path", zipPath)
	return os.WriteFile(zipPath, data, 0644)
}

// GetRevoBDZIP retrieves cached RevoBD ZIP archive bytes if available and less than 30 days old.
func GetRevoBDZIP(appID uint32) ([]byte, bool) {
	if disableCache.Load() {
		return nil, false
	}
	dir := getCacheDir()
	if dir == "" {
		return nil, false
	}
	zipPath := filepath.Join(dir, "revobd", fmt.Sprintf("%d.zip", appID))
	info, err := os.Stat(zipPath)
	if err != nil || info.Size() == 0 {
		return nil, false
	}
	if time.Since(info.ModTime()) >= 30*24*time.Hour {
		slog.Debug("RevoBD ZIP cache expired (>30 days), will re-fetch", "appID", appID)
		return nil, false
	}
	data, err := os.ReadFile(zipPath)
	if err == nil && len(data) > 0 {
		slog.Info("Using cached RevoBD ZIP archive from disk (0 API requests consumed)", "appID", appID)
		return data, true
	}
	return nil, false
}

// SaveRevoBDZIP writes RevoBD ZIP archive bytes to local disk cache.
func SaveRevoBDZIP(appID uint32, data []byte) error {
	if disableCache.Load() || len(data) == 0 {
		return nil
	}
	dir := getCacheDir()
	if dir == "" {
		return nil
	}
	zipPath := filepath.Join(dir, "revobd", fmt.Sprintf("%d.zip", appID))
	slog.Debug("Caching RevoBD ZIP archive to disk", "appID", appID, "path", zipPath)
	return os.WriteFile(zipPath, data, 0644)
}

// ==========================================
// 5. SteamDepotDB Cache (24h TTL)
// ==========================================

// GetSteamDepotDB retrieves cached SteamDepotDB fallback keys if less than 24 hours old.
func GetSteamDepotDB() ([]byte, bool) {
	if disableCache.Load() {
		return nil, false
	}
	dir := getCacheDir()
	if dir == "" {
		return nil, false
	}
	path := filepath.Join(dir, "steamdepotdb", "fallback_depotkeys.json")
	return readTTLFile(path, 24*time.Hour)
}

// SaveSteamDepotDB saves SteamDepotDB fallback keys to disk for 24 hours.
func SaveSteamDepotDB(data []byte) error {
	if disableCache.Load() || len(data) == 0 {
		return nil
	}
	dir := getCacheDir()
	if dir == "" {
		return nil
	}
	path := filepath.Join(dir, "steamdepotdb", "fallback_depotkeys.json")
	return writeTTLFile(path, data)
}

// ==========================================
// 6. Manifesthub Cache (24h TTL)
// ==========================================

// GetManifesthubDB retrieves cached GitLab Manifesthub depot keys if less than 24 hours old.
func GetManifesthubDB() ([]byte, bool) {
	if disableCache.Load() {
		return nil, false
	}
	dir := getCacheDir()
	if dir == "" {
		return nil, false
	}
	path := filepath.Join(dir, "manifesthub", "depotkeys.json")
	return readTTLFile(path, 24*time.Hour)
}

// SaveManifesthubDB saves GitLab Manifesthub depot keys to disk for 24 hours.
func SaveManifesthubDB(data []byte) error {
	if disableCache.Load() || len(data) == 0 {
		return nil
	}
	dir := getCacheDir()
	if dir == "" {
		return nil
	}
	path := filepath.Join(dir, "manifesthub", "depotkeys.json")
	return writeTTLFile(path, data)
}

// ==========================================
// 7. GitHub Release Cache (1 Hour TTL)
// ==========================================

// GetGitHubRelease retrieves cached GitHub release metadata if less than 1 hour old.
func GetGitHubRelease(key string) ([]byte, bool) {
	if disableCache.Load() || key == "" {
		return nil, false
	}
	dir := getCacheDir()
	if dir == "" {
		return nil, false
	}
	path := filepath.Join(dir, "github", fmt.Sprintf("%s.json", key))
	return readTTLFile(path, 1*time.Hour)
}

// SaveGitHubRelease caches GitHub release metadata to disk for 1 hour.
func SaveGitHubRelease(key string, data []byte) error {
	if disableCache.Load() || key == "" || len(data) == 0 {
		return nil
	}
	dir := getCacheDir()
	if dir == "" {
		return nil
	}
	path := filepath.Join(dir, "github", fmt.Sprintf("%s.json", key))
	return writeTTLFile(path, data)
}

// ==========================================
// 8. SteamGridDB Artwork Metadata Cache (7 Days TTL)
// ==========================================

// GetSGDBArtwork retrieves cached SteamGridDB artwork metadata if less than 7 days old.
func GetSGDBArtwork(lookupKey string) ([]byte, bool) {
	if disableCache.Load() || lookupKey == "" {
		return nil, false
	}
	dir := getCacheDir()
	if dir == "" {
		return nil, false
	}
	path := filepath.Join(dir, "sgdb", fmt.Sprintf("%s.json", lookupKey))
	return readTTLFile(path, 7*24*time.Hour)
}

// SaveSGDBArtwork caches SteamGridDB artwork metadata to disk for 7 days.
func SaveSGDBArtwork(lookupKey string, data []byte) error {
	if disableCache.Load() || lookupKey == "" || len(data) == 0 {
		return nil
	}
	dir := getCacheDir()
	if dir == "" {
		return nil
	}
	path := filepath.Join(dir, "sgdb", fmt.Sprintf("%s.json", lookupKey))
	return writeTTLFile(path, data)
}

// ==========================================
// 9. Multi-Store Metadata Cache (24 Hours TTL)
// ==========================================

// GetStoreMetadata retrieves cached catalog metadata (e.g. GOG, RAWG) if less than 24 hours old.
func GetStoreMetadata(storeName, gameID string) ([]byte, bool) {
	if disableCache.Load() || storeName == "" || gameID == "" {
		return nil, false
	}
	dir := getCacheDir()
	if dir == "" {
		return nil, false
	}
	safeID := filepath.Clean(gameID)
	path := filepath.Join(dir, "store_meta", fmt.Sprintf("%s_%s.json", storeName, safeID))
	return readTTLFile(path, 24*time.Hour)
}

// SaveStoreMetadata caches catalog metadata to disk for 24 hours.
func SaveStoreMetadata(storeName, gameID string, data []byte) error {
	if disableCache.Load() || storeName == "" || gameID == "" || len(data) == 0 {
		return nil
	}
	dir := getCacheDir()
	if dir == "" {
		return nil
	}
	safeID := filepath.Clean(gameID)
	path := filepath.Join(dir, "store_meta", fmt.Sprintf("%s_%s.json", storeName, safeID))
	return writeTTLFile(path, data)
}

// ==========================================
// 10. Warm Library Cache
// ==========================================

// CachedGameRecord stores an inspected domain.Game entity validated against directory and profile timestamps.
type CachedGameRecord struct {
	DirMtime     int64        `json:"dir_mtime"`
	ProfileMtime int64        `json:"profile_mtime"`
	Game         *domain.Game `json:"game"`
}

func getGameCacheKey(gameDir string) string {
	h := sha256.Sum256([]byte(gameDir))
	return fmt.Sprintf("%x", h[:16])
}

// GetCachedGame retrieves a cached domain.Game record if dirMtime and profileMtime match.
func GetCachedGame(gameDir string, dirMtime int64, profileMtime int64) (*domain.Game, bool) {
	if disableCache.Load() || gameDir == "" {
		return nil, false
	}

	// 1. Fast in-memory check
	gamesMu.RLock()
	if cached, ok := gamesMemoryCache[gameDir]; ok {
		if cached.DirMtime == dirMtime && cached.ProfileMtime == profileMtime && cached.Game != nil {
			gamesMu.RUnlock()
			return cached.Game, true
		}
	}
	gamesMu.RUnlock()

	// 2. Disk cache check
	dir := getCacheDir()
	if dir == "" {
		return nil, false
	}
	cacheFile := filepath.Join(dir, "games", fmt.Sprintf("%s.json", getGameCacheKey(gameDir)))
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, false
	}

	var rec CachedGameRecord
	if err := json.Unmarshal(data, &rec); err == nil && rec.Game != nil {
		if rec.DirMtime == dirMtime && rec.ProfileMtime == profileMtime {
			gamesMu.Lock()
			gamesMemoryCache[gameDir] = rec
			gamesMu.Unlock()
			return rec.Game, true
		}
	}
	return nil, false
}

// SaveCachedGame persists an inspected domain.Game record with timestamps to in-memory and disk cache.
func SaveCachedGame(gameDir string, dirMtime int64, profileMtime int64, game *domain.Game) error {
	if disableCache.Load() || gameDir == "" || game == nil {
		return nil
	}

	rec := CachedGameRecord{
		DirMtime:     dirMtime,
		ProfileMtime: profileMtime,
		Game:         game,
	}

	gamesMu.Lock()
	gamesMemoryCache[gameDir] = rec
	gamesMu.Unlock()

	dir := getCacheDir()
	if dir == "" {
		return nil
	}
	cacheFile := filepath.Join(dir, "games", fmt.Sprintf("%s.json", getGameCacheKey(gameDir)))
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return os.WriteFile(cacheFile, data, 0644)
}

// InvalidateGameCache clears in-memory and disk cache entries for a specific game path.
func InvalidateGameCache(gameDir string) {
	if gameDir == "" {
		return
	}
	gamesMu.Lock()
	delete(gamesMemoryCache, gameDir)
	gamesMu.Unlock()

	dir := getCacheDir()
	if dir != "" {
		cacheFile := filepath.Join(dir, "games", fmt.Sprintf("%s.json", getGameCacheKey(gameDir)))
		_ = os.Remove(cacheFile)
	}
}

// ==========================================
// 11. Disk Cache Pruning
// ==========================================

// Prune removes expired cache files from all TTL-based subdirectories.
// For JSON caches using TTLWrapper, entries older than 2× their TTL are removed.
// For ZIP caches (hubcap, revobd), files older than 2× 30 days are removed.
// The "games" subdirectory is skipped since it uses mtime-based invalidation.
func Prune() (removed int) {
	dir := getCacheDir()
	if dir == "" {
		return 0
	}

	for subdir, ttl := range subdirTTLs {
		maxAge := 2 * ttl
		subdirPath := filepath.Join(dir, subdir)

		entries, err := os.ReadDir(subdirPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			filePath := filepath.Join(subdirPath, entry.Name())

			if isExpiredCacheFile(filePath, subdir, maxAge) {
				if err := os.Remove(filePath); err == nil {
					removed++
					slog.Debug("Pruned expired cache file", "path", filePath, "subdir", subdir)
				}
			}
		}
	}

	if removed > 0 {
		slog.Info("Cache pruning completed", "removed", removed)
	}
	return removed
}

// isExpiredCacheFile checks whether a cache file has exceeded maxAge.
// For ZIP files (hubcap, revobd), uses file mtime.
// For JSON TTLWrapper files, reads the embedded timestamp.
func isExpiredCacheFile(path, subdir string, maxAge time.Duration) bool {
	// ZIP archives use file mtime
	if subdir == "hubcap" || subdir == "revobd" {
		info, err := os.Stat(path)
		if err != nil {
			return false
		}
		return time.Since(info.ModTime()) >= maxAge
	}

	// TTLWrapper JSON files: read the embedded timestamp
	if !strings.HasSuffix(path, ".json") {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var wrapper TTLWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		// Can't parse → treat as stale
		return true
	}
	if wrapper.Timestamp.IsZero() {
		return true
	}
	return time.Since(wrapper.Timestamp) >= maxAge
}
