#!/usr/bin/env bash
# Preflight for `make generate`. Guards the two recurring codegen footguns:
#   1. STREAMKAP_BACKEND_PATH unset/unreadable -> tfgen silently emits wrong output.
#   2. Backend on a branch other than main -> production runs older configs than
#      feature branches, so regenerating against one strips/adds fields that aren't live.
# Override the branch check with ALLOW_NONMAIN=1 when you deliberately target a branch.
set -euo pipefail

if [ -z "${STREAMKAP_BACKEND_PATH:-}" ]; then
  echo "codegen aborted: STREAMKAP_BACKEND_PATH is unset — tfgen would emit wrong output." >&2
  exit 1
fi

if ! ls "$STREAMKAP_BACKEND_PATH" >/dev/null 2>&1; then
  echo "codegen aborted: STREAMKAP_BACKEND_PATH ($STREAMKAP_BACKEND_PATH) is not a readable directory." >&2
  exit 1
fi

branch="$(git -C "$STREAMKAP_BACKEND_PATH" rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
echo "codegen backend: $STREAMKAP_BACKEND_PATH @ $branch"

if [ "$branch" != "main" ] && [ "${ALLOW_NONMAIN:-}" != "1" ]; then
  echo "codegen aborted: backend is on '$branch', not 'main'." >&2
  echo "  Regenerate against main, or re-run with ALLOW_NONMAIN=1 if you intend this branch." >&2
  exit 1
fi
