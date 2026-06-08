//go:build prod

package main

import "embed"

//go:embed build build/*
var appFS embed.FS
