package embedded

import "embed"

//go:embed templates/* ansible/*
var Assets embed.FS
