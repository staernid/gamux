package steam

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

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
	restore := testutil.SilenceLogging()
	code := m.Run()
	restore()
	os.Exit(code)
}




func TestFetchAppName(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	t.Run("successful response", func(t *testing.T) {
		mockClient := &http.Client{
			Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if !strings.Contains(req.URL.Path, "appdetails") {
					t.Errorf("unexpected URL path: %s", req.URL.Path)
				}
				jsonResp := `{"123": {"success": true, "data": {"name": "Super Tux Run"}}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
					Header:     make(http.Header),
				}, nil
			}),
		}

		ctx := WithHTTPClient(context.Background(), mockClient)
		name, err := FetchAppName(ctx, "123")
		if err != nil {
			t.Fatalf("FetchAppName failed: %v", err)
		}
		if name != "Super Tux Run" {
			t.Errorf("expected name 'Super Tux Run', got '%s'", name)
		}
	})

	t.Run("rate limit HTTP 429", func(t *testing.T) {
		mockClient := &http.Client{
			Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusTooManyRequests,
					Body:       io.NopCloser(bytes.NewBufferString("Too Many Requests")),
					Header:     make(http.Header),
				}, nil
			}),
		}

		ctx := WithHTTPClient(context.Background(), mockClient)
		_, err := FetchAppName(ctx, "123")
		if err == nil {
			t.Fatal("expected error due to HTTP 429, got nil")
		}
		if !strings.Contains(err.Error(), "HTTP 429") {
			t.Errorf("expected error message to contain 'HTTP 429', got: %v", err)
		}
	})

	t.Run("release_date as object", func(t *testing.T) {
		mockClient := &http.Client{
			Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				jsonResp := `{"3711820": {"success": true, "data": {"name": "Blue Prince DLC", "release_date": {"coming_soon": false, "date": "1 Aug, 2026"}}}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
					Header:     make(http.Header),
				}, nil
			}),
		}

		ctx := WithHTTPClient(context.Background(), mockClient)
		name, err := FetchAppName(ctx, "3711820")
		if err != nil {
			t.Fatalf("FetchAppName failed: %v", err)
		}
		if name != "Blue Prince DLC" {
			t.Errorf("expected name 'Blue Prince DLC', got '%s'", name)
		}
	})
}

func TestFetchDLCs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "testfetchdlcs")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "ajaxgetfilteredrecommendations") {
				htmlResp := `{"results_html": "<div data-ds-appid=\"456\"></div><div data-ds-appid=\"789\"></div>"}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(htmlResp)),
					Header:     make(http.Header),
				}, nil
			}
			if strings.Contains(req.URL.Path, "appdetails") {
				jsonResp := `{"456": {"success": true, "data": {"name": "DLC Expansion Pack 1"}}, "789": {"success": true, "data": {"name": "DLC Expansion Pack 2"}}}`
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
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	err = FetchDLCs(ctx, "123", tmpDir, false)
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

func TestFetchDLCs_Structured(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	tmpDir, err := os.MkdirTemp("", "testfetchdlcs_structured")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "ajaxgetfilteredrecommendations") {
				t.Fatalf("Structured DLC discovery should not call legacy ajax recommendations endpoint!")
			}
			if strings.Contains(req.URL.Path, "appdetails") {
				if strings.Contains(req.URL.RawQuery, "appids=846770") {
					jsonResp := `{"846770": {"success": true, "data": {"name": "Dysmantle", "dlc": [101, 102]}}}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
						Header:     make(http.Header),
					}, nil
				}
				if strings.Contains(req.URL.RawQuery, "appids=101") {
					jsonResp := `{"101": {"success": true, "data": {"name": "Underworld DLC"}}}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
						Header:     make(http.Header),
					}, nil
				}
				if strings.Contains(req.URL.RawQuery, "appids=102") {
					jsonResp := `{"102": {"success": true, "data": {"name": "Doomsday DLC"}}}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
						Header:     make(http.Header),
					}, nil
				}
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	err = FetchDLCs(ctx, "846770", tmpDir, false)
	if err != nil {
		t.Fatalf("FetchDLCs failed: %v", err)
	}

	iniPath := filepath.Join(tmpDir, "steam_settings", "configs.app.ini")
	iniContent, err := os.ReadFile(iniPath)
	if err != nil {
		t.Fatalf("failed to read configs.app.ini: %v", err)
	}

	iniStr := string(iniContent)
	if !strings.Contains(iniStr, "101=Underworld DLC") {
		t.Errorf("expected 101=Underworld DLC, got:\n%s", iniStr)
	}
	if !strings.Contains(iniStr, "102=Doomsday DLC") {
		t.Errorf("expected 102=Doomsday DLC, got:\n%s", iniStr)
	}
}

