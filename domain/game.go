package domain

import (
	"strings"

	"github.com/staernid/gamux/util"
)

// Store constants
const (
	StoreSteam      = "Steam"
	StoreGOG        = "GOG"
	StoreEpic       = "Epic"
	StoreItch       = "Itch"
	StoreStandalone = "Standalone"
	StoreCustom     = "Custom"
)

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

// GameSource represents how and where the game is installed and run.
type GameSource = string

const (
	SourceSteam      GameSource = StoreSteam
	SourceGOG        GameSource = StoreGOG
	SourceEpic       GameSource = StoreEpic
	SourceItch       GameSource = StoreItch
	SourceStandalone GameSource = StoreStandalone
	SourceCustom     GameSource = StoreCustom
)

// Canonical asset types
const (
	AssetCover  = "cover"
	AssetBanner = "banner"
	AssetHero   = "hero"
	AssetLogo   = "logo"
	AssetIcon   = "icon"
)

// LaunchCandidate represents an executable or configured launch target in a game directory.
type LaunchCandidate struct {
	Name        string `json:"name"`
	Executable  string `json:"executable"`
	Arguments   string `json:"arguments,omitempty"`
	Description string `json:"description,omitempty"`
}

// NewsItem represents a store patch note or announcement.
type NewsItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Author    string `json:"author,omitempty"`
	Contents  string `json:"contents"`
	FeedLabel string `json:"feedlabel,omitempty"`
	Date      int64  `json:"date"`
	FeedName  string `json:"feedname,omitempty"`
	AppID     uint32 `json:"appid,omitempty"`
}

// AchievementItem represents a game achievement with metadata, unlock status, and rarity.
type AchievementItem struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	IconURL       string  `json:"icon_url"`
	IconGrayURL   string  `json:"icon_gray_url"`
	Hidden        bool    `json:"hidden"`
	Unlocked      bool    `json:"unlocked"`
	UnlockTime    int64   `json:"unlock_time,omitempty"`
	GlobalPercent float64 `json:"global_percent,omitempty"`
	Rarity        string  `json:"rarity,omitempty"` // "ultra_rare", "rare", "uncommon", "common"
}

// ComputeRarityTier classifies an achievement unlock percentage into standard rarity tiers.
func ComputeRarityTier(percent float64) string {
	if percent <= 0 {
		return ""
	}
	if percent < 5.0 {
		return "ultra_rare"
	}
	if percent < 15.0 {
		return "rare"
	}
	if percent < 40.0 {
		return "uncommon"
	}
	return "common"
}

// SteamProfile represents resolved public Steam profile data for persona configuration.
type SteamProfile struct {
	SteamID     string `json:"steam_id"`
	PersonaName string `json:"persona_name"`
	AvatarURL   string `json:"avatar_url"`
	CountryCode string `json:"country_code,omitempty"`
	ProfileURL  string `json:"profile_url,omitempty"`
}

// VersionStatus encapsulates update verification metrics and patch freshness.
type VersionStatus struct {
	IsUpToDate       bool   `json:"is_up_to_date"`
	Checked          bool   `json:"checked"`
	CurrentBuildID   string `json:"current_build_id,omitempty"`
	LatestBuildID    string `json:"latest_build_id,omitempty"`
	LatestPatchDate  int64  `json:"latest_patch_date,omitempty"`
	LatestPatchTitle string `json:"latest_patch_title,omitempty"`
	Message          string `json:"message,omitempty"`
}

// DLCItem represents a structured DLC entry associated with a game.
type DLCItem struct {
	AppID string `json:"app_id"`
	Name  string `json:"name"`
}

// SteamBranch represents an official release or beta content branch for a Steam game.
type SteamBranch struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	BuildID     string `json:"build_id,omitempty"`
	TimeUpdated int64  `json:"time_updated,omitempty"`
	PwdRequired bool   `json:"pwd_required,omitempty"`
}


// ArtworkOption represents a selectable artwork asset candidate discovered from a metadata provider.
type ArtworkOption struct {
	ID       string `json:"id"`
	Provider string `json:"provider"` // "steam", "steamgriddb", "gog", "custom"
	Type     string `json:"type"`     // "cover", "banner", "hero", "logo", "icon"
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Style    string `json:"style,omitempty"` // "official", "alternate", "blurred", "material", "no_logo"
	Score    int    `json:"score,omitempty"`
	Author   string `json:"author,omitempty"`
}

