package profiles

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/util"
)

// ProfilesData represents the persisted profiles metadata file.
type ProfilesData struct {
	ActiveProfileID string            `json:"active_profile_id"`
	Profiles        []*domain.Profile `json:"profiles"`
}

// Manager manages user profiles, filesystem isolation homes, and active persona sync.
type Manager struct {
	mu         sync.RWMutex
	baseDir    string
	configFile string
	activeID   string
	profiles   []*domain.Profile
	cfg        *config.Config
}

// DefaultBaseDir returns the canonical profiles base directory (~/.local/share/gamux/profiles).
func DefaultBaseDir() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			dataHome = filepath.Join(homeDir, ".local", "share")
		} else {
			dataHome = filepath.Join(os.TempDir(), ".local", "share")
		}
	}
	return filepath.Join(dataHome, "gamux", "profiles")
}

// DefaultConfigFile returns ~/.config/gamux/profiles.json respecting XDG_CONFIG_HOME.
func DefaultConfigFile() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configHome = filepath.Join(homeDir, ".config")
		} else {
			configHome = filepath.Join(os.TempDir(), ".config")
		}
	}
	return filepath.Join(configHome, "gamux", "profiles.json")
}

// NewManager creates a profile manager for the given base directory and configuration.
func NewManager(baseDir string, cfg *config.Config) (*Manager, error) {
	configFile := DefaultConfigFile()
	if baseDir == "" {
		baseDir = DefaultBaseDir()
	} else {
		configFile = filepath.Join(baseDir, "profiles.json")
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create profiles base directory: %w", err)
	}

	m := &Manager{
		baseDir:    baseDir,
		configFile: configFile,
		cfg:        cfg,
	}

	if err := m.Load(); err != nil {
		return nil, fmt.Errorf("load profiles: %w", err)
	}

	return m, nil
}

// dataFilePath returns the path to profiles.json.
func (m *Manager) dataFilePath() string {
	if m.configFile != "" {
		return m.configFile
	}
	return DefaultConfigFile()
}

// Load loads profiles from disk, or auto-seeds defaults if not present.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dataFile := m.dataFilePath()
	if !util.FileExists(dataFile) {
		// Migrate from legacy location if present (~/.local/share/gamux/profiles/profiles.json)
		legacyFile := filepath.Join(m.baseDir, "profiles.json")
		if util.FileExists(legacyFile) {
			slog.Info("Migrating profiles.json from data directory to config directory", "legacy", legacyFile, "new", dataFile)
			if data, err := os.ReadFile(legacyFile); err == nil {
				_ = os.MkdirAll(filepath.Dir(dataFile), 0755)
				_ = os.WriteFile(dataFile, data, 0644)
				_ = os.Remove(legacyFile)
			}
		}
	}

	if util.FileExists(dataFile) {
		data, err := os.ReadFile(dataFile)
		if err != nil {
			return fmt.Errorf("read profiles.json: %w", err)
		}
		var pd ProfilesData
		if err := json.Unmarshal(data, &pd); err != nil {
			return fmt.Errorf("parse profiles.json: %w", err)
		}

		m.profiles = pd.Profiles
		m.activeID = pd.ActiveProfileID

		// Ensure main profile always exists
		hasMain := false
		for _, p := range m.profiles {
			if p.IsMain || p.ID == "main" {
				hasMain = true
				p.IsMain = true
				p.Index = 1
				break
			}
		}
		if !hasMain {
			mainProf := m.seedMainProfile()
			m.profiles = append([]*domain.Profile{mainProf}, m.profiles...)
		}

		if m.activeID == "" || m.findProfileLocked(m.activeID) == nil {
			m.activeID = "main"
		}
		return nil
	}

	// Auto-seed initial profiles
	mainProf := m.seedMainProfile()
	p2Prof := m.seedP2Profile()

	m.profiles = []*domain.Profile{mainProf, p2Prof}
	m.activeID = "main"

	return m.saveLocked()
}

