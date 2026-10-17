package util

import (
	"os"
	"testing"
)

func TestGetHash(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	content := "test content for hashing"
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	expectedHash := "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d" // SHA1 hash of "hello"

	// Re-create temp file with "hello" content
	tmpfile2, err := os.CreateTemp("", "testfile2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile2.Name()) // clean up
	if _, err := tmpfile2.WriteString("hello"); err != nil {
		t.Fatal(err)
	}
	tmpfile2.Close()

	hash2, err := GetHash(tmpfile2.Name())
	if err != nil {
		t.Fatalf("GetHash failed for 'hello': %v", err)
	}
	if hash2 != expectedHash {
		t.Errorf("Expected hash %q, got %q", expectedHash, hash2)
	}

	// Test non-existent file
	_, err = GetHash("nonexistentfile.txt")
	if err == nil {
		t.Fatalf("GetHash was expected to fail for non-existent file but succeeded")
	}
	if !os.IsNotExist(err) {
		t.Errorf("Expected 'file not exist' error, got %v", err)
	}
}
