# SPDX-License-Identifier: AGPL-3.0-only

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: dev check test lint generate build

dev:
	go run ./cmd/wickframe serve

check: lint test

test:
	go test -race ./...

lint:
	go tool -modfile=tools/go.mod golangci-lint run ./...

generate:
	go generate ./...

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/wickframe ./cmd/wickframe
