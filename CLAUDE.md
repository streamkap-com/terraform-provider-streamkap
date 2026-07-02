# CLAUDE.md

## Documentation map

| Doc | What it covers |
|---|---|
| `README.md` | User-facing intro, install, basic usage. |
| `AGENTS.md` | AI-agent guide for *consumers* of the provider — resource catalog, common patterns. |
| `docs/index.md` | Auto-generated registry landing page (do not hand-edit; regen via `make generate`). |
| `docs/resources/`, `docs/data-sources/` | Auto-generated per-resource pages (regen via `make generate`). |
| `docs/ARCHITECTURE.md` | Layered diagram + CRUD flow + design rationale. |
| `docs/CODE_GENERATOR.md` | tfgen internals: parser, generator, overrides.json, "adding a new connector" walkthrough. |
| `docs/MIGRATION.md` | v2 → v3 deprecations, removed attributes, action items for users. |
| `docs/audits/<date>/`, `docs/plans/<date>-*.md` | Point-in-time audit reports and execution plans. Append new files; do not rewrite history. |
| `CHANGELOG.md` | User-visible changes per release. Update before tagging. |
| `env.md` | Local env-var setup notes. |

### Keep docs in sync

Changing something in the map above means updating its doc in the same PR. Specifically: a new/removed resource → `AGENTS.md` tables + `docs/ARCHITECTURE.md` counts + `make generate`; a deprecated alias → a `TestAcc<Connector>_MigrationFromLegacy` row in `internal/provider/migration_test.go` + `docs/MIGRATION.md`; a tfgen type-mapping change → `docs/CODE_GENERATOR.md`; a CRUD/retry/pagination change → `docs/ARCHITECTURE.md` + the API-quirks list below; any user-visible change → `CHANGELOG.md`. If a doc is wrong, fix it — don't write a parallel one. New doc files belong only in `docs/audits/` and `docs/plans/`.

## Branches

- **`main`** — v2.x stable (current released line). Bug fixes and back-compatible changes only.
- **`develop`** — v3.x beta (new). Default target for feature branches and PRs.

**Do not merge `develop` into `main`** until v3 is promoted from beta to stable. Until then, treat the two lines as independent: a fix that needs to ship to v2 users goes to `main` directly (and may need a separate cherry-pick to `develop`); v3-only work stays on `develop`. Cut feature branches from the line you're targeting.

Sessions here typically open right after a merge landed elsewhere — `git pull` and confirm which line you're on (`develop` = v3, `main` = v2) before starting. Don't assume the working tree is current.

### Release flow

"Release the next beta" = checkout `develop`, `git pull`, verify clean tree + passing tests, find the latest tag (`git tag --sort=-v:refname | head -1`), bump the beta number by one. **Pushing the tag triggers the public release workflow** — surface the exact `git tag`/`git push` commands and STOP for explicit approval. Never push tags or push `develop` unprompted.

## Public repository — content hygiene

This repo is public. Do not commit internal ticket IDs/URLs, customer or tenant identifiers, real credentials, internal hostnames, verbatim production traces, or unannounced roadmap details. Public GitHub issue numbers (`#75`) are fine. Prefer `https://api.streamkap.com` and `https://docs.streamkap.com` when referencing surfaces. Flag any new occurrences you find.

## Related backend

The provider is built against the Streamkap Python FastAPI backend. Set `STREAMKAP_BACKEND_PATH` to your local clone — `cmd/tfgen` reads `configuration.latest.json` plugin specs from there. OpenAPI: `https://api.streamkap.com/openapi.json`.

Path differs per developer — keep it in shell env or `.env`, never hardcode it here; confirm with `ls` before running codegen. Cross-check schema against the backend's `origin/main` (production runs older configs than feature branches), not whatever branch is checked out. If you switch the backend repo's branch to regenerate, restore the original branch before finishing — never leave it on a changed branch.

**Before any codegen, assert the backend branch.** Run `git -C "$STREAMKAP_BACKEND_PATH" rev-parse --abbrev-ref HEAD` and confirm it is `main` (or a branch you were explicitly told to use) — never regenerate against whatever happens to be checked out; that has shipped schemas with fields silently stripped. Restore the original backend branch when done, then `git status` the *provider* tree to catch stray generated connector files before committing.

When handed a reported issue (GitHub, Slack, a customer error), confirm it's real and not already fixed by pending work, and report that verdict, before changing code.

