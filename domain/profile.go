package domain

import (
	"fmt"
)

// Profile represents a user/player identity in Gamux.
// Profiles can be selected for solo play or assigned to seats in split-screen co-op.
type Profile struct {
	ID        string `json:"id"`                   // Unique identifier, e.g. "main", "p2", "prof_123"
	Index     int    `json:"index"`                // 1 for main, 2 for p2, 3 for p3...
	Name      string `json:"name"`                 // Display name / persona nickname
	SteamID   string `json:"steam_id"`             // 17-digit SteamID64
	Language  string `json:"language"`             // Emulation language (default "english")
	IPCountry string `json:"ip_country"`           // Two-letter country code (default "US")
	AvatarURL string `json:"avatar_url,omitempty"` // Custom avatar URL or local image path
	IsMain    bool   `json:"is_main"`              // True for the primary host profile (uses host $HOME directly)
}

// Normalize sets sensible defaults for missing fields in a Profile.
func (p *Profile) Normalize() {
	if p.Name == "" {
		if p.IsMain {
			p.Name = "GamuxPlayer"
		} else if p.Index > 0 {
			p.Name = fmt.Sprintf("Player%d", p.Index)
		} else {
			p.Name = "Player"
		}
	}
	if p.SteamID == "" {
		if p.IsMain {
			p.SteamID = "76561197960287930"
		} else {
			p.SteamID = "76561198998877665"
		}
	}
	if p.Language == "" {
		p.Language = "english"
	}
	if p.IPCountry == "" {
		p.IPCountry = "US"
	}
}