func (m *Manager) seedMainProfile() *domain.Profile {
	// Try reading host Goldberg user config first
	name := "GamuxPlayer"
	steamID := "76561197960287930"
	lang := "english"
	country := "US"

	gseCfg, err := gbe.LoadGSEUserConfig()
	if err == nil && gseCfg != nil {
		if gseCfg.AccountName != "" {
			name = gseCfg.AccountName
		}
		if gseCfg.AccountSteamID != "" {
			steamID = gseCfg.AccountSteamID
		}
		if gseCfg.Language != "" {
			lang = gseCfg.Language
		}
		if gseCfg.IPCountry != "" {
			country = gseCfg.IPCountry
		}
	}

	p := &domain.Profile{
		ID:        "main",
		Index:     1,
		Name:      name,
		SteamID:   steamID,
		Language:  lang,
		IPCountry: country,
		IsMain:    true,
	}
	p.Normalize()
	return p
}

func (m *Manager) seedP2Profile() *domain.Profile {
	name := "PlayerTwo"
	steamID := "76561198998877665"
	lang := "english"
	country := "US"

	// Single source of truth: check if isolated profile already has a configs.user.ini
	p2GseIni := filepath.Join(m.baseDir, "p2", "home", ".local", "share", "GSE Saves", "settings", "configs.user.ini")
	if gseCfg, err := gbe.LoadGSEUserConfigFromPath(p2GseIni); err == nil && gseCfg != nil {
		if gseCfg.AccountName != "" {
			name = gseCfg.AccountName
		}
		if gseCfg.AccountSteamID != "" {
			steamID = gseCfg.AccountSteamID
		}
		if gseCfg.Language != "" {
			lang = gseCfg.Language
		}
		if gseCfg.IPCountry != "" {
			country = gseCfg.IPCountry
		}
	}

	p := &domain.Profile{
		ID:        "p2",
		Index:     2,
		Name:      name,
		SteamID:   steamID,
		Language:  lang,
		IPCountry: country,
		IsMain:    false,
	}
	p.Normalize()
	return p
}

func (m *Manager) findProfileLocked(id string) *domain.Profile {
	for _, p := range m.profiles {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (m *Manager) saveLocked() error {
	pd := ProfilesData{
		ActiveProfileID: m.activeID,
		Profiles:        m.profiles,
	}
	data, err := json.MarshalIndent(pd, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profiles: %w", err)
	}

	dataFile := m.dataFilePath()
	if err := os.MkdirAll(filepath.Dir(dataFile), 0755); err != nil {
		return fmt.Errorf("create profiles config directory: %w", err)
	}

	tmpPath := dataFile + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write profiles tmp file: %w", err)
	}
	if err := os.Rename(tmpPath, m.dataFilePath()); err != nil {
		return fmt.Errorf("commit profiles file: %w", err)
	}
	return nil
}

// List returns a copy of all registered profiles.
func (m *Manager) List() []*domain.Profile {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*domain.Profile, len(m.profiles))
	for i, p := range m.profiles {
		cp := *p
		out[i] = &cp
	}
	return out
}

// Get returns the profile with the given ID.
func (m *Manager) Get(id string) (*domain.Profile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p := m.findProfileLocked(id)
	if p == nil {
		return nil, fmt.Errorf("profile not found: %s", id)
	}
	cp := *p
	return &cp, nil
}

// GetActive returns the currently active profile.
func (m *Manager) GetActive() *domain.Profile {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p := m.findProfileLocked(m.activeID)
	if p == nil && len(m.profiles) > 0 {
		p = m.profiles[0]
	}
	if p == nil {
		return m.seedMainProfile()
	}
	cp := *p
	return &cp
}

// SetActive sets the active profile ID, persists it, and materializes its files.
func (m *Manager) SetActive(id string) error {
	m.mu.Lock()
	p := m.findProfileLocked(id)
	if p == nil {
		m.mu.Unlock()
		return fmt.Errorf("profile not found: %s", id)
	}
	m.activeID = id
	if err := m.saveLocked(); err != nil {
		m.mu.Unlock()
		return err
	}
	m.mu.Unlock()

	return m.Materialize(id)
}

// Create registers a new secondary profile with the next available index and materializes it.
func (m *Manager) Create(name, steamID, language, country, avatarURL string) (*domain.Profile, error) {
	m.mu.Lock()

	maxIndex := 1
	for _, p := range m.profiles {
		if p.Index > maxIndex {
			maxIndex = p.Index
		}
	}
	nextIndex := maxIndex + 1
	id := fmt.Sprintf("p%d", nextIndex)

	newProf := &domain.Profile{
		ID:        id,
		Index:     nextIndex,
		Name:      name,
		SteamID:   steamID,
		Language:  language,
		IPCountry: country,
		AvatarURL: avatarURL,
		IsMain:    false,
	}
	newProf.Normalize()

	m.profiles = append(m.profiles, newProf)
	if err := m.saveLocked(); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	m.mu.Unlock()

	if err := m.Materialize(id); err != nil {
		slog.Warn("Failed to materialize newly created profile", "id", id, "error", err)
	}

	cp := *newProf
	return &cp, nil
}

