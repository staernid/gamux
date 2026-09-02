package github

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/util"
	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
	restore := testutil.SilenceLogging()
	cache.DisableCache = true
	code := m.Run()
	cache.DisableCache = false
	restore()
	os.Exit(code)
}

type mockRoundTripper func(req *http.Request) (*http.Response, error)

func (m mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m(req)
}


func mustTempDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("", "github-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

func TestUpdateGBE_HTTPError(t *testing.T) {
	mockClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       io.NopCloser(bytes.NewBufferString("Not found")),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	err := UpdateGBE(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "HTTP 404 Not Found") {
		t.Fatalf("expected 404 HTTP error, got: %v", err)
	}
}

func TestUpdateGBE_InvalidJSON(t *testing.T) {
	mockClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString("invalid json")),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	err := UpdateGBE(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to decode JSON") {
		t.Fatalf("expected JSON decode error, got: %v", err)
	}
}

func TestUpdateGBE_MissingAssets(t *testing.T) {
	jsonResponse := `{
		"tag_name": "v1.0.0",
		"updated_at": "2026-01-01T00:00:00Z",
		"body": "Release notes",
		"assets": []
	}`

	mockClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	err := UpdateGBE(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to find Linux download URL") {
		t.Fatalf("expected missing Linux asset error, got: %v", err)
	}
}

func TestUpdateGBE_AlreadyUpToDate(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	jsonResponse := `{
		"tag_name": "v1.0.0",
		"updated_at": "` + now.Format(time.RFC3339) + `",
		"body": "Release notes",
		"assets": [
			{"name": "linux-release.tar.bz2", "browser_download_url": "http://example.com/linux"},
			{"name": "win-release.7z", "browser_download_url": "http://example.com/win"}
		]
	}`

	cfg := &config.Config{
		GbeDir: tmpDir,
	}
	// Create pre-existing timestamp file matching release updated_at
	gbeHome := tmpDir
	if err := os.MkdirAll(gbeHome, 0755); err != nil {
		t.Fatal(err)
	}
	var rel Release
	_ = json.Unmarshal([]byte(jsonResponse), &rel)
	timestampFile := filepath.Join(gbeHome, ".gbe_timestamp")
	if err := os.WriteFile(timestampFile, []byte(now.Format(time.RFC3339)), 0644); err != nil {
		t.Fatal(err)
	}

	mockClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	err := UpdateGBE(ctx, cfg)
	if err != nil {
		t.Fatalf("expected nil error for up-to-date check, got: %v", err)
	}
}

func TestFetchUMURelease(t *testing.T) {
	jsonResponse := `{
		"tag_name": "1.1.0",
		"updated_at": "2026-08-01T00:00:00Z",
		"published_at": "2026-08-01T00:00:00Z",
		"name": "umu-launcher 1.1.0",
		"body": "Release notes for umu",
		"html_url": "https://github.com/Open-Wine-Components/umu-launcher/releases/tag/1.1.0",
		"assets": [
			{"name": "umu-run", "browser_download_url": "https://example.com/umu-run"}
		]
	}`

	mockClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	info, err := FetchUMURelease(ctx, nil)
	if err != nil {
		t.Fatalf("FetchUMURelease failed: %v", err)
	}
	if info.Key != "umu" || info.LatestTag != "1.1.0" {
		t.Errorf("unexpected ToolInfo: %+v", info)
	}
}

func TestUpdateUMUAssets(t *testing.T) {
	tmpDir := t.TempDir()
	jsonResponse := `{
		"tag_name": "1.1.0",
		"updated_at": "2026-08-01T00:00:00Z",
		"assets": [
			{"name": "umu-run", "browser_download_url": "https://example.com/umu-run"}
		]
	}`

	mockClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "api.github.com") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString("#!/usr/bin/env python3\nprint('umu')\n")),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	err := UpdateUMUAssets(ctx, nil, tmpDir)
	if err != nil {
		t.Fatalf("UpdateUMUAssets failed: %v", err)
	}

	downloaded := filepath.Join(tmpDir, "umu-run")
	if data, err := os.ReadFile(downloaded); err != nil || !strings.Contains(string(data), "umu") {
		t.Errorf("expected downloaded umu-run file, got err: %v, data: %s", err, string(data))
	}
}

func TestUpdateUMUAssets_TarZipapp(t *testing.T) {
	tmpDir := t.TempDir()
	jsonResponse := `{
		"tag_name": "1.4.4",
		"updated_at": "2026-08-01T00:00:00Z",
		"assets": [
			{"name": "python3-umu-launcher_1.4.4.deb", "browser_download_url": "https://example.com/deb"},
			{"name": "umu-launcher-1.4.4-zipapp.tar", "browser_download_url": "https://example.com/zipapp.tar"}
		]
	}`

	// Create valid tar in memory with umu/umu-run
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	scriptContent := []byte("#!/usr/bin/env python3\n# umu standalone zipapp\n")
	_ = tw.WriteHeader(&tar.Header{
		Name: "umu/umu-run",
		Mode: 0755,
		Size: int64(len(scriptContent)),
	})
	_, _ = tw.Write(scriptContent)
	_ = tw.Close()

	mockClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "api.github.com") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewReader(tarBuf.Bytes())),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	err := UpdateUMUAssets(ctx, nil, tmpDir)
	if err != nil {
		t.Fatalf("UpdateUMUAssets failed with tar: %v", err)
	}

	downloaded := filepath.Join(tmpDir, "umu-run")
	if data, err := os.ReadFile(downloaded); err != nil || !strings.Contains(string(data), "umu standalone zipapp") {
		t.Errorf("expected extracted umu-run from zipapp.tar, got err: %v, data: %s", err, string(data))
	}
}

func TestExtractZipArchive_ZipSlipProtection(t *testing.T) {
	tmpDir := t.TempDir()
	zipFile := filepath.Join(tmpDir, "test.zip")
	destDir := filepath.Join(tmpDir, "extracted")
	_ = os.MkdirAll(destDir, 0755)

	// Create a zip with safe file and malicious ZipSlip paths
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	safeWriter, _ := zw.Create("safe/file.txt")
	_, _ = safeWriter.Write([]byte("safe content"))

	maliciousWriter1, _ := zw.Create("../evil.txt")
	_, _ = maliciousWriter1.Write([]byte("evil content"))

	maliciousWriter2, _ := zw.Create("../../etc/passwd")
	_, _ = maliciousWriter2.Write([]byte("root:x:0:0"))

	_ = zw.Close()
	_ = os.WriteFile(zipFile, buf.Bytes(), 0644)

	err := extractZipArchive(zipFile, destDir)
	if err != nil {
		t.Fatalf("extractZipArchive failed: %v", err)
	}

	// Safe file should exist
	if !util.FileExists(filepath.Join(destDir, "safe", "file.txt")) {
		t.Errorf("expected safe/file.txt to be extracted")
	}

	// Malicious files outside destDir must NOT exist
	if util.FileExists(filepath.Join(tmpDir, "evil.txt")) {
		t.Errorf("ZipSlip vulnerability detected: evil.txt was extracted outside destDir")
	}
}