// ArtworkQuery encapsulates structured criteria for resolving artwork candidates.
type ArtworkQuery struct {
	GameDir    string `json:"game_dir,omitempty"`
	Title      string `json:"title,omitempty"`
	SteamAppID uint32 `json:"steam_app_id,omitempty"`
	GOGID      string `json:"gog_id,omitempty"`
	SGDBID     string `json:"sgdb_id,omitempty"`
	AssetType  string `json:"asset_type"` // "cover", "banner", "hero", "logo", "icon"
}

// ExternalIDs tracks cross-platform and database identifiers for a game.
type ExternalIDs struct {
	SteamAppID uint32 `json:"steam_appid,omitempty"`
	GOGID      string `json:"gog_id,omitempty"`
	EpicApp    string `json:"epic_app,omitempty"`
	SGDBID     string `json:"sgdb_id,omitempty"`
	RAWGID     string `json:"rawg_id,omitempty"`
	IGDBID     string `json:"igdb_id,omitempty"`
}

// DepotSummary encapsulates metadata and disk installation state for a single Steam content depot.
type DepotSummary struct {
	DepotID         uint32 `json:"depot_id"`
	Name            string `json:"name"`
	Category        string `json:"category"` // "common", "windows", "linux", "macos", "soundtrack"
	ContentCategory string `json:"content_category,omitempty"` // "core", "locale:<language>", "soundtrack", "optional_hd", "dlc"
	Language        string `json:"language,omitempty"`          // e.g. "french", "german", "english"
	Installed       bool   `json:"installed"`
	Size            uint64 `json:"size"`
	DownloadSize    uint64 `json:"download_size,omitempty"`
	FileCount       int    `json:"file_count"`
	ManifestID      uint64 `json:"manifest_id"`
	OSArch          string `json:"osarch,omitempty"`
	DLCAppID        uint32 `json:"dlc_app_id,omitempty"`
	IsDLC           bool   `json:"is_dlc,omitempty"`
	DLCName         string `json:"dlc_name,omitempty"`
}

// ManifestStatus holds depot manifest and file integrity state.
type ManifestStatus struct {
	ManifestID        string   `json:"manifest_id,omitempty"`
	BuildID           string   `json:"build_id,omitempty"`
	RemoteManifestID  string   `json:"remote_manifest_id,omitempty"`
	OfficialFileCount int      `json:"official_file_count"`
	ModifiedFiles     []string `json:"modified_files,omitempty"`
	MissingFiles      []string `json:"missing_files,omitempty"`
	UntrackedFiles    []string `json:"untracked_files,omitempty"`
	RedistModified    []string `json:"redist_modified,omitempty"`
	RedistMissing     []string `json:"redist_missing,omitempty"`
	HasUpdate         bool     `json:"has_update"`
}

// StoreMetadata holds store-sourced remote metadata (news, patch notes, synopsis, credits).
type StoreMetadata struct {
	RecentPatchNote string     `json:"recent_patch_note,omitempty"`
	NewsItems       []NewsItem `json:"news_items,omitempty"`
	Summary         string     `json:"summary,omitempty"`
	ReleaseDate     string     `json:"release_date,omitempty"`
	Developers      []string   `json:"developers,omitempty"`
	Publishers      []string   `json:"publishers,omitempty"`
	Genres          []string   `json:"genres,omitempty"`
}

// AssetLocation encapsulates paths and URLs for a specific graphical asset.
type AssetLocation struct {
	RemoteURL string `json:"remote_url,omitempty"`
	LocalPath string `json:"local_path,omitempty"`
	LocalURL  string `json:"local_url,omitempty"`
	CustomURL string `json:"custom_url,omitempty"`
}

