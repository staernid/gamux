package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    `-arg1 -arg2 --data-dir="My Games/Saves" -name='Hero Quest'`,
			expected: []string{"-arg1", "-arg2", "--data-dir=My Games/Saves", "-name=Hero Quest"},
		},
		{
			input:    `  foo   bar   "baz qux"  `,
			expected: []string{"foo", "bar", "baz qux"},
		},
		{
			input:    `--eval="hello \"world\""`,
			expected: []string{`--eval=hello "world"`},
		},
		{
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		got := SplitArgs(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("SplitArgs(%q) len = %d (%v), want %d (%v)", tt.input, len(got), got, len(tt.expected), tt.expected)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("SplitArgs(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestParseColdClientLoaderExe(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Missing file returns empty string
	if got := ParseColdClientLoaderExe(tmpDir); got != "" {
		t.Errorf("expected empty string for missing file, got %q", got)
	}

	// 2. Relative exe in [Loader] section
	iniContent := `
[SmartSteamEmu]
AppId = 12345

[Loader]
# Target game executable
Exe = bin/game.exe
CommandLine = -novid
`
	if err := os.WriteFile(filepath.Join(tmpDir, "ColdClientLoader.ini"), []byte(iniContent), 0644); err != nil {
		t.Fatal(err)
	}

	expected := JoinPaths(tmpDir, "bin/game.exe")
	if got := ParseColdClientLoaderExe(tmpDir); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}

	// 3. Quoted path in [Launcher]
	iniContent2 := `
[Launcher]
Exe = "game_special.exe"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "ColdClientLoader.ini"), []byte(iniContent2), 0644); err != nil {
		t.Fatal(err)
	}
	expected2 := JoinPaths(tmpDir, "game_special.exe")
	if got := ParseColdClientLoaderExe(tmpDir); got != expected2 {
		t.Errorf("expected %q, got %q", expected2, got)
	}
}
