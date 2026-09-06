package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/staernid/gamux/util"
)

// ResolveLogPath determines the log file location for a game.
func ResolveLogPath(gameDir, gameName, appID string) string {
	slug := util.Slugify(gameName)
	if slug == "" {
		if appID != "" {
			slug = appID
		} else if gameDir != "" {
			slug = util.Slugify(filepath.Base(gameDir))
		}
	}
	if slug == "" {
		slug = "game"
	}

	if gameDir != "" {
		gamuxDir := filepath.Join(gameDir, ".gamux")
		_ = os.MkdirAll(gamuxDir, 0755)
		return filepath.Join(gamuxDir, "last_run.log")
	}

	home, _ := os.UserHomeDir()
	logsDir := filepath.Join(home, ".local", "share", "gamux", "logs")
	_ = os.MkdirAll(logsDir, 0755)
	return filepath.Join(logsDir, slug+".log")
}

func createLogFile(logPath string, resolved LaunchOptions, cmd *exec.Cmd) (*os.File, error) {
	if logPath == "" {
		return nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	header := fmt.Sprintf("=== Gamux Launch Log ===\nTime: %s\nGame: %s\nAppID: %s\nPlatform: %s\nMode: %s\nRunner: %s\nExecutable: %s\nWorkingDir: %s\nCommand: %s\n========================\n\n",
		time.Now().Format(time.RFC1123),
		resolved.Name,
		resolved.AppID,
		resolved.Platform,
		resolved.OperatingMode,
		resolved.Runner,
		resolved.ExePath,
		cmd.Dir,
		cmd.String(),
	)
	_, _ = f.WriteString(header)
	_ = f.Sync()
	return f, nil
}

// GetLog reads the last or current execution log for a game.
func GetLog(gameDir, gameName, appID string) (string, error) {
	logPath := ResolveLogPath(gameDir, gameName, appID)
	if !util.FileExists(logPath) && gameDir != "" {
		legacyPath := filepath.Join(gameDir, "steam_settings", "last_run.log")
		if util.FileExists(legacyPath) {
			logPath = legacyPath
		}
	}
	if !util.FileExists(logPath) {
		slug := util.Slugify(gameName)
		if slug == "" && appID != "" {
			slug = appID
		}
		if slug != "" {
			home, _ := os.UserHomeDir()
			altPath := filepath.Join(home, ".local", "share", "gamux", "logs", slug+".log")
			if util.FileExists(altPath) {
				logPath = altPath
			}
		}
	}

	if !util.FileExists(logPath) {
		return "", fmt.Errorf("no launch log available for this game yet")
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		return "", fmt.Errorf("read game log: %w", err)
	}

	if len(data) > 512*1024 {
		data = data[len(data)-512*1024:]
	}

	return string(data), nil
}