// Update updates an existing profile's mutable fields.
func (m *Manager) Update(updated *domain.Profile) error {
	if updated == nil {
		return fmt.Errorf("profile cannot be nil")
	}

	m.mu.Lock()
	p := m.findProfileLocked(updated.ID)
	if p == nil {
		m.mu.Unlock()
		return fmt.Errorf("profile not found: %s", updated.ID)
	}

	p.Name = updated.Name
	p.SteamID = updated.SteamID
	p.Language = updated.Language
	p.IPCountry = updated.IPCountry
	p.AvatarURL = updated.AvatarURL
	p.Normalize()

	if err := m.saveLocked(); err != nil {
		m.mu.Unlock()
		return err
	}
	isActive := m.activeID == p.ID
	isMain := p.IsMain
	m.mu.Unlock()

	// Materialize if it's the active profile or main profile
	if isActive || isMain {
		return m.Materialize(updated.ID)
	}
	return nil
}

// Delete removes a non-main profile. Returns error if is_main is true.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	p := m.findProfileLocked(id)
	if p == nil {
		m.mu.Unlock()
		return fmt.Errorf("profile not found: %s", id)
	}
	if p.IsMain || p.ID == "main" {
		m.mu.Unlock()
		return fmt.Errorf("cannot delete main profile")
	}

	filtered := make([]*domain.Profile, 0, len(m.profiles)-1)
	for _, prof := range m.profiles {
		if prof.ID != id {
			filtered = append(filtered, prof)
		}
	}
	m.profiles = filtered

	if m.activeID == id {
		m.activeID = "main"
	}

	if err := m.saveLocked(); err != nil {
		m.mu.Unlock()
		return err
	}
	m.mu.Unlock()

	// Clean up profile storage directory
	profHome := filepath.Join(m.baseDir, id)
	_ = os.RemoveAll(profHome)

	return nil
}

// ResolveProfileHome returns the filesystem home directory for a profile ID ("main", "p2", etc.).
// For "main", "p1", or empty string, returns the host user's home directory.
// For secondary profiles ("p2", etc.), returns ~/.local/share/gamux/profiles/<seatID>/home.
func ResolveProfileHome(profileID string) string {
	seatID := compat.SeatIDFromProfileID(profileID)
	if seatID == "p1" {
		home, _ := os.UserHomeDir()
		return home
	}
	return filepath.Join(DefaultBaseDir(), seatID, "home")
}

// ResolveProfileEnv returns environment variable overrides (HOME, XDG_*) for the given profile ID.
// For secondary profiles, it redirects HOME, XDG_DATA_HOME, XDG_CONFIG_HOME, and XDG_CACHE_HOME.
// For "main", "p1", or empty string, it returns an empty map (host environment is retained).
func ResolveProfileEnv(profileID string) map[string]string {
	seatID := compat.SeatIDFromProfileID(profileID)
	if seatID == "p1" {
		return map[string]string{}
	}
	profileHome := ResolveProfileHome(profileID)
	if !util.FileExists(profileHome) {
		return map[string]string{}
	}
	env := map[string]string{
		"HOME":            profileHome,
		"XDG_DATA_HOME":   filepath.Join(profileHome, ".local", "share"),
		"XDG_CONFIG_HOME": filepath.Join(profileHome, ".config"),
		"XDG_CACHE_HOME":  filepath.Join(profileHome, ".cache"),
	}
	hostHome, _ := os.UserHomeDir()
	if hostHome != "" {
		hostCompat := filepath.Join(hostHome, ".local", "share", "Steam", "compatibilitytools.d")
		if !util.FileExists(hostCompat) {
			hostCompat = filepath.Join(hostHome, ".steam", "steam", "compatibilitytools.d")
		}
		if util.FileExists(hostCompat) {
			env["STEAM_EXTRA_COMPAT_TOOLS_PATHS"] = hostCompat
		}
	}
	return env
}

