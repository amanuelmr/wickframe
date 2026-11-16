#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only

# Fails if a non-merge commit in the range lacks a Signed-off-by trailer for its author.
set -euo pipefail

range=${1:?usage: scripts/dco.sh <base>..<head>}
commits=$(git rev-list --no-merges "$range")
status=0
for c in $commits; do
  author=$(git show -s --format='%an <%ae>' "$c")
  if ! git show -s --format='%(trailers:key=Signed-off-by,valueonly)' "$c" | grep -qxF "$author"; then
    echo "$(git show -s --format='%h %s' "$c"): missing Signed-off-by: $author" >&2
    status=1
  fi
done
exit "$status"
