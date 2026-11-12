# AGENTS.md — AI Coding Agent Guide

Audience: AI agents (or humans) **using** the Streamkap Terraform provider in customer configs. Changing the provider itself: see [Provider development](#provider-development).

Terraform source address: `streamkap-com/streamkap` (`registry.terraform.io/streamkap-com/streamkap`).

## Versions and branches

- **v2.x (legacy)**: bug and security fixes only through 15 October 2026. Pin `version = "~> 2.2"` until ready to migrate. Branch `v2`.
- **v3.x (stable)**: new resources (`streamkap_destination_weaviate`, `streamkap_kafka_user`, `streamkap_client_credential`, `streamkap_roles` data source) plus schema and API changes. Deprecated v2 aliases remain supported through v3.x. Branch `main`. Pin `version = "3.1.0"` and review the migration guide before upgrading from v2.

## What it manages

- **Sources**: CDC, event and API connectors (databases, queues, webhooks, S3, SaaS APIs).
- **Destinations**: warehouses, lakes, queues, vector DBs.
- **Pipelines**: connect a CDC source to a destination, optionally through transforms. An API source has none: `streamkap_topic_destination` sends its topics, and `streamkap_pipeline` refuses an API source and a managed pipeline.
- **Transforms**: in-flight data transformations (JS map/filter, enrich, SQL join, rollup, fan-out).
- **Topics**: Kafka topic configuration.
- **Kafka users** and **Client credentials** (v3+): access control and machine auth.

## Authentication

OAuth2 client credentials. Prefer env vars over inline literals:

```bash
export STREAMKAP_CLIENT_ID=...
export STREAMKAP_SECRET=...
# optional: export STREAMKAP_HOST=https://api.streamkap.com
```

```hcl
provider "streamkap" {} # picks up env vars
```

## Minimal example

```hcl
terraform {
  required_providers {
    streamkap = { source = "streamkap-com/streamkap", version = "3.1.0" }
  }
}

variable "db_password" {
  type      = string
  sensitive = true
}

variable "snowflake_key" {
  type        = string
  sensitive   = true
  description = "Snowflake private key without PEM headers, footers, or line breaks."
}

resource "streamkap_source_postgresql" "main" {
  name                = "my-postgres"
  database_hostname   = "db.example.com"
  database_user       = "streamkap"
  database_password   = var.db_password
  database_dbname     = "mydb"
  schema_include_list = "public"
  table_include_list  = "public.customers"
}

resource "streamkap_destination_snowflake" "main" {
  name                    = "my-snowflake"
  snowflake_url_name      = "org-account.snowflakecomputing.com"
  snowflake_user_name     = "STREAMKAP_USER"
  snowflake_private_key   = var.snowflake_key
  snowflake_database_name = "STREAMKAP"
  snowflake_schema_name   = "PUBLIC"
}

resource "streamkap_pipeline" "main" {
  name = "postgres-to-snowflake"

  source = {
    id        = streamkap_source_postgresql.main.id
    name      = streamkap_source_postgresql.main.name
    connector = streamkap_source_postgresql.main.connector
    topics    = ["public.customers"]
  }

  destination = {
    id        = streamkap_destination_snowflake.main.id
    name      = streamkap_destination_snowflake.main.name
    connector = streamkap_destination_snowflake.main.connector
  }
}
```

`source` and `destination` are nested objects, not bare IDs: each carries `id`, `name` and `connector` (the source also takes `topics`). Wire them from the resources' own attributes as above.

Full per-resource examples: `examples/resources/streamkap_<name>/{basic,complete}.tf`.

## Resources

A snapshot; the canonical list is `terraform providers schema -json` or the registry.

### Sources
`streamkap_source_postgresql`, `mysql`, `mongodb`, `mongodbhosted`, `dynamodb`, `sqlserver`, `oracle`, `oracleaws`, `db2`, `informix` (v3), `mariadb`, `alloydb`, `documentdb`, `elasticsearch`, `planetscale`, `redis`, `s3`, `supabase`, `vitess`, `webhook`, `salesforce_webhook` (v3), `zendesk_webhook` (v3), `shopify_webhook` (v3), `stripe_webhook` (v3), `kafkadirect`.

API sources: `streamkap_source_hubspot`, `salesforce`, `netsuite`, `stripe`, `zendesk`, `google_analytics`, `facebook_ads`, `google_ads`. They start by themselves; route their topics with `streamkap_topic_destination`, not `streamkap_pipeline`. An OAuth mode needs an `oauth_grant_id` from Connect, which Terraform cannot run: each resource page gives the CLI handoff.

### Destinations
`streamkap_destination_snowflake`, `clickhouse`, `databricks`, `postgresql`, `mysql`, `sqlserver`, `oracle`, `db2`, `cockroachdb`, `bigquery`, `redshift`, `motherduck`, `starburst`, `s3`, `gcs`, `r2`, `azblob`, `iceberg`, `kafka`, `kafkadirect`, `httpsink`, `redis`, `weaviate` (v3), `pinecone` (v3).

### Transforms
`streamkap_transform_map_filter`, `enrich`, `enrich_async`, `sql_join`, `rollup`, `fan_out`, `topic_router` (v3).

### Other resources
`streamkap_pipeline`, `streamkap_topic_destination`, `streamkap_topic`, `streamkap_tag`, `streamkap_kafka_user` (v3), `streamkap_client_credential` (v3).

### Data sources
`streamkap_transform`, `streamkap_tag`, `streamkap_tags` (v3), `streamkap_topics`, `streamkap_topic`, `streamkap_topic_metrics`, `streamkap_roles` (v3).

## Patterns

**Tags**: every source, destination, transform, topic and pipeline accepts an optional `tags = [streamkap_tag.<name>.id, ...]`. When set, Terraform manages them and reverts drift on apply. When unset, the provider keeps whatever tags the backend has. `tags = []` clears them (null and empty list differ on the wire).

```hcl
resource "streamkap_tag" "prod" {
  name = "production"
  type = ["sources", "destinations", "pipelines"]
}

resource "streamkap_source_postgresql" "orders" {
  # ...
  tags = [streamkap_tag.prod.id]
}
```

Look up existing tag IDs with the `streamkap_tags` data source (`filter_name`, `filter_type` or `filter_ids`) instead of hardcoding them, and check the results match the intended tags before using their IDs.

**Sensitive values**: declare `variable { sensitive = true }`; never inline secrets.

**Table selection**: most CDC sources take comma-separated patterns:
```hcl
table_include_list = "public.customers,public.orders,inventory.*"
# or
table_exclude_list = "public.temp_*,public.logs"
```

**Insert modes**: destinations typically support `insert` and `upsert` via `insert_mode`.

**Transform chaining**: `transforms` on a pipeline is an ordered list of objects, each with the transform's `id` and the `topics` it processes (both required):

```hcl
transforms = [
  { id = streamkap_transform_map_filter.redact.id, topics = ["public.customers"] },
]
```

**Dynamic transform output topics**: when a transform (topic router, fan-out) generates topic names that can't be listed in `transforms[].topics`, set `topic_auto_discovery_transforms = [{ transform_id = ..., regex = "..." }]` on the pipeline. The backend matches the regex against the transform's output topics and adds them.

**Transform implementation code**: most transform resources accept `implementation_json`:
```hcl
implementation_json = jsonencode({
  language        = "JAVASCRIPT"
  value_transform = "function _streamkap_transform(inputObj) { return inputObj; }"
})
```
Omit it to manage code outside Terraform; Terraform then preserves the existing implementation.

## Schema discovery

Every attribute has `Description` and `MarkdownDescription`, with enums listed inline ("Valid values: `insert`, `upsert`"), defaults ("Defaults to `5432`") and a `**Security:**` note on sensitive fields. Read them with `terraform providers schema -json` or the Terraform MCP Server.

## Import

All resources support import; the ID comes from the Streamkap UI or API:

```bash
terraform import streamkap_source_postgresql.main <resource-id>
```

## Errors and retry

The provider retries eligible operations on HTTP 429, 502/503/504, network timeouts and transient Kafka errors. Client-credential creation does not retry failed responses: the API has no idempotency key, so a replay can issue an unmanaged credential. Common errors:

- `Unable to Create Streamkap API Client`: check `STREAMKAP_CLIENT_ID` / `STREAMKAP_SECRET`.
- `404 Not Found` on read: the resource was deleted out of band; `terraform refresh` drops it from state.
- `Invalid value for [field]`: check enum values and types in the schema.
- `422 already exists` on create: another resource in the tenant has the same name, or an orphaned record exists in another service of the same tenant (see [API quirks](#api-quirks-non-obvious)).

## References

- User docs: <https://docs.streamkap.com/streamkap-provider-for-terraform>
- Streamkap API: <https://api.streamkap.com> · OpenAPI: <https://api.streamkap.com/openapi.json>
- Registry: <https://registry.terraform.io/providers/streamkap-com/streamkap>
- Migration (v2 → v3): `docs/guides/migration.md`

---

# Provider development

Everything above is for consuming the provider. What follows is for changing it.

## Docs

- `docs/ARCHITECTURE.md`: layers, CRUD flow, directory structure, test setup, CI workflows.
- `docs/CODE_GENERATOR.md`: tfgen internals, type mapping, `overrides.json`, adding a new connector.
- `docs/API_SOURCES.md`: API-source codegen and traps. Read it before touching an API source.
- `docs/RELEASING.md`: release flow and stable promotion. Read it before any release or tag.
- `docs/index.md`, `docs/resources/`, `docs/data-sources/`: generated by `make generate`; never hand-edit.
- `templates/guides/migration.md`: source of the v2 → v3 guide; `go generate main.go` copies it to `docs/guides/migration.md` for the Registry. `docs/MIGRATION.md` keeps the old GitHub entry point.
- `docs/audits/<date>/`, `docs/plans/<date>-*.md`: gitignored, so local-only scratch that nobody who clones the repo can see. Append new files rather than rewriting. Whether the gitignore is intentional is an open question: leave `.gitignore` alone and don't commit audits.
- `CHANGELOG.md`: user-visible changes per release.

`env.md` is not a repo doc: it is untracked, gitignored and holds live credentials. Never commit it, quote it into a tracked file or link to it.

### Keep docs in sync

Update the matching doc in the same PR:

- New or removed resource: the Resources lists above, counts in `docs/ARCHITECTURE.md`, `make generate`, and `examples/resources/streamkap_<name>/{basic,complete}.tf` (`templates/resources.md.tmpl` embeds both, so `make generate` fails if either is missing).
- Deprecated alias: a `TestAcc<Connector>_MigrationFromLegacy` row in `internal/provider/migration_test.go` and `templates/guides/migration.md`.
- tfgen type-mapping change: `docs/CODE_GENERATOR.md`.
- CRUD, retry or pagination change: `docs/ARCHITECTURE.md` and [API quirks](#api-quirks-non-obvious).
- Any user-visible change: `CHANGELOG.md`.

If a doc is wrong, fix it rather than writing a parallel one. Customer guides published on the Registry go in `docs/guides/`.

## Branches

- **`main`**: v3.x stable, the default target for features and PRs.
- **`v2`**: v2.x legacy maintenance, bug and security fixes only.

Keep the histories independent: never merge `main` and `v2`. A fix for v2 users goes on `v2`; a shared fix is a separate commit on each line. Cut feature branches from the target line, and pull and confirm the branch before starting.

**Pushing a tag triggers the public release workflow.** Surface the exact `git tag`/`git push` commands and stop for explicit approval. Never push tags or branches unprompted. For any release, beta, stable promotion or failed release, read `docs/RELEASING.md` first.

## Public repository: content hygiene

This repo is public. Do not commit internal ticket IDs/URLs, customer or tenant identifiers, real credentials, internal hostnames, verbatim production traces or unannounced roadmap details. Public GitHub issue numbers (`#75`) are fine. Reference `https://api.streamkap.com` and `https://docs.streamkap.com`. Flag any new occurrences you find.

## Reported issues

When handed a reported issue (GitHub, Slack, a customer error), confirm it is real and not already fixed by pending work, and report that verdict, before changing code.

## Related backend

The provider is built against the Streamkap Python FastAPI backend (OpenAPI: `https://api.streamkap.com/openapi.json`). `cmd/tfgen` reads `configuration.latest.json` plugin specs from the clone at `STREAMKAP_BACKEND_PATH`. The path differs per developer: keep it in shell env or `.env`, never hardcode it.

Backend areas worth knowing:
- `app/api/{sources,destinations,kafka_access,auth}_api.py`: endpoint definitions
- `app/models/api/{sources,destinations,kafka_access,app_auth}/`: Pydantic request/response models
- `app/{sources,destinations}/plugins/<connector>/`: `configuration.latest.json` (tfgen's schema source) and `dynamic_utils.py`
- `app/utils/entity_changes.py`: CRUD logic and `created_from` handling

## Codegen

Regenerate only with `STREAMKAP_BACKEND_PATH=<path> make generate`, never `go generate ./...`. `go generate ./...` runs `tfplugindocs` (root `main.go`) before `tfgen` (`internal/generated/doc.go`), so a new field lands in `internal/generated/*.go` but ships missing from `docs/resources/*.md`. To regenerate one connector, run `cmd/tfgen` directly with `--entity-type sources --connector postgresql`. API sources: follow `docs/API_SOURCES.md`.

Before codegen:
- Abort if `STREAMKAP_BACKEND_PATH` is unset or `ls "$STREAMKAP_BACKEND_PATH"` fails: tfgen then silently emits wrong output.
- Run `git -C "$STREAMKAP_BACKEND_PATH" rev-parse --abbrev-ref HEAD` and confirm `main` (the release baseline) or a branch you were explicitly told to use. Never regenerate against whatever happens to be checked out.

After regenerating, verify each new attribute appears in both the `.go` schema and its `docs/resources/*.md` page, and report the backend branch and commit the run used.

### Post-regen checklist

`make generate` rewrites every connector, so backend changes you didn't ask for ride along. Before committing:

1. `make snapshots`, then read the diff: it is the regen's changelog. `added=` are new backend fields. `removed=` means the backend dropped a field: check by hand whether a deprecated alias in `internal/resource/{source,destination}/*_generated.go` still points at it. `TestDeprecatedAliasTargetsExist` does not catch this; it only checks that the replacement named in a `DeprecationMessage` exists in the Terraform schema and that `ConflictsWith` paths resolve.
2. Treat `changed=` on a `Sensitive` flag as a security regression until proven otherwise. Some backend credential fields omit `encrypt`/`control`. `isSecretField` (`cmd/tfgen/generator.go`) handles known credential names; add any other unmarked credential there, never to `internal/generated/`. It is not consulted for fields from `overrides.json`, which need their own `Sensitive` handling. Fix in the generator or override, then regenerate.
3. `git status` the provider tree for stray connector files (a wrong-branch run adds or removes plugins).
4. Put the backend repo back on its original branch.
5. Every registered resource needs a snapshot, or it silently skips drift checks (`TestEveryResourceHasSchemaSnapshot` enforces this).

### Fix the generator, not the generated output

Files under `internal/generated/` (`// Code generated by tfgen. DO NOT EDIT.`) are rewritten on every `make generate`, so hand-edits there are lost. Fix the source instead: tfgen logic (`cmd/tfgen/parser.go`, `generator.go`), `cmd/tfgen/overrides.json`, or the backend `configuration.latest.json`. Then `make generate` and commit source and regenerated files together. Grep every connector for the same defect class and fix all of them in that PR. `docs/CODE_GENERATOR.md` says which source owns which kind of wrongness and holds the type-mapping rules.

`internal/resource/{source,destination}/<connector>_generated.go` files are hand-maintained despite the suffix: they embed `generated.<Name>Model`, wire schema and CRUD, and host the deprecated-alias wrappers. Edit them freely. Deprecated v2 aliases are not in tfgen.

tfgen rules that are easy to miss:
- Required plus default becomes `Optional: true, Computed: true` (a Required attribute cannot have a default).
- An override's `api_field_name` must resolve to a field the backend declares, or the build fails (both `field_overrides` and `additional_fields`). A matching override wins over the auto-parsed field. Review overrides when backend fields change so Terraform does not accept attributes the API ignores.
- Every connector merges `configurations_for_all.json` common fields except `kafkadirect`.

## Local development

- Tests auto-load `.env` via godotenv. For multiline Snowflake PEM keys, `source scripts/load-pem-keys.sh`.
- Local override: a `dev_overrides` block in `~/.terraformrc` mapping `streamkap-com/streamkap` to your `$GOPATH/bin` (where `make install` puts the binary), plus an empty `direct {}`.

## Architecture

Layers and CRUD flow: `docs/ARCHITECTURE.md`. Points worth knowing up front:
- Every API request goes through `doRequest` in `internal/api/` (bearer token, error unwrapping from `detail`, retry in `retry.go`). Creates inject `created_from: TERRAFORM`.
- Connector resources map models to the API by reflection (`internal/resource/shared/marshaling.go`, per-connector `fieldMappings`). `BuildTfsdkFieldIndex` recurses into embedded structs, which is what makes the deprecated-alias wrappers work. Pipeline, topic, tag, kafka_user and client_credential implement CRUD directly.

## API quirks (non-obvious)

- Unknown `admin_tenant_id` or `admin_service_id` must produce diagnostics before authentication (in the Configure context); treating unknown as empty can target the default tenant.
- Empty connector maps (`{}`) clear mappings; null leaves them unset. Keep nil and non-nil empty Go maps distinct when marshaling.
- Transform `deploy = true` reports a terminal deployment failure as an error, but keeps the saved transform ID in state so the deployment can be recovered.
- Redact transform implementation bodies and validation-error input values before logging.
- **Round-trip `periodic_audit` on every pipeline update.** A PUT that omits it deletes the audit: `UpdatePipelineReq.periodic_audit` defaults to `None` and the field is in `_CONDITIONAL_ENTITY_FIELDS`, so the backend `$unset`s it. Terraform does not manage the field, so Update reads the live pipeline and echoes the value back (`preservePeriodicAudit`). This is safe because Update and Read convert topic names and IDs symmetrically (`app/utils/entity_searches.py`). The backend rejects audit topics that are not a subset of the pipeline's topics, so topics dropped from the pipeline are dropped from the audit with a warning instead of failing the apply.
- `streamkap_topics` `messages_7d` and `messages_30d` are always null (the topic details API never returned them) and are deprecated.
- Sources Read uses `?secret_returned=true` to get sensitive fields back.
- POST/PUT/DELETE on `/sources`, `/destinations`, `/pipelines` must include `&wait=false`; mock URLs too.
- List endpoints default to `page_size=10` (max 100). `ListSources/ListDestinations/ListPipelines` paginate until `resp.Total`; anything else silently truncates tenants with more than 10 resources (sweepers, adopt-on-exists).
- `/sources`, `/destinations`, `/pipelines` filter by `partial_name` only, with no exact-name filter. Adopt-by-name uses `partial_name=<name>&page_size=100` and matches client-side.
- Create returns 422 "already exists" when a non-deleted record with the same `{tenant_id, name}` exists. No connector, pipeline or transform resource adopts on this. From the client, "a previous apply created the record but lost the response" and "a `create_before_destroy` replace whose deposed instance still holds the name" are the same 422, and adopting in the second case destroys live data (the new state entry inherits the deposed backend ID, which Terraform then deletes). Create fails with recovery guidance instead: `terraform import` for a lost response, state backup and collision recovery for deposed entries. Tags still adopt deliberately (leaked CI tags).
- Use `stringplanmodifier.UseStateForUnknown()` on computed fields that don't change, to avoid spurious diffs.
- Kafka users (`/kafka-access/kafka-users`): username is the resource ID, password is write-only, and there is no individual GET, so Read filters the list.
- Kafka ACLs are asymmetric: requests may send the topic as `name` or its input alias `topic_name`, but responses always use `name`. `api.KafkaACL` marshals `topic_name` and unmarshals both; decoding only `topic_name` blanked the value and failed every apply. Any Pydantic `Field(alias=...)` on a response model behaves this way: check the endpoint for `by_alias=True` before trusting an alias.
- Kafka usernames must pass both the request model (`min_length=3, max_length=64`) and the service regex `^(?!-)[a-zA-Z0-9-]{1,24}(?<!-)$`: 3-24 chars, alphanumeric and hyphen, no leading or trailing hyphen. Underscores pass the model and fail the service.
- Client credentials (`/auth/client-credentials`): `PATCH` updates `description` and `role_ids` in place; `service_id` has no update path and stays ForceNew. The secret is real only in the create response; the backend stores a masked copy (`abc******xyz`) that list and update echo, so Read and Update restore it from state. `role_ids` is a Set because the backend orders roles by its own catalog.
- `produced an unexpected new value: was cty.StringVal(""), but now null` is an `Optional+Computed` echo mismatch: the API echo differs from the default. It hits static-transform fields (`insert_static_*`), deprecated aliases and placeholder `<...>` defaults (`HasPlaceholderDefault`). When you touch one, audit every sibling of that class across every connector. The backend nulls the stored value of any field whose dynamic `kafka_config` function returns false (`entity_changes.py`), i.e. a conditional field whose gate is unmet (`iceberg_catalog_scope` unless `iceberg_catalog_auth_mode` is `oauth2`; Oracle `lob_enabled` when `log_mining_strategy` is `hybrid`). That is type-agnostic, so a fix scoped to one Terraform type is incomplete. Defaulted `Optional+Computed` string, bool and int64 attributes are refilled when the API echoes null: from the plan on Create/Update, from prior state on Read (`shared.DefaultedAttrNames`, `shared.FillNullFields`). A non-null echo that differs from the plan is left alone so it still fails loudly.
- Secrets are not faithfully echoed even with `secret_returned=true`: an `encrypt: true` field whose stored value is absent or decrypts to `"null"` comes back `null` (`app/utils/entity_searches.py`), e.g. Snowflake `snowflake_private_key_passphrase` on a key without a passphrase. The symptom is `inconsistent values for sensitive attribute`. Create/Update restore the planned value of every `Sensitive` string attribute after `configMapToModel` (`shared.CaptureFields`/`PreserveKnownFields`): the configured credential is authoritative. Read restores from prior state only where the API echoed null (`shared.FillNullFields`), so refresh still sees a credential rotated outside Terraform and a null echo does not produce a diff on every plan. Connector and transform base resources both do this.

## Deprecated attribute pattern (v2 → v3 aliases)

When the backend keeps a config field but the Terraform attribute name changes, add a deprecated alias: a wrapper struct embedding the generated model, an `Optional+Computed` schema entry with `DeprecationMessage`/`ConflictsWith`, and a `fieldMappings` row to the same API target. Ownership: `docs/CODE_GENERATOR.md#deprecated-attribute-aliases`; example: `internal/resource/source/postgresql_generated.go`. Then add a `TestAcc<Connector>_MigrationFromLegacy` case in `internal/provider/migration_test.go`.

Not aliasable (document in the migration guide and the exceptions map):
- Required fields (an alias must be `Optional`, so a required-only rename is a breaking change).
- API-field renames where old and new map to different backend fields.
- Type changes (e.g. map-of-objects → JSON string).

## Testing

`make test-all` runs unit, schema-compat and validator tests offline. `make testacc` and `make test-migration` hit the real API (`TF_ACC=1`, `STREAMKAP_CLIENT_ID`, `STREAMKAP_SECRET`).

Schema-compat tests fail on any drift between a snapshot and the current schema: an added or removed attribute, or a flipped `Required`/`Optional`/`Computed`/`Sensitive` flag, including nested attributes and blocks. Snapshots are the schema of record, so they must never lag: after an intentional schema change, run `make snapshots` and review the diff.

Migration tests start from v2.2.0, accept an initial no-op or in-place update, verify that resource IDs survive, and require a converged plan after applying. Inspect changed defaults even for in-place updates. A skipped case is not upgrade evidence.

Offline API tests use `httpmock` (see `internal/api/client_test.go`, `internal/provider/state_conflict_test.go`). Recorded fixtures must redact request and response bodies as well as headers.

## Conventions

- Every resource and data source needs `Description` and `MarkdownDescription`. tfgen emits them; for a hand-added attribute, document enums (valid values), defaults and a `**Security:**` note on sensitive fields.
- Use `streamkap-com/streamkap` in Terraform configs and local overrides; the Go module path is `github.com/streamkap-com/terraform-provider-streamkap`.
- Connector status values (read-only): `Active`, `Paused`, `Stopped`, `Broken`, `Starting`, `Unassigned`, `Unknown`; API sources also report `Pending`.
- **Never log a connector `Config`/`configMap`.** It is keyed by API field name and holds decrypted credentials. Request bodies are logged through `redactSensitiveJSON` (`internal/api/redact.go`); anything logged outside `internal/api` bypasses it. Secrets travel under dotted Kafka-Connect names (`api.key`, `snowflake.private.key`), so a new redaction pattern must treat `.` as a separator.
