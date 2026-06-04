package ui

import "embed"

// Assets contains the static file assets for the web UI dashboard dashboard.
//go:embed dist/*
var Assets embed.FS
