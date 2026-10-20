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

// LaunchCandidate represents an executable or configured launch target in a game directory.
type LaunchCandidate struct {
	Name        string `json:"name"`
	Executable  string `json:"executable"`
	Arguments   string `json:"arguments,omitempty"`
	Description string `json:"description,omitempty"`
}

// GameRecord captures the immutable on-disk identity, directory layout, and inventory metrics of an installed game.
// All fields reflect physical on-disk baseline data discovered via detector, ACF manifests, and depot scanners.
// To override execution or environment behaviors, use LaunchProfile instead.
type GameRecord struct {
	GameDir            string     `json:"game_dir"`
	AppID              string     `json:"app_id"`
	NumericAppID       uint32     `json:"numeric_app_id"`
	Name               string     `json:"name"`
	InstallDir         string     `json:"install_dir,omitempty"`
	Store              string     `json:"store"`    // "Steam", "GOG", "Epic", "Itch", "Standalone", "Custom"
	Source             GameSource `json:"source"`   // Canonical distribution source
	Platform           string     `json:"platform"` // "linux", "win64", "win32"
	Branch             string     `json:"branch,omitempty"`
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
	ModTopology        ModTopology       `json:"mod_topology"`
	Mods               []InstalledMod    `json:"mods,omitempty"`
	OutdatedModsCount  int               `json:"outdated_mods_count,omitempty"`
}

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

// Game is the canonical entity representing a managed game installation.
// It composes the static on-disk record, user launch configuration, computed readiness,
// manifest integrity status, store metadata, and graphical assets.
type Game struct {
	GameRecord      `json:",inline"`
	LaunchProfile   *LaunchProfile `json:"launch_profile,omitempty"`
	ReadinessStatus `json:",inline"`
	GameArtwork     `json:",inline"`
	ManifestStatus  `json:",inline"`
	StoreMetadata   `json:",inline"`
	VersionStatus   *VersionStatus `json:"version_status,omitempty"`
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
	if g.ModTopology.SupportedFormats != nil {
		cp.ModTopology.SupportedFormats = append([]string(nil), g.ModTopology.SupportedFormats...)
	}
	if g.Mods != nil {
		cp.Mods = make([]InstalledMod, len(g.Mods))
		for i, m := range g.Mods {
			cp.Mods[i] = m
			if m.Files != nil {
				cp.Mods[i].Files = append([]string(nil), m.Files...)
			}
		}
	}
	return &cp
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
				if g.NumericAppID == 0 || g.AppID == "" || g.AppID == "0" || launchProfileExists {
					g.AppID = profile.MetadataID
					g.NumericAppID = num
				}
			}
		} else if strings.EqualFold(profile.MetadataProvider, "gog") {
			g.ExternalIDs.GOGID = profile.MetadataID
			g.GOGID = profile.MetadataID
		} else if strings.EqualFold(profile.MetadataProvider, "rawg") {
			g.ExternalIDs.RAWGID = profile.MetadataID
			g.RAWGID = profile.MetadataID
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
		g.GOGID = profile.GOGID
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
