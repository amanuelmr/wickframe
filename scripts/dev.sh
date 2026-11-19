#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only

# Runs the server and the Vite dev server, which proxies /api to it. Ctrl-C
# stops both. Restart after changing Go code.
set -euo pipefail

(cd web && pnpm install --frozen-lockfile)
# A built binary, not go run, so the trap stops the server itself.
go build -o bin/wickframe ./cmd/wickframe
bin/wickframe serve &
server=$!
trap 'kill $server 2>/dev/null' EXIT
cd web && pnpm dev
