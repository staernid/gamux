package manifest

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
)

type mockRoundTripper func(req *http.Request) (*http.Response, error)

func (f mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetchHubcapKeys_LuaFormat(t *testing.T) {
	oldTransport := HTTPClient.Transport
	defer func() {
		HTTPClient.Transport = oldTransport
	}()

	HTTPClient.Transport = mockRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "Bearer MOCK_HUBCAP_KEY" {
			t.Errorf("expected Authorization header Bearer MOCK_HUBCAP_KEY, got %s", req.Header.Get("Authorization"))
		}
		luaBody := `addappid(2088570, 1, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(luaBody)),
			Header:     make(http.Header),
		}, nil
	})

	parsed, err := FetchHubcapKeys(context.Background(), 2088570, "MOCK_HUBCAP_KEY")
	if err != nil {
		t.Fatalf("FetchHubcapKeys failed: %v", err)
	}

	if parsed.AppID != 2088570 {
		t.Errorf("expected AppID 2088570, got %d", parsed.AppID)
	}
	if len(parsed.Depots) != 1 {
		t.Fatalf("expected 1 depot, got %d", len(parsed.Depots))
	}
	if parsed.Depots[0].DecryptionKey != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Errorf("unexpected key: %s", parsed.Depots[0].DecryptionKey)
	}
}

func TestResolveKeys_Priority(t *testing.T) {
	tmpDir := t.TempDir()
	luaFile := tmpDir + "/test.lua"
	luaContent := `addappid(99999, 1, "1111111111111111111111111111111111111111111111111111111111111111")`

	if err := os.WriteFile(luaFile, []byte(luaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Local file priority
	parsed, err := ResolveKeys(context.Background(), 99999, luaFile, "HUBCAPKEY")
	if err != nil {
		t.Fatalf("ResolveKeys failed: %v", err)
	}
	if parsed.AppID != 99999 {
		t.Errorf("expected AppID 99999, got %d", parsed.AppID)
	}

	// 2. Error when neither is supplied
	_, err = ResolveKeys(context.Background(), 0, "", "")
	if err == nil {
		t.Error("expected error when neither lua nor hubcap key is present")
	}
}
