package steam

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mockRoundTripper func(req *http.Request) (*http.Response, error)

func (f mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetchAppName(t *testing.T) {
	// Backup and restore default transport
	oldTransport := http.DefaultClient.Transport
	defer func() {
		http.DefaultClient.Transport = oldTransport
	}()

	t.Run("successful response", func(t *testing.T) {
		http.DefaultClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.Path, "appdetails") {
				t.Errorf("unexpected URL path: %s", req.URL.Path)
			}
			jsonResp := `{"123": {"success": true, "data": {"name": "Super Tux Run"}}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
				Header:     make(http.Header),
			}, nil
		})

		name, err := FetchAppName(context.Background(), "123")
		if err != nil {
			t.Fatalf("FetchAppName failed: %v", err)
		}
		if name != "Super Tux Run" {
			t.Errorf("expected name 'Super Tux Run', got '%s'", name)
		}
	})

	t.Run("rate limit HTTP 429", func(t *testing.T) {
		http.DefaultClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(bytes.NewBufferString("Too Many Requests")),
				Header:     make(http.Header),
			}, nil
		})

		_, err := FetchAppName(context.Background(), "123")
		if err == nil {
			t.Fatal("expected error due to HTTP 429, got nil")
		}
		if !strings.Contains(err.Error(), "HTTP 429") {
			t.Errorf("expected error message to contain 'HTTP 429', got: %v", err)
		}
	})
}

func TestFetchDLCs(t *testing.T) {
	oldTransport := http.DefaultClient.Transport
	defer func() {
		http.DefaultClient.Transport = oldTransport
	}()

	tmpDir, err := os.MkdirTemp("", "testfetchdlcs")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	http.DefaultClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "ajaxgetfilteredrecommendations") {
			htmlResp := `{"results_html": "<div data-ds-appid=\"456\"></div><div data-ds-appid=\"789\"></div>"}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(htmlResp)),
				Header:     make(http.Header),
			}, nil
		}
		if strings.Contains(req.URL.Path, "appdetails") {
			var jsonResp string
			if strings.Contains(req.URL.RawQuery, "appids=456") {
				jsonResp = `{"456": {"success": true, "data": {"name": "DLC Expansion Pack 1"}}}`
			} else if strings.Contains(req.URL.RawQuery, "appids=789") {
				jsonResp = `{"789": {"success": true, "data": {"name": "DLC Expansion Pack 2"}}}`
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
				Header:     make(http.Header),
			}, nil
		}
		return nil, nil
	})

	err = FetchDLCs(context.Background(), "123", tmpDir, false)
	if err != nil {
		t.Fatalf("FetchDLCs failed: %v", err)
	}

	// Verify steam_appid.txt
	appidContent, err := os.ReadFile(filepath.Join(tmpDir, "steam_appid.txt"))
	if err != nil {
		t.Fatalf("failed to read steam_appid.txt: %v", err)
	}
	if string(appidContent) != "123" {
		t.Errorf("expected steam_appid.txt content to be '123', got '%s'", string(appidContent))
	}

	// Verify configs.app.ini
	iniPath := filepath.Join(tmpDir, "steam_settings", "configs.app.ini")
	iniContent, err := os.ReadFile(iniPath)
	if err != nil {
		t.Fatalf("failed to read configs.app.ini: %v", err)
	}

	iniStr := string(iniContent)
	if !strings.Contains(iniStr, "[app::dlcs]") {
		t.Error("expected configs.app.ini to contain [app::dlcs] section")
	}
	if !strings.Contains(iniStr, "456=DLC Expansion Pack 1") {
		t.Error("expected configs.app.ini to contain DLC 456 definition")
	}
	if !strings.Contains(iniStr, "789=DLC Expansion Pack 2") {
		t.Error("expected configs.app.ini to contain DLC 789 definition")
	}
}

func TestDownloadAchievementImages(t *testing.T) {
	oldTransport := http.DefaultClient.Transport
	defer func() {
		http.DefaultClient.Transport = oldTransport
	}()

	tmpDir, err := os.MkdirTemp("", "testachievements")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	http.DefaultClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "ach1.png") {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("fake image data")),
				Header:     make(http.Header),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
			Header:     make(http.Header),
		}, nil
	})

	err = DownloadAchievementImages(context.Background(), 123, []string{"ach1.png", "ach2.png"}, tmpDir)
	if err != nil {
		t.Fatalf("DownloadAchievementImages failed: %v", err)
	}

	// Verify ach1.png exists and has correct content
	imgContent, err := os.ReadFile(filepath.Join(tmpDir, "ach1.png"))
	if err != nil {
		t.Fatalf("failed to read ach1.png: %v", err)
	}
	if string(imgContent) != "fake image data" {
		t.Errorf("expected ach1.png content to be 'fake image data', got '%s'", string(imgContent))
	}

	// Verify ach2.png does not exist (failed download)
	_, err = os.Stat(filepath.Join(tmpDir, "ach2.png"))
	if !os.IsNotExist(err) {
		t.Error("expected ach2.png to not exist since download was mocked to return 404")
	}
}
