package manifest

import "errors"

var (
	// ErrDepotKeyNotFound indicates no decryption key is available for the given depot.
	ErrDepotKeyNotFound = errors.New("no depot decryption key found")

	// ErrManifestNotFound indicates the requested Steam depot manifest was not found.
	ErrManifestNotFound = errors.New("steam depot manifest not found")

	// ErrPayloadCorrupt indicates the manifest payload failed decryption or decompression.
	ErrPayloadCorrupt = errors.New("manifest payload is corrupt or invalid")
)