func TestFetchDLCs_PreservesExistingSections(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "testdlc_preserve")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	settingsDir := filepath.Join(tmpDir, "steam_settings")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatal(err)
	}
	existingIni := `# App general config
[app::general]
enable_lan_only=1

[app::custom]
custom_param=abc
`
	if err := os.WriteFile(filepath.Join(settingsDir, "configs.app.ini"), []byte(existingIni), 0644); err != nil {
		t.Fatal(err)
	}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "ajaxgetfilteredrecommendations") {
				htmlResp := `{"results_html": "<div data-ds-appid=\"456\"></div>"}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(htmlResp)),
					Header:     make(http.Header),
				}, nil
			}
			if strings.Contains(req.URL.Path, "appdetails") {
				jsonResp := `{"456": {"success": true, "data": {"name": "DLC Expansion Pack 1"}}}`
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
	if err := FetchDLCs(ctx, "123", tmpDir, false); err != nil {
		t.Fatalf("FetchDLCs failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(settingsDir, "configs.app.ini"))
	if err != nil {
		t.Fatal(err)
	}
	iniStr := string(data)

	if !strings.Contains(iniStr, "# App general config") {
		t.Errorf("comment in configs.app.ini lost:\n%s", iniStr)
	}
	if !strings.Contains(iniStr, "[app::general]") || !strings.Contains(iniStr, "enable_lan_only=1") {
		t.Errorf("[app::general] section lost:\n%s", iniStr)
	}
	if !strings.Contains(iniStr, "[app::custom]") || !strings.Contains(iniStr, "custom_param=abc") {
		t.Errorf("[app::custom] section lost:\n%s", iniStr)
	}
	if !strings.Contains(iniStr, "[app::dlcs]") || !strings.Contains(iniStr, "456=DLC Expansion Pack 1") {
		t.Errorf("[app::dlcs] not added:\n%s", iniStr)
	}
}

func TestDownloadAchievementImages(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "testachievements")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
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
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	err = DownloadAchievementImages(ctx, 123, []string{"ach1.png", "ach2.png"}, tmpDir)
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

func TestSearchAppID(t *testing.T) {
	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "storesearch") {
				jsonResp := `{"total": 1, "items": [{"id": 2088570, "name": "Tiny Rogues"}]}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
					Header:     make(http.Header),
				}, nil
			}
			return nil, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	appID, name, err := SearchAppID(ctx, "Tiny Rogues")
	if err != nil {
		t.Fatalf("SearchAppID failed: %v", err)
	}
	if appID != "2088570" {
		t.Errorf("expected AppID 2088570, got %s", appID)
	}
	if name != "Tiny Rogues" {
		t.Errorf("expected Name 'Tiny Rogues', got %s", name)
	}
}

