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

59 resources in total (25 sources + 24 destinations + 7 transforms + pipeline,
topic, tag) and 6 data sources. `internal/provider/provider.go` is the register
of record — `Resources()` / `DataSources()`.
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

## Code Generation Architecture

The `tfgen` tool generates Terraform provider schemas from backend `configuration.latest.json` files.

### Generation Flow

```
┌──────────────────────────────────────────────────────────────────────┐
│  Backend Repository (python-be-streamkap)                            │
│  app/{sources,destinations,transforms}/plugins/*/configuration.latest.json │
└───────────────────────────────┬──────────────────────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────────────┐
│  cmd/tfgen/parser.go                                                 │
│  - Parse JSON into ConfigEntry structs                               │
│  - Extract: name, type, control, default, required, sensitive        │
│  - Filter: user_defined=true fields only                             │
└───────────────────────────────┬──────────────────────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────────────┐
│  cmd/tfgen/generator.go                                              │
│  - Load overrides from overrides.json                                │
│  - Apply automatic type conversions (port fields → Int64)            │
│  - Convert ConfigEntry → FieldData                                   │
│  - Apply Go template to generate code                                │
└───────────────────────────────┬──────────────────────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────────────┐
│  internal/generated/{source,destination,transform}_*.go              │
│  Generated code contains:                                            │
│  - Nested model types (for map_nested overrides)                     │
│  - Main model struct with tfsdk tags                                 │
│  - Schema function returning schema.Schema                           │
│  - Field mappings map[string]string                                  │
└──────────────────────────────────────────────────────────────────────┘
```

### Parser: Backend Config → ConfigEntry

The parser reads `configuration.latest.json` and extracts field metadata:

```go
type ConfigEntry struct {
    Name        string      // API field name: "database.hostname.user.defined"
    Description string      // Field description
    DisplayName string      // Human-readable name
    UserDefined bool        // true = user-editable field
    Value       ValueConfig // Type info, defaults, validation
}

type ValueConfig struct {
    Control string      // UI control: string, password, number, boolean, one-select, etc.
    Type    string      // raw, list
    Default interface{} // Default value
    Values  []string    // Enum values for one-select
    Min     *float64    // Min for slider
    Max     *float64    // Max for slider
}
```

### Type Mapping: Control → Terraform Type

| Backend Control | Terraform Type | Go Type | Schema Attribute |
|-----------------|----------------|---------|------------------|
| `string` | String | `types.String` | `schema.StringAttribute` |
| `password` | String (sensitive) | `types.String` | `schema.StringAttribute` |
| `textarea` | String | `types.String` | `schema.StringAttribute` |
| `json` | String | `types.String` | `schema.StringAttribute` |
| `datetime` | String | `types.String` | `schema.StringAttribute` |
| `number` | Int64 | `types.Int64` | `schema.Int64Attribute` |
| `slider` | Int64 | `types.Int64` | `schema.Int64Attribute` |
| `boolean` | Bool | `types.Bool` | `schema.BoolAttribute` |
| `toggle` | Bool | `types.Bool` | `schema.BoolAttribute` |
| `one-select` | String | `types.String` | `schema.StringAttribute` |
| `multi-select` | List[String] | `types.List` | `schema.ListAttribute` |

### Automatic Type Conversions

#### Port Fields → Int64

Fields named `port` or ending in `_port` are automatically converted from String to Int64:

```
Backend: "ssh.port" with control="string", default="22"
    ↓
Generated: SSHPort types.Int64 with int64default.StaticInt64(22)
```

**Detection:** `tfAttrName == "port" || strings.HasSuffix(tfAttrName, "_port")`

#### Go Abbreviation Handling

Common abbreviations are preserved in uppercase per Go conventions:

