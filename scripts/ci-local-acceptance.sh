#!/usr/bin/env bash
#
# Local mirror of .github/workflows/pr-acceptance.yml — validates the 1Password
# env.json pull + fan-out + acceptance run WITHOUT pushing to GitHub.
#
# There are two ways to supply the credentials blob:
#   * 1Password (default) — pull the single env.json blob via `op read`, using
#     the same Service Account token CI uses. It first lists the vaults the token
#     can see (handy for confirming the Service Account has access).
#   * Local file — set ENV_JSON_FILE to a local env.json (e.g. the output of
#     `go run ./cmd/envtojson -from-env -o env.json`). 1Password is skipped
#     entirely, so you can test the fan-out + tests with no token/vault set up.
# Either way it then fans the flat JSON map out into the environment and runs TestAcc.
#
# Usage:
#   # From 1Password (default):
#   export OP_SERVICE_ACCOUNT_TOKEN=ops_...            # same token CI will use
#   export OP_ENV_REF="op://Streamkap-CI/ci-acceptance-env/env_json"  # optional override
#   scripts/ci-local-acceptance.sh                     # run the curated set (scripts/acceptance-tests.txt)
#   scripts/ci-local-acceptance.sh TestAccSourcePostgreSQLResource   # one test (override)
#
#   # From a local file (no 1Password):
#   ENV_JSON_FILE=env.json scripts/ci-local-acceptance.sh TestAccSourcePostgreSQLResource
#
# With no argument it runs only the curated acceptance tests listed in
# scripts/acceptance-tests.txt (the same set CI runs). Pass a -run pattern as $1
# to override and run just that test / regex instead.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_PATTERN="${1:-$("$SCRIPT_DIR/acceptance-run-pattern.sh")}"
OP_ENV_REF="${OP_ENV_REF:-op://Streamkap-CI/ci-acceptance-env/env_json}"
ENV_JSON_FILE="${ENV_JSON_FILE:-}"

if [ -n "$ENV_JSON_FILE" ]; then
  if [ ! -f "$ENV_JSON_FILE" ]; then
    echo "ENV_JSON_FILE=$ENV_JSON_FILE does not exist." >&2
    exit 1
  fi
  echo ">> Reading env.json from local file: $ENV_JSON_FILE (1Password skipped)"
  ENV_JSON="$(cat "$ENV_JSON_FILE")"
else
  if [ -z "${OP_SERVICE_ACCOUNT_TOKEN:-}" ]; then
    echo "OP_SERVICE_ACCOUNT_TOKEN is not set — export the Service Account token first," >&2
    echo "or set ENV_JSON_FILE=<path> to read a local env.json instead." >&2
    exit 1
  fi

  echo ">> Vaults visible to this 1Password token:"
  if op vault list 2>/tmp/op-vault-err; then
    :
  else
    echo "   (could not list vaults: $(cat /tmp/op-vault-err))" >&2
  fi
  rm -f /tmp/op-vault-err
  echo

  echo ">> Pulling env.json from 1Password: $OP_ENV_REF"
  ENV_JSON="$(op read "$OP_ENV_REF")"
fi

printf '%s' "$ENV_JSON" | jq -e 'type == "object"' >/dev/null \
  || { echo "env.json must be a flat JSON object of {\"VAR\": \"value\"}." >&2; exit 1; }

echo ">> Exporting $(printf '%s' "$ENV_JSON" | jq -r 'keys | length') variables into the environment"
while IFS= read -r key; do
  value="$(printf '%s' "$ENV_JSON" | jq -r --arg k "$key" '.[$k]')"
  export "$key=$value"
done < <(printf '%s' "$ENV_JSON" | jq -r 'keys[]')

echo ">> Connectors present in env.json (TF_VAR_ keys):"
printf '%s' "$ENV_JSON" | jq -r 'keys[] | select(startswith("TF_VAR_"))' | sed 's/^/   /'

echo ">> Running: TF_ACC=1 go test -count=1 -run '$RUN_PATTERN'"
# -count=1 bypasses Go's test result cache so every invocation re-runs against
# the real API instead of replaying a cached PASS.
TF_ACC=1 go test -count=1 -v -timeout 90m -run "$RUN_PATTERN" ./internal/provider/...
