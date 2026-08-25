package gbe

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// interfaceRegex matches official Steamworks interface versions (e.g. SteamClient021, STEAMAPPS_INTERFACE_VERSION008).
var interfaceRegex = regexp.MustCompile(`\b(STEAM[A-Za-z0-9_]*?\d{3}|Steam[A-Za-z0-9_]*?\d{3})\b`)

// ExtractSteamInterfaces scans a binary or DLL byte payload and extracts sorted, deduplicated Steamworks interface version strings.
func ExtractSteamInterfaces(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read binary for interface extraction: %w", err)
	}

	matches := interfaceRegex.FindAll(data, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	seen := make(map[string]bool)
	var interfaces []string

	for _, m := range matches {
		s := string(m)
		if len(s) < 10 || len(s) > 64 {
			continue
		}
		if !seen[s] {
			seen[s] = true
			interfaces = append(interfaces, s)
		}
	}

	sort.Strings(interfaces)
	return interfaces, nil
}

// GenerateAndSaveInterfaces scans the source Steam API binary, generates steam_interfaces.txt,
// and saves it to the specified target directories (e.g. steam_settings/ and alongside the DLL).
func GenerateAndSaveInterfaces(sourceDllPath string, targetDirs ...string) error {
	interfaces, err := ExtractSteamInterfaces(sourceDllPath)
	if err != nil {
		return err
	}

	if len(interfaces) == 0 {
		slog.Debug("No Steamworks interface strings found in binary", "path", sourceDllPath)
		return nil
	}

	content := strings.Join(interfaces, "\n") + "\n"

	for _, dir := range targetDirs {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			slog.Warn("Failed to create directory for steam_interfaces.txt", "dir", dir, "error", err)
			continue
		}

		outPath := filepath.Join(dir, "steam_interfaces.txt")
		if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
			slog.Warn("Failed to write steam_interfaces.txt", "path", outPath, "error", err)
			continue
		}
		slog.Info("Successfully generated steam_interfaces.txt in steam_settings", "path", outPath, "interfaces", len(interfaces))
	}

	return nil
}
