# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Documentation map

| Doc | What it covers |
|---|---|
| `README.md` | User-facing intro, install, basic usage. |
| `AGENTS.md` | AI-agent guide for *consumers* of the provider — resource catalog, common patterns. |
| `docs/index.md` | Auto-generated registry landing page (do not hand-edit; regen via `go generate`). |
| `docs/resources/`, `docs/data-sources/` | Auto-generated per-resource pages (regen via `go generate`). |
| `docs/ARCHITECTURE.md` | Layered diagram + CRUD flow + design rationale. |
| `docs/CODE_GENERATOR.md` | tfgen internals: parser, generator, overrides.json, deprecations.json, "adding a new connector" walkthrough. |
| `docs/MIGRATION.md` | v2 → v3 deprecations, removed attributes, action items for users. |
| `docs/audits/<date>/`, `docs/plans/<date>-*.md` | Point-in-time audit reports and execution plans. Append new files; do not rewrite history. |
| `CHANGELOG.md` | User-visible changes per release. Update before tagging. |
| `env.md` | Local env-var setup notes. |

### Keep docs in sync

When you change something in this list, update the matching doc in the same PR:

| Change | Update |
|---|---|
| Add/remove a resource or data source | `AGENTS.md` resource tables, `docs/ARCHITECTURE.md` counts, run `go generate` for `docs/resources/` and `docs/index.md`. |
| Add a connector via tfgen | run `go generate ./...`; if the override system grew (new `map_string`/`map_nested` case) update `docs/CODE_GENERATOR.md`. |
| Add a deprecated alias | add to `internal/provider/v2_backward_compat_test.go` and `docs/MIGRATION.md`. |
| Plan to remove a deprecated attribute | move it into the "Deprecated Attribute Removal" table in `docs/MIGRATION.md`. |
| Change tfgen type mapping or special-case rules | update the table in `docs/CODE_GENERATOR.md` and the table in this file. |
| Change CRUD flow, retry, or pagination behavior | update `docs/ARCHITECTURE.md` and the API quirks list below. |
| User-visible behavior change | add an entry to `CHANGELOG.md`. |
| New env var or test command | update `env.md` and `GNUmakefile`. |

If a doc is wrong, fix it — don't write a parallel one. New doc files belong only in `docs/audits/` (dated audit reports) and `docs/plans/` (dated execution plans).

## Branches

- **`main`** — v2.x stable (current released line). Bug fixes and back-compatible changes only.
- **`develop`** — v3.x beta (new). Default target for feature branches and PRs.

**Do not merge `develop` into `main`** until v3 is promoted from beta to stable. Until then, treat the two lines as independent: a fix that needs to ship to v2 users goes to `main` directly (and may need a separate cherry-pick to `develop`); v3-only work stays on `develop`. Cut feature branches from the line you're targeting.

## Public repository — content hygiene

This repo is public. Do not commit internal ticket IDs/URLs, customer or tenant identifiers, real credentials, internal hostnames, verbatim production traces, or unannounced roadmap details. Public GitHub issue numbers (`#75`) are fine. Prefer `https://api.streamkap.com` and `https://docs.streamkap.com` when referencing surfaces. A previous sweep (`chore: scrub internal references for public repo hygiene`) cleaned existing drift; flag any new occurrences you find.

## Related backend

The provider is built against the Streamkap Python FastAPI backend. Set `STREAMKAP_BACKEND_PATH` to your local clone — `cmd/tfgen` reads `configuration.latest.json` plugin specs from there. OpenAPI: `https://api.streamkap.com/openapi.json`.

Backend areas worth knowing:
- `app/api/{sources,destinations,kafka_access,auth}_api.py` — endpoint definitions
- `app/models/api/{sources,destinations,kafka_access,app_auth}/` — Pydantic request/response models
- `app/{sources,destinations}/plugins/<connector>/` — `configuration.latest.json` (schema source for tfgen) and `dynamic_utils.py`
- `app/utils/entity_changes.py` — CRUD logic and `created_from` handling

## Commands

Use `make help` for the full list. Common ones:

| Make target | What it does |
|---|---|
| `make install` | `go install .` to `$GOBIN` (used by `dev_overrides`) |
| `make test` | Unit tests (`-short`, no API) |
| `make test-all` | Unit + schema-compat + validators + integration (VCR) |
| `make testacc` | Acceptance tests, `TF_ACC=1`, ~15m, hits real API |
| `make test-migration` | v2→v3 migration acceptance tests |
| `make cassettes` | Re-record VCR cassettes (`UPDATE_CASSETTES=1`) |
| `make snapshots` | Update schema-compat snapshots after intentional schema changes |
| `make sweep` | Clean orphaned test resources |
| `make lint` / `make fmt` | golangci-lint / gofmt |

