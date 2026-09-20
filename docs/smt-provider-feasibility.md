# SMT chain on connector resources: provider feasibility

A prototype under `internal/smtproto/` tests whether the selected nested SMT representation works in the framework version this provider ships with. It is not registered in the provider and regenerates nothing. Its tests are the evidence; this page records what each one proves, what it does not, and which representation decisions the result forces.

## Versions

- `github.com/hashicorp/terraform-plugin-framework v1.19.0` (from `go.mod`). Write-only attributes exist since v1.14; `IsWriteOnly()` is on the attribute interface, and the server nulls write-only values in `PlanResourceChange`, `Create` and `Update` (`internal/fwserver/write_only_nullification.go`).
- `github.com/hashicorp/terraform-plugin-go v0.31.0`, `github.com/hashicorp/terraform-plugin-testing v1.16.0`, Go 1.27.1.
- Terraform CLI: v1.16.3 locally (the `resource.UnitTest` cases shell out to it); CI pins 1.16.1. Write-only attributes need Terraform 1.11 or later, so those cases carry `tfversion.SkipBelow(1.11.0)`.
- Contract corpus copy: `internal/smtproto/testdata/contract_corpus.json`, byte-identical to `cmp/backend/app/services/smt/fixtures/contract_corpus.json` at backend revision `05ee1dd2e`, digest `sha256:50ba3ac5ef381a480cb608a5f5b2c00bacc5c868948337335078ef6d36ca04f7`, two types (`contract_fixture_nested`, `contract_fixture_deep`). Never hand-edited.

## What the prototype is

- `corpus.go` loads the artifact. `schema.go` derives one typed config child per catalog type from `data_schema`, strips `writeOnly` leaves and fails on every unsupported construct with a path. `chain.go` builds `smt_chain` (ordered list; per entry `key`, `type`, `name`, computed `id`, `alias`, `schema_version`, and one child attribute per catalog type) and `smt_secrets` (per entry `key`, stateful `version`, write-only `values` rows of `pointer`, `value`, `clear`).
- `convert.go` is the typed `attr.Value` to wire conversion that replaces `shared.ExtractTerraformValue` for nested data. `correlate.go` joins plan to prior state by key. `secrets.go` handles canonical pointers and rotation. `resource.go` is a resource whose schema is the released `streamkap_source_postgresql` schema plus the two attributes, with the flat attributes passed through and the chain driven over a `Backend` interface the tests fake in memory.
- Nothing in the prototype touches `internal/generated/`, tfgen, or a released resource. The released schema is reused by calling `source.NewPostgreSQLResource().Schema`.

## Invariants

Each entry names the test that shows it. "Under terraform" means a `resource.UnitTest` case drove the real CLI against the in-process provider and the in-memory backend; everything else is a Go-level test over real framework values.

- Value-state fidelity: HOLDS.
  - `TestValueFidelity_RoundTrip` round-trips whole-chain null, empty chain, nested object null and known, nullable scalar null and known, `retries = 0`, `enabled = false`, `output.prefix = ""`, empty and null scalar lists, a scalar list, an empty object list and a two-row object list in a fixed order, through `ToWire`, a JSON encode/decode and `FromWire`, and requires `Equal`. Empty and null lists stay distinct on the wire (`[]` versus absent). Unknown anywhere is refused with the path (`ToWire: v.routes[0].endpoint is unknown`).
  - `TestValueFidelity_ReadProjection` decodes both corpus `examples.read_projection` shapes into the typed child and encodes them back to the same bytes.
  - `TestValueFidelity_NullVsUnknownThroughPlan` writes an entry whose `id`, `fallback` and `labels` are unknown and whose `output` and `routes` are null into a `tfsdk.Plan` and reads it back: each is still exactly unknown or exactly null. A whole-chain unknown survives the same way. Null and unknown remain distinguishable end to end.
  - `TestReleasedMarshalerCollapsesStates` records why the released reflection path cannot carry this: it returns nil for both a null and an unknown list, drops every object row and does not handle objects at all.
  - `TestResource_ChainLifecycle` shows the same states under terraform in state: `retries = 0`, `enabled = false`, `output.prefix = ""`, `labels.# = 0`, `fallback` absent, rows in order.