// GameArtwork stores remote URLs and local file paths for graphical assets.
type GameArtwork struct {
	CoverURL        string `json:"cover_url,omitempty"`
	BannerURL       string `json:"banner_url,omitempty"`
	HeroURL         string `json:"hero_url,omitempty"`
	LogoURL         string `json:"logo_url,omitempty"`
	IconURL         string `json:"icon_url,omitempty"`
	LocalCover      string `json:"local_cover,omitempty"`
	LocalBanner     string `json:"local_banner,omitempty"`
	LocalHero       string `json:"local_hero,omitempty"`
	LocalLogo       string `json:"local_logo,omitempty"`
	LocalIcon       string `json:"local_icon,omitempty"`
	LocalCoverURL   string `json:"local_cover_url,omitempty"`
	LocalBannerURL  string `json:"local_banner_url,omitempty"`
	LocalHeroURL    string `json:"local_hero_url,omitempty"`
	LocalLogoURL    string `json:"local_logo_url,omitempty"`
	CustomCoverURL  string `json:"custom_cover_url,omitempty"`
	CustomBannerURL string `json:"custom_banner_url,omitempty"`
	CustomHeroURL   string `json:"custom_hero_url,omitempty"`
	CustomLogoURL   string `json:"custom_logo_url,omitempty"`
	CustomIconURL   string `json:"custom_icon_url,omitempty"`
	LockedArtwork   bool   `json:"locked_artwork,omitempty"`
}

// ---------------------------------------------------------------------------
// 1. Dedicated Model: GameRecord (Static Identity & On-Disk Inventory)
// ---------------------------------------------------------------------------

// GameRecord captures the immutable on-disk identity, directory layout, and inventory metrics of an installed game.
// All fields reflect physical on-disk baseline data discovered via detector, ACF manifests, and depot scanners.
// To override execution or environment behaviors, use LaunchProfile instead.
type GameRecord struct {
	GameDir            string            `json:"game_dir"`
	AppID              string            `json:"app_id"`
	NumericAppID       uint32            `json:"numeric_app_id"`
	Name               string            `json:"name"`
	InstallDir         string            `json:"install_dir,omitempty"`
	Store              string            `json:"store"`  // "Steam", "GOG", "Epic", "Itch", "Standalone", "Custom"
	Source             GameSource        `json:"source"` // Canonical distribution source
	Platform           string            `json:"platform"` // "linux", "win64", "win32"
	Branch             string            `json:"branch,omitempty"`
	ExternalIDs        `json:",inline"`
	DiskSizeBytes      int64             `json:"disk_size_bytes"`
	FileCount          int               `json:"file_count"`
	DLCCount           int               `json:"dlc_count,omitempty"`
	AchievementCount   int               `json:"achievement_count,omitempty"`
	InstalledAt        int64             `json:"installed_at,omitempty"`
	PlaytimeSeconds    int64             `json:"playtime_seconds,omitempty"`
	LastPlayed         int64             `json:"last_played,omitempty"`
	LaunchCandidates   []LaunchCandidate `json:"launch_candidates,omitempty"`
	Depots             []DepotSummary    `json:"depots,omitempty"`
	AvailablePlatforms []string          `json:"available_platforms,omitempty"`
	SupportedLanguages []string          `json:"supported_languages,omitempty"`
	TargetLibPath      string            `json:"target_lib_path,omitempty"`
	ManifestPath       string            `json:"manifest_path,omitempty"`
	RawDepotLayout     bool              `json:"raw_depot_layout,omitempty"`
	ExePath            string            `json:"exe_path,omitempty"` // Detected default primary executable
	ExeArgs            string            `json:"exe_args,omitempty"`
}

// ---------------------------------------------------------------------------
// 2. Dedicated Model: LaunchProfile (Persistent User Configuration)
// ---------------------------------------------------------------------------