| Abbreviation | Example Input | Go Field Name |
|--------------|---------------|---------------|
| `ID` | `connector_id` | `ConnectorID` |
| `SSH` | `ssh_port` | `SSHPort` |
| `SSL` | `ssl_enabled` | `SSLEnabled` |
| `SQL` | `delete_sql_execute` | `DeleteSQLExecute` |
| `DB` | `db_name` | `DBName` |
| `URL` | `api_url` | `APIURL` |
| `API` | `api_key` | `APIKey` |
| `AWS` | `aws_region` | `AWSRegion` |
| `ARN` | `role_arn` | `RoleARN` |
| `QA` | `auto_qa_dedupe` | `AutoQADedupe` |

### Override System

Some fields require special handling that can't be auto-generated. These are defined in `cmd/tfgen/overrides.json`.

#### Override Types

**`map_string`** - Simple string maps:
```go
// Generated model field:
AutoQADedupeTableMapping map[string]types.String `tfsdk:"auto_qa_dedupe_table_mapping"`

// Generated schema:
"auto_qa_dedupe_table_mapping": schema.MapAttribute{
    ElementType: types.StringType,
    Optional:    true,
}
```

**`map_nested`** - Nested object maps:
```go
// Generated nested model:
type clickHouseTopicsConfigMapItemModel struct {
    DeleteSQLExecute types.String `tfsdk:"delete_sql_execute"`
}

// Generated model field:
TopicsConfigMap map[string]clickHouseTopicsConfigMapItemModel `tfsdk:"topics_config_map"`

// Generated schema:
"topics_config_map": schema.MapNestedAttribute{
    Optional: true,
    NestedObject: schema.NestedAttributeObject{
        Attributes: map[string]schema.Attribute{
            "delete_sql_execute": schema.StringAttribute{Optional: true},
        },
    },
}
```

#### Current Overrides

| Connector | Field | Type | Purpose |
|-----------|-------|------|---------|
| snowflake | `auto_qa_dedupe_table_mapping` | `map_string` | Table deduplication mapping |
| clickhouse | `topics_config_map` | `map_nested` | Per-topic delete SQL config |

#### Override Precedence

When an `api_field_name` in overrides matches a field in the backend config, the override takes precedence and the backend field is skipped. This prevents duplicate fields.

An override's `api_field_name` **must** resolve to a field the backend declares; tfgen fails the build otherwise. Overrides are hand-written and nothing else validates them, so an override can outlive the backend field it targets and go on generating an attribute that Terraform accepts and the backend silently discards. That is not hypothetical: `sqlserveraws.snapshot_custom_table_config` did exactly that until the check was added.

### Generated Code Structure

Each generated file contains:

```go
// 1. Nested model types (if map_nested overrides exist)
type clickHouseTopicsConfigMapItemModel struct {
    DeleteSQLExecute types.String `tfsdk:"delete_sql_execute"`
}

// 2. Main model struct
type DestinationClickhouseModel struct {
    ID              types.String   `tfsdk:"id"`
    Name            types.String   `tfsdk:"name"`
    Connector       types.String   `tfsdk:"connector"`
    // ... connector-specific fields
    TopicsConfigMap map[string]clickHouseTopicsConfigMapItemModel `tfsdk:"topics_config_map"`
    Timeouts        timeouts.Value `tfsdk:"timeouts"`
}

// 3. Schema function
func DestinationClickhouseSchema() schema.Schema {
    return schema.Schema{
        Description: "Manages a ClickHouse destination connector.",
        Attributes: map[string]schema.Attribute{
            // ... all attributes with descriptions, defaults, validators
        },
    }
}

// 4. Field mappings
var DestinationClickhouseFieldMappings = map[string]string{
    "hostname": "connection.hostname",
    "port":     "connection.port.user.defined",
    // ... TF attribute → API field name
}
```

### Validator Generation

Validators are automatically generated based on backend config:

| Backend Config | Generated Validator |
|----------------|---------------------|
| `control: "one-select"` with `values: ["a", "b"]` | `stringvalidator.OneOf("a", "b")` |
| `control: "slider"` with `min: 1, max: 100` | `int64validator.Between(1, 100)` |

