package saves

// Location category constants
const (
	CategorySave     = "save"
	CategoryConfig   = "config"
	CategoryCloud    = "cloud"
	CategoryPortable = "portable"
	CategoryOther    = "other"
)

// SaveLocation represents a resolved on-disk save or configuration path for a game.
type SaveLocation struct {
	Path        string   `json:"path"`
	DisplayPath string   `json:"display_path"`
	Category    string   `json:"category"` // "save", "config", "cloud", "portable", "other"
	Tags        []string `json:"tags,omitempty"`
	Exists      bool     `json:"exists"`
	IsDir       bool     `json:"is_dir"`
	FileCount   int      `json:"file_count"`
	SizeBytes   int64    `json:"size_bytes"`
	Source      string   `json:"source"` // "ludusavi", "gbe", "wine_prefix", "portable"
	SharedWith  []string `json:"shared_with,omitempty"`
}

// SaveLocationsResult consolidates all discovered data locations and aggregate stats for a game.
type SaveLocationsResult struct {
	GameDir           string         `json:"game_dir"`
	GameName          string         `json:"game_name"`
	AppID             string         `json:"app_id,omitempty"`
	Platform          string         `json:"platform,omitempty"`
	Locations         []SaveLocation `json:"locations"`
	TotalSaveBytes    int64          `json:"total_save_bytes"`
	TotalConfigFiles  int            `json:"total_config_files"`
	HasCloudSaves     bool           `json:"has_cloud_saves"`
	PrimarySavePath   string         `json:"primary_save_path,omitempty"`
	PrimaryConfigPath string         `json:"primary_config_path,omitempty"`
}

// ManifestConstraint defines condition filters for when a rule applies.
type ManifestConstraint struct {
	OS    string `json:"os,omitempty"`
	Store string `json:"store,omitempty"`
}

// ManifestFileRule defines file glob path templates and associated tags/conditions in Ludusavi.
type ManifestFileRule struct {
	Tags []string             `json:"tags,omitempty"`
	When []ManifestConstraint `json:"when,omitempty"`
}

// ManifestGame defines a single game definition in the Ludusavi database.
type ManifestGame struct {
	Alias string                      `json:"alias,omitempty"`
	Files map[string]ManifestFileRule `json:"files,omitempty"`
	Steam struct {
		ID uint32 `json:"id,omitempty"`
	} `json:"steam,omitempty"`
	GOG struct {
		ID uint64 `json:"id,omitempty"`
	} `json:"gog,omitempty"`
	Cloud struct {
		Steam  bool `json:"steam,omitempty"`
		Epic   bool `json:"epic,omitempty"`
		GOG    bool `json:"gog,omitempty"`
		Origin bool `json:"origin,omitempty"`
		Uplay  bool `json:"uplay,omitempty"`
	} `json:"cloud,omitempty"`
}

// BackupFileEntry represents an individual file archived within a backup snapshot.
type BackupFileEntry struct {
	SemanticPath         string `json:"semantic_path"`          // e.g. "<winAppData>/IronGate/Valheim/save.dat"
	RelativeZipPath      string `json:"relative_zip_path"`      // Relative path inside .zip
	OriginalPhysicalPath string `json:"original_physical_path"` // Physical path at time of backup
	SizeBytes            int64  `json:"size_bytes"`
	Hash                 string `json:"hash,omitempty"`
	ModifiedTime         int64  `json:"modified_time"`
}

// BackupMetadata stores semantic mapping and metadata inside backup_meta.json in each archive.
type BackupMetadata struct {
	ID             string            `json:"id"`
	AppID          string            `json:"app_id,omitempty"`
	GameTitle      string            `json:"game_title"`
	GameDir        string            `json:"game_dir,omitempty"`
	Platform       string            `json:"platform,omitempty"`
	CreatedAt      int64             `json:"created_at"`
	CreatedBy      string            `json:"created_by"` // "manual" | "auto_exit"
	Note           string            `json:"note,omitempty"`
	Files          []BackupFileEntry `json:"files"`
	TotalSizeBytes int64             `json:"total_size_bytes"`
}

// BackupInfo summarizes an available backup snapshot for UI presentation and listing.
type BackupInfo struct {
	ID        string `json:"id"`
	ZipPath   string `json:"zip_path"`
	AppID     string `json:"app_id,omitempty"`
	GameTitle string `json:"game_title"`
	CreatedAt int64  `json:"created_at"`
	CreatedBy string `json:"created_by"`
	Note      string `json:"note,omitempty"`
	FileCount int    `json:"file_count"`
	SizeBytes int64  `json:"size_bytes"`
}