// LaunchProfile defines execution overrides and runtime environment settings for a game,
// persisted atomically to <gameDir>/steam_settings/launch_profile.json.
// Fields in LaunchProfile take precedence over detected defaults in GameRecord.
type LaunchProfile struct {
	GameDir           string            `json:"game_dir"`
	AppID             string            `json:"app_id,omitempty"`
	Name              string            `json:"name,omitempty"`
	Platform          string            `json:"platform,omitempty"`       // "linux", "win64", "win32"
	ExePath           string            `json:"exe_path"`
	ExeArgs           string            `json:"exe_args,omitempty"`
	OperatingMode     string            `json:"operating_mode,omitempty"` // "standalone" | "portable" | "loader" | "native" | "none"
	Runner            string            `json:"runner,omitempty"`         // "umu", "wine", "proton", "linux", "auto"
	WinePrefix        string            `json:"wine_prefix,omitempty"`
	ProtonPath        string            `json:"proton_path,omitempty"`
	Language          string            `json:"language,omitempty"`
	MangoHud          *bool             `json:"mangohud,omitempty"`
	Gamescope         *bool             `json:"gamescope,omitempty"`
	GamescopeWidth    int               `json:"gamescope_width,omitempty"`
	GamescopeHeight   int               `json:"gamescope_height,omitempty"`
	GamescopeFPS      int               `json:"gamescope_fps,omitempty"`
	DisableOverlay    *bool             `json:"disable_overlay,omitempty"`
	OfflineMode       *bool             `json:"offline_mode,omitempty"`
	DllOverrides      string            `json:"dll_overrides,omitempty"`  // Custom Wine DLL overrides string, e.g. "d3d11=n,b;dxgi=n,b"
	InhibitSleep      *bool             `json:"inhibit_sleep,omitempty"`
	PreferSDL         *bool             `json:"prefer_sdl,omitempty"`
	EnableWayland     *bool             `json:"enable_wayland,omitempty"`
	CustomEnv         map[string]string `json:"custom_env,omitempty"`
	WorkingDir        string            `json:"working_dir,omitempty"`
	ProfileID         string            `json:"profile_id,omitempty"`

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

// Clone returns a deep copy of the Game aggregate.
func (g *Game) Clone() *Game {
	if g == nil {
		return nil
	}
	cp := *g
	if g.LaunchProfile != nil {
		cp.LaunchProfile = g.LaunchProfile.Clone()
	}
	if g.VersionStatus != nil {
		vs := *g.VersionStatus
		cp.VersionStatus = &vs
	}
	if g.LaunchCandidates != nil {
		cp.LaunchCandidates = append([]LaunchCandidate(nil), g.LaunchCandidates...)
	}
	if g.Depots != nil {
		cp.Depots = append([]DepotSummary(nil), g.Depots...)
	}
	if g.AvailablePlatforms != nil {
		cp.AvailablePlatforms = append([]string(nil), g.AvailablePlatforms...)
	}
	if g.SupportedLanguages != nil {
		cp.SupportedLanguages = append([]string(nil), g.SupportedLanguages...)
	}
	if g.OriginalBackups != nil {
		cp.OriginalBackups = append([]string(nil), g.OriginalBackups...)
	}
	if g.SettingsDirs != nil {
		cp.SettingsDirs = append([]string(nil), g.SettingsDirs...)
	}
	return &cp
}

// ---------------------------------------------------------------------------
// 3. Dedicated Model: ReadinessStatus (Computed Health & Integration State)
// ---------------------------------------------------------------------------

// ReadinessStatus holds computed operational flags derived from inspection data and configurations.
type ReadinessStatus struct {
	State              string   `json:"state"` // "Original", "Loader-Configured", "Portable-Patched"
	IsInstalled        bool     `json:"is_installed"`
	IsStub             bool     `json:"is_stub"`
	IsPatched          bool     `json:"is_patched"`
	IsPortable         bool     `json:"is_portable"`
	HasUpdatesOrIssues bool     `json:"has_updates_or_issues"`
	IsReady            bool     `json:"is_ready"`
	OriginalBackups    []string `json:"original_backups,omitempty"`
	SettingsDirExists  bool     `json:"settings_dir_exists"`
	SettingsDirs       []string `json:"settings_dirs,omitempty"`
	SteamAppIDTxtFound bool     `json:"steam_appid_txt_found"`
	CurrentLanguage    string   `json:"current_language,omitempty"`
}

// ---------------------------------------------------------------------------
// Unified Domain Aggregate: Game
// ---------------------------------------------------------------------------

// Game is the canonical entity representing a managed game installation.
// It composes the static on-disk record, user launch configuration, computed readiness,
// manifest integrity status, store metadata, and graphical assets.
type Game struct {
	GameRecord                 `json:",inline"`
	LaunchProfile *LaunchProfile `json:"launch_profile,omitempty"`
	ReadinessStatus            `json:",inline"`
	GameArtwork                `json:",inline"`
	ManifestStatus             `json:",inline"`
	StoreMetadata              `json:",inline"`
	VersionStatus *VersionStatus `json:"version_status,omitempty"`
}

// NewGame initializes a Game entity with parsed numeric AppID and standard defaults.
func NewGame(name, gameDir, appID, platform string) *Game {
	numAppID, _ := util.ParseAppID(appID)
	store := StoreSteam
	if numAppID == 0 && appID != "" && appID != "0" {
		store = StoreStandalone
	}
	return &Game{
		GameRecord: GameRecord{
			Name:         name,
			GameDir:      gameDir,
			AppID:        appID,
			NumericAppID: numAppID,
			Platform:     platform,
			Store:        store,
			Source:       store,
			ExternalIDs: ExternalIDs{
				SteamAppID: numAppID,
			},
			SupportedLanguages: []string{"english"},
			LaunchCandidates:   make([]LaunchCandidate, 0),
		},
		ReadinessStatus: ReadinessStatus{
			State:           "Original",
			OriginalBackups: make([]string, 0),
		},
		ManifestStatus: ManifestStatus{
			ModifiedFiles:  make([]string, 0),
			MissingFiles:   make([]string, 0),
			UntrackedFiles: make([]string, 0),
			RedistModified: make([]string, 0),
			RedistMissing:  make([]string, 0),
		},
		StoreMetadata: StoreMetadata{
			NewsItems:  make([]NewsItem, 0),
			Developers: make([]string, 0),
			Publishers: make([]string, 0),
			Genres:     make([]string, 0),
		},
	}
}

// ApplyProfile synchronizes custom user profile overrides, metadata bindings, external IDs,
// artwork, and playtime metrics from the launch profile onto the Game entity.
func (g *Game) ApplyProfile(profile *LaunchProfile, launchProfileExists bool) {
	if g == nil || profile == nil {
		return
	}
	g.LaunchProfile = profile
	if profile.SourceOverride != "" {
		g.Source = profile.SourceOverride
	} else if profile.Store != "" && (g.Source == "" || g.Source == StoreStandalone) {
		switch strings.ToLower(profile.Store) {
		case "steam":
			g.Source = SourceSteam
			g.Store = StoreSteam
		case "gog":
			g.Source = SourceGOG
			g.Store = StoreGOG
		case "epic":
			g.Source = SourceEpic
			g.Store = StoreEpic
		case "itch":
			g.Source = SourceItch
			g.Store = StoreItch
		default:
			g.Store = profile.Store
		}
	}
	if launchProfileExists && profile.Store != "" && !strings.EqualFold(profile.Store, string(g.Source)) {
		g.Store = profile.Store
	}
	if profile.MetadataID != "" {
		if strings.EqualFold(profile.MetadataProvider, "steam") {
			if num, ok := util.ParseAppID(profile.MetadataID); ok && num > 0 {
				g.ExternalIDs.SteamAppID = num
			}
		} else if strings.EqualFold(profile.MetadataProvider, "gog") {
			g.ExternalIDs.GOGID = profile.MetadataID
		} else if strings.EqualFold(profile.MetadataProvider, "rawg") {
			g.ExternalIDs.RAWGID = profile.MetadataID
		}
	}
	if launchProfileExists {
		appID := strings.TrimSpace(profile.StoreAppID)
		if appID == "" {
			appID = strings.TrimSpace(profile.AppID)
		}
		if appID != "" {
			g.AppID = appID
			g.NumericAppID, _ = util.ParseAppID(appID)
			if g.NumericAppID > 0 {
				g.ExternalIDs.SteamAppID = g.NumericAppID
			}
		}
	}
	if profile.GOGID != "" {
		g.ExternalIDs.GOGID = profile.GOGID
	}
	if profile.SGDBID != "" {
		g.ExternalIDs.SGDBID = profile.SGDBID
	}
	if profile.CustomCoverURL != "" {
		g.CustomCoverURL = profile.CustomCoverURL
	}
	if profile.CustomBannerURL != "" {
		g.CustomBannerURL = profile.CustomBannerURL
	}
	if profile.CustomHeroURL != "" {
		g.CustomHeroURL = profile.CustomHeroURL
	}
	if profile.CustomLogoURL != "" {
		g.CustomLogoURL = profile.CustomLogoURL
	}
	if profile.CustomIconURL != "" {
		g.CustomIconURL = profile.CustomIconURL
	}
	if profile.LockedArtwork {
		g.LockedArtwork = true
	}
	if profile.InstalledAt > 0 {
		g.InstalledAt = profile.InstalledAt
	} else if g.InstalledAt > 0 {
		profile.InstalledAt = g.InstalledAt
	}
	g.PlaytimeSeconds = profile.PlaytimeSeconds
	g.LastPlayed = profile.LastPlayed
	if profile.Branch != "" {
		g.Branch = profile.Branch
	}
}

// EffectiveExePath returns the resolved executable path (profile override taking priority over detected binary).
func (g *Game) EffectiveExePath() string {
	if g.LaunchProfile != nil && g.LaunchProfile.ExePath != "" {
		return g.LaunchProfile.ExePath
	}
	if g.GameRecord.ExePath != "" {
		return g.GameRecord.ExePath
	}
	if len(g.LaunchCandidates) > 0 {
		return g.LaunchCandidates[0].Executable
	}
	return ""
}

// EffectiveExeArgs returns the launch arguments (profile override taking priority).
func (g *Game) EffectiveExeArgs() string {
	if g.LaunchProfile != nil && g.LaunchProfile.ExeArgs != "" {
		return g.LaunchProfile.ExeArgs
	}
	if g.GameRecord.ExeArgs != "" {
		return g.GameRecord.ExeArgs
	}
	if len(g.LaunchCandidates) > 0 {
		return g.LaunchCandidates[0].Arguments
	}
	return ""
}

// CandidateExes returns all unique executable paths across LaunchProfile, GameRecord, and LaunchCandidates.
func (g *Game) CandidateExes() []string {
	seen := make(map[string]bool)
	var exes []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p != "" && !seen[p] {
			seen[p] = true
			exes = append(exes, p)
		}
	}
	if g.LaunchProfile != nil {
		add(g.LaunchProfile.ExePath)
	}
	add(g.GameRecord.ExePath)
	for _, c := range g.LaunchCandidates {
		add(c.Executable)
	}
	return exes
}

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

