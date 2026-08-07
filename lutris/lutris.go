// Package lutris generates Lutris YAML game config files.
//
// The generated YAML conforms to the schema that Lutris reads from
// ~/.config/lutris/games/*.yml and is suitable for manually-installed
// Windows games (Wine/Proton) as well as native Linux games.
package lutris

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all user-supplied values needed to generate a Lutris
// game entry.  Fields that are left at their zero value are omitted from
// the generated YAML.
type Config struct {
	// Name is the human-readable game title (required).
	Name string

	// Slug is a short, stable identifier used in the filename and inside
	// the YAML "slug" field.  When empty, Name is slugified automatically.
	Slug string

	// GamePath is the absolute path to the game executable (required).
	GamePath string

	// PrefixPath is the absolute path to the Wine prefix directory.
	// Only meaningful when Runner is "wine"; ignored otherwise.
	PrefixPath string

	// Runner selects the runtime environment:
	//   "wine"   – Windows game launched through Wine/Proton
	//   "linux"  – native Linux binary
	// When empty it defaults to "wine".  Case-insensitive.
	Runner string

	// Arch specifies the CPU architecture for the Wine prefix. Typical
	// values are "win32", "win64", or "auto".  Only used for Wine games.
	Arch string

	// Args are additional command-line arguments passed to the exe.
	Args string

	// Wine holds Wine-specific settings (battleye, eac, etc.).
	Wine *WineConfig

	// System holds system-level overrides (MangoHud, GPU, env vars).
	System *SystemConfig

	// Env is a flat map of environment variables applied at launch.
	// It is merged into the "system.env" block in the YAML.
	Env map[string]string

	// Version is an optional version string (e.g. "GOG", "1.2.3").
	Version string

	// Year is an optional release year string.
	Year string

	// CreateDesktopShortcut creates a desktop launcher (~/Desktop/net.lutris.<slug>.desktop).
	CreateDesktopShortcut bool

	// CreateMenuShortcut creates an application menu launcher (~/.local/share/applications/net.lutris.<slug>.desktop).
	CreateMenuShortcut bool
}

// WineConfig groups Wine/Proton toggles and the runner version string.
type WineConfig struct {
	Battleye  bool   `yaml:"battleye,omitempty"`
	EAC       bool   `yaml:"eac,omitempty"`
	FSR       bool   `yaml:"fsr,omitempty"`
	VKD3D     bool   `yaml:"vkd3d,omitempty"`
	DXVKNVAPI bool   `yaml:"dxvk_nvapi,omitempty"`
	Version   string `yaml:"version,omitempty"`
}

// SystemConfig groups optional system-level overrides.
type SystemConfig struct {
	MangoHud bool   `yaml:"mangohud,omitempty"`
	GPU      string `yaml:"gpu,omitempty"`
	VkICD    string `yaml:"vk_icd,omitempty"`
	Locale   string `yaml:"locale,omitempty"`
}

// ── YAML model ──────────────────────────────────────────────────────

// The private document type mirrors the actual Lutris YAML structure.
type doc struct {
	Game     gameSection    `yaml:"game"`
	GameSlug string         `yaml:"game_slug,omitempty"`
	Name     string         `yaml:"name"`
	Slug     string         `yaml:"slug"`
	Runner   string         `yaml:"runner,omitempty"`
	Version  string         `yaml:"version,omitempty"`
	Year     string         `yaml:"year,omitempty"`
	Wine     *wineSection   `yaml:"wine,omitempty"`
	System   *systemSection `yaml:"system,omitempty"`
}

type installerDoc struct {
	Name     string          `yaml:"name"`
	GameSlug string          `yaml:"game_slug"`
	Slug     string          `yaml:"slug"`
	Version  string          `yaml:"version,omitempty"`
	Runner   string          `yaml:"runner"`
	Script   installerScript `yaml:"script"`
}

type installerScript struct {
	Game   gameSection    `yaml:"game"`
	Wine   *wineSection   `yaml:"wine,omitempty"`
	System *systemSection `yaml:"system,omitempty"`
}

type gameSection struct {
	Arch       string `yaml:"arch,omitempty"`
	Args       string `yaml:"args,omitempty"`
	Exe        string `yaml:"exe"`
	WorkingDir string `yaml:"working_dir,omitempty"`
	Prefix     string `yaml:"prefix,omitempty"`
	GogID      string `yaml:"gogid,omitempty"`
}

