.PHONY: run run-go-dev run-app build build-win build-mac import-sqlite import-sqlite-initial import-sqlite-overwrite import-sqlite-dry-run

-include server/.env.production

GO ?= go
NPM ?= npm
APP ?= MAP.exe
NODE ?= node

ifeq ($(OS),Windows_NT)
SET_GOOS_DARWIN_ARM64 = set GOOS=darwin&& set GOARCH=arm64&&
else
SET_GOOS_DARWIN_ARM64 = GOOS=darwin GOARCH=arm64
endif

MODE ?= prod
IMPORT_MODE ?= overwrite

run:
	@$(MAKE) -j2 run-go-dev run-app

run-go-dev:
	$(GO) -C server run . -mode=dev -port=8080

run-app:
	$(NPM) run dev

sync-server-assets:
	$(NODE) server/scripts/sync-assets.mjs

assert-sqlite-path:
ifndef SQLITE_PATH
	$(error SQLITE_PATH is required for build targets; set it in environment or .env.production)
endif

build:
	$(NPM) run build
	$(MAKE) build-win
	$(MAKE) build-mac

build-win: assert-sqlite-path sync-server-assets
	cd server && $(GO) build -tags=prod -o ../MAP.exe .

build-mac: assert-sqlite-path sync-server-assets
	cd server && $(SET_GOOS_DARWIN_ARM64) $(GO) build -tags=prod -o ../MAP .

import-sqlite:
	$(GO) -C server run ./scripts/import_sqlite.go -mode=$(MODE) -import-mode=$(IMPORT_MODE)

import-sqlite-initial:
	$(MAKE) import-sqlite MODE=$(MODE) IMPORT_MODE=initial

import-sqlite-overwrite:
	$(MAKE) import-sqlite MODE=$(MODE) IMPORT_MODE=overwrite

import-sqlite-dry-run:
	$(MAKE) import-sqlite MODE=$(MODE) IMPORT_MODE=dry-run