// EffectiveOperatingMode returns the active operating mode override from the profile or empty string if unset.
func (g *Game) EffectiveOperatingMode() string {
	if g.LaunchProfile != nil && g.LaunchProfile.OperatingMode != "" {
		return NormalizeOperatingMode(g.LaunchProfile.OperatingMode)
	}
	return ""
}

// EffectiveRunner returns the active runner from the profile or defaults to RunnerAuto.
func (g *Game) EffectiveRunner() string {
	if g.LaunchProfile != nil && g.LaunchProfile.Runner != "" {
		return g.LaunchProfile.Runner
	}
	return RunnerAuto
}

// EffectiveWinePrefix returns the active Wine prefix configured in the launch profile.
func (g *Game) EffectiveWinePrefix() string {
	if g.LaunchProfile != nil && g.LaunchProfile.WinePrefix != "" {
		return g.LaunchProfile.WinePrefix
	}
	return ""
}

// EffectivePlaytime returns the total playtime in seconds and last played timestamp.
func (g *Game) EffectivePlaytime() (int64, int64) {
	if g.LaunchProfile != nil && (g.LaunchProfile.PlaytimeSeconds > 0 || g.LaunchProfile.LastPlayed > 0) {
		return g.LaunchProfile.PlaytimeSeconds, g.LaunchProfile.LastPlayed
	}
	return g.GameRecord.PlaytimeSeconds, g.GameRecord.LastPlayed
}

