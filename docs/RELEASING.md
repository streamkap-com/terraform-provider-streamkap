# Branches and releases

Maintainer procedure for the two release lines. Pushing a tag publishes public
artifacts, so every tag needs explicit approval — the agent rules in
[AGENTS.md](../AGENTS.md) stop before `git push`.

## Branches

| Branch | Line | Receives |
|---|---|---|
| `main` | v3.x stable (default) | Features, fixes, PRs |
| `v2` | v2.x legacy | Bug and security fixes only, through 15 October 2026; support ends 16 October 2026 |

- Keep the histories independent: never merge `main` and `v2`. A fix for v2
  users lands on `v2`; a shared fix is a separate commit on each line.
- Cut feature branches from the line you target. Pull and confirm the branch
  before starting; do not assume the working tree is current.
- Workflows run from the tagged commit; a change on `main` does not update `v2`.
- Existing v2 artifacts stay available after v3 GA.

### Clones with the pre-promotion branch names

Former `main` is now `v2`; former `develop` is now `main`. In a clone that still
uses the old names: preserve local commits, check for existing local branches
before renaming, rename local `main` → `v2` and `develop` → `main`, `git fetch
--prune`, set each upstream to its matching remote branch, then
`git remote set-head origin -a`. Never overwrite local history.

## Release flow

"Release the next beta" means: use `main`, pull, verify a clean tree and passing
checks, select the next unused beta tag, and stop for approval before pushing.

Two preflight gates reject a tag before goreleaser runs; check both before tagging:

1. `git fetch --prune origin`, then `bash scripts/release-preflight.sh <tag>`.
   Pruning removes stale pre-promotion refs. Preflight checks tag syntax, branch
   ancestry (v2 patches → `v2`; v3 beta and stable → `main`) and the matching
   changelog heading.
2. A `## [<version>] - <date>` changelog section for **every** release,
   including a trial beta. Preserve previous release sections.

**Bump the version pins in the same commit that cuts the CHANGELOG, before
tagging.** The registry renders each version's pages from that version's own
tag, so a pin updated after the tag ships a page telling users to install the
*previous* release. Pins live in `examples/provider/provider.tf` (`docs/index.md`
is generated from it — edit the example and re-render with `go generate
main.go`, never hand-edit the doc), `README.md` and `templates/guides/migration.md`.

**Push the approved branch commit before tagging it.**

### Stable promotion checklist

1. Verify the intended release commit and both lines' release workflows.
2. Verify the required checks on an ordinary PR: `Build, vet and credential-free
   tests`, `golangci-lint`, `Workflow lint`, `Docs match committed schemas`,
   `Go vulnerability analysis`, `trivy`, `checkov` and `Secret scan`.
   Acceptance and migration remain advisory; record which cases ran, failed
   or skipped before authorizing a release. A skipped case is not upgrade evidence.
3. Prepare the stable release's changelog and pins, run preflight and checks,
   and obtain tag approval. A trial beta is optional, not a prerequisite.
4. After publishing, verify the registry download and a clean `terraform init`
   with an exact pin. A GitHub release alone does not prove registry ingestion.
5. Publish the migration guide before sending the customer notice.

## Security gate

Publishing v3 is gated on the security workflow. Trivy fails on any HIGH or
CRITICAL advisory, including transitive dependencies, regardless of code-path
reachability.

When that happens, `goreleaser` shows as `skipped` and **nothing is published**
— no GitHub release, no registry artifacts. Confirm with `gh release view <tag>`
returning "release not found". Then bump the dependency, push the approved
release branch, and re-point the tag (`git push origin :refs/tags/<tag>`, re-tag,
push). Re-pointing is safe *only* because the tag published nothing; once a
release exists, burn the number and cut the next one instead.
