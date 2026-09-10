GO ?= go
VERSION ?= 0.2.0
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o dis ./cmd/dis
	cp -f dis DiscordCli

test:
	$(GO) test ./...
