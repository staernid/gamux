package util

import "testing"

func TestParseAppID(t *testing.T) {
	tests := []struct {
		input   string
		wantVal uint32
		wantOK  bool
	}{
		{"12345", 12345, true},
		{" 730\n", 730, true},
		{"0", 0, false},
		{"", 0, false},
		{"custom", 0, false},
		{"Custom", 0, false},
		{"invalid", 0, false},
		{"-100", 0, false},
	}

	for _, tt := range tests {
		gotVal, gotOK := ParseAppID(tt.input)
		if gotVal != tt.wantVal || gotOK != tt.wantOK {
			t.Errorf("ParseAppID(%q) = (%d, %v), want (%d, %v)", tt.input, gotVal, gotOK, tt.wantVal, tt.wantOK)
		}
		if IsSteamAppID(tt.input) != tt.wantOK {
			t.Errorf("IsSteamAppID(%q) = %v, want %v", tt.input, IsSteamAppID(tt.input), tt.wantOK)
		}
	}
}

func TestIsPlaceholderTitle(t *testing.T) {
	tests := []struct {
		name  string
		appID string
		want  bool
	}{
		{"", "1458140", true},
		{"   ", "1458140", true},
		{"Unknown Game", "1458140", true},
		{"unknown", "1458140", true},
		{"1458140", "1458140", true},
		{"Steam App 1458140", "1458140", true},
		{"steam app 1458140", "1458140", true},
		{"Steam AppID 1458140", "1458140", true},
		{"SteamApp 1458140", "1458140", true},
		{"Game 1458140", "1458140", true},
		{"Steam App 9999", "", true},
		{"Pacific Drive", "1458140", false},
		{"Half-Life 2", "220", false},
		{"Cyberpunk 2077", "1091500", false},
		{"1984", "12345", false},
	}

	for _, tt := range tests {
		got := IsPlaceholderTitle(tt.name, tt.appID)
		if got != tt.want {
			t.Errorf("IsPlaceholderTitle(%q, %q) = %v, want %v", tt.name, tt.appID, got, tt.want)
		}
	}
}
