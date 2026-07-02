#!/usr/bin/env bash
#
# Prints the `go test -run` regex for the curated acceptance test set, built
# from scripts/acceptance-tests.txt (one test name per line; `#` comments and
# blank lines ignored). This is the single source of truth shared by
# scripts/ci-local-acceptance.sh and .github/workflows/pr-acceptance.yml so the
# two cannot drift.
#
# The regex is anchored at the start only, e.g. ^(TestAccSourcePostgreSQLResource|...),
# so a listed name also matches its sub-variants (TestAccSourcePostgreSQLResource
# also matches TestAccSourcePostgreSQLResource_WithTimeout) — matching how each
# test runs when passed individually.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIST_FILE="${1:-$SCRIPT_DIR/acceptance-tests.txt}"

if [ ! -f "$LIST_FILE" ]; then
  echo "acceptance test list not found: $LIST_FILE" >&2
  exit 1
fi

joined="$(grep -vE '^[[:space:]]*(#|$)' "$LIST_FILE" | sed -E 's/[[:space:]]+//g' | tr '\n' '|' | sed 's/|$//')"

if [ -z "$joined" ]; then
  echo "no test names found in $LIST_FILE" >&2
  exit 1
fi

printf '^(%s)\n' "$joined"
