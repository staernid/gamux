package detector

import (
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/domain"
)

// loaderWrapperMarkers identify Goldberg/crack loader binaries that wrap, but are not, the game.
var loaderWrapperMarkers = []string{"steamclient_loader", "coldclientloader"}

// isLoaderWrapper reports whether path names a loader/wrapper binary.
func isLoaderWrapper(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	for _, marker := range loaderWrapperMarkers {
		if strings.Contains(base, marker) {
			return true
		}
	}
	return false
}

// chooseDefaultExe picks the default executable (and its arguments) by source authority:
//
//  1. executables declared by the store manifest (ACF launch options / Steam AppInfo),
//  2. the executable named by ColdClientLoader.ini,
//  3. filesystem heuristics, already ordered best-first by ScoreExecutable.
//
// Loader/wrapper binaries are never chosen. An empty result means no candidate qualified.
// A user's explicit choice lives in the launch profile and is applied on top of this
// default (Game.EffectiveExePath), so rescans never override it.
func chooseDefaultExe(manifestDeclared []domain.LaunchCandidate, iniDeclared string, heuristic []domain.LaunchCandidate) (exe, args string) {
	if c, ok := firstUsable(manifestDeclared); ok {
		return c.Executable, c.Arguments
	}
	if iniDeclared != "" && !isLoaderWrapper(iniDeclared) {
		return iniDeclared, ""
	}
	if c, ok := firstUsable(heuristic); ok {
		return c.Executable, c.Arguments
	}
	return "", ""
}

func firstUsable(cands []domain.LaunchCandidate) (domain.LaunchCandidate, bool) {
	for _, c := range cands {
		if c.Executable != "" && !isLoaderWrapper(c.Executable) {
			return c, true
		}
	}
	return domain.LaunchCandidate{}, false
}
