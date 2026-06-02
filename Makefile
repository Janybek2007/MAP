.PHONY: run run-go-dev run-app build

GO ?= go
NPM ?= npm
APP ?= MAP.exe

run:
	@$(MAKE) -j2 run-go-dev run-app

run-go-dev:
	$(GO) run . -mode=dev -port=8080

run-app:
	$(NPM) run dev

build:
	$(NPM) run build
	$(GO) build -tags=prod -o $(APP) .