### Sensitive Field Detection

Fields are marked sensitive (`Sensitive: true`) when:
- `control: "password"`
- `encrypt: true` in backend config
- the attribute is named or suffixed `api_key` / `authorization` (`isSecretField`
  in `cmd/tfgen/generator.go`) — the webhook plugins ship `api.key` and the
  http-sink ships `http.headers.authorization` with neither flag set, and the
  backend keeps regressing it. Fix credential-marking gaps there, never by editing
  `internal/generated/`.

### Default Value Handling

| Backend | Generated |
|---------|-----------|
| `default: "value"` | `stringdefault.StaticString("value")` |
| `default: 5432` (on port field) | `int64default.StaticInt64(5432)` |
| `default: true` | `booldefault.StaticBool(true)` |

### Required/Optional/Computed Logic

| Backend Config | Terraform Schema |
|----------------|------------------|
| `required: true`, no default | `Required: true` |
| `required: true`, has default | `Optional: true, Computed: true` |
| `required: false` | `Optional: true` |
| `user_defined: false` | Field skipped (not user-editable) |

## Supported Type Mappings

### Nested Map Types

The base resource supports nested map types (`map[string]struct`) for complex configurations:

**ClickHouse `topics_config_map`:**
```hcl
resource "streamkap_destination_clickhouse" "example" {
  # ...
  topics_config_map = {
    "my_topic" = {
      delete_sql_execute = "DELETE FROM my_table WHERE id = ?"
    }
  }
}
```

(The backend plugin is named `sqlserveraws`, so the generated schema is
`internal/generated/source_sqlserveraws.go` — but the Terraform resource it backs
is `streamkap_source_sqlserver`. `overrides.json` keys on the backend name.)

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

**Note:** If `implementation_json` is not specified, the implementation is managed outside Terraform (e.g., via Streamkap UI) and is preserved during updates.

## Non-Connector Resources

Resources that don't follow the connector pattern have their own implementations:
`pipeline`, `topic`, and `tag` implement CRUD directly against the API client.

> **Not currently registered:** `kafka_user`, `client_credential`, and the `roles`
> data source below are **not** in `provider.go`'s `Resources()` / `DataSources()`
> — they were unregistered in `084d08f`, with the source preserved for future
> re-enabling. A config referencing them today fails with "provider does not
> support resource type". The design notes are kept here for whoever re-enables
> them; they are not a description of the shipping provider.

### Kafka User (`internal/resource/kafka_user/`) — not registered
- CRUD via `/kafka-access/kafka-users` endpoints
- `username` is the resource ID (ForceNew — cannot change after creation)
- `password` is write-only (not returned by API on read, uses `UseStateForUnknown`)
- `kafka_acls` is a `ListNestedBlock` with ACL rules (topic_name, operation, resource_pattern_type, resource)
- Import uses username as the ID
- No individual GET endpoint — reads filter from list

### Client Credential (`internal/resource/client_credential/`) — not registered
- Create/List/Delete only — no Update endpoint exists in the backend
- All writable fields (`role_ids`, `description`, `service_id`) use ForceNew plan modifiers
- `secret` is only returned on creation, preserved in state on reads
- `roles` is a computed `ListNestedBlock` resolved from `role_ids`
- `role_ids` uses the framework-standard `listplanmodifier.RequiresReplace()`

### Roles Data Source (`internal/datasource/roles.go`) — not registered
- Lists available roles from `/auth/roles`
- Would be used to discover role IDs for `streamkap_client_credential` resources

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
│   │   ├── client_credential.go  # Client Credential Create/List/Delete
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
│   │   ├── kafka_user/           # Kafka user resource (not registered)
│   │   └── client_credential/    # Client credential resource (not registered)
│   │
│   ├── datasource/               # Data sources
│   │   ├── transform.go          # Transform datasource
│   │   ├── tag.go                # Tag datasource (single, by id)
│   │   ├── tags.go               # Tags list/filter datasource
│   │   ├── topics.go             # Topics list datasource
│   │   ├── topic.go              # Topic datasource
│   │   ├── topic_metrics.go      # Topic metrics datasource
│   │   ├── topic_serialization.go # Shared topic (de)serialization helpers
│   │   └── roles.go              # Roles list datasource (not registered)
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

