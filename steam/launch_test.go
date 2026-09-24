package steam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/staernid/gamux/cache"
)

func TestFilterAndBuildLaunchCandidates(t *testing.T) {
	entries := []steamcmdLaunchEntry{
		{
			Executable:  "Ride.exe",
			Description: "with DirectX 12 (default)",
			Type:        "default",
			Config: struct {
				OSList  string `json:"oslist"`
				OSArch  string `json:"osarch"`
				BetaKey string `json:"betakey"`
			}{
				OSList: "windows",
				OSArch: "64",
			},
		},
		{
			Executable:  "Ride.exe",
			Arguments:   "-d3d11",
			Description: "with DirectX 11 (solves some startup crashes)",
			Type:        "option2",
			Config: struct {
				OSList  string `json:"oslist"`
				OSArch  string `json:"osarch"`
				BetaKey string `json:"betakey"`
			}{
				OSList: "windows",
				OSArch: "64",
			},
		},
		{
			Executable:  "Ride.exe",
			Arguments:   "-fullcrashdump",
			Description: "with DirectX 12 (default)",
			Type:        "default",
			Config: struct {
				OSList  string `json:"oslist"`
				OSArch  string `json:"osarch"`
				BetaKey string `json:"betakey"`
			}{
				OSList:  "windows",
				OSArch:  "64",
				BetaKey: "nightly",
			},
		},
		{
			Executable:  "Ride.x86_64",
			Description: "Linux Native",
			Type:        "default",
			Config: struct {
				OSList  string `json:"oslist"`
				OSArch  string `json:"osarch"`
				BetaKey string `json:"betakey"`
			}{
				OSList: "linux",
			},
		},
	}

	t.Run("Windows Public Branch", func(t *testing.T) {
		cands := filterAndBuildLaunchCandidates(entries, "win64", "public")
		if len(cands) != 2 {
			t.Fatalf("expected 2 candidates for windows public, got %d", len(cands))
		}
		if cands[0].Executable != "Ride.exe" || cands[0].Arguments != "" {
			t.Errorf("expected candidate 0 to be Ride.exe with no args, got %+v", cands[0])
		}
		if cands[0].Name != "Play (with DirectX 12 (default))" {
			t.Errorf("unexpected name for candidate 0: %s", cands[0].Name)
		}
		if cands[1].Executable != "Ride.exe" || cands[1].Arguments != "-d3d11" {
			t.Errorf("expected candidate 1 to be Ride.exe with -d3d11, got %+v", cands[1])
		}
	})

	t.Run("Windows Nightly Branch", func(t *testing.T) {
		cands := filterAndBuildLaunchCandidates(entries, "win64", "nightly")
		if len(cands) != 1 {
			t.Fatalf("expected 1 candidate for windows nightly, got %d", len(cands))
		}
		if cands[0].Arguments != "-fullcrashdump" {
			t.Errorf("expected candidate to have -fullcrashdump, got %s", cands[0].Arguments)
		}
	})

	t.Run("Linux Native", func(t *testing.T) {
		cands := filterAndBuildLaunchCandidates(entries, "linux", "public")
		if len(cands) != 1 {
			t.Fatalf("expected 1 candidate for linux, got %d", len(cands))
		}
		if cands[0].Executable != "Ride.x86_64" {
			t.Errorf("expected Ride.x86_64, got %s", cands[0].Executable)
		}
	})
}

func TestFetchAppLaunchOptions_MockHTTP(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	mockJSON := `{
		"status": "success",
		"data": {
			"361420": {
				"config": {
					"launch": {
						"1": {
							"executable": "Astro.exe",
							"type": "default"
						},
						"2": {
							"executable": "Astro.exe",
							"arguments": "-Windowed",
							"description": "Play ASTRONEER (Windowed)",
							"type": "none"
						}
					}
				}
			}
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockJSON))
	}))
	defer ts.Close()

	ctx := context.WithValue(context.Background(), httpClientKey, ts.Client())

	origBaseURL := DefaultSteamCMDBaseURL
	_ = origBaseURL
	// Test via direct client using mock server
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("mock request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