- Stable instance keys: HOLDS.
  - `TestCorrelate` is table-driven over unchanged, reorder, key change, type change, duplicate key, empty key and removal. A matched key keeps its id and alias; a changed key has no id; a type change under a matched key is an error whose detail names the server id and says to remove the entry in one apply and add the new type under a new key.
  - `TestResource_ChainLifecycle` under terraform: reorder plus display-name rename keeps both ids and the alias with their keys; a key change gets a new server id, keeps the untouched key's id, leaves two instances on the server and rotates under the new id only. `TestResource_RejectionsHappenBeforeAnyWrite` shows the type change fails at plan time.
  - Mechanism: the framework marks every computed nested attribute that is null in config as unknown whenever anything in the resource changes, and it does so by list index. `UseStateForUnknown` would therefore pin the wrong id after a reorder. `ModifyPlan` instead correlates by key and sets `id`, `alias` and `schema_version` for matched entries; new entries stay unknown until apply.
- Wire projection: HOLDS.
  - `TestProject` marshals the DTOs: a matched entry echoes its id, a new entry has no `id` key, `name` is present only when configured, `type` is sent, `config` is the unwrapped child equal to the corpus read projection, and `key`, `alias`, `schema_version` and the per-type child names are absent.
  - `TestProject_RejectsWrongOrMissingChild` rejects a null child for the selected type and a populated child for another type.
  - `TestResource_RejectionsHappenBeforeAnyWrite` under terraform: a configuration with both `smt_chain` and `transforms_value_to_key_fields_include_list` fails in `ValidateConfig`, and the backend write counter is unchanged after every rejected step.
- Import and adoption: HOLDS for the chain, NOT PROVEN for the flat attributes.
  - `TestImportKeys`: keys are the server ids verbatim, so the same ids derive the same keys and two instances cannot share one; a duplicate or empty id is an error.
  - `TestResource_ChainLifecycle` import step under terraform: the imported state carries both instances with `key == id` and no `smt_secrets`.
  - `TestResource_ImportAdoption` under terraform: a connector created outside terraform is imported, HCL using the derived keys plans an Update whose chain entries keep `inst-2` and `inst-3`, the apply issues no secret operation and disturbs no stored secret, and the plan after it is empty.
  - `TestReleasedStateDecodesAndConverges` at the protocol level: a released state refreshes with the chain still null; the chain is never populated behind a flat-managed connector.
  - Not proven: the prototype's Read does not fill the flat attributes from the backend, so the adoption plan is an Update because the required flat attributes are null after import. Whether the integrated resource ends up with both surfaces populated after import depends on whether the backend echoes chain-managed instances through the legacy `transforms.*` config keys; see the decisions below.
