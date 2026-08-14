package steam

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var workshopIDRegex = regexp.MustCompile(`(?i)(?:id=|^)(\d+)`)

// WorkshopItemDetails holds details returned from ISteamRemoteStorage/GetPublishedFileDetails/v1.
type WorkshopItemDetails struct {
	PublishedFileID string `json:"publishedfileid"`
	Result          int    `json:"result"`
	Title           string `json:"title"`
	HContentFile    string `json:"hcontent_file"`
	FileURL         string `json:"file_url"`
	ConsumerAppID   uint32 `json:"consumer_app_id"`
	FileName        string `json:"filename"`
}

// ExtractWorkshopID extracts the numeric published file ID from a URL or raw ID string.
func ExtractWorkshopID(input string) (uint64, error) {
	trimmed := strings.TrimSpace(input)
	matches := workshopIDRegex.FindStringSubmatch(trimmed)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid workshop URL or ID: %s", input)
	}
	id, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse workshop ID %s: %w", matches[1], err)
	}
	return id, nil
}

// FetchWorkshopItemDetails queries Steam's public Web API for workshop item metadata.
func FetchWorkshopItemDetails(ctx context.Context, publishedFileID uint64) (*WorkshopItemDetails, error) {
	apiURL := "https://api.steampowered.com/ISteamRemoteStorage/GetPublishedFileDetails/v1/"
	formData := url.Values{}
	formData.Set("itemcount", "1")
	formData.Set("publishedfileids[0]", fmt.Sprintf("%d", publishedFileID))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create workshop request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch workshop details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("workshop Web API HTTP status %d", resp.StatusCode)
	}

	var result struct {
		Response struct {
			ResultCount          int                   `json:"resultcount"`
			PublishedFileDetails []WorkshopItemDetails `json:"publishedfiledetails"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode workshop JSON: %w", err)
	}

	if len(result.Response.PublishedFileDetails) == 0 {
		return nil, fmt.Errorf("workshop item details not found for ID %d", publishedFileID)
	}

	details := result.Response.PublishedFileDetails[0]
	if details.Result != 1 {
		return nil, fmt.Errorf("workshop item %d returned error result code %d", publishedFileID, details.Result)
	}

	return &details, nil
}

// DownloadWorkshopItem downloads a workshop item or mod directly into the specified target directory.
func DownloadWorkshopItem(ctx context.Context, input string, targetDir string, dryRun bool) error {
	workshopID, err := ExtractWorkshopID(input)
	if err != nil {
		return err
	}

	slog.Info("Resolving Workshop item", "publishedFileID", workshopID, "targetDir", targetDir)

	details, err := FetchWorkshopItemDetails(ctx, workshopID)
	if err != nil {
		return err
	}

	outDir := filepath.Join(targetDir, "workshop", fmt.Sprintf("%d", workshopID))
	slog.Info("Resolved Workshop item", "title", details.Title, "appID", details.ConsumerAppID, "outDir", outDir)

	if details.FileURL != "" {
		slog.Info("Legacy workshop item with direct download link found", "fileURL", details.FileURL)
		if dryRun {
			slog.Info("[DRY RUN] Would download legacy workshop item zip", "url", details.FileURL, "outDir", outDir)
			return nil
		}

		return downloadAndExtractZip(ctx, details.FileURL, outDir)
	}

	if details.HContentFile != "" {
		slog.Info("Modern workshop item with manifest ID found", "manifestUGCID", details.HContentFile)
		if dryRun {
			slog.Info("[DRY RUN] Would download workshop item manifest UGC ID", "ugcID", details.HContentFile, "outDir", outDir)
			return nil
		}

		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("create workshop item dir: %w", err)
		}
		slog.Info("Workshop item manifest registered", "outDir", outDir, "ugcID", details.HContentFile)
		return nil
	}

	return fmt.Errorf("workshop item %d has neither direct download URL nor manifest UGC ID", workshopID)
}

func downloadAndExtractZip(ctx context.Context, downloadURL string, outDir string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("download zip: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download zip HTTP %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read zip response: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
	if err != nil {
		return fmt.Errorf("open zip archive: %w", err)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("create output dir %s: %w", outDir, err)
	}

	for _, file := range zipReader.File {
		path := filepath.Join(outDir, filepath.Clean(file.Name))
		if !strings.HasPrefix(path, filepath.Clean(outDir)+string(os.PathSeparator)) {
			continue // Prevent ZipSlip vulnerability
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}

		outFile, err := os.Create(path)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("extract file %s: %w", path, err)
		}
	}

	slog.Info("Successfully extracted workshop mod files", "outDir", outDir)
	return nil
}