// MaterializeIsolatedHome prepares the isolated home directory and Goldberg persona files for a profile/seat.
func MaterializeIsolatedHome(homeDir string, userCfg *gbe.GSEUserConfig) error {
	if homeDir == "" {
		return fmt.Errorf("home directory cannot be empty")
	}
	gseDir := filepath.Join(homeDir, ".local", "share", "GSE Saves", "settings")
	if err := os.MkdirAll(gseDir, 0755); err != nil {
		return fmt.Errorf("create gse settings dir: %w", err)
	}

	if userCfg == nil {
		userCfg = &gbe.GSEUserConfig{
			AccountName:    "Player",
			AccountSteamID: "76561197960287930",
			Language:       "english",
			IPCountry:      "US",
		}
	}
	if userCfg.AccountName == "" {
		userCfg.AccountName = "Player"
	}
	if userCfg.AccountSteamID == "" {
		userCfg.AccountSteamID = "76561197960287930"
	}
	if userCfg.Language == "" {
		userCfg.Language = "english"
	}
	if userCfg.IPCountry == "" {
		userCfg.IPCountry = "US"
	}

	iniPath := filepath.Join(gseDir, "configs.user.ini")
	if err := gbe.SaveGSEUserConfigToPath(iniPath, userCfg); err != nil {
		return fmt.Errorf("write configs.user.ini: %w", err)
	}

	_ = os.WriteFile(filepath.Join(gseDir, "account_name.txt"), []byte(userCfg.AccountName), 0644)
	_ = os.WriteFile(filepath.Join(gseDir, "user_steam_id.txt"), []byte(userCfg.AccountSteamID), 0644)
	_ = os.WriteFile(filepath.Join(gseDir, "language.txt"), []byte(userCfg.Language), 0644)
	_ = os.WriteFile(filepath.Join(gseDir, "custom_broadcasts.txt"), []byte("127.0.0.1\n10.0.2.2\n"), 0644)

	// Ensure steam_settings symlink points to settings inside GSE Saves
	gseRoot := filepath.Join(homeDir, ".local", "share", "GSE Saves")
	steamSettingsLink := filepath.Join(gseRoot, "steam_settings")
	if fi, err := os.Lstat(steamSettingsLink); err != nil {
		_ = os.Symlink("settings", steamSettingsLink)
	} else if fi.Mode()&os.ModeSymlink == 0 {
		_ = os.RemoveAll(steamSettingsLink)
		_ = os.Symlink("settings", steamSettingsLink)
	}

	// Ensure Goldberg experimental overlay and global emulator configs exist in the isolated profile
	hostHome, _ := os.UserHomeDir()
	targetOverlay := filepath.Join(gseDir, "configs.overlay.ini")
	if !util.FileExists(targetOverlay) {
		if hostHome != "" && hostHome != homeDir {
			hostOverlay := filepath.Join(hostHome, ".local", "share", "GSE Saves", "settings", "configs.overlay.ini")
			if util.FileExists(hostOverlay) {
				_ = util.CopyFile(hostOverlay, targetOverlay)
			}
		}
		if !util.FileExists(targetOverlay) {
			_ = os.WriteFile(targetOverlay, []byte("[overlay::general]\nenable_experimental_overlay=1\n"), 0644)
		}
	}

	targetMain := filepath.Join(gseDir, "configs.main.ini")
	if !util.FileExists(targetMain) && hostHome != "" && hostHome != homeDir {
		hostMain := filepath.Join(hostHome, ".local", "share", "GSE Saves", "settings", "configs.main.ini")
		if util.FileExists(hostMain) {
			_ = util.CopyFile(hostMain, targetMain)
		}
	}

	_ = os.MkdirAll(filepath.Join(homeDir, ".cache"), 0755)
	_ = os.MkdirAll(filepath.Join(homeDir, ".config"), 0755)
	_ = os.MkdirAll(filepath.Join(homeDir, ".local", "share"), 0755)

	// Clean up any stale compatibilitytools.d symlinks to ensure bubblewrap mounts succeed
	seatCompat := filepath.Join(homeDir, ".local", "share", "Steam", "compatibilitytools.d")
	if fi, err := os.Lstat(seatCompat); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		_ = os.Remove(seatCompat)
	}
	_ = os.MkdirAll(seatCompat, 0755)
	_ = os.MkdirAll(filepath.Join(homeDir, ".local", "share", "umu"), 0755)

	return nil
}