type wineSection struct {
	Battleye  bool   `yaml:"battleye,omitempty"`
	EAC       bool   `yaml:"eac,omitempty"`
	FSR       bool   `yaml:"fsr,omitempty"`
	VKD3D     bool   `yaml:"vkd3d,omitempty"`
	DXVKNVAPI bool   `yaml:"dxvk_nvapi,omitempty"`
	Version   string `yaml:"version,omitempty"`
}

type systemSection struct {
	Env      map[string]string `yaml:"env,omitempty"`
	MangoHud bool              `yaml:"mangohud,omitempty"`
	GPU      string            `yaml:"gpu,omitempty"`
	VkICD    string            `yaml:"vk_icd,omitempty"`
	Locale   string            `yaml:"locale,omitempty"`
}

// ── Public API ──────────────────────────────────────────────────────

// Generate produces the Lutris YAML game entry as a []byte slice.
func Generate(cfg Config) ([]byte, error) {
	if err := validate(cfg); err != nil {
		return nil, err
	}

	d := buildDoc(cfg)

	out, err := yaml.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("lutris: marshal YAML: %w", err)
	}
	return out, nil
}

// GenerateInstaller produces a Lutris installer YAML script suitable for `lutris -i`.
func GenerateInstaller(cfg Config) ([]byte, error) {
	if err := validate(cfg); err != nil {
		return nil, err
	}

	d := buildInstallerDoc(cfg)

	out, err := yaml.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("lutris: marshal installer YAML: %w", err)
	}
	return out, nil
}

// Write generates the YAML and writes it to the given directory (which
// should be ~/.config/lutris/games/ or equivalent). The filename stem matches
// Lutris's expected timestamped configpath format (<slug>-<timestamp>.yml).
// It also updates existing matching Lutris config files, registers/updates
// the game in Lutris's SQLite database (pga.db), and generates desktop/menu shortcuts.
func Write(cfg Config, dir string) error {
	out, err := Generate(cfg)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("lutris: create dir %s: %w", dir, err)
	}

	slug := strings.TrimSpace(cfg.Slug)
	if slug == "" {
		slug = Slugify(cfg.Name)
	}

	timestamp := time.Now().Unix()
	configpath := fmt.Sprintf("%s-%d", slug, timestamp)

	// 1. Primary path: <configpath>.yml
	primaryPath := filepath.Join(dir, configpath+".yml")
	if err := os.WriteFile(primaryPath, out, 0644); err != nil {
		return fmt.Errorf("lutris: write file %s: %w", primaryPath, err)
	}

	// 2. Also write <slug>.yml for fallback compatibility
	fallbackPath := filepath.Join(dir, slug+".yml")
	_ = os.WriteFile(fallbackPath, out, 0644)

	// 3. Overwrite any existing Lutris game files in dir matching <slug>*.yml
	// so existing Lutris UI entries update immediately.
	shortPrefix := strings.Split(slug, "-")[0]
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".yml") {
				nameNoExt := strings.TrimSuffix(e.Name(), ".yml")
				if strings.HasPrefix(nameNoExt, slug) || (len(shortPrefix) >= 4 && strings.HasPrefix(nameNoExt, shortPrefix)) {
					targetPath := filepath.Join(dir, e.Name())
					_ = os.WriteFile(targetPath, out, 0644)
				}
			}
		}
	}

	// 4. Attempt to register in Lutris's SQLite database (non-fatal if database or sqlite3 not present)
	_ = updateLutrisDB(cfg, configpath)

	// 5. Create XDG Desktop and/or Application Menu shortcuts if requested
	_ = CreateXDGShortcuts(cfg, slug)

	return nil
}

func sqlQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func updateLutrisDB(cfg Config, configpath string) error {
	return updateLutrisDBWithPath(cfg, configpath, "")
}

