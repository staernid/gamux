package util

import "testing"

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Stationeers", "Stationeers"},
		{"Stationeers: Space & Survival", "Stationeers - Space & Survival"},
		{"Game/Title\\Test?", "Game - Title - Test -"},
		{"  ", "Game"},
	}

	for _, tt := range tests {
		res := SanitizeFilename(tt.input)
		if res != tt.expected {
			t.Errorf("SanitizeFilename(%q) = %q, expected %q", tt.input, res, tt.expected)
		}
	}
}

func TestSanitizeInstallDir(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"The Witcher® 3: Wild Hunt", "The Witcher 3 Wild Hunt"},
		{"DARK SOULS™ III", "DARK SOULS III"},
		{"Cyberpunk 2077", "Cyberpunk 2077"},
		{"Stationeers: Space & Survival", "Stationeers Space & Survival"},
		{"Game/Title\\Test?", "GameTitleTest"},
		{"   ", "Game"},
		{"Game Name...", "Game Name"},
	}

	for _, tt := range tests {
		res := SanitizeInstallDir(tt.input)
		if res != tt.expected {
			t.Errorf("SanitizeInstallDir(%q) = %q, expected %q", tt.input, res, tt.expected)
		}
	}
}

func TestResolveOriginalBackupPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		isBackup bool
	}{
		{
			input:    "/games/Cyberpunk.2077/bin/x64/Cyberpunk2077.exe.20260901-155237.ORIGINAL",
			expected: "/games/Cyberpunk.2077/bin/x64/Cyberpunk2077.exe",
			isBackup: true,
		},
		{
			input:    "/games/Anno.2070/main.exe.ORIGINAL",
			expected: "/games/Anno.2070/main.exe",
			isBackup: true,
		},
		{
			input:    "/games/20xx/game.exe",
			expected: "/games/20xx/game.exe",
			isBackup: false,
		},
	}

	for _, tt := range tests {
		got, isBak := ResolveOriginalBackupPath(tt.input)
		if isBak != tt.isBackup {
			t.Errorf("ResolveOriginalBackupPath(%q) isBackup = %v, want %v", tt.input, isBak, tt.isBackup)
		}
		if got != tt.expected {
			t.Errorf("ResolveOriginalBackupPath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
