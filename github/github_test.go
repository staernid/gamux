package github

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/staernid/gamux/config"
)

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
	oldClient := HTTPClient
	defer func() { HTTPClient = oldClient }()

	HTTPClient = &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       io.NopCloser(bytes.NewBufferString("Not found")),
			}, nil
		}),
	}

	err := UpdateGBE(context.Background())
	if err == nil || !strings.Contains(err.Error(), "HTTP 404 Not Found") {
		t.Fatalf("expected 404 HTTP error, got: %v", err)
	}
}

func TestUpdateGBE_InvalidJSON(t *testing.T) {
	oldClient := HTTPClient
	defer func() { HTTPClient = oldClient }()

	HTTPClient = &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString("invalid json")),
			}, nil
		}),
	}

	err := UpdateGBE(context.Background())
	if err == nil || !strings.Contains(err.Error(), "failed to decode JSON") {
		t.Fatalf("expected JSON decode error, got: %v", err)
	}
}

func TestUpdateGBE_MissingAssets(t *testing.T) {
	oldClient := HTTPClient
	defer func() { HTTPClient = oldClient }()

	jsonResponse := `{
		"tag_name": "v1.0.0",
		"updated_at": "2026-01-01T00:00:00Z",
		"body": "Release notes",
		"assets": []
	}`

	HTTPClient = &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
			}, nil
		}),
	}

	err := UpdateGBE(context.Background())
	if err == nil || !strings.Contains(err.Error(), "failed to find Linux download URL") {
		t.Fatalf("expected missing Linux asset error, got: %v", err)
	}
}

func TestUpdateGBE_AlreadyUpToDate(t *testing.T) {
	oldHome := os.Getenv("HOME")
	tmpHome := mustTempDir(t)
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	oldClient := HTTPClient
	defer func() { HTTPClient = oldClient }()

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

	// Create pre-existing timestamp file matching release updated_at
	gbeHome := filepath.Join(tmpHome, config.GbeDir)
	if err := os.MkdirAll(gbeHome, 0755); err != nil {
		t.Fatal(err)
	}
	timestampFile := filepath.Join(gbeHome, ".gbe_timestamp")
	if err := os.WriteFile(timestampFile, []byte(now.String()), 0644); err != nil {
		t.Fatal(err)
	}

	HTTPClient = &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
			}, nil
		}),
	}

	err := UpdateGBE(context.Background())
	if err != nil {
		t.Fatalf("expected nil error for up-to-date check, got: %v", err)
	}
}
