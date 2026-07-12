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
| `docs/audits/<date>/`, `docs/plans/<date>-*.md` | Point-in-time audit reports and execution plans. Append new files; do not rewrite history. **Both directories are gitignored** (`.gitignore`), so everything you write there is local-only and invisible to anyone who clones the repo. Treat them as a working scratchpad, not shared history, and never assume a reader can see them. Whether that gitignore is intentional is an open question — don't "fix" it by committing audits or by editing `.gitignore` on your own. |
| `CHANGELOG.md` | User-visible changes per release. Update before tagging. |

`env.md` is **not** a repo doc: it is untracked, gitignored, and holds live credentials. Never commit it, never quote it into a tracked file, and never link to it.

### Keep docs in sync

Changing something in the map above means updating its doc in the same PR. Specifically: a new/removed resource → `AGENTS.md` tables + `docs/ARCHITECTURE.md` counts + `make generate`; a deprecated alias → a `TestAcc<Connector>_MigrationFromLegacy` row in `internal/provider/migration_test.go` + `docs/MIGRATION.md`; a tfgen type-mapping change → `docs/CODE_GENERATOR.md`; a CRUD/retry/pagination change → `docs/ARCHITECTURE.md` + the API-quirks list below; any user-visible change → `CHANGELOG.md`. If a doc is wrong, fix it — don't write a parallel one. New doc files belong only in `docs/audits/` and `docs/plans/`.

A new resource also needs `examples/resources/streamkap_<name>/{basic,complete}.tf` — `templates/resources.md.tmpl` embeds both into the registry page, so `tfplugindocs` (and therefore `make generate`) fails if either is missing.

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
| `make test-all` | Unit + schema-compat + validators — no API; excludes `testacc`/`test-migration` |
| `make testacc` | Acceptance tests, `TF_ACC=1`, ~15m, hits real API |
| `make test-migration` | v2→v3 migration acceptance tests |
| `make snapshots` | Update schema-compat snapshots after intentional schema changes |
| `make sweep` | Clean orphaned test resources |

Schema regeneration: `STREAMKAP_BACKEND_PATH=/path/to/python-be-streamkap make generate` (or run `cmd/tfgen` directly per-connector with `--entity-type sources --connector postgresql`).

**Regenerate ONLY with `STREAMKAP_BACKEND_PATH=<path> make generate` — never `go generate ./...`.** `go generate ./...` runs `tfplugindocs` (root `main.go`) *before* `tfgen` (`internal/generated/doc.go`), so docs render against the previous schema: a new field lands in `internal/generated/*.go` but ships missing from `docs/resources/*.md` (this shipped in beta.18). `make generate` runs `tfgen` first, then `tfplugindocs`. Abort codegen if `STREAMKAP_BACKEND_PATH` is unset or `ls "$STREAMKAP_BACKEND_PATH"` fails — `go generate` with it unset silently emits wrong output. After regenerating, verify each newly added attribute appears in **both** the `.go` schema and its `docs/resources/*.md` page, and report which backend branch+commit the run used.

#### Post-regen checklist

`make generate` rewrites every connector, so a backend change you didn't ask for rides along. Work this list before committing:

1. `make snapshots`, then **read the diff**. It is the regen's changelog. `added=` are new backend fields; `removed=` means the backend dropped a field — check whether a hand-maintained alias in `internal/resource/{source,destination}/*_generated.go` still points at it (`TestDeprecatedAliasTargetsExist` catches this).
2. **`changed=` on a `Sensitive` flag is a security regression until proven otherwise.** The backend repeatedly ships `api.key` and `http.headers.authorization` with no `encrypt`/`control`. `isSecretField` forces those; if a *different* credential shows up unmarked, add it there — never to `internal/generated/`.
3. `git status` the provider tree for stray connector files (a wrong-branch run adds/removes plugins).
4. Confirm the backend repo is back on its original branch. Regenerating leaves it on `main` otherwise.
5. Every registered resource needs a snapshot or it silently skips drift checks — `TestEveryResourceHasSchemaSnapshot` enforces this.

Never hand-edit `internal/generated/` to restore a `Sensitive` flag. That fix is invisible in review and dies on the next regen; it already happened twice with the webhook `api_key`.

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
- Fields named/suffixed `api_key` or `authorization` → forced `Sensitive: true` (`isSecretField`), because the webhook plugins ship `api.key` and the http-sink ships `http.headers.authorization` with neither flag, and the backend keeps regressing it. Never re-fix this by editing `internal/generated/`.
- Every connector merges the entity-wide `configurations_for_all.json` common fields **except `kafkadirect`**, which the backend (`_load_global_configuration`) resolves from its plugin config alone. tfgen mirrors this skip in `Generate()`; the Kafka Direct source/destination expose only their plugin fields.
- Go field naming preserves: `ID SSH SSL SQL DB URL API AWS ARN QA` uppercase. So `ssh_port` → `SSHPort`, `role_arn` → `RoleARN`.

`cmd/tfgen/overrides.json` handles fields the parser can't synthesize:
- `map_string` — `map[string]types.String` (e.g. snowflake `auto_qa_dedupe_table_mapping`).
- `map_nested` — map of nested objects (e.g. clickhouse `topics_config_map`).

