package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/staernid/gamux/domain"
)

var (
	profileURLRegex = regexp.MustCompile(`(?i)steamcommunity\.com/(id|profiles)/([^/?#]+)`)
	steamID64Regex  = regexp.MustCompile(`^\d{17}$`)
)

type resolveVanityURLResponse struct {
	Response struct {
		SteamID string `json:"steamid"`
		Success int    `json:"success"`
		Message string `json:"message"`
	} `json:"response"`
}

type playerSummariesResponse struct {
	Response struct {
		Players []struct {
			SteamID        string `json:"steamid"`
			PersonaName    string `json:"personaname"`
			ProfileURL     string `json:"profileurl"`
			AvatarFull     string `json:"avatarfull"`
			LocCountryCode string `json:"loccountrycode"`
		} `json:"players"`
	} `json:"response"`
}

// CleanSteamProfileInput parses raw profile URLs, vanity handles, or numeric SteamIDs.
// Returns the candidate identifier and whether it was identified as an explicit SteamID64 or vanity handle.
func CleanSteamProfileInput(input string) (identifier string, isSteamID64 bool) {
	trimmed := strings.TrimSpace(input)
	trimmed = strings.TrimRight(trimmed, "/")

	if matches := profileURLRegex.FindStringSubmatch(trimmed); len(matches) == 3 {
		kind := strings.ToLower(matches[1])
		val := matches[2]
		if kind == "profiles" && steamID64Regex.MatchString(val) {
			return val, true
		}
		return val, false
	}

	if steamID64Regex.MatchString(trimmed) {
		return trimmed, true
	}

	return trimmed, false
}

// ResolveSteamProfile resolves a vanity handle, custom URL, or numeric SteamID into a canonical SteamProfile.
func ResolveSteamProfile(ctx context.Context, apiKey string, input string) (*domain.SteamProfile, error) {
	if strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("steam profile identifier cannot be empty")
	}

	idCandidate, is64 := CleanSteamProfileInput(input)
	steamID64 := idCandidate

	if !is64 {
		if apiKey == "" {
			return nil, fmt.Errorf("a Steam Web API key is required to resolve vanity profile %q", input)
		}

		vanityURL := fmt.Sprintf("%s/ISteamUser/ResolveVanityURL/v1/?key=%s&vanityurl=%s", DefaultSteamWebAPI, url.QueryEscape(apiKey), url.QueryEscape(idCandidate))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, vanityURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create resolve vanity request: %w", err)
		}

		resp, err := HTTPClientFromContext(ctx).Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch vanity url: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("resolve vanity URL HTTP %d", resp.StatusCode)
		}

		var vRes resolveVanityURLResponse
		if err := json.NewDecoder(resp.Body).Decode(&vRes); err != nil {
			return nil, fmt.Errorf("decode vanity URL json: %w", err)
		}

		if vRes.Response.Success != 1 || vRes.Response.SteamID == "" {
			msg := vRes.Response.Message
			if msg == "" {
				msg = "no match found"
			}
			return nil, fmt.Errorf("could not resolve Steam vanity identifier %q: %s", idCandidate, msg)
		}

		steamID64 = vRes.Response.SteamID
	}

	// If no API key is set but we already have a 17-digit SteamID64, return basic profile
	if apiKey == "" {
		return &domain.SteamProfile{
			SteamID:     steamID64,
			PersonaName: steamID64,
			ProfileURL:  "https://steamcommunity.com/profiles/" + steamID64,
		}, nil
	}

	summariesURL := fmt.Sprintf("%s/ISteamUser/GetPlayerSummaries/v2/?key=%s&steamids=%s", DefaultSteamWebAPI, url.QueryEscape(apiKey), steamID64)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, summariesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create player summaries request: %w", err)
	}

	resp, err := HTTPClientFromContext(ctx).Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch player summaries: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("player summaries HTTP %d", resp.StatusCode)
	}

	var pRes playerSummariesResponse
	if err := json.NewDecoder(resp.Body).Decode(&pRes); err != nil {
		return nil, fmt.Errorf("decode player summaries json: %w", err)
	}

	if len(pRes.Response.Players) == 0 {
		return nil, fmt.Errorf("no player profile found for SteamID %s", steamID64)
	}

	player := pRes.Response.Players[0]
	return &domain.SteamProfile{
		SteamID:     player.SteamID,
		PersonaName: player.PersonaName,
		AvatarURL:   player.AvatarFull,
		CountryCode: player.LocCountryCode,
		ProfileURL:  player.ProfileURL,
	}, nil
}