// EffectiveInstalledAt returns the resolved installation timestamp (profile override taking priority over recorded on-disk timestamp).
func (g *Game) EffectiveInstalledAt() int64 {
	if g.LaunchProfile != nil && g.LaunchProfile.InstalledAt > 0 {
		return g.LaunchProfile.InstalledAt
	}
	return g.GameRecord.InstalledAt
}

// SteamAppID returns the numeric AppID if present.
func (g *Game) SteamAppID() (uint32, bool) {
	if g.ExternalIDs.SteamAppID > 0 {
		return g.ExternalIDs.SteamAppID, true
	}
	if g.NumericAppID > 0 {
		return g.NumericAppID, true
	}
	if g.LaunchProfile != nil && g.LaunchProfile.AppID != "" {
		if parsed, ok := util.ParseAppID(g.LaunchProfile.AppID); ok && parsed > 0 {
			return parsed, true
		}
	}
	if g.AppID != "" {
		return util.ParseAppID(g.AppID)
	}
	return 0, false
}

// IsSteam returns true if game originated from or integrates with Steam.
func (g *Game) IsSteam() bool {
	return strings.EqualFold(g.Store, StoreSteam) ||
		g.NumericAppID > 0 ||
		g.ExternalIDs.SteamAppID > 0
}

// IsWindows returns true if the target executable platform is Windows (win64 or win32).
func (g *Game) IsWindows() bool {
	plat := strings.ToLower(strings.TrimSpace(g.Platform))
	return plat == "win64" || plat == "win32" || plat == "windows"
}

