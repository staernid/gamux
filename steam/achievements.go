package steam

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/sync/errgroup"
)

// DownloadAchievementImages downloads achievement images from Steam CDN.
func DownloadAchievementImages(ctx context.Context, appID int, imageNames []string, outputFolder string) error {
	slog.Info("Downloading achievement images", "appID", appID, "count", len(imageNames), "output", outputFolder)

	if len(imageNames) == 0 {
		return nil
	}

	if err := os.MkdirAll(outputFolder, 0755); err != nil {
		return fmt.Errorf("failed to create output folder: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10) // Limit concurrency

	urls := []string{
		"https://cdn.akamai.steamstatic.com/steamcommunity/public/images/apps/",
		"https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps/",
	}

	for _, name := range imageNames {
		imgName := name
		g.Go(func() error {
			succeeded := false
			for _, baseURL := range urls {
				url := fmt.Sprintf("%s%d/%s", baseURL, appID, imgName)
				err := func() error {
					req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
					if err != nil {
						return err
					}

					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						return err
					}
					defer resp.Body.Close()

					if resp.StatusCode != http.StatusOK {
						return fmt.Errorf("HTTP status %d", resp.StatusCode)
					}

					imgPath := filepath.Join(outputFolder, imgName)
					out, err := os.Create(imgPath)
					if err != nil {
						return fmt.Errorf("failed to create image file: %w", err)
					}
					defer out.Close()

					if _, err := io.Copy(out, resp.Body); err != nil {
						return fmt.Errorf("failed to write image data: %w", err)
					}
					if err := out.Close(); err != nil {
						return err
					}
					succeeded = true
					return nil
				}()
				if err == nil {
					break
				}
			}

			if !succeeded {
				slog.Warn("Failed to download achievement image", "name", imgName)
			}
			return nil
		})
	}

	return g.Wait()
}
