package lutris

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func mustTempDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("", "lutris-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

// ── Validation ──────────────────────────────────────────────────────

func TestGenerate_NoName(t *testing.T) {
	_, err := Generate(Config{GamePath: "/tmp/game.exe"})
	if err == nil || !strings.Contains(err.Error(), "Name") {
		t.Fatalf("expected Name error, got: %v", err)
	}
}

func TestGenerate_NoGamePath(t *testing.T) {
	_, err := Generate(Config{Name: "Foo"})
	if err == nil || !strings.Contains(err.Error(), "GamePath") {
		t.Fatalf("expected GamePath error, got: %v", err)
	}
}

func TestGenerate_BadRunner(t *testing.T) {
	_, err := Generate(Config{Name: "Foo", GamePath: "/tmp/game.exe", Runner: "macos"})
	if err == nil || !strings.Contains(err.Error(), "Runner") {
		t.Fatalf("expected Runner error, got: %v", err)
	}
}

// ── Minimal wine game ───────────────────────────────────────────────

func TestGenerate_MinimalWine(t *testing.T) {
	out, err := Generate(Config{
		Name:     "Balatro",
		GamePath: "/mnt/games/Balatro/Balatro.exe",
	})
	if err != nil {
		t.Fatal(err)
	}

	var d doc
	if err := yaml.Unmarshal(out, &d); err != nil {
		t.Fatal(err)
	}

	if d.Name != "Balatro" {
		t.Errorf("name = %q, want Balatro", d.Name)
	}
	if d.Game.Exe != "/mnt/games/Balatro/Balatro.exe" {
		t.Errorf("exe = %q", d.Game.Exe)
	}
	if d.Game.Arch != "win64" {
		t.Errorf("arch = %q, want win64", d.Game.Arch)
	}
	if d.Wine == nil {
		t.Fatal("wine section missing")
	}
	if d.Wine.Battleye || d.Wine.EAC || d.Wine.FSR || d.Wine.VKD3D || d.Wine.DXVKNVAPI {
		t.Errorf("wine toggles should all be false for minimal config")
	}
}

// ── Slug generation ─────────────────────────────────────────────────

func TestGenerate_AutoSlug(t *testing.T) {
	out, err := Generate(Config{
		Name:     "Castle Crashers",
		GamePath: "/mnt/games/CC/castle.exe",
	})
	if err != nil {
		t.Fatal(err)
	}

	var d doc
	if err := yaml.Unmarshal(out, &d); err != nil {
		t.Fatal(err)
	}

	if d.Slug != "castle-crashers" {
		t.Errorf("slug = %q, want castle-crashers", d.Slug)
	}
}

func TestGenerate_ExplicitSlug(t *testing.T) {
	out, err := Generate(Config{
		Name:     "Deus Ex: Game of the Year Edition",
		Slug:     "deus-ex-gog",
		GamePath: "/mnt/games/deus-ex/System/DeusEx.exe",
	})
	if err != nil {
		t.Fatal(err)
	}

	var d doc
	if err := yaml.Unmarshal(out, &d); err != nil {
		t.Fatal(err)
	}

	if d.Slug != "deus-ex-gog" {
		t.Errorf("slug = %q, want deus-ex-gog", d.Slug)
	}
}

// ── Full wine config ────────────────────────────────────────────────

func TestGenerate_FullWineGame(t *testing.T) {
	out, err := Generate(Config{
		Name:       "Deep Rock Galactic",
		Slug:       "deep-rock-galactic",
		GamePath:   "/mnt/games/DRG/FSD.exe",
		PrefixPath: "/mnt/games/DRG/prefix",
		Runner:     "wine",
		Arch:       "auto",
		Args:       "-dx12",
		Wine: &WineConfig{
			Battleye:  true,
			EAC:       true,
			FSR:       true,
			VKD3D:     true,
			DXVKNVAPI: true,
			Version:   "GE-Proton (Latest)",
		},
		System: &SystemConfig{
			MangoHud: true,
			GPU:      "card1",
			VkICD:    "/usr/share/vulkan/icd.d/radeon_icd.x86_64.json",
		},
		Env: map[string]string{
			"DXVK_ASYNC":        "1",
			"WINEDEBUG":         "-all",
			"WINEHAGS":          "1",
			"mesa_glthread":     "true",
			"__GL_THREADED_OPTIMIZATIONS": "1",
		},
		Version: "1.0",
		Year:    "2020",
	})
	if err != nil {
		t.Fatal(err)
	}

	var d doc
	if err := yaml.Unmarshal(out, &d); err != nil {
		t.Fatal(err)
	}

	if d.Game.Arch != "auto" {
		t.Errorf("arch = %q, want auto", d.Game.Arch)
	}
	if d.Game.Prefix != "/mnt/games/DRG/prefix" {
		t.Errorf("prefix = %q", d.Game.Prefix)
	}
	if d.Game.Args != "-dx12" {
		t.Errorf("args = %q, want -dx12", d.Game.Args)
	}

	if d.Wine == nil {
		t.Fatal("wine section missing")
	}
	if !d.Wine.Battleye {
		t.Error("wine.battleye should be true")
	}
	if !d.Wine.EAC {
		t.Error("wine.eac should be true")
	}
	if !d.Wine.FSR {
		t.Error("wine.fsr should be true")
	}
	if !d.Wine.VKD3D {
		t.Error("wine.vkd3d should be true")
	}
	if !d.Wine.DXVKNVAPI {
		t.Error("wine.dxvk_nvapi should be true")
	}
	if d.Wine.Version != "GE-Proton (Latest)" {
		t.Errorf("wine.version = %q", d.Wine.Version)
	}

	if d.System == nil {
		t.Fatal("system section missing")
	}
	if !d.System.MangoHud {
		t.Error("system.mangohud should be true")
	}
	if d.System.GPU != "card1" {
		t.Errorf("system.gpu = %q", d.System.GPU)
	}

	expectedEnv := map[string]string{
		"DXVK_ASYNC":        "1",
		"WINEDEBUG":         "-all",
	}
	for k, v := range expectedEnv {
		if d.System.Env[k] != v {
			t.Errorf("env[%s] = %q, want %q", k, d.System.Env[k], v)
		}
	}

	if d.Version != "1.0" {
		t.Errorf("version = %q", d.Version)
	}
	if d.Year != "2020" {
		t.Errorf("year = %q", d.Year)
	}
}

// ── Native Linux game ───────────────────────────────────────────────

func TestGenerate_LinuxGame(t *testing.T) {
	out, err := Generate(Config{
		Name:     "SuperTuxKart",
		Slug:     "supertuxkart",
		GamePath: "/usr/games/supertuxkart",
		Runner:   "linux",
		Args:     "--fullscreen",
		Env: map[string]string{
			"SDL_VIDEODRIVER": "wayland",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var d doc
	if err := yaml.Unmarshal(out, &d); err != nil {
		t.Fatal(err)
	}

	if d.Wine != nil {
		t.Error("wine section should be omitted for linux runner")
	}
	if d.Game.Arch != "" {
		t.Errorf("arch should be empty for linux runner, got %q", d.Game.Arch)
	}
	if d.Game.Prefix != "" {
		t.Errorf("prefix should be empty for linux runner, got %q", d.Game.Prefix)
	}
	if d.Game.Args != "--fullscreen" {
		t.Errorf("args = %q", d.Game.Args)
	}
	if d.System == nil {
		t.Fatal("system section missing")
	}
	if d.System.Env["SDL_VIDEODRIVER"] != "wayland" {
		t.Errorf("env SDL_VIDEODRIVER = %q", d.System.Env["SDL_VIDEODRIVER"])
	}
}

// ── Write ───────────────────────────────────────────────────────────

func TestWrite_WritesFile(t *testing.T) {
	dir := mustTempDir(t)

	err := Write(Config{
		Name:     "Biped",
		Slug:     "biped",
		GamePath: "/mnt/games/Biped/biped.exe",
	}, dir)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "biped.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var d doc
	if err := yaml.Unmarshal(data, &d); err != nil {
		t.Fatal(err)
	}
	if d.Name != "Biped" {
		t.Errorf("name = %q, want Biped", d.Name)
	}
}

func TestWrite_AutoSlugFile(t *testing.T) {
	dir := mustTempDir(t)

	err := Write(Config{
		Name:     "Castle Crashers",
		GamePath: "/mnt/games/CC/castle.exe",
	}, dir)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "castle-crashers.yml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("expected file %s to exist", path)
	}
}

// ── Round-trip: generate → unmarshal → re-marshal → equivalence ─────

func TestRoundTrip(t *testing.T) {
	cfg := Config{
		Name:       "Hades",
		Slug:       "hades",
		GamePath:   "/mnt/games/Hades/Hades.exe",
		PrefixPath: "/mnt/games/Hades/prefix",
		Runner:     "wine",
		Wine: &WineConfig{
			FSR:     true,
			Version: "GE-Proton (Latest)",
		},
		System: &SystemConfig{
			MangoHud: true,
		},
		Env: map[string]string{
			"PROTON_ENABLE_WAYLAND": "0",
		},
		Version: "Steam",
	}

	out1, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var d doc
	if err := yaml.Unmarshal(out1, &d); err != nil {
		t.Fatal(err)
	}

	out2, err := yaml.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}

	if string(out1) != string(out2) {
		t.Errorf("round-trip mismatch:\n--- generate ---\n%s\n--- re-marshal ---\n%s", out1, out2)
	}
}

func TestGenerateInstaller(t *testing.T) {
	out, err := GenerateInstaller(Config{
		Name:     "Blue Prince",
		GamePath: "/home/user/Games/Blue Prince/BLUE PRINCE.exe",
		Runner:   "wine",
		Env: map[string]string{
			"SteamClient64Dll": "/tmp/steamclient64.dll",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var inst installerDoc
	if err := yaml.Unmarshal(out, &inst); err != nil {
		t.Fatal(err)
	}

	if inst.Name != "Blue Prince" {
		t.Errorf("name = %q, want Blue Prince", inst.Name)
	}
	if inst.GameSlug != "blue-prince" {
		t.Errorf("game_slug = %q, want blue-prince", inst.GameSlug)
	}
	if inst.Slug != "blue-prince-installer" {
		t.Errorf("slug = %q, want blue-prince-installer", inst.Slug)
	}
	if inst.Script.Game.Exe != "/home/user/Games/Blue Prince/BLUE PRINCE.exe" {
		t.Errorf("script.game.exe = %q", inst.Script.Game.Exe)
	}
	if inst.Script.Game.WorkingDir != "/home/user/Games/Blue Prince" {
		t.Errorf("script.game.working_dir = %q", inst.Script.Game.WorkingDir)
	}
	if inst.Script.System == nil || inst.Script.System.Env["SteamClient64Dll"] != "/tmp/steamclient64.dll" {
		t.Errorf("env SteamClient64Dll missing or incorrect")
	}
}

func TestWriteInstaller(t *testing.T) {
	dir := mustTempDir(t)
	path := filepath.Join(dir, "lutris-installer.yml")

	err := WriteInstaller(Config{
		Name:     "Blue Prince",
		GamePath: "/home/user/Games/Blue Prince/BLUE PRINCE.exe",
	}, path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("expected installer file %s to exist", path)
	}
}

func TestGenerate_RunnerInYAML(t *testing.T) {
	out, err := Generate(Config{
		Name:     "Test Game",
		GamePath: "/tmp/game.exe",
		Runner:   "wine",
	})
	if err != nil {
		t.Fatal(err)
	}

	var d doc
	if err := yaml.Unmarshal(out, &d); err != nil {
		t.Fatal(err)
	}

	if d.Runner != "wine" {
		t.Errorf("runner = %q, want wine", d.Runner)
	}
}

func TestCreateXDGShortcuts(t *testing.T) {
	tmpHome := mustTempDir(t)
	t.Setenv("HOME", tmpHome)

	cfg := Config{
		Name:                  "Approximately Up",
		Slug:                  "approximately-up",
		CreateMenuShortcut:    true,
		CreateDesktopShortcut: true,
	}

	if err := CreateXDGShortcuts(cfg, "approximately-up"); err != nil {
		t.Fatalf("CreateXDGShortcuts failed: %v", err)
	}

	menuPath := filepath.Join(tmpHome, ".local", "share", "applications", "net.lutris.approximately-up.desktop")
	desktopPath := filepath.Join(tmpHome, "Desktop", "net.lutris.approximately-up.desktop")

	for _, path := range []string{menuPath, desktopPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read shortcut file %s: %v", path, err)
		}
		content := string(data)
		if !strings.Contains(content, "Name=Approximately Up") {
			t.Errorf("%s missing Name", path)
		}
		if !strings.Contains(content, "Exec=env LUTRIS_SKIP_INIT=1 lutris lutris:rungame/approximately-up") {
			t.Errorf("%s missing Exec line", path)
		}
	}
}

func TestGenerate_PreLaunchScript(t *testing.T) {
	out, err := Generate(Config{
		Name:     "Prelaunch Test Game",
		GamePath: "/tmp/game.exe",
		Runner:   "wine",
		System: &SystemConfig{
			PreLaunchScript: "/usr/local/bin/gamux",
			PreLaunchArg:    "notify-launch --path \"/tmp/game\"",
			PreLaunchWait:   true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var d doc
	if err := yaml.Unmarshal(out, &d); err != nil {
		t.Fatal(err)
	}

	if d.System == nil {
		t.Fatal("expected system section to be present")
	}
	if d.System.PreLaunchScript != "/usr/local/bin/gamux" {
		t.Errorf("PreLaunchScript = %q, want /usr/local/bin/gamux", d.System.PreLaunchScript)
	}
	if d.System.PreLaunchArg != "notify-launch --path \"/tmp/game\"" {
		t.Errorf("PreLaunchArg = %q", d.System.PreLaunchArg)
	}
	if !d.System.PreLaunchWait {
		t.Errorf("PreLaunchWait should be true")
	}
}

