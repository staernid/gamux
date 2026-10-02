package gui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/util"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// -----------------------------------------------------------------------------
// System & OS Integration
// -----------------------------------------------------------------------------

// SelectDirectory opens a native OS directory picker dialog.
func (a *App) SelectDirectory(title, defaultDir string) (string, error) {
	wailsApp := application.Get()
	if wailsApp == nil || wailsApp.Dialog == nil {
		return "", fmt.Errorf("GUI runtime context not ready")
	}
	if defaultDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			defaultDir = home
		}
	}
	dialog := wailsApp.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle(title).
		SetDirectory(defaultDir)

	if a.mainWindow != nil {
		dialog.AttachToWindow(a.mainWindow)
	}

	return dialog.PromptForSingleSelection()
}

// SelectFile opens a native OS file picker dialog.
func (a *App) SelectFile(title, filterName, filterPattern string) (string, error) {
	wailsApp := application.Get()
	if wailsApp == nil || wailsApp.Dialog == nil {
		return "", fmt.Errorf("GUI runtime context not ready")
	}
	dialog := wailsApp.Dialog.OpenFile().
		CanChooseFiles(true).
		CanChooseDirectories(false).
		SetTitle(title)

	if filterName != "" && filterPattern != "" {
		dialog.AddFilter(filterName, filterPattern)
	}

	if a.mainWindow != nil {
		dialog.AttachToWindow(a.mainWindow)
	}

	return dialog.PromptForSingleSelection()
}

// OpenDirectory opens the specified directory or containing folder in the system default file manager.
func (a *App) OpenDirectory(dirPath string) error {
	dirPath = util.CanonicalPath(dirPath)
	if dirPath == "" {
		return fmt.Errorf("directory path cannot be empty")
	}
	if !util.FileExists(dirPath) {
		return fmt.Errorf("directory does not exist: %s", dirPath)
	}

	target := dirPath
	if fi, err := os.Stat(dirPath); err == nil && !fi.IsDir() {
		target = filepath.Dir(dirPath)
	}

	cmd := exec.Command("xdg-open", target)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open file manager: %w", err)
	}
	return nil
}

// OpenURL opens the specified URL in the user's default web browser.
func (a *App) OpenURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	if wailsApp := application.Get(); wailsApp != nil && wailsApp.Browser != nil {
		if err := wailsApp.Browser.OpenURL(rawURL); err == nil {
			return nil
		}
	}
	cmd := exec.Command("xdg-open", rawURL)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open URL: %w", err)
	}
	return nil
}

// GetDiskSpace returns filesystem disk usage metrics (total, free, used bytes) for the given path.
func (a *App) GetDiskSpace(path string) (*util.DiskSpaceInfo, error) {
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	return a.Engine.GetDiskSpace(path)
}