// IsLinux returns true if the target executable platform is Linux native.
func (g *Game) IsLinux() bool {
	plat := strings.ToLower(strings.TrimSpace(g.Platform))
	return plat == "linux" || plat == "linux32"
}

// HasBackups returns true if any .ORIGINAL backup libraries are found.
func (g *Game) HasBackups() bool {
	return len(g.OriginalBackups) > 0
}

// IsClean returns true if the game directory has not been patched or modified.
func (g *Game) IsClean() bool {
	return !g.IsPatched && !g.IsPortable && !g.HasBackups() && !g.SettingsDirExists
}

// IsLoaderConfigured returns true if Goldberg loader configuration files or binaries are active.
func (g *Game) IsLoaderConfigured() bool {
	mode := NormalizeOperatingMode(g.EffectiveOperatingMode())
	return mode == ModeLoader ||
		g.State == "Loader-Configured" ||
		strings.Contains(strings.ToLower(g.EffectiveExePath()), "steamclient_loader")
}

// IsPortablePatched returns true if Goldberg emulator libraries are directly injected into the game tree.
func (g *Game) IsPortablePatched() bool {
	mode := NormalizeOperatingMode(g.EffectiveOperatingMode())
	return mode == ModePortable || g.IsPortable || g.State == "Portable-Patched" || g.HasBackups()
}

// RequiresSteamEmulator returns true if the game requires Steamworks mocking.
func (g *Game) RequiresSteamEmulator() bool {
	mode := NormalizeOperatingMode(g.EffectiveOperatingMode())
	if mode == ModeNone || mode == ModeStandalone {
		return false
	}
	if !g.IsSteam() && g.AppID == "" {
		return false
	}
	return true
}

// RecomputeState recalculates computed flags (IsPatched, IsPortable, IsReady, HasUpdatesOrIssues, IsInstalled, IsStub).
func (g *Game) RecomputeState() {
	mode := NormalizeOperatingMode(g.EffectiveOperatingMode())
	if mode == ModePortable {
		g.State = "Portable-Patched"
	} else if mode == ModeLoader {
		g.State = "Loader-Configured"
	} else if mode == ModeNone {
		g.State = "Original"
	} else if len(g.OriginalBackups) > 0 {
		g.State = "Portable-Patched"
	} else if g.SettingsDirExists {
		g.State = "Loader-Configured"
	} else if g.State == "" {
		g.State = "Original"
	}

	g.IsPortable = g.State == "Portable-Patched"
	g.IsPatched = g.IsPortable || (g.State == "Loader-Configured" && mode != ModeNone)
	g.HasUpdatesOrIssues = g.IsSteam() && (g.HasUpdate || len(g.ModifiedFiles) > 0 || len(g.MissingFiles) > 0)
	g.IsInstalled = g.EffectiveExePath() != "" || g.FileCount > 0
	g.IsStub = g.GameDir != "" && !g.IsInstalled
	g.IsReady = g.IsInstalled && (g.IsPatched || g.EffectiveExePath() != "")
}

// GetAsset returns the location information for a specific asset type.
func (g *Game) GetAsset(assetType string) AssetLocation {
	switch strings.ToLower(assetType) {
	case AssetCover:
		return AssetLocation{
			RemoteURL: g.CoverURL,
			LocalPath: g.LocalCover,
			LocalURL:  g.LocalCoverURL,
			CustomURL: g.CustomCoverURL,
		}
	case AssetBanner:
		return AssetLocation{
			RemoteURL: g.BannerURL,
			LocalPath: g.LocalBanner,
			LocalURL:  g.LocalBannerURL,
			CustomURL: g.CustomBannerURL,
		}
	case AssetHero:
		return AssetLocation{
			RemoteURL: g.HeroURL,
			LocalPath: g.LocalHero,
			LocalURL:  g.LocalHeroURL,
			CustomURL: g.CustomHeroURL,
		}
	case AssetLogo:
		return AssetLocation{
			RemoteURL: g.LogoURL,
			LocalPath: g.LocalLogo,
			LocalURL:  g.LocalLogoURL,
			CustomURL: g.CustomLogoURL,
		}
	case AssetIcon:
		return AssetLocation{
			RemoteURL: g.IconURL,
			LocalPath: g.LocalIcon,
			CustomURL: g.CustomIconURL,
		}
	default:
		return AssetLocation{}
	}
}

