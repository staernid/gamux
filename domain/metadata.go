package domain

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
	RemoteManifestID string `json:"remote_manifest_id,omitempty"`
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
	Category        string `json:"category"`                   // "common", "windows", "linux", "macos", "soundtrack"
	ContentCategory string `json:"content_category,omitempty"` // "core", "locale:<language>", "soundtrack", "optional_hd", "dlc"
	Language        string `json:"language,omitempty"`         // e.g. "french", "german", "english"
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

// ModTopology defines the detected game engine mod layout and active mod directory.
type ModTopology struct {
	Engine           string   `json:"engine"`                     // "unity", "unreal", "generic"
	ModDir           string   `json:"mod_dir"`                    // Relative path from GameDir (e.g. "BepInEx/plugins", "Content/Paks/~mods", "mods")
	AbsoluteModDir   string   `json:"absolute_mod_dir,omitempty"` // Absolute path on disk
	Detected         bool     `json:"detected"`
	SupportedFormats []string `json:"supported_formats,omitempty"` // e.g. [".dll"], [".pak"]
}

// InstalledMod represents an installed game mod or workshop item on disk.
type InstalledMod struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	WorkshopID        uint64   `json:"workshop_id,omitempty"`
	Enabled           bool     `json:"enabled"`
	Path              string   `json:"path"`
	RelativePath      string   `json:"relative_path"`
	TimeUpdated       int64    `json:"time_updated,omitempty"`
	LatestTimeUpdated int64    `json:"latest_time_updated,omitempty"`
	HasUpdate         bool     `json:"has_update"`
	Description       string   `json:"description,omitempty"`
	PreviewURL        string   `json:"preview_url,omitempty"`
	Size              int64    `json:"size"`
	Files             []string `json:"files,omitempty"`
}

// GameModStatus encapsulates the mod topology, installed mods list, and update status for a game.
type GameModStatus struct {
	Topology      ModTopology    `json:"topology"`
	Mods          []InstalledMod `json:"mods"`
	OutdatedCount int            `json:"outdated_count"`
}
