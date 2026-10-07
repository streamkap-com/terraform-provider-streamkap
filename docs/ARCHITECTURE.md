# Architecture Overview

## High-Level Design

```
┌─────────────────────────────────────────────────────────────────┐
│                     Terraform Provider                           │
├─────────────────────────────────────────────────────────────────┤
│  provider.go                                                     │
│  - Registers all resources and datasources                       │
│  - Handles authentication (OAuth2 token exchange)                │
└─────────────┬───────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Resources                                   │
├──────────────────┬──────────────────┬──────────────────┬────────┤
│  Sources (25)    │  Destinations(24)│  Transforms (7)  │ Other  │
│  PostgreSQL      │  Snowflake       │  MapFilter       │Pipeline│
│  MySQL, MongoDB  │  ClickHouse      │  Enrich          │ Topic  │
│  DynamoDB        │  Databricks      │  EnrichAsync     │  Tag   │
│  SQLServer       │  PostgreSQL, S3  │  SQLJoin         │        │
│  KafkaDirect     │  Iceberg, Kafka  │  Rollup, FanOut  │        │
│  Oracle, Redis   │  BigQuery, GCS   │  TopicRouter     │        │
│  + 17 more...    │  + 15 more...    │                  │        │
└──────────────────┴──────────────────┴──────────────────┴────────┘

61 resources in total (25 sources + 24 destinations + 7 transforms + pipeline,
topic, tag, kafka_user, client_credential) and 7 data sources.
`internal/provider/provider.go` is the register of record — `Resources()` /
`DataSources()`.
              │
              ▼
┌─────────────────────────────────────────────────────────────────┐
│               BaseConnectorResource                              │
│  - Generic CRUD implementation                                   │
│  - Reflection-based model ↔ API conversion                       │
│  - Field mapping (TF attrs → API fields)                         │
└─────────────────────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      API Client                                  │
│  - HTTP client with Bearer token auth                            │
│  - Source, Destination, Transform, Pipeline, Topic, Tag CRUD     │
│  - Kafka User, Client Credential, Role APIs                      │
└─────────────────────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Streamkap API                                  │
│  https://api.streamkap.com                                       │
└─────────────────────────────────────────────────────────────────┘
```

## Code generation

`cmd/tfgen` reads backend plugin configurations, merges common fields, applies
`overrides.json`, and emits schemas, models and field mappings under
`internal/generated/`. The handwritten resource wrappers add CRUD wiring and
v2 attribute aliases.

Use `STREAMKAP_BACKEND_PATH=<backend-main-checkout> make generate` to generate
schemas before registry documentation. See [Code Generator](CODE_GENERATOR.md)
for type mapping, defaults, sensitivity, overrides and the new-connector procedure.

## BaseTransformResource Design

The `BaseTransformResource` provides a generic implementation for all transform resources.

### Transform Implementation Management

All transforms support the `implementation_json` attribute for managing transform code/logic:

```hcl
resource "streamkap_transform_map_filter" "example" {
  name                = "my-transform"
  transforms_language = "JavaScript"

  # Optional: Manage implementation via Terraform
  implementation_json = jsonencode({
    language        = "JAVASCRIPT"
    value_transform = "function _streamkap_transform(inputObj) { return inputObj; }"
  })
}
```

If `implementation_json` is omitted, updates preserve the implementation managed
outside Terraform. With `deploy = true`, terminal failed or stopped deployment
status produces an error while preserving the saved transform in state.

## Non-Connector Resources

Resources that don't follow the connector pattern have their own implementations:
`pipeline`, `topic`, `tag`, `kafka_user`, and `client_credential` implement CRUD
directly against the API client.

### Kafka User (`internal/resource/kafka_user/`)
- CRUD via `/kafka-access/kafka-users` endpoints
- `username` is the resource ID (ForceNew — cannot change after creation), 3-24
  chars alphanumeric+hyphen: the intersection of the request model's
  `min_length=3` and the service layer's 1-24 regex
- `password` is write-only (never returned by the API; Create/Update keep the
  configured value, Read keeps the prior state value)
- `kafka_acls` is a `ListNestedBlock` with ACL rules (topic_name, operation,
  resource_pattern_type, resource). Schema snapshots track these nested fields
  and their flags, including pipeline attributes and timeout blocks.
