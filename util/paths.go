package util

import (
	"os"
	"path/filepath"
	"strings"
)

// FileExists checks if a file exists on disk.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// DirExists checks if a path exists and is a directory.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ExpandPath expands ~ to the user's home directory and standardizes slashes.
func ExpandPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = strings.ReplaceAll(path, "\\", "/")
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, strings.TrimPrefix(path[1:], "/"))
		}
	}
	return path
}

// CanonicalPath expands tildes, standardizes backslashes to forward slashes,
// resolves absolute paths, evaluates symlinks safely if the target exists, and cleans trailing slashes and redundant segments.
func CanonicalPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	expanded := ExpandPath(path)
	abs, err := filepath.Abs(expanded)
	if err == nil {
		if realPath, err := filepath.EvalSymlinks(abs); err == nil {
			return filepath.Clean(realPath)
		}
		return filepath.Clean(abs)
	}
	return filepath.Clean(expanded)
}

// DisplayPath replaces the user's home directory prefix with ~ for concise display.
func DisplayPath(path string) string {
	if path == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" && strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

// NormalizeRelativePath standardizes slashes and cleans relative paths, removing leading slashes and Windows drive letters if present.
func NormalizeRelativePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = strings.ReplaceAll(p, "\\", "/")
	if len(p) >= 2 && p[1] == ':' && ((p[0] >= 'a' && p[0] <= 'z') || (p[0] >= 'A' && p[0] <= 'Z')) {
		p = strings.TrimPrefix(p[2:], "/")
	}
	p = filepath.Clean(p)
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimPrefix(p, "./")
	return p
}

// JoinPaths safely joins a base directory with one or more subpaths,
// ensuring Windows backslashes and redundant dots in subpaths are normalized first.
func JoinPaths(base string, subpaths ...string) string {
	cleanBase := CanonicalPath(base)
	parts := make([]string, 0, len(subpaths)+1)
	parts = append(parts, cleanBase)
	for _, sub := range subpaths {
		norm := NormalizeRelativePath(sub)
		if norm != "" {
			parts = append(parts, norm)
		}
	}
	return filepath.Clean(filepath.Join(parts...))
}

// IsSamePath returns true if two filesystem paths refer to the exact same canonical path.
func IsSamePath(pathA, pathB string) bool {
	if pathA == "" || pathB == "" {
		return pathA == pathB
	}
	return CanonicalPath(pathA) == CanonicalPath(pathB)
}
