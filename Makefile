.PHONY: run run-go-dev run-app build build-win build-mac

-include .env.production

GO ?= go
NPM ?= npm
APP ?= MAP.exe
LDFLAGS ?= -X main.buildAPIKey=$(API_KEY)

ifeq ($(OS),Windows_NT)
SET_GOOS_DARWIN_ARM64 = set GOOS=darwin&& set GOARCH=arm64&&
else
SET_GOOS_DARWIN_ARM64 = GOOS=darwin GOARCH=arm64
endif

run:
	@$(MAKE) -j2 run-go-dev run-app

run-go-dev:
	$(GO) run . -mode=dev -port=8080

run-app:
	$(NPM) run dev

assert-api-key:
ifndef API_KEY
	$(error API_KEY is required for build targets; set it in environment or .env.production)
endif

build:
	$(NPM) run build
	$(MAKE) build-win
	$(MAKE) build-mac

build-win: assert-api-key
	$(GO) build -tags=prod -ldflags "$(LDFLAGS)" -o MAP.exe .

build-mac: assert-api-key
	$(SET_GOOS_DARWIN_ARM64) $(GO) build -tags=prod -ldflags "$(LDFLAGS)" -o MAP .
