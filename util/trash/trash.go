package trash

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/staernid/gamux/util"
)

// DefaultTimeout for trashing commands.
const DefaultTimeout = 30 * time.Second

// SafetyCheck ensures that accidental destructive operations are not performed on critical system locations.
func SafetyCheck(path string) error {
	clean := filepath.Clean(path)
	if clean == "" || clean == "." {
		return fmt.Errorf("invalid empty path")
	}
	if clean == "/" {
		return fmt.Errorf("refusing to trash system root")
	}

	// Home directory protection
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		homeClean := filepath.Clean(home)
		if clean == homeClean {
			return fmt.Errorf("refusing to trash user home directory: %s", clean)
		}
		if homeClean != "/" && clean == filepath.Dir(homeClean) {
			return fmt.Errorf("refusing to trash parent of user home directory: %s", clean)
		}
	}

	// Critical system directories
	systemDirs := []string{
		"/bin", "/boot", "/dev", "/etc", "/lib", "/lib32", "/lib64",
		"/proc", "/root", "/run", "/sbin", "/sys", "/usr", "/var",
	}
	for _, sys := range systemDirs {
		if clean == sys {
			return fmt.Errorf("refusing to trash system directory: %s", clean)
		}
	}

	return nil
}

// Trash moves the given file or directory to the user's trash conforming to the
// FreeDesktop.org (XDG) Trash Specification.
// It executes a cascade of standard tools (gio, trash-put, kioclient),
// falling back to a pure Go implementation of the XDG Trash specification.
func Trash(targetPath string) error {
	return TrashWithContext(context.Background(), targetPath)
}

// TrashWithContext moves the given file or directory to trash with a context for timeout/cancellation.
func TrashWithContext(ctx context.Context, targetPath string) error {
	canonical := util.CanonicalPath(targetPath)
	if canonical == "" {
		return fmt.Errorf("target path cannot be empty")
	}

	if err := SafetyCheck(canonical); err != nil {
		return err
	}

	if _, err := os.Lstat(canonical); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file or directory does not exist: %s", canonical)
		}
		return fmt.Errorf("cannot stat target: %w", err)
	}

	// Try standard CLI tools first
	if isExecutableAvailable("gio") {
		if err := runCommand(ctx, "gio", "trash", canonical); err == nil {
			slog.Debug("trashed target using gio", "path", canonical)
			return nil
		} else {
			slog.Warn("gio trash failed, trying fallback", "path", canonical, "error", err)
		}
	}

	if isExecutableAvailable("trash-put") {
		if err := runCommand(ctx, "trash-put", canonical); err == nil {
			slog.Debug("trashed target using trash-put", "path", canonical)
			return nil
		} else {
			slog.Warn("trash-put failed, trying fallback", "path", canonical, "error", err)
		}
	}

	if isExecutableAvailable("kioclient5") {
		if err := runCommand(ctx, "kioclient5", "move", canonical, "trash:/"); err == nil {
			slog.Debug("trashed target using kioclient5", "path", canonical)
			return nil
		} else {
			slog.Warn("kioclient5 failed, trying fallback", "path", canonical, "error", err)
		}
	} else if isExecutableAvailable("kioclient") {
		if err := runCommand(ctx, "kioclient", "move", canonical, "trash:/"); err == nil {
			slog.Debug("trashed target using kioclient", "path", canonical)
			return nil
		} else {
			slog.Warn("kioclient failed, trying fallback", "path", canonical, "error", err)
		}
	}

	// Pure Go XDG Trash Specification Fallback
	return trashXDGHome(canonical)
}

func isExecutableAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func runCommand(ctx context.Context, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v failed: %w (output: %s)", name, args, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// trashXDGHome moves the target path to $XDG_DATA_HOME/Trash (or ~/.local/share/Trash)
// conforming strictly to FreeDesktop.org Trash Specification 1.0.
func trashXDGHome(targetPath string) error {
	trashDir := getXDGTrashDir()
	filesDir := filepath.Join(trashDir, "files")
	infoDir := filepath.Join(trashDir, "info")

	if err := os.MkdirAll(filesDir, 0700); err != nil {
		return fmt.Errorf("create trash files directory: %w", err)
	}
	if err := os.MkdirAll(infoDir, 0700); err != nil {
		return fmt.Errorf("create trash info directory: %w", err)
	}

	baseName := filepath.Base(targetPath)
	destName, err := findUniqueTrashName(filesDir, infoDir, baseName)
	if err != nil {
		return fmt.Errorf("determine unique trash name: %w", err)
	}

	infoPath := filepath.Join(infoDir, destName+".trashinfo")
	destFilePath := filepath.Join(filesDir, destName)

	// Step 1: Write .trashinfo file FIRST (spec requirement)
	now := time.Now().Format("2006-01-02T15:04:05")
	escapedPath := url.PathEscape(targetPath)
	infoContent := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", escapedPath, now)

	if err := os.WriteFile(infoPath, []byte(infoContent), 0600); err != nil {
		return fmt.Errorf("write trashinfo metadata: %w", err)
	}

	// Step 2: Move the actual file/directory
	if err := moveOrCopy(targetPath, destFilePath); err != nil {
		// Clean up the orphaned .trashinfo on failure
		_ = os.Remove(infoPath)
		return fmt.Errorf("move target to trash files: %w", err)
	}

	return nil
}

// findUniqueTrashName finds an available name in the trash directories, resolving collisions
// by appending .1, .2, etc. per the XDG Trash spec.
func findUniqueTrashName(filesDir, infoDir, baseName string) (string, error) {
	name := baseName
	counter := 1

	for {
		filePath := filepath.Join(filesDir, name)
		infoPath := filepath.Join(infoDir, name+".trashinfo")

		_, errFile := os.Lstat(filePath)
		_, errInfo := os.Lstat(infoPath)

		if os.IsNotExist(errFile) && os.IsNotExist(errInfo) {
			return name, nil
		}

		name = baseName + "." + strconv.Itoa(counter)
		counter++
		if counter > 10000 {
			return "", fmt.Errorf("too many collisions for name %s", baseName)
		}
	}
}

// getXDGTrashDir returns the XDG Trash directory path.
func getXDGTrashDir() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.TempDir()
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "Trash")
}

// moveOrCopy attempts an atomic rename. If that fails across devices (EXDEV),
// it copies recursively and deletes the source.
func moveOrCopy(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// If cross-device move, perform copy and remove
	if isCrossDeviceError(err) {
		if copyErr := copyPath(src, dst); copyErr != nil {
			_ = os.RemoveAll(dst)
			return copyErr
		}
		return os.RemoveAll(src)
	}

	return err
}

func isCrossDeviceError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "invalid cross-device link") ||
		strings.Contains(err.Error(), "cross-device")
}

func copyPath(src, dst string) error {
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	}

	if fi.IsDir() {
		if err := os.MkdirAll(dst, fi.Mode()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			s := filepath.Join(src, entry.Name())
			d := filepath.Join(dst, entry.Name())
			if err := copyPath(s, d); err != nil {
				return err
			}
		}
		return nil
	}

	// Regular file copy
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fi.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
