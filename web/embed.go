// Package web embeds the built UI so the ossalot binary serves it from any working directory.
package web

import "embed"

// Assets holds dist/, the Vite build output. The committed placeholder dist/.gitkeep keeps the
// pattern valid before a frontend build, so Go compiles without Node (ADR-029).
// TRACEABILITY_MAPPING: REQ-052, REQ-053
//
//go:embed all:dist
var Assets embed.FS
