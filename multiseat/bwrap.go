package multiseat

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/util"
)

// BwrapOptions holds settings for generating the Bubblewrap sandbox arguments.
type BwrapOptions struct {
	SeatID          string            // "p1" | "p2"
	ProfileDir      string            // Base path for profile data (~/.local/share/gamux/profiles/<seat_id>)
	GameDir         string            // Path to game installation (mounted read-only)
	WinePrefix      string            // Isolated Wine prefix
	GamepadNode     string            // Specific /dev/input/eventX node (all others masked)
	KeyboardNode    string            // Specific /dev/input/eventX keyboard node (all other keyboards masked)
	MouseNode       string            // Specific /dev/input/eventX mouse node (all other mice masked)
	MaskInputNodes  []string          // Explicit additional /dev/input nodes to mask with /dev/null
	AudioSink       string            // PULSE_SINK value
	WaylandDisplay  string            // WAYLAND_DISPLAY value
	DisplayEnv      string            // DISPLAY value
	CustomEnv       map[string]string // Additional environment variables
	Command         []string          // Inner command to execute
	DieWithParent   bool
}

// EnsureSeatProfile prepares the isolated filesystem directory and Goldberg persona files for a seat.
func EnsureSeatProfile(profileDir string, profile PlayerProfile) error {
	if profileDir == "" {
		return fmt.Errorf("profile dir cannot be empty")
	}

	homeDir := filepath.Join(profileDir, "home")
	gseDir := filepath.Join(homeDir, ".local", "share", "GSE Saves", "settings")
	if err := os.MkdirAll(gseDir, 0755); err != nil {
		return fmt.Errorf("create gse settings dir: %w", err)
	}

	// 1. Write Goldberg configs.user.ini
	name := profile.AccountName
	if name == "" {
		name = "Player"
	}
	steamID := profile.AccountSteamID
	if steamID == "" {
		steamID = "76561197960287930"
	}
	lang := profile.Language
	if lang == "" {
		lang = "english"
	}
	country := profile.IPCountry
	if country == "" {
		country = "US"
	}

	userCfg := &gbe.GSEUserConfig{
		AccountName:    name,
		AccountSteamID: steamID,
		Language:       lang,
		IPCountry:      country,
	}
	if err := gbe.SaveGSEUserConfigToPath(filepath.Join(gseDir, "configs.user.ini"), userCfg); err != nil {
		return fmt.Errorf("write configs.user.ini: %w", err)
	}

	// 2. Also write standard fallback text files (used by various emulator forks)
	_ = os.WriteFile(filepath.Join(gseDir, "account_name.txt"), []byte(name), 0644)
	_ = os.WriteFile(filepath.Join(gseDir, "user_steam_id.txt"), []byte(steamID), 0644)
	_ = os.WriteFile(filepath.Join(gseDir, "language.txt"), []byte(lang), 0644)
	_ = os.WriteFile(filepath.Join(gseDir, "custom_broadcasts.txt"), []byte("127.0.0.1\n10.0.2.2\n"), 0644)

	// 3. Ensure base cache and config directories exist for the seat
	_ = os.MkdirAll(filepath.Join(homeDir, ".cache"), 0755)
	_ = os.MkdirAll(filepath.Join(homeDir, ".config"), 0755)

	// Clean up any stale compatibilitytools.d symlink so bwrap won't fail to mount on it
	seatSteamCompat := filepath.Join(homeDir, ".local", "share", "Steam", "compatibilitytools.d")
	if fi, err := os.Lstat(seatSteamCompat); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		_ = os.Remove(seatSteamCompat)
	}

	return nil
}

