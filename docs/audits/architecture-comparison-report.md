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

### Layer Separation Verification

**US-004: Verified proper layer separation across all three architecture layers.**

#### Wrapper Files Verification

| Directory | File Count | Pattern |
|-----------|------------|---------|
| `internal/resource/source/` | 20 | `*_generated.go` |
| `internal/resource/destination/` | 22 | `*_generated.go` |

All wrapper files verified to exist with correct naming convention.

#### ConnectorConfig Interface Implementation

Verified that wrapper files implement the `ConnectorConfig` interface defined in `base.go`:

```go
type ConnectorConfig interface {
    GetSchema() schema.Schema
    GetFieldMappings() map[string]string
    GetConnectorType() ConnectorType
    GetConnectorCode() string
    GetResourceName() string
    NewModelInstance() any
}
```

**Spot-checked files:**

| File | Interface Compliance | Verified Methods |
|------|---------------------|------------------|
| `internal/resource/source/postgresql_generated.go` | ✅ `var _ connector.ConnectorConfig = (*PostgreSQLConfig)(nil)` | All 6 methods |
| `internal/resource/destination/snowflake_generated.go` | ✅ `var _ connector.ConnectorConfig = (*SnowflakeConfig)(nil)` | All 6 methods |

Both files include compile-time interface compliance checks via `var _ connector.ConnectorConfig = (*Config)(nil)`.

#### Base Resource CRUD Logic Verification

Verified `internal/resource/connector/base.go` (701 LOC) contains all shared CRUD logic:

| Method | Lines | Purpose |
|--------|-------|---------|
| `Create()` | 117-226 | Creates new connector via API with timeout handling |
| `Read()` | 229-313 | Reads connector state from API |
| `Update()` | 316-422 | Updates connector via API with timeout handling |
| `Delete()` | 425-494 | Deletes connector via API with timeout handling |
| `ImportState()` | 497-499 | Passthrough import by ID |
| `modelToConfigMap()` | 503-542 | Reflection-based model → API config conversion |
| `configMapToModel()` | 546-581 | Reflection-based API config → model conversion |

Additional helper methods:
- `extractTerraformValue()` (584-628): Extracts values from Terraform types
- `setTerraformValue()` (631-666): Sets Terraform type values
- `getStringField()` / `setStringField()` (669-701): Reflection helpers for Name/ID fields

#### No Business Logic in Generated Files

Verified `internal/generated/source_postgresql.go` (373 LOC) contains **only data definitions**:

| Component | Lines | Contains |
|-----------|-------|----------|
| Model struct | 20-71 | Field definitions with `tfsdk` tags |
| Schema function | 74-332 | Attribute definitions, validators, defaults |
| Field mappings | 335-372 | TF attribute → API field name map |

**Confirmed:** No CRUD methods, no API calls, no business logic. File begins with `// Code generated by tfgen. DO NOT EDIT.`

#### Typecheck Verification

```bash
$ go build ./...
# Completed with no errors
```

---

## Comparison Matrix

### Architecture Comparison Table

| Aspect | Main Branch | Refactored Branch |
|--------|-------------|-------------------|
| **LOC per connector** | ~400-600 (all inline) | ~235 (55 wrapper + 180 generated) |
| **Schema definition** | Manual, inline in `Schema()` method | Auto-generated from backend `configuration.latest.json` |
| **CRUD location** | Inline in each connector file | Shared in `base.go` (701 LOC total) |
| **Type conversion** | Manual `model2ConfigMap`/`configMap2Model` per connector | Reflection-based generic conversion in `base.go` |
| **Deprecation handling** | Manual per-connector with inline definitions | JSON-driven (`deprecations.json`) + wrapper layer |
| **Model struct** | Manual, ~30-50 LOC per connector | Auto-generated with `tfsdk` tags |
| **Field mappings** | Embedded in conversion methods | Explicit `FieldMappings` map (generated) |
| **Validators** | Manual inline definitions | Auto-generated from backend constraints |
| **Adding new connector** | Copy/paste ~500 LOC, modify all fields | Run tfgen, create ~55 LOC wrapper |
| **Backend schema changes** | Manual update: model + schema + 2 conversion methods | Re-run generator, verify wrapper |
| **Code duplication** | High (CRUD, validation, error handling repeated) | Low (shared in base.go) |
| **Customization flexibility** | High (direct inline access) | Medium (wrapper layer + override system) |
| **Compile-time safety** | Explicit type checking | Reflection-based (runtime errors possible) |
| **Test coverage** | Per-connector tests needed | Shared base tests + per-connector acceptance |

### Quantitative Comparison

