#!/usr/bin/env bash
set -euo pipefail

script="$(cd "$(dirname "$0")" && pwd)/release-preflight.sh"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
git init -q "$fixture"
cd "$fixture"
git config user.name 'Release test'
git config user.email 'release-test@example.com'
git commit -q --allow-empty -m v2
v2_sha="$(git rev-parse HEAD)"
git update-ref refs/remotes/origin/main "$v2_sha"
git checkout -q --orphan develop
git commit -q --allow-empty -m v3
v3_sha="$(git rev-parse HEAD)"
git update-ref refs/remotes/origin/develop "$v3_sha"

expect_pass() {
  printf '## [%s] - 2026-09-14\n' "${1#v}" > CHANGELOG.md
  actual="$(bash "$script" "$1")"
  test "$actual" = "$2" || { echo "Expected branch $2, got $actual" >&2; exit 1; }
}

expect_fail() {
  if bash "$script" "$1" >result 2>&1; then
    echo "Unexpectedly accepted $1" >&2
    exit 1
  fi
}

expect_pass v3.0.0-beta.32 develop
expect_pass v3.1.0-beta.1 develop
expect_fail v3.0.0
expect_fail v2.2.1
expect_fail v4.0.0-beta.1
expect_fail v3.00.0-beta.1
expect_fail v3.0.0-rc.1
printf '## [Unreleased]\n' > CHANGELOG.md
expect_fail v3.0.0-beta.32

git checkout -q --detach "$v2_sha"
expect_pass v2.2.1 main
expect_fail v3.0.0

git update-ref refs/remotes/origin/v2 "$v2_sha"
git update-ref refs/remotes/origin/main "$v3_sha"
git update-ref -d refs/remotes/origin/develop
expect_pass v2.2.1 v2
expect_fail v3.0.0

git checkout -q --detach "$v3_sha"
expect_pass v3.0.0-beta.32 main
expect_pass v3.1.0-beta.1 main
expect_pass v3.0.0 main
expect_fail v2.2.1
echo 'PASS release preflight before and after branch promotion'
