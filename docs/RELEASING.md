# Releasing

## Release flow

"Release the next beta" = use `main`, pull, verify a clean tree and passing checks, and select the next unused beta tag. **Pushing the tag triggers the public release workflow** — surface the exact `git tag`/`git push` commands and STOP for explicit approval. Never push tags or branches unprompted.

Two preflight gates reject a tag before goreleaser runs, so check both *before* tagging:

- Run `git fetch --prune origin`, then `bash scripts/release-preflight.sh <tag>`.
  Pruning removes stale pre-promotion refs. Preflight checks tag syntax, branch
  ancestry and the matching changelog heading. v2
  patches use `v2`; v3 beta and stable releases use `main`.
- Prepare a matching `## [<version>] - <date>` changelog section for **every**
  release, including a trial beta. Preserve previous release sections. Update
  the version pins and rendered docs in the same commit. **Push the approved
  branch commit before tagging it.**

## Stable promotion

The branch reorganization is complete: former `main` is `v2`, former
`develop` is the default `main`, and both histories are preserved.
Public tags require separate approval.

1. Verify the intended release commit and both lines' release workflows.
   Workflows run from the tagged commit; changes on `main` do not update `v2`.
2. Verify the required checks on an ordinary PR: `Build, vet and credential-free
   tests`, `golangci-lint`, `Workflow lint`, `Docs match committed schemas`,
   `Go vulnerability analysis`, `trivy`, `checkov` and `Secret scan`.
   Acceptance and migration remain advisory; record which cases ran, failed
   or skipped before authorizing a release. A skipped case is not upgrade evidence.
3. Prepare the stable release's changelog and pins, run preflight and checks,
   and obtain tag approval. A trial beta is optional, not a prerequisite. After
   publishing, verify the registry download and a clean `terraform init` with
   an exact pin. Publish the migration guide before sending the customer notice.

For clones that still use the old names, preserve local commits, rename the old
local `main` to `v2` and `develop` to `main`, then fetch/prune, set each upstream
to its matching remote branch, and run `git remote set-head origin -a`. Check
for existing local branches before renaming; do not overwrite local history.

From v3 GA, v2 is the legacy maintenance line: bug and security fixes only
through 15 October 2026, with support ending 16 October 2026. Existing v2
artifacts remain available. Verify registry availability and installability;
a GitHub release alone does not prove registry ingestion.

**Publishing v3 is gated on the security workflow.** Trivy fails on any HIGH or CRITICAL advisory, including transitive dependencies, regardless of code-path reachability.

When that happens, `goreleaser` shows as `skipped` and **nothing is published** — no GitHub release, no registry artifacts. Confirm with `gh release view <tag>` returning "release not found". Then bump the dependency, push the approved release branch, and re-point the tag (`git push origin :refs/tags/<tag>` then re-tag and push). Re-pointing is safe *only* because the tag published nothing; once a release exists, burn the number and cut the next one instead.

**Bump the version pins in the same commit that cuts the CHANGELOG, before tagging.** The registry renders each version's pages from that version's own tag, so a pin updated after the tag ships a page telling users to install the *previous* release. The pin lives in `examples/provider/provider.tf` (`docs/index.md` is generated from it — edit the example and re-render with `go generate main.go`, never hand-edit the doc), plus `README.md` and `templates/guides/migration.md`.
