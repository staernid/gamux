package steam

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/util/testutil"
)

func TestCleanSteamProfileInput(t *testing.T) {
	cases := []struct {
		input       string
		expectedID  string
		expectedIs64 bool
	}{
		{"https://steamcommunity.com/id/gaben", "gaben", false},
		{"https://steamcommunity.com/id/gaben/", "gaben", false},
		{"http://steamcommunity.com/profiles/76561197960287930", "76561197960287930", true},
		{"76561197960287930", "76561197960287930", true},
		{"random_handle", "random_handle", false},
	}

	for _, c := range cases {
		id, is64 := CleanSteamProfileInput(c.input)
		if id != c.expectedID || is64 != c.expectedIs64 {
			t.Errorf("CleanSteamProfileInput(%q) = (%q, %v), expected (%q, %v)", c.input, id, is64, c.expectedID, c.expectedIs64)
		}
	}
}

func TestResolveSteamProfile(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	t.Run("resolve vanity url successfully", func(t *testing.T) {
		mockClient := &http.Client{
			Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if strings.Contains(req.URL.Path, "ResolveVanityURL") {
					resp := `{"response": {"steamid": "76561197960287930", "success": 1}}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(resp)),
						Header:     make(http.Header),
					}, nil
				}
				if strings.Contains(req.URL.Path, "GetPlayerSummaries") {
					resp := `{"response": {"players": [{"steamid": "76561197960287930", "personaname": "Gabe Newell", "profileurl": "https://steamcommunity.com/id/gaben/", "avatarfull": "https://avatars.steamstatic.com/gaben.jpg", "loccountrycode": "US"}]}}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(resp)),
						Header:     make(http.Header),
					}, nil
				}
				t.Fatalf("unexpected request: %s", req.URL.String())
				return nil, nil
			}),
		}

		ctx := WithHTTPClient(context.Background(), mockClient)
		prof, err := ResolveSteamProfile(ctx, "MOCK_KEY", "https://steamcommunity.com/id/gaben")
		if err != nil {
			t.Fatalf("ResolveSteamProfile failed: %v", err)
		}

		if prof.SteamID != "76561197960287930" {
			t.Errorf("expected SteamID 76561197960287930, got %s", prof.SteamID)
		}
		if prof.PersonaName != "Gabe Newell" {
			t.Errorf("expected PersonaName 'Gabe Newell', got %s", prof.PersonaName)
		}
		if prof.CountryCode != "US" {
			t.Errorf("expected CountryCode 'US', got %s", prof.CountryCode)
		}
		if prof.AvatarURL != "https://avatars.steamstatic.com/gaben.jpg" {
			t.Errorf("expected AvatarURL, got %s", prof.AvatarURL)
		}
	})

	t.Run("resolve raw steamid without api key", func(t *testing.T) {
		prof, err := ResolveSteamProfile(context.Background(), "", "76561197960287930")
		if err != nil {
			t.Fatalf("ResolveSteamProfile failed: %v", err)
		}
		if prof.SteamID != "76561197960287930" {
			t.Errorf("expected SteamID 76561197960287930, got %s", prof.SteamID)
		}
	})
}

func TestFetchCurrentPlayers(t *testing.T) {
	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "GetNumberOfCurrentPlayers") {
				resp := `{"response": {"player_count": 42100, "result": 1}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(resp)),
					Header:     make(http.Header),
				}, nil
			}
			t.Fatalf("unexpected request: %s", req.URL.String())
			return nil, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	count, err := FetchCurrentPlayers(ctx, 730)
	if err != nil {
		t.Fatalf("FetchCurrentPlayers failed: %v", err)
	}
	if count != 42100 {
		t.Errorf("expected player count 42100, got %d", count)
	}
}

func TestFetchGlobalAchievementPercentages(t *testing.T) {
	cache.SetDisabled(true)
	defer cache.SetDisabled(false)

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "GetGlobalAchievementPercentagesForApp") {
				resp := `{"achievementpercentages": {"achievements": [{"name": "ACH_01", "percent": 4.5}, {"name": "ACH_02", "percent": 52.8}]}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(resp)),
					Header:     make(http.Header),
				}, nil
			}
			t.Fatalf("unexpected request: %s", req.URL.String())
			return nil, nil
		}),
	}

	ctx := WithHTTPClient(context.Background(), mockClient)
	pcts, err := FetchGlobalAchievementPercentages(ctx, 123)
	if err != nil {
		t.Fatalf("FetchGlobalAchievementPercentages failed: %v", err)
	}

	if len(pcts) != 2 {
		t.Fatalf("expected 2 achievements, got %d", len(pcts))
	}
	if pcts["ACH_01"] != 4.5 {
		t.Errorf("expected ACH_01 to be 4.5, got %f", pcts["ACH_01"])
	}
	if pcts["ACH_02"] != 52.8 {
		t.Errorf("expected ACH_02 to be 52.8, got %f", pcts["ACH_02"])
	}
}
