# SPDX-License-Identifier: AGPL-3.0-only

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: dev check test test-integration lint vuln web-check spdx generate build

dev:
	go run ./cmd/wickframe serve

check: lint test web-check spdx

test:
	go test -race ./...

test-integration:
	go test -race -tags integration ./internal/store/...

lint:
	go tool -modfile=tools/go.mod golangci-lint run ./...

vuln:
	go tool -modfile=tools/go.mod govulncheck ./...

web-check:
	cd web && pnpm lint && pnpm typecheck && pnpm test && pnpm build

spdx:
	scripts/spdx.sh

generate:
	go generate ./...
	go tool -modfile=tools/go.mod sqlc generate -f internal/store/sqlite/sqlc.yaml
	go tool -modfile=tools/go.mod sqlc generate -f internal/store/postgres/sqlc.yaml

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/wickframe ./cmd/wickframe
