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

	"github.com/staernid/gamux/cache"
)

// FetchRevoBDKeys attempts to download and extract a .lua manifest key file from the RevoBD public API with caching.
func FetchRevoBDKeys(ctx context.Context, appID uint32) (*ParsedLua, error) {
	if appID == 0 {
		return nil, fmt.Errorf("invalid appID 0")
	}

	var bodyBytes []byte
	if cachedData, ok := cache.GetRevoBDZIP(appID); ok && len(cachedData) > 0 {
		bodyBytes = cachedData
	} else {
		url := fmt.Sprintf("https://api.luagen.revobd.club/%d.zip", appID)
		slog.Info("Fetching manifest keys from RevoBD API", "appID", appID, "url", url)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create revoBD request: %w", err)
		}

		req.Header.Set("User-Agent", "gamux/1.0")

		resp, err := HTTPClientFromContext(ctx).Do(req)
		if err != nil {
			return nil, fmt.Errorf("revoBD request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("revoBD HTTP status %d", resp.StatusCode)
		}

		readBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read revoBD zip response: %w", err)
		}
		bodyBytes = readBytes
		_ = cache.SaveRevoBDZIP(appID, bodyBytes)
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

func isDepotBelongingToApp(dID uint32, depotStr string, appID uint32, appIDStr string) bool {
	if strings.HasPrefix(depotStr, appIDStr) {
		return true
	}
	if dID >= appID && dID <= appID+100 {
		return true
	}
	if strings.HasSuffix(appIDStr, "0") && len(appIDStr) > 2 {
		basePrefix := strings.TrimSuffix(appIDStr, "0")
		if strings.HasPrefix(depotStr, basePrefix) && len(depotStr) == len(appIDStr) {
			return true
		}
	}
	return false
}

// FetchSteamDepotDBKeys fetches depot decryption keys from the KoriaPolis Steam-Depot raw GitHub database with caching.
func FetchSteamDepotDBKeys(ctx context.Context, appID uint32) (*ParsedLua, error) {
	if appID == 0 {
		return nil, fmt.Errorf("invalid appID 0")
	}

	var bodyBytes []byte
	if cachedData, ok := cache.GetSteamDepotDB(); ok && len(cachedData) > 0 {
		bodyBytes = cachedData
	} else {
		url := "https://raw.githubusercontent.com/KoriaPolis/Steam-Depot/main/fallback_depotkeys.json"
		slog.Info("Fetching depot keys from Steam-Depot GitHub DB", "appID", appID)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create steam-depot request: %w", err)
		}

		req.Header.Set("User-Agent", "gamux/1.0")

		resp, err := HTTPClientFromContext(ctx).Do(req)
		if err != nil {
			return nil, fmt.Errorf("steam-depot request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("steam-depot HTTP status %d", resp.StatusCode)
		}

		readBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read steam-depot response: %w", err)
		}
		bodyBytes = readBytes
		_ = cache.SaveSteamDepotDB(bodyBytes)
	}

	type DepotEntry struct {
		Key         string `json:"key"`
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
			dID, err := strconv.ParseUint(depotStr, 10, 32)
			if err == nil && isDepotBelongingToApp(uint32(dID), depotStr, appID, appIDStr) {
				depots = append(depots, DepotKeyPair{
					DepotID:       uint32(dID),
					DecryptionKey: keyStr,
				})
			}
			continue
		}

		// Try parsing as object
		var entry DepotEntry
		if err := json.Unmarshal(rawMsg, &entry); err == nil {
			dID, err := strconv.ParseUint(depotStr, 10, 32)
			if err == nil && (entry.ParentAppID == appIDStr || isDepotBelongingToApp(uint32(dID), depotStr, appID, appIDStr)) {
				if entry.Key != "" {
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

// FetchManifesthubKeys fetches depot decryption keys from the GitLab steamautocracks/manifesthub repository with caching.
func FetchManifesthubKeys(ctx context.Context, appID uint32) (*ParsedLua, error) {
	if appID == 0 {
		return nil, fmt.Errorf("invalid appID 0")
	}

	var bodyBytes []byte
	if cachedData, ok := cache.GetManifesthubDB(); ok && len(cachedData) > 0 {
		bodyBytes = cachedData
	} else {
		url := "https://gitlab.com/steamautocracks/manifesthub/-/raw/main/depotkeys.json"
		slog.Info("Fetching depot keys from GitLab Manifesthub", "appID", appID)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create manifesthub request: %w", err)
		}

		req.Header.Set("User-Agent", "gamux/1.0")

		resp, err := HTTPClientFromContext(ctx).Do(req)
		if err != nil {
			return nil, fmt.Errorf("manifesthub request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("manifesthub HTTP status %d", resp.StatusCode)
		}

		readBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read manifesthub response: %w", err)
		}
		bodyBytes = readBytes
		_ = cache.SaveManifesthubDB(bodyBytes)
	}

	var rawMap map[string]string
	if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
		return nil, fmt.Errorf("unmarshal manifesthub json: %w", err)
	}

	appIDStr := fmt.Sprintf("%d", appID)
	depots := make([]DepotKeyPair, 0)

	for depotStr, keyStr := range rawMap {
		dID, err := strconv.ParseUint(depotStr, 10, 32)
		if err == nil && isDepotBelongingToApp(uint32(dID), depotStr, appID, appIDStr) && keyStr != "" {
			depots = append(depots, DepotKeyPair{
				DepotID:       uint32(dID),
				DecryptionKey: keyStr,
			})
		}
	}

	if len(depots) == 0 {
		return nil, fmt.Errorf("no depot keys matching AppID %d found in Manifesthub database", appID)
	}

	slog.Info("Successfully resolved keys from GitLab Manifesthub", "appID", appID, "depots", len(depots))
	return &ParsedLua{
		AppID:  appID,
		Depots: depots,
	}, nil
}

// FetchGMRCRequestCode queries a GMRC endpoint to resolve the 64-bit manifest request code for a given manifest GID.
func FetchGMRCRequestCode(ctx context.Context, manifestGID string, customEndpoint string) (string, error) {
	if manifestGID == "" {
		return "", fmt.Errorf("manifest GID cannot be empty")
	}

	endpoint := customEndpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("http://gmrc.wudrm.com/manifest/%s", manifestGID)
	} else if strings.Contains(endpoint, "%s") {
		endpoint = fmt.Sprintf(endpoint, manifestGID)
	}

	slog.Info("Resolving manifest request code from GMRC", "manifestGID", manifestGID, "url", endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create GMRC request: %w", err)
	}

	req.Header.Set("User-Agent", "gamux/1.0")
	req.Header.Set("Referer", "http://gmrc.wudrm.com/")

	resp, err := HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return "", fmt.Errorf("GMRC request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read GMRC response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GMRC HTTP status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	code := strings.TrimSpace(string(bodyBytes))
	if code == "" || strings.HasPrefix(code, "<") || !isDigitsOnly(code) {
		return "", fmt.Errorf("invalid manifest request code returned from GMRC: %s", code)
	}

	return code, nil
}

func isDigitsOnly(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// DownloadManifestFromSteamCDN streams the encrypted binary .manifest from SteamPipe CDN using the provided GMRC request code.
func DownloadManifestFromSteamCDN(ctx context.Context, depotID uint32, manifestGID string, reqCode string, cdnHost string) ([]byte, error) {
	if depotID == 0 || manifestGID == "" || reqCode == "" {
		return nil, fmt.Errorf("depotID, manifestGID, and reqCode are required to download from Steam CDN")
	}

	host := cdnHost
	if host == "" {
		host = "cache10-fra1.steamcontent.com"
	}

	manifestURL := fmt.Sprintf("http://%s/depot/%d/manifest/%s/5/%s", host, depotID, manifestGID, reqCode)
	slog.Info("Downloading binary manifest from SteamPipe CDN", "depotID", depotID, "manifestGID", manifestGID, "url", manifestURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create manifest CDN request: %w", err)
	}
	req.Header.Set("User-Agent", "Valve/Steam HTTP Client 1.0")

	resp, err := HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return nil, fmt.Errorf("manifest CDN request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest CDN HTTP status %d", resp.StatusCode)
	}

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read manifest payload from CDN: %w", err)
	}

	// Decompress if Steam served the manifest inside a ZIP archive container
	if len(payload) >= 4 && payload[0] == 0x50 && payload[1] == 0x4B && payload[2] == 0x03 && payload[3] == 0x04 {
		zipReader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
		if err == nil && len(zipReader.File) > 0 {
			rc, err := zipReader.File[0].Open()
			if err == nil {
				defer rc.Close()
				if extracted, err := io.ReadAll(rc); err == nil && len(extracted) > 0 {
					return extracted, nil
				}
			}
		}
	}

	return payload, nil
}

