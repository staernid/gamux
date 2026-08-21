package testutil

import (
	"io"
	"log/slog"
	"os"
	"testing"
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

// InitTestLogger silences test logs for the duration of a test and automatically restores on cleanup.
func InitTestLogger(t testing.TB) {
	t.Helper()
	cleanup := SilenceLogging()
	t.Cleanup(cleanup)
}
