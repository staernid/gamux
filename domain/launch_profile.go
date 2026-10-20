package domain

import "strings"

// Operating Mode Constants
const (
	ModePortable   = "portable"
	ModeLoader     = "loader"
	ModeNone       = "none"
	ModeNative     = "native"
	ModeStandalone = "standalone"
)

// Runner constants
const (
	RunnerAuto   = "auto"
	RunnerUMU    = "umu"
	RunnerWine   = "wine"
	RunnerProton = "proton"
	RunnerLinux  = "linux"
)

// NormalizeOperatingMode normalizes alias strings (e.g. "injected", "standalone", "drm-free") to canonical operating modes.
func NormalizeOperatingMode(mode string) string {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "portable", "injected", "patch", "patched", "portable-patched":
		return ModePortable
	case "loader", "loader-configured":
		return ModeLoader
	case "none", "standalone", "clean", "drm-free":
		return ModeNone
	case "native":
		return ModeNative
	default:
		return m
	}
}

// LaunchProfile defines execution overrides and runtime environment settings for a game,
// persisted atomically to <gameDir>/steam_settings/launch_profile.json.
// Fields in LaunchProfile take precedence over detected defaults in GameRecord.
type LaunchProfile struct {
	GameDir         string            `json:"game_dir"`
	AppID           string            `json:"app_id,omitempty"`
	Name            string            `json:"name,omitempty"`
	Platform        string            `json:"platform,omitempty"` // "linux", "win64", "win32"
	ExePath         string            `json:"exe_path"`
	ExeArgs         string            `json:"exe_args,omitempty"`
	OperatingMode   string            `json:"operating_mode,omitempty"` // "standalone" | "portable" | "loader" | "native" | "none"
	Runner          string            `json:"runner,omitempty"`         // "umu", "wine", "proton", "linux", "auto"
	WinePrefix      string            `json:"wine_prefix,omitempty"`
	ProtonPath      string            `json:"proton_path,omitempty"`
	Language        string            `json:"language,omitempty"`
	MangoHud        *bool             `json:"mangohud,omitempty"`
	Gamescope       *bool             `json:"gamescope,omitempty"`
	GamescopeWidth  int               `json:"gamescope_width,omitempty"`
	GamescopeHeight int               `json:"gamescope_height,omitempty"`
	GamescopeFPS    int               `json:"gamescope_fps,omitempty"`
	DisableOverlay  *bool             `json:"disable_overlay,omitempty"`
	OfflineMode     *bool             `json:"offline_mode,omitempty"`
	DllOverrides    string            `json:"dll_overrides,omitempty"` // Custom Wine DLL overrides string, e.g. "d3d11=n,b;dxgi=n,b"
	InhibitSleep    *bool             `json:"inhibit_sleep,omitempty"`
	PreferSDL       *bool             `json:"prefer_sdl,omitempty"`
	EnableWayland   *bool             `json:"enable_wayland,omitempty"`
	CustomEnv       map[string]string `json:"custom_env,omitempty"`
	WorkingDir      string            `json:"working_dir,omitempty"`
	ProfileID       string            `json:"profile_id,omitempty"`
	ForceEmulation  *bool             `json:"force_emulation,omitempty"`

	// Store & Metadata bindings
	Store            string `json:"store,omitempty"`
	StoreAppID       string `json:"store_app_id,omitempty"`
	Branch           string `json:"branch,omitempty"`
	BranchPassword   string `json:"branch_password,omitempty"`
	SourceOverride   string `json:"source_override,omitempty"`
	MetadataProvider string `json:"metadata_provider,omitempty"`
	MetadataID       string `json:"metadata_id,omitempty"`
	GOGID            string `json:"gog_id,omitempty"`
	SGDBID           string `json:"sgdb_id,omitempty"`
	CustomCoverURL   string `json:"custom_cover_url,omitempty"`
	CustomBannerURL  string `json:"custom_banner_url,omitempty"`
	CustomHeroURL    string `json:"custom_hero_url,omitempty"`
	CustomLogoURL    string `json:"custom_logo_url,omitempty"`
	CustomIconURL    string `json:"custom_icon_url,omitempty"`
	LockedArtwork    bool   `json:"locked_artwork,omitempty"`
	InstalledAt      int64  `json:"installed_at,omitempty"`
	PlaytimeSeconds  int64  `json:"playtime_seconds,omitempty"`
	LastPlayed       int64  `json:"last_played,omitempty"`
}

