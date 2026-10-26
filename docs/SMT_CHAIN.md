# SMT chain on connector resources

Maintainer notes for the nested single message transform (SMT) chain that source and destination resources expose as `smt_chain`, `smt_secrets`, `smt_predicates`, `smt_predicate_secrets`, `smt_managed_overrides` and `smt_chain_revision`. Which connectors carry the chain and how tfgen derives that opt-in is in [CODE_GENERATOR.md](CODE_GENERATOR.md#smt-chain-opt-in); where the chain write sits in Create and Update is in [ARCHITECTURE.md](ARCHITECTURE.md#crud-flow).

Code map:

- `internal/smt`: the representation. `corpus.go` parses the catalog and selects each type's current release; `schema.go` derives the typed config child of a type; `chain.go` builds the chain, predicate and secret attributes and converts between state and the wire; `convert.go` is the value codec; `correlate.go` joins plan to state by key; `secrets.go` resolves pointers and plans rotations; `surface.go` is what the connector resource composes (`ValidateConfig`, `ModifyPlan`, `PlanWrite`, `ApplyGraph`, `RefreshGraph`, error mapping).
- `cmd/tfgen/smt.go`: validates the catalog with the runtime builder and writes `internal/generated/smt_catalog.{json,go}`.
- `internal/resource/connector/base.go`: adds the attributes to an opted-in model's schema and drives the chain in each CRUD step.
- `internal/api/smt_chain.go`: the wire DTOs and `SMTChainAPI`.

## Schema from the pinned catalog

The chain schema is not emitted into connector files. `internal/generated/smt_catalog.json` is a byte-identical copy of the backend's catalog artifact, and the header of `internal/generated/smt_catalog.go` records its provenance: the artifact sha256, the envelope digest, the backend revision it was copied from, and each host's authorable types. Never edit either file; regenerate them as described in [CODE_GENERATOR.md](CODE_GENERATOR.md#smt-chain-opt-in).

At runtime `generated.SMTChain(host)` builds the host's chain once from the catalog's publishable, authorable types for that host (`destinations` or `sources`). Transform types become config children of `smt_chain` entries; predicate types become config children of `smt_predicates` entries. The `type` validators enumerate exactly those ids.

- Releases: a type can have several releases, identified by role, type id, schema version and codec version. New instances are authored against the release the catalog marks current. A persisted instance keeps its release, and Read refuses one this provider does not carry, or whose Terraform shape or write semantics differ from the current release, instead of rewriting it.
- Derivation (`schema.go`): each leaf with a `default` is Optional + Computed + Default; an object whose children all default gets an object default; an optional list defaults to `[]`; a nullable scalar is plain Optional. `writeOnly` leaves are removed from the config child and become secret pointers. A construct the typed representation cannot carry (polymorphic union, tuple, surviving `$ref`, homogeneous map) fails generation with the type id and schema path; nothing falls back to a dynamic type.
- Semantic constraints such as `enum`, `pattern`, bounds and `format` are not plan-time validators. The backend validates them at apply, and the error mapping below locates the issue.
- Every chain write carries `catalog_precondition`: the contract version and digest of the pinned catalog release the write was rendered against.
- Unit tests build their chains from `internal/smt/testdata/contract_corpus.json`, which exercises rows, pointer escapes and nested defaults the published types do not. `ConnectorConfigWithSMTChain` is the seam that substitutes it.

## Wire contract alignment

`internal/api/testdata/smt_wire_fixtures.json` is a copy of the backend's wire fixture; `internal/api/smt_chain.go` names its provenance and the decode test pins the DTOs to it. `TestSMTWire_RequestsMatchFixture` checks every key the provider emits against the fixture's OpenAPI components, which are `additionalProperties: false`. The contract shaped the representation as follows:

- Secret operations are not a separate endpoint. They ride inside the instance or predicate they target (`secret_operations`, `operation: replace|clear`) under the same `expected_revision`, so a rotation is a chain write. An unchanged chain with no rotation sends nothing.
- `name` is required on the wire, so `smt_chain[*].name` is Required. `enabled` defaults to true and is echoed on read, so it is Optional + Computed + Default(true) and a toggle made elsewhere shows as drift.
- The read projection is `desired`, `last_applied` and `application_status`. Every connector reads under a `desired.chain_revision`, an opaque token, whether or not it has user transforms. The provider manages `desired` only and stores the revision in `smt_chain_revision`. Every write to an existing connector sends `expected_revision`; only a connector create, which embeds its first chain, sends none. `managed` and `order` remain server projections.
- An instance id is its transform alias and a predicate id is its predicate alias, so both stay stable for as long as the alias exists.
- `alias`, `schema_version`, `codec_version` and `configured_secret_paths` are server-owned and never sent.
- Errors are `{"detail": {code, message, issues}}`, from the chain route and from a connector route that carried an embedded chain alike. `smt_revision_conflict` means the chain changed outside Terraform, unless its issue is `smt_expected_revision_required`: the write lacked its revision, which is a provider defect. `smt_connector_deleting` means the deletion is pending. A 503 `smt_read_failed` on a write leaves the outcome unknown (the server answers it whether loading the connector failed before saving or the saved chain could not be read back), so the diagnostic asks for a refresh. `smt_order_rejected` arrives as an issue of `smt_validation_failed` and points at the entry that must move. A 409 `smt_view_invalid`, on a chain read or any connector save, means the connector's stored chain metadata is malformed: nothing succeeds until the server side is repaired, so it is reported as that rather than as drift.
- Validation issues carry a runtime pointer into the request body and are attached to the attribute it locates: `/transforms/1/config/routes/1/endpoint` becomes `smt_chain[1].<type>.routes[1].endpoint`, `/predicates/i/config/...` lands in `smt_predicates[i].<type>`, secret-operation issues land on `smt_secrets` or `smt_predicate_secrets`, and `/managed_overrides` on `smt_managed_overrides`.
- An instance the schema cannot represent fails Read with a diagnostic rather than being dropped by the next write: an `opaque` instance, an unknown instance kind, or a predicate attachment outside the returned predicate pool.
- A selector is write-only and Terraform has no attribute for it. The server compiles a selector saved elsewhere into the instance's `fields_include` and `fields_exclude` and reads back only those lists (the read's `selector` is always null), so a refresh shows them as drift on the plain attributes, and HCL that carries the compiled lists converges.

## Source and destination write paths

A create embeds the chain in the connector request as `smt_chain`, for sources and destinations alike, so a refused chain creates no connector. After an embedded write the chain is read back for state.

Every write to an existing connector reads the chain first (`Surface.ExpectedRevision`), before the connector request:

- When state holds a revision and the chain now reads under a different one, the apply is refused as a change outside Terraform and nothing is sent. The write carries the revision it read, which the server compares again atomically.
- When state holds no revision, as on the first chain of a resource that was on the flat surface, the write takes the current revision. If the connector already has user transforms or predicates and the write replaces the user chain, it is refused with import guidance instead, since the replacement would delete them. An override-only write replaces no transforms and proceeds.
- A write that sends nothing reads nothing.

The chain rides in the connector create or update body for sources and destinations alike, so one request saves the connector settings and the chain, and a refused chain saves neither. Destinations also have a chain write route, `PUT /destinations/{id}/smt-chain`, which the provider does not use: Terraform's Update sends the connector body whatever changed, so embedding keeps every update to one request, a chain-only change included, without a second code path. Sources have chain read and catalog routes but no chain write route.

- A body carrying a chain, on either connector type, carries secret values, so it is logged without the chain and never retried; a replay would collide on the name or be refused as stale.
- A source response carries `smt_chain_receipt.chain_revision`. The provider reads the chain back and records it only if it is that revision; a different revision is reported as a change outside Terraform, and a missing receipt as an unconfirmed outcome. A destination response carries no receipt, so its read-back is recorded as is; on an update an edit landing between the save and the read-back cannot be told apart, as with the chain route, whose response is also read after the save.
- A refusal with a structured SMT envelope is mapped onto the chain attributes; any other refusal is reported against the connector, because the same request carried the connector settings.
- Managed slots are destination-scoped, so `ValidateConfig` refuses a non-empty `smt_managed_overrides` on a source.

## Decisions the representation forces

- Correlation is by provider `key` only (`correlate.go`). Position, display name and server ids play no part: a matched key keeps its server id and alias across reorder and rename, a new key creates a new server instance that inherits no secrets, and a type change under an existing key is refused at plan time.
- Defaults are load-bearing. The plan carries the fully defaulted tree and the backend read projection materialises the same defaults; a backend that dropped one would show a diff on every plan.
- Flat and nested surfaces never mix. The released flat `transforms_*` and `predicates_*` attributes keep their client-side defaults, so the conflict is judged on configuration only: `ValidateConfig` refuses a flat attribute set alongside `smt_chain`. With the chain configured, `ModifyPlan` nulls flat attributes that would otherwise be unknown, `StripFlat` removes every `transforms.*` and `predicates.*` wire key (derived from the runtime field mappings, so wrapper aliases are covered), and Create, Update and Read pin those attributes to the plan or prior state so the API echo cannot move them.
- A null `smt_chain` is the released surface: refresh never populates it, and removing the attribute from configuration returns the resource to that surface without touching the server chain. Import writes `[]` as the adoption marker.
- `smt_chain_revision` is the last-read revision, an opaque token. `ModifyPlan` keeps it when the planned chain, predicates and overrides equal prior state and nothing rotates; otherwise it is unknown. It is null while neither the chain nor overrides are managed. The read before a write compares it, so an edit made after the plan's refresh is refused instead of overwritten.
- `smt_predicates` has no schema default: a default differs from the null a released state holds and makes the framework plan computed nulls unknown, so released resources would never converge. When the chain is configured and `smt_predicates` is not, the plan sets `[]`. `TestSMTChainReleasedStatePlansNoChangeOnEveryResource` plans a released state on every chained resource.

## Predicates

`smt_predicates` holds shared, immutable, typed predicate records; a chain entry attaches one through `predicate = { key, negate }`.

- Validation before any write: the attached key must exist, the predicate type must list the transform type in its `compatible_transform_types`, `negate` needs `supports_negate`, every predicate must be attached to at least one transform, and predicates need a managed `smt_chain`.
- Records are immutable. Editing the config of a retained predicate is refused; add the edited predicate under a new key and move the attachments.
- On the wire a retained predicate is sent as its id only and a new one as `request_key`, type and config; attachments reference either the retained id or the new request key. After the write, returned predicate ids are mapped back to provider keys through the attachments, and an ambiguous mapping is an error.

## Managed overrides

The backend adds platform-managed transform slots to a destination's effective chain. `smt_managed_overrides` maps a stable slot id to a boolean override.

- Omitting the attribute preserves server ownership; `{}` restores automatic behavior.
- Overrides are independent of the user chain. An override-only write omits `transforms` and `predicates`, so it cannot replace the user chain, and with `smt_chain` null a refresh reads back only the overrides and the revision.

## Write-only secrets and rotation

Secret leaves never appear in the config child, in plan or in state. They are written through `smt_secrets` (per `smt_chain` key) and `smt_predicate_secrets` (per `smt_predicates` key): a stateful `version` plus write-only `values` rows of `{pointer, value}` or `{pointer, clear = true}`. Write-only attributes need Terraform 1.11 or later.

- A pointer is one of the type's catalog `secret_paths`, with each `*` replaced by the escaped row key of that list row (`~` as `~0`, `/` as `~1`). Index addressing and pointers outside the secret paths are refused. `mask_field` has one: its `mask_salt` leaf is written at `/mask_salt`.
- `smt_chain[*].secret_version` records the last applied rotation and follows the key, so removing and re-adding an `smt_secrets` entry cannot restart below it. A lower version is refused, an equal version executes nothing even if the values changed, and a higher version executes every configured row exactly once; a higher version without values is refused. Secret input still unknown at apply is refused rather than sent.
- Predicate secrets are applied only when the predicate key is created, and a new predicate whose type requires secrets must supply them. Rotating a retained predicate's secret is refused: add the predicate under a new key.
- Secret values are never read back. The server reports which paths hold a secret in `configured_secret_paths`.
- `ModifyPlan` checks that a pointer names a row the planned config has, when that config is known; the backend checks it again at apply.

## Adoption

Import writes the empty chain marker, and the Read that follows adopts the server instances and predicates under keys equal to their server ids. Renaming such a key later is a remove-and-add on the server. Import does not invent `smt_secrets`, so `secret_version` starts at 0.

The keys are the ids the import returned; everything else is the server's current configuration, and defaulted attributes may be omitted:

```hcl
import {
  to = streamkap_destination_postgresql.orders
  id = "<destination id>"
}

resource "streamkap_destination_postgresql" "orders" {
  # released attributes unchanged

  smt_chain = [
    {
      key          = "<server id of the router>"
      type         = "regex_router"
      name         = "Orders"
      regex_router = { regex = "^public\\.orders$", replacement = "orders" }
    },
    {
      key        = "<server id of the mask>"
      type       = "mask_field"
      name       = "Mask email"
      mask_field = { fields_include = ["email"], mask_function = "REDACT" }
    },
  ]

  # Optional: put the salt under Terraform control. Executes one rotation.
  smt_secrets = [
    {
      key     = "<server id of the mask>"
      version = 1
      values  = [{ pointer = "/mask_salt", value = var.mask_salt }]
    },
  ]
}
```

The first apply after import plans an Update, because import leaves the flat SMT attributes null and their defaults fill them; that Update pins the imported ids, leaves the chain unchanged and rotates nothing unless `smt_secrets` asks for it. The next plan is empty. A connector without user transforms imports as an empty managed chain under its current revision, and its first chain is written under that revision. A resource on the flat surface cannot take over transforms created elsewhere; it is told to import.
