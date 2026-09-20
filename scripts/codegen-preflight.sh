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

# Connectors listed under smt_chain_connectors in cmd/tfgen/overrides.json build
# their chain schema from a pinned catalog artifact; tfgen refuses to run for
# them without the artifact and the backend revision it was copied from.
if [ -z "${STREAMKAP_SMT_CATALOG:-}" ] || [ ! -f "$STREAMKAP_SMT_CATALOG" ]; then
  echo "codegen aborted: STREAMKAP_SMT_CATALOG is unset or not a readable file — the SMT chain schemas are built from that pinned artifact." >&2
  exit 1
fi
if [ -z "${STREAMKAP_BACKEND_REVISION:-}" ]; then
  echo "codegen aborted: STREAMKAP_BACKEND_REVISION is unset — the catalog artifact carries no revision, so the run must record the backend commit it came from." >&2
  exit 1
fi
echo "codegen smt catalog: $STREAMKAP_SMT_CATALOG @ $STREAMKAP_BACKEND_REVISION"

branch="$(git -C "$STREAMKAP_BACKEND_PATH" rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
echo "codegen backend: $STREAMKAP_BACKEND_PATH @ $branch"

if [ "$branch" != "main" ] && [ "${ALLOW_NONMAIN:-}" != "1" ]; then
  echo "codegen aborted: backend is on '$branch', not 'main'." >&2
  echo "  Regenerate against main, or re-run with ALLOW_NONMAIN=1 if you intend this branch." >&2
  exit 1
fi
