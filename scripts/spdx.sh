#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only

# Fails if a tracked source file lacks an SPDX-License-Identifier in its first lines.
set -euo pipefail

files=$(git ls-files -- '*.go' '*.ts' '*.tsx' '*.css' 'Makefile' '*/Makefile' '*.yml' '*.yaml' \
  ':!:*pnpm-lock.yaml' ':!:api/openapi.yaml' ':!:*.gen.ts' ':!:web/src/api/generated/**' ':!:internal/store/*/db/**')
status=0
for f in $files; do
  if ! head -n 3 "$f" | grep -q 'SPDX-License-Identifier:'; then
    echo "$f: missing SPDX-License-Identifier header" >&2
    status=1
  fi
done
exit "$status"
