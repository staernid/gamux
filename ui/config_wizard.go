package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/staernid/gamux/config"
)

// RenderConfigSummary displays a formatted dashboard of all configuration fields.
func RenderConfigSummary(cfg *config.Config, configPath string) {
	defaults := config.DefaultConfig()

	fmt.Printf("\n%s\n", bold(cyan("⚙️  gamux Configuration Dashboard")))
	fmt.Printf("%s\n", cyan("========================================================================"))
	if configPath != "" {
		fmt.Printf("  Config File: %s\n", yellow(configPath))
	} else {
		fmt.Printf("  Config File: %s\n", yellow("(using built-in defaults)"))
	}
	fmt.Printf("%s\n\n", cyan("------------------------------------------------------------------------"))

	renderSectionHeader("📁 Storage & Directory Paths")
	renderConfigRow("gbe_dir", cfg.GbeDir, defaults.GbeDir)
	renderConfigRow("steamless_dir", cfg.SteamlessDir, defaults.SteamlessDir)
	renderConfigRow("steam_userdata", cfg.SteamUserdata, defaults.SteamUserdata)

	renderSectionHeader("🔑 API Keys & Authentication")
	renderConfigRow("steam_web_api_key", maskKey(cfg.SteamWebAPIKey), "(not set)")
	renderConfigRow("hubcap_api_key", maskKey(cfg.HubcapAPIKey), "(not set)")
	renderConfigRow("steamgriddb_api_key", maskKey(cfg.SteamGridDBAPIKey), "(not set)")
	renderConfigRow("rawg_api_key", maskKey(cfg.RAWGAPIKey), "(not set)")

	renderSectionHeader("⚡ Operational Settings")
	renderConfigRow("operating_mode", fallback(cfg.OperatingMode, defaults.OperatingMode), defaults.OperatingMode)
	renderConfigRow("default_runner", fallback(cfg.DefaultRunner, defaults.DefaultRunner), defaults.DefaultRunner)
	renderConfigRow("steamless", fmt.Sprintf("%t", cfg.Steamless), fmt.Sprintf("%t", defaults.Steamless))
	renderConfigRow("achievements", fmt.Sprintf("%t", cfg.Achievements), fmt.Sprintf("%t", defaults.Achievements))
	renderConfigRow("normalize", fmt.Sprintf("%t", cfg.Normalize), fmt.Sprintf("%t", defaults.Normalize))
	renderConfigRow("mangohud", fmt.Sprintf("%t", cfg.MangoHud), fmt.Sprintf("%t", defaults.MangoHud))
	renderConfigRow("wine_prefix", fallback(cfg.WinePrefix, "(auto)"), "(auto)")

	renderSectionHeader("🔔 Launch Notifications")
	renderConfigRow("enable_launch_notify", fmt.Sprintf("%t", cfg.EnableLaunchNotify), fmt.Sprintf("%t", defaults.EnableLaunchNotify))

	fmt.Printf("%s\n\n", cyan("========================================================================"))
}

func renderSectionHeader(title string) {
	fmt.Printf("  %s\n", bold(title))
}

func renderConfigRow(key, current, defaultVal string) {
	match := ""
	if current == defaultVal || (current == "" && defaultVal == "(not set)") {
		match = dim("(default)")
	} else {
		match = green("(custom)")
	}
	fmt.Printf("    • %-22s : %-30s %s\n", key, yellow(current), match)
}

func maskKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) <= 6 {
		return "******"
	}
	return key[:3] + "..." + key[len(key)-3:]
}

func fallback(val, defaultStr string) string {
	if val == "" {
		return defaultStr
	}
	return val
}

