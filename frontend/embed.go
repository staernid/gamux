package frontend

import "embed"

// Dist contains the embedded Vite-compiled frontend assets.
//
//go:embed all:dist
var Dist embed.FS
