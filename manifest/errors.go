package manifest

import (
	"errors"
	"fmt"
)

var (
	// ErrDepotKeyNotFound indicates no decryption key is available for the given depot.
	ErrDepotKeyNotFound = errors.New("no depot decryption key found")

	// ErrManifestNotFound indicates the requested Steam depot manifest was not found.
	ErrManifestNotFound = errors.New("steam depot manifest not found")

	// ErrPayloadCorrupt indicates the manifest payload failed decryption or decompression.
	ErrPayloadCorrupt = errors.New("manifest payload is corrupt or invalid")
)

// PlatformMismatchError indicates that manifests were found and parsed,
// but zero depots matched the requested target platform.
type PlatformMismatchError struct {
	RequestedPlatform  string
	AvailablePlatforms []string // platforms actually present in the depots
	ManifestDir        string
}

func (e *PlatformMismatchError) Error() string {
	if len(e.AvailablePlatforms) > 0 {
		return fmt.Sprintf("no files available for %s — this game's depots contain only %s files",
			e.RequestedPlatform, joinPlatforms(e.AvailablePlatforms))
	}
	return fmt.Sprintf("no files available for %s — no platform-specific depots were found", e.RequestedPlatform)
}

// LanguageMismatchError indicates that depots exist for the platform but none
// matched the requested language (including the relaxed fallback).
type LanguageMismatchError struct {
	RequestedLanguage string
	RequestedPlatform string
	ManifestDir       string
}

func (e *LanguageMismatchError) Error() string {
	return fmt.Sprintf("no depots matched language %q (platform %s) — all available depots were filtered out by language",
		e.RequestedLanguage, e.RequestedPlatform)
}

// NoManifestFilesError indicates the [Manifests] directory exists but contains
// zero parseable .manifest files.
type NoManifestFilesError struct {
	ManifestDir string
	ParseErrors []string // per-file parse failures, if any
}

func (e *NoManifestFilesError) Error() string {
	if len(e.ParseErrors) > 0 {
		return fmt.Sprintf("no valid .manifest files in %s (%d file(s) failed to parse)", e.ManifestDir, len(e.ParseErrors))
	}
	return fmt.Sprintf("no .manifest files found in %s", e.ManifestDir)
}

// EncryptedDepotError indicates a manifest was parsed but its filenames are
// encrypted and no decryption key is available.
type EncryptedDepotError struct {
	DepotID uint32
}

func (e *EncryptedDepotError) Error() string {
	return fmt.Sprintf("depot %d has encrypted filenames and no decryption key was found — cannot download without a valid key", e.DepotID)
}

// joinPlatforms formats a slice of platform names for display.
func joinPlatforms(platforms []string) string {
	if len(platforms) == 0 {
		return "unknown"
	}
	if len(platforms) == 1 {
		return platforms[0]
	}
	return fmt.Sprintf("%s and %s", platforms[0], platforms[len(platforms)-1])
}