// BuildBwrapArgs builds the bubblewrap CLI arguments.
func BuildBwrapArgs(opts BwrapOptions) ([]string, error) {
	if opts.GameDir == "" {
		return nil, fmt.Errorf("game dir is required")
	}
	if opts.ProfileDir == "" {
		return nil, fmt.Errorf("profile dir is required")
	}
	if len(opts.Command) == 0 {
		return nil, fmt.Errorf("inner command is required")
	}

	realHome, _ := os.UserHomeDir()
	isP1 := opts.SeatID == "p1"

	var homeDir string
	if isP1 {
		homeDir = realHome
	} else {
		homeDir = filepath.Join(opts.ProfileDir, "home")
		if err := os.MkdirAll(homeDir, 0755); err != nil {
			return nil, fmt.Errorf("create profile home dir: %w", err)
		}
	}

	var args []string

	if opts.DieWithParent {
		args = append(args, "--die-with-parent")
	}

	// Always ensure non-root execution inside sandbox:
	// Network namespaces (such as pasta) map the caller to root (uid 0) inside their user namespace.
	// Running as root causes Proton to abort ("ERROR: This script should never be run as the root user")
	// and causes DBus security policy rejections for GameMode.
	// Map back to the user's real host UID and GID via --unshare-user.
	uid := os.Getuid()
	gid := os.Getgid()
	if uid != 0 {
		args = append(args, "--unshare-user", "--uid", fmt.Sprintf("%d", uid), "--gid", fmt.Sprintf("%d", gid))
	}

	// 1. Base system read-only bindings
	systemDirs := []string{"/usr", "/etc"}
	for _, dir := range systemDirs {
		if util.FileExists(dir) {
			args = append(args, "--ro-bind", dir, dir)
		}
	}

	// Handle symlinked or real root directories (/bin, /lib, /lib64, /sbin)
	rootLinks := []string{"/bin", "/lib", "/lib64", "/sbin"}
	for _, p := range rootLinks {
		if fi, err := os.Lstat(p); err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				if target, err := os.Readlink(p); err == nil {
					args = append(args, "--symlink", target, p)
				}
			} else if fi.IsDir() {
				args = append(args, "--ro-bind", p, p)
			}
		}
	}

	// Proc, Dev, Run, Tmp
	userRunDir := fmt.Sprintf("/run/user/%d", uid)
	args = append(args,
		"--proc", "/proc",
		"--dev-bind", "/dev", "/dev",
		"--ro-bind-try", "/sys", "/sys",
		"--ro-bind-try", "/run", "/run",
		"--bind-try", userRunDir, userRunDir,
		"--tmpfs", "/tmp",
		"--bind-try", "/tmp/.X11-unix", "/tmp/.X11-unix",
		"--bind-try", "/dev/shm", "/dev/shm",
	)

	// 2. Input Isolation:
	// Find all host gamepad devices across event*, js*, and hidraw* subsystems.
	// Mask any gamepad nodes not assigned to this seat with /dev/null so Wine/SDL
	// cannot detect or poll unassigned gamepads.
	allGamepadNodes := FindAllHostGamepadNodes()
	assignedNodes := make(map[string]bool)
	if isExplicitDeviceNode(opts.GamepadNode) {
		assignedNodes[opts.GamepadNode] = true
		if util.FileExists(opts.GamepadNode) {
			jsNodes, hidrawNodes, siblingEventNodes := FindGamepadCompanionNodes(opts.GamepadNode)
			for _, n := range jsNodes {
				assignedNodes[n] = true
			}
			for _, n := range hidrawNodes {
				assignedNodes[n] = true
			}
			for _, n := range siblingEventNodes {
				assignedNodes[n] = true
			}
		}
	}

	for _, node := range allGamepadNodes {
		if !assignedNodes[node] && util.FileExists(node) {
			args = append(args, "--ro-bind", "/dev/null", node)
		}
	}

	// Explicit additional input nodes to mask
	for _, node := range opts.MaskInputNodes {
		if node != "" && util.FileExists(node) {
			args = append(args, "--ro-bind", "/dev/null", node)
		}
	}

	// 3. User Home Directory
	if realHome != "" && util.FileExists(realHome) {
		if isP1 {
			// Player 1 is the primary host user: bind real home read-write directly
			// without virtualizing $HOME, cache, config, or save overlays.
			args = append(args, "--bind", realHome, realHome)
		} else {
			// Player 2 is the guest seat: base realHome is read-only
			args = append(args, "--ro-bind", realHome, realHome)

			// Python (umu, protonfixes), GLib, and C runtimes query /etc/passwd via getpwuid(),
			// which resolves '~' directly to realHome rather than the sandboxed $HOME.
			// Expose seat-isolated writable .cache and .config over realHome to prevent
			// 'Read-only file system' crashes during startup.
			cacheDir := filepath.Join(homeDir, ".cache")
			_ = os.MkdirAll(cacheDir, 0755)
			realCache := filepath.Join(realHome, ".cache")
			if util.FileExists(realCache) {
				args = append(args, "--bind", cacheDir, realCache)
			}

			configDir := filepath.Join(homeDir, ".config")
			_ = os.MkdirAll(configDir, 0755)
			realConfig := filepath.Join(realHome, ".config")
			if util.FileExists(realConfig) {
				args = append(args, "--bind", configDir, realConfig)
			}

			// UMU runtime and pressure-vessel require write access for locks (umu.lock) and runtime refs (.ref)
			realUmu := filepath.Join(realHome, ".local", "share", "umu")
			if util.FileExists(realUmu) {
				args = append(args, "--bind", realUmu, realUmu)
			}

			// Mirror Steam compatibilitytools.d into seat home so umu discovers installed Protons
			realSteamCompat := filepath.Join(realHome, ".local", "share", "Steam", "compatibilitytools.d")
			seatSteamCompat := filepath.Join(homeDir, ".local", "share", "Steam", "compatibilitytools.d")
			if util.FileExists(realSteamCompat) {
				if fi, err := os.Lstat(seatSteamCompat); err == nil && fi.Mode()&os.ModeSymlink != 0 {
					_ = os.Remove(seatSteamCompat)
				}
				_ = os.MkdirAll(seatSteamCompat, 0755)
				args = append(args, "--ro-bind-try", realSteamCompat, seatSteamCompat)
			}
		}
	}

	// 4. Read-only Game Directory (if outside realHome or specified explicitly)
	canonicalGameDir := util.CanonicalPath(opts.GameDir)
	if canonicalGameDir != "" && util.FileExists(canonicalGameDir) {
		args = append(args, "--ro-bind", canonicalGameDir, canonicalGameDir)
	}

	// 5. Seat Profile Directory (read-write overlay for saves, configs, logs)
	if opts.ProfileDir != "" {
		_ = os.MkdirAll(opts.ProfileDir, 0755)
		args = append(args, "--bind", opts.ProfileDir, opts.ProfileDir)
	}

	// 6. Wine Prefix & Central User Data (read-write)
	if opts.WinePrefix != "" {
		if err := os.MkdirAll(opts.WinePrefix, 0755); err == nil {
			args = append(args, "--bind", opts.WinePrefix, opts.WinePrefix)
		}

		// Check if prefix has a symlinked steamuser profile (e.g. to central Syncthing user data)
		// and ensure that central target directory is also bound read-write so Proton/Wine
		// profile migrations can write without read-only errors.
		steamuserSymlink := filepath.Join(opts.WinePrefix, "drive_c", "users", "steamuser")
		if fi, err := os.Lstat(steamuserSymlink); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			if target, err := os.Readlink(steamuserSymlink); err == nil && target != "" {
				if !filepath.IsAbs(target) {
					target = filepath.Join(filepath.Dir(steamuserSymlink), target)
				}
				if err := os.MkdirAll(target, 0755); err == nil {
					args = append(args, "--bind", target, target)
				}
			}
		}
	}

	// 7. Isolated GSE Saves overlay
	// Overlay the profile's GSE Saves directory onto realHome so paths resolving to realHome hit isolated saves.
	if !isP1 && realHome != "" {
		gseProfileDir := filepath.Join(homeDir, ".local", "share", "GSE Saves")
		gseRealDir := filepath.Join(realHome, ".local", "share", "GSE Saves")
		_ = os.MkdirAll(gseRealDir, 0755)
		_ = os.MkdirAll(gseProfileDir, 0755)
		args = append(args, "--bind", gseProfileDir, gseRealDir)
	}

	// 8. Environment variables
	if isP1 {
		args = append(args,
			"--setenv", "HOME", realHome,
			"--setenv", "XDG_RUNTIME_DIR", userRunDir,
		)
		if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
			args = append(args, "--setenv", "XDG_DATA_HOME", xdgData)
		}
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			args = append(args, "--setenv", "XDG_CONFIG_HOME", xdgConfig)
		}
		if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
			args = append(args, "--setenv", "XDG_CACHE_HOME", xdgCache)
		}
	} else {
		args = append(args,
			"--setenv", "HOME", homeDir,
			"--setenv", "XDG_DATA_HOME", filepath.Join(homeDir, ".local", "share"),
			"--setenv", "XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"),
			"--setenv", "XDG_CACHE_HOME", filepath.Join(homeDir, ".cache"),
			"--setenv", "XDG_RUNTIME_DIR", userRunDir,
		)
	}

	if realHome != "" {
		realSteamCompat := filepath.Join(realHome, ".local", "share", "Steam", "compatibilitytools.d")
		if util.FileExists(realSteamCompat) {
			args = append(args, "--setenv", "STEAM_EXTRA_COMPAT_TOOLS_PATHS", realSteamCompat)
		}
	}

	if dbusBus := os.Getenv("DBUS_SESSION_BUS_ADDRESS"); dbusBus != "" {
		args = append(args, "--setenv", "DBUS_SESSION_BUS_ADDRESS", dbusBus)
	}

	cursorTheme := os.Getenv("XCURSOR_THEME")
	if cursorTheme == "" {
		cursorTheme = "default"
	}
	cursorSize := os.Getenv("XCURSOR_SIZE")
	if cursorSize == "" {
		cursorSize = "24"
	}
	cursorPath := os.Getenv("XCURSOR_PATH")
	if cursorPath == "" {
		cursorPath = "/usr/share/icons:/usr/share/pixmaps:~/.local/share/icons:~/.icons"
	}
	args = append(args,
		"--setenv", "XCURSOR_THEME", cursorTheme,
		"--setenv", "XCURSOR_SIZE", cursorSize,
		"--setenv", "XCURSOR_PATH", cursorPath,
	)

	if opts.AudioSink != "" {
		args = append(args, "--setenv", "PULSE_SINK", opts.AudioSink)
	}
	if opts.WaylandDisplay != "" {
		args = append(args, "--setenv", "WAYLAND_DISPLAY", opts.WaylandDisplay)
	}
	if opts.DisplayEnv != "" {
		args = append(args, "--setenv", "DISPLAY", opts.DisplayEnv)
	}

	for k, v := range opts.CustomEnv {
		if opts.WinePrefix != "" && (k == "WINEPREFIX" || k == "STEAM_COMPAT_DATA_PATH") {
			continue
		}
		args = append(args, "--setenv", k, v)
	}

	if opts.WinePrefix != "" {
		args = append(args,
			"--setenv", "WINEPREFIX", opts.WinePrefix,
			"--setenv", "STEAM_COMPAT_DATA_PATH", opts.WinePrefix,
		)
	}

	// 7. Command terminator
	args = append(args, "--")
	args = append(args, opts.Command...)

	return args, nil
}
