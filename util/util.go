package util

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/staernid/gamux/ini"
)

var backupTimestampRegex = regexp.MustCompile(`\.\d{8}-\d{6}$`)

// ResolveOriginalBackupPath inspects a file path and determines the original target path if the file is a backup (.ORIGINAL).
// It reliably preserves directory paths containing dates or numbers (e.g. Anno.2070, Cyberpunk.2077).
func ResolveOriginalBackupPath(path string) (string, bool) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	if !strings.HasSuffix(base, ".ORIGINAL") {
		return path, false
	}
	trimmed := strings.TrimSuffix(base, ".ORIGINAL")
	trimmed = backupTimestampRegex.ReplaceAllString(trimmed, "")
	return filepath.Join(dir, trimmed), true
}

// SplitArgs parses a command-line argument string into individual arguments respecting quotes and backslash escaping.
func SplitArgs(argsStr string) []string {
	var args []string
	var cur strings.Builder
	var inQuote rune
	var escaped bool

	for _, r := range strings.TrimSpace(argsStr) {
		if escaped {
			cur.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' && inQuote != '\'' {
			escaped = true
			continue
		}

		if (r == '"' || r == '\'') && (inQuote == 0 || inQuote == r) {
			if inQuote == r {
				inQuote = 0
			} else {
				inQuote = r
			}
			continue
		}

		if unicode.IsSpace(r) && inQuote == 0 {
			if cur.Len() > 0 {
				args = append(args, cur.String())
				cur.Reset()
			}
			continue
		}

		cur.WriteRune(r)
	}

	if cur.Len() > 0 {
		args = append(args, cur.String())
	}

	return args
}

// runCmd executes a command and returns its output or an error.
func RunCmd(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("command failed: %s %v\nstdout: %s\nstderr: %s", name, args, stdout.String(), stderr.String())
	}
	return stdout.Bytes(), nil
}

// GetSHA1Hash returns the SHA1 hash of a file using buffered streaming to prevent memory ballooning on large files.
func GetSHA1Hash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha1.New()
	buf := make([]byte, 256*1024)
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// GetHash returns the SHA1 hash of a file.
func GetHash(filePath string) (string, error) {
	return GetSHA1Hash(filePath)
}

// BackupAndReplace backs up a file and replaces it.
func BackupAndReplace(src, dest string) error {
	timestamp := time.Now().Format("20060102-150405")
	// Check if the destination file exists before attempting to backup
	if _, err := os.Stat(dest); err == nil {
		backupPath := fmt.Sprintf("%s.%s.ORIGINAL", dest, timestamp)
		if err := os.Rename(dest, backupPath); err != nil {
			return fmt.Errorf("failed to backup %s: %w", dest, err)
		}
		slog.Info("Backed up file", "from", dest, "to", backupPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat destination file %s: %w", dest, err)
	}

	if err := CopyFile(src, dest); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", src, dest, err)
	}
	slog.Info("Replaced file", "src", src, "dest", dest)

	return nil
}

// CopyFile is a helper function to copy a file.
func CopyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// DiskSpaceInfo represents disk usage metrics for a filesystem volume.
type DiskSpaceInfo struct {
	Path       string `json:"path"`
	TotalBytes uint64 `json:"total_bytes"`
	FreeBytes  uint64 `json:"free_bytes"`
	UsedBytes  uint64 `json:"used_bytes"`
}

