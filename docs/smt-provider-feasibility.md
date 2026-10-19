# SMT chain on connector resources

The nested SMT chain representation was first proved by a prototype under `internal/smtproto/`. That prototype has moved into the real generator and the real base resource: `internal/smt` is the representation, `cmd/tfgen` pins the catalog it is built from, and `internal/resource/connector` drives it for any generated connector that opts in. Every source and destination except Kafka Direct carries the chain, the set the backend's chain routes admit (23 destinations, 24 sources). The chain is generated per host from the backend's published catalog: destinations admit `regex_router`, `mask_field` and `drop_fields`, sources admit `filter`, and both share the `topic_name_matches` predicate. The postgresql destination has driven a chain end to end against a live backend; no source chain has run against a live backend yet. This page records what is integrated, what the tests prove, the decisions the representation and the wire contract force, and what remains.

## Versions and inputs

- `github.com/hashicorp/terraform-plugin-framework v1.19.0`, `terraform-plugin-go v0.31.0`, `terraform-plugin-testing v1.16.0`, Go 1.27.1. The default local Terraform CLI is v1.16.4; CI pins 1.16.1. Write-only attributes need Terraform 1.11 or later, so the cases that configure `smt_secrets[*].values` carry `tfversion.SkipBelow(1.11.0)`. The version matrix reuses the released flat-surface fixture for the 1.10.5 compatibility case and passes the expected CLI version into an in-process check so a stale `TF_ACC_TERRAFORM_PATH` cannot turn a requested version into false evidence.
- Catalog artifact: `internal/generated/smt_catalog.json`, byte-identical to the backend's `app/services/smt/fixtures/catalog.json` at backend revision `478d0e7416312f8c5269f788873241c5d6fe3e93`, file sha256 `4dce0e8dcad79eff561d6950327192c63e0587928bbd09c752d212036394a7ef`, envelope digest `sha256:14490b96403b4c36f0592777270fe8a6ee9abc75a59386f3d6de9c548cc7f2f9`. All three types are publishable: `regex_router` derives the config child `{regex (Required), replacement (Optional + Computed, default "$0")}` and has no secret; `mask_field` derives `{fields_include (Required list), fields_exclude (default []), mask_function (default SHA256_TRUNCATE), mask_char (default "*"), mask_fixed_value (default "***"), replace_null_with_default (default true)}` with its writeOnly `mask_salt` stripped into `smt_secrets` under the pointer `/mask_salt`; `drop_fields` derives `{fields_include (Required list), fields_exclude (default []), replace_null_with_default (default true)}` and has no secret. The enum on `mask_function` and the item pattern on the field lists are not represented; the backend validates them at apply. The contract corpus with the nested fixture types stays the input of the unit tests (`internal/smt/testdata/contract_corpus.json`), since it exercises rows, escapes and nested defaults the published types do not.
- Connector plugin configs: backend `main` at `9d834aa3f109d6a4db92b0313e157a0ffdaa3ec3`, whose plugin configs are identical to the `85b3218fb` the released files were generated from; the three regenerated files differ from the released ones only by the three model fields.
- Wire contract: `internal/api/testdata/smt_wire_fixtures.json`, copied from the backend fixture named in its provenance header. The read projection carries populated `desired.managed[]` rows and an `order[]` that interleaves managed slots with user instances. The provider manages typed `origin: user` entries and exposes `smt_managed_overrides` as a separate map of stable slot ids to booleans. Omitting that map preserves backend ownership; `{}` restores automatic behavior. Override-only writes omit `transforms` and `predicates`, so they cannot replace the user chain. Selectors remain unsupported and make provider-owned chain refresh fail rather than drop rules on the next write.

## What moved where