func updateLutrisDBWithPath(cfg Config, configpath string, overrideDbPath string) error {
	dbPath := overrideDbPath
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		dbLocations := []string{
			filepath.Join(home, ".local", "share", "lutris", "pga.db"),
			filepath.Join(home, ".var", "app", "net.lutris.Lutris", "data", "lutris", "pga.db"),
		}

		if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
			dbLocations = append([]string{filepath.Join(dataHome, "lutris", "pga.db")}, dbLocations...)
		}

		for _, loc := range dbLocations {
			if _, err := os.Stat(loc); err == nil {
				dbPath = loc
				break
			}
		}
	}

	if dbPath == "" {
		return nil
	}

	slug := strings.TrimSpace(cfg.Slug)
	if slug == "" {
		slug = Slugify(cfg.Name)
	}

	runner := strings.ToLower(strings.TrimSpace(cfg.Runner))
	if runner == "" {
		runner = "wine"
	}

	platform := "Windows"
	if runner == "linux" {
		platform = "Linux"
	}

	gameDir := filepath.Dir(cfg.GamePath)
	now := time.Now().Unix()

	// Check if game already exists in pga.db
	checkQuery := fmt.Sprintf(`SELECT COUNT(*) FROM games WHERE slug = %s;`, sqlQuote(slug))
	cmdCheck := exec.Command("sqlite3", dbPath, checkQuery)
	out, err := cmdCheck.Output()

	exists := false
	if err == nil && strings.TrimSpace(string(out)) != "0" {
		exists = true
	}

	var query string
	if exists {
		query = fmt.Sprintf(`UPDATE games SET name = %s, runner = %s, platform = %s, executable = %s, directory = %s, installed = 1, installed_at = %d, configpath = %s WHERE slug = %s;`,
			sqlQuote(cfg.Name), sqlQuote(runner), sqlQuote(platform), sqlQuote(cfg.GamePath), sqlQuote(gameDir), now, sqlQuote(configpath), sqlQuote(slug))
	} else {
		query = fmt.Sprintf(`INSERT INTO games (name, slug, runner, platform, executable, directory, installed, installed_at, configpath) VALUES (%s, %s, %s, %s, %s, %s, 1, %d, %s);`,
			sqlQuote(cfg.Name), sqlQuote(slug), sqlQuote(runner), sqlQuote(platform), sqlQuote(cfg.GamePath), sqlQuote(gameDir), now, sqlQuote(configpath))
	}

	cmd := exec.Command("sqlite3", dbPath, query)
	return cmd.Run()
}

