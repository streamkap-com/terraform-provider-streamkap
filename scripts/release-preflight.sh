#!/usr/bin/env bash
set -euo pipefail

tag="${1:?release-preflight: pass the release tag}"
if [[ "$tag" =~ ^v2\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  release_branch=main
  if git show-ref --verify --quiet refs/remotes/origin/v2; then
    release_branch=v2
  fi
elif [[ "$tag" =~ ^v3\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)-beta\.(0|[1-9][0-9]*)$ ]]; then
  release_branch=main
  if git show-ref --verify --quiet refs/remotes/origin/develop; then
    release_branch=develop
  fi
elif [[ "$tag" =~ ^v3\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  release_branch=main
  if git show-ref --verify --quiet refs/remotes/origin/develop; then
    echo 'release-preflight: complete the branch promotion before releasing v3 stable' >&2
    exit 1
  fi
else
  echo "release-preflight: unsupported tag $tag" >&2
  exit 1
fi

if ! git merge-base --is-ancestor HEAD "refs/remotes/origin/$release_branch"; then
  echo "release-preflight: $tag commit is not contained in origin/$release_branch" >&2
  exit 1
fi

if ! awk -v heading="## [${tag#v}]" '$0 == heading || index($0, heading " - ") == 1 {found=1} END {exit !found}' CHANGELOG.md; then
  echo "release-preflight: CHANGELOG.md has no [${tag#v}] heading" >&2
  exit 1
fi
printf '%s\n' "$release_branch"