- `internal/smt`: `corpus.go` parses the artifact and filters `publishable`; `schema.go` derives one typed config child per type (recursive objects, homogeneous lists, nullable scalars, load-bearing defaults) and rejects every unsupported construct with the type and schema path; `chain.go` builds `smt_chain` and `smt_secrets`; `convert.go` is the typed value codec; `correlate.go` joins plan to state by key; `secrets.go` handles canonical pointers and rotation; `surface.go` is what the base resource composes: `ValidateConfig`, `ModifyPlan`, `PlanWrite`, `Apply`, `Refresh`, the flat-projection strip and the error mapping.
- `cmd/tfgen`: `--smt-catalog` and `--backend-revision` pin the input; the opt-in is derived from the catalog: every source and destination model except `kafkadirect` carries the chain fields and an `SMTChainHost()` method when the catalog publishes a type for its host (`docs/CODE_GENERATOR.md#smt-chain-opt-in`); `smt.go` validates each host's chain with the same `smt.BuildChain` the runtime uses before anything is written, and writes `internal/generated/smt_catalog.{json,go}` with the provenance header. `go generate` and `scripts/codegen-preflight.sh` require both inputs; `.github/workflows/regenerate.yml` resolves the backend ref to one commit, reads the artifact from that commit and passes both through, so the schema authority is never a mutable branch.
- `internal/resource/connector/base.go`: a model with `SMTChainHost()` gets the chain attributes added in `Schema` from `generated.SMTChain(host)`, `ValidateConfig` for the mixed-surface rejection, the chain step of `ModifyPlan`, and Create/Read/Update/Import composing `smt.Surface`. `ConnectorConfigWithSMTChain` remains as the test seam that substitutes the contract corpus. A destination writes the chain with `PUT .../smt-chain` after its connector request; a source embeds the write in its create or update body (`smt_chain`), never retries or logs that body, and reads the chain back under the revision in `smt_chain_receipt`.
- `internal/api/smt_chain.go`: the DTOs and `SMTChainAPI`, aligned to the wire fixture, with `DetailJSON` kept on `APIError` so the structured envelope reaches the resource.
- `internal/provider/schema_compat_test.go`: snapshots record `write_only`; `breakingChanges` fails a WriteOnly attribute that becomes stateful and, under `smt_chain.`, a removed child, a Required child that becomes Optional, or a Dynamic type. `TestBreakingChanges_SMTGates` exercises each rule; `TestSMTChainOptInIsConsistent` fails a model and schema that disagree on the opt-in.

## Wire contract alignment

The fixture settled the spellings the prototype had left as placeholders, and three of them changed the representation:

- Secret operations are not a separate endpoint. They ride inside the instance they target (`transforms[i].secret_operations`, `operation: replace|clear`), under the same `expected_revision`. A rotation is therefore a chain write, and an unchanged chain with no rotation sends nothing at all.
- `name` is required on the wire (`minLength: 1`), so `smt_chain[*].name` is Required in the schema rather than Optional.
- `enabled` exists on the instance (default true) and is echoed on read, so `smt_chain[*].enabled` is Optional + Computed + Default(true); without it a UI toggle would be invisible drift.
- The read projection is `desired`/`last_applied`/`application_status`; the provider manages `desired.transforms` and stores `desired.chain_revision` in the computed `smt_chain_revision`. An update sends it as `expected_revision`; a create sends none.
- Error envelopes are `{"detail": {code, message, issues}}`. `smt_revision_conflict` and `smt_chain_exists` are "changed outside Terraform, plan again"; `smt_expected_revision_required` is "the connector already has a chain this resource does not manage, import it"; `smt_connector_deleting` is "the connector is being deleted, plan again once it is gone"; `smt_chain_absent` on read is an empty managed chain with no revision. Every 422 issue is attached to the attribute its runtime pointer locates (`/transforms/1/config/routes/1/endpoint` becomes `smt_chain[1].<type>.routes[1].endpoint`; secret-operation issues land on `smt_secrets`).
- Sources have a chain read route and catalog routes but no chain write route: the chain rides in the source body. The source routes answer a refused embedded chain with a plain 400 detail, so source diagnostics name the issue codes and pointers but are not attached to chain attributes, and the provider reads an existing source chain before the first write so an unmanaged one is refused with import guidance. Managed-slot overrides are destination-scoped; a source refuses a non-empty `smt_managed_overrides` at validate time. Predicates are accepted by the wire but not represented here: an instance that carries one, or an opaque instance, fails Read with a diagnostic instead of being dropped by the next write.
- `alias`, `schema_version`, `codec_version` and `configured_secret_paths` are server-owned and never sent; `TestSMTWire_RequestsMatchFixture` checks every key the provider emits against the OpenAPI components, which are `additionalProperties: false`.

## What the integrated resource proves

Every case below runs the real `BaseConnectorResource` over test-only wrappers around the released PostgreSQL source and Snowflake destination models, with the chain built from the two-type contract corpus and an in-memory API that echoes JSON, guards writes by revision and validates pointers by row key. Production sources do not implement this chain surface. "Under terraform" means `resource.UnitTest` drove the CLI.

