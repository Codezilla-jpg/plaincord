GO ?= go
VERSION ?= 0.2.0
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o dc ./cmd/dc
	cp -f dc DiscordCli

test:
	$(GO) test ./...
