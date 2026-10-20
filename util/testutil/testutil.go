package testutil

import (
	"bytes"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

func isLoopbackHost(host string) bool {
	h := host
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		h = parsedHost
	}
	h = strings.Trim(h, "[]")
	if strings.EqualFold(h, "localhost") {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

// SetupTestEnvironment isolates the test environment from the user's home directories
// by redirecting XDG_CONFIG_HOME, XDG_DATA_HOME, XDG_CACHE_HOME, and XDG_RUNTIME_DIR to a temporary directory,
// and blocks accidental external network calls unless TEST_LIVE_NETWORK=1 is set.
func SetupTestEnvironment() (cleanup func()) {
	tmpDir, err := os.MkdirTemp("", "gamux-test-xdg-*")
	if err != nil {
		return func() {}
	}

	origConfig := os.Getenv("XDG_CONFIG_HOME")
	origData := os.Getenv("XDG_DATA_HOME")
	origCache := os.Getenv("XDG_CACHE_HOME")
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origTransport := http.DefaultTransport

	if os.Getenv("TEST_LIVE_NETWORK") != "1" {
		http.DefaultTransport = RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL != nil && isLoopbackHost(req.URL.Host) {
				return origTransport.RoundTrip(req)
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("external network disabled in unit tests")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		})
	}

	_ = os.MkdirAll(filepath.Join(tmpDir, "config"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpDir, "share"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpDir, "cache"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpDir, "run"), 0755)

	// Populate minimal hermetic Goldberg Emulator layout in XDG_DATA_HOME so unit tests
	// do not depend on host installations or attempt GitHub release bootstrap downloads.
	for _, gbeBase := range []string{
		filepath.Join(tmpDir, "share", "gbe_fork"),
		filepath.Join(tmpDir, "share", "gamux", "tools", "gbe_fork"),
	} {
		gbeStubs := map[string][]string{
			filepath.Join(gbeBase, "linux_release", "experimental", "x64"): {"libsteam_api.so", "steamclient.so"},
			filepath.Join(gbeBase, "linux_release", "experimental", "x32"): {"libsteam_api.so", "steamclient.so"},
			filepath.Join(gbeBase, "win_release", "experimental", "x64"):   {"steam_api64.dll", "steamclient64.dll"},
			filepath.Join(gbeBase, "win_release", "experimental", "x32"):   {"steam_api.dll", "steamclient.dll"},
			filepath.Join(gbeBase, "win_release", "steamclient_experimental"): {
				"steamclient64.dll", "steamclient.dll", "GameOverlayRenderer64.dll", "GameOverlayRenderer.dll",
			},
		}
		for dir, files := range gbeStubs {
			_ = os.MkdirAll(dir, 0755)
			for _, f := range files {
				_ = os.WriteFile(filepath.Join(dir, f), []byte("stub_gbe_binary_"+f), 0755)
			}
		}
	}

	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	os.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "share"))
	os.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, "cache"))
	os.Setenv("XDG_RUNTIME_DIR", filepath.Join(tmpDir, "run"))

	return func() {
		http.DefaultTransport = origTransport
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
