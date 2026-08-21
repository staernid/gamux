package steam

import "errors"

var (
	// ErrRateLimited indicates Steam API returned HTTP 429 Too Many Requests.
	ErrRateLimited = errors.New("steam API rate limit reached (HTTP 429)")

	// ErrAppNotFound indicates no metadata or store listing exists for the given AppID.
	ErrAppNotFound = errors.New("app details not found or unavailable on Steam")

	// ErrNetworkUnavailable indicates network timeout or DNS resolution failure.
	ErrNetworkUnavailable = errors.New("network unavailable or Steam endpoint unreachable")
)
