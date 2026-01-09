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
├──────────────────┬──────────────────┬───────────────────────────┤
│  Sources (6)     │  Destinations (7)│  Other                    │
│  - PostgreSQL    │  - Snowflake     │  - Pipeline               │
│  - MySQL         │  - ClickHouse    │  - Topic                  │
│  - MongoDB       │  - Databricks    │  - Tag                    │
│  - DynamoDB      │  - PostgreSQL    │                           │
│  - SQLServer     │  - S3            │                           │
│  - KafkaDirect   │  - Iceberg       │                           │
│                  │  - Kafka         │                           │
└──────────────────┴──────────────────┴───────────────────────────┘
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
│  - Source, Destination, Pipeline, Topic, Tag CRUD                │
└─────────────────────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Streamkap API                                  │
│  https://api.streamkap.com                                       │
└─────────────────────────────────────────────────────────────────┘
```

## Code Generation Flow

```
┌──────────────────────────┐
│  Backend Repository      │
│  configuration.latest.json│
└───────────┬──────────────┘
            │
            ▼
┌──────────────────────────┐
│  cmd/tfgen/parser.go     │
│  - Parse JSON config     │
│  - Extract field info    │
│  - Map control types     │
└───────────┬──────────────┘
            │
            ▼
┌──────────────────────────┐
│  cmd/tfgen/generator.go  │
│  - Apply templates       │
│  - Generate Go code      │
│  - Create field mappings │
└───────────┬──────────────┘
            │
            ▼
┌──────────────────────────┐
│  internal/generated/     │
│  - source_*.go           │
│  - destination_*.go      │
│  Contains:               │
│  - Schema function       │
│  - Model struct          │
│  - Field mappings        │
└──────────────────────────┘
```

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
│   │   └── transform.go          # Transform read
│   │
│   ├── generated/                # Generated code (DO NOT EDIT)
│   │   ├── doc.go                # Package doc
│   │   ├── source_*.go           # Generated source schemas
│   │   └── destination_*.go      # Generated destination schemas
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
