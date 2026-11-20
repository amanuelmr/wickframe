# SPDX-License-Identifier: AGPL-3.0-only

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: dev check test test-integration lint vuln web-check spdx generate migration web-build build

dev:
	scripts/dev.sh

check: lint test web-check spdx

test:
	go test -race ./...

test-integration:
	go test -race -tags integration ./internal/store/...

lint:
	go tool -modfile=tools/lint/go.mod golangci-lint run ./...

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
	go run ./cmd/wickframe openapi > api/openapi.yaml

migration:
	scripts/new-migration.sh "$(name)"

web-build:
	cd web && pnpm install --frozen-lockfile && pnpm build
	find internal/webui/dist -mindepth 1 ! -name .gitkeep -exec rm -rf {} +
	cp -R web/dist/. internal/webui/dist/

build: web-build
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/wickframe ./cmd/wickframe