- Import uses username as the ID
- No individual GET endpoint — reads filter from list

### Client Credential (`internal/resource/client_credential/`)
- Create/List/Update/Delete. `PATCH /auth/client-credentials/{client_id}` accepts
  `description` and `role_ids`, so both change in place; only `service_id` is
  ForceNew, because the update endpoint does not accept it
- `secret` is real only in the create response; list and update echo a masked
  form, so Create captures it and Read/Update preserve it from state
- `roles` is a computed `ListNestedAttribute` resolved from `role_ids`
- `role_ids` is a `SetAttribute`: the backend resolves roles in its own catalog
  order, which need not match the configured order

### Roles Data Source (`internal/datasource/roles.go`)
- Lists available roles from `/auth/roles`; requires the `fe.secure.read.roles`
  permission
- Used to discover role IDs for `streamkap_client_credential` resources

## BaseConnectorResource Design

The `BaseConnectorResource` provides a generic implementation for all connector resources.

### ConnectorConfig Interface

```go
type ConnectorConfig interface {
    // GetSchema returns the Terraform schema for this connector
    GetSchema() schema.Schema

    // GetFieldMappings maps TF attribute names to API field names
    // Example: "database_hostname" -> "database.hostname.user.defined"
    GetFieldMappings() map[string]string

    // GetConnectorType returns "source" or "destination"
    GetConnectorType() ConnectorType

    // GetConnectorCode returns the backend connector code
    // Example: "postgresql", "snowflake"
    GetConnectorCode() string

    // GetResourceName returns the TF resource name suffix
    // Example: "source_postgresql" (becomes "streamkap_source_postgresql")
    GetResourceName() string

    // NewModelInstance creates a new model struct instance
    NewModelInstance() any
}
```

### CRUD Flow

```
Create:
1. Read the Terraform plan into the model
2. Capture planned sensitive strings and defaulted values
3. Extract name from model
4. Convert model to API config (ModelToConfigMap)
5. Call API CreateSource/CreateDestination
6. Convert API response to model, then restore planned secrets and fill null defaults
7. Store ID in state

Read:
1. Capture prior sensitive strings and defaulted values
2. Call API GetSource/GetDestination; remove from state on 404
3. Convert API response to model (ConfigMapToModel)
4. Fill null secrets and defaults from prior state; retain non-null API values
5. Update state

Update:
1. Read the Terraform plan into the model
2. Capture planned sensitive strings and defaulted values
3. Convert to API config
4. Call API UpdateSource/UpdateDestination
5. Convert API response to model, then restore planned secrets and fill null defaults
6. Update state

Delete:
1. Call API DeleteSource/DeleteDestination
2. Remove from state
```

## Field Mappings

Field mappings translate between Terraform attribute names and API field names.

### Example

```go
var SourcePostgresqlFieldMappings = map[string]string{
    "database_hostname": "database.hostname.user.defined",
    "database_port":     "database.port.user.defined",
    "database_user":     "database.user",
    "database_password": "database.password",
    "database_dbname":   "database.dbname",
}
```

### Mapping Rules

| Terraform Attribute | API Field | Notes |
|---------------------|-----------|-------|
| `database_hostname` | `database.hostname.user.defined` | `.user.defined` suffix for user-editable |
| `database_sslmode` | `database.sslmode` | Direct mapping |

## Reflection-Based Marshaling

The shared implementation is in
[`internal/resource/shared/marshaling.go`](../internal/resource/shared/marshaling.go):

- `ModelToConfigMap` converts Terraform values into API configuration and returns
  an error for invalid or conflicting mappings.
- `ConfigMapToModel` applies API values to the Terraform model.
- `BuildTfsdkFieldIndex` follows embedded structs, including deprecated aliases.
- `PreserveKnownFields` restores planned secrets after create/update;
  `FillNullFields` fills missing defaults and preserves null-returned secrets on
  read without hiding non-null credential changes.

The connector and transform base resources supply conversion hooks for their
complex field types. Null and empty maps remain distinct so an empty map can
clear a mapping.

## Directory Structure

