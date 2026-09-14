#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
sed -n 's/^func \(TestAcc.*_MigrationFromLegacy\)(t \*testing.T) {.*/\1/p' "$script_dir/../internal/provider/migration_test.go" |
  jq -Rn 'inputs | {Package: "github.com/streamkap-com/terraform-provider-streamkap/internal/provider", Test: ., Action: "pass"}' > "$fixture/pass.json"
bash "$script_dir/check-migration-results.sh" "$fixture/pass.json" >/dev/null

expect_failure() {
  if bash "$script_dir/check-migration-results.sh" "$1" > "$fixture/output" 2>&1; then
    echo "Unexpectedly accepted incomplete migration results: $1" >&2
    exit 1
  fi
}

jq '.Action = "skip"' "$fixture/pass.json" > "$fixture/skipped.json"
expect_failure "$fixture/skipped.json"
jq -s '.[1:][]' "$fixture/pass.json" > "$fixture/missing.json"
expect_failure "$fixture/missing.json"
jq -s '.[0].Action = "fail" | .[]' "$fixture/pass.json" > "$fixture/failed.json"
expect_failure "$fixture/failed.json"
jq -s '.[0].Action = "skip" | .[]' "$fixture/pass.json" > "$fixture/partial.json"
expect_failure "$fixture/partial.json"
touch "$fixture/empty.json"
expect_failure "$fixture/empty.json"
echo 'PASS migration results reject skipped, missing, failed and empty runs'
