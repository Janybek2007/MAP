package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultProdPort    = 27436
	DefaultDevPort     = 8080
	serverMarkerHeader = "X-Map-Server"
)

type Config struct {
	Mode                 string
	Port                 int
	APIKey               string
	SQLitePath           string
	ProtectTokenEndpoint bool
}

func loadConfig(args []string) (Config, error) {
	mode := "prod"
	port := DefaultProdPort

	for _, arg := range args {
		if strings.HasPrefix(arg, "-mode=") {
			mode = strings.TrimSpace(strings.TrimPrefix(arg, "-mode="))
			continue
		}

		if strings.HasPrefix(arg, "-port=") {
			value := strings.TrimSpace(strings.TrimPrefix(arg, "-port="))
			number, err := strconv.Atoi(value)
			if err == nil && number > 0 && number < 65536 {
				port = number
			}
		}
	}

	if mode != "dev" && mode != "prod" {
		return Config{}, fmt.Errorf("неверный -mode=%s (используй dev|prod)", mode)
	}

	if port == DefaultProdPort && mode == "dev" {
		port = DefaultDevPort
	}

	loadEnvFiles(mode)

	apiKey := resolveAPIKey()
	sqlitePath := strings.TrimSpace(os.Getenv("SQLITE_PATH"))

	var missing []string
	if apiKey == "" {
		missing = append(missing, "API_KEY")
	}
	if sqlitePath == "" {
		missing = append(missing, "SQLITE_PATH")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("не заданы переменные окружения: %s", strings.Join(missing, ", "))
	}

	return Config{
		Mode:                 mode,
		Port:                 port,
		APIKey:               apiKey,
		SQLitePath:           sqlitePath,
		ProtectTokenEndpoint: mode == "prod",
	}, nil
}

func resolveAPIKey() string {
	return strings.TrimSpace(os.Getenv("API_KEY"))
}
