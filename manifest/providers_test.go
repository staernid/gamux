package manifest

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/staernid/gamux/cache"
)

func TestFetchRevoBDKeys(t *testing.T) {
	cache.SetCacheDir(t.TempDir())
	mockClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			buf := new(bytes.Buffer)
			w := zip.NewWriter(buf)
			f, _ := w.Create("2088570.lua")
			f.Write([]byte(`addappid(2088570)
addappid(2088571, 1, "122e3270335372157a11a595038524e4cd4bfd7b2b9851c40ca5c25be3f85914")`))
			w.Close()

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	parsed, err := FetchRevoBDKeys(ctx, 2088570)
	if err != nil {
		t.Fatalf("FetchRevoBDKeys failed: %v", err)
	}

	if parsed.AppID != 2088570 {
		t.Errorf("expected AppID 2088570, got %d", parsed.AppID)
	}
	if len(parsed.Depots) == 0 {
		t.Fatalf("expected at least 1 depot, got 0")
	}
	foundKey := false
	for _, d := range parsed.Depots {
		if d.DecryptionKey == "122e3270335372157a11a595038524e4cd4bfd7b2b9851c40ca5c25be3f85914" {
			foundKey = true
			break
		}
	}
	if !foundKey {
		t.Errorf("expected decryption key not found in parsed depots")
	}
}

func TestFetchSteamDepotDBKeys(t *testing.T) {
	cache.SetCacheDir(t.TempDir())
	mockClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			jsonResp := `{
				"2088571": {
					"key": "122e3270335372157a11a595038524e4cd4bfd7b2b9851c40ca5c25be3f85914",
					"parent_appid": "2088570"
				}
			}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	parsed, err := FetchSteamDepotDBKeys(ctx, 2088570)
	if err != nil {
		t.Fatalf("FetchSteamDepotDBKeys failed: %v", err)
	}

	if len(parsed.Depots) != 1 {
		t.Fatalf("expected 1 depot, got %d", len(parsed.Depots))
	}
	if parsed.Depots[0].DepotID != 2088571 {
		t.Errorf("expected depot 2088571, got %d", parsed.Depots[0].DepotID)
	}
}

func TestFetchManifesthubKeys(t *testing.T) {
	cache.SetCacheDir(t.TempDir())
	mockClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			jsonResp := `{
				"1313141": "e7826851aae21726cd847e809a97378d381ba16449186ce1cdb57c572d703460",
				"1313142": "242e4da3afdadce6aa2329f9f1c8c71fe09c06f51a80c105a444734a12c9e3f2",
				"9999991": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	parsed, err := FetchManifesthubKeys(ctx, 1313140)
	if err != nil {
		t.Fatalf("FetchManifesthubKeys failed: %v", err)
	}

	if len(parsed.Depots) != 2 {
		t.Fatalf("expected 2 depots for AppID 1313140, got %d", len(parsed.Depots))
	}
	for _, d := range parsed.Depots {
		if d.DepotID != 1313141 && d.DepotID != 1313142 {
			t.Errorf("unexpected depot ID: %d", d.DepotID)
		}
	}
}

func TestFetchGMRCRequestCode(t *testing.T) {
	mockClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("1827391029384756")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	code, err := FetchGMRCRequestCode(ctx, "987654321", "")
	if err != nil {
		t.Fatalf("FetchGMRCRequestCode failed: %v", err)
	}
	if code != "1827391029384756" {
		t.Errorf("expected request code '1827391029384756', got '%s'", code)
	}
}

func TestDownloadManifestFromSteamCDN(t *testing.T) {
	expectedManifestBytes := []byte("raw_binary_depot_manifest_content")

	mockClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			// Package inside a zip container to test ZIP unpacking
			buf := new(bytes.Buffer)
			w := zip.NewWriter(buf)
			f, _ := w.Create("1313141_987654321.manifest")
			f.Write(expectedManifestBytes)
			w.Close()

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
				Header:     make(http.Header),
			}, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	data, err := DownloadManifestFromSteamCDN(ctx, 1313141, "987654321", "1827391029384756", "test.steamcontent.com")
	if err != nil {
		t.Fatalf("DownloadManifestFromSteamCDN failed: %v", err)
	}
	if !bytes.Equal(data, expectedManifestBytes) {
		t.Errorf("expected manifest content '%s', got '%s'", string(expectedManifestBytes), string(data))
	}
}