Schema regeneration: `STREAMKAP_BACKEND_PATH=/path/to/python-be-streamkap go generate ./...` (or run `cmd/tfgen` directly per-connector with `--entity-type sources --connector postgresql`).

`.env` is auto-loaded by tests via godotenv. For Snowflake PEM keys (multiline), `source scripts/load-pem-keys.sh`.

Local dev override in `~/.terraformrc`:
```hcl
provider_installation {
  dev_overrides { "github.com/streamkap-com/streamkap" = "<your $GOPATH/bin>" }
  direct {}
}
```

## Architecture

### Layers
- `internal/provider/` — provider entrypoint, resource/datasource registration, `*_test.go` (acceptance + integration + schema-compat).
- `internal/api/` — `StreamkapAPI` HTTP client. All requests funnel through `doRequest` (bearer token, error unwrapping from `detail`, retry with backoff in `retry.go`). Create operations inject `created_from: TERRAFORM`.
- `internal/resource/` — connector resources (sources, destinations, transforms) plus pipelines, topics, kafka_user, client_credential.
- `internal/datasource/` — read-only listings (transforms, tags, topics, topic, topic_metrics, roles).
- `internal/generated/` — schemas, model structs, field mappings produced by tfgen. **Do not hand-edit.**
- `internal/helper/` — type conversion, deprecation utilities, timeouts.
- `internal/resource/shared/marshaling.go` — reflection bridge between models and the API.
- `cmd/tfgen/` — code generator (parser + generator + `overrides.json`).

### Connector resources (BaseConnectorResource)
Each connector has a `connector_code` and a flat `Config map[string]any`. CRUD flow:

1. **Create**: plan → `ModelToAPIConfig` → POST → response → `APIConfigToModel` → state.
2. **Read**: GET → `APIConfigToModel` → state. 404 → `resp.State.RemoveResource`.
3. **Update**: plan → `ModelToAPIConfig` → PUT → response → `APIConfigToModel` → state.
4. **Delete**: DELETE → `RemoveResource`.

`ModelToAPIConfig` walks the model by reflection, reads `tfsdk:"..."` tags, and maps to API fields via the per-connector `fieldMappings` map. `BuildTfsdkFieldIndex` recurses into embedded structs — that is what makes the deprecated-alias wrapper pattern work.

Non-connector resources (pipeline, topic, tag, kafka_user, client_credential) implement CRUD directly without the reflection layer.

### tfgen (code generator)
Reads backend `configuration.latest.json`, emits Go schemas + models + field mappings.

Backend control → Terraform type:

| Control | TF type | Schema attribute |
|---|---|---|
| `string`, `textarea`, `json`, `datetime`, `one-select` | String | `schema.StringAttribute` |
| `password` | String (Sensitive) | `schema.StringAttribute` |
| `number`, `slider` | Int64 | `schema.Int64Attribute` |
| `boolean`, `toggle` | Bool | `schema.BoolAttribute` |
| `multi-select` | List[String] | `schema.ListAttribute` |

Special cases:
- Fields named `port` or ending `_port` are forced to Int64 even if the backend says `control: "string"`.
- Required+default → `Optional: true, Computed: true` (a Required field cannot have a default in TF).
- `user_defined: false` → field skipped entirely.
- `control: "password"` OR `encrypt: true` → `Sensitive: true`.
- Go field naming preserves: `ID SSH SSL SQL DB URL API AWS ARN QA` uppercase. So `ssh_port` → `SSHPort`, `role_arn` → `RoleARN`.

`cmd/tfgen/overrides.json` handles fields the parser can't synthesize:
- `map_string` — `map[string]types.String` (e.g. snowflake `auto_qa_dedupe_table_mapping`).
- `map_nested` — map of nested objects (e.g. clickhouse `topics_config_map`, sqlserveraws `snapshot_custom_table_config`).
When an override's `api_field_name` matches a backend field, the override wins and the auto-parsed version is dropped.

### Transforms
The API client exposes `CreateTransform / GetTransform / UpdateTransform / DeleteTransform / GetTransformImplementationDetails / UpdateTransformImplementationDetails`. All transform resources accept `implementation_json`:

```hcl
implementation_json = jsonencode({
  language        = "JavaScript"
  value_transform = "return record;"
})
```

If unset, implementation is managed outside Terraform and preserved on update.

