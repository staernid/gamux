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
	"time"

	"github.com/staernid/gamux/cache"
)


// HTTPClient allows injecting custom HTTP transport for testing.
var HTTPClient = &http.Client{Timeout: 30 * time.Second}

// HubcapKeyEntry represents a single depot key item returned from Hubcap API.
type HubcapKeyEntry struct {
	DepotID       uint32 `json:"depot_id"`
	DecryptionKey string `json:"decryption_key"`
}

// HubcapManifestResponse represents JSON response payload from Hubcap Manifest API.
type HubcapManifestResponse struct {
	AppID  uint32           `json:"app_id"`
	Depots []HubcapKeyEntry `json:"depots"`
}

// FetchHubcapKeys queries the Hubcap Manifest API for depot keys corresponding to an AppID.
func FetchHubcapKeys(ctx context.Context, appID uint32, apiKey string) (*ParsedLua, error) {
	// 0. Check local disk cache first (saves 100% of API quota)
	if cachedData, ok := cache.GetHubcapZIP(appID); ok {
		if zipReader, err := zip.NewReader(bytes.NewReader(cachedData), int64(len(cachedData))); err == nil {
			var parsedLua *ParsedLua
			manifestFiles := make(map[string][]byte)

			for _, file := range zipReader.File {
				rc, err := file.Open()
				if err != nil {
					continue
				}
				content, err := io.ReadAll(rc)
				_ = rc.Close()
				if err != nil {
					continue
				}

				name := file.Name
				if strings.HasSuffix(strings.ToLower(name), ".lua") {
					if p, err := ParseLua(string(content)); err == nil && p != nil {
						parsedLua = p
					}
				} else if strings.HasSuffix(strings.ToLower(name), ".manifest") {
					manifestFiles[name] = content
				}
			}

			if parsedLua != nil {
				parsedLua.ManifestFiles = manifestFiles
				slog.Info("Successfully loaded keys and manifests from local Hubcap cache", "appID", parsedLua.AppID, "depots", len(parsedLua.Depots), "manifestFiles", len(parsedLua.ManifestFiles))
				return parsedLua, nil
			}
		}
	}

	if apiKey == "" {
		return nil, fmt.Errorf("hubcap API key cannot be empty")
	}

	endpoints := []string{
		fmt.Sprintf("https://hubcapmanifest.com/api/v1/manifest/%d", appID),
		fmt.Sprintf("https://hubcapmanifest.com/hubcaptools/api/keys/%d", appID),
		fmt.Sprintf("https://hubcapmanifest.com/api/v1/keys/%d", appID),
	}

	slog.Info("Querying Hubcap Manifest API for depot keys and manifests", "appID", appID)

	var lastErr error
	for _, apiURL := range endpoints {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("X-API-Key", apiKey)
		req.Header.Set("User-Agent", "gamux/1.0")

		resp, err := HTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("hubcap request failed for %s: %w", apiURL, err)
			continue
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("read hubcap response body: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("hubcap API HTTP status %d: %s", resp.StatusCode, string(bodyBytes))
			continue
		}

		// 1. Try parsing as a ZIP archive (containing .lua and binary .manifest files)
		if zipReader, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes))); err == nil {
			var parsedLua *ParsedLua
			manifestFiles := make(map[string][]byte)

			for _, file := range zipReader.File {
				rc, err := file.Open()
				if err != nil {
					continue
				}
				content, err := io.ReadAll(rc)
				_ = rc.Close()
				if err != nil {
					continue
				}

				name := file.Name
				if strings.HasSuffix(strings.ToLower(name), ".lua") {
					if p, err := ParseLua(string(content)); err == nil && p != nil {
						parsedLua = p
					}
				} else if strings.HasSuffix(strings.ToLower(name), ".manifest") {
					manifestFiles[name] = content
				}
			}

			if parsedLua != nil {
				parsedLua.ManifestFiles = manifestFiles
				_ = cache.SaveHubcapZIP(appID, bodyBytes)
				slog.Info("Successfully resolved keys and manifests from Hubcap ZIP and cached to disk", "appID", parsedLua.AppID, "depots", len(parsedLua.Depots), "manifestFiles", len(parsedLua.ManifestFiles))
				return parsedLua, nil
			}
		}


		// 2. Try parsing as addappid(...) raw lua text
		bodyStr := string(bodyBytes)
		if strings.Contains(bodyStr, "addappid") {
			parsedLua, err := ParseLua(bodyStr)
			if err == nil && parsedLua != nil && len(parsedLua.Depots) > 0 {
				slog.Info("Successfully resolved depot keys from Hubcap API (lua format)", "appID", parsedLua.AppID, "depots", len(parsedLua.Depots))
				return parsedLua, nil
			}
		}

		// Try parsing as JSON structure
		var jsonPayload HubcapManifestResponse
		if err := json.Unmarshal(bodyBytes, &jsonPayload); err == nil && len(jsonPayload.Depots) > 0 {
			targetAppID := jsonPayload.AppID
			if targetAppID == 0 {
				targetAppID = appID
			}

			depots := make([]DepotKeyPair, 0, len(jsonPayload.Depots))
			for _, entry := range jsonPayload.Depots {
				depots = append(depots, DepotKeyPair{
					DepotID:       entry.DepotID,
					DecryptionKey: entry.DecryptionKey,
				})
			}

			slog.Info("Successfully resolved depot keys from Hubcap API (JSON format)", "appID", targetAppID, "depots", len(depots))
			return &ParsedLua{
				AppID:  targetAppID,
				Depots: depots,
			}, nil
		}

		// Generic JSON map fallback e.g. {"depot_id": "key_hex"}
		var rawMap map[string]string
		if err := json.Unmarshal(bodyBytes, &rawMap); err == nil && len(rawMap) > 0 {
			depots := make([]DepotKeyPair, 0, len(rawMap))
			for k, v := range rawMap {
				dID, err := strconv.ParseUint(k, 10, 32)
				if err == nil {
					depots = append(depots, DepotKeyPair{
						DepotID:       uint32(dID),
						DecryptionKey: v,
					})
				}
			}
			if len(depots) > 0 {
				slog.Info("Successfully resolved depot keys from Hubcap API (map format)", "appID", appID, "depots", len(depots))
				return &ParsedLua{
					AppID:  appID,
					Depots: depots,
				}, nil
			}
		}

		if strings.HasPrefix(strings.TrimSpace(strings.ToLower(bodyStr)), "<!") || strings.HasPrefix(strings.TrimSpace(strings.ToLower(bodyStr)), "<html") {
			lastErr = fmt.Errorf("Hubcap API returned non-JSON HTML response (no keys for AppID %d)", appID)
		} else {
			if len(bodyStr) > 120 {
				bodyStr = bodyStr[:117] + "..."
			}
			lastErr = fmt.Errorf("could not parse valid depot keys from Hubcap response: %s", bodyStr)
		}
	}


	if lastErr != nil {
		return nil, fmt.Errorf("failed to resolve depot keys from Hubcap API for AppID %d: %w", appID, lastErr)
	}

	return nil, fmt.Errorf("hubcap API returned no depot keys for AppID %d", appID)
}