// GetDiskSpace inspects the filesystem volume containing the given directory path.
func GetDiskSpace(path string) (*DiskSpaceInfo, error) {
	cleanPath := ExpandPath(path)
	if cleanPath == "" {
		var err error
		cleanPath, err = os.UserHomeDir()
		if err != nil {
			cleanPath = "/"
		}
	}

	// If path doesn't exist, ascend to the nearest existing parent
	target := cleanPath
	for {
		if _, err := os.Stat(target); err == nil {
			break
		}
		parent := filepath.Dir(target)
		if parent == target {
			break
		}
		target = parent
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(target, &stat); err != nil {
		return nil, fmt.Errorf("statfs failed for %s: %w", target, err)
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := uint64(0)
	if total >= free {
		used = total - free
	}

	return &DiskSpaceInfo{
		Path:       target,
		TotalBytes: total,
		FreeBytes:  free,
		UsedBytes:  used,
	}, nil
}

// GetMusicDir returns the standard system-wide music directory (XDG_MUSIC_DIR or ~/Music).
func GetMusicDir() string {
	if xdgMusic := os.Getenv("XDG_MUSIC_DIR"); xdgMusic != "" {
		return ExpandPath(xdgMusic)
	}
	// Try reading ~/.config/user-dirs.dirs
	if home, err := os.UserHomeDir(); err == nil {
		userDirsPath := filepath.Join(home, ".config", "user-dirs.dirs")
		if data, err := os.ReadFile(userDirsPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "XDG_MUSIC_DIR=") {
					val := strings.TrimPrefix(line, "XDG_MUSIC_DIR=")
					val = strings.Trim(val, "\"")
					val = strings.ReplaceAll(val, "$HOME", home)
					if val != "" {
						return ExpandPath(val)
					}
				}
			}
		}
		return filepath.Join(home, "Music")
	}
	return "~/Music"
}

// WriteFileAtomic writes data to a temporary file in the same directory as filePath
// and renames it atomically to filePath to prevent file corruption or truncation.
func WriteFileAtomic(filePath string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, filepath.Base(filePath)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmpFile.Name()

	cleanUp := true
	defer func() {
		if cleanUp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp file %s: %w", tmpPath, err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync temp file %s: %w", tmpPath, err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file %s: %w", tmpPath, err)
	}

	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("chmod temp file %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpPath, filePath, err)
	}

	cleanUp = false
	return nil
}

// ParseColdClientLoaderExe parses ColdClientLoader.ini if present and returns the resolved executable target path.
func ParseColdClientLoaderExe(gameDir string) string {
	iniPath := filepath.Join(gameDir, "ColdClientLoader.ini")
	f, err := ini.LoadFile(iniPath)
	if err != nil {
		return ""
	}

	var rawExe string
	// Check standard launcher sections first: [Loader], [Launcher], [SmartSteamEmu], and default section ""
	for _, secName := range []string{"Loader", "Launcher", "SmartSteamEmu", ""} {
		if sec, ok := f.GetSection(secName); ok {
			if val, ok := sec.Get("Exe"); ok && strings.TrimSpace(val) != "" {
				rawExe = strings.TrimSpace(val)
				break
			}
		}
	}

	// Fallback: check all sections for an Exe key
	if rawExe == "" {
		for _, sec := range f.Sections() {
			if val, ok := sec.Get("Exe"); ok && strings.TrimSpace(val) != "" {
				rawExe = strings.TrimSpace(val)
				break
			}
		}
	}

	if rawExe == "" {
		return ""
	}

	rawExe = strings.Trim(rawExe, "\"'")
	if rawExe == "" {
		return ""
	}

	if !filepath.IsAbs(rawExe) {
		return JoinPaths(gameDir, rawExe)
	}
	return CanonicalPath(rawExe)
}

// PreallocateFile attempts to preallocate contiguous disk space for a file using fallocate (Linux) or truncate fallback.
// If disk space is exhausted (ENOSPC), it returns syscall.ENOSPC immediately.
func PreallocateFile(f *os.File, size int64) error {
	if f == nil {
		return fmt.Errorf("preallocate file: nil file pointer")
	}
	if size <= 0 {
		return nil
	}

	// Try fallocate on Linux (system call)
	err := syscall.Fallocate(int(f.Fd()), 0, 0, size)
	if err == nil {
		return nil
	}

	// If disk is full, return ENOSPC immediately without masking it
	if errors.Is(err, syscall.ENOSPC) {
		return fmt.Errorf("disk full (ENOSPC) preallocating %d bytes: %w", size, err)
	}

	// If fallocate is not supported on this filesystem (e.g. ENOSYS, EOPNOTSUPP, EINVAL on some mounts), fall back to Truncate
	return f.Truncate(size)
}
