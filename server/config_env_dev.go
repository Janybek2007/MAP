//go:build !prod

package main

import "github.com/joho/godotenv"

func loadEnvFiles(mode string) {
	_ = godotenv.Load(".env")
	if mode == "dev" {
		_ = godotenv.Overload(".env.development")
		return
	}
	_ = godotenv.Overload(".env.production")
}
