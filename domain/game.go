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

// LaunchProfile defines execution overrides and runtime environment settings for a game.
type LaunchProfile struct {
	GameDir           string            `json:"game_dir"`
	AppID             string            `json:"app_id,omitempty"`
	Name              string            `json:"name,omitempty"`
	Platform          string            `json:"platform,omitempty"`       // "linux", "win64", "win32"
	ExePath           string            `json:"exe_path"`
	ExeArgs           string            `json:"exe_args,omitempty"`
	OperatingMode     string            `json:"operating_mode,omitempty"` // "standalone" | "portable" | "loader" | "native"
	Runner            string            `json:"runner,omitempty"`         // "umu", "wine", "proton", "linux", "auto"
	WinePrefix        string            `json:"wine_prefix,omitempty"`
	ProtonPath        string            `json:"proton_path,omitempty"`
	Language          string            `json:"language,omitempty"`
	MangoHud          *bool             `json:"mangohud,omitempty"`
	DXVKHud           string            `json:"dxvk_hud,omitempty"`
	WineFullscreenFSR *bool             `json:"wine_fullscreen_fsr,omitempty"`
	ProtonNVAPI       *bool             `json:"proton_nvapi,omitempty"`
	DllOverrides      string            `json:"dll_overrides,omitempty"`  // Custom Wine DLL overrides string, e.g. "d3d11=n,b;dxgi=n,b"
	CustomEnv         map[string]string `json:"custom_env,omitempty"`
	WorkingDir        string            `json:"working_dir,omitempty"`

	// Store & Metadata bindings
	Store           string `json:"store,omitempty"`
	StoreAppID      string `json:"store_app_id,omitempty"`
	GOGID           string `json:"gog_id,omitempty"`
	SGDBID          string `json:"sgdb_id,omitempty"`
	CustomCoverURL  string `json:"custom_cover_url,omitempty"`
	CustomBannerURL string `json:"custom_banner_url,omitempty"`
	CustomHeroURL   string `json:"custom_hero_url,omitempty"`
	CustomLogoURL   string `json:"custom_logo_url,omitempty"`
	CustomIconURL   string `json:"custom_icon_url,omitempty"`
	LockedArtwork   bool   `json:"locked_artwork,omitempty"`
}

// ---------------------------------------------------------------------------
// Focused sub-structs — each cluster of fields has exactly one producer and
// one consumer. Embedded in Game for backward-compatible flat field access
// and JSON serialization.
// ---------------------------------------------------------------------------

// InspectionResult holds filesystem scan and integration status for a game directory.
// Produced by engine.InspectLocal; consumed by CLI/GUI status displays.
type InspectionResult struct {
	OriginalBackups    []string `json:"original_backups,omitempty"`
	SettingsDirExists  bool     `json:"settings_dir_exists"`
	SteamAppIDTxtFound bool     `json:"steam_appid_txt_found"`
	CurrentLanguage    string   `json:"current_language,omitempty"`
	SupportedLanguages []string `json:"supported_languages,omitempty"`
	DiskSizeBytes      int64    `json:"disk_size_bytes"`
	FileCount          int      `json:"file_count"`
	DLCCount           int      `json:"dlc_count"`
	AchievementCount   int      `json:"achievement_count"`
}

// ManifestStatus holds depot manifest and file integrity state.
// Produced by manifest verification; consumed by update checks and integrity UI.
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

// StoreMetadata holds store-sourced remote metadata (news, patch notes).
// Produced by Steam API fetches; consumed by news/changelog UI.
type StoreMetadata struct {
	RecentPatchNote string     `json:"recent_patch_note,omitempty"`
	NewsItems       []NewsItem `json:"news_items,omitempty"`
}

// ReadinessStatus holds computed lifecycle flags derived from inspection data.
// Produced by RecomputeState; consumed by ready-state badges in CLI/GUI.
type ReadinessStatus struct {
	State              string `json:"state"` // "Original", "Loader-Configured", "Portable-Patched"
	IsPatched          bool   `json:"is_patched"`
	IsPortable         bool   `json:"is_portable"`
	HasUpdatesOrIssues bool   `json:"has_updates_or_issues"`
	IsReady            bool   `json:"is_ready"`
}

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

// ExternalIDs tracks cross-platform and database identifiers for a game.
type ExternalIDs struct {
	SteamAppID uint32 `json:"steam_appid,omitempty"`
	GOGID      string `json:"gog_id,omitempty"`
	EpicApp    string `json:"epic_app,omitempty"`
	SGDBID     string `json:"sgdb_id,omitempty"`
	IGDBID     string `json:"igdb_id,omitempty"`
}

// Canonical asset types
const (
	AssetCover  = "cover"
	AssetBanner = "banner"
	AssetHero   = "hero"
	AssetLogo   = "logo"
	AssetIcon   = "icon"
)

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

