package steam

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/util/testutil"
)

func TestCheckGameVersion_Tier1_UpToDate(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "UpToDateCheck") {
				jsonResp := `{"response": {"success": true, "up_to_date": true, "version_is_listable": true, "required_version": 1000}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	status, err := CheckGameVersion(ctx, 440, "1000", 1700000000)
	if err != nil {
		t.Fatalf("CheckGameVersion failed: %v", err)
	}

	if !status.Checked {
		t.Errorf("expected status.Checked == true")
	}
	if !status.IsUpToDate {
		t.Errorf("expected status.IsUpToDate == true")
	}
	if status.CurrentBuildID != "1000" {
		t.Errorf("expected CurrentBuildID '1000', got %q", status.CurrentBuildID)
	}
}

func TestCheckGameVersion_Tier1_Outdated(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "UpToDateCheck") {
				jsonResp := `{"response": {"success": true, "up_to_date": false, "version_is_listable": true, "required_version": 1050, "message": "A newer build is required"}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	status, err := CheckGameVersion(ctx, 440, "1000", 1700000000)
	if err != nil {
		t.Fatalf("CheckGameVersion failed: %v", err)
	}

	if !status.Checked {
		t.Errorf("expected status.Checked == true")
	}
	if status.IsUpToDate {
		t.Errorf("expected status.IsUpToDate == false")
	}
	if status.LatestBuildID != "1050" {
		t.Errorf("expected LatestBuildID '1050', got %q", status.LatestBuildID)
	}
}

func TestCheckGameVersion_Tier3_PatchNotesFallback_Outdated(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	now := time.Now().Unix()
	installTime := now - (7 * 86400) // Installed 7 days ago
	patchTime := now - (2 * 86400)   // Patch released 2 days ago (5 days after install)

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "UpToDateCheck") {
				jsonResp := `{"response": {"success": false, "error": "Couldn't get app info"}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
					Header:     make(http.Header),
				}, nil
			}
			if strings.Contains(req.URL.Path, "GetNewsForApp") {
				jsonResp := fmt.Sprintf(`{
					"appnews": {
						"newsitems": [
							{
								"title": "Major Patch v1.2 Released",
								"url": "https://store.steampowered.com/news/123",
								"contents": "Patch notes contents",
								"feedlabel": "Community Announcements",
								"date": %d
							}
						]
					}
				}`, patchTime)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	status, err := CheckGameVersion(ctx, 846770, "0", installTime)
	if err != nil {
		t.Fatalf("CheckGameVersion failed: %v", err)
	}

	if !status.Checked {
		t.Errorf("expected status.Checked == true")
	}
	if status.IsUpToDate {
		t.Errorf("expected status.IsUpToDate == false because patch is 5 days newer than install")
	}
	if status.LatestPatchTitle != "Major Patch v1.2 Released" {
		t.Errorf("expected latest patch title 'Major Patch v1.2 Released', got %q", status.LatestPatchTitle)
	}
}

func TestCheckGameVersion_Tier3_PatchNotesFallback_UpToDate(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	now := time.Now().Unix()
	installTime := now - (2 * 86400) // Installed 2 days ago
	patchTime := now - (7 * 86400)   // Patch released 7 days ago (older than install)

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "UpToDateCheck") {
				jsonResp := `{"response": {"success": false, "error": "Couldn't get app info"}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
					Header:     make(http.Header),
				}, nil
			}
			if strings.Contains(req.URL.Path, "GetNewsForApp") {
				jsonResp := fmt.Sprintf(`{
					"appnews": {
						"newsitems": [
							{
								"title": "Old Hotfix v1.1",
								"url": "https://store.steampowered.com/news/120",
								"contents": "Old notes",
								"feedlabel": "Community Announcements",
								"date": %d
							}
						]
					}
				}`, patchTime)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	status, err := CheckGameVersion(ctx, 846770, "0", installTime)
	if err != nil {
		t.Fatalf("CheckGameVersion failed: %v", err)
	}

	if !status.Checked {
		t.Errorf("expected status.Checked == true")
	}
	if !status.IsUpToDate {
		t.Errorf("expected status.IsUpToDate == true because install is newer than latest patch")
	}
}