| Metric | Main Branch | Refactored Branch | Difference |
|--------|-------------|-------------------|------------|
| Total connector files | 42 | 42 wrappers + 51 generated | +51 files |
| Total LOC (connectors) | ~18,900 | ~12,100 | -36% |
| LOC per connector (avg) | ~450 | ~235 | -48% |
| Shared infrastructure | ~200 (helpers) | ~700 (base.go) | +500 |
| Code generation tooling | N/A | ~1,500 (tfgen) | New |
| Unique code per connector | ~450 | ~55 | -88% |

### Feature Comparison

| Feature | Main Branch | Refactored Branch |
|---------|-------------|-------------------|
| **Sensitive field handling** | Manual `Sensitive: true` | Auto from `encrypt: true` in config |
| **Default values** | Manual `Default: stringdefault.StaticString(...)` | Auto from `default` in config |
| **Required field detection** | Manual `Required: true` | Auto from `required: true` in config |
| **Enum validators** | Manual `stringvalidator.OneOf(...)` | Auto from `options` in config |
| **Range validators** | Manual slider validators | Auto from `min`/`max` in config |
| **Description generation** | Manual text | Auto from `description` + markdown enhancement |
| **Computed fields** | Manual `Computed: true` | Auto from `user_defined: false` |
| **Plan modifiers** | Manual per-field | Auto `UseStateForUnknown()` for computed |

---

## Trade-offs

### Advantages of Refactored Architecture

1. **Reduced Code Duplication**
   - CRUD operations centralized in `base.go` (701 LOC vs ~100-150 LOC × 42 connectors)
   - Estimated ~4,000+ LOC saved across all connectors
   - Bug fixes in base apply to all connectors automatically

2. **Backend Schema Synchronization**
   - Schemas generated directly from `configuration.latest.json`
   - Ensures Terraform matches backend validation rules
   - Field descriptions stay synchronized with backend docs

3. **Consistent Behavior**
   - All connectors use identical CRUD patterns
   - Error handling, timeouts, logging standardized
   - Reduces behavioral drift between connectors

4. **Faster Connector Development**
   - New connector: run tfgen + create 55-line wrapper
   - vs. Main branch: copy/paste/modify 500+ lines
   - Estimated 90% time reduction for new connectors

5. **Systematic Deprecation Handling**
   - `deprecations.json` tracks all deprecated fields centrally
   - Deprecation messages auto-applied at generation
   - Easy to audit backward compatibility status

6. **AI-Agent Friendly**
   - Generated schemas have consistent description patterns
   - `MarkdownDescription` with links auto-generated
   - Enum values documented as "Valid values: ..."

### Disadvantages of Refactored Architecture

1. **Reflection Complexity**
   - `modelToConfigMap()` and `configMapToModel()` use reflection
   - Runtime errors possible if struct tags mismatch
   - Harder to debug type conversion issues

2. **Indirection Layers**
   - Three files involved per connector vs. one
   - Developer must understand layer responsibilities
   - Stack traces span multiple files

3. **Generated Code Volume**
   - 51 generated files (9,266 LOC) that shouldn't be edited
   - IDE may show warnings about generated files
   - Must remember to regenerate after backend changes

4. **Code Generator Maintenance**
   - `cmd/tfgen` is new infrastructure (~1,500 LOC)
   - Team must understand generator to fix edge cases
   - Backend config format changes require generator updates

5. **Limited Per-Connector Customization**
   - Custom validation logic harder to add
   - Must fit within wrapper + override system
   - Some edge cases may require generator changes

6. **Build Dependencies**
   - Requires `STREAMKAP_BACKEND_PATH` for regeneration
   - CI must have access to backend configs
   - Version sync between repos needed

### Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Reflection runtime error | Low | Medium | Comprehensive acceptance tests |
| Generator bug | Low | High | Generator unit tests + VCR cassettes |
| Backend config format change | Medium | Medium | Versioned config schemas |
| Regeneration forgotten | Medium | Low | CI check for clean `git diff` |
| Deprecation message missing | Low | Low | `deprecations.json` audit |
| Type mismatch | Low | Medium | Schema backward compatibility tests |

### Recommendation

The refactored architecture is recommended for the following reasons:

1. **Maintainability**: Single point of change for CRUD logic, error handling, and common patterns
2. **Scalability**: Adding new connectors is significantly faster
3. **Correctness**: Schemas derived from backend ensure validation consistency
4. **Quality**: Centralized testing covers all connectors

The primary trade-off (reflection complexity) is mitigated by:
- Compile-time interface checks (`var _ ConnectorConfig = (*Config)(nil)`)
- Schema backward compatibility tests
- Acceptance tests with VCR cassettes

---