### Generator Tests (`cmd/tfgen/*_test.go`)

**Unit tests** verify individual components:
- `TestToPascalCase` - Attribute name to Go field name conversion
- `TestFieldTypeMapping` - Control type → Terraform type mapping
- `TestSensitiveFieldHandling` - Sensitive field detection
- `TestDefaultValueHandling` - Default value generation
- `TestValidatorGeneration` - Validator code generation

**Integration tests** verify the full pipeline:
- `TestGenerateFile_Integration` - Full file generation
- `TestGeneratePostgreSQL_Integration` - Real connector generation
- `TestGenerateSnowflake_Integration` - Connector with overrides

```go
// Example integration test
func TestGeneratePostgreSQL_Integration(t *testing.T) {
    backendPath := os.Getenv("STREAMKAP_BACKEND_PATH")
    if backendPath == "" {
        t.Skip("STREAMKAP_BACKEND_PATH not set")
    }
    // Parse real backend config
    // Generate code
    // Verify output compiles
}
```

### Acceptance Tests (`internal/provider/*_test.go`)

Acceptance tests create real resources in the Streamkap API:

```go
func TestAccSourcePostgreSQL_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Step 1: Create
            {
                Config: testAccSourcePostgreSQLConfig("test-source"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr(
                        "streamkap_source_postgresql.test", "name", "test-source"),
                    resource.TestCheckResourceAttrSet(
                        "streamkap_source_postgresql.test", "id"),
                ),
            },
            // Step 2: Import
            {
                ResourceName:      "streamkap_source_postgresql.test",
                ImportState:       true,
                ImportStateVerify: true,
                ImportStateVerifyIgnore: []string{"database_password"},
            },
            // Step 3: Update
            {
                Config: testAccSourcePostgreSQLConfig("test-source-updated"),
                Check: resource.TestCheckResourceAttr(
                    "streamkap_source_postgresql.test", "name", "test-source-updated"),
            },
        },
    })
}
```

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
3. Unit + schema-compat + validator tests - `make test test-schema test-validators`
4. Lint - `golangci-lint` (separate job)

It pins `TF_ACC=""` for the whole workflow so a stray value can never turn the
credential-free tiers into a live-API run.

### docs-drift.yml - Docs match committed schemas

On PRs touching `internal/generated/`, `docs/`, `templates/`, `examples/provider/`,
`main.go` or `go.mod`: re-renders the docs from the *committed* schemas with
`tfplugindocs` (no backend needed) and fails if `docs/` changes. This structurally
prevents the beta.18 bug where `go generate ./...` rendered docs one regen behind.

### acceptance.yml / pr-acceptance.yml / migration.yml - Acceptance suites

`acceptance.yml` runs the full `TestAcc` suite on a schedule and on push;
`pr-acceptance.yml` runs a curated subset (`scripts/acceptance-tests.txt`) on PRs;
`migration.yml` runs the v2 → v3 `TestAcc.*Migration` suite. All three need API
credentials, sourced from 1Password, so they skip fork PRs.

### security.yml - Security Scanning

Runs on push and PR:
- **Trivy** - Vulnerability scanning
- **Checkov** - Infrastructure-as-code security

### regenerate.yml - Schema Regeneration

Manually dispatched workflow to regenerate connector schemas against the backend
repository. Locally the equivalent is
`STREAMKAP_BACKEND_PATH=<path> make generate` — never `go generate ./...`.

### release.yml - Release Automation

Triggered on version tags (`v*`):
1. Run GoReleaser
2. Build binaries for all platforms
3. Publish to Terraform Registry
