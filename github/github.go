package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gbe_fork_helper/config"
	"gbe_fork_helper/util"

	"github.com/charmbracelet/glamour"
	"golang.org/x/sync/errgroup"
)

// UpdateGBE fetches and extracts the latest GBE fork.
func UpdateGBE(ctx context.Context) error {
	slog.Info("Fetching latest GBE fork from GitHub")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	gbeHome := filepath.Join(homeDir, config.GbeDir)
	timestampFile := filepath.Join(gbeHome, ".gbe_timestamp")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.GithubAPIURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch release information: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch release information: HTTP %s", resp.Status)
	}

	var release config.Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	if _, err := os.Stat(timestampFile); err == nil {
		timestamp, err := os.ReadFile(timestampFile)
		if err == nil && string(timestamp) == release.UpdatedAt.String() {
			slog.Info("GBE fork is already up-to-date")
			return nil
		}
	}
	// Create a new renderer with the desired style
	// glamour.WithAutoStyle() automatically detects the current terminal's dark/light mode
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
	)
	if err != nil {
		slog.Error("Error creating markdown renderer", "error", err)
		// Fallback to raw text
		fmt.Println(release.Body)
	} else {
		// Render the markdown text
		renderedText, err := renderer.Render(release.Body)
		if err != nil {
			slog.Error("Error rendering markdown", "error", err)
			fmt.Println(release.Body)
		} else {
			// Print the rendered output to the command line
			fmt.Println(renderedText)
		}
	}

	if err := os.MkdirAll(gbeHome, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", gbeHome, err)
	}

	// Find download URLs
	linuxURL := ""
	winURL := ""
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, "linux-release.tar.bz2") {
			linuxURL = asset.BrowserDownloadURL
		} else if strings.HasSuffix(asset.Name, "win-release.7z") {
			winURL = asset.BrowserDownloadURL
		}
	}

	if linuxURL == "" {
		return fmt.Errorf("failed to find Linux download URL")
	}
	if winURL == "" {
		return fmt.Errorf("failed to find Windows download URL")
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("Downloading Linux release")
		if err := util.DownloadAndExtract(gCtx, linuxURL, filepath.Join(gbeHome, "linux_release"), "tar.bz2"); err != nil {
			return fmt.Errorf("failed to update Linux release: %w", err)
		}
		slog.Info("Linux release extracted")
		return nil
	})

	g.Go(func() error {
		slog.Info("Downloading Windows release")
		if err := util.DownloadAndExtract(gCtx, winURL, filepath.Join(gbeHome, "win_release"), "7z"); err != nil {
			return fmt.Errorf("failed to update Windows release: %w", err)
		}
		slog.Info("Windows release extracted")
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	if err := os.WriteFile(timestampFile, []byte(release.UpdatedAt.String()), 0644); err != nil {
		return fmt.Errorf("failed to write timestamp file: %w", err)
	}

	slog.Info("GBE fork updated successfully")
	return nil
}