Backend areas worth knowing:
- `app/api/{sources,destinations,kafka_access,auth}_api.py` — endpoint definitions
- `app/models/api/{sources,destinations,kafka_access,app_auth}/` — Pydantic request/response models
- `app/{sources,destinations}/plugins/<connector>/` — `configuration.latest.json` (schema source for tfgen) and `dynamic_utils.py`
- `app/utils/entity_changes.py` — CRUD logic and `created_from` handling

## Commands

Use `make help` for the full list. Common ones:

| Make target | What it does |
|---|---|
| `make test-all` | Unit + schema-compat + validators + integration (VCR) — no API; excludes `testacc`/`test-migration` |
| `make testacc` | Acceptance tests, `TF_ACC=1`, ~15m, hits real API |
| `make test-migration` | v2→v3 migration acceptance tests |
| `make cassettes` | Re-record VCR cassettes (`UPDATE_CASSETTES=1`) |
| `make snapshots` | Update schema-compat snapshots after intentional schema changes |
| `make sweep` | Clean orphaned test resources |

Schema regeneration: `STREAMKAP_BACKEND_PATH=/path/to/python-be-streamkap make generate` (or run `cmd/tfgen` directly per-connector with `--entity-type sources --connector postgresql`).

**Regenerate ONLY with `STREAMKAP_BACKEND_PATH=<path> make generate` — never `go generate ./...`.** `go generate ./...` runs `tfplugindocs` (root `main.go`) *before* `tfgen` (`internal/generated/doc.go`), so docs render against the previous schema: a new field lands in `internal/generated/*.go` but ships missing from `docs/resources/*.md` (this shipped in beta.18). `make generate` runs `tfgen` first, then `tfplugindocs`. Abort codegen if `STREAMKAP_BACKEND_PATH` is unset or `ls "$STREAMKAP_BACKEND_PATH"` fails — `go generate` with it unset silently emits wrong output. After regenerating, verify each newly added attribute appears in **both** the `.go` schema and its `docs/resources/*.md` page, and report which backend branch+commit the run used.

`.env` is auto-loaded by tests via godotenv. For Snowflake PEM keys (multiline), `source scripts/load-pem-keys.sh`.

Local dev override: add a `dev_overrides` block in `~/.terraformrc` mapping `github.com/streamkap-com/streamkap` → your `$GOPATH/bin` (where `make install` lands the binary), plus an empty `direct {}`.

## Architecture

### Layers
- `internal/provider/` — provider entrypoint, resource/datasource registration, `*_test.go` (acceptance + integration + schema-compat).
- `internal/api/` — `StreamkapAPI` HTTP client. All requests funnel through `doRequest` (bearer token, error unwrapping from `detail`, retry with backoff in `retry.go`). Create operations inject `created_from: TERRAFORM`.
- `internal/resource/` — connector resources (sources, destinations, transforms) plus pipelines, topics, kafka_user, client_credential.
- `internal/datasource/` — read-only listings (transforms, tags, topics, topic, topic_metrics, roles).
- `internal/generated/` — schemas, model structs, field mappings produced by tfgen (generated; see "Fix the generator, not the generated output").
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

Control→TF-type mapping table lives in `docs/CODE_GENERATOR.md` (kept in sync per the doc-sync table above). Non-obvious special cases:
- Fields named `port` or ending `_port` are forced to Int64 even if the backend says `control: "string"`.
- Required+default → `Optional: true, Computed: true` (a Required field cannot have a default in TF).
- `user_defined: false` → field skipped entirely.
- `control: "password"` OR `encrypt: true` → `Sensitive: true`.
- Every connector merges the entity-wide `configurations_for_all.json` common fields **except `kafkadirect`**, which the backend (`_load_global_configuration`) resolves from its plugin config alone. tfgen mirrors this skip in `Generate()`; the Kafka Direct source/destination expose only their plugin fields.
- Go field naming preserves: `ID SSH SSL SQL DB URL API AWS ARN QA` uppercase. So `ssh_port` → `SSHPort`, `role_arn` → `RoleARN`.

`cmd/tfgen/overrides.json` handles fields the parser can't synthesize:
- `map_string` — `map[string]types.String` (e.g. snowflake `auto_qa_dedupe_table_mapping`).
- `map_nested` — map of nested objects (e.g. clickhouse `topics_config_map`, sqlserveraws `snapshot_custom_table_config`).
When an override's `api_field_name` matches a backend field, the override wins and the auto-parsed version is dropped.

