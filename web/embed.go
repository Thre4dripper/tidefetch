// Package web embeds the compiled single-page application served by
// `tidefetch serve`. Run `make web` (npm build in ./web) to refresh dist.
package web

import "embed"

// Assets holds the built frontend (dist/).
//
//go:embed all:dist
var Assets embed.FS
