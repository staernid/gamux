package gbe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractSteamInterfaces(t *testing.T) {
	tmpDir := t.TempDir()
	mockDll := filepath.Join(tmpDir, "steam_api64.dll")

	// Create binary payload with null-terminated and embedded interface strings
	payload := []byte("SOME_HEADER\x00SteamClient021\x00SteamUser023\x00STEAMAPPS_INTERFACE_VERSION008\x00STEAMUSERSTATS_INTERFACE_VERSION012\x00SteamInput006\x00SOME_FOOTER")
	if err := os.WriteFile(mockDll, payload, 0644); err != nil {
		t.Fatal(err)
	}

	interfaces, err := ExtractSteamInterfaces(mockDll)
	if err != nil {
		t.Fatalf("ExtractSteamInterfaces failed: %v", err)
	}

	expected := []string{
		"STEAMAPPS_INTERFACE_VERSION008",
		"STEAMUSERSTATS_INTERFACE_VERSION012",
		"SteamClient021",
		"SteamInput006",
		"SteamUser023",
	}

	if len(interfaces) != len(expected) {
		t.Fatalf("expected %d interfaces, got %d: %v", len(expected), len(interfaces), interfaces)
	}

	for i, exp := range expected {
		if interfaces[i] != exp {
			t.Errorf("interface[%d] expected %q, got %q", i, exp, interfaces[i])
		}
	}
}

func TestGenerateAndSaveInterfaces(t *testing.T) {
	tmpDir := t.TempDir()
	mockDll := filepath.Join(tmpDir, "steam_api64.dll")
	payload := []byte("Header\x00SteamClient020\x00SteamUser021\x00Footer")
	if err := os.WriteFile(mockDll, payload, 0644); err != nil {
		t.Fatal(err)
	}

	rootSettings := filepath.Join(tmpDir, "steam_settings")
	pluginSettings := filepath.Join(tmpDir, "Plugins", "steam_settings")

	err := GenerateAndSaveInterfaces(mockDll, rootSettings, pluginSettings)
	if err != nil {
		t.Fatalf("GenerateAndSaveInterfaces failed: %v", err)
	}

	for _, dir := range []string{rootSettings, pluginSettings} {
		outPath := filepath.Join(dir, "steam_interfaces.txt")
		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("failed to read generated %s: %v", outPath, err)
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) != 2 || lines[0] != "SteamClient020" || lines[1] != "SteamUser021" {
			t.Errorf("unexpected content in %s: %q", outPath, string(content))
		}
	}
}

func TestExtractSteamInterfaces_RealDLL(t *testing.T) {
	tmpDir := t.TempDir()
	mockDll := filepath.Join(tmpDir, "steam_api64.dll")

	// Comprehensive mock payload matching real Steamworks SDK strings
	payload := []byte(strings.Join([]string{
		"HEADER_BYTES\x00",
		"STEAMAPPLIST_INTERFACE_VERSION001\x00",
		"STEAMAPPS_INTERFACE_VERSION008\x00",
		"STEAMHTMLSURFACE_INTERFACE_VERSION_005\x00",
		"STEAMHTTP_INTERFACE_VERSION003\x00",
		"STEAMINVENTORY_INTERFACE_V003\x00",
		"STEAMMUSICREMOTE_INTERFACE_VERSION001\x00",
		"STEAMMUSIC_INTERFACE_VERSION001\x00",
		"STEAMPARENTALSETTINGS_INTERFACE_VERSION001\x00",
		"STEAMREMOTEPLAY_INTERFACE_VERSION001\x00",
		"STEAMREMOTESTORAGE_INTERFACE_VERSION016\x00",
		"STEAMSCREENSHOTS_INTERFACE_VERSION003\x00",
		"STEAMUGC_INTERFACE_VERSION017\x00",
		"STEAMUSERSTATS_INTERFACE_VERSION012\x00",
		"STEAMVIDEO_INTERFACE_V002\x00",
		"SteamClient021\x00",
		"SteamFriends017\x00",
		"SteamGameServer015\x00",
		"SteamInput006\x00",
		"SteamMatchMaking009\x00",
		"SteamNetworking006\x00",
		"SteamUser023\x00",
		"SteamUtils010\x00",
		"TRAILER_BYTES",
	}, ""))

	if err := os.WriteFile(mockDll, payload, 0644); err != nil {
		t.Fatal(err)
	}

	interfaces, err := ExtractSteamInterfaces(mockDll)
	if err != nil {
		t.Fatalf("failed to extract interfaces from fixture: %v", err)
	}
	if len(interfaces) != 22 {
		t.Fatalf("expected 22 interfaces, got %d: %v", len(interfaces), interfaces)
	}
}