```
terraform-provider-streamkap/
├── cmd/
│   └── tfgen/                    # Schema generator CLI
│       ├── main.go               # CLI entry point
│       ├── parser.go             # JSON config parser
│       ├── generator.go          # Go code generator
│       └── *_test.go             # Unit tests
│
├── internal/
│   ├── api/                      # API client
│   │   ├── client.go             # Interface + base client
│   │   ├── auth.go               # OAuth2 authentication
│   │   ├── source.go             # Source CRUD
│   │   ├── destination.go        # Destination CRUD
│   │   ├── pipeline.go           # Pipeline CRUD
│   │   ├── topic.go              # Topic CRUD
│   │   ├── tag.go                # Tag CRUD
│   │   ├── transform.go          # Transform CRUD
│   │   ├── kafka_user.go         # Kafka User CRUD
│   │   ├── client_credential.go  # Client Credential Create/List/Update/Delete
│   │   └── role.go               # Role list
│   │
│   ├── generated/                # Generated code (DO NOT EDIT)
│   │   ├── doc.go                # Package doc
│   │   ├── source_*.go           # Generated source schemas
│   │   ├── destination_*.go      # Generated destination schemas
│   │   └── transform_*.go        # Generated transform schemas
│   │
│   ├── provider/                 # Provider + tests
│   │   ├── provider.go           # Main provider
│   │   ├── provider_test.go      # Test helpers
│   │   └── *_resource_test.go    # Acceptance tests
│   │
│   ├── resource/
│   │   ├── connector/            # Generic base resource
│   │   │   └── base.go           # BaseConnectorResource
│   │   ├── shared/               # Reflection bridge (marshaling.go)
│   │   ├── source/               # Source config wrappers — hand-maintained
│   │   │   └── *_generated.go    # ConnectorConfig impls (NOT generated; see below)
│   │   ├── destination/          # Destination config wrappers — hand-maintained
│   │   │   └── *_generated.go    # ConnectorConfig impls (NOT generated; see below)
│   │   ├── transform/            # Transform config wrappers
│   │   │   ├── base.go           # BaseTransformResource
│   │   │   └── *_generated.go    # TransformConfig implementations
│   │   ├── pipeline/             # Pipeline resource
│   │   ├── topic/                # Topic resource
│   │   ├── tag/                  # Tag resource
│   │   ├── kafka_user/           # Kafka user resource
│   │   └── client_credential/    # Client credential resource
│   │
│   ├── datasource/               # Data sources
│   │   ├── transform.go          # Transform datasource
│   │   ├── tag.go                # Tag datasource (single, by id)
│   │   ├── tags.go               # Tags list/filter datasource
│   │   ├── topics.go             # Topics list datasource
│   │   ├── topic.go              # Topic datasource
│   │   ├── topic_metrics.go      # Topic metrics datasource
│   │   ├── topic_serialization.go # Shared topic (de)serialization helpers
│   │   └── roles.go              # Roles list datasource
│   │
│   └── helper/                   # Utility functions
│       ├── helper.go             # Type conversion helpers
│       └── timeouts.go           # Default CRUD timeouts
│
├── examples/                     # Example TF configs (also the docs source)
│   ├── provider/provider.tf      # Embedded in docs/index.md
│   ├── data-sources/streamkap_*/data-source.tf
│   └── resources/streamkap_*/    # basic.tf, complete.tf, import.sh
│
├── templates/                    # tfplugindocs templates
│   ├── index.md.tmpl             # Provider index page
│   └── resources.md.tmpl         # Per-resource page (embeds basic.tf + complete.tf)
│
├── docs/                         # Generated registry docs + hand-written guides
│   ├── index.md                  # Generated
│   ├── resources/, data-sources/ # Generated
│   ├── ARCHITECTURE.md           # This file
│   ├── CODE_GENERATOR.md         # tfgen internals
│   └── guides/migration.md       # v2 → v3 migration guide
│
└── .github/workflows/            # CI/CD (see "CI/CD Workflows" below)
    ├── ci.yml                    # Build, vet, lint, credential-free tests
    ├── docs-drift.yml            # docs/ must match committed schemas
    ├── acceptance.yml            # Push + manual acceptance suite
    ├── pr-acceptance.yml         # Curated acceptance subset on PRs
    ├── migration.yml             # v2 → v3 migration acceptance tests
    ├── security.yml              # Security scans
    ├── regenerate.yml            # Schema regeneration
    └── release.yml               # Release automation
```

