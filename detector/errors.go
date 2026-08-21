package detector

import "errors"

var (
	// ErrGameNotFound indicates no valid game directory or executable was found.
	ErrGameNotFound = errors.New("game directory or executable not found")

	// ErrNoExecutableFound indicates no viable game executable was detected.
	ErrNoExecutableFound = errors.New("no viable game executable found in directory")

	// ErrInvalidManifest indicates the manifest is corrupt, malformed, or missing essential keys.
	ErrInvalidManifest = errors.New("invalid or unparseable Steam manifest")
)
