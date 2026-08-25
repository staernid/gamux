package util

import (
	"strconv"
	"strings"
)

// ParseAppID parses a string into a valid non-zero Steam uint32 AppID.
// It returns the uint32 value and true if valid, or 0 and false if invalid or "0" / "custom".
func ParseAppID(s string) (uint32, bool) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || trimmed == "0" || strings.EqualFold(trimmed, "custom") {
		return 0, false
	}
	val, err := strconv.ParseUint(trimmed, 10, 32)
	if err != nil || val == 0 {
		return 0, false
	}
	return uint32(val), true
}

// IsSteamAppID checks if a string represents a valid numeric Steam AppID.
func IsSteamAppID(s string) bool {
	_, ok := ParseAppID(s)
	return ok
}