// MergeFrom copies non-zero fields from other into p if they are not already set in p.
// Environment variables in CustomEnv are merged without overwriting existing keys.
func (p *LaunchProfile) MergeFrom(other *LaunchProfile) {
	if p == nil || other == nil {
		return
	}
	if p.GameDir == "" {
		p.GameDir = other.GameDir
	}
	if p.Name == "" {
		p.Name = other.Name
	}
	if p.AppID == "" {
		p.AppID = other.AppID
	}
	if p.Platform == "" {
		p.Platform = other.Platform
	}
	if p.ExePath == "" {
		p.ExePath = other.ExePath
	}
	if p.ExeArgs == "" {
		p.ExeArgs = other.ExeArgs
	}
	if p.OperatingMode == "" {
		p.OperatingMode = other.OperatingMode
	}
	if p.Runner == "" {
		p.Runner = other.Runner
	}
	if p.WinePrefix == "" {
		p.WinePrefix = other.WinePrefix
	}
	if p.ProtonPath == "" {
		p.ProtonPath = other.ProtonPath
	}
	if p.Language == "" {
		p.Language = other.Language
	}
	if p.MangoHud == nil && other.MangoHud != nil {
		p.MangoHud = other.MangoHud
	}
	if p.Gamescope == nil && other.Gamescope != nil {
		p.Gamescope = other.Gamescope
	}
	if p.GamescopeWidth <= 0 && other.GamescopeWidth > 0 {
		p.GamescopeWidth = other.GamescopeWidth
	}
	if p.GamescopeHeight <= 0 && other.GamescopeHeight > 0 {
		p.GamescopeHeight = other.GamescopeHeight
	}
	if p.GamescopeFPS <= 0 && other.GamescopeFPS > 0 {
		p.GamescopeFPS = other.GamescopeFPS
	}
	if p.DisableOverlay == nil && other.DisableOverlay != nil {
		p.DisableOverlay = other.DisableOverlay
	}
	if p.OfflineMode == nil && other.OfflineMode != nil {
		p.OfflineMode = other.OfflineMode
	}
	if p.DllOverrides == "" {
		p.DllOverrides = other.DllOverrides
	}
	if p.InhibitSleep == nil && other.InhibitSleep != nil {
		p.InhibitSleep = other.InhibitSleep
	}
	if p.PreferSDL == nil && other.PreferSDL != nil {
		p.PreferSDL = other.PreferSDL
	}
	if p.EnableWayland == nil && other.EnableWayland != nil {
		p.EnableWayland = other.EnableWayland
	}
	if p.WorkingDir == "" {
		p.WorkingDir = other.WorkingDir
	}
	if p.ProfileID == "" {
		p.ProfileID = other.ProfileID
	}
	if p.ForceEmulation == nil && other.ForceEmulation != nil {
		p.ForceEmulation = other.ForceEmulation
	}
	if p.SourceOverride == "" {
		p.SourceOverride = other.SourceOverride
	}
	if len(other.CustomEnv) > 0 {
		if p.CustomEnv == nil {
			p.CustomEnv = make(map[string]string)
		}
		for k, v := range other.CustomEnv {
			if _, exists := p.CustomEnv[k]; !exists {
				p.CustomEnv[k] = v
			}
		}
	}
	p.MergeMetadataAndStatsFrom(other)
}

// MergeMetadataAndStatsFrom preserves on-disk statistics and metadata/artwork bindings from other
// into p if they are not already set in p. It does not overwrite execution or runner configuration.
func (p *LaunchProfile) MergeMetadataAndStatsFrom(other *LaunchProfile) {
	if p == nil || other == nil {
		return
	}
	if p.InstalledAt <= 0 && other.InstalledAt > 0 {
		p.InstalledAt = other.InstalledAt
	}
	if p.PlaytimeSeconds <= 0 && other.PlaytimeSeconds > 0 {
		p.PlaytimeSeconds = other.PlaytimeSeconds
	}
	if p.LastPlayed <= 0 && other.LastPlayed > 0 {
		p.LastPlayed = other.LastPlayed
	}
	if p.Store == "" {
		p.Store = other.Store
	}
	if p.StoreAppID == "" {
		p.StoreAppID = other.StoreAppID
	}
	if p.MetadataProvider == "" {
		p.MetadataProvider = other.MetadataProvider
	}
	if p.MetadataID == "" {
		p.MetadataID = other.MetadataID
	}
	if p.GOGID == "" {
		p.GOGID = other.GOGID
	}
	if p.SGDBID == "" {
		p.SGDBID = other.SGDBID
	}
	if p.CustomCoverURL == "" {
		p.CustomCoverURL = other.CustomCoverURL
	}
	if p.CustomBannerURL == "" {
		p.CustomBannerURL = other.CustomBannerURL
	}
	if p.CustomHeroURL == "" {
		p.CustomHeroURL = other.CustomHeroURL
	}
	if p.CustomLogoURL == "" {
		p.CustomLogoURL = other.CustomLogoURL
	}
	if p.CustomIconURL == "" {
		p.CustomIconURL = other.CustomIconURL
	}
	if !p.LockedArtwork && other.LockedArtwork {
		p.LockedArtwork = true
	}
	if p.Branch == "" {
		p.Branch = other.Branch
	}
	if p.BranchPassword == "" {
		p.BranchPassword = other.BranchPassword
	}
}

// Clone returns a deep copy of the LaunchProfile.
func (p *LaunchProfile) Clone() *LaunchProfile {
	if p == nil {
		return nil
	}
	cp := *p
	if p.CustomEnv != nil {
		cp.CustomEnv = make(map[string]string, len(p.CustomEnv))
		for k, v := range p.CustomEnv {
			cp.CustomEnv[k] = v
		}
	}
	return &cp
}