An override's `api_field_name` must resolve to a field the backend actually declares — tfgen fails the build otherwise. Nothing else validates overrides, so one can outlive its backend field and keep generating an attribute Terraform accepts and the backend silently drops; `sqlserveraws.snapshot_custom_table_config` did exactly that for several releases.
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
- Create returns 422 "already exists" when a non-deleted record with the same `{tenant_id, name}` exists. **No connector resource auto-adopts on this.** Sources and destinations used to; they no longer do (pipelines, transforms and tags never did). From inside the client, "a previous apply created the record but lost the response" and "a `create_before_destroy` replace whose deposed instance still holds the name" are the same 422 — and adopting is right for the first but destroys live data in the second (the new state entry inherits the deposed entry's backend id, so Terraform's next step deletes it). Create now fails with recovery guidance instead: `terraform import` for the lost-response case, `terraform state rm` of the deposed entries otherwise. Tags still adopt deliberately (leaked CI tags, different trade-off).
- Use `stringplanmodifier.UseStateForUnknown()` for computed fields that don't change to avoid spurious diffs.
- Kafka Users (`/kafka-access/kafka-users`): username is the resource ID; password is write-only; no individual GET — Read filters from list.
- Client Credentials (`/auth/client-credentials`): no Update endpoint, all fields are ForceNew; secret is only returned at creation.
- `"produced an unexpected new value: was cty.StringVal(\"\"), but now null"` is an `Optional+Computed` echo mismatch — the API echo differs from the default. It is not just `insert_static_*`/static-transform fields: it also hits deprecated aliases and placeholder `<...>` defaults (see `HasPlaceholderDefault`). When you touch one, audit **all** siblings of that class across **every** connector — past fixes only patched a subset.
- Secrets aren't faithfully echoed even with `secret_returned=true`: the backend returns `null` for an `encrypt: true` field whose stored value is absent or decrypts to `"null"` (`app/utils/entity_searches.py`), e.g. Snowflake `snowflake_private_key_passphrase` on a non-passphrase-secured key. The sensitive variant of the echo mismatch is `inconsistent values for sensitive attribute`. Create/Update restore the planned value for every `Sensitive` string attribute after `configMapToModel` (`shared.CaptureStringFields`/`PreserveKnownStringFields`) — the configured credential is authoritative. Read is different: it restores from prior *state*, and only where the API echoed null (`shared.FillNullStringFields`). Restoring unconditionally on Read would blind refresh to a credential genuinely rotated outside Terraform; restoring nothing leaves the null echo in state and produces a diff on every plan. Both connector and transform base resources do this.

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
| Acceptance | `TestAcc` | Yes | ~15m |
| Migration | `TestAcc.*Migration` | Yes | ~30m |

Schema-compat detects: required attribute removed (breaking), optional→required (breaking), computed removed (warning). It also fails on *any* drift between a snapshot and the current schema — an added attribute, a removed one, or a flipped `Required`/`Optional`/`Computed`/`Sensitive` flag. Snapshots are the schema of record read by humans and tooling, so they must never lag. After an intentional schema change run `make snapshots` and review the diff.

If `TestAcc.*Migration` produces a non-empty plan, the new provider diverges from v2.1.18 — inspect the plan to see which attribute differs; that signals a potential breaking change.

**Offline API coverage is httpmock, not VCR.** A VCR/cassette tier was scaffolded and never implemented; it was removed rather than left to imply coverage it did not have. Offline tests that need a fake backend belong in `internal/api/client_test.go` or `internal/provider/state_conflict_test.go`, which use `httpmock`. If a cassette tier is ever revived, redact bodies as well as header keys — the old hook matched key names only, so hostnames and tenant IDs would have landed in committed cassettes.

Required env vars for acceptance: `TF_ACC=1`, `STREAMKAP_CLIENT_ID`, `STREAMKAP_SECRET`. Optional: `STREAMKAP_HOST` (defaults to `https://api.streamkap.com`), `UPDATE_SNAPSHOTS`, `TF_LOG`.

## Conventions

- AI-agent-friendly schema descriptions: every resource/data source needs both `Description` and `MarkdownDescription`. tfgen emits these automatically; if you add an attribute by hand, document enums (list valid values), defaults, and `**Security:**` notes for sensitive fields.
- Each resource needs `examples/resources/streamkap_<name>/{basic,complete}.tf`.
- Provider address: `github.com/streamkap-com/streamkap` (differs from the module path).
- Connector status values (read-only): `Active`, `Paused`, `Stopped`, `Broken`, `Starting`, `Unassigned`, `Unknown`.
- **Never log a connector `Config`/`configMap`.** It is keyed by API field name and holds decrypted credentials. Request bodies are logged through `redactSensitiveJSON` (`internal/api/redact.go`); anything logged outside `internal/api` bypasses it. Secrets travel under dotted Kafka-Connect names (`api.key`, `snowflake.private.key`), so any new redaction pattern must treat `.` as a separator.

For deeper detail follow the Documentation map at the top of this file.
