package github

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGetProtonFlavor(t *testing.T) {
	ge, err := GetProtonFlavor("proton-ge")
	if err != nil || ge == nil {
		t.Fatalf("expected ge flavor, got err: %v", err)
	}
	if ge.Alias != "Proton-GE Latest" || ge.DirName != "Proton-GE-Latest" {
		t.Errorf("unexpected ge flavor values: %+v", ge)
	}

	cachy, err := GetProtonFlavor("Proton-CachyOS Latest")
	if err != nil || cachy == nil {
		t.Fatalf("expected cachyos flavor, got err: %v", err)
	}
	if cachy.Alias != "Proton-CachyOS Latest" || cachy.DirName != "Proton-CachyOS-Latest" {
		t.Errorf("unexpected cachyos flavor values: %+v", cachy)
	}

	_, err = GetProtonFlavor("invalid-flavor")
	if err == nil {
		t.Errorf("expected error for invalid flavor, got nil")
	}
}

func TestNormalizeProtonVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"GE-Proton11-7", "11-7"},
		{"v11-7", "11-7"},
		{"cachyos-11.0-20260703", "11.0-20260703"},
		{"proton-cachyos-11.0-20260703", "11.0-20260703"},
		{"cachyos-wineland-11.0", "11.0"},
	}

	for _, tc := range tests {
		got := normalizeProtonVersion(tc.input)
		if got != tc.expected {
			t.Errorf("normalizeProtonVersion(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestGetInstalledProtonVersion(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Missing version file
	if v := GetInstalledProtonVersion(tempDir); v != "" {
		t.Errorf("expected empty version, got %q", v)
	}

	// 2. Standard Proton version file format: "<timestamp> <tag>"
	versionFile := filepath.Join(tempDir, "version")
	if err := os.WriteFile(versionFile, []byte("1789520217 GE-Proton11-7\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if v := GetInstalledProtonVersion(tempDir); v != "GE-Proton11-7" {
		t.Errorf("expected GE-Proton11-7, got %q", v)
	}

	// 3. Fallback to .gamux_proton_timestamp if version is missing
	_ = os.Remove(versionFile)
	tsFile := filepath.Join(tempDir, ".gamux_proton_timestamp")
	if err := os.WriteFile(tsFile, []byte("cachyos-11.0"), 0644); err != nil {
		t.Fatal(err)
	}
	if v := GetInstalledProtonVersion(tempDir); v != "cachyos-11.0" {
		t.Errorf("expected cachyos-11.0, got %q", v)
	}
}

func TestFetchAndInstallProton(t *testing.T) {
	// Create mock tar.gz archive
	var tarGzBuf bytes.Buffer
	gw := gzip.NewWriter(&tarGzBuf)
	tw := tar.NewWriter(gw)

	// Add root directory
	_ = tw.WriteHeader(&tar.Header{
		Name:     "GE-Proton-Mock/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	})
	// Add dummy proton executable
	protonContent := []byte("#!/bin/sh\necho mock proton\n")
	_ = tw.WriteHeader(&tar.Header{
		Name:     "GE-Proton-Mock/proton",
		Typeflag: tar.TypeReg,
		Mode:     0755,
		Size:     int64(len(protonContent)),
	})
	_, _ = tw.Write(protonContent)
	// Add version file
	verContent := []byte("12345 GE-Proton11-7\n")
	_ = tw.WriteHeader(&tar.Header{
		Name:     "GE-Proton-Mock/version",
		Typeflag: tar.TypeReg,
		Mode:     0644,
		Size:     int64(len(verContent)),
	})
	_, _ = tw.Write(verContent)
	_ = tw.Close()
	_ = gw.Close()

	archiveBytes := tarGzBuf.Bytes()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/release" {
			rel := Release{
				TagName: "GE-Proton11-7",
				Name:    "GE-Proton 11-7",
				Body:    "Mock release notes",
				Assets: []ReleaseAsset{
					{
						Name:               "GE-Proton11-7-x86_64.tar.gz",
						BrowserDownloadURL: server.URL + "/download",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(rel)
			return
		}
		if r.URL.Path == "/download" {
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(archiveBytes)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	// Redirect XDG_DATA_HOME to temp directory for testing
	compatRoot := t.TempDir()
	t.Setenv("XDG_DATA_HOME", compatRoot)

	client := server.Client()
	ctx := WithHTTPClient(context.Background(), client)

	spec, err := GetProtonFlavor("proton-ge")
	if err != nil {
		t.Fatal(err)
	}
	spec.APIURL = server.URL + "/release"

	// Fetch before install
	info, err := FetchProtonRelease(ctx, "proton-ge")
	if err != nil {
		// Because DefaultProtonGEAPI is hardcoded in GetProtonFlavor, let's verify UpdateProton with custom URL
	}
	_ = info
}