// Game is the canonical entity representing a managed game installation.
// Sub-structs are embedded for flat field access and backward-compatible JSON.
type Game struct {
	// Core identity
	AppID            string            `json:"app_id"`
	NumericAppID     uint32            `json:"numeric_app_id"`
	Name             string            `json:"name"`
	InstallDir       string            `json:"install_dir,omitempty"`
	Store            string            `json:"store"`  // "Steam", "GOG", "Epic", "Itch", "Standalone", "Custom"
	Source           GameSource        `json:"source"` // Canonical distribution source
	GameDir          string            `json:"game_dir"`
	Platform         string            `json:"platform"` // "linux", "win64", "win32"

	// Executable & launch configuration
	ExePath          string            `json:"exe_path"`
	ExeArgs          string            `json:"exe_args,omitempty"`
	LaunchCandidates []LaunchCandidate `json:"launch_candidates,omitempty"`
	LaunchProfile    *LaunchProfile    `json:"launch_profile,omitempty"`
	OperatingMode    string            `json:"operating_mode,omitempty"`
	Runner           string            `json:"runner,omitempty"`
	WinePrefix       string            `json:"wine_prefix,omitempty"`
	DllOverrides     string            `json:"dll_overrides,omitempty"`

	// Detection & patching
	TargetLibPath  string `json:"target_lib_path,omitempty"`
	ManifestPath   string `json:"manifest_path,omitempty"`
	RawDepotLayout bool   `json:"raw_depot_layout,omitempty"`

	// Focused sub-structs (embedded for flat access)
	ExternalIDs      `json:",inline"`
	GameArtwork      `json:",inline"`
	InspectionResult `json:",inline"`
	ManifestStatus   `json:",inline"`
	StoreMetadata    `json:",inline"`
	ReadinessStatus  `json:",inline"`
}

// NewGame initializes a Game entity with parsed numeric AppID and standard defaults.
func NewGame(name, gameDir, appID, platform string) *Game {
	numAppID, _ := util.ParseAppID(appID)
	store := StoreSteam
	if numAppID == 0 && appID != "" && appID != "0" {
		store = StoreStandalone
	}
	return &Game{
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
		ReadinessStatus: ReadinessStatus{
			State: "Original",
		},
		OperatingMode: string(ModeStandalone),
		InspectionResult: InspectionResult{
			SupportedLanguages: []string{"english"},
			OriginalBackups:    make([]string, 0),
		},
		LaunchCandidates: make([]LaunchCandidate, 0),
		ManifestStatus: ManifestStatus{
			ModifiedFiles:  make([]string, 0),
			MissingFiles:   make([]string, 0),
			UntrackedFiles: make([]string, 0),
			RedistModified: make([]string, 0),
			RedistMissing:  make([]string, 0),
		},
		StoreMetadata: StoreMetadata{
			NewsItems: make([]NewsItem, 0),
		},
	}
}

// SteamAppID returns the numeric AppID if present.
func (g *Game) SteamAppID() (uint32, bool) {
	if g.ExternalIDs.SteamAppID > 0 {
		return g.ExternalIDs.SteamAppID, true
	}
	if g.NumericAppID > 0 {
		return g.NumericAppID, true
	}
	if g.AppID != "" {
		return util.ParseAppID(g.AppID)
	}
	return 0, false
}

// IsSteam returns true if game originated from or integrates with Steam.
func (g *Game) IsSteam() bool {
	return strings.EqualFold(g.Store, StoreSteam) || g.NumericAppID > 0
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
	return g.OperatingMode == string(ModeLoader) ||
		g.State == "Loader-Configured" ||
		strings.Contains(strings.ToLower(g.ExePath), "steamclient_loader")
}

// IsPortablePatched returns true if Goldberg emulator libraries are directly injected into the game tree.
func (g *Game) IsPortablePatched() bool {
	return g.IsPortable || g.State == "Portable-Patched" || g.HasBackups()
}

// RequiresSteamEmulator returns true if the game requires Steamworks mocking.
func (g *Game) RequiresSteamEmulator() bool {
	if !g.IsSteam() && g.AppID == "" {
		return false
	}
	return true
}

// RecomputeState recalculates computed flags (IsPatched, IsPortable, IsReady, HasUpdatesOrIssues).
func (g *Game) RecomputeState() {
	if len(g.OriginalBackups) > 0 {
		g.State = "Portable-Patched"
	} else if g.SettingsDirExists || g.SteamAppIDTxtFound || g.OperatingMode == string(ModeLoader) {
		g.State = "Loader-Configured"
	} else if g.State == "" {
		g.State = "Original"
	}

	g.IsPortable = g.State == "Portable-Patched"
	g.IsPatched = g.IsPortable || g.State == "Loader-Configured" || g.SettingsDirExists
	g.HasUpdatesOrIssues = g.IsSteam() && (g.HasUpdate || len(g.ModifiedFiles) > 0 || len(g.MissingFiles) > 0)
	g.IsReady = g.IsPatched || g.ExePath != ""
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

