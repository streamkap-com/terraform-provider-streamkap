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
│  Sources (20)    │  Destinations(23)│  Transforms (8)  │ Other  │
│  PostgreSQL      │  Snowflake       │  MapFilter       │Pipeline│
│  MySQL, MongoDB  │  ClickHouse      │  Enrich          │ Topic  │
│  DynamoDB        │  Databricks      │  EnrichAsync     │  Tag   │
│  SQLServer       │  PostgreSQL, S3  │  SQLJoin         │        │
│  KafkaDirect     │  Iceberg, Kafka  │  Rollup, FanOut  │        │
│  Oracle, Redis   │  BigQuery, GCS   │  ToastHandling   │        │
│  + 12 more...    │  + 15 more...    │  UnNesting       │        │
└──────────────────┴──────────────────┴──────────────────┴────────┘
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
| sqlserveraws | `snapshot_custom_table_config` | `map_nested` | Custom snapshot parallelism |

#### Override Precedence

When an `api_field_name` in overrides matches a field in the backend config, the override takes precedence and the backend field is skipped. This prevents duplicate fields.

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
2. Extract name from model
3. Convert model to API config (ModelToAPIConfig)
4. Call API CreateSource/CreateDestination
5. Store ID in state

Read:
1. Call API GetSource/GetDestination
2. Convert API response to model (APIConfigToModel)
3. Update state

Update:
1. Read TF config into model
2. Convert to API config
3. Call API UpdateSource/UpdateDestination
4. Update state

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
│   │   └── transform.go          # Transform CRUD
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
│   │   ├── source/               # Source config wrappers
│   │   │   └── *_generated.go    # ConnectorConfig implementations
│   │   ├── destination/          # Destination config wrappers
│   │   │   └── *_generated.go    # ConnectorConfig implementations
│   │   ├── transform/            # Transform config wrappers
│   │   │   ├── base.go           # BaseTransformResource
│   │   │   └── *_generated.go    # TransformConfig implementations
│   │   ├── pipeline/             # Pipeline resource
│   │   ├── topic/                # Topic resource
│   │   └── tag/                  # Tag resource
│   │
│   ├── datasource/               # Data sources
│   │   ├── transform.go          # Transform datasource
│   │   └── tag.go                # Tag datasource
│   │
│   └── helper/                   # Utility functions
│       └── helper.go             # Type conversion helpers
│
├── examples/                     # Example TF configs
│   └── resources/
│       └── streamkap_*/
│
├── docs/                         # Documentation
│   ├── DEVELOPMENT.md            # Developer guide
│   ├── ARCHITECTURE.md           # This file
│   └── plans/                    # Implementation plans
│
└── .github/workflows/            # CI/CD
    ├── ci.yml                    # Build + test
    ├── security.yml              # Security scans
    ├── regenerate.yml            # Schema regeneration
    └── release.yml               # Release automation
```

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
| `STREAMKAP_HOST` | API endpoint | Acceptance tests |
| `STREAMKAP_CLIENT_ID` | OAuth client ID | Acceptance tests |
| `STREAMKAP_SECRET` | OAuth secret | Acceptance tests |
| `STREAMKAP_BACKEND_PATH` | Backend repo path | Generator integration tests |

### Running Tests

```bash
# Unit tests only (fast, no external deps)
go test -v -short ./...

# Generator tests
go test -v ./cmd/tfgen/...

# Generator integration tests (requires backend)
STREAMKAP_BACKEND_PATH=/path/to/backend go test -v ./cmd/tfgen/...

# Acceptance tests (creates real resources)
TF_ACC=1 STREAMKAP_HOST=https://api.streamkap.com \
  STREAMKAP_CLIENT_ID=xxx STREAMKAP_SECRET=xxx \
  go test -v ./internal/provider -timeout 30m

# Single acceptance test
go test -v ./internal/provider -run TestAccSourcePostgreSQL_basic
```

### Test Best Practices

1. **Use unique resource names** - Include timestamp or random suffix to avoid conflicts
2. **Clean up resources** - Tests should delete resources they create
3. **Skip when credentials missing** - Use `t.Skip()` for optional tests
4. **Verify import** - All resources should support import
5. **Test updates** - Verify in-place updates work correctly

## CI/CD Workflows

### ci.yml - Continuous Integration

Runs on every PR and push to main:
1. Build - `go build ./...`
2. Lint - `golangci-lint run`
3. Unit Tests - `go test -short ./...`
4. Generator Tests - `go test ./cmd/tfgen/...`

### security.yml - Security Scanning

Runs security scans:
- **Trivy** - Vulnerability scanning
- **Checkov** - Infrastructure-as-code security

### regenerate.yml - Schema Regeneration

Manual workflow to regenerate all connector schemas:
1. Checkout backend repository
2. Run `go run ./cmd/tfgen generate --backend-path=...`
3. Create PR with changes

### release.yml - Release Automation

Triggered on version tags (`v*`):
1. Run GoReleaser
2. Build binaries for all platforms
3. Publish to Terraform Registry
