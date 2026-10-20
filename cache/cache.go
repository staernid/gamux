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

	// In-memory thread-safe cache for inspected game records (Warm Library Cache)
	gamesMu          sync.RWMutex
	gamesMemoryCache = make(map[string]CachedGameRecord)

	resettableCaches []interface{ reset() }
)

type ttlCache[K comparable] struct {
	mu       sync.RWMutex
	mem      map[K][]byte
	inMemory bool
	subdir   string
	ttl      time.Duration
	isZero   func(K) bool
	fileFn   func(K) string
}

func newTTLCache[K comparable](subdir string, ttl time.Duration, inMemory bool, isZero func(K) bool, fileFn func(K) string) *ttlCache[K] {
	c := &ttlCache[K]{
		subdir:   subdir,
		ttl:      ttl,
		inMemory: inMemory,
		isZero:   isZero,
		fileFn:   fileFn,
	}
	if inMemory {
		c.mem = make(map[K][]byte)
	}
	resettableCaches = append(resettableCaches, c)
	return c
}

func (c *ttlCache[K]) reset() {
	if !c.inMemory {
		return
	}
	c.mu.Lock()
	c.mem = make(map[K][]byte)
	c.mu.Unlock()
}

func (c *ttlCache[K]) get(key K) ([]byte, bool) {
	if disableCache.Load() || (c.isZero != nil && c.isZero(key)) {
		return nil, false
	}

	if c.inMemory {
		c.mu.RLock()
		if cached, ok := c.mem[key]; ok && len(cached) > 0 {
			c.mu.RUnlock()
			return cached, true
		}
		c.mu.RUnlock()
	}

	dir := getCacheDir()
	if dir == "" {
		return nil, false
	}
	path := filepath.Join(dir, c.subdir, c.fileFn(key))
	if data, ok := readTTLFile(path, c.ttl); ok {
		if c.inMemory {
			c.mu.Lock()
			c.mem[key] = data
			c.mu.Unlock()
		}
		return data, true
	}
	return nil, false
}

func (c *ttlCache[K]) save(key K, data []byte) error {
	if disableCache.Load() || (c.isZero != nil && c.isZero(key)) || len(data) == 0 {
		return nil
	}

	if c.inMemory {
		c.mu.Lock()
		c.mem[key] = data
		c.mu.Unlock()
	}

	dir := getCacheDir()
	if dir == "" {
		return nil
	}
	path := filepath.Join(dir, c.subdir, c.fileFn(key))
	return writeTTLFile(path, data)
}

var (
	nonEmptyStr   = func(k string) bool { return k == "" }
	zeroUint32    = func(k uint32) bool { return k == 0 }
	jsonKeyFile   = func(k string) string { return fmt.Sprintf("%s.json", k) }
	uint32KeyFile = func(k uint32) string { return fmt.Sprintf("%d.json", k) }

	detailsCache        = newTTLCache("appdetails", 7*24*time.Hour, true, nonEmptyStr, jsonKeyFile)
	newsCache           = newTTLCache("news", 24*time.Hour, false, nonEmptyStr, jsonKeyFile)
	newsItemsCache      = newTTLCache("news", 12*time.Hour, true, nonEmptyStr, func(k string) string { return fmt.Sprintf("%s_items.json", k) })
	achPercentagesCache = newTTLCache("achievements", 7*24*time.Hour, true, nonEmptyStr, func(k string) string { return fmt.Sprintf("%s_percentages.json", k) })
	versionsCache       = newTTLCache("versions", 30*time.Minute, true, nonEmptyStr, func(k string) string { return fmt.Sprintf("%s.json", strings.ReplaceAll(k, ":", "_")) })
	branchesCache       = newTTLCache("branches", 6*time.Hour, true, zeroUint32, uint32KeyFile)
	depotsCache         = newTTLCache("depots", 6*time.Hour, true, zeroUint32, uint32KeyFile)
	launchCache         = newTTLCache("launch", 6*time.Hour, true, zeroUint32, uint32KeyFile)
	steamDepotDBCache   = newTTLCache("steamdepotdb", 24*time.Hour, false, nil, func(_ string) string { return "fallback_depotkeys.json" })
	manifesthubCache    = newTTLCache("manifesthub", 24*time.Hour, false, nil, func(_ string) string { return "depotkeys.json" })
	githubReleaseCache  = newTTLCache("github", 1*time.Hour, false, nonEmptyStr, jsonKeyFile)
	sgdbArtworkCache    = newTTLCache("sgdb", 7*24*time.Hour, false, nonEmptyStr, jsonKeyFile)
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
	"achievements",
	"versions",
	"branches",
	"depots",
	"launch",
}

