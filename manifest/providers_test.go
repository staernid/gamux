package manifest

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
)

func TestFetchRevoBDKeys(t *testing.T) {
	oldTransport := HTTPClient.Transport
	defer func() {
		HTTPClient.Transport = oldTransport
	}()

	HTTPClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
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
	})

	parsed, err := FetchRevoBDKeys(context.Background(), 2088570)
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
	oldTransport := HTTPClient.Transport
	defer func() {
		HTTPClient.Transport = oldTransport
	}()

	HTTPClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
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
	})

	parsed, err := FetchSteamDepotDBKeys(context.Background(), 2088570)
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