// GetHomeDir returns the home directory for the given profile.
// For Main profile: returns the host user's actual home directory.
// For isolated profiles (p2, p3...): returns ~/.local/share/gamux/profiles/<id>/home.
func (m *Manager) GetHomeDir(id string) string {
	m.mu.RLock()
	p := m.findProfileLocked(id)
	m.mu.RUnlock()

	if p == nil || p.IsMain || p.ID == "main" || p.ID == "p1" {
		home, _ := os.UserHomeDir()
		return home
	}
	return filepath.Join(m.baseDir, compat.SeatIDFromProfileID(id), "home")
}

// GetEnv returns environment variable overrides (HOME, XDG_*) for the given profile ID.
func (m *Manager) GetEnv(id string) map[string]string {
	m.mu.RLock()
	p := m.findProfileLocked(id)
	m.mu.RUnlock()

	if p == nil || p.IsMain || p.ID == "main" || p.ID == "p1" {
		return map[string]string{}
	}
	home := m.GetHomeDir(id)
	if !util.FileExists(home) {
		return map[string]string{}
	}
	return map[string]string{
		"HOME":            home,
		"XDG_DATA_HOME":   filepath.Join(home, ".local", "share"),
		"XDG_CONFIG_HOME": filepath.Join(home, ".config"),
		"XDG_CACHE_HOME":  filepath.Join(home, ".cache"),
	}
}

// GetPrefixDir returns the Wine prefix directory for the given profile.
// For all profiles, prefixes are stored in ~/Games/umu relative to the profile's home directory.
func (m *Manager) GetPrefixDir(id string) string {
	home := m.GetHomeDir(id)
	return filepath.Join(home, "Games", "umu")
}

// Materialize writes the Goldberg persona settings on disk for the specified profile.
func (m *Manager) Materialize(id string) error {
	m.mu.RLock()
	p := m.findProfileLocked(id)
	m.mu.RUnlock()

	if p == nil {
		return fmt.Errorf("profile not found: %s", id)
	}

	homeDir := m.GetHomeDir(id)
	userCfg := &gbe.GSEUserConfig{
		AccountName:    p.Name,
		AccountSteamID: p.SteamID,
		Language:       p.Language,
		IPCountry:      p.IPCountry,
	}

	if p.IsMain || p.ID == "main" || p.ID == "p1" {
		gseDir := filepath.Join(homeDir, ".local", "share", "GSE Saves", "settings")
		if err := os.MkdirAll(gseDir, 0755); err != nil {
			return fmt.Errorf("create gse settings dir: %w", err)
		}

		iniPath := filepath.Join(gseDir, "configs.user.ini")
		if err := gbe.SaveGSEUserConfigToPath(iniPath, userCfg); err != nil {
			return fmt.Errorf("write configs.user.ini: %w", err)
		}

		_ = os.WriteFile(filepath.Join(gseDir, "account_name.txt"), []byte(p.Name), 0644)
		_ = os.WriteFile(filepath.Join(gseDir, "user_steam_id.txt"), []byte(p.SteamID), 0644)
		_ = os.WriteFile(filepath.Join(gseDir, "language.txt"), []byte(p.Language), 0644)
		_ = os.WriteFile(filepath.Join(gseDir, "custom_broadcasts.txt"), []byte("127.0.0.1\n10.0.2.2\n"), 0644)
	} else {
		if err := MaterializeIsolatedHome(homeDir, userCfg); err != nil {
			return err
		}
	}

	// Single source of truth: ensure Wine prefix AppData/Roaming/GSE Saves symlinks to this profile's GSE directory
	prefixBase := m.GetPrefixDir(id)
	gseBaseDir := filepath.Join(homeDir, ".local", "share", "GSE Saves")
	_ = compat.EnsurePrefixGSESymlink(prefixBase, gseBaseDir)
	_ = compat.EnsurePrefixGSESymlink(filepath.Join(prefixBase, "umu-default"), gseBaseDir)

	// Also symlink any existing game prefixes in this profile's prefix base
	if entries, err := os.ReadDir(prefixBase); err == nil {
		for _, e := range entries {
			if e.IsDir() && e.Name() != "umu-default" {
				_ = compat.EnsurePrefixGSESymlink(filepath.Join(prefixBase, e.Name()), gseBaseDir)
			}
		}
	}

	return nil
}
