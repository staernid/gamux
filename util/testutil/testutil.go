package testutil

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// SilenceLogging silences slog output by routing it to io.Discard during unit tests.
// If the TEST_VERBOSE_LOGS environment variable is set to "1" or "true", logs are preserved.
func SilenceLogging() func() {
	if os.Getenv("TEST_VERBOSE_LOGS") == "1" || os.Getenv("TEST_VERBOSE_LOGS") == "true" {
		return func() {}
	}

	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	return func() {
		slog.SetDefault(oldLogger)
	}
}

// SetupTestEnvironment isolates the test environment from the user's home directories
// by redirecting XDG_CONFIG_HOME, XDG_DATA_HOME, XDG_CACHE_HOME, and XDG_RUNTIME_DIR to a temporary directory.
func SetupTestEnvironment() (cleanup func()) {
	tmpDir, err := os.MkdirTemp("", "gamux-test-xdg-*")
	if err != nil {
		return func() {}
	}

	origConfig := os.Getenv("XDG_CONFIG_HOME")
	origData := os.Getenv("XDG_DATA_HOME")
	origCache := os.Getenv("XDG_CACHE_HOME")
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")

	_ = os.MkdirAll(filepath.Join(tmpDir, "config"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpDir, "share"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpDir, "cache"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpDir, "run"), 0755)

	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	os.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "share"))
	os.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, "cache"))
	os.Setenv("XDG_RUNTIME_DIR", filepath.Join(tmpDir, "run"))

	return func() {
		if origConfig != "" {
			os.Setenv("XDG_CONFIG_HOME", origConfig)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
		if origData != "" {
			os.Setenv("XDG_DATA_HOME", origData)
		} else {
			os.Unsetenv("XDG_DATA_HOME")
		}
		if origCache != "" {
			os.Setenv("XDG_CACHE_HOME", origCache)
		} else {
			os.Unsetenv("XDG_CACHE_HOME")
		}
		if origRuntime != "" {
			os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		} else {
			os.Unsetenv("XDG_RUNTIME_DIR")
		}
		_ = os.RemoveAll(tmpDir)
	}
}