## API quirks (non-obvious)

- Sources Read uses `?secret_returned=true` to get sensitive fields back.
- POST/PUT/DELETE on `/sources`, `/destinations`, `/pipelines` must include `&wait=false` — VCR/mock URLs need it too.
- List endpoints default `page_size=10` (max 100). `ListSources/ListDestinations/ListPipelines` paginate until `resp.Total`; anything else silently truncates tenants with >10 resources (affects sweepers and adopt-on-exists).
- `/sources`, `/destinations`, `/pipelines` accept `partial_name` only — there is no exact-name filter. Adopt-by-name uses `partial_name=<name>&page_size=100` and matches client-side.
- Create returns 422 "already exists" when a non-deleted record with the same `{tenant_id, name}` exists. List additionally filters by `service_id`, so a source orphaned in a different service of the same tenant triggers 422 but is invisible to list — adopt fails with "reported as existing but not found in list". Known backend asymmetry.
- Use `stringplanmodifier.UseStateForUnknown()` for computed fields that don't change to avoid spurious diffs.
- Kafka Users (`/kafka-access/kafka-users`): username is the resource ID; password is write-only; no individual GET — Read filters from list.
- Client Credentials (`/auth/client-credentials`): no Update endpoint, all fields are ForceNew; secret is only returned at creation.

## Deprecated attribute pattern (v2 → v3 aliases)

When the backend keeps a config field but the Terraform attribute name changes, add a deprecated alias so existing v2 configs keep working:

1. In `internal/resource/source/<connector>_generated.go` (or destination), define a wrapper struct embedding the generated model:
   ```go
   type source<Name>ModelWithDeprecated struct {
       generated.Source<Name>Model
       InsertStaticKeyField2Old types.String `tfsdk:"insert_static_key_field_2"`
   }
   ```
2. Override `NewModelInstance()` to return the wrapper (so reflection sees the extra fields).
3. Override `GetSchema()` to register the old name as `Optional: true, Computed: true, DeprecationMessage: "Use 'new_name' instead."` plus `stringvalidator.ConflictsWith(path.MatchRoot("new_name"))`.
4. Add to the field-mapping map: `mappings["insert_static_key_field_2"] = "transforms.InsertStaticKey2.static.field"` — same API target as the new name.
5. Add to `internal/provider/v2_backward_compat_test.go` and `docs/MIGRATION.md`.

Int64 aliases follow the same shape with `int64validator.ConflictsWith`.

Not aliasable (document in MIGRATION.md + exceptions map):
- Required fields (alias must be `Optional`, so required-only renames force a breaking change).
- API-field renames where old and new map to different backend fields.
- Type changes (e.g. map-of-objects → JSON string).

## Testing

| Tier | Pattern | API | Duration |
|---|---|---|---|
| Unit | `Test[^Acc]` (`-short`) | No | ~5s |
| Schema compat | `TestSchemaBackwardsCompatibility` | No | ~2s |
| Validators | `Test.*Validator` | No | ~2s |
| Integration (VCR) | `TestIntegration_` | No | ~30s |
| Acceptance | `TestAcc` | Yes | ~15m |
| Migration | `TestAcc.*Migration` | Yes | ~30m |

Schema-compat detects: required attribute removed (breaking), optional→required (breaking), computed removed (warning). After an intentional schema change run `make snapshots`.

If `TestAcc.*Migration` produces a non-empty plan, the new provider diverges from v2.1.18 — inspect the plan to see which attribute differs; that signals a potential breaking change.

VCR cassettes live next to their test files. Re-record with `make cassettes` (needs API credentials).

Required env vars for acceptance: `TF_ACC=1`, `STREAMKAP_CLIENT_ID`, `STREAMKAP_SECRET`. Optional: `STREAMKAP_HOST`, `UPDATE_CASSETTES`, `UPDATE_SNAPSHOTS`, `TF_LOG`.

## Conventions

- AI-agent-friendly schema descriptions: every resource/data source needs both `Description` and `MarkdownDescription`. tfgen emits these automatically; if you add an attribute by hand, document enums (list valid values), defaults, and `**Security:**` notes for sensitive fields.
- Each resource needs `examples/resources/streamkap_<name>/{basic,complete}.tf`.
- Provider address: `github.com/streamkap-com/streamkap`. Go 1.24+, Terraform Plugin Framework.
- Connector status values (read-only): `Active`, `Paused`, `Stopped`, `Broken`, `Starting`, `Unassigned`, `Unknown`.

For deeper detail follow the Documentation map at the top of this file.