// CreateXDGShortcuts creates application menu (.local/share/applications) and/or desktop (~/Desktop) launcher files.
func CreateXDGShortcuts(cfg Config, slug string) error {
	if !cfg.CreateMenuShortcut && !cfg.CreateDesktopShortcut {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Icon=lutris_%s
Exec=env LUTRIS_SKIP_INIT=1 lutris lutris:rungame/%s
Categories=Game;
`, cfg.Name, slug, slug)

	filename := fmt.Sprintf("net.lutris.%s.desktop", slug)

	if cfg.CreateMenuShortcut {
		appsDir := filepath.Join(home, ".local", "share", "applications")
		_ = os.MkdirAll(appsDir, 0755)
		menuPath := filepath.Join(appsDir, filename)
		if err := os.WriteFile(menuPath, []byte(content), 0755); err != nil {
			return fmt.Errorf("lutris: write menu shortcut %s: %w", menuPath, err)
		}
	}

	if cfg.CreateDesktopShortcut {
		desktopDir := filepath.Join(home, "Desktop")
		_ = os.MkdirAll(desktopDir, 0755)
		desktopPath := filepath.Join(desktopDir, filename)
		if err := os.WriteFile(desktopPath, []byte(content), 0755); err != nil {
			return fmt.Errorf("lutris: write desktop shortcut %s: %w", desktopPath, err)
		}
	}

	return nil
}

// WriteInstaller generates the installer YAML and writes it to path.
func WriteInstaller(cfg Config, path string) error {
	out, err := GenerateInstaller(cfg)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("lutris: create dir %s: %w", dir, err)
	}

	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("lutris: write installer file %s: %w", path, err)
	}
	return nil
}

// LaunchInstaller launches `lutris -i <path>` in the background.
func LaunchInstaller(path string) error {
	lutrisPath, err := exec.LookPath("lutris")
	if err != nil {
		return fmt.Errorf("lutris executable not found in PATH: %w", err)
	}

	cmd := exec.Command(lutrisPath, "-i", path)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start lutris -i: %w", err)
	}
	return nil
}

// ── helpers ─────────────────────────────────────────────────────────

// Slugify converts a game title into a clean Lutris-compatible slug.
func Slugify(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "'", "")
	// Remove any remaining characters that aren't alphanumeric, dash, or dot.
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-")
}

func validate(cfg Config) error {
	if strings.TrimSpace(cfg.Name) == "" {
		return fmt.Errorf("lutris: Name is required")
	}
	if strings.TrimSpace(cfg.GamePath) == "" {
		return fmt.Errorf("lutris: GamePath is required")
	}
	runner := strings.ToLower(strings.TrimSpace(cfg.Runner))
	if runner == "" {
		runner = "wine"
	}
	switch runner {
	case "wine", "linux":
	default:
		return fmt.Errorf("lutris: Runner must be \"wine\" or \"linux\", got %q", cfg.Runner)
	}
	return nil
}

func buildDoc(cfg Config) doc {
	runner := strings.ToLower(strings.TrimSpace(cfg.Runner))
	if runner == "" {
		runner = "wine"
	}

	slug := strings.TrimSpace(cfg.Slug)
	if slug == "" {
		slug = Slugify(cfg.Name)
	}

	var w *wineSection
	var prefix string
	arch := cfg.Arch

	if runner == "wine" {
		prefix = cfg.PrefixPath
		if arch == "" {
			arch = "win64"
		}

		w = &wineSection{
			Battleye:  false,
			EAC:       false,
			FSR:       false,
			VKD3D:     false,
			DXVKNVAPI: false,
		}
		if cfg.Wine != nil {
			w.Battleye = cfg.Wine.Battleye
			w.EAC = cfg.Wine.EAC
			w.FSR = cfg.Wine.FSR
			w.VKD3D = cfg.Wine.VKD3D
			w.DXVKNVAPI = cfg.Wine.DXVKNVAPI
			w.Version = cfg.Wine.Version
		}
	}

	var ss *systemSection
	hasSystem := false
	if cfg.System != nil {
		if cfg.System.MangoHud ||
			cfg.System.GPU != "" ||
			cfg.System.VkICD != "" ||
			cfg.System.Locale != "" {
			hasSystem = true
		}
	}
	if len(cfg.Env) > 0 {
		hasSystem = true
	}

	if hasSystem {
		ss = &systemSection{
			Env: make(map[string]string),
		}
		if cfg.System != nil {
			ss.MangoHud = cfg.System.MangoHud
			ss.GPU = cfg.System.GPU
			ss.VkICD = cfg.System.VkICD
			ss.Locale = cfg.System.Locale
		}
		for k, v := range cfg.Env {
			ss.Env[k] = v
		}
		// Keep env set even if empty to match real Lutris files that
		// sometimes include "env: {}".
		if len(ss.Env) == 0 {
			ss.Env = nil // omit from YAML when truly empty
		}
	}

	// For native Linux games omit wine + arch + prefix.
	if runner != "wine" {
		prefix = ""
		arch = ""
		w = nil
	}

	return doc{
		Game: gameSection{
			Arch:   arch,
			Args:   cfg.Args,
			Exe:    cfg.GamePath,
			Prefix: prefix,
		},
		GameSlug: slug,
		Name:     cfg.Name,
		Slug:     slug,
		Runner:   runner,
		Version:  cfg.Version,
		Year:     cfg.Year,
		Wine:     w,
		System:   ss,
	}
}

func buildInstallerDoc(cfg Config) installerDoc {
	runner := strings.ToLower(strings.TrimSpace(cfg.Runner))
	if runner == "" {
		runner = "wine"
	}

	slug := strings.TrimSpace(cfg.Slug)
	if slug == "" {
		slug = Slugify(cfg.Name)
	}

	version := cfg.Version
	if version == "" {
		version = "Gamux"
	}

	d := buildDoc(cfg)
	if d.Game.WorkingDir == "" && d.Game.Exe != "" {
		d.Game.WorkingDir = filepath.Dir(d.Game.Exe)
	}

	return installerDoc{
		Name:     cfg.Name,
		GameSlug: slug,
		Slug:     slug + "-installer",
		Version:  version,
		Runner:   runner,
		Script: installerScript{
			Game:   d.Game,
			Wine:   d.Wine,
			System: d.System,
		},
	}
}