func TestFetchAchievementSchema(t *testing.T) {
	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "GetSchemaForGame") {
				jsonResp := `{
					"game": {
						"availableGameStats": {
							"achievements": [
								{
									"name": "ACH_KILL_10",
									"displayName": "Slayer",
									"description": "Kill 10 enemies",
									"hidden": 0,
									"icon": "https://cdn.steam.com/img/icon_slayer.png",
									"icongray": "https://cdn.steam.com/img/icon_slayer_gray.png"
								}
							]
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
	achs, err := FetchAchievementSchema(ctx, 12345, "MOCKKEY", "english")
	if err != nil {
		t.Fatalf("FetchAchievementSchema failed: %v", err)
	}

	if len(achs) != 1 {
		t.Fatalf("expected 1 achievement, got %d", len(achs))
	}
	if achs[0].Name != "ACH_KILL_10" {
		t.Errorf("expected name ACH_KILL_10, got %s", achs[0].Name)
	}
	if achs[0].Icon != "img/icon_slayer.png" {
		t.Errorf("expected Icon img/icon_slayer.png, got %s", achs[0].Icon)
	}
}

func TestExtractWorkshopID(t *testing.T) {
	id, err := ExtractWorkshopID("https://steamcommunity.com/sharedfiles/filedetails/?id=315783921")
	if err != nil {
		t.Fatalf("ExtractWorkshopID failed: %v", err)
	}
	if id != 315783921 {
		t.Errorf("expected ID 315783921, got %d", id)
	}

	id, err = ExtractWorkshopID("315783921")
	if err != nil {
		t.Fatalf("ExtractWorkshopID raw ID failed: %v", err)
	}
	if id != 315783921 {
		t.Errorf("expected ID 315783921, got %d", id)
	}
}

func TestFetchWorkshopItemDetails(t *testing.T) {
	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "GetPublishedFileDetails") {
				jsonResp := `{
					"response": {
						"resultcount": 1,
						"publishedfiledetails": [
							{
								"publishedfileid": "315783921",
								"result": 1,
								"title": "Awesome Mod",
								"hcontent_file": "1234567890",
								"consumer_app_id": 2088570
							}
						]
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
	details, err := FetchWorkshopItemDetails(ctx, 315783921)
	if err != nil {
		t.Fatalf("FetchWorkshopItemDetails failed: %v", err)
	}

	if details.Title != "Awesome Mod" {
		t.Errorf("expected title 'Awesome Mod', got %s", details.Title)
	}
	if details.HContentFile != "1234567890" {
		t.Errorf("expected hcontent_file '1234567890', got %s", details.HContentFile)
	}
}

func TestGetSupportedLanguageCodes(t *testing.T) {
	details := &AppDetails{
		SupportedLanguages: "English<strong>*</strong>, French<strong>*</strong>, German<strong>*</strong>, Spanish - Spain<strong>*</strong>, Japanese<strong>*</strong>, Simplified Chinese<strong>*</strong><br><strong>*</strong>languages with full audio support",
	}

	codes := details.GetSupportedLanguageCodes()
	expected := []string{"english", "french", "german", "spanish", "japanese", "schinese"}

	if len(codes) != len(expected) {
		t.Fatalf("expected %d codes, got %d: %v", len(expected), len(codes), codes)
	}

	for i, exp := range expected {
		if codes[i] != exp {
			t.Errorf("at index %d expected %s, got %s", i, exp, codes[i])
		}
	}

	// Empty fallback
	emptyDetails := &AppDetails{}
	emptyCodes := emptyDetails.GetSupportedLanguageCodes()
	if len(emptyCodes) != 1 || emptyCodes[0] != "english" {
		t.Errorf("expected ['english'] for empty details, got %v", emptyCodes)
	}
}

func TestFetchAppNewsItems_CacheUpgrade(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpDir)
	cache.SetDisabled(false)
	defer cache.SetDisabled(true)

	// First, simulate a cached 1-item news response on disk
	singleItemJSON := []byte(`[{"title":"Update 1","url":"https://steam.com/1","author":"Dev","contents":"Bugfix","feedlabel":"News","date":1700000000}]`)
	cache.SaveNewsItems("12345", singleItemJSON)

	// Verify the cache has 1 item
	cachedData, ok := cache.GetNewsItems("12345")
	if !ok || len(cachedData) == 0 {
		t.Fatalf("expected cache to have saved items")
	}

	// Now set up mock HTTP client returning 3 news items from Steam API
	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "GetNewsForApp") {
				jsonResp := `{
					"appnews": {
						"appid": 12345,
						"newsitems": [
							{"title": "Update 1", "url": "https://steam.com/1", "author": "Dev", "contents": "Bugfix 1", "feedlabel": "News", "date": 1700000000},
							{"title": "Update 2", "url": "https://steam.com/2", "author": "Dev", "contents": "Bugfix 2", "feedlabel": "News", "date": 1700001000},
							{"title": "Update 3", "url": "https://steam.com/3", "author": "Dev", "contents": "Bugfix 3", "feedlabel": "News", "date": 1700002000}
						]
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

	// Requesting count = 3 must NOT return only 1 item; it must fetch from API and return 3 items
	items, err := FetchAppNewsItems(ctx, "12345", 3)
	if err != nil {
		t.Fatalf("FetchAppNewsItems failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 news items, got %d", len(items))
	}

	// Verify that the disk cache was upgraded and now contains 3 items
	upgradedBytes, ok := cache.GetNewsItems("12345")
	if !ok {
		t.Fatalf("expected upgraded cache to exist")
	}
	var upgradedItems []NewsItem
	if err := json.Unmarshal(upgradedBytes, &upgradedItems); err != nil {
		t.Fatalf("unmarshal upgraded cache failed: %v", err)
	}
	if len(upgradedItems) < 3 {
		t.Errorf("expected upgraded cache to have at least 3 items, got %d", len(upgradedItems))
	}

	// Verify that subsequent SaveNewsItems with 1 item does NOT downgrade the cache
	cache.SaveNewsItems("12345", singleItemJSON)
	protectedBytes, _ := cache.GetNewsItems("12345")
	var protectedItems []NewsItem
	_ = json.Unmarshal(protectedBytes, &protectedItems)
	if len(protectedItems) < 3 {
		t.Errorf("cache was downgraded! expected at least 3 items, got %d", len(protectedItems))
	}
}

