package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/util"
)

// isValidExecutable checks if a file exists, is non-empty, and is not a corrupted ar archive (!<arch>).
func isValidExecutable(path string) bool {
	if !util.FileExists(path) {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 8)
	n, _ := f.Read(buf)
	if n >= 7 && strings.HasPrefix(string(buf[:n]), "!<arch>") {
		return false
	}
	return true
}

// FindUMUBinary searches PATH and standard runtime paths for the umu-run binary.
func FindUMUBinary() (string, error) {
	if bin, err := exec.LookPath("umu-run"); err == nil && isValidExecutable(bin) {
		return bin, nil
	}
	home, _ := os.UserHomeDir()
	var candidates []string
	if home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".local", "share", "umu", "umu-run"),
			filepath.Join(home, ".local", "share", "umu", "umu", "umu-run"),
			filepath.Join(home, ".local", "bin", "umu-run"),
			filepath.Join(home, ".cargo", "bin", "umu-run"),
		)
	}
	candidates = append(candidates,
		"/usr/local/bin/umu-run",
		"/usr/bin/umu-run",
		"/var/lib/flatpak/exports/bin/umu-run",
	)
	for _, c := range candidates {
		if isValidExecutable(c) {
			return c, nil
		}
	}
	return "", fmt.Errorf("umu-run binary not found in PATH or standard runtime directories (~/.local/share/umu/umu-run)")
}

// GetAvailableRunners inspects system PATH to detect available execution tools.
func GetAvailableRunners() []string {
	var runners []string
	if _, err := FindUMUBinary(); err == nil {
		runners = append(runners, RunnerUMU)
	}
	if _, err := exec.LookPath("wine"); err == nil {
		runners = append(runners, RunnerWine)
	}
	if _, err := exec.LookPath("proton"); err == nil {
		runners = append(runners, RunnerProton)
	}
	runners = append(runners, RunnerLinux)
	return runners
}

// detectBinaryPlatform inspects executable magic bytes to determine platform (ELF vs PE).
func detectBinaryPlatform(exePath string) string {
	if !util.FileExists(exePath) {
		return ""
	}
	f, err := os.Open(exePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	header := make([]byte, 4)
	if n, err := f.Read(header); err != nil || n < 2 {
		return ""
	}

	if len(header) >= 4 && string(header[:4]) == "\x7fELF" {
		return "linux"
	}
	if string(header[:2]) == "MZ" {
		return "win64"
	}
	return ""
}

// findCandidateExecutable searches gameDir for the most likely primary executable.
func findCandidateExecutable(gameDir string) string {
	if !util.FileExists(gameDir) {
		return ""
	}
	var candidates []string
	_ = filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		nameLower := strings.ToLower(d.Name())
		if strings.HasSuffix(nameLower, ".exe") || (d.Type().IsRegular() && !strings.Contains(d.Name(), ".")) {
			if strings.Contains(nameLower, "steamclient_loader") ||
				strings.Contains(nameLower, "coldclientloader") ||
				strings.Contains(nameLower, "unitycrashhandler") ||
				strings.Contains(nameLower, "crashreport") ||
				strings.Contains(nameLower, "unins000") ||
				strings.Contains(nameLower, "setup") ||
				strings.Contains(nameLower, "dxsetup") {
				return nil
			}
			candidates = append(candidates, path)
		}
		return nil
	})
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}