- Write-only secrets and rotation: HOLDS.
  - `TestResource_ChainLifecycle` under terraform: `smt_secrets[*].values` is null in the pre-apply plan (`plancheck.ExpectKnownValue` with `knownvalue.Null()`) and in state (`statecheck` plus `TestCheckNoResourceAttr("smt_secrets.0.values.#")`); `version` is in state. The initial apply executes exactly three operations, addressed by server instance id and canonical pointer, including `/routes/row~0west~11/token` for the `row~west/1` row and `/services/svc-a/endpoints/ep~11/secret` for the nested `ep/1` row. Re-applying the same configuration with the values still written executes nothing. Editing only a value without bumping the version executes nothing; this is the documented limitation, tested as a no-op. Bumping `a` to version 2 executes `a`'s two rows once and nothing for `b`.
  - The last applied rotation version is recorded on the chain entry (`smt_chain[*].secret_version`, computed, re-pinned by key in `ModifyPlan` like `id`), not inferred from the `smt_secrets` entry in state. `TestResource_RotationVersionFollowsTheKey` under terraform: rotate `a` to 5, remove its `smt_secrets` entry (the entry is gone, `secret_version` stays 5, nothing rotates), re-add at version 1 (rejected: `below the 5 last applied to "a"`), re-add at 5 (no-op), bump to 6 (exactly one operation). A changed key still starts from zero because it is a new instance (`TestResource_ChainLifecycle`, key change step).
  - `TestPlanSecretOps` is table-driven: version below 1, decreased version, removed-then-re-added entry below the chain's version, removed entry leaving the version alone, unchanged version with and without rows, increased version with no rows, unknown key, pointer outside the type's secret paths, pointer of another type, value and clear together, neither, and the two-level keyed pointer. It also checks the version map the chain records afterwards.
  - `TestResource_RejectionsHappenBeforeAnyWrite` under terraform: decreased version, increased version without rows, unknown key, index-addressed pointer (`/routes/1/token`) and `version = 0` all fail at plan time with no backend write.
  - `TestPointerEscaping`, `TestResolvePointer`, `TestRowExists_UsesDeclaredRowKeys`: `~0`/`~1` escaping both ways, templates resolved per type, rows looked up by the `row_keys` property (`key` at `/routes`, `id` at `/services/*/endpoints`), never by index.
- Released v3 compatibility: HOLDS, with the two proofs covering different layers.
  - `TestReleasedStateDecodesAndConverges` is the framework in isolation, no CLI. The fixture is the released schema's own attribute set with the resolved default of every Default-bearing attribute (34 of them, the flat SMT defaults included), the required attributes, two flat SMT values without a default and two that override a default (`transforms_oversized_records_max_field_size_bytes = 2097152`, `transforms_oversized_records_oversized_field_behavior = "NULLIFY"`), and no chain keys. It goes through the prototype's `UpgradeResourceState`, decodes with `smt_chain` and `smt_secrets` null and every populated value present, refreshes to an identical state, and, with a proposed new state built the way core builds it (config where set, prior state for computed attributes, null otherwise), plans a state equal to the prior on every attribute. What it does not prove is core's own config/prior merge, since that merge is hand-built here.
  - `TestResource_FlatSurfaceUnchanged` under terraform is the real-CLI proof of flat-surface convergence: flat HCL using canonical and deprecated-alias attributes applies with the chain attributes present in the schema, has no chain in state, and re-plans empty.
  - `TestSchemaBackwardsCompatibility_SMTPrototype` in `internal/provider`: the snapshot at `testdata/schemas/smtproto_source_postgresql_v0.json` records `write_only` for the four secret rows; `checkAttributeCompatibility` now fails with `BREAKING CHANGE (SECURITY)` when a snapshot says WriteOnly and the schema does not, and the test asserts the flag directly. Removing `WriteOnly` from `smt_secrets.values.value` was tried once and produced that failure; the change was reverted. Released snapshots are unchanged because the flag is omitted when false.
- Generation-time diagnostics: HOLDS. Homogeneous maps: WORK at the value level, kept out of the config child.
  - `TestBuildConfigChild_RejectsUnsupportedConstructs`: polymorphic `anyOf`, a polymorphic `type` list, `oneOf`, `prefixItems`, an `items` array, a `$ref` (the contract inlines local refs, so a survivor is a recursive reference), a union inside a list row, a bare secret list and a map inside a config child each fail with the construct name and the schema path, and the error never mentions a dynamic type. `TestBuildConfigChild_CorpusShapes` asserts no path in either corpus type has a `Dynamic` type and that the schema passes the framework's own `ValidateImplementation`.
  - `TestBuildMapAttribute_Gate` and `TestValueFidelity_MapGate`: `additionalProperties` objects become `MapAttribute` or `MapNestedAttribute`, and null, empty and populated maps round-trip. No corpus type needs a map, keyed secret addressing has no row key inside a map, and the walk refuses a map inside a child, so maps stay a separate admission decision.

