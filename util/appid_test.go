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
