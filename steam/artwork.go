package steam

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// ArtworkResult holds local file paths for fetched artwork.
type ArtworkResult struct {
	CoverPath  string
	BannerPath string
}

// DownloadFile downloads a URL to a local target file path.
func DownloadFile(ctx context.Context, downloadURL, targetPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", downloadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", downloadURL, resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("create dir %s: %w", filepath.Dir(targetPath), err)
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("create file %s: %w", targetPath, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

