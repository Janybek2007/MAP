//go:build !prod

package main

import "embed"

// appFS is unused in dev mode but must exist for compilation.
var appFS embed.FS

