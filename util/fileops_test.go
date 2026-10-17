package util

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyFile(t *testing.T) {
	srcContent := "This is the source file content."
	tmpDir, err := os.MkdirTemp("", "testcopyfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "source.txt")
	destPath := filepath.Join(tmpDir, "destination.txt")

	// Create source file
	if err := os.WriteFile(srcPath, []byte(srcContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Copy file
	if err := CopyFile(srcPath, destPath); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination file exists and has correct content
	destContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(destContent) != srcContent {
		t.Errorf("Expected destination content %q, got %q", srcContent, string(destContent))
	}

	// Test copying to a non-existent directory (should fail)
	nonExistentDirPath := filepath.Join(tmpDir, "nonexistent", "destination.txt")
	err = CopyFile(srcPath, nonExistentDirPath)
	if err == nil {
		t.Fatalf("CopyFile was expected to fail for non-existent directory but succeeded")
	}
}

func TestBackupAndReplace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "testbackupreplace")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcContent := "new content"
	destContent := "original content"

	srcPath := filepath.Join(tmpDir, "source.txt")
	destPath := filepath.Join(tmpDir, "destination.txt")

	// Create source and destination files
	if err := os.WriteFile(srcPath, []byte(srcContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destPath, []byte(destContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Backup and replace
	if err := BackupAndReplace(srcPath, destPath); err != nil {
		t.Fatalf("BackupAndReplace failed: %v", err)
	}

	// Verify new destination content
	newDestContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read new destination file: %v", err)
	}
	if string(newDestContent) != srcContent {
		t.Errorf("Expected new destination content %q, got %q", srcContent, string(newDestContent))
	}

	// Verify backup file exists and has original content
	// The backup file name includes a timestamp, so we need to find it
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	backupFound := false
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "destination.txt.") && strings.HasSuffix(f.Name(), ".ORIGINAL") {
			backupPath := filepath.Join(tmpDir, f.Name())
			backupContent, err := os.ReadFile(backupPath)
			if err != nil {
				t.Fatalf("Failed to read backup file %s: %v", backupPath, err)
			}
			if string(backupContent) != destContent {
				t.Errorf("Expected backup content %q, got %q", destContent, string(backupContent))
			}
			backupFound = true
			break
		}
	}
	if !backupFound {
		t.Errorf("Backup file not found")
	}

	// Test case where destination file does not exist initially (should create it)
	os.Remove(destPath) // Remove the destination file
	if err := BackupAndReplace(srcPath, destPath); err != nil {
		t.Fatalf("BackupAndReplace failed when dest did not exist: %v", err)
	}
	newDestContentAfterNoBackup, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read new destination file after no backup: %v", err)
	}
	if string(newDestContentAfterNoBackup) != srcContent {
		t.Errorf("Expected new destination content %q, got %q", srcContent, string(newDestContentAfterNoBackup))
	}
}

func TestWriteFileAtomic(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "sub", "atomic.txt")

	data := []byte("atomic content payload 12345")
	if err := WriteFileAtomic(targetFile, data, 0644); err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	readData, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to read atomic file: %v", err)
	}
	if !bytes.Equal(readData, data) {
		t.Errorf("expected %q, got %q", string(data), string(readData))
	}

	// Verify overwrite
	newData := []byte("updated atomic content")
	if err := WriteFileAtomic(targetFile, newData, 0644); err != nil {
		t.Fatalf("WriteFileAtomic overwrite failed: %v", err)
	}

	readNew, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to read overwritten file: %v", err)
	}
	if !bytes.Equal(readNew, newData) {
		t.Errorf("expected %q, got %q", string(newData), string(readNew))
	}
}

func TestPreallocateFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_prealloc.bin")

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("open file failed: %v", err)
	}
	defer f.Close()

	// Preallocate 10 MB
	const size = 10 * 1024 * 1024
	if err := PreallocateFile(f, size); err != nil {
		t.Fatalf("PreallocateFile failed: %v", err)
	}

	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if fi.Size() != size {
		t.Errorf("expected size %d, got %d", size, fi.Size())
	}

	// Nil file pointer check
	if err := PreallocateFile(nil, 100); err == nil {
		t.Errorf("expected error on nil file pointer")
	}

	// Zero or negative size no-op
	if err := PreallocateFile(f, 0); err != nil {
		t.Errorf("expected no-op on zero size, got %v", err)
	}
}