Note the naming trap: `internal/resource/{source,destination,transform}/*_generated.go`
are **hand-maintained** despite the suffix. The generated code lives in
`internal/generated/`; the wrappers embed it and add the deprecated-alias
attributes. Edit the wrappers freely; never edit `internal/generated/`.

Connector `ModifyPlan` aligns deprecated aliases that share an API field.
The configured name determines both planned values, including unknown values,
so a canonical default cannot override an explicitly configured legacy alias.
When neither name is configured, normal default and computed behavior applies.

## API Client

`internal/api/` holds the `StreamkapAPI` HTTP client. Every request funnels
through `doRequest` (bearer token, error unwrapping from `detail`, retry with
backoff in `retry.go`). Create operations inject `created_from: TERRAFORM`.
Request bodies are logged through `redactSensitiveJSON` (`redact.go`); anything
logged outside `internal/api` bypasses it.

## Authentication Flow

```
1. Provider reads client_id + secret from config or env vars
2. POST /auth/access-token with credentials
3. Receive access token; renew with client credentials before expiry or after a 401
4. All subsequent requests include: Authorization: Bearer <token>
```

## Backend contract quirks

Behaviour of the Streamkap API that the provider compensates for. Each entry
names the provider code that owns the workaround; change both together.

### Wire conventions

- Sources Read uses `?secret_returned=true` to get sensitive fields back.
- POST/PUT/DELETE on `/sources`, `/destinations`, `/pipelines` must include
  `&wait=false` — mock URLs need it too.
- List endpoints default `page_size=10` (max 100). `ListSources`,
  `ListDestinations` and `ListPipelines` paginate until `resp.Total`; anything
  else silently truncates tenants with more than 10 resources (affects sweepers
  and adopt-on-exists).
- `/sources`, `/destinations`, `/pipelines` accept `partial_name` only — there
  is no exact-name filter. Adopt-by-name uses `partial_name=<name>&page_size=100`
  and matches client-side.
- Empty connector maps (`{}`) clear mappings; null leaves them unset. Nil and
  non-nil empty Go maps stay distinct in marshaling.
- Unknown `admin_tenant_id` or `admin_service_id` in the provider Configure
  context must produce diagnostics before authentication; treating unknown as
  empty would target the credential's default tenant.
- `streamkap_topics` exposes `messages_7d` and `messages_30d`, which the topic
  details API has never returned — both are always null and are deprecated.

### 422 "already exists" on create — no adoption

