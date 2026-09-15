#!/usr/bin/env bash
set -euo pipefail

results="${1:?check-migration-results: pass the go test JSON results file}"
script_dir="$(cd "$(dirname "$0")" && pwd)"
expected="$(sed -n 's/^func \(TestAcc.*_MigrationFromLegacy\)(t \*testing.T) {.*/\1/p' "$script_dir/../internal/provider/migration_test.go")"

if [ -z "$expected" ]; then
  echo 'check-migration-results: no migration tests found' >&2
  exit 1
fi

jq -s --arg expected "$expected" '
  . as $events |
  ($expected | split("\n")) as $tests |
  [$tests[] | . as $name |
    {name: $name, result: ([$events[] |
      select(.Package == "github.com/streamkap-com/terraform-provider-streamkap/internal/provider" and .Test == $name) |
      select(.Action == "pass" or .Action == "fail" or .Action == "skip") |
      .Action] | last // "not run")}] |
  .[] | "\(.result): \(.name)"
' -r "$results"

if ! jq -e -s --arg expected "$expected" '
  . as $events |
  all($expected | split("\n")[]; . as $name |
    any($events[]; .Package == "github.com/streamkap-com/terraform-provider-streamkap/internal/provider" and .Test == $name and .Action == "pass")) and
  all($events[]; .Action != "fail" and .Action != "skip")
' "$results" >/dev/null; then
  echo 'check-migration-results: migration coverage is incomplete; every case must run and pass' >&2
  exit 1
fi
echo 'PASS migration coverage: every case ran and passed'