### Fix the generator, not the generated output

Files under `internal/generated/` carry `// Code generated by tfgen. DO NOT EDIT.` and are rewritten on every `make generate`. **Hand-edits there are lost on the next regen** (churn is higher on `develop`). When a bug surfaces there, fix it at the source — tfgen logic (`cmd/tfgen/parser.go`, `generator.go`), `cmd/tfgen/overrides.json`, or the backend `configuration.latest.json` — never patch the generated file, then `make generate` and commit source + regenerated files together. Grep every connector for the same defect class and fix all occurrences in that one PR. Full walkthrough (which source owns which kind of wrongness): `docs/CODE_GENERATOR.md`. Deprecated v2 aliases are *not* in tfgen — they live in the hand-maintained wrappers below.

**Hand-maintained exception:** `internal/resource/{source,destination}/<connector>_generated.go` files are *not* generated despite the suffix in the name — they embed `generated.<Name>Model`, register schema/CRUD wiring, and host the deprecated-attribute wrapper struct described below. Edit those freely.

### Transforms
The API client exposes `CreateTransform / GetTransform / UpdateTransform / DeleteTransform / GetTransformImplementationDetails / UpdateTransformImplementationDetails`. All transform resources accept `implementation_json` — a `jsonencode({ language = ..., value_transform = ... })` blob. If unset, implementation is managed outside Terraform and preserved on update.

## API quirks (non-obvious)

- Sources Read uses `?secret_returned=true` to get sensitive fields back.
- POST/PUT/DELETE on `/sources`, `/destinations`, `/pipelines` must include `&wait=false` — VCR/mock URLs need it too.
- List endpoints default `page_size=10` (max 100). `ListSources/ListDestinations/ListPipelines` paginate until `resp.Total`; anything else silently truncates tenants with >10 resources (affects sweepers and adopt-on-exists).
- `/sources`, `/destinations`, `/pipelines` accept `partial_name` only — there is no exact-name filter. Adopt-by-name uses `partial_name=<name>&page_size=100` and matches client-side.
- Create returns 422 "already exists" when a non-deleted record with the same `{tenant_id, name}` exists. List additionally filters by `service_id`, so a source orphaned in a different service of the same tenant triggers 422 but is invisible to list — adopt fails with "reported as existing but not found in list". Known backend asymmetry.
- Use `stringplanmodifier.UseStateForUnknown()` for computed fields that don't change to avoid spurious diffs.
- Kafka Users (`/kafka-access/kafka-users`): username is the resource ID; password is write-only; no individual GET — Read filters from list.
- Client Credentials (`/auth/client-credentials`): no Update endpoint, all fields are ForceNew; secret is only returned at creation.
- `"produced an unexpected new value: was cty.StringVal(\"\"), but now null"` is an `Optional+Computed` echo mismatch — the API echo differs from the default. It is not just `insert_static_*`/static-transform fields: it also hits deprecated aliases and placeholder `<...>` defaults (see `HasPlaceholderDefault`). When you touch one, audit **all** siblings of that class across **every** connector — past fixes only patched a subset.
- Secrets aren't faithfully echoed even with `secret_returned=true`: the backend returns `null` for an `encrypt: true` field whose stored value is absent or decrypts to `"null"` (`app/utils/entity_searches.py`), e.g. Snowflake `snowflake_private_key_passphrase` on a non-passphrase-secured key. The sensitive variant of the echo mismatch is `inconsistent values for sensitive attribute`. `BaseConnectorResource` Create/Update restore the planned value for every `Sensitive` string attribute after `configMapToModel` (`shared.CaptureStringFields`/`PreserveKnownStringFields`) — the configured credential is authoritative. Read keeps the API value.

## Deprecated attribute pattern (v2 → v3 aliases)

When the backend keeps a config field but the Terraform attribute name changes, add a deprecated alias (wrapper struct embedding the generated model + `Optional+Computed` schema entry with `DeprecationMessage`/`ConflictsWith` + a `fieldMappings` row to the same API target). Full step-by-step with code is in `docs/MIGRATION.md`. After adding one, add a `TestAcc<Connector>_MigrationFromLegacy` case in `internal/provider/migration_test.go`.

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
- Provider address: `github.com/streamkap-com/streamkap` (differs from the module path).
- Connector status values (read-only): `Active`, `Paused`, `Stopped`, `Broken`, `Starting`, `Unassigned`, `Unknown`.

For deeper detail follow the Documentation map at the top of this file.
