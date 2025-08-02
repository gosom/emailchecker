package static

import "embed"

//go:embed src/*
var StaticFiles embed.FS
