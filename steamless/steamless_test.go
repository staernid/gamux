package steamless

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
	restore := testutil.SilenceLogging()
	code := m.Run()
	restore()
	os.Exit(code)
}


func TestEnsureBinary_EmbeddedExtraction(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	cfg.SteamlessDir = t.TempDir()

	binPath, err := EnsureBinary(ctx, cfg)
	if err != nil {
		t.Fatalf("EnsureBinary failed: %v", err)
	}

	if fi, err := os.Stat(binPath); err != nil || fi.Size() == 0 {
		t.Errorf("Extracted binary at %s is empty or missing", binPath)
	}
}

func TestUnpackExecutable_NonExistent(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()

	unpacked, err := UnpackExecutable(ctx, cfg, "/non/existent/path/game.exe", false)
	if err != nil {
		t.Fatalf("Expected no error for non-existent executable, got %v", err)
	}
	if unpacked {
		t.Errorf("Expected unpacked to be false for non-existent file")
	}
}

func TestUnpackExecutable_DryRun(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()

	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "game.exe")
	if err := os.WriteFile(exePath, []byte("fake exe content"), 0755); err != nil {
		t.Fatalf("failed to write fake exe: %v", err)
	}

	unpacked, err := UnpackExecutable(ctx, cfg, exePath, true)
	if err != nil {
		t.Fatalf("Expected no error in dryRun mode, got %v", err)
	}
	if unpacked {
		t.Errorf("Expected unpacked to be false in dryRun mode")
	}
}
