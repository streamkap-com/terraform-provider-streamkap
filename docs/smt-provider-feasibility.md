# SMT chain on connector resources

The nested SMT chain representation was first proved by a prototype under `internal/smtproto/`. That prototype has moved into the real generator and the real base resource: `internal/smt` is the representation, `cmd/tfgen` pins the catalog it is built from, and `internal/resource/connector` drives it for any generated connector that opts in. Two representative resources are regenerated with the chain enabled, `streamkap_source_postgresql` and `streamkap_destination_snowflake`; the rest of the fleet is untouched. This page records what is integrated, what the tests prove, the decisions the representation and the wire contract force, and what remains.

## Versions and inputs

- `github.com/hashicorp/terraform-plugin-framework v1.19.0`, `terraform-plugin-go v0.31.0`, `terraform-plugin-testing v1.16.0`, Go 1.27.1. Terraform CLI v1.16.3 locally (the `resource.UnitTest` cases shell out to it); CI pins 1.16.1. Write-only attributes need Terraform 1.11 or later, so the cases that configure `smt_secrets[*].values` carry `tfversion.SkipBelow(1.11.0)`; a chain without secrets and every flat configuration work on any version.
- Catalog artifact: `internal/generated/smt_catalog.json`, byte-identical to `cmp/backend/app/services/smt/fixtures/contract_corpus.json` at backend revision `05ee1dd2e26d753c12715805daf4cc63981cfb2a`, file sha256 `5d3528b46923f371ac8d3bb089f117cec53e5bcd96ce93a1f573874e89807ad0`, envelope digest `sha256:50ba3ac5ef381a480cb608a5f5b2c00bacc5c868948337335078ef6d36ca04f7`. Both types in it are `publishable: false`, so the generated chain admits no type until the backend publishes one; the tests build the chain from the same artifact with the exclusion lifted (`internal/smt/testdata/contract_corpus.json`, the same bytes).
- Connector plugin configs: backend `main` at `85b3218fb59fdf6b5e5b4c98dd19eb5da806588b`, which reproduces the released generated files byte for byte; the two regenerated files differ from the released ones only by the three model fields.
- Wire contract: `internal/api/testdata/smt_wire_fixtures.json`, byte-identical to `cmp/backend/app/services/smt/fixtures/wire_fixtures.json` at backend revision `3bd029995a1e23931db0859fbb22f491abaa38c6` (last changed in `d435615bd`), sha256 `3d90dce763b44ae1841e35ee63bada7efd433c1e12e2ec73ee8061f9c8f2f442`, `wire_version` 1.

## What moved where

- `internal/smt`: `corpus.go` parses the artifact and filters `publishable`; `schema.go` derives one typed config child per type (recursive objects, homogeneous lists, nullable scalars, load-bearing defaults) and rejects every unsupported construct with the type and schema path; `chain.go` builds `smt_chain` and `smt_secrets`; `convert.go` is the typed value codec; `correlate.go` joins plan to state by key; `secrets.go` handles canonical pointers and rotation; `surface.go` is what the base resource composes: `ValidateConfig`, `ModifyPlan`, `PlanWrite`, `Apply`, `Refresh`, the flat-projection strip and the error mapping.
- `cmd/tfgen`: `--smt-catalog` and `--backend-revision` pin the input; `smt_chain_connectors` in `overrides.json` names the connectors whose model carries `smt_chain`, `smt_secrets` and `smt_chain_revision`; `smt.go` validates the artifact with the same `smt.BuildChain` the runtime uses before anything is written, refuses `kafkadirect` and transforms, and writes `internal/generated/smt_catalog.{json,go}` with the provenance header. `go generate` and `scripts/codegen-preflight.sh` require both inputs; `.github/workflows/regenerate.yml` resolves the backend ref to one commit, reads the artifact from that commit and passes both through, so the schema authority is never a mutable branch.
- `internal/resource/connector/base.go`: a config implementing `ConnectorConfigWithSMTChain` gets the three attributes added in `Schema`, `ValidateConfig` for the mixed-surface rejection, the chain step of `ModifyPlan`, and Create/Read/Update/Import composing `smt.Surface`. The wrappers `source/postgresql_generated.go` and `destination/snowflake_generated.go` opt in by returning `generated.SMTChain()`.
- `internal/api/smt_chain.go`: the DTOs and `SMTChainAPI`, aligned to the wire fixture, with `DetailJSON` kept on `APIError` so the structured envelope reaches the resource.
- `internal/provider/schema_compat_test.go`: snapshots record `write_only`; `breakingChanges` fails a WriteOnly attribute that becomes stateful and, under `smt_chain.`, a removed child, a Required child that becomes Optional, or a Dynamic type. `TestBreakingChanges_SMTGates` exercises each rule; `TestSMTChainOptInIsConsistent` fails a model and schema that disagree on the opt-in.

