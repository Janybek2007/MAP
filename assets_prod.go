//go:build prod

package main

import "embed"

//go:embed build build/* data data/*
var appFS embed.FS