// RunConfigWizard guides the user interactively through updating their configuration.
func RunConfigWizard(cfg *config.Config, configPath string) error {
	defaults := config.DefaultConfig()
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\n%s\n", bold(cyan("========================================================================")))
	fmt.Printf("%s\n", bold(cyan("  🧙  gamux Guided Configuration Wizard")))
	fmt.Printf("%s\n", cyan("========================================================================"))
	fmt.Printf("  Config File Path: %s\n", yellow(configPath))
	fmt.Printf("  Press %s to keep the current value. Type %s or %s to clear/unset.\n", bold("Enter"), bold("'-'"), bold("'none'"))
	fmt.Printf("%s\n\n", cyan("------------------------------------------------------------------------"))

	// Section 1: Storage & Directories
	fmt.Printf("%s\n", bold("📁 CATEGORY 1: Storage & Directory Paths"))
	cfg.GbeDir = promptString(reader, "Goldberg Emulator Assets Directory [gbe_dir]", cfg.GbeDir, defaults.GbeDir)
	cfg.SteamlessDir = promptString(reader, "Steamless Binaries Directory [steamless_dir]", cfg.SteamlessDir, defaults.SteamlessDir)
	cfg.SteamUserdata = promptString(reader, "Steam User Data Directory [steam_userdata]", cfg.SteamUserdata, defaults.SteamUserdata)

	// Section 2: API Keys
	fmt.Printf("\n%s\n", bold("🔑 CATEGORY 2: API Keys & Remote Services"))
	cfg.HubcapAPIKey = promptString(reader, "Hubcap Manifest API Key [hubcap_api_key]", cfg.HubcapAPIKey, "(not set)")
	cfg.SteamWebAPIKey = promptString(reader, "Steam Web API Key [steam_web_api_key]", cfg.SteamWebAPIKey, "(not set)")
	cfg.SteamGridDBAPIKey = promptString(reader, "SteamGridDB Artwork API Key [steamgriddb_api_key]", cfg.SteamGridDBAPIKey, "(not set)")
	cfg.RAWGAPIKey = promptString(reader, "RAWG Metadata API Key [rawg_api_key]", cfg.RAWGAPIKey, "(not set)")

	// Section 3: Operational Settings
	fmt.Printf("\n%s\n", bold("⚡ CATEGORY 3: Operational Settings"))
	cfg.OperatingMode = promptString(reader, "Primary Operating Mode (loader, portable, standalone, native, none) [operating_mode]", cfg.OperatingMode, defaults.OperatingMode)
	cfg.DefaultRunner = promptString(reader, "Default Standalone Runner (umu, wine, proton) [default_runner]", cfg.DefaultRunner, defaults.DefaultRunner)
	cfg.Steamless = promptBool(reader, "Enable automatic Steamless DRM unpacking? [steamless]", cfg.Steamless)
	cfg.Achievements = promptBool(reader, "Fetch achievement schema & icon assets? [achievements]", cfg.Achievements)
	cfg.Normalize = promptBool(reader, "Normalize directory names to official 1:1 Steam install dir? [normalize]", cfg.Normalize)
	cfg.MangoHud = promptBool(reader, "Enable MangoHud performance overlay by default? [mangohud]", cfg.MangoHud)
	cfg.WinePrefix = promptString(reader, "Wine Prefix path [wine_prefix]", cfg.WinePrefix, "(auto)")

	// Section 4: Notifications
	fmt.Printf("\n%s\n", bold("🔔 CATEGORY 4: Launch Notifications"))
	cfg.EnableLaunchNotify = promptBool(reader, "Enable pre-launch game update & issue alerts? [enable_launch_notify]", cfg.EnableLaunchNotify)

	// Save
	if err := config.SaveConfig(cfg, configPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("\n%s\n", cyan("========================================================================"))
	fmt.Printf("  %s %s\n", green("✨ Configuration saved successfully to"), yellow(configPath))
	fmt.Printf("%s\n\n", cyan("========================================================================"))

	return nil
}

func promptString(reader *bufio.Reader, label, currentVal, defaultVal string) string {
	displayCurrent := currentVal
	if displayCurrent == "" {
		displayCurrent = "(not set)"
	}
	fmt.Printf("  • %s\n", label)
	fmt.Printf("    Current: %s | Default: %s\n", yellow(displayCurrent), dim(defaultVal))
	fmt.Printf("    New value (Enter to keep, '-' or 'none' to clear): ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return currentVal
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return currentVal
	}
	if input == "-" || strings.EqualFold(input, "none") || strings.EqualFold(input, "clear") || strings.EqualFold(input, "unset") {
		return ""
	}
	return input
}

func promptBool(reader *bufio.Reader, label string, currentVal bool) bool {
	defaultHint := "[Y/n]"
	if !currentVal {
		defaultHint = "[y/N]"
	}
	fmt.Printf("  • %s %s: ", label, yellow(defaultHint))

	input, err := reader.ReadString('\n')
	if err != nil {
		return currentVal
	}
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return currentVal
	}
	if input == "y" || input == "yes" || input == "true" || input == "1" {
		return true
	}
	if input == "n" || input == "no" || input == "false" || input == "0" {
		return false
	}
	return currentVal
}
