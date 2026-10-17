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

func TestCheckGameVersion_CustomBetaBranch(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/v1/info/440") {
				jsonResp := `{
					"status": "success",
					"data": {
						"440": {
							"depots": {
								"branches": {
									"public": {
										"buildid": "1000",
										"timeupdated": "1700000000"
									},
									"beta": {
										"buildid": "1200",
										"timeupdated": "1700500000"
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
	// Local build is 1000. If checking "beta", target build is 1200 -> outdated.
	status, err := CheckGameVersion(ctx, 440, "1000", 1700000000, "beta")
	if err != nil {
		t.Fatalf("CheckGameVersion failed: %v", err)
	}

	if !status.Checked {
		t.Errorf("expected status.Checked == true")
	}
	if status.IsUpToDate {
		t.Errorf("expected status.IsUpToDate == false for outdated beta branch")
	}
	if status.LatestBuildID != "1200" {
		t.Errorf("expected LatestBuildID '1200', got %q", status.LatestBuildID)
	}
}

func TestCheckGameVersion_DepotManifest_Outdated(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/v1/info/2067820") {
				jsonResp := `{
					"status": "success",
					"data": {
						"2067820": {
							"depots": {
								"2067821": {
									"manifests": {
										"public": {
											"gid": "5345558827008687336"
										}
									}
								},
								"branches": {
									"public": {
										"buildid": "25753711",
										"timeupdated": "1791370841"
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
	// Local buildid is "0", but installed manifest GID is "622701312170627612" (differs from remote "5345558827008687336")
	status, err := CheckGameVersion(ctx, 2067820, "0", 1700000000, "public", "622701312170627612")
	if err != nil {
		t.Fatalf("CheckGameVersion failed: %v", err)
	}

	if !status.Checked {
		t.Errorf("expected status.Checked == true for known depot manifest")
	}
	if status.IsUpToDate {
		t.Errorf("expected status.IsUpToDate == false (outdated manifest)")
	}
	if status.LatestBuildID != "25753711" {
		t.Errorf("expected LatestBuildID '25753711', got %q", status.LatestBuildID)
	}
	if status.RemoteManifestID != "5345558827008687336" {
		t.Errorf("expected RemoteManifestID '5345558827008687336', got %q", status.RemoteManifestID)
	}
}

func TestCheckGameVersion_DepotManifest_UpToDate(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/v1/info/2067820") {
				jsonResp := `{
					"status": "success",
					"data": {
						"2067820": {
							"depots": {
								"2067821": {
									"manifests": {
										"public": {
											"gid": "5345558827008687336"
										}
									}
								},
								"branches": {
									"public": {
										"buildid": "25753711",
										"timeupdated": "1791370841"
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
	// Local buildid is "0", but installed manifest GID matches remote "5345558827008687336"
	status, err := CheckGameVersion(ctx, 2067820, "0", 1700000000, "public", "5345558827008687336")
	if err != nil {
		t.Fatalf("CheckGameVersion failed: %v", err)
	}

	if !status.Checked {
		t.Errorf("expected status.Checked == true for known depot manifest")
	}
	if !status.IsUpToDate {
		t.Errorf("expected status.IsUpToDate == true (matching manifest)")
	}
	if status.LatestBuildID != "25753711" {
		t.Errorf("expected LatestBuildID '25753711', got %q", status.LatestBuildID)
	}
}
