package runner

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/staernid/gamux/detector"
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

// detectBinaryPlatform inspects executable magic bytes and PE headers to determine platform (ELF vs PE win32/win64).
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
		// Inspect PE header for 32-bit vs 64-bit architecture
		// Offset 0x3C in DOS header contains the offset to the PE signature
		if _, err := f.Seek(0x3C, io.SeekStart); err == nil {
			var peOffset uint32
			if err := binary.Read(f, binary.LittleEndian, &peOffset); err == nil && peOffset > 0 && peOffset < 10*1024*1024 {
				if _, err := f.Seek(int64(peOffset), io.SeekStart); err == nil {
					peSig := make([]byte, 4)
					if _, err := f.Read(peSig); err == nil && string(peSig) == "PE\x00\x00" {
						var machine uint16
						if err := binary.Read(f, binary.LittleEndian, &machine); err == nil {
							// 0x014c = IMAGE_FILE_MACHINE_I386 (32-bit x86)
							// 0x01c0 = IMAGE_FILE_MACHINE_ARMNT (32-bit ARM)
							if machine == 0x014c || machine == 0x01c0 {
								return "win32"
							}
							// 0x8664 = IMAGE_FILE_MACHINE_AMD64 (64-bit x86_64)
							// 0xaa64 = IMAGE_FILE_MACHINE_ARM64 (64-bit ARM64)
							if machine == 0x8664 || machine == 0xaa64 {
								return "win64"
							}
						}
					}
				}
			}
		}
		return "win64"
	}
	return ""
}

// findCandidateExecutable searches gameDir for the most likely primary executable using heuristic scoring and file permission validation.
func findCandidateExecutable(gameDir string, gameName ...string) string {
	if !util.FileExists(gameDir) {
		return ""
	}
	title := ""
	if len(gameName) > 0 {
		title = gameName[0]
	}

	var candidates []string
	_ = filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		nameLower := strings.ToLower(d.Name())
		if d.IsDir() {
			if path != gameDir && (nameLower == "drive_c" || nameLower == "dosdevices" || nameLower == "pfx" || nameLower == ".git" || nameLower == ".gamux" || nameLower == ".gamux_backup" || nameLower == "node_modules" || detector.IsRunnerPrefixDir(path)) {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.Contains(nameLower, "unitycrashhandler") ||
			strings.Contains(nameLower, "unins") ||
			strings.Contains(nameLower, "setup") ||
			strings.Contains(nameLower, "redist") ||
			strings.Contains(nameLower, "dxsetup") ||
			strings.Contains(nameLower, "vcredist") ||
			strings.Contains(nameLower, "crashdumper") ||
			strings.Contains(nameLower, "bssndrpt") ||
			strings.Contains(nameLower, "bugreport") ||
			strings.Contains(nameLower, "errorreport") ||
			strings.Contains(nameLower, "steamclient_loader") ||
			strings.Contains(nameLower, "coldclientloader") ||
			strings.Contains(nameLower, "steamless") ||
			strings.HasSuffix(nameLower, ".console.exe") {
			return nil
		}

		if strings.HasSuffix(nameLower, ".exe") || strings.HasSuffix(nameLower, ".sh") || strings.HasSuffix(nameLower, ".x86_64") || !strings.Contains(d.Name(), ".") {
			fileInfo, err := d.Info()
			if err == nil && (strings.HasSuffix(nameLower, ".exe") || fileInfo.Mode()&0111 != 0) {
				candidates = append(candidates, path)
			}
		}
		return nil
	})

	if len(candidates) == 0 {
		return ""
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		scoreI := detector.ScoreExecutable(candidates[i], gameDir, title)
		scoreJ := detector.ScoreExecutable(candidates[j], gameDir, title)
		if scoreI != scoreJ {
			return scoreI > scoreJ
		}
		return candidates[i] < candidates[j]
	})

	return candidates[0]
}