// SetAsset updates the fields corresponding to the specified asset type.
func (g *Game) SetAsset(assetType string, loc AssetLocation) {
	switch strings.ToLower(assetType) {
	case AssetCover:
		if loc.RemoteURL != "" {
			g.CoverURL = loc.RemoteURL
		}
		if loc.LocalPath != "" {
			g.LocalCover = loc.LocalPath
		}
		if loc.LocalURL != "" {
			g.LocalCoverURL = loc.LocalURL
		}
		if loc.CustomURL != "" {
			g.CustomCoverURL = loc.CustomURL
		}
	case AssetBanner:
		if loc.RemoteURL != "" {
			g.BannerURL = loc.RemoteURL
		}
		if loc.LocalPath != "" {
			g.LocalBanner = loc.LocalPath
		}
		if loc.LocalURL != "" {
			g.LocalBannerURL = loc.LocalURL
		}
		if loc.CustomURL != "" {
			g.CustomBannerURL = loc.CustomURL
		}
	case AssetHero:
		if loc.RemoteURL != "" {
			g.HeroURL = loc.RemoteURL
		}
		if loc.LocalPath != "" {
			g.LocalHero = loc.LocalPath
		}
		if loc.LocalURL != "" {
			g.LocalHeroURL = loc.LocalURL
		}
		if loc.CustomURL != "" {
			g.CustomHeroURL = loc.CustomURL
		}
	case AssetLogo:
		if loc.RemoteURL != "" {
			g.LogoURL = loc.RemoteURL
		}
		if loc.LocalPath != "" {
			g.LocalLogo = loc.LocalPath
		}
		if loc.LocalURL != "" {
			g.LocalLogoURL = loc.LocalURL
		}
		if loc.CustomURL != "" {
			g.CustomLogoURL = loc.CustomURL
		}
	case AssetIcon:
		if loc.RemoteURL != "" {
			g.IconURL = loc.RemoteURL
		}
		if loc.LocalPath != "" {
			g.LocalIcon = loc.LocalPath
		}
		if loc.CustomURL != "" {
			g.CustomIconURL = loc.CustomURL
		}
	}
}

// EffectiveMetadataProvider returns the active metadata provider in lowercase (e.g. "steam", "gog", "custom", "standalone").
func (g *Game) EffectiveMetadataProvider() string {
	if g.LaunchProfile != nil && g.LaunchProfile.MetadataProvider != "" {
		return strings.ToLower(g.LaunchProfile.MetadataProvider)
	}
	if g.Store != "" && !strings.EqualFold(g.Store, StoreStandalone) && !strings.EqualFold(g.Store, StoreCustom) {
		return strings.ToLower(g.Store)
	}
	if g.NumericAppID > 0 || g.ExternalIDs.SteamAppID > 0 {
		return "steam"
	}
	if g.GOGID != "" {
		return "gog"
	}
	return "standalone"
}

// EffectiveMetadataID returns the identifier associated with the active metadata provider.
func (g *Game) EffectiveMetadataID() string {
	if g.LaunchProfile != nil && g.LaunchProfile.MetadataID != "" {
		return g.LaunchProfile.MetadataID
	}
	if g.AppID != "" && g.AppID != "0" {
		return g.AppID
	}
	if g.GOGID != "" {
		return g.GOGID
	}
	if g.RAWGID != "" {
		return g.RAWGID
	}
	return ""
}

// HasCustomMetadata returns true if the game has a custom metadata provider override configured.
func (g *Game) HasCustomMetadata() bool {
	if g.LaunchProfile != nil {
		return g.LaunchProfile.MetadataProvider != "" || g.LaunchProfile.MetadataID != ""
	}
	return false
}