// subdirTTLs maps each cache subdirectory to its canonical TTL.
// Used by Prune to determine which entries are expired.
var subdirTTLs = map[string]time.Duration{
	"appdetails":   7 * 24 * time.Hour,
	"news":         24 * time.Hour,
	"github":       1 * time.Hour,
	"steamdepotdb": 24 * time.Hour,
	"manifesthub":  24 * time.Hour,
	"sgdb":         7 * 24 * time.Hour,
	"store_meta":   24 * time.Hour,
	"hubcap":       30 * 24 * time.Hour,
	"revobd":       30 * 24 * time.Hour,
	"achievements": 7 * 24 * time.Hour,
	"versions":     30 * time.Minute,
	"branches":     6 * time.Hour,
	"depots":       6 * time.Hour,
	"launch":       6 * time.Hour,
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

	gamesMu.Lock()
	gamesMemoryCache = make(map[string]CachedGameRecord)
	gamesMu.Unlock()

	for _, c := range resettableCaches {
		c.reset()
	}
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
// 2. Steam & Provider TTL Caches
// ==========================================

// GetAppDetails retrieves cached Steam store AppDetails JSON if less than 7 days old.
func GetAppDetails(appID string) ([]byte, bool) { return detailsCache.get(appID) }

// SaveAppDetails writes Steam store AppDetails JSON to in-memory and disk cache for 7 days.
func SaveAppDetails(appID string, data []byte) error { return detailsCache.save(appID, data) }

// GetNews retrieves cached Steam patch notes if less than 24 hours old.
func GetNews(appID string) (string, bool) {
	data, ok := newsCache.get(appID)
	return string(data), ok
}

// SaveNews caches Steam patch notes to disk for 24 hours.
func SaveNews(appID, news string) { _ = newsCache.save(appID, []byte(news)) }

// GetNewsItems retrieves cached Steam patch notes list if less than 12 hours old.
func GetNewsItems(appID string) ([]byte, bool) { return newsItemsCache.get(appID) }

// SaveNewsItems caches Steam patch notes list to in-memory and disk cache for 12 hours.
func SaveNewsItems(appID string, itemsJSON []byte) {
	if disableCache.Load() || appID == "" || len(itemsJSON) == 0 {
		return
	}

	// Never downgrade cache: if existing cache has more items and is valid, keep it
	if existingBytes, ok := GetNewsItems(appID); ok {
		var existingItems, newItems []json.RawMessage
		if json.Unmarshal(existingBytes, &existingItems) == nil && json.Unmarshal(itemsJSON, &newItems) == nil {
			if len(existingItems) > len(newItems) {
				return
			}
		}
	}
	_ = newsItemsCache.save(appID, itemsJSON)
}

// GetAchievementPercentages retrieves cached Steam global achievement percentages if less than 7 days old.
func GetAchievementPercentages(appID string) ([]byte, bool) { return achPercentagesCache.get(appID) }

// SaveAchievementPercentages caches Steam global achievement percentages to in-memory and disk cache for 7 days.
func SaveAchievementPercentages(appID string, data []byte) { _ = achPercentagesCache.save(appID, data) }

// GetVersionStatus retrieves cached Steam version status if less than 30 minutes old.
func GetVersionStatus(key string) ([]byte, bool) { return versionsCache.get(key) }

// SaveVersionStatus caches Steam version status to in-memory and disk cache for 30 minutes.
func SaveVersionStatus(key string, data []byte) { _ = versionsCache.save(key, data) }

// GetSteamBranches retrieves cached Steam branches JSON if less than 6 hours old.
func GetSteamBranches(appID uint32) ([]byte, bool) { return branchesCache.get(appID) }

// SaveSteamBranches caches Steam branches to in-memory and disk cache for 6 hours.
func SaveSteamBranches(appID uint32, data []byte) { _ = branchesCache.save(appID, data) }

// GetSteamDepots retrieves cached Steam depots metadata JSON if less than 6 hours old.
func GetSteamDepots(appID uint32) ([]byte, bool) { return depotsCache.get(appID) }

// SaveSteamDepots caches Steam depots metadata to in-memory and disk cache for 6 hours.
func SaveSteamDepots(appID uint32, data []byte) { _ = depotsCache.save(appID, data) }

// GetSteamLaunch retrieves cached Steam launch options JSON if less than 6 hours old.
func GetSteamLaunch(appID uint32) ([]byte, bool) { return launchCache.get(appID) }

// SaveSteamLaunch caches Steam launch options to in-memory and disk cache for 6 hours.
func SaveSteamLaunch(appID uint32, data []byte) { _ = launchCache.save(appID, data) }

// GetSteamDepotDB retrieves cached SteamDepotDB fallback keys if less than 24 hours old.
func GetSteamDepotDB() ([]byte, bool) { return steamDepotDBCache.get("") }

// SaveSteamDepotDB saves SteamDepotDB fallback keys to disk for 24 hours.
func SaveSteamDepotDB(data []byte) error { return steamDepotDBCache.save("", data) }

// GetManifesthubDB retrieves cached GitLab Manifesthub depot keys if less than 24 hours old.
func GetManifesthubDB() ([]byte, bool) { return manifesthubCache.get("") }

// SaveManifesthubDB saves GitLab Manifesthub depot keys to disk for 24 hours.
func SaveManifesthubDB(data []byte) error { return manifesthubCache.save("", data) }

// GetGitHubRelease retrieves cached GitHub release metadata if less than 1 hour old.
func GetGitHubRelease(key string) ([]byte, bool) { return githubReleaseCache.get(key) }

// SaveGitHubRelease caches GitHub release metadata to disk for 1 hour.
func SaveGitHubRelease(key string, data []byte) error { return githubReleaseCache.save(key, data) }

// GetSGDBArtwork retrieves cached SteamGridDB artwork metadata if less than 7 days old.
func GetSGDBArtwork(lookupKey string) ([]byte, bool) { return sgdbArtworkCache.get(lookupKey) }

// SaveSGDBArtwork caches SteamGridDB artwork metadata to disk for 7 days.
func SaveSGDBArtwork(lookupKey string, data []byte) error {
	return sgdbArtworkCache.save(lookupKey, data)
}

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
// 3. Hubcap & RevoBD Manifest ZIP Cache (30 Days)
// ==========================================

func getZIPCache(subdir, label string, appID uint32) ([]byte, bool) {
	if disableCache.Load() {
		return nil, false
	}
	dir := getCacheDir()
	if dir == "" {
		return nil, false
	}
	zipPath := filepath.Join(dir, subdir, fmt.Sprintf("%d.zip", appID))
	info, err := os.Stat(zipPath)
	if err != nil || info.Size() == 0 {
		return nil, false
	}
	if time.Since(info.ModTime()) >= 30*24*time.Hour {
		slog.Debug(label+" ZIP cache expired (>30 days), will re-fetch", "appID", appID)
		return nil, false
	}
	data, err := os.ReadFile(zipPath)
	if err == nil && len(data) > 0 {
		slog.Info("Using cached "+label+" ZIP archive from disk (0 API requests consumed)", "appID", appID)
		return data, true
	}
	return nil, false
}

func saveZIPCache(subdir, label string, appID uint32, data []byte) error {
	if disableCache.Load() || len(data) == 0 {
		return nil
	}
	dir := getCacheDir()
	if dir == "" {
		return nil
	}
	zipPath := filepath.Join(dir, subdir, fmt.Sprintf("%d.zip", appID))
	slog.Debug("Caching "+label+" ZIP archive to disk", "appID", appID, "path", zipPath)
	return os.WriteFile(zipPath, data, 0644)
}

// GetHubcapZIP retrieves cached Hubcap ZIP archive bytes if available and less than 30 days old.
func GetHubcapZIP(appID uint32) ([]byte, bool) { return getZIPCache("hubcap", "Hubcap", appID) }

// SaveHubcapZIP writes Hubcap ZIP archive bytes to local disk cache.
func SaveHubcapZIP(appID uint32, data []byte) error {
	return saveZIPCache("hubcap", "Hubcap", appID, data)
}

// GetRevoBDZIP retrieves cached RevoBD ZIP archive bytes if available and less than 30 days old.
func GetRevoBDZIP(appID uint32) ([]byte, bool) { return getZIPCache("revobd", "RevoBD", appID) }

// SaveRevoBDZIP writes RevoBD ZIP archive bytes to local disk cache.
func SaveRevoBDZIP(appID uint32, data []byte) error {
	return saveZIPCache("revobd", "RevoBD", appID, data)
}

// ==========================================
// 4. Warm Library Cache
// ==========================================

// CachedGameRecord stores an inspected domain.Game entity validated against sentinel signature and executable stats.
type CachedGameRecord struct {
	Signature string       `json:"signature"`
	ExePath   string       `json:"exe_path,omitempty"`
	ExeMtime  int64        `json:"exe_mtime,omitempty"`
	ExeSize   int64        `json:"exe_size,omitempty"`
	Game      *domain.Game `json:"game"`
}

func isExecutableValid(rec CachedGameRecord) bool {
	if rec.ExePath == "" {
		return true
	}
	fi, err := os.Stat(rec.ExePath)
	if err != nil {
		return false
	}
	return fi.ModTime().UnixNano() == rec.ExeMtime && fi.Size() == rec.ExeSize
}

func getGameCacheKey(gameDir string) string {
	h := sha256.Sum256([]byte(gameDir))
	return fmt.Sprintf("%x", h[:16])
}

// GetCachedGame retrieves a cached domain.Game record if signature matches and executable is intact.
func GetCachedGame(gameDir string, signature string) (*domain.Game, bool) {
	if disableCache.Load() || gameDir == "" {
		return nil, false
	}

	// 1. Fast in-memory check
	gamesMu.RLock()
	if cached, ok := gamesMemoryCache[gameDir]; ok {
		if cached.Signature == signature && cached.Game != nil && isExecutableValid(cached) {
			cloned := cached.Game.Clone()
			gamesMu.RUnlock()
			return cloned, true
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
		if rec.Signature == signature && isExecutableValid(rec) {
			gamesMu.Lock()
			gamesMemoryCache[gameDir] = rec
			gamesMu.Unlock()
			return rec.Game.Clone(), true
		}
	}
	return nil, false
}

// SaveCachedGame persists an inspected domain.Game record with sentinel signature to in-memory and disk cache.
func SaveCachedGame(gameDir string, signature string, game *domain.Game) error {
	if disableCache.Load() || gameDir == "" || game == nil {
		return nil
	}

	var exePath string
	var exeMtime, exeSize int64
	if effExe := game.EffectiveExePath(); effExe != "" {
		resolvedExe := effExe
		if !filepath.IsAbs(resolvedExe) {
			resolvedExe = filepath.Join(gameDir, resolvedExe)
		}
		if fi, err := os.Stat(resolvedExe); err == nil && !fi.IsDir() {
			exePath = resolvedExe
			exeMtime = fi.ModTime().UnixNano()
			exeSize = fi.Size()
		}
	}

	rec := CachedGameRecord{
		Signature: signature,
		ExePath:   exePath,
		ExeMtime:  exeMtime,
		ExeSize:   exeSize,
		Game:      game.Clone(),
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
// 5. Disk Cache Pruning
// ==========================================

// Prune removes expired cache files from all TTL-based subdirectories.
// For JSON caches using TTLWrapper, entries older than 2× their TTL are removed.
// For ZIP caches (hubcap, revobd), files older than 2× 30 days are removed.
// The "games" subdirectory is skipped since it uses sentinel-based invalidation.
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

func isExpiredCacheFile(path, subdir string, maxAge time.Duration) bool {
	if subdir == "hubcap" || subdir == "revobd" {
		info, err := os.Stat(path)
		if err != nil {
			return false
		}
		return time.Since(info.ModTime()) >= maxAge
	}

	if !strings.HasSuffix(path, ".json") {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var wrapper TTLWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return true
	}
	if wrapper.Timestamp.IsZero() {
		return true
	}
	return time.Since(wrapper.Timestamp) >= maxAge
}
