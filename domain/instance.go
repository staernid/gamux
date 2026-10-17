package domain

import (
	"fmt"
	"strings"
)

// RunningInstance represents an active game execution process,
// whether a single-player session or a seat within a multi-seat session.
type RunningInstance struct {
	InstanceID  string `json:"instance_id"`       // Canonical unique ID: "<gameDir>" or "<gameDir>:<profileID>"
	GameDir     string `json:"game_dir"`          // Canonical game root directory
	GameName    string `json:"game_name"`         // Display title
	AppID       string `json:"app_id,omitempty"`  // Steam AppID or identifier
	ProfileID   string `json:"profile_id"`        // Gamux profile ID ("main", "p2", "p3", etc.)
	SeatID      string `json:"seat_id,omitempty"` // Multi-seat seat ID ("p1", "p2", etc., empty for solo)
	PID         int    `json:"pid"`               // Process ID
	IsMultiseat bool   `json:"is_multiseat"`      // True if part of a multi-seat session
	Mode        string `json:"mode,omitempty"`    // "local", "stream", etc.
	StartTime   int64  `json:"start_time"`        // Unix start timestamp
}

// MakeInstanceID generates the canonical instance identifier.
// For the primary profile ("main", "p1", or empty), it equals the canonical gameDir.
// For secondary profiles, it formats "<gameDir>:<profileID>".
func MakeInstanceID(gameDir, profileID string) string {
	cleanDir := strings.TrimRight(gameDir, "/")
	if profileID == "" || profileID == "main" || profileID == "p1" {
		return cleanDir
	}
	return fmt.Sprintf("%s:%s", cleanDir, profileID)
}

// ParseInstanceID splits a canonical instance ID into gameDir and profileID.
func ParseInstanceID(instanceID string) (gameDir, profileID string) {
	if idx := strings.LastIndex(instanceID, ":"); idx > 0 {
		return instanceID[:idx], instanceID[idx+1:]
	}
	return instanceID, "main"
}
