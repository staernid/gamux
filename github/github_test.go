package github

import (
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

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
	restore := testutil.SilenceLogging()
	code := m.Run()
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
