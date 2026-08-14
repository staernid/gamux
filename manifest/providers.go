package manifest

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

// FetchRevoBDKeys attempts to download and extract a .lua manifest key file from the RevoBD public API.
func FetchRevoBDKeys(ctx context.Context, appID uint32) (*ParsedLua, error) {
	if appID == 0 {
		return nil, fmt.Errorf("invalid appID 0")
	}

	url := fmt.Sprintf("https://api.luagen.revobd.club/%d.zip", appID)
	slog.Info("Fetching manifest keys from RevoBD API", "appID", appID, "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create revoBD request: %w", err)
	}

	req.Header.Set("User-Agent", "gamux/1.0")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("revoBD request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("revoBD HTTP status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read revoBD zip response: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("open revoBD zip archive: %w", err)
	}

	var luaContent string
	manifestFiles := make(map[string][]byte)

	for _, file := range zipReader.File {
		nameLower := strings.ToLower(file.Name)
		if strings.HasSuffix(nameLower, ".lua") {
			rc, err := file.Open()
			if err != nil {
				continue
			}
			contentBytes, err := io.ReadAll(rc)
			rc.Close()
			if err == nil {
				luaContent = string(contentBytes)
			}
		} else if strings.HasSuffix(nameLower, ".manifest") {
			rc, err := file.Open()
			if err != nil {
				continue
			}
			manifestBytes, err := io.ReadAll(rc)
			rc.Close()
			if err == nil {
				manifestFiles[file.Name] = manifestBytes
			}
		}
	}

	if luaContent == "" {
		return nil, fmt.Errorf("no .lua file found in RevoBD zip archive for AppID %d", appID)
	}

	parsed, err := ParseLua(luaContent)
	if err != nil {
		return nil, fmt.Errorf("parse RevoBD lua content: %w", err)
	}
	parsed.ManifestFiles = manifestFiles

	slog.Info("Successfully resolved keys and manifests from RevoBD", "appID", parsed.AppID, "depots", len(parsed.Depots), "manifestFiles", len(manifestFiles))
	return parsed, nil
}

// FetchSteamDepotDBKeys fetches depot decryption keys from the KoriaPolis Steam-Depot raw GitHub database.
func FetchSteamDepotDBKeys(ctx context.Context, appID uint32) (*ParsedLua, error) {
	url := "https://raw.githubusercontent.com/KoriaPolis/Steam-Depot/main/fallback_depotkeys.json"
	slog.Info("Fetching depot keys from Steam-Depot GitHub DB", "appID", appID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create steam-depot request: %w", err)
	}

	req.Header.Set("User-Agent", "gamux/1.0")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("steam-depot request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam-depot HTTP status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read steam-depot response: %w", err)
	}

	type DepotEntry struct {
		Key          string `json:"key"`
		ParentAppID string `json:"parent_appid"`
	}

	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
		return nil, fmt.Errorf("unmarshal steam-depot json: %w", err)
	}

	appIDStr := fmt.Sprintf("%d", appID)
	depots := make([]DepotKeyPair, 0)

	for depotStr, rawMsg := range rawMap {
		var keyStr string

		// Try parsing as string key
		if err := json.Unmarshal(rawMsg, &keyStr); err == nil {
			// Check if depot string starts with appID
			if strings.HasPrefix(depotStr, appIDStr) {
				dID, err := strconv.ParseUint(depotStr, 10, 32)
				if err == nil {
					depots = append(depots, DepotKeyPair{
						DepotID:       uint32(dID),
						DecryptionKey: keyStr,
					})
				}
			}
			continue
		}

		// Try parsing as object
		var entry DepotEntry
		if err := json.Unmarshal(rawMsg, &entry); err == nil {
			if entry.ParentAppID == appIDStr || strings.HasPrefix(depotStr, appIDStr) {
				dID, err := strconv.ParseUint(depotStr, 10, 32)
				if err == nil && entry.Key != "" {
					depots = append(depots, DepotKeyPair{
						DepotID:       uint32(dID),
						DecryptionKey: entry.Key,
					})
				}
			}
		}
	}

	if len(depots) == 0 {
		return nil, fmt.Errorf("no depot keys matching AppID %d found in Steam-Depot database", appID)
	}

	slog.Info("Successfully resolved keys from Steam-Depot DB", "appID", appID, "depots", len(depots))
	return &ParsedLua{
		AppID:  appID,
		Depots: depots,
	}, nil
}
