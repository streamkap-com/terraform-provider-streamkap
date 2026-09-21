# SMT chain on connector resources

The nested SMT chain representation was first proved by a prototype under `internal/smtproto/`. That prototype has moved into the real generator and the real base resource: `internal/smt` is the representation, `cmd/tfgen` pins the catalog it is built from, and `internal/resource/connector` drives it for any generated connector that opts in. Three resources are regenerated with the chain enabled, `streamkap_source_postgresql`, `streamkap_destination_snowflake` and `streamkap_destination_postgresql`; the rest of the fleet is untouched. The chain is generated from the backend's published catalog, so `regex_router` and `mask_field` are the types it admits, and the postgresql destination has driven a chain end to end against a live backend. This page records what is integrated, what the tests prove, the decisions the representation and the wire contract force, and what remains.

## Versions and inputs

- `github.com/hashicorp/terraform-plugin-framework v1.19.0`, `terraform-plugin-go v0.31.0`, `terraform-plugin-testing v1.16.0`, Go 1.27.1. Terraform CLI v1.16.3 locally (the `resource.UnitTest` cases shell out to it); CI pins 1.16.1. Write-only attributes need Terraform 1.11 or later, so the cases that configure `smt_secrets[*].values` carry `tfversion.SkipBelow(1.11.0)`; a chain without secrets and every flat configuration work on any version.
- Catalog artifact: `internal/generated/smt_catalog.json`, byte-identical to `cmp/backend/app/services/smt/fixtures/catalog.json` at backend revision `0769c70e77d093f3bca91ba445d67689dbad64d1`, file sha256 `53fb62493a2820f80ae99798e6263668244d09e147a002c75b4f2c471e283f26`, envelope digest `sha256:d480e9429775b68f93bdb6a243873c4e5313834e25aa71d5941c2d060a4acbae`. Both types are publishable: `regex_router` derives the config child `{regex (Required), replacement (Optional + Computed, default "$0")}` and has no secret; `mask_field` derives `{fields_include (Required list), fields_exclude (default []), mask_function (default SHA256_TRUNCATE), mask_char (default "*"), mask_fixed_value (default "***"), replace_null_with_default (default true)}` with its writeOnly `mask_salt` stripped into `smt_secrets` under the pointer `/mask_salt`. The enum on `mask_function` and the item pattern on the field lists are not represented; the backend validates them at apply. The contract corpus with the nested fixture types stays the input of the unit tests (`internal/smt/testdata/contract_corpus.json`), since it exercises rows, escapes and nested defaults the published types do not.
- Connector plugin configs: backend `main` at `9d834aa3f109d6a4db92b0313e157a0ffdaa3ec3`, whose plugin configs are identical to the `85b3218fb` the released files were generated from; the three regenerated files differ from the released ones only by the three model fields.
- Wire contract: `internal/api/testdata/smt_wire_fixtures.json`, byte-identical to `cmp/backend/app/services/smt/fixtures/wire_fixtures.json` at backend revision `580de048f89e925d53689873c3d6fe64009c1e6e` (last changed in `195061ab4`), sha256 `6fbdae97cb50de50670570904f49f4638487cd102dd00b261eedec9ca5af5659`, `wire_version` 1. The read projection now carries the populated `desired.managed[]` rows and an `order[]` that interleaves managed slots with the user instances; the provider decodes the managed rows raw, manages only the `origin: user` entries, and never sends `managed_overrides`, so the issue codes it can raise (`smt_managed_slot_unknown`, `smt_managed_required_locked`, `smt_managed_parity_pending`) are not classified. The 409 code `smt_connector_deleting`, which the routes raise for a write against a destination whose deletion is pending, is classified from the route source (`app/api/smt_chains_api.py`); the fixture does not carry it yet.

## What moved where

