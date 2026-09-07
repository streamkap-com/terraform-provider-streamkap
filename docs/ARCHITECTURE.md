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
│  + 17 more...    │  + 16 more...    │                  │        │
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
  name = "my-transform"
  transforms_language = "JavaScript"

  # Optional: Manage implementation via Terraform
  implementation_json = jsonencode({
    language        = "JavaScript"
    value_transform = "return record;"
    key_transform   = ""
    topic_transform = ""
    common_transform = ""
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

`kafka_user`, `client_credential`, and the `roles` data source were unregistered
in `084d08f` and re-registered once their wire formats were reconciled with the
backend (see the API-quirks list in `AGENTS.md` for the two mismatches that made
them unusable).

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
1. Read TF config into model struct
2. Capture planned Sensitive string values
3. Extract name from model
4. Convert model to API config (ModelToAPIConfig)
5. Call API CreateSource/CreateDestination
6. Convert API response to model, then restore captured secrets
7. Store ID in state

Read:
1. Call API GetSource/GetDestination
2. Convert API response to model (APIConfigToModel)
3. Update state

Update:
1. Read TF config into model
2. Capture planned Sensitive string values
3. Convert to API config
4. Call API UpdateSource/UpdateDestination
5. Convert API response to model, then restore captured secrets
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
    "database_user":     "database.user.user.defined",
    "database_password": "database.password",
    "database_dbname":   "database.dbname",
}
```

### Mapping Rules

| Terraform Attribute | API Field | Notes |
|---------------------|-----------|-------|
| `database_hostname` | `database.hostname.user.defined` | `.user.defined` suffix for user-editable |
| `ssl_mode` | `database.sslmode` | Direct mapping |
| `snapshot_mode` | `snapshot.mode` | Nested config |

## Reflection-Based Marshaling

### ModelToAPIConfig

Converts a typed Terraform model struct to `map[string]any` for API calls:

```go
func ModelToAPIConfig(ctx context.Context, model any, fieldMappings map[string]string) map[string]any {
    config := make(map[string]any)
    v := reflect.ValueOf(model).Elem()
    t := v.Type()

    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        tfTag := field.Tag.Get("tfsdk")
        if tfTag == "" || tfTag == "id" || tfTag == "name" || tfTag == "connector" {
            continue
        }

        apiKey, exists := fieldMappings[tfTag]
        if !exists {
            continue
        }

        fieldValue := v.Field(i)
        // Convert types.String, types.Int64, types.Bool to native Go types
        config[apiKey] = convertTFTypeToNative(fieldValue)
    }
    return config
}
```

### APIConfigToModel

Converts API response `map[string]any` back to typed model struct.

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
│   └── MIGRATION.md              # v2 → v3 migration guide
│
└── .github/workflows/            # CI/CD (see "CI/CD Workflows" below)
    ├── ci.yml                    # Build, vet, lint, credential-free tests
    ├── docs-drift.yml            # docs/ must match committed schemas
    ├── acceptance.yml            # Nightly + push acceptance suite
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

## Authentication Flow

```
1. Provider reads client_id + secret from config or env vars
2. POST /auth/access-token with credentials
3. Receive JWT access token + refresh token
4. All subsequent requests include: Authorization: Bearer <token>
```

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

```go
// API errors are returned in JSON format:
{
    "detail": "Error message here"
}

// Provider surfaces these as Terraform diagnostics:
resp.Diagnostics.AddError(
    "Unable to Create Source",
    "Streamkap API Error: " + err.Error(),
)
```

## State Management

- Empty connector maps clear mappings; null leaves them unset. Marshaling preserves this distinction.
- ID is stored as `id` attribute (computed)
- Sensitive fields use `Sensitive: true` in schema
- Computed fields use `UseStateForUnknown()` plan modifier
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
`httpmock`; acceptance tests use the Terraform testing framework against a real
backend. Schema snapshots track the public attribute contract. Read
`internal/provider/schema_compat_test.go` for what is compared and
`internal/provider/migration_test.go` for the v2 configurations exercised.

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
`make test` clears `TF_ACC` for exactly this reason.

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
TF_ACC=1 go test -v ./internal/provider -run TestAccSourcePostgreSQL_basic
```

### Test Best Practices

1. **Use unique resource names** - Include timestamp or random suffix to avoid conflicts
2. **Clean up resources** - Tests should delete resources they create
3. **Skip when credentials missing** - Use `t.Skip()` for optional tests
4. **Verify import** - All resources should support import
5. **Test updates** - Verify in-place updates work correctly

## CI/CD Workflows

### ci.yml - Continuous Integration

The core gate. Runs on every PR and push to `develop` / `main`, needs no
credentials (so it also covers fork PRs):
1. Build - `go build ./...`
2. Vet - `go vet ./...`
3. Unit + schema-compat + validator tests - `make test-all`
4. Lint - `golangci-lint` (separate job)
5. Workflow validation - `actionlint` (separate job)

It pins `TF_ACC=""` for the whole workflow so a stray value can never turn the
credential-free tiers into a live-API run.

### docs-drift.yml - Docs match committed schemas

On PRs touching `internal/`, `docs/`, `templates/`, `examples/`,
`main.go` or `go.mod`: re-renders the docs from the *committed* schemas with
`tfplugindocs` (no backend needed) and fails if `docs/` changes. This structurally
prevents the beta.18 bug where `go generate ./...` rendered docs one regen behind.

### acceptance.yml / pr-acceptance.yml / migration.yml - Acceptance suites

`acceptance.yml` runs the full `TestAcc` suite on a schedule, pushes to `main` and `develop`,
and manual dispatch, using Terraform 1.0.11, the existing 1.8–1.11 lines, and 1.16.1. Scheduled runs use the
repository default branch. `pr-acceptance.yml` runs a curated subset
(`scripts/acceptance-tests.txt`) on same-repository PRs. `migration.yml` runs
the v2 → v3 `TestAcc.*Migration` suite on same-repository PRs to `main` and
`develop`, and on manual dispatch.

All three share the `streamkap-staging-fixtures` concurrency group and do not
cancel active runs. `queue: max` retains up to 100 waiting jobs, because their fixtures and sweepers share a tenant. PR
acceptance loads credentials from 1Password; nightly acceptance and migration
use environment secrets. Tests may skip when connector credentials are absent;
a green job is not proof that every connector ran.

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
1. Verify stable tags belong to `main` and beta tags to `develop`; require a matching
   changelog heading, unchanged module metadata, a successful build and
   credential-free tests.
2. Require the reusable security scans to pass.
3. Build and sign release archives with GoReleaser, then publish GitHub release assets for the Terraform Registry.

Prerelease tags are marked as prereleases in GitHub. Pushing a tag publishes
public artifacts; it still requires explicit release approval.
