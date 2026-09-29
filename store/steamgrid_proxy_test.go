package store

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSteamGridProxyProvider_FetchCoverURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search/Terraria" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`"https://cdn2.steamgriddb.com/grid/terraria.png"`))
			return
		}
		if r.URL.Path == "/search/UnknownGame" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("404"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	provider := NewSteamGridProxyProvider().WithCustomClient(server.Client(), server.URL+"/search")
	ctx := context.Background()

	// 1. Success case
	url, err := provider.FetchCoverURL(ctx, "Terraria")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://cdn2.steamgriddb.com/grid/terraria.png"
	if url != expected {
		t.Errorf("got %q, expected %q", url, expected)
	}

	// 2. Not found case
	_, err = provider.FetchCoverURL(ctx, "UnknownGame")
	if err == nil {
		t.Errorf("expected 404 error, got nil")
	}

	// 3. QueryArtworkCandidates
	opts, err := provider.QueryArtworkCandidates(ctx, ArtworkQuery{
		Title:     "Terraria",
		AssetType: "cover",
	})
	if err != nil {
		t.Fatalf("QueryArtworkCandidates failed: %v", err)
	}
	if len(opts) != 1 {
		t.Fatalf("expected 1 option, got %d", len(opts))
	}
	if opts[0].URL != expected {
		t.Errorf("expected URL %q, got %q", expected, opts[0].URL)
	}

	// 4. Non-cover asset type returns empty list
	optsBanner, err := provider.QueryArtworkCandidates(ctx, ArtworkQuery{
		Title:     "Terraria",
		AssetType: "banner",
	})
	if err != nil {
		t.Fatalf("unexpected error for banner query: %v", err)
	}
	if len(optsBanner) != 0 {
		t.Errorf("expected 0 options for banner, got %d", len(optsBanner))
	}
}