- Chain lifecycle under terraform (`TestSMT_ChainLifecycle`): create with two typed instances and three secret operations addressed by canonical pointer including both escapes; nested defaults, `retries = 0`, `enabled = false`, `output.prefix = ""`, empty lists and ordered rows in state; an unchanged configuration sends nothing; an edited value without a version bump rotates nothing; reorder plus rename keeps both ids and the alias with their keys and rotates only the bumped key; a key change is a new server instance that does not inherit the old secrets; import derives keys from the server ids and does not populate the flat surface or invent secret entries.
- Rejections before any write (`TestSMT_RejectionsHappenBeforeAnyWrite`): decreased version, increased version without operations, unknown key, index-addressed pointer, `version = 0`, type change under an existing key and flat-plus-chain all fail at plan time with the backend write counter unchanged.
- Rotation version follows the key (`TestSMT_RotationVersionFollowsTheKey`): `secret_version` lives on the chain entry, survives removing the `smt_secrets` entry, refuses a lower re-add, no-ops at the recorded version and rotates once above it.
- Released flat surface unchanged (`TestSMT_FlatSurfaceUnchanged`, `TestSMT_ReleasedStateDecodesAndConverges`): flat HCL with canonical and deprecated attributes applies with the flat keys reaching the API, no chain in state, no chain endpoint called, an empty re-plan, and a flat value edited on the server still showing as drift. A state file with the released attribute set (every resolved default, every alias holding its canonical's default, no chain keys) decodes with the three attributes null, refreshes to an identical state and plans no change on any attribute at the protocol level. The CLI compatibility matrix selected and verified the actual executables against provider test source `3da395b`: Terraform 1.10.5 passed this released flat fixture and the framework's configured-write-only refusal with zero API writes, while Terraform 1.11.4 and 1.16.1 each passed chain lifecycle, pre-write rejection and keyed rotation. The run completed 8/8 cases with no failures or skips at `2026-10-09T10:13:48.851Z`. These are real CLI processes over the in-memory HTTP fixture, not `TF_ACC` or live connector deployment.
- One owning surface (`TestSMT_FlatProjectionStrippedAndPinned`): with the chain configured no `transforms.*` or `predicates.*` key reaches the API on create or update, the defaulted flat attributes keep their defaults, the others are null, and a backend that echoes the chain through a legacy flat key never gets that value into state on apply or refresh.
- Adoption (`TestSMT_ImportAdoption`, `TestSMT_AdoptionPaths`): an imported chain plans an Update that pins the imported ids, rotates nothing and disturbs no stored secret, then re-plans empty (the Update records the flat defaults the import left null); a connector without a chain imports as an empty managed chain with no revision and the first chain is created without an expected revision; a resource on the flat surface cannot take over a chain the UI created and is told to import.
- Revision guard (`TestSMT_StaleRevisionIsRejected`): a write that lands after an edit elsewhere fails with the stale-revision diagnostic, state keeps the prior chain, the next plan refreshes the revision and the apply goes through.
- Failure containment (`TestSMT_FailedChainWriteAfterCreateIsTainted`, `TestSMT_UnrepresentableAndRejectedWrites`): a chain write that fails after the connector was created leaves the connector recorded and tainted so the next apply replaces it instead of colliding on the name; a 422 is attached to the located attribute and leaves state and server alone; a predicate on a server instance fails refresh.
- Destination kind (`TestSMT_DestinationChain`): the same chain through `streamkap_destination_snowflake` under `destinations`.
- Published types (`TestSMT_CatalogTypesOnDestinationPostgresql`): the real `streamkap_destination_postgresql`, chain built from the pinned catalog rather than the corpus, over the same fake: two `regex_router` instances and a `mask_field` whose salt is rotated through `smt_secrets` at `/mask_salt`; `replacement` and the mask defaults materialised in state and in the stored config with no `mask_salt` leaf; an empty re-plan; reorder plus rename keeping ids and aliases; a stale revision refused and the retry going through; a key change minting a new instance and dropping the old one; a rotation at version 2 executing one replace and recorded on the entry; a decreased version and an unknown pointer rejected at plan; import deriving the keys from the ids with `secret_version` 0 and no secret entries.
- Package-level (`internal/smt`, 22 top-level tests): value fidelity through the codec and through a plan, correlation, projection with inline operations, pointer escaping and resolution, row existence by declared key, rotation planning, unsupported-construct diagnostics, the map gate and issue-pointer mapping. Key and type expressions that are unknown during configuration validation are correlated only after Terraform resolves them during planning; known empty, duplicate and type-changing identities remain errors, and apply repeats strict correlation.

## Live acceptance

`TestAccDestinationPostgresqlSMTChain` (`internal/provider/destination_postgresql_smt_test.go`) is gated on `TF_ACC` like the rest of the acceptance tier and runs against whatever `STREAMKAP_HOST` names. It ran against a development backend with `SMT_CHAIN_MANAGEMENT_ENABLED` on:

- Create: a postgresql destination with a two-instance `regex_router` chain. The backend answered `POST /destinations` 202 then `PUT /destinations/{id}/smt-chain` 200; state holds two distinct server ids and aliases, `schema_version` 1, `enabled` true and a `smt_chain_revision`.
- Re-plan: empty.
- Reorder plus rename: one `PUT .../smt-chain` 200 under the expected revision; the ids and aliases stayed with their keys and the revision advanced.
- Import/readback: a separate acceptance lifecycle creates one exact API-owned destination and two-instance chain, registers exact-id cleanup immediately, imports into empty Terraform state with persistence, requires every imported key to equal its exact server id, preserves ids, aliases, names, config and revision, invents no `smt_secrets`, applies the same ID-keyed HCL without replacing those identities, then requires an empty plan.
- `smt_chain = []`: one `PUT .../smt-chain` 200 with `transforms: []`; state holds an empty chain with a revision and the next plan is empty.
- Destroy: `DELETE /destinations/{id}` 202, then the read answered 404 and `CheckDestroy` passed.

Both `TestAccDestinationPostgresqlSMTChain` and `TestAccDestinationPostgresqlSMTChainImport` passed, and cleanup returned the destination list to the exact empty pre-test set. The connector update between plan and chain write is the base resource's own `PUT /destinations/{id}`, sent on every update as released. Not exercised live: `mask_field` with a secret rotation and a stale revision; those are covered by the fake-backend cases above.

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

- The fleet: every source and destination except Kafka Direct opts in through tfgen. `smt_predicates` carries no schema default, because a default differs from a released state's null and made the framework plan computed nulls such as `mongodb_connection_hostname` unknown; `TestSMTChainReleasedStatePlansNoChangeOnEveryResource` plans a released state on every chained resource.
- Live coverage: chain management sits behind `SMT_CHAIN_MANAGEMENT_ENABLED`; source chains are covered by the fake API only, and the legacy alias adapter remains outside this provider surface. `managed` and `order` remain server projections. Terraform can own only the explicit `managed_overrides` map, independently of the typed user chain. Selectors remain unsupported and make a provider-owned chain refresh fail because the Terraform schema cannot preserve them. The `mask_field` secret lifecycle, override lifecycle and stale-revision paths have focused local coverage only. Offline compatibility evidence does not establish live rotation, provider publication or registry installation.
- The applied projection: `last_applied` and `application_status` are decoded and ignored; the provider manages the desired chain only.

## Commands

- `go test ./internal/smt/... -count=1`: 22 top-level tests, 0 failures.
- `go test ./internal/resource/connector/ -run TestSMT_ -count=1`: 14 tests, 0 failures.
- `go test ./internal/api/ -run TestSMTWire -count=1` and `go test ./cmd/tfgen/ -run SMT -count=1`: 2 and 3 tests, 0 failures.

- The Terraform CLI compatibility matrix (1.10.5, 1.11.4, 1.16.1) passed 8/8 cases with no failures or skips.

- `TF_ACC=1 go test ./internal/provider/ -run '^(TestAccDestinationPostgresqlSMTChain|TestAccDestinationPostgresqlSMTChainImport)$' -v -count=1` with `STREAMKAP_HOST`, `STREAMKAP_CLIENT_ID` and `STREAMKAP_SECRET` set for a development backend: 2 tests, 0 failures; exact-id cleanup restored the empty pre-test destination set.
- `make test-schema`: 72 passed, 0 failed.
- `make test-all`: every package ok, 1809 passed, 0 failed, 110 skipped (the `TestAcc` tier, gated on `TF_ACC`, plus the pre-existing `-short` skips).

## Destination staging contract checkpoint

The consumer wire fixture now declares the backend’s structured 503 response. Deterministic staging refusal returns 422 before chain acceptance. An unknown commit response requires readback before retrying; it must not be described as a saved chain or a proven no-op. The API fixture is copied byte-for-byte after backend formatting, with its producer revision and digest recorded in the decode test. No Terraform schema or generated resource changes are needed for this response-only update. This checkpoint supplies no new Terraform acceptance or release evidence.