## Decisions the representation forces

- Defaults are load-bearing. Every leaf with a `data_schema.default` is `Optional + Computed + Default`; an object whose children all default gets an object default; an optional list defaults to `[]`; a nullable scalar with `default: null` is plain Optional. The plan therefore carries the fully defaulted tree, and the apply is consistent only if the backend's read projection materialises the same defaults, which the corpus examples do (`output: {prefix: "", label: "", retry_limit: 0}`, `labels: []`, `endpoints: []`). A backend that drops a defaulted key on read breaks every apply with "inconsistent result after apply". This must be a contract rule, not a hope.
- `name` is Optional and not Computed, so the backend must echo exactly what it received, null included. If the server ever assigns a display name, the attribute has to become Optional + Computed.
- The released flat SMT attributes carry client-side Defaults (`transforms_source_regex_support_regex_replacement = "_REGEX_"`, `transforms_oversized_records_replace_null_with_default = true`, and more). They are present in every plan and state whether or not the user wrote them. Consequences: the flat versus nested conflict can only be judged on `req.Config`, which is what the prototype does; state cannot be used to prove a connector is on one surface; and when `smt_chain` is set, the wire projection must drop the `transforms.*` and `predicates.*` keys from the flat config map (derive the list from the field mappings, as `FlatSMTAttributes` does), or the backend must define precedence. The prototype does not exercise the flat wire path, so this is a requirement, not a proof.
- Import assigns keys the user did not choose: the server ids. Adoption HCL must use those keys; renaming one later is a remove-and-add on the server, by design. Import does not invent `smt_secrets`; the user adds entries at `version = 1` with values to put the secrets under provider control, which executes one rotation.
- Refresh never populates a null chain. A connector on the released surface keeps `smt_chain = null` forever unless it is imported (import writes an empty chain as the adoption marker) or the user configures `smt_chain`.
- Reading secrets back is impossible by construction, so a rotation done outside Terraform is invisible and produces no diff. The stateful parts are the `smt_secrets[*].version` the user writes and the provider-owned `smt_chain[*].secret_version` it was last applied as; the latter is what monotonicity is judged against, so dropping and re-adding a secret entry cannot restart a live instance below its recorded version.
- Write-only attributes require Terraform 1.11 or later. The acceptance matrix still runs 1.0.11 through 1.10.5; on those versions a non-null `smt_secrets[*].values` is rejected by the framework. A chain without secrets works on any version.
- The `type` validator is `OneOf` over the catalog. The product build must exclude `publishable: false` types from that list; the prototype admits the fixture types because they are its only input.
- Row existence for a pointer is checked in `ModifyPlan` when the child is known and again by the backend at apply. A pointer to a row that only becomes known at apply is accepted at plan and can fail at apply.

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

## Not covered

- No live API, no acceptance run, no runtime evidence. The backend in every test is `fakeBackend` in `fake_backend_test.go`; the public HTTP shape is not frozen, so the prototype talks to a `Backend` interface carrying the DTOs.
- The flat attributes pass through plan and state; the released marshaling, deprecated-alias plan alignment and secret preservation are not replicated and not re-proven here.
- `golangci-lint` did not run: the installed binary was built with Go 1.26 and refuses the module's Go 1.27.1. `go vet ./...` is clean.

## Commands

- `go test ./internal/smtproto/... -v -count=1`: 23 tests, 52 subtests, 0 failures, about 10s (five cases drive terraform).
- `make test-schema`: 69 passed including `TestSchemaBackwardsCompatibility_SMTPrototype`, 0 failed.
- `make test-all`: every package ok, 423 passed, 0 failed, 109 skipped (the `TestAcc` tier, gated on `TF_ACC`, plus the pre-existing `-short` skips).
