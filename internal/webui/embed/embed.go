package embed

import "embed"

// DistFS contains the built SPA assets.
// For the MVP we embed a tiny placeholder `dist` so the server can serve something.
// In a full build, this folder should contain the Vite build output.
//
//go:embed all:dist
var DistFS embed.FS
