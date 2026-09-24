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

func TestCheckGameVersion_SteamCMD_UpToDate(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

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
			if strings.Contains(req.URL.Path, "/v1/info/846770") {
				jsonResp := `{
					"status": "success",
					"data": {
						"846770": {
							"depots": {
								"branches": {
									"public": {
										"buildid": "20918527",
										"timeupdated": "1789000000"
									}
								}
							}
						}
					}
				}`
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
	status, err := CheckGameVersion(ctx, 846770, "20918527", 1700000000)
	if err != nil {
		t.Fatalf("CheckGameVersion failed: %v", err)
	}

	if !status.Checked {
		t.Errorf("expected status.Checked == true")
	}
	if !status.IsUpToDate {
		t.Errorf("expected status.IsUpToDate == true")
	}
	if status.LatestBuildID != "20918527" {
		t.Errorf("expected LatestBuildID '20918527', got %q", status.LatestBuildID)
	}
}

func TestCheckGameVersion_SteamCMD_Outdated(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

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
			if strings.Contains(req.URL.Path, "/v1/info/846770") {
				jsonResp := `{
					"status": "success",
					"data": {
						"846770": {
							"depots": {
								"branches": {
									"public": {
										"buildid": "20918527",
										"timeupdated": "1789000000"
									}
								}
							}
						}
					}
				}`
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
	status, err := CheckGameVersion(ctx, 846770, "20000000", 1700000000)
	if err != nil {
		t.Fatalf("CheckGameVersion failed: %v", err)
	}

	if !status.Checked {
		t.Errorf("expected status.Checked == true")
	}
	if status.IsUpToDate {
		t.Errorf("expected status.IsUpToDate == false")
	}
	if status.LatestBuildID != "20918527" {
		t.Errorf("expected LatestBuildID '20918527', got %q", status.LatestBuildID)
	}
	if !strings.Contains(status.Message, "New build available") {
		t.Errorf("expected message containing 'New build available', got %q", status.Message)
	}
}

func TestCheckGameVersion_UnpopulatedSentinel_BuildIDZero(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/v1/info/3949040") {
				jsonResp := `{
					"status": "success",
					"data": {
						"3949040": {
							"depots": {
								"branches": {
									"public": {
										"buildid": "25237455",
										"timeupdated": "1789065129"
									}
								}
							}
						}
					}
				}`
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
	// Local build ID is "0" (unpopulated sentinel)
	status, err := CheckGameVersion(ctx, 3949040, "0", 1788634189)
	if err != nil {
		t.Fatalf("CheckGameVersion failed: %v", err)
	}

	// AGENTS.md Core Tenet 4: unpopulated sentinels must NOT trigger mismatch alerts
	if status.Checked {
		t.Errorf("expected status.Checked == false for unpopulated sentinel")
	}
	if !status.IsUpToDate {
		t.Errorf("expected status.IsUpToDate == true (no false positive update alerts on sentinel)")
	}
	if status.LatestBuildID != "25237455" {
		t.Errorf("expected LatestBuildID '25237455', got %q", status.LatestBuildID)
	}
}

func TestCheckGameVersion_UnpopulatedSentinel_EmptyBuildID(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/v1/info/3949040") {
				jsonResp := `{
					"status": "success",
					"data": {
						"3949040": {
							"depots": {
								"branches": {
									"public": {
										"buildid": "25237455",
										"timeupdated": "1789065129"
									}
								}
							}
						}
					}
				}`
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
	// Local build ID is "" (unpopulated sentinel)
	status, err := CheckGameVersion(ctx, 3949040, "", 1788634189)
	if err != nil {
		t.Fatalf("CheckGameVersion failed: %v", err)
	}

	if status.Checked {
		t.Errorf("expected status.Checked == false for empty buildID sentinel")
	}
	if !status.IsUpToDate {
		t.Errorf("expected status.IsUpToDate == true")
	}
	if status.LatestBuildID != "25237455" {
		t.Errorf("expected LatestBuildID '25237455', got %q", status.LatestBuildID)
	}
}

func TestCheckGameVersion_NewsDoesNotTriggerOutdated(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	now := time.Now().Unix()
	installTime := now - (7 * 86400) // Installed 7 days ago
	patchTime := now - (2 * 86400)   // News item published 2 days ago

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "GetNewsForApp") {
				jsonResp := fmt.Sprintf(`{
					"appnews": {
						"newsitems": [
							{
								"title": "Amanita Design will be at The Triple-i Initiative on October 8th!",
								"url": "https://store.steampowered.com/news/123",
								"contents": "Announcement text",
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
	status, err := CheckGameVersion(ctx, 1049280, "0", installTime)
	if err != nil {
		t.Fatalf("CheckGameVersion failed: %v", err)
	}

	// News items must never mark a game as outdated
	if !status.IsUpToDate {
		t.Errorf("expected status.IsUpToDate == true; news announcements must never trigger outdated flag")
	}
	if status.LatestPatchTitle != "Amanita Design will be at The Triple-i Initiative on October 8th!" {
		t.Errorf("expected latest patch title to be recorded, got %q", status.LatestPatchTitle)
	}
}
