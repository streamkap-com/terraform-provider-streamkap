# Terraform Provider Architecture Comparison Report

**Date:** 2026-01-21
**Auditor:** Claude Agent
**Branch Under Audit:** `ralph/terraform-provider-audit` (based on `ralph-refactored-terraform`)

---

## Table of Contents

1. [Main Branch Architecture](#main-branch-architecture)
2. [Refactored Architecture](#refactored-architecture)
3. [Comparison Matrix](#comparison-matrix)
4. [Trade-offs](#trade-offs)

---

## Main Branch Architecture

The main branch uses a **monolithic connector pattern** where each connector is a self-contained unit with all logic embedded in a single file.

### Representative Files Analyzed

| File | Lines of Code | Purpose |
|------|---------------|---------|
| `internal/resource/source/postgresql.go` | 589 | PostgreSQL source connector |
| `internal/resource/destination/snowflake.go` | 503 | Snowflake destination connector |

### Architecture Characteristics

#### 1. Manual Schema Definitions (~100-150 lines per connector)
Each connector manually defines its Terraform schema inline within the `Schema()` method:

```go
func (r *SourcePostgreSQLResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Source PostgreSQL resource",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{...},
            "database_hostname": schema.StringAttribute{
                Required:    true,
                Description: "PostgreSQL Hostname...",
            },
            // ... 20+ more attributes manually defined
        },
    }
}
```

#### 2. Inline CRUD Operations (~100-150 lines per connector)
Each connector implements its own Create, Read, Update, Delete methods with connector-specific logic:

```go
func (r *SourcePostgreSQLResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
    var plan SourcePostgreSQLResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

    config, err := r.model2ConfigMap(plan)
    // ... connector-specific create logic
}
```

#### 3. Type Conversion Methods (~100-200 lines per connector)
Each connector has dedicated `model2ConfigMap()` and `configMap2Model()` methods for converting between Terraform types and API maps:

```go
func (r *SourcePostgreSQLResource) model2ConfigMap(model SourcePostgreSQLResourceModel) (map[string]any, error) {
    configMap := map[string]any{
        "database.hostname.user.defined": model.DatabaseHostname.ValueString(),
        "database.port.user.defined":     int(model.DatabasePort.ValueInt64()),
        // ... manual mapping for each field
    }
    return configMap, nil
}

func (r *SourcePostgreSQLResource) configMap2Model(cfg map[string]any, model *SourcePostgreSQLResourceModel) {
    model.DatabaseHostname = helper.GetTfCfgString(cfg, "database.hostname.user.defined")
    model.DatabasePort = helper.GetTfCfgInt64(cfg, "database.port.user.defined")
    // ... manual mapping for each field
}
```

#### 4. Static Resource Model (~30-50 lines per connector)
Each connector defines its own Terraform model struct:

```go
type SourcePostgreSQLResourceModel struct {
    ID               types.String `tfsdk:"id"`
    Name             types.String `tfsdk:"name"`
    Connector        types.String `tfsdk:"connector"`
    DatabaseHostname types.String `tfsdk:"database_hostname"`
    DatabasePort     types.Int64  `tfsdk:"database_port"`
    // ... 20+ more fields
}
```

### Main Branch Pattern Summary

| Component | Location | Approximate LOC |
|-----------|----------|-----------------|
| Resource model struct | Inline in connector file | 30-50 |
| Schema definition | `Schema()` method | 100-150 |
| CRUD operations | Create/Read/Update/Delete methods | 100-150 |
| Type conversion | `model2ConfigMap`/`configMap2Model` | 100-200 |
| Metadata/Configure/Import | Standard methods | ~50 |
| **Total per connector** | Single file | **~400-600** |

### Key Observations

1. **Self-contained**: Each connector is completely independent with no shared logic
2. **Repetitive**: Significant code duplication across connectors (CRUD patterns, model struct patterns)
3. **Manual maintenance**: Schema changes require manual updates to model struct, schema, and both conversion methods
4. **Coupled**: Schema definitions tightly coupled to implementation logic
5. **Flexible**: Easy to add connector-specific validation or behavior
6. **Uses helpers**: Type conversion uses shared `helper.GetTfCfgString/Int64/Bool` functions

---

## Refactored Architecture

The refactored branch uses a **three-layer code generation architecture** that separates concerns into distinct layers: Generated Schemas, Thin Wrappers, and a Shared Base Resource.

### Three-Layer Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Layer 1: Generated Schemas                        │
│                   internal/generated/*.go                            │
│    ┌────────────────┐  ┌──────────────────┐  ┌─────────────────┐    │
│    │ source_*.go    │  │ destination_*.go │  │ transform_*.go  │    │
│    │ (20 files)     │  │ (23 files)       │  │ (8 files)       │    │
│    │ 4,477 LOC      │  │ 3,889 LOC        │  │ 900 LOC         │    │
│    └────────────────┘  └──────────────────┘  └─────────────────┘    │
│    // Code generated by tfgen. DO NOT EDIT.                          │
│    Contains: Model structs, Schema functions, Field mappings         │
└──────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Layer 2: Thin Wrappers                            │
│         internal/resource/source/*_generated.go                      │
│         internal/resource/destination/*_generated.go                 │
│    ┌────────────────────────────┐  ┌────────────────────────────┐   │
│    │ Source wrappers (20 files) │  │ Dest wrappers (22 files)   │   │
│    │ 1,095 LOC total (~55 each) │  │ 1,141 LOC total (~52 each) │   │
│    └────────────────────────────┘  └────────────────────────────┘   │
│    Implements: ConnectorConfig interface                             │
│    Contains: Deprecated field handling, Custom overrides             │
└──────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Layer 3: Shared Base Resource                     │
│                 internal/resource/connector/base.go                  │
│                            (1 file, 701 LOC)                         │
│    ┌─────────────────────────────────────────────────────────────┐  │
│    │ BaseConnectorResource                                        │  │
│    │ - Implements Resource, ResourceWithConfigure, ImportState   │  │
│    │ - Generic Create/Read/Update/Delete with reflection         │  │
│    │ - modelToConfigMap() / configMapToModel() via reflection    │  │
│    │ - Timeout handling, error handling, logging                  │  │
│    └─────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────┘
```

### Verification: DO NOT EDIT Markers

All generated files in `internal/generated/` contain the marker:
```go
// Code generated by tfgen. DO NOT EDIT.
```

Verified in 3 sample files:
- ✅ `internal/generated/destination_azblob.go`
- ✅ `internal/generated/destination_bigquery.go`
- ✅ `internal/generated/destination_clickhouse.go`

### File Counts by Layer

| Layer | Directory | File Count | Total LOC | Avg LOC/File |
|-------|-----------|------------|-----------|--------------|
| **Generated Schemas** | `internal/generated/` | 52 | 9,266+ | ~178 |
| └─ Source schemas | `source_*.go` | 20 | 4,477 | ~224 |
| └─ Destination schemas | `destination_*.go` | 23 | 3,889 | ~169 |
| └─ Transform schemas | `transform_*.go` | 8 | 900 | ~113 |
| └─ Documentation | `doc.go` | 1 | - | - |
| **Thin Wrappers** | `internal/resource/source/` | 20 | 1,095 | ~55 |
|                   | `internal/resource/destination/` | 22 | 1,141 | ~52 |
| **Shared Base** | `internal/resource/connector/` | 1 | 701 | 701 |

### Layer 1: Generated Schemas (internal/generated/)

The code generator (`cmd/tfgen`) reads backend `configuration.latest.json` files and generates:

**Model Struct** (auto-generated from config fields):
```go
// Code generated by tfgen. DO NOT EDIT.

// SourcePostgresqlModel is the Terraform model for the postgresql source.
type SourcePostgresqlModel struct {
    ID               types.String `tfsdk:"id"`
    Name             types.String `tfsdk:"name"`
    DatabaseHostname types.String `tfsdk:"database_hostname"`
    DatabasePort     types.Int64  `tfsdk:"database_port"`
    // ... auto-generated from backend config
}
```

**Schema Function** (auto-generated with validators, defaults, descriptions):
```go
// SourcePostgresqlSchema returns the Terraform schema for the postgresql source.
func SourcePostgresqlSchema() schema.Schema {
    return schema.Schema{
        Description: "Manages a PostgreSQL source connector.",
        MarkdownDescription: "Manages a **PostgreSQL source connector**...",
        Attributes: map[string]schema.Attribute{
            "database_hostname": schema.StringAttribute{
                Required:    true,
                Description: "PostgreSQL server hostname or IP address",
                // ... auto-generated validators, defaults
            },
        },
    }
}
```

**Field Mappings** (auto-generated TF attribute → API field name):
```go
// SourcePostgresqlFieldMappings maps Terraform attributes to API field names.
var SourcePostgresqlFieldMappings = map[string]string{
    "database_hostname": "database.hostname.user.defined",
    "database_port":     "database.port.user.defined",
    // ... auto-generated mappings
}
```

### Layer 2: Thin Wrappers (internal/resource/source/, internal/resource/destination/)

Each wrapper file (~50-60 LOC) implements the `ConnectorConfig` interface:

```go
// PostgreSQLConfig implements the ConnectorConfig interface for PostgreSQL sources.
type PostgreSQLConfig struct{}

// GetSchema returns the Terraform schema (with any deprecated field additions).
func (c *PostgreSQLConfig) GetSchema() schema.Schema {
    s := generated.SourcePostgresqlSchema()
    // Add deprecated aliases for backward compatibility
    s.Attributes["insert_static_key_field_1"] = schema.StringAttribute{
        Optional:           true,
        Computed:           true,
        DeprecationMessage: "Use 'transforms_insert_static_key1_static_field' instead.",
    }
    return s
}

// GetFieldMappings returns mappings including deprecated aliases.
func (c *PostgreSQLConfig) GetFieldMappings() map[string]string {
    return postgresqlFieldMappings // includes deprecated → API mappings
}

// GetConnectorType, GetConnectorCode, GetResourceName, NewModelInstance...
```

### Layer 3: Shared Base Resource (internal/resource/connector/base.go)

The base resource (701 LOC) handles all CRUD operations generically via reflection:

```go
// ConnectorConfig interface that wrappers implement
type ConnectorConfig interface {
    GetSchema() schema.Schema
    GetFieldMappings() map[string]string
    GetConnectorType() ConnectorType  // "source" or "destination"
    GetConnectorCode() string         // e.g., "postgresql"
    GetResourceName() string          // e.g., "source_postgresql"
    NewModelInstance() any            // creates model for reflection
}

// BaseConnectorResource - generic resource for all connectors
type BaseConnectorResource struct {
    client api.StreamkapAPI
    config ConnectorConfig
}

// Create uses reflection to convert model → API config map
func (r *BaseConnectorResource) Create(ctx context.Context, ...) {
    model := r.config.NewModelInstance()
    req.Plan.Get(ctx, model)
    configMap := r.modelToConfigMap(model) // reflection-based
    result, err := r.client.CreateSource(...)
    r.configMapToModel(result.Config, model) // reflection-based
    resp.State.Set(ctx, model)
}
```

### Key Architecture Characteristics

1. **Separation of Concerns**
   - Generated code: Pure data (schemas, mappings, models)
   - Wrappers: Connector-specific customizations (deprecations, overrides)
   - Base: Shared behavior (CRUD, reflection, error handling)

2. **Code Generation**
   - Schemas derived from backend `configuration.latest.json`
   - Type mappings handled automatically (string→types.String, etc.)
   - Validators auto-generated from constraints

3. **Reflection-Based Marshaling**
   - `modelToConfigMap()`: Uses struct tags + field mappings
   - `configMapToModel()`: Uses field mappings + type inference
   - Handles nested structures, optional fields, defaults

4. **Deprecation System**
   - Deprecated fields defined in wrappers
   - Map to same API fields as new names
   - Automatic `DeprecationMessage` on schema attributes

### Refactored Pattern Summary

| Component | Location | Per-Connector LOC | Total |
|-----------|----------|-------------------|-------|
| Model + Schema + Mappings | `internal/generated/` | ~180 (generated) | 9,266 |
| Wrapper (ConnectorConfig) | `internal/resource/{source,dest}/` | ~55 | 2,236 |
| Shared CRUD | `internal/resource/connector/base.go` | 0 (shared) | 701 |
| **Total per connector** | | **~235** | - |

Compared to main branch's ~500 LOC per connector, the refactored architecture achieves **~53% reduction in per-connector code** while centralizing shared logic.

---

## Comparison Matrix

*Section to be completed in US-005*

---

## Trade-offs

*Section to be completed in US-005*

---