## Wire contract alignment

The fixture settled the spellings the prototype had left as placeholders, and three of them changed the representation:

- Secret operations are not a separate endpoint. They ride inside the instance they target (`transforms[i].secret_operations`, `operation: replace|clear`), under the same `expected_revision`. A rotation is therefore a chain write, and an unchanged chain with no rotation sends nothing at all.
- `name` is required on the wire (`minLength: 1`), so `smt_chain[*].name` is Required in the schema rather than Optional.
- `enabled` exists on the instance (default true) and is echoed on read, so `smt_chain[*].enabled` is Optional + Computed + Default(true); without it a UI toggle would be invisible drift.
- The read projection is `desired`/`last_applied`/`application_status`; the provider manages `desired.transforms` and stores `desired.chain_revision` in the computed `smt_chain_revision`. An update sends it as `expected_revision`; a create sends none.
- Error envelopes are `{"detail": {code, message, issues}}`. `smt_revision_conflict` and `smt_chain_exists` are "changed outside Terraform, plan again"; `smt_expected_revision_required` is "the connector already has a chain this resource does not manage, import it"; `smt_chain_absent` on read is an empty managed chain with no revision. Every 422 issue is attached to the attribute its runtime pointer locates (`/transforms/1/config/routes/1/endpoint` becomes `smt_chain[1].<type>.routes[1].endpoint`; secret-operation issues land on `smt_secrets`).
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
- Package-level (`internal/smt`, 18 tests, 52 subtests): value fidelity through the codec and through a plan, correlation, projection with inline operations, pointer escaping and resolution, row existence by declared key, rotation planning, unsupported-construct diagnostics, the map gate, issue-pointer mapping.

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
  to = streamkap_source_postgresql.orders
  id = "conn-1"
}

resource "streamkap_source_postgresql" "orders" {
  # released attributes unchanged

  smt_chain = [
    {
      key  = "inst-2"
      type = "contract_fixture_nested"
      name = "East"
      contract_fixture_nested = {
        routes = [
          { key = "row-east",   endpoint = "example.invalid", label = "east" },
          { key = "row~west/1", endpoint = "other.invalid",   label = "west" },
        ]
      }
    },
    {
      key  = "inst-3"
      type = "contract_fixture_deep"
      name = "Deep"
      contract_fixture_deep = {
        services = [{ key = "svc-a", endpoints = [{ id = "ep/1" }] }]
      }
    },
  ]

  # Optional: take the secrets under provider control. Executes one rotation.
  smt_secrets = [
    {
      key     = "inst-2"
      version = 1
      values  = [{ pointer = "/routes/row-east/token", value = var.east_token }]
    },
  ]
}
```

## What remains

- The fleet: only `source_postgresql` and `destination_snowflake` opt in. Regenerating the rest is a change to `smt_chain_connectors` plus one method per wrapper, gated on the snapshot review of each resource.
- A published catalog: the pinned artifact publishes no type, so the generated chain admits none. The first backend catalog artifact with `publishable: true` types is what makes the attribute usable; regeneration then records its digest and revision.
- Live API and acceptance: no request has been made against a backend. The routes exist for destinations only and behind `SMT_CHAIN_MANAGEMENT_ENABLED`; sources, predicates, managed overrides and the legacy alias adapter are not represented here. Focused acceptance on the CLI matrix (1.10.5 rejecting configured write-only values, 1.11.4 and 1.16.1 running the secret lifecycle) waits on a permitted environment and a published type.
- The applied projection: `last_applied` and `application_status` are decoded and ignored; the provider manages the desired chain only.
- `golangci-lint` did not run: the installed binary was built with Go 1.26 and refuses the module's Go 1.27.1. `go vet ./...` and `gofmt -l internal cmd` are clean.

## Commands

- `go test ./internal/smt/... -count=1`: 18 tests, 52 subtests, 0 failures.
- `go test ./internal/resource/connector/ -run TestSMT_ -count=1`: 12 tests, 0 failures, about 16s (ten cases drive terraform).
- `go test ./internal/api/ -run TestSMTWire -count=1` and `go test ./cmd/tfgen/ -run SMT -count=1`: 2 and 3 tests, 0 failures.
- `make test-schema`: 71 passed, 0 failed.
- `make test-all`: every package ok, 1806 passed, 0 failed, 109 skipped (the `TestAcc` tier, gated on `TF_ACC`, plus the pre-existing `-short` skips).