- `internal/smt`: `corpus.go` parses the artifact and filters `publishable`; `schema.go` derives one typed config child per type (recursive objects, homogeneous lists, nullable scalars, load-bearing defaults) and rejects every unsupported construct with the type and schema path; `chain.go` builds `smt_chain` and `smt_secrets`; `convert.go` is the typed value codec; `correlate.go` joins plan to state by key; `secrets.go` handles canonical pointers and rotation; `surface.go` is what the base resource composes: `ValidateConfig`, `ModifyPlan`, `PlanWrite`, `Apply`, `Refresh`, the flat-projection strip and the error mapping.
- `cmd/tfgen`: `--smt-catalog` and `--backend-revision` pin the input; `smt_chain_connectors` in `overrides.json` names the connectors whose model carries `smt_chain`, `smt_secrets` and `smt_chain_revision` (`source_postgresql`, `destination_snowflake`, `destination_postgresql`); `smt.go` validates the artifact with the same `smt.BuildChain` the runtime uses before anything is written, refuses `kafkadirect` and transforms, and writes `internal/generated/smt_catalog.{json,go}` with the provenance header. `go generate` and `scripts/codegen-preflight.sh` require both inputs; `.github/workflows/regenerate.yml` resolves the backend ref to one commit, reads the artifact from that commit and passes both through, so the schema authority is never a mutable branch.
- `internal/resource/connector/base.go`: a config implementing `ConnectorConfigWithSMTChain` gets the three attributes added in `Schema`, `ValidateConfig` for the mixed-surface rejection, the chain step of `ModifyPlan`, and Create/Read/Update/Import composing `smt.Surface`. The wrappers `source/postgresql_generated.go`, `destination/snowflake_generated.go` and `destination/postgresql_generated.go` opt in by returning `generated.SMTChain()`.
- `internal/api/smt_chain.go`: the DTOs and `SMTChainAPI`, aligned to the wire fixture, with `DetailJSON` kept on `APIError` so the structured envelope reaches the resource.
- `internal/provider/schema_compat_test.go`: snapshots record `write_only`; `breakingChanges` fails a WriteOnly attribute that becomes stateful and, under `smt_chain.`, a removed child, a Required child that becomes Optional, or a Dynamic type. `TestBreakingChanges_SMTGates` exercises each rule; `TestSMTChainOptInIsConsistent` fails a model and schema that disagree on the opt-in.

## Wire contract alignment

The fixture settled the spellings the prototype had left as placeholders, and three of them changed the representation:

- Secret operations are not a separate endpoint. They ride inside the instance they target (`transforms[i].secret_operations`, `operation: replace|clear`), under the same `expected_revision`. A rotation is therefore a chain write, and an unchanged chain with no rotation sends nothing at all.
- `name` is required on the wire (`minLength: 1`), so `smt_chain[*].name` is Required in the schema rather than Optional.
- `enabled` exists on the instance (default true) and is echoed on read, so `smt_chain[*].enabled` is Optional + Computed + Default(true); without it a UI toggle would be invisible drift.
- The read projection is `desired`/`last_applied`/`application_status`; the provider manages `desired.transforms` and stores `desired.chain_revision` in the computed `smt_chain_revision`. An update sends it as `expected_revision`; a create sends none.
- Error envelopes are `{"detail": {code, message, issues}}`. `smt_revision_conflict` and `smt_chain_exists` are "changed outside Terraform, plan again"; `smt_expected_revision_required` is "the connector already has a chain this resource does not manage, import it"; `smt_connector_deleting` is "the connector is being deleted, plan again once it is gone"; `smt_chain_absent` on read is an empty managed chain with no revision. Every 422 issue is attached to the attribute its runtime pointer locates (`/transforms/1/config/routes/1/endpoint` becomes `smt_chain[1].<type>.routes[1].endpoint`; secret-operation issues land on `smt_secrets`).
- The routes are published for destinations only; `kind` is a path segment and sources answer a plain 404 until the backend wires them. Predicates are accepted by the wire but not represented here: an instance that carries one, or an opaque instance, fails Read with a diagnostic instead of being dropped by the next write.
- `alias`, `schema_version`, `codec_version` and `configured_secret_paths` are server-owned and never sent; `TestSMTWire_RequestsMatchFixture` checks every key the provider emits against the OpenAPI components, which are `additionalProperties: false`.

## What the integrated resource proves

Every case below runs the real `BaseConnectorResource` over the real `PostgreSQLConfig` (generated model included) and, for the destination kind, the real `SnowflakeConfig`, with the chain built from the corpus copy and an in-memory API that echoes JSON, guards writes by revision and validates pointers by row key. "Under terraform" means `resource.UnitTest` drove the CLI.

