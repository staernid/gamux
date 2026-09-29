package trash

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafetyCheck(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"empty path", "", true},
		{"dot path", ".", true},
		{"root path", "/", true},
		{"system bin", "/bin", true},
		{"system etc", "/etc", true},
		{"safe game path", "/home/user/Games/Cyberpunk2077", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := SafetyCheck(tc.path)
			if (err != nil) != tc.wantErr {
				t.Fatalf("SafetyCheck(%q) error = %v, wantErr = %v", tc.path, err, tc.wantErr)
			}
		})
	}
}

func TestTrashXDGHome(t *testing.T) {
	tempDir := t.TempDir()
	fakeDataHome := filepath.Join(tempDir, "data_home")
	t.Setenv("XDG_DATA_HOME", fakeDataHome)

	// Create test directory to trash
	targetDir := filepath.Join(tempDir, "MyTestGame")
	if err := os.MkdirAll(filepath.Join(targetDir, "bin"), 0755); err != nil {
		t.Fatalf("failed to create target game dir: %v", err)
	}
	testFile := filepath.Join(targetDir, "bin", "game.exe")
	if err := os.WriteFile(testFile, []byte("dummy binary"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run trashXDGHome
	if err := trashXDGHome(targetDir); err != nil {
		t.Fatalf("trashXDGHome failed: %v", err)
	}

	// Verify target no longer exists at original location
	if _, err := os.Lstat(targetDir); !os.IsNotExist(err) {
		t.Fatalf("target still exists at original path after trash: %v", err)
	}

	// Verify target exists in Trash/files
	trashedFile := filepath.Join(fakeDataHome, "Trash", "files", "MyTestGame", "bin", "game.exe")
	content, err := os.ReadFile(trashedFile)
	if err != nil {
		t.Fatalf("could not read trashed file: %v", err)
	}
	if string(content) != "dummy binary" {
		t.Fatalf("trashed file content mismatch: got %q, want %q", string(content), "dummy binary")
	}

	// Verify .trashinfo metadata
	infoFile := filepath.Join(fakeDataHome, "Trash", "info", "MyTestGame.trashinfo")
	infoContent, err := os.ReadFile(infoFile)
	if err != nil {
		t.Fatalf("could not read .trashinfo file: %v", err)
	}
	infoStr := string(infoContent)
	if !strings.Contains(infoStr, "[Trash Info]") {
		t.Errorf("missing [Trash Info] header in .trashinfo: %s", infoStr)
	}
	expectedEscaped := url.PathEscape(targetDir)
	if !strings.Contains(infoStr, "Path="+expectedEscaped) {
		t.Errorf("missing or incorrect Path in .trashinfo: %s", infoStr)
	}
	if !strings.Contains(infoStr, "DeletionDate=") {
		t.Errorf("missing DeletionDate in .trashinfo: %s", infoStr)
	}
}

func TestTrashCollisionHandling(t *testing.T) {
	tempDir := t.TempDir()
	fakeDataHome := filepath.Join(tempDir, "data_home")
	t.Setenv("XDG_DATA_HOME", fakeDataHome)

	// Create and trash first instance
	target1 := filepath.Join(tempDir, "CollisionGame")
	if err := os.WriteFile(target1, []byte("version 1"), 0644); err != nil {
		t.Fatalf("failed to create target1: %v", err)
	}
	if err := trashXDGHome(target1); err != nil {
		t.Fatalf("trash target1 failed: %v", err)
	}

	// Create and trash second instance with same name
	target2 := filepath.Join(tempDir, "CollisionGame")
	if err := os.WriteFile(target2, []byte("version 2"), 0644); err != nil {
		t.Fatalf("failed to create target2: %v", err)
	}
	if err := trashXDGHome(target2); err != nil {
		t.Fatalf("trash target2 failed: %v", err)
	}

	// Both should exist in Trash/files
	trashed1 := filepath.Join(fakeDataHome, "Trash", "files", "CollisionGame")
	trashed2 := filepath.Join(fakeDataHome, "Trash", "files", "CollisionGame.1")

	c1, err1 := os.ReadFile(trashed1)
	if err1 != nil || string(c1) != "version 1" {
		t.Fatalf("trashed1 error or mismatch: err=%v, content=%s", err1, string(c1))
	}

	c2, err2 := os.ReadFile(trashed2)
	if err2 != nil || string(c2) != "version 2" {
		t.Fatalf("trashed2 error or mismatch: err=%v, content=%s", err2, string(c2))
	}

	// Both .trashinfo files should exist
	info1 := filepath.Join(fakeDataHome, "Trash", "info", "CollisionGame.trashinfo")
	info2 := filepath.Join(fakeDataHome, "Trash", "info", "CollisionGame.1.trashinfo")

	if _, err := os.Stat(info1); err != nil {
		t.Errorf("info1 missing: %v", err)
	}
	if _, err := os.Stat(info2); err != nil {
		t.Errorf("info2 missing: %v", err)
	}
}

func TestTrashWithContext_NonExistent(t *testing.T) {
	err := TrashWithContext(context.Background(), "/non/existent/path/for/gamux/test")
	if err == nil {
		t.Fatalf("expected error for non-existent path, got nil")
	}
}