Create returns 422 when a non-deleted record with the same `{tenant_id, name}`
exists. **No connector resource auto-adopts on this.** Sources and destinations
used to; pipelines and transforms also refuse. From inside the client, "a
previous apply created the record but lost the response" and "a
`create_before_destroy` replace whose deposed instance still holds the name" are
the same 422 — adopting is right for the first but destroys live data in the
second (the new state entry inherits the deposed entry's backend id, so
Terraform's next step deletes it). Create fails with recovery guidance instead:
`terraform import` for the lost-response case, state backup and collision
recovery for deposed entries otherwise. Tags still adopt deliberately (leaked CI
tags, different trade-off).

### `periodic_audit` round-trip on pipeline update

`UpdatePipelineReq.periodic_audit` defaults to `None`, the backend only copies
it into the entity body when non-null, and the field is in
`_CONDITIONAL_ENTITY_FIELDS` — so a PUT that omits the key `$unset`s it.
Terraform does not manage the field, so Update reads the live pipeline and
echoes the value back (`preservePeriodicAudit`, `internal/resource/pipeline/`);
without that, every apply deleted an audit configured in the UI. The round-trip
is symmetric: Update converts pretty topic names to ids via
`get_topic_ids_from_pretty_name`, and Read converts them back via
`get_pretty_topic_name_from_id` (`app/utils/entity_searches.py`), so what GET
returns is exactly what PUT expects. The backend rejects an update whose audit
topics are not a subset of the pipeline's topics, so topics dropped from the
pipeline are dropped from the audit with a warning rather than failing the
apply. Covered by `TestAccPipeline_PeriodicAuditSurvivesUpdate`.

### `Optional+Computed` echo mismatch

`"produced an unexpected new value: was cty.StringVal(\"\"), but now null"` is
an `Optional+Computed` echo mismatch — the API echo differs from the default.
It is not limited to `insert_static_*`/static-transform fields: it also hits
deprecated aliases and placeholder `<...>` defaults (`HasPlaceholderDefault`,
`cmd/tfgen/parser.go`). When you touch one, audit **all** siblings of that class
across **every** connector — past fixes only patched a subset.

Defaulted `Optional+Computed` string, bool and int64 attributes are refilled
when the API echoes null — from the plan on Create/Update and from prior state on
Read (`shared.DefaultedAttrNames` + `shared.FillNullFields`) — because the
backend nulls the stored value of any field whose dynamic `kafka_config`
function returns false (`app/utils/entity_changes.py`), i.e. a conditional field
whose gate is unmet: `iceberg_catalog_scope` unless `iceberg_catalog_auth_mode`
is `oauth2`, Oracle `lob_enabled` when `log_mining_strategy` is `hybrid`. The
backend behaviour is type-agnostic, so a fix scoped to one Terraform type is
incomplete. A non-null echo that differs from the plan is left alone so it still
fails loudly.

### Secrets are not faithfully echoed

Even with `secret_returned=true`, the backend returns `null` for an
`encrypt: true` field whose stored value is absent or decrypts to `"null"`
(`app/utils/entity_searches.py`), e.g. Snowflake
`snowflake_private_key_passphrase` on a non-passphrase-secured key. The
sensitive variant of the echo mismatch is `inconsistent values for sensitive
attribute`. Create/Update restore the planned value for every `Sensitive` string
attribute after `configMapToModel` (`shared.CaptureFields` /
`PreserveKnownFields`) — the configured credential is authoritative. Read
restores from prior *state*, and only where the API echoed null
(`shared.FillNullFields`): restoring unconditionally would blind refresh to a
credential rotated outside Terraform; restoring nothing leaves the null echo in
state and produces a diff on every plan. Both connector and transform base
resources do this.

### Kafka access

- Kafka ACLs are asymmetric on the wire. The backend model declares the topic
  field as `name` with `topic_name` only as an *input* alias, and serialises
  responses by field name — requests may carry either, but every response comes
  back as `name`. `api.KafkaACL` marshals `topic_name` and unmarshals both
  (`UnmarshalJSON`); decoding `topic_name` alone silently blanked the value and
  every apply failed with "produced an unexpected new value". Any Pydantic
  `Field(alias=...)` on a response model has this shape — check the endpoint for
  `by_alias=True` before trusting the alias.
- Kafka usernames must satisfy *two* backend checks: the request model's
  `min_length=3, max_length=64` and the service layer's
  `^(?!-)[a-zA-Z0-9-]{1,24}(?<!-)$`. Only the intersection works — 3-24 chars,
  alphanumeric and hyphen, no leading or trailing hyphen. Underscores pass the
  model and fail the service.
- Kafka users have no individual GET; Read filters from the list. Client
  credential secrets are real only in the create response (the backend stores a
  masked copy). Details under [Non-Connector Resources](#non-connector-resources).

### Transforms

- `deploy = true` must report terminal deployment failure as an error while
  keeping the saved transform ID in state so the failed deployment can be recovered.
- Transform implementation bodies and validation-error input values are redacted
  before logging.

## Related backend

The provider is built against the Streamkap Python FastAPI backend
(OpenAPI: `https://api.streamkap.com/openapi.json`). `cmd/tfgen` reads
`configuration.latest.json` plugin specs from a local clone at
`STREAMKAP_BACKEND_PATH`. The path differs per developer — keep it in shell env
or `.env`. Cross-check schema against the backend's `origin/main` (the release
baseline), which `scripts/codegen-preflight.sh` enforces (`ALLOW_NONMAIN=1`
overrides deliberately).

Backend areas worth knowing:

- `app/api/{sources,destinations,kafka_access,auth}_api.py` — endpoint definitions
- `app/models/api/{sources,destinations,kafka_access,app_auth}/` — Pydantic request/response models
- `app/{sources,destinations}/plugins/<connector>/` — `configuration.latest.json` (schema source for tfgen) and `dynamic_utils.py`
- `app/utils/entity_changes.py` — CRUD logic and `created_from` handling

## Error Handling

Authentication honors the provider Configure context. Unknown admin tenant or
service scope is rejected before client creation, preventing a fallback to the
credential's default tenant.

The client retries transient failures for eligible operations. Client-credential
creation does not retry failed responses. The endpoint has no idempotency key,
and a failed response can hide successful creation, so replaying the request
could issue an extra credential.

Structured validation errors are reported without their input values, and
transform implementation bodies are omitted from request logs.

API error details are surfaced as Terraform diagnostics with the operation context.

## State Management

- Empty connector maps clear mappings; null leaves them unset. Marshaling preserves this distinction.
- ID is stored as `id` attribute (computed)
- Sensitive fields use `Sensitive: true` in schema
- Stable computed fields use `UseStateForUnknown()`; derived fields can recompute on update.
- Set-once fields use `RequiresReplace()` plan modifier

## Testing Architecture

### Test Types

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Test Pyramid                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│     ▲  Acceptance Tests (internal/provider/*_test.go)               │
│    ╱ ╲   - Create real resources via API                            │
│   ╱   ╲  - Verify state management                                  │
│  ╱     ╲ - Test import functionality                                │
│ ╱───────╲─────────────────────────────────────────────────────────  │
│╱         ╲                                                           │
│  Integration Tests (cmd/tfgen/*_test.go)                            │
│    - Test full generation pipeline                                   │
│    - Verify generated code compiles                                  │
│    - Uses real backend configs when available                        │
│ ────────────────────────────────────────────────────────────────── │
│                                                                      │
│  Unit Tests (cmd/tfgen/*_test.go, internal/*_test.go)               │
│    - Test individual functions                                       │
│    - Mock inputs, verify outputs                                     │
│    - Fast, no external dependencies                                  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

Generator tests cover parser inputs and emitted schemas. Offline API tests use
`httpmock` (`internal/api/client_test.go`,
`internal/provider/state_conflict_test.go`); recorded fixtures must redact
request and response bodies as well as headers. Acceptance tests use the
Terraform testing framework against a real backend.

### Test Tiers

| Tier | Pattern | API | Duration |
|---|---|---|---|
| Unit | `Test[^Acc]` (`-short`) | No | ~5s |
| Schema compat | `TestSchemaBackwardsCompatibility` | No | ~2s |
| Validators | `Test.*Validator` | No | ~2s |
| Acceptance | `TestAcc` | Yes | ~15m |
| Migration | `TestAcc.*Migration` | Yes | ~30m |

**Schema compat** (`internal/provider/schema_compat_test.go`) classifies: required
attribute removed (breaking), optional→required (breaking), computed removed
(warning). It also fails on *any* drift between a snapshot and the current
schema — an added attribute, a removed one, or a flipped
`Required`/`Optional`/`Computed`/`Sensitive` flag, including nested attributes
and block fields. Snapshots are the schema of record read by humans and tooling,
so they must never lag: after an intentional schema change run `make snapshots`
and review the diff. `TestEveryResourceHasSchemaSnapshot` fails for any
registered resource without a snapshot; without one the resource silently skips
drift checks.

**Migration** (`internal/provider/migration_test.go`) starts from v2.2.0,
accepts an initial no-op or in-place update, verifies that resource IDs survive,
and requires a converged plan after applying. Inspect changed defaults even for
in-place updates; skipped cases provide no upgrade evidence. Every deprecated
alias needs a `TestAcc<Connector>_MigrationFromLegacy` case.

### Test Environment Variables

| Variable | Purpose | Required For |
|----------|---------|--------------|
| `TF_ACC=1` | Enable acceptance tests | Acceptance tests |
| `STREAMKAP_CLIENT_ID` | OAuth client ID | Acceptance tests |
| `STREAMKAP_SECRET` | OAuth secret | Acceptance tests |
| `STREAMKAP_HOST` | API endpoint | Optional — defaults to `https://api.streamkap.com` (`provider.go`) |
| `STREAMKAP_BACKEND_PATH` | Backend repo path | Code generation; generator integration tests |
| `UPDATE_SNAPSHOTS=1` | Rewrite schema snapshots | `make snapshots` |

### Running Tests

Tests auto-load `.env` via godotenv, and a `TF_ACC=1` line there turns any
unfiltered `go test` into a live-API acceptance run. Prefer the `make` targets —
`make test` clears `TF_ACC` for exactly this reason. Multiline Snowflake PEM
keys do not fit a `.env` line; `source scripts/load-pem-keys.sh` instead.

```bash
# Unit + schema-compat + validators (fast, no API)
make test-all

# Generator tests
go test -v ./cmd/tfgen/...

# Generator integration tests (requires backend)
STREAMKAP_BACKEND_PATH=/path/to/backend go test -v ./cmd/tfgen/...

# Acceptance tests (creates real resources; needs STREAMKAP_CLIENT_ID/SECRET)
make testacc

# Single acceptance test
TF_ACC=1 go test -v ./internal/provider -run '^TestAccSourcePostgreSQLResource$'
```

### Test Best Practices

1. **Use unique resource names** - Include timestamp or random suffix to avoid conflicts
2. **Clean up resources** - Tests should delete resources they create
3. **Skip when credentials missing** - Use `t.Skip()` for optional tests
4. **Verify import** - All resources should support import
5. **Test updates** - Verify in-place updates work correctly

## CI/CD Workflows

### ci.yml - Continuous Integration

The core gate. Runs on every PR and push to `main`, needs no
credentials (so it also covers fork PRs):
1. Build - `go build ./...`
2. Vet - `go vet ./...`
3. Unit + schema-compat + validator tests - `make test-all`
4. Lint - `golangci-lint` (separate job)
5. Workflow validation - `actionlint` (separate job)

It pins `TF_ACC=""` for the whole workflow so a stray value can never turn the
credential-free tiers into a live-API run.

### docs-drift.yml - Docs match committed schemas

On every PR to `main`: re-renders the docs from the *committed* schemas with
`tfplugindocs` (no backend needed) and fails if `docs/` changes. This structurally
prevents the beta.18 bug where `go generate ./...` rendered docs one regen behind.

### acceptance.yml / pr-acceptance.yml / migration.yml - Acceptance suites

`acceptance.yml` runs the full `TestAcc` suite on pushes to `main` and manual
dispatch. Pushes use Terraform 1.16.1; manual runs also cover Terraform 1.0.11
and the existing 1.8–1.11 lines. `pr-acceptance.yml` runs a curated subset
(`scripts/acceptance-tests.txt`) on same-repository PRs. `migration.yml` runs
the v2 → v3 `TestAcc.*Migration` suite on same-repository PRs to `main`
and on manual dispatch.

Acceptance runs share a concurrency group. Tests may skip when connector
credentials are absent; a green job is not proof that every connector ran.

### security.yml - Security Scanning

Runs on push and PR:
- **govulncheck** - Go call-graph vulnerability analysis
- **Trivy** - Vulnerability scanning
- **Checkov** - Infrastructure-as-code security
- **Gitleaks** - Secret scanning of the checked-out tree

### regenerate.yml - Schema Regeneration

Manually dispatched workflow to regenerate connector schemas against the backend
repository. Locally the equivalent is
`STREAMKAP_BACKEND_PATH=<path> make generate` — never `go generate ./...`.

### release.yml - Release Automation

Triggered on version tags (`v*`):
1. Verify v3 beta and stable tags belong to `main`; require a matching
   changelog heading, unchanged module metadata, a successful build and
   credential-free tests.
2. Require the reusable v3 security scans to pass.
3. Build and sign release archives with GoReleaser, then publish GitHub release assets for the Terraform Registry.

The `v2` branch has a separate release workflow for v2 patch tags. Its preflight
checks branch ancestry and changelog entries, builds, runs vet and offline tests,
and runs `govulncheck` before publishing.

Beta tags retain their prerelease version suffix but publish as full GitHub
releases so the Terraform Registry can ingest them. Pushing a tag publishes
public artifacts; it still requires explicit release approval.
