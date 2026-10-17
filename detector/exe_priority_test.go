package detector

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/staernid/gamux/domain"
)

func TestChooseDefaultExe_PriorityTiers(t *testing.T) {
	manifest := []domain.LaunchCandidate{{Name: "Play", Executable: "bin/real.exe", Arguments: "-fullscreen"}}
	heuristic := []domain.LaunchCandidate{{Name: "guess", Executable: "guess.exe", Arguments: "-g"}}

	tests := []struct {
		name      string
		manifest  []domain.LaunchCandidate
		ini       string
		heuristic []domain.LaunchCandidate
		wantExe   string
		wantArgs  string
	}{
		{"manifest beats ini and heuristic", manifest, "ini.exe", heuristic, "bin/real.exe", "-fullscreen"},
		{"ini beats heuristic", nil, "ini.exe", heuristic, "ini.exe", ""},
		{"heuristic last resort", nil, "", heuristic, "guess.exe", "-g"},
		{"loader in manifest is skipped", []domain.LaunchCandidate{{Executable: "steamclient_loader_x64.exe"}, {Executable: "game.exe"}}, "", nil, "game.exe", ""},
		{"loader ini target ignored", nil, "ColdClientLoader.exe", heuristic, "guess.exe", "-g"},
		{"only loaders yields nothing", []domain.LaunchCandidate{{Executable: "ColdClientLoader.exe"}}, "", []domain.LaunchCandidate{{Executable: "steamclient_loader.exe"}}, "", ""},
		{"nothing yields nothing", nil, "", nil, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exe, args := chooseDefaultExe(tt.manifest, tt.ini, tt.heuristic)
			if exe != tt.wantExe || args != tt.wantArgs {
				t.Errorf("got (%q, %q), want (%q, %q)", exe, args, tt.wantExe, tt.wantArgs)
			}
		})
	}
}

func TestScoreExecutable_DemotesLoaderWrappers(t *testing.T) {
	dir := "/games/Zed"
	game := ScoreExecutable(dir+"/Zed.exe", dir, "Zed")
	for _, loader := range []string{"steamclient_loader_x64.exe", "ColdClientLoader.exe"} {
		if got := ScoreExecutable(dir+"/"+loader, dir, "Zed"); got >= game || got > -1000 {
			t.Errorf("%s scored %d, want far below real game (%d)", loader, got, game)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestDetect_ManifestDeclaredExeOutranksIniAndHeuristics(t *testing.T) {
	gameDir := filepath.Join(t.TempDir(), "Zed")
	writeFile(t, filepath.Join(gameDir, "bin", "real_game.exe"), "exe")
	writeFile(t, filepath.Join(gameDir, "Zed.exe"), "heuristic favourite")
	writeFile(t, filepath.Join(gameDir, "ini_target.exe"), "exe")
	writeFile(t, filepath.Join(gameDir, "ColdClientLoader.ini"), "[SteamClient]\nExe = ini_target.exe\n")
	writeFile(t, filepath.Join(gameDir, "[Manifests]", "appmanifest_424242.acf"), GenerateACFContent(&ACFData{
		AppID: "424242", Name: "Zed", InstallDir: "Zed",
		LaunchOptions: []LaunchCandidate{{Name: "Play", Executable: "bin/real_game.exe", Arguments: "-novid"}},
	}))

	info, err := Detect(context.Background(), gameDir)
	if err != nil {
		t.Fatal(err)
	}
	if info.ExePath != "bin/real_game.exe" {
		t.Errorf("ExePath = %q, want manifest-declared bin/real_game.exe", info.ExePath)
	}
	if info.ExeArgs != "-novid" {
		t.Errorf("ExeArgs = %q, want -novid", info.ExeArgs)
	}
}

func TestDetect_IniDeclaredExeOutranksHeuristicsWithoutManifest(t *testing.T) {
	gameDir := filepath.Join(t.TempDir(), "Zed")
	writeFile(t, filepath.Join(gameDir, "Zed.exe"), "heuristic favourite")
	writeFile(t, filepath.Join(gameDir, "ini_target.exe"), "exe")
	writeFile(t, filepath.Join(gameDir, "ColdClientLoader.ini"), "[SteamClient]\nExe = ini_target.exe\n")

	info, err := Detect(context.Background(), gameDir)
	if err != nil {
		t.Fatal(err)
	}
	if info.ExePath != "ini_target.exe" {
		t.Errorf("ExePath = %q, want ini_target.exe", info.ExePath)
	}
}

func TestDetect_LoaderBinaryNeverBecomesExePath(t *testing.T) {
	gameDir := filepath.Join(t.TempDir(), "Zed")
	writeFile(t, filepath.Join(gameDir, "steamclient_loader_x64.exe"), "loader")
	writeFile(t, filepath.Join(gameDir, "ColdClientLoader.exe"), "loader")

	info, err := Detect(context.Background(), gameDir)
	if err != nil {
		t.Fatal(err)
	}
	if info.ExePath != "" {
		t.Errorf("ExePath = %q, want empty (loaders are not games)", info.ExePath)
	}
}

func TestDetect_UserPickedExeSurvivesRescans(t *testing.T) {
	gameDir := filepath.Join(t.TempDir(), "Zed")
	writeFile(t, filepath.Join(gameDir, "Zed.exe"), "heuristic favourite")
	writeFile(t, filepath.Join(gameDir, "tools", "alt.exe"), "user's choice")
	writeFile(t, filepath.Join(gameDir, ".gamux", "launch_profile.json"),
		`{"game_dir": "`+gameDir+`", "name": "Zed", "exe_path": "tools/alt.exe", "exe_args": "-alt"}`)

	for scan := 1; scan <= 2; scan++ { // second pass exercises the warm cache
		info, err := Detect(context.Background(), gameDir)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.EffectiveExePath(); got != "tools/alt.exe" {
			t.Errorf("scan %d: EffectiveExePath = %q, want the user's pick", scan, got)
		}
		if got := info.EffectiveExeArgs(); got != "-alt" {
			t.Errorf("scan %d: EffectiveExeArgs = %q, want -alt", scan, got)
		}
	}
}