- Chain lifecycle under terraform (`TestSMT_ChainLifecycle`): create with two typed instances and three secret operations addressed by canonical pointer including both escapes; nested defaults, `retries = 0`, `enabled = false`, `output.prefix = ""`, empty lists and ordered rows in state; an unchanged configuration sends nothing; an edited value without a version bump rotates nothing; reorder plus rename keeps both ids and the alias with their keys and rotates only the bumped key; a key change is a new server instance that does not inherit the old secrets; import derives keys from the server ids and does not populate the flat surface or invent secret entries.
- Rejections before any write (`TestSMT_RejectionsHappenBeforeAnyWrite`): decreased version, increased version without operations, unknown key, index-addressed pointer, `version = 0`, type change under an existing key and flat-plus-chain all fail at plan time with the backend write counter unchanged.
- Rotation version follows the key (`TestSMT_RotationVersionFollowsTheKey`): `secret_version` lives on the chain entry, survives removing the `smt_secrets` entry, refuses a lower re-add, no-ops at the recorded version and rotates once above it.
- Released flat surface unchanged (`TestSMT_FlatSurfaceUnchanged`, `TestSMT_ReleasedStateDecodesAndConverges`): flat HCL with canonical and deprecated attributes applies with the flat keys reaching the API, no chain in state, no chain endpoint called, an empty re-plan, and a flat value edited on the server still showing as drift. A state file with the released attribute set (every resolved default, every alias holding its canonical's default, no chain keys) decodes with the three attributes null, refreshes to an identical state and plans no change on any attribute at the protocol level.
- One owning surface (`TestSMT_FlatProjectionStrippedAndPinned`): with the chain configured no `transforms.*` or `predicates.*` key reaches the API on create or update, the defaulted flat attributes keep their defaults, the others are null, and a backend that echoes the chain through a legacy flat key never gets that value into state on apply or refresh.
- Adoption (`TestSMT_ImportAdoption`, `TestSMT_AdoptionPaths`): an imported chain plans an Update that pins the imported ids, rotates nothing and disturbs no stored secret, then re-plans empty (the Update records the flat defaults the import left null); a connector without a chain imports as an empty managed chain with no revision and the first chain is created without an expected revision; a resource on the flat surface cannot take over a chain the UI created and is told to import.
- Revision guard (`TestSMT_StaleRevisionIsRejected`): a write that lands after an edit elsewhere fails with the stale-revision diagnostic, state keeps the prior chain, the next plan refreshes the revision and the apply goes through.
- Failure containment (`TestSMT_FailedChainWriteAfterCreateIsTainted`, `TestSMT_UnrepresentableAndRejectedWrites`): a chain write that fails after the connector was created leaves the connector recorded and tainted so the next apply replaces it instead of colliding on the name; a 422 is attached to the located attribute and leaves state and server alone; a predicate on a server instance fails refresh.
- Destination kind (`TestSMT_DestinationChain`): the same chain through `streamkap_destination_snowflake` under `destinations`.
- Published types (`TestSMT_CatalogTypesOnDestinationPostgresql`): the real `streamkap_destination_postgresql`, chain built from the pinned catalog rather than the corpus, over the same fake: two `regex_router` instances and a `mask_field` whose salt is rotated through `smt_secrets` at `/mask_salt`; `replacement` and the mask defaults materialised in state and in the stored config with no `mask_salt` leaf; an empty re-plan; reorder plus rename keeping ids and aliases; a stale revision refused and the retry going through; a key change minting a new instance and dropping the old one; a rotation at version 2 executing one replace and recorded on the entry; a decreased version and an unknown pointer rejected at plan; import deriving the keys from the ids with `secret_version` 0 and no secret entries.
- Package-level (`internal/smt`, 18 tests, 52 subtests): value fidelity through the codec and through a plan, correlation, projection with inline operations, pointer escaping and resolution, row existence by declared key, rotation planning, unsupported-construct diagnostics, the map gate, issue-pointer mapping.

## Live acceptance

`TestAccDestinationPostgresqlSMTChain` (`internal/provider/destination_postgresql_smt_test.go`) is gated on `TF_ACC` like the rest of the acceptance tier and runs against whatever `STREAMKAP_HOST` names. It ran against the local stack (`STREAMKAP_HOST=http://127.0.0.1:3011`, credentials `local`/`local`, auth bypassed) with the backend on `ENG-2557/smt-foundation` at `0769c70e7` and again at `580de048f` (managed rows populated on reads), `SMT_CHAIN_MANAGEMENT_ENABLED` on:

- Create: the postgresql destination from `specs/55_SMT_Transforms/30_Destination/postgresql.json` with a two-instance `regex_router` chain. The backend answered `POST /destinations` 202 then `PUT /destinations/{id}/smt-chain` 200; state holds two distinct server ids and aliases, `schema_version` 1, `enabled` true and a `smt_chain_revision`.
- Re-plan: empty.
- Reorder plus rename: one `PUT .../smt-chain` 200 under the expected revision; the ids and aliases stayed with their keys and the revision advanced.
- `smt_chain = []`: one `PUT .../smt-chain` 200 with `transforms: []`; state holds an empty chain with a revision and the next plan is empty.
- Destroy: `DELETE /destinations/{id}` 202, then the read answered 404 and `CheckDestroy` passed.

`go test ./internal/provider/ -run TestAccDestinationPostgresqlSMTChain -v -count=1` with `TF_ACC=1`: `--- PASS: TestAccDestinationPostgresqlSMTChain (47.32s)` at `0769c70e7`, `(43.59s)` at `580de048f`. The connector update between plan and chain write is the base resource's own `PUT /destinations/{id}`, sent on every update as released. Not exercised live: `mask_field` with a secret rotation, import, and a stale revision; those are covered by the fake-backend cases above.

## Decisions the representation forces

- Defaults are load-bearing. Every leaf with a `data_schema.default` is Optional + Computed + Default, an object whose children all default gets an object default, an optional list defaults to `[]`, a nullable scalar is plain Optional. The plan carries the fully defaulted tree; the backend read projection materialises the same defaults (the fixture's `mask_field` read shows `mask_function`, `mask_char`, `replace_null_with_default` filled in), and a backend that dropped one would break every apply.
- The released flat SMT attributes keep their client-side defaults and are present in every plan. The flat-versus-chain conflict is judged on `req.Config` only. When the chain is configured, `ModifyPlan` nulls every flat SMT attribute that would otherwise be unknown, the wire projection drops every `transforms.*` and `predicates.*` key (the set is derived from the runtime field mappings, wrapper aliases and code/name mismatches included), and Create/Update/Read pin those attributes to the plan or to prior state so the API echo cannot move them. `kafkadirect` has no such surface and cannot opt in.
- A null `smt_chain` means the connector stays on the released surface: refresh never populates it, and removing the block from HCL returns the resource to that state without touching the server chain. Import writes `[]` as the adoption marker.
- `smt_chain_revision` is the last-read revision, pinned in `ModifyPlan` when the planned chain equals prior state and nothing rotates, unknown otherwise, null whenever the chain is null. It is what makes stale-write detection possible without a re-read at apply.
- Import assigns keys the user did not choose, the server ids; renaming one later is a remove-and-add on the server. Import does not invent `smt_secrets`.
- Reading secrets back is impossible by construction. The stateful parts are `smt_secrets[*].version`, which the user writes, and `smt_chain[*].secret_version`, which the provider records; monotonicity is judged against the latter.
- The `type` validator enumerates the publishable catalog; with none published its description says so and every value is rejected.
- Row existence for a pointer is checked in `ModifyPlan` when the child is known and again by the backend at apply.

## Adoption HCL

The keys are the ids the import returned; everything else is the server's current configuration. Defaults may be omitted.

```hcl
import {
  to = streamkap_destination_postgresql.orders
  id = "conn-1"
}

resource "streamkap_destination_postgresql" "orders" {
  # released attributes unchanged

  smt_chain = [
    {
      key  = "inst-2"
      type = "regex_router"
      name = "Orders"
      regex_router = { regex = "^public\\.orders$", replacement = "orders" }
    },
    {
      key  = "inst-3"
      type = "mask_field"
      name = "Mask email"
      mask_field = { fields_include = ["email"], mask_function = "REDACT" }
    },
  ]

  # Optional: take the salt under provider control. Executes one rotation.
  smt_secrets = [
    {
      key     = "inst-3"
      version = 1
      values  = [{ pointer = "/mask_salt", value = var.mask_salt }]
    },
  ]
}
```

## What remains

- The fleet: `source_postgresql`, `destination_snowflake` and `destination_postgresql` opt in. Regenerating the rest is a change to `smt_chain_connectors` plus one method per wrapper, gated on the snapshot review of each resource.
- Live coverage: the routes exist for destinations only and behind `SMT_CHAIN_MANAGEMENT_ENABLED`; sources, predicates, the `managed`/`order` rows and `managed_overrides` the backend is adding to the read and write models, and the legacy alias adapter are not represented here (the DTOs ignore the fields they do not decode, so a populated `managed` row does not break a read). The `mask_field` secret lifecycle, import and the stale-revision path have run against the fake backend only. Acceptance on the CLI matrix (1.10.5 rejecting configured write-only values, 1.11.4 and 1.16.1 running the secret lifecycle) has not run.
- The applied projection: `last_applied` and `application_status` are decoded and ignored; the provider manages the desired chain only.
- `golangci-lint` did not run: the installed binary was built with Go 1.26 and refuses the module's Go 1.27.1. `go vet ./...` and `gofmt -l internal cmd` are clean.

## Commands

- `go test ./internal/smt/... -count=1`: 18 tests, 52 subtests, 0 failures.
- `go test ./internal/resource/connector/ -run TestSMT_ -count=1`: 13 tests, 0 failures, about 22s (eleven cases drive terraform).
- `go test ./internal/api/ -run TestSMTWire -count=1` and `go test ./cmd/tfgen/ -run SMT -count=1`: 2 and 3 tests, 0 failures.
- `TF_ACC=1 STREAMKAP_HOST=http://127.0.0.1:3011 STREAMKAP_CLIENT_ID=local STREAMKAP_SECRET=local go test ./internal/provider/ -run TestAccDestinationPostgresqlSMTChain -v -count=1`: 1 test, 0 failures, 47s against the local stack.
- `make test-schema`: 72 passed, 0 failed.
- `make test-all`: every package ok, 1809 passed, 0 failed, 110 skipped (the `TestAcc` tier, gated on `TF_ACC`, plus the pre-existing `-short` skips).
