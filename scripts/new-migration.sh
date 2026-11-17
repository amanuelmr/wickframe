#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only

# Creates the next numbered, empty goose migration in both dialect folders, so
# they keep the same numbers and names (08-storage §3).
set -euo pipefail

name=${1:?usage: scripts/new-migration.sh <snake_case_name>}
if [[ ! $name =~ ^[a-z][a-z0-9_]*$ ]]; then
  echo "migration name must be snake_case: $name" >&2
  exit 1
fi
last=$(find migrations/sqlite migrations/postgres -name '[0-9][0-9][0-9][0-9][0-9]_*.sql' -exec basename {} \; | sort | tail -1)
next=$(printf '%05d' $((10#${last:0:5} + 1)))
for dialect in sqlite postgres; do
  file=migrations/$dialect/${next}_$name.sql
  printf -- '-- SPDX-License-Identifier: AGPL-3.0-only\n\n-- +goose Up\n' > "$file"
  echo "$file"
done
