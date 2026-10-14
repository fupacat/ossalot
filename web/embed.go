// Package web embeds the UI so the ossalot binary serves it from any working directory.
package web

import "embed"

// Assets holds dist/ (the page) and the src/ modules the page imports at runtime, without their tests.
// TRACEABILITY_MAPPING: REQ-040
//
//go:embed dist src/model.mjs src/*_model.mjs
var Assets embed.FS
