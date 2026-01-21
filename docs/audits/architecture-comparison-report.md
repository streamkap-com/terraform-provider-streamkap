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
5. [API Client](#api-client)
   - [Retry Logic](#retry-logic)
6. [Architectural Decisions](#architectural-decisions)
7. [Helper Functions](#helper-functions)
8. [Base Resource](#base-resource)
9. [Schema Backward Compatibility Tests](#schema-backward-compatibility-tests)
10. [Deprecated Fields](#deprecated-fields)
11. [Migration Tests](#migration-tests)
12. [Code Generator Parser](#code-generator-parser)
13. [Code Generator Type Mapping](#code-generator-type-mapping)
14. [Override and Deprecation System](#override-and-deprecation-system)
15. [Code Regeneration Test](#code-regeneration-test)

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

## API Client

This section documents the API client implementation and authentication mechanisms.

### OAuth2 Token Exchange

**Location**: `internal/api/auth.go`

The API client implements OAuth2 token exchange via the `GetAccessToken()` method:

```go
type Token struct {
    AccessToken  string `json:"accessToken"`
    Expires      string `json:"expires"`
    ExpiresIn    int64  `json:"expiresIn"`
    RefreshToken string `json:"refreshToken"`
}

type GetAccessTokenRequest struct {
    ClientID string `json:"client_id"`
    Secret   string `json:"secret"`
}

func (s *streamkapAPI) GetAccessToken(clientID, secret string) (*Token, error) {
    body := &GetAccessTokenRequest{
        ClientID: clientID,
        Secret:   secret,
    }
    // POST to /auth/access-token
    req, err := http.NewRequestWithContext(ctx, http.MethodPost,
        s.cfg.BaseURL+"/auth/access-token", bytes.NewBuffer(payload))
    // ...
}
```

**Verification**: ✅ OAuth2 token exchange function exists at `auth.go:22-45`

### Authorization Header

**Location**: `internal/api/client.go:80-86`

All authenticated requests include the Bearer token in the `doRequest()` method:

```go
func (s *streamkapAPI) doRequest(ctx context.Context, req *http.Request, result interface{}) error {
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    if s.token != nil {
        req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token.AccessToken))
    }
    // ...
}
```

**Verification**: ✅ Authorization header set at `client.go:84-86`

### Secret Returned Parameter

**Location**: All source, destination, transform, and pipeline API operations

The `?secret_returned=true` query parameter is included in all CRUD operations to ensure sensitive fields are returned in API responses. This is necessary for Terraform to properly manage state for resources with sensitive configuration.

**Usage Summary**:

| API File | Operations | Uses `secret_returned=true` |
|----------|------------|----------------------------|
| `source.go` | Create, Get, Delete, Update | ✅ All operations |
| `destination.go` | Create, Get, Delete, Update | ✅ All operations |
| `transform.go` | Get, Create, Update | ✅ All operations |
| `pipeline.go` | Create, Get, Delete, Update | ✅ All operations |

**Example** (from `source.go:47`):
```go
req, err := http.NewRequestWithContext(ctx, http.MethodPost,
    s.cfg.BaseURL+"/sources?secret_returned=true", bytes.NewBuffer(payload))
```

**Verification**: ✅ `?secret_returned=true` consistently used across 15 API endpoints

### API Client Interface

**Location**: `internal/api/client.go:15-53`

The `StreamkapAPI` interface defines all available API operations:

```go
type StreamkapAPI interface {
    GetAccessToken(clientID, secret string) (*Token, error)
    SetToken(token *Token)

    // Source APIs (4 methods)
    CreateSource, UpdateSource, GetSource, DeleteSource

    // Destination APIs (4 methods)
    CreateDestination, UpdateDestination, GetDestination, DeleteDestination

    // Pipeline APIs (4 methods)
    CreatePipeline, UpdatePipeline, GetPipeline, DeletePipeline

    // Transform APIs (4 methods)
    CreateTransform, UpdateTransform, GetTransform, DeleteTransform

    // Tags APIs (4 methods)
    GetTag, CreateTag, UpdateTag, DeleteTag

    // Topic APIs (3 methods)
    GetTopic, UpdateTopic, DeleteTopic
}
```

### Typecheck Verification

```bash
$ go build ./...
# Completed with no errors
```

### Error Handling

**Location**: `internal/api/client.go:55-110`

The API client extracts error details from the `detail` field in API error responses:

```go
type APIErrorResponse struct {
    Detail string `json:"detail"`
}

func (s *streamkapAPI) doRequest(ctx context.Context, req *http.Request, result interface{}) error {
    // ...
    if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
        var apiErr APIErrorResponse
        if err := respBodyDecoder.Decode(&apiErr); err != nil {
            tflog.Debug(ctx, fmt.Sprintf("...Failed to parse API error response: %v", err))
            return err
        } else {
            return errors.New(apiErr.Detail)  // Extract detail field
        }
    }
    // ...
}
```

**Verification**: ✅ API `detail` field errors extracted at `client.go:96-110`

### Resource Origin Tracking (created_from)

**Location**: All Create operations in `source.go`, `destination.go`, `transform.go`, `pipeline.go`, `tag.go`

All resource creation operations inject `created_from: terraform` to track resource origin in the backend:

| API File | Function | Line | Pattern |
|----------|----------|------|---------|
| `source.go` | `CreateSource()` | 40 | `payloadMap["created_from"] = constants.TERRAFORM` |
| `destination.go` | `CreateDestination()` | 40 | `payloadMap["created_from"] = constants.TERRAFORM` |
| `transform.go` | `CreateTransform()` | 77-78 | `reqPayload.CreatedFrom = constants.TERRAFORM` |
| `pipeline.go` | `CreatePipeline()` | 64 | `payloadMap["created_from"] = constants.TERRAFORM` |
| `tag.go` | `CreateTag()` | 74 | `payloadMap["created_from"] = constants.TERRAFORM` |

**Verification**: ✅ `created_from: terraform` injected on all 5 resource type create operations

### API Unit Test Results

**Command**: `go test -v -short ./internal/api/...`

**Results**: All 21 tests passed (1.158s)

| Test Category | Tests | Status |
|---------------|-------|--------|
| Source CRUD | `TestGetSource_Success`, `TestGetSource_NotFound`, `TestGetSource_APIError`, `TestCreateSource_Success`, `TestCreateSource_ValidationError`, `TestCreateSource_Unauthorized`, `TestDeleteSource_Success`, `TestDeleteSource_NotFound`, `TestUpdateSource_Success` | ✅ 9 PASS |
| Authentication | `TestGetAccessToken_Success`, `TestGetAccessToken_InvalidCredentials`, `TestSetToken`, `TestNewClient` | ✅ 4 PASS |
| Retry Logic | `TestIsRetryableError` (16 sub-tests), `TestRetryWithBackoff_SucceedsOnFirstTry`, `TestRetryWithBackoff_RetriesOnTransientError`, `TestRetryWithBackoff_FailsImmediatelyOnNonRetryable`, `TestRetryWithBackoff_ContextCancellation`, `TestRetryWithBackoff_MaxRetriesExhausted`, `TestDefaultRetryConfig` | ✅ 8 PASS |

**Noteworthy Tests**:
- `TestCreateSource_Success`: Verifies `created_from: terraform` is included in request body (line 161-165)
- `TestIsRetryableError`: Verifies retry logic for 502, 503, 504 status codes and transient errors

### Retry Logic

**Location**: `internal/api/retry.go` (121 lines)

The API client implements retry logic with exponential backoff for transient errors. This ensures resilience against temporary infrastructure issues and backend exhaustion of internal retries.

#### Retryable Error Detection

The `IsRetryableError()` function (lines 12-63) determines whether an error should trigger a retry. It checks for four categories of transient errors:

| Category | Patterns | Description |
|----------|----------|-------------|
| **Backend Timeout** | `kafkaconnecttimeout`, `request timed out`, `sockettimeoutexception` | Backend exhausted internal KC server retries |
| **Gateway Errors** | `502`, `503`, `504`, `bad gateway`, `service unavailable`, `gateway timeout` | Infrastructure/load balancer issues |
| **Network Errors** | `connection refused`, `connection reset`, `no such host`, `network unreachable`, `i/o timeout` | Network connectivity problems |
| **Kafka Transient** | `rebalance_in_progress`, `leader_not_available`, `not_leader_for_partition` | Kafka cluster instability |

**Verification**: ✅ Status codes 502, 503, 504 included in `gatewayPatterns` at line 31

#### Exponential Backoff Implementation

The `RetryWithBackoff()` function (lines 82-120) implements the retry mechanism:

```go
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, operation func() error) error {
    var lastErr error
    delay := cfg.MinDelay

    for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
        lastErr = operation()
        if lastErr == nil {
            return nil
        }

        if !IsRetryableError(lastErr) {
            return lastErr // Non-retryable, fail immediately
        }

        // Wait with context cancellation support
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(delay):
        }

        // Exponential backoff with cap
        delay = min(delay*2, cfg.MaxDelay)
    }
    return lastErr
}
```

**Verification**: ✅ Exponential backoff implemented via `delay = min(delay*2, cfg.MaxDelay)` at line 116

#### Default Configuration

The `DefaultRetryConfig()` function (lines 72-80) provides conservative defaults:

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| `MaxRetries` | 3 | Reasonable limit before giving up |
| `MinDelay` | 10 seconds | Conservative: backend may be retrying internally |
| `MaxDelay` | 60 seconds | Cap to avoid excessive waits |

#### Key Characteristics

1. **Context-Aware**: Supports cancellation via `ctx.Done()` channel
2. **Non-Retryable Fast-Fail**: Validation/auth errors fail immediately without retry
3. **Logging**: Debug-level logging of retry attempts with delay and error info
4. **Backend-Aware**: Conservative delays account for backend's internal KC server retries

#### Usage in API Client

The retry logic is integrated via `doRequestWithRetry()` in `client.go` which wraps `doRequest()`:

```go
func (s *streamkapAPI) doRequestWithRetry(ctx context.Context, req *http.Request, result interface{}) error {
    return RetryWithBackoff(ctx, DefaultRetryConfig(), func() error {
        return s.doRequest(ctx, req, result)
    })
}
```

This ensures all API operations automatically benefit from retry handling without per-operation configuration.

---

## Architectural Decisions

This section documents the rationale behind key architectural decisions in the refactored Terraform provider.

### Decision 1: Code Generation Approach

**Context**: Each connector (source/destination) requires a Terraform schema, model struct, and field mappings that must match the backend API configuration. The main branch approach required ~400-600 LOC of manually maintained code per connector.

**Decision**: Implement a code generator (`cmd/tfgen`) that reads backend `configuration.latest.json` files and auto-generates:
- Model structs with `tfsdk` tags
- Schema functions with validators, defaults, and descriptions
- Field mapping tables (Terraform attribute → API field name)

**Rationale**:
1. **Single Source of Truth**: Backend `configuration.latest.json` already defines all field metadata (name, type, control, default, required, sensitive). Generating from this ensures Terraform schemas always match backend validation rules.
2. **Reduced Human Error**: Manual schema maintenance led to drift between Terraform and backend. Typos in field names, missing validators, and outdated descriptions were common.
3. **Scalability**: With 42+ connectors, manually maintaining ~450 LOC each is unsustainable. Generation reduces per-connector unique code to ~55 LOC.
4. **Consistency**: All generated schemas follow identical patterns for descriptions, validators, defaults, and plan modifiers.

**Trade-offs Accepted**:
- Generator infrastructure (~1,500 LOC) must be maintained
- Requires `STREAMKAP_BACKEND_PATH` environment variable for regeneration
- Team must understand generator to handle edge cases

### Decision 2: Reflection-Based Marshaling

**Context**: Terraform Plugin Framework requires converting between typed `types.String`/`types.Int64` values and `map[string]any` for API calls. The main branch used manual `model2ConfigMap()` and `configMap2Model()` methods in each connector (~100-200 LOC each).

**Decision**: Implement generic reflection-based conversion in `base.go`:
- `modelToConfigMap()`: Iterates struct fields, reads `tfsdk` tags, extracts values using type switches
- `configMapToModel()`: Uses field mappings + helper functions to set Terraform types

**Rationale**:
1. **Code Reuse**: Single implementation serves all 42+ connectors vs. duplicating conversion logic
2. **Declarative Mappings**: `FieldMappings` table clearly shows Terraform ↔ API relationship without implementation details
3. **Type Safety**: Despite using reflection, the code uses type switches (`types.String`, `types.Int64`, `types.Bool`, `types.List`) that handle all supported Terraform types
4. **Maintainability**: Bug fixes apply to all connectors. Adding support for new types (e.g., `types.Float64`) requires one change.

**Implementation Details** (from `base.go`):
```go
// extractTerraformValue - handles all Terraform types via type switch
switch v := fieldValue.Interface().(type) {
case types.String:
    if v.IsNull() || v.IsUnknown() { return nil }
    return v.ValueString()
case types.Int64:
    if v.IsNull() || v.IsUnknown() { return nil }
    return v.ValueInt64()
// ... cases for Bool, Float64, List
}
```

**Trade-offs Accepted**:
- Runtime errors possible if struct tags mismatch (mitigated by compile-time interface checks)
- Harder to debug type issues (mitigated by acceptance tests)
- Performance overhead from reflection (negligible for Terraform operations)

### Decision 3: Thin Wrapper Pattern

**Context**: Generated schemas are immutable (DO NOT EDIT), but connectors need:
- Deprecated field aliases for backward compatibility
- Custom field overrides for complex types (maps, nested objects)
- Interface implementation for the base resource

**Decision**: Create thin wrapper files (~55 LOC each) that implement `ConnectorConfig` interface:
```go
type ConnectorConfig interface {
    GetSchema() schema.Schema        // Returns generated schema + deprecations
    GetFieldMappings() map[string]string  // Returns generated mappings + deprecated aliases
    GetConnectorType() ConnectorType
    GetConnectorCode() string
    GetResourceName() string
    NewModelInstance() any
}
```

**Rationale**:
1. **Separation of Concerns**: Generated code contains pure data; wrappers contain customization logic
2. **Compile-Time Safety**: Wrappers include `var _ connector.ConnectorConfig = (*Config)(nil)` to verify interface compliance
3. **Deprecation Layer**: Wrappers can add deprecated attributes with `DeprecationMessage` without modifying generated code
4. **Override Flexibility**: Complex fields (map types, nested objects) defined in wrapper, not generator

**Implementation Example** (from wrapper):
```go
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
```

**Trade-offs Accepted**:
- Three files per connector vs. one (generated schema + wrapper + shared base)
- Developers must understand layer responsibilities

### Decision 4: JSON-Based Deprecations and Overrides

**Context**: Managing deprecated field aliases and type overrides across 42+ connectors requires a systematic approach. Inline handling per connector would scatter this information across many files.

**Decision**: Centralize configuration in JSON files:
- `cmd/tfgen/deprecations.json`: Defines deprecated attribute → new attribute mappings
- `cmd/tfgen/overrides.json`: Defines complex field types (map_string, map_nested)

**Deprecations JSON Structure**:
```json
{
  "deprecated_fields": [
    {
      "connector": "postgresql",
      "entity_type": "sources",
      "deprecated_attr": "insert_static_key_field_1",
      "new_attr": "transforms_insert_static_key1_static_field",
      "type": "string"
    }
  ]
}
```

**Overrides JSON Structure**:
```json
{
  "field_overrides": [
    {
      "connector": "snowflake",
      "entity_type": "destinations",
      "api_field_name": "auto.qa.dedupe.table.mapping",
      "terraform_attr_name": "auto_qa_dedupe_table_mapping",
      "type": "map_string",
      "optional": true,
      "description": "Mapping between tables..."
    }
  ]
}
```

**Rationale**:
1. **Centralized Audit**: Single file shows all deprecated fields across all connectors. Easy to review backward compatibility status.
2. **Generator Integration**: Generator reads JSON and adds deprecated fields to model structs automatically
3. **Type Flexibility**: `overrides.json` handles types the generator can't infer (map types, nested objects)
4. **Version Control**: JSON changes are easy to review in PRs. Clear diff of what deprecations were added/removed.

**Current Coverage**:
- `deprecations.json`: 10 deprecated field definitions (9 PostgreSQL source, 1 Snowflake destination)
- `overrides.json`: 3 field overrides (Snowflake map_string, ClickHouse map_nested, SQL Server map_nested)

**Trade-offs Accepted**:
- JSON must stay synchronized with actual field usage
- Generator must parse and apply JSON correctly
- Additional configuration files to maintain

---

## Helper Functions

This section documents the helper function implementations and their test coverage.

### Helper Package Overview

**Location**: `internal/helper/`

The helper package provides utility functions for:
1. **Type Conversion**: Converting API response `map[string]any` to Terraform types
2. **Deprecated Field Migration**: Handling deprecated attribute names

### Test Results

**Command**: `go test -v ./internal/helper/...`

**Results**: All 10 tests passed with 50+ sub-tests (0.704s)

| Test | Sub-tests | Status |
|------|-----------|--------|
| `TestMigrateDeprecatedValues` | 4 | ✅ PASS |
| `TestPostgreSQLDeprecatedAliases` | - | ✅ PASS |
| `TestSnowflakeDeprecatedAliases` | - | ✅ PASS |
| `TestGetTfCfgString` | 10 | ✅ PASS |
| `TestGetTfCfgInt64` | 13 | ✅ PASS |
| `TestGetTfCfgBool` | 7 | ✅ PASS |
| `TestGetTfCfgListString` | 10 | ✅ PASS |
| `TestGetTfCfgStringNilMap` | - | ✅ PASS |
| `TestGetTfCfgInt64NilMap` | - | ✅ PASS |
| `TestGetTfCfgBoolNilMap` | - | ✅ PASS |
| `TestGetTfCfgListStringNilMap` | - | ✅ PASS |

### Type Conversion Functions

The helper package provides four primary type conversion functions used by both main branch and refactored connectors:

| Function | Input Type(s) | Output | Purpose |
|----------|--------------|--------|---------|
| `GetTfCfgString(cfg, key)` | `string` | `types.String` | Extract string values |
| `GetTfCfgInt64(cfg, key)` | `float64`, `string` | `types.Int64` | Extract integer values (handles JSON number encoding) |
| `GetTfCfgBool(cfg, key)` | `bool` | `types.Bool` | Extract boolean values |
| `GetTfCfgListString(ctx, cfg, key)` | `[]interface{}` | `types.List` | Extract string list values |

### Key Test Coverage

#### GetTfCfgString
- Valid string values (including unicode, special characters)
- Empty strings
- Missing keys
- `nil` values
- Type coercion from non-string types

#### GetTfCfgInt64
- `float64` values (JSON number encoding)
- String integer values
- Negative numbers
- Decimal truncation
- Invalid string handling (returns zero)

#### GetTfCfgBool
- `true`/`false` values
- Missing keys (returns `types.BoolNull()`)
- Non-bool types (returns `types.BoolValue(false)`)

#### GetTfCfgListString
- Valid lists with elements
- Empty lists
- Missing keys (returns null list)
- List element type verification

### Deprecated Field Migration

The `MigrateDeprecatedValues()` function handles backward compatibility for renamed fields:

```go
func MigrateDeprecatedValues(model any, deprecatedField, newField string) {
    // If new field is null/empty and deprecated field has value,
    // copy deprecated value to new field
}
```

**Test Coverage**:
- Migrates value when new field not set
- Does not overwrite existing new values
- Handles `nil` and empty deprecated values

### Nil Map Safety

All helper functions safely handle `nil` maps by returning appropriate null/zero values:

| Function | Nil Map Return |
|----------|---------------|
| `GetTfCfgString` | `types.StringNull()` |
| `GetTfCfgInt64` | `types.Int64Null()` |
| `GetTfCfgBool` | `types.BoolNull()` |
| `GetTfCfgListString` | `types.ListNull(types.StringType)` |

### Typecheck Verification

```bash
$ go build ./...
# Completed with no errors
```

---

## Base Resource

This section documents the base connector resource implementation that provides shared CRUD logic for all connectors.

### Overview

**Location**: `internal/resource/connector/base.go` (701 lines)

The `BaseConnectorResource` is the central component of the refactored architecture, providing generic CRUD operations for all source and destination connectors. It implements the Terraform Plugin Framework interfaces and uses reflection for model-to-config conversion.

### Interface Implementation

The base resource implements three Terraform Plugin Framework interfaces:

```go
var (
    _ resource.Resource                = &BaseConnectorResource{}
    _ resource.ResourceWithConfigure   = &BaseConnectorResource{}
    _ resource.ResourceWithImportState = &BaseConnectorResource{}
)
```

| Interface | Purpose |
|-----------|---------|
| `resource.Resource` | Core CRUD operations (Create, Read, Update, Delete) |
| `resource.ResourceWithConfigure` | API client injection from provider |
| `resource.ResourceWithImportState` | Resource import capability |

### Model-Config Conversion Functions

The base resource uses two reflection-based functions for converting between Terraform models and API config maps:

#### modelToConfigMap (lines 501-542)

Converts a Terraform model struct to an API configuration map:

```go
func (r *BaseConnectorResource) modelToConfigMap(ctx context.Context, model any) (map[string]any, error) {
    configMap := make(map[string]any)
    mappings := r.config.GetFieldMappings()

    // Get reflect value (dereference if pointer)
    v := reflect.ValueOf(model)
    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }

    // Build tfsdk tag to field index mapping
    tfsdkToField := make(map[string]int)
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        tag := field.Tag.Get("tfsdk")
        if tag != "" && tag != "-" {
            tfsdkToField[tag] = i
        }
    }

    // Extract values using field mappings
    for tfAttr, apiField := range mappings {
        fieldIdx, ok := tfsdkToField[tfAttr]
        if !ok { continue }
        apiValue := r.extractTerraformValue(ctx, v.Field(fieldIdx))
        if apiValue != nil {
            configMap[apiField] = apiValue
        }
    }
    return configMap, nil
}
```

**Key Features**:
- Uses `tfsdk` struct tags for field identification
- Filters out nil/unknown values
- Maps Terraform attribute names to API field names via `GetFieldMappings()`

#### configMapToModel (lines 544-581)

Updates a Terraform model struct from an API configuration map:

```go
func (r *BaseConnectorResource) configMapToModel(ctx context.Context, cfg map[string]any, model any) {
    mappings := r.config.GetFieldMappings()

    // Get reflect value (dereference if pointer)
    v := reflect.ValueOf(model)
    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }

    // Build tfsdk tag to field index mapping
    tfsdkToField := make(map[string]int)
    // ... same pattern as modelToConfigMap

    // Set values using field mappings + helper functions
    for tfAttr, apiField := range mappings {
        fieldIdx, ok := tfsdkToField[tfAttr]
        if !ok { continue }
        r.setTerraformValue(ctx, cfg, apiField, v.Field(fieldIdx))
    }
}
```

**Key Features**:
- Uses helper functions (`helper.GetTfCfgString`, etc.) for type conversion
- Handles all supported Terraform types: `types.String`, `types.Int64`, `types.Bool`, `types.Float64`, `types.List`
- Gracefully handles missing fields in API response

### Timeout Context in CRUD Methods

All CRUD methods (Create, Update, Delete) implement timeout handling using the Terraform Plugin Framework timeouts block:

| Method | Lines | Timeout Source | Default |
|--------|-------|----------------|---------|
| `Create()` | 126-141 | `helper.DefaultCreateTimeout` | Configurable |
| `Update()` | 325-340 | `helper.DefaultUpdateTimeout` | Configurable |
| `Delete()` | 434-449 | `helper.DefaultDeleteTimeout` | Configurable |

**Implementation Pattern** (from Create, lines 126-141):

```go
// Get timeout from config
var timeoutsValue timeouts.Value
resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("timeouts"), &timeoutsValue)...)

createTimeout, diags := timeoutsValue.Create(ctx, helper.DefaultCreateTimeout)
resp.Diagnostics.Append(diags...)

// Create context with timeout
ctx, cancel := context.WithTimeout(ctx, createTimeout)
defer cancel()
```

**Key Features**:
- Supports user-configurable timeouts via `timeouts {}` block in Terraform config
- Falls back to default values when not specified
- Uses `context.WithTimeout` for cancellation support
- Properly defers `cancel()` to prevent context leaks

### Import State Passthrough

**Location**: Lines 496-499

The base resource implements standard ID-based import using the Terraform Plugin Framework's passthrough helper:

```go
func (r *BaseConnectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

**Behavior**:
- Accepts resource ID as import argument
- Sets the `id` attribute in state
- Subsequent `Read()` operation fetches full resource state from API

**Usage Example**:
```bash
terraform import streamkap_source_postgresql.example <connector-id>
```

### Type Extraction and Setting

The base resource includes helper methods for extracting and setting Terraform type values:

#### extractTerraformValue (lines 583-628)

Extracts Go values from Terraform types:

| Terraform Type | Returns | Null/Unknown Handling |
|---------------|---------|----------------------|
| `types.String` | `string` | Returns `nil` |
| `types.Int64` | `int64` | Returns `nil` |
| `types.Bool` | `bool` | Returns `nil` |
| `types.Float64` | `float64` | Returns `nil` |
| `types.List` | `[]string` | Returns `nil` |

#### setTerraformValue (lines 630-666)

Sets Terraform type values from API response:

| Field Type | Helper Used |
|-----------|-------------|
| `types.String` | `helper.GetTfCfgString()` |
| `types.Int64` | `helper.GetTfCfgInt64()` |
| `types.Bool` | `helper.GetTfCfgBool()` |
| `types.Float64` | Inline handling |
| `types.List` | `helper.GetTfCfgListString()` |

### String Field Helpers

Two specialized helpers for accessing common string fields (ID, Name, Connector):

| Helper | Purpose | Lines |
|--------|---------|-------|
| `getStringField(model, fieldName)` | Get `types.String` value by field name | 669-686 |
| `setStringField(model, fieldName, value)` | Set `types.String` value by field name | 688-701 |

These are used for accessing standard fields like `ID`, `Name`, and `Connector` that exist on all connector models.

### CRUD Method Summary

| Method | Lines | API Operations | Key Features |
|--------|-------|----------------|--------------|
| `Create()` | 116-226 | `CreateSource` or `CreateDestination` | Timeout, modelToConfigMap, configMapToModel |
| `Read()` | 228-313 | `GetSource` or `GetDestination` | configMapToModel, nil check for deleted resources |
| `Update()` | 315-422 | `UpdateSource` or `UpdateDestination` | Timeout, modelToConfigMap, configMapToModel |
| `Delete()` | 424-494 | `DeleteSource` or `DeleteDestination` | Timeout, ID validation |
| `ImportState()` | 496-499 | None | Passthrough ID |

### Typecheck Verification

```bash
$ go build ./...
# Completed with no errors
```

---

## Schema Backward Compatibility Tests

This section documents the results of schema backward compatibility testing, which verifies that the refactored architecture maintains compatibility with existing Terraform configurations.

### Test Overview

**Command**: `go test -v -run 'TestSchemaBackwardsCompatibility' ./internal/provider/...`

**Results**: All 16 tests passed (0.862s)

### Test Results Summary

| Test Name | Baseline Attrs | Current Attrs | New Attrs | Status |
|-----------|---------------|---------------|-----------|--------|
| `TestSchemaBackwardsCompatibility_SourcePostgreSQL` | 48 | 48 | 0 | ✅ PASS |
| `TestSchemaBackwardsCompatibility_SourceMySQL` | 34 | 34 | 0 | ✅ PASS |
| `TestSchemaBackwardsCompatibility_SourceMongoDB` | 23 | 23 | 0 | ✅ PASS |
| `TestSchemaBackwardsCompatibility_SourceDynamoDB` | 19 | 19 | 0 | ✅ PASS |
| `TestSchemaBackwardsCompatibility_SourceSQLServer` | 30 | 30 | 1* | ✅ PASS |
| `TestSchemaBackwardsCompatibility_SourceKafkaDirect` | 7 | 7 | 0 | ✅ PASS |
| `TestSchemaBackwardsCompatibility_DestinationSnowflake` | 24 | 24 | 0 | ✅ PASS |
| `TestSchemaBackwardsCompatibility_DestinationClickHouse` | 15 | 15 | 0 | ✅ PASS |
| `TestSchemaBackwardsCompatibility_DestinationDatabricks` | 17 | 17 | 0 | ✅ PASS |
| `TestSchemaBackwardsCompatibility_DestinationPostgreSQL` | 23 | 23 | 0 | ✅ PASS |
| `TestSchemaBackwardsCompatibility_DestinationS3` | 12 | 12 | 0 | ✅ PASS |
| `TestSchemaBackwardsCompatibility_DestinationIceberg` | 14 | 14 | 0 | ✅ PASS |
| `TestSchemaBackwardsCompatibility_DestinationKafka` | 9 | 9 | 0 | ✅ PASS |
| `TestSchemaBackwardsCompatibility_TransformMapFilter` | 8 | 8 | 0 | ✅ PASS |
| `TestSchemaBackwardsCompatibility_TransformEnrich` | 6 | 6 | 0 | ✅ PASS |
| `TestSchemaBackwardsCompatibility_TransformSqlJoin` | 9 | 9 | 0 | ✅ PASS |

*Note: SQL Server source has 1 new attribute (`snapshot_custom_table_config`) which is additive and backward-compatible.

### Verification: No Breaking Changes Detected

**Verified**: ✅ No "required attribute removed" errors
**Verified**: ✅ No "optional changed to required" errors

The tests verify that:
1. All baseline attributes from v2.1.18 are present in the current schema
2. No required attributes have been removed (would break existing configs)
3. No optional attributes have been changed to required (would break existing configs)
4. New attributes are additive only (backward-compatible)

### Schema Compatibility Test Categories

| Category | Count | Description |
|----------|-------|-------------|
| Source Connectors | 6 | PostgreSQL, MySQL, MongoDB, DynamoDB, SQLServer, KafkaDirect |
| Destination Connectors | 7 | Snowflake, ClickHouse, Databricks, PostgreSQL, S3, Iceberg, Kafka |
| Transform Resources | 3 | MapFilter, Enrich, SqlJoin |
| **Total** | **16** | |

### Total Attribute Coverage

| Entity Type | Total Attributes Verified |
|-------------|--------------------------|
| Sources | 161 |
| Destinations | 114 |
| Transforms | 23 |
| **Total** | **298** |

### Typecheck Verification

```bash
$ go build ./...
# Completed with no errors
```

---

## Deprecated Fields

This section documents the deprecated field handling implementation, which maintains backward compatibility with existing Terraform configurations while guiding users toward new attribute names.

### Deprecations Registry

**Location**: `cmd/tfgen/deprecations.json`

The deprecations registry contains **10 deprecated field definitions** across 2 connectors:

| Connector | Entity Type | Count | Description |
|-----------|-------------|-------|-------------|
| PostgreSQL | sources | 9 | Static field transforms and topic pattern |
| Snowflake | destinations | 1 | Schema creation toggle |

### Deprecated Field Mappings

#### PostgreSQL Source (9 fields)

| Deprecated Attribute | New Attribute | Type |
|---------------------|---------------|------|
| `insert_static_key_field_1` | `transforms_insert_static_key1_static_field` | string |
| `insert_static_key_value_1` | `transforms_insert_static_key1_static_value` | string |
| `insert_static_value_field_1` | `transforms_insert_static_value1_static_field` | string |
| `insert_static_value_1` | `transforms_insert_static_value1_static_value` | string |
| `insert_static_key_field_2` | `transforms_insert_static_key2_static_field` | string |
| `insert_static_key_value_2` | `transforms_insert_static_key2_static_value` | string |
| `insert_static_value_field_2` | `transforms_insert_static_value2_static_field` | string |
| `insert_static_value_2` | `transforms_insert_static_value2_static_value` | string |
| `predicates_istopictoenrich_pattern` | `predicates_is_topic_to_enrich_pattern` | string |

#### Snowflake Destination (1 field)

| Deprecated Attribute | New Attribute | Type |
|---------------------|---------------|------|
| `auto_schema_creation` | `create_schema_auto` | bool |

### DeprecationMessage Verification

Verified that deprecated fields include proper `DeprecationMessage` in their schema definitions:

#### PostgreSQL Source (spot check)

**File**: `internal/resource/source/postgresql_generated.go`

| Field | Line | DeprecationMessage |
|-------|------|--------------------|
| `insert_static_key_field_1` | 41-46 | ✅ `"Use 'transforms_insert_static_key1_static_field' instead."` |
| `insert_static_key_value_1` | 47-51 | ✅ `"Use 'transforms_insert_static_key1_static_value' instead."` |

**Example Implementation** (lines 41-46):
```go
s.Attributes["insert_static_key_field_1"] = schema.StringAttribute{
    Optional:           true,
    Computed:           true,
    DeprecationMessage: "Use 'transforms_insert_static_key1_static_field' instead.",
    Description:        "DEPRECATED: Use 'transforms_insert_static_key1_static_field' instead.",
}
```

#### Snowflake Destination (spot check)

**File**: `internal/resource/destination/snowflake_generated.go`

| Field | Line | DeprecationMessage |
|-------|------|--------------------|
| `auto_schema_creation` | 33-37 | ✅ `"Use 'create_schema_auto' instead."` |

**Example Implementation** (lines 33-37):
```go
s.Attributes["auto_schema_creation"] = schema.BoolAttribute{
    Optional:           true,
    Computed:           true,
    DeprecationMessage: "Use 'create_schema_auto' instead.",
    Description:        "DEPRECATED: Use 'create_schema_auto' instead.",
}
```

### Implementation Architecture

Deprecated fields are handled at the **wrapper layer** (Layer 2 of the three-layer architecture):

1. **Field Mappings Extended**: Wrapper files extend generated field mappings with deprecated aliases
   ```go
   // From postgresql_generated.go
   var postgresqlFieldMappings = func() map[string]string {
       mappings := make(map[string]string)
       for k, v := range generated.SourcePostgresqlFieldMappings {
           mappings[k] = v
       }
       // Deprecated aliases - map to same API fields
       mappings["insert_static_key_field_1"] = "transforms.InsertStaticKey1.static.field"
       // ...
   }()
   ```

2. **Schema Attributes Added**: `GetSchema()` adds deprecated attributes to the generated schema
   ```go
   func (c *PostgreSQLConfig) GetSchema() schema.Schema {
       s := generated.SourcePostgresqlSchema()
       // Add deprecated aliases for backward compatibility
       s.Attributes["insert_static_key_field_1"] = schema.StringAttribute{...}
       return s
   }
   ```

3. **API Field Mapping**: Deprecated and new attributes map to the **same API field name**, ensuring both work correctly

### Key Design Characteristics

| Characteristic | Implementation |
|----------------|----------------|
| **Backward Compatibility** | Deprecated fields remain functional, mapped to same API fields |
| **User Guidance** | Clear `DeprecationMessage` guides users to new attribute names |
| **Schema Validation** | Both deprecated and new fields work in Terraform configs |
| **Description Clarity** | Description explicitly states "DEPRECATED: Use 'X' instead." |
| **Centralized Registry** | All deprecations tracked in `deprecations.json` for audit |

### Typecheck Verification

```bash
$ go build ./...
# Completed with no errors
```

---

## Migration Tests

This section documents the migration testing status, which verifies that resources can be migrated between provider versions without unexpected changes.

### Test Status

**Status**: ⚠️ SKIPPED - No credentials available

**Reason**: The environment variables `STREAMKAP_CLIENT_ID` and `STREAMKAP_SECRET` are not set. Migration tests require active Streamkap API credentials to:
1. Create resources using the baseline provider version (v2.1.18)
2. Apply the refactored provider to verify migration compatibility
3. Confirm no unexpected plan changes occur

### Required Environment Variables

| Variable | Status | Purpose |
|----------|--------|---------|
| `STREAMKAP_CLIENT_ID` | ❌ Not set | OAuth2 client ID for API authentication |
| `STREAMKAP_SECRET` | ❌ Not set | OAuth2 client secret for API authentication |
| `TF_ACC` | ❌ Not set | Flag to enable acceptance tests |

### Command (Skipped)

```bash
TF_ACC=1 go test -v -timeout 30m -run 'TestAcc.*Migration' ./internal/provider/... -count=1
```

### Workaround for Credential-less Verification

While migration tests could not be executed, the following alternative verifications provide confidence in migration compatibility:

1. **Schema Backward Compatibility Tests**: All 16 tests passed (see [Schema Backward Compatibility Tests](#schema-backward-compatibility-tests))
   - Verified all 298 attributes across 16 resources match baseline v2.1.18
   - No required attributes removed
   - No optional attributes changed to required

2. **Deprecated Field Handling**: All 10 deprecated fields verified with proper `DeprecationMessage` (see [Deprecated Fields](#deprecated-fields))
   - Both deprecated and new attribute names work correctly
   - Map to same API field names

3. **API Client Tests**: All 21 tests passed
   - Verified authentication flow
   - Verified error handling
   - Verified `created_from` tracking

### Recommendation

Migration tests should be run manually before production release using:

```bash
export TF_ACC=1
export STREAMKAP_CLIENT_ID="<your-client-id>"
export STREAMKAP_SECRET="<your-secret>"
go test -v -timeout 30m -run 'TestAcc.*Migration' ./internal/provider/... -count=1
```

### Typecheck Verification

```bash
$ go build ./...
# Completed with no errors
```

---

## Code Generator Parser

This section documents the code generator parser implementation, which reads backend `configuration.latest.json` files and extracts field metadata for Terraform schema generation.

### Overview

**Location**: `cmd/tfgen/parser.go` (429 lines)

The parser is the first stage of the code generation pipeline. It reads backend configuration files and transforms them into structured Go types that the generator can use to create Terraform schemas.

### ParseConnectorConfig Function

**Location**: Lines 100-113

The `ParseConnectorConfig()` function reads and parses `configuration.latest.json` files:

```go
func ParseConnectorConfig(filepath string) (*ConnectorConfig, error) {
    data, err := os.ReadFile(filepath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file %s: %w", filepath, err)
    }

    var config ConnectorConfig
    if err := json.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse config file %s: %w", filepath, err)
    }

    return &config, nil
}
```

**Verification**: ✅ Function reads `configuration.latest.json` format via `os.ReadFile()` and `json.Unmarshal()`

### User-Defined Filter Logic

**Location**: Lines 117-119, 368-376

The parser filters configuration entries to only expose user-editable fields in Terraform schemas:

#### IsUserDefined Method (lines 117-119)

```go
// IsUserDefined returns true if this config entry should become a Terraform attribute.
// Only fields with user_defined=true should be exposed in the Terraform schema.
func (e *ConfigEntry) IsUserDefined() bool {
    return e.UserDefined
}
```

#### UserDefinedEntries Method (lines 368-376)

```go
// UserDefinedEntries returns only the config entries that should become Terraform attributes.
// These are entries where user_defined=true.
func (c *ConnectorConfig) UserDefinedEntries() []ConfigEntry {
    var entries []ConfigEntry
    for _, entry := range c.Config {
        if entry.IsUserDefined() {
            entries = append(entries, entry)
        }
    }
    return entries
}
```

**Verification**: ✅ `user_defined: true` filter logic exists and is used to filter configuration entries

### Field Metadata Extraction

The parser extracts comprehensive field metadata from the backend configuration:

#### ConfigEntry Structure (lines 38-61)

```go
type ConfigEntry struct {
    Name                 string       `json:"name"`
    Description          string       `json:"description,omitempty"`
    UserDefined          bool         `json:"user_defined"`
    Required             *bool        `json:"required,omitempty"`
    OrderOfDisplay       *int         `json:"order_of_display,omitempty"`
    DisplayName          string       `json:"display_name,omitempty"`
    Value                ValueObject  `json:"value"`
    Tab                  string       `json:"tab,omitempty"`
    Encrypt              bool         `json:"encrypt,omitempty"`
    // ... additional fields
}
```

#### ValueObject Structure (lines 63-81)

```go
type ValueObject struct {
    Control       string   `json:"control,omitempty"`
    Type          string   `json:"type,omitempty"`          // "raw" or "dynamic"
    Default       any      `json:"default,omitempty"`       // Can be string, int, bool, etc.
    RawValues     []any    `json:"raw_values,omitempty"`    // Options for select controls
    Max           *float64 `json:"max,omitempty"`           // Slider: maximum value
    Min           *float64 `json:"min,omitempty"`           // Slider: minimum value
    Step          *float64 `json:"step,omitempty"`          // Slider: step increment
    // ... additional fields
}
```

### Metadata Accessor Methods

| Metadata Field | Accessor Method | Lines | Return Type |
|----------------|-----------------|-------|-------------|
| **control** | `e.Value.Control` | 65 | `string` |
| **type** | `e.Value.Type`, `IsDynamic()` | 66, 147-149 | `string` / `bool` |
| **default** | `GetDefault()`, `GetDefaultString()`, `GetDefaultInt64()`, `GetDefaultBool()` | 163-241 | `any` / `string` / `int64` / `bool` |
| **required** | `IsRequired()` | 128-133 | `bool` |
| **encrypt** | `IsSensitive()` | 123-125 | `bool` |
| **raw_values** | `GetRawValues()` | 321-340 | `[]string` |
| **min/max/step** | `GetSliderMin()`, `GetSliderMax()`, `GetSliderStep()` | 343-364 | `int64` |

**Verification**: ✅ Field metadata extraction implemented for control, type, default, required, and all other relevant fields

### TerraformType Mapping

**Location**: Lines 245-264

The parser maps backend control types to Terraform types:

```go
func (e *ConfigEntry) TerraformType() TerraformType {
    switch e.Value.Control {
    case "string", "password", "textarea", "json", "datetime":
        return TerraformTypeString
    case "number":
        return TerraformTypeInt64
    case "boolean", "toggle":
        return TerraformTypeBool
    case "one-select":
        return TerraformTypeString
    case "multi-select":
        return TerraformTypeList
    case "slider":
        return TerraformTypeInt64
    default:
        return TerraformTypeString
    }
}
```

| Backend Control | Terraform Type |
|-----------------|----------------|
| `string`, `password`, `textarea`, `json`, `datetime` | `types.String` |
| `number` | `types.Int64` |
| `boolean`, `toggle` | `types.Bool` |
| `one-select` | `types.String` |
| `multi-select` | `types.List[types.String]` |
| `slider` | `types.Int64` |

### Terraform Attribute Name Conversion

**Location**: Lines 271-286

The parser converts backend field names to Terraform-friendly attribute names:

```go
func (e *ConfigEntry) TerraformAttributeName() string {
    name := e.Name

    // Remove common suffixes that indicate user-facing fields
    name = strings.TrimSuffix(name, ".user.defined")
    name = strings.TrimSuffix(name, ".user.displayed")

    // Replace dots and hyphens with underscores
    name = strings.ReplaceAll(name, ".", "_")
    name = strings.ReplaceAll(name, "-", "_")

    // Convert camelCase to snake_case
    name = camelToSnake(name)

    return strings.ToLower(name)
}
```

**Example Transformations**:
- `database.hostname.user.defined` → `database_hostname`
- `ssl-mode` → `ssl_mode`
- `sslMode` → `ssl_mode`

### Condition Handling

**Location**: Lines 83-88, 152-154, 389-428

The parser supports conditional field visibility based on other field values:

```go
type Condition struct {
    Operator string `json:"operator"` // "EQ", "NE", "IN"
    Config   string `json:"config"`   // The config field to check
    Value    any    `json:"value"`    // The value to compare against
}

func (e *ConfigEntry) IsConditional() bool {
    return len(e.Conditions) > 0
}
```

### Key Design Characteristics

| Characteristic | Implementation |
|----------------|----------------|
| **JSON Unmarshaling** | Standard library `encoding/json` with struct tags |
| **Optional Fields** | Pointer types (`*bool`, `*int`) for optional JSON fields |
| **Type Flexibility** | `any` type for fields with multiple possible types (`default`, `value`) |
| **Suffix Handling** | Strips `.user.defined` and `.user.displayed` suffixes |
| **Case Conversion** | Custom `camelToSnake()` function for consistent naming |
| **Null Safety** | All accessor methods handle nil values gracefully |

### Typecheck Verification

```bash
$ go build ./...
# Completed with no errors
```

---

## Code Generator Type Mapping

This section documents the code generator's type mapping implementation in `cmd/tfgen/generator.go`, which converts backend control types to Terraform schema types.

### Overview

**Location**: `cmd/tfgen/generator.go` (993 lines)

The generator is the second stage of the code generation pipeline. It takes parsed configuration data from the parser and generates Go code with Terraform schemas, model structs, and field mappings.

### Type Mapping Architecture

The type mapping occurs in two primary locations:
1. **Parser** (`cmd/tfgen/parser.go`): `TerraformType()` method maps backend controls to Terraform type constants
2. **Generator** (`cmd/tfgen/generator.go`): `entryToFieldData()` method uses these types to generate schema attributes

### Backend Control to Terraform Type Mapping

**Location**: `parser.go` lines 245-264 (TerraformType method)

| Backend Control | Terraform Type Constant | Go Type |
|-----------------|-------------------------|---------|
| `string` | `TerraformTypeString` | `types.String` |
| `password` | `TerraformTypeString` | `types.String` (+ Sensitive) |
| `textarea` | `TerraformTypeString` | `types.String` |
| `json` | `TerraformTypeString` | `types.String` |
| `datetime` | `TerraformTypeString` | `types.String` |
| `number` | `TerraformTypeInt64` | `types.Int64` |
| `boolean` | `TerraformTypeBool` | `types.Bool` |
| `toggle` | `TerraformTypeBool` | `types.Bool` |
| `one-select` | `TerraformTypeString` | `types.String` (+ OneOf validator) |
| `multi-select` | `TerraformTypeList` | `types.List` |
| `slider` | `TerraformTypeInt64` | `types.Int64` (+ Between validator) |

**Verification**: ✅ `string` → `types.String` mapping confirmed (line 247-248)
**Verification**: ✅ `number` → `types.Int64` mapping confirmed (line 249-251)
**Verification**: ✅ `boolean`/`toggle` → `types.Bool` mapping confirmed (line 252-253)

### Terraform Type Constants

**Location**: `parser.go` lines 90-98

```go
type TerraformType string

const (
    TerraformTypeString TerraformType = "types.String"
    TerraformTypeInt64  TerraformType = "types.Int64"
    TerraformTypeBool   TerraformType = "types.Bool"
    TerraformTypeList   TerraformType = "types.List[types.String]"
)
```

### Password Control to Sensitive Attribute Mapping

**Location**: `parser.go` lines 121-125 (IsSensitive method)

```go
// IsSensitive returns true if this config entry should be marked as sensitive in Terraform.
// Fields with encrypt=true or control=password are considered sensitive.
func (e *ConfigEntry) IsSensitive() bool {
    return e.Encrypt || e.Value.Control == "password"
}
```

**Verification**: ✅ `password` control generates `Sensitive: true` via `IsSensitive()` check

### encrypt=true to Sensitive=true Mapping

**Location**: `generator.go` lines 644 (field initialization)

The generator sets the `Sensitive` field from the parser's `IsSensitive()` result:

```go
field := FieldData{
    GoFieldName:    goFieldName,
    GoType:         string(entry.TerraformType()),
    TfsdkTag:       tfAttrName,
    TfAttrName:     tfAttrName,
    Description:    entry.Description,
    Sensitive:      entry.IsSensitive(),  // ← Uses IsSensitive()
    APIFieldName:   entry.Name,
    RequiresReplace: entry.IsSetOnce(),
}
```

**Verification**: ✅ `encrypt=true` generates `Sensitive=true` via `IsSensitive()` which returns `e.Encrypt || e.Value.Control == "password"`

### Schema Attribute Type Assignment

**Location**: `generator.go` lines 663-678

The generator assigns the appropriate schema attribute type based on the Terraform type:

```go
// Determine schema attribute type (respecting port field override)
if forceInt64 {
    field.SchemaAttrType = "schema.Int64Attribute"
} else {
    switch entry.TerraformType() {
    case TerraformTypeString:
        field.SchemaAttrType = "schema.StringAttribute"
    case TerraformTypeInt64:
        field.SchemaAttrType = "schema.Int64Attribute"
    case TerraformTypeBool:
        field.SchemaAttrType = "schema.BoolAttribute"
    case TerraformTypeList:
        field.SchemaAttrType = "schema.ListAttribute"
        field.GoType = "types.List"
        field.IsListType = true // Flag for template to add ElementType
    }
}
```

### Generated Schema Template

**Location**: `generator.go` lines 853-992 (schemaTemplate constant)

The template generates schema attributes with the appropriate type, sensitivity, and modifiers:

```go
"{{ .TfAttrName }}": {{ .SchemaAttrType }}{
{{- if .Sensitive }}
    Sensitive:           true,
{{- end }}
{{- if .Description }}
    Description:         {{ printf "%q" .Description }},
    MarkdownDescription: {{ printf "%q" .MarkdownDescription }},
{{- end }}
    // ... validators, defaults, plan modifiers
},
```

### Type Mapping Test Coverage

The `IsSensitive()` function is covered by dedicated tests in `parser_test.go`:

**Test**: `TestIsSensitive` (lines 237-272)

| Test Case | encrypt | control | Expected |
|-----------|---------|---------|----------|
| encrypt=true | `true` | `string` | `true` |
| password control | `false` | `password` | `true` |
| neither | `false` | `string` | `false` |
| both | `true` | `password` | `true` |

### Port Field Override

**Location**: `generator.go` lines 625-628, 636

Port fields are special-cased to use `Int64` type for better UX, even when the backend stores them as strings:

```go
// isPortField returns true if the field name indicates it's a port field.
// Port fields should use Int64 type for better UX.
func isPortField(tfAttrName string) bool {
    return tfAttrName == "port" ||
        strings.HasSuffix(tfAttrName, "_port")
}
```

This is applied in `entryToFieldData()`:

```go
// Check if this is a port field that should be Int64
forceInt64 := isPortField(tfAttrName)
```

### Validator Generation

The generator also creates validators for specific control types:

#### OneOf Validator (one-select control)

**Location**: `generator.go` lines 772-779

```go
func (g *Generator) oneOfValidator(entry *ConfigEntry) string {
    values := entry.GetRawValues()
    quoted := make([]string, len(values))
    for i, v := range values {
        quoted[i] = fmt.Sprintf("%q", v)
    }
    return fmt.Sprintf("stringvalidator.OneOf(%s)", strings.Join(quoted, ", "))
}
```

#### Between Validator (slider control)

**Location**: `generator.go` lines 782-786

```go
func (g *Generator) rangeValidator(entry *ConfigEntry) string {
    min := entry.GetSliderMin()
    max := entry.GetSliderMax()
    return fmt.Sprintf("int64validator.Between(%d, %d)", min, max)
}
```

### Key Implementation Details

| Feature | Implementation Location | Description |
|---------|------------------------|-------------|
| Type mapping | `parser.go:245-264` | Maps backend controls to TerraformType constants |
| Sensitivity | `parser.go:121-125` | `encrypt=true` OR `control=password` → `Sensitive=true` |
| Schema type | `generator.go:663-678` | Assigns `schema.*Attribute` based on TerraformType |
| Port override | `generator.go:625-628` | Forces `Int64` for port fields |
| OneOf validator | `generator.go:720-736` | Generated for `one-select` controls |
| Range validator | `generator.go:739-742` | Generated for `slider` controls |

### Summary of Verified Mappings

| Requirement | Status | Evidence |
|-------------|--------|----------|
| `string` → `types.String` | ✅ Verified | `parser.go:247-248` |
| `password` → `types.String` (Sensitive) | ✅ Verified | `parser.go:247-248`, `parser.go:123-124` |
| `number` → `types.Int64` | ✅ Verified | `parser.go:249-251` |
| `boolean` → `types.Bool` | ✅ Verified | `parser.go:252-253` |
| `encrypt=true` → `Sensitive=true` | ✅ Verified | `parser.go:123`, `generator.go:644` |

### Typecheck Verification

```bash
$ go build ./...
# Completed with no errors
```

---

## Override and Deprecation System

This section documents the JSON-based configuration files that control field overrides and deprecations in the code generation system.

### Overview

The code generator uses two JSON configuration files to handle cases that cannot be automatically inferred from backend `configuration.latest.json` files:

| File | Purpose | Location |
|------|---------|----------|
| `overrides.json` | Defines complex field types (map_string, map_nested) | `cmd/tfgen/overrides.json` |
| `deprecations.json` | Defines deprecated field aliases | `cmd/tfgen/deprecations.json` |

### overrides.json Structure

**Location**: `cmd/tfgen/overrides.json`

The overrides file contains **3 field overrides** for complex types that the generator cannot automatically infer:

```json
{
  "field_overrides": [
    {
      "connector": "<connector_name>",
      "entity_type": "<sources|destinations>",
      "api_field_name": "<backend.field.name>",
      "terraform_attr_name": "<terraform_attribute>",
      "type": "<map_string|map_nested>",
      "optional": true,
      "description": "<field description>",
      "nested_model_name": "<ModelName>",  // For map_nested only
      "nested_fields": [...]               // For map_nested only
    }
  ]
}
```

### Override Types

#### map_string Override

Used for fields that map string keys to string values.

**Example**: Snowflake `auto_qa_dedupe_table_mapping`

| Property | Value |
|----------|-------|
| **connector** | `snowflake` |
| **entity_type** | `destinations` |
| **api_field_name** | `auto.qa.dedupe.table.mapping` |
| **terraform_attr_name** | `auto_qa_dedupe_table_mapping` |
| **type** | `map_string` |
| **optional** | `true` |

**Verification**: ✅ At least one `map_string` override exists (Snowflake destination)

#### map_nested Override

Used for fields that map string keys to nested object values.

**Example 1**: ClickHouse `topics_config_map`

| Property | Value |
|----------|-------|
| **connector** | `clickhouse` |
| **entity_type** | `destinations` |
| **api_field_name** | `topics.config.map` |
| **terraform_attr_name** | `topics_config_map` |
| **type** | `map_nested` |
| **nested_model_name** | `ClickHouseTopicsConfigMapItemModel` |
| **nested_fields** | `[{name: "delete_sql_execute", type: "string", optional: true}]` |

**Example 2**: SQL Server `snapshot_custom_table_config`

| Property | Value |
|----------|-------|
| **connector** | `sqlserveraws` |
| **entity_type** | `sources` |
| **api_field_name** | `streamkap.snapshot.custom.table.config.user.defined` |
| **terraform_attr_name** | `snapshot_custom_table_config` |
| **type** | `map_nested` |
| **nested_model_name** | `SnapshotCustomTableConfigModel` |
| **nested_fields** | `[{name: "chunks", type: "int64", required: true, validators: [{type: "int64_at_least", value: 1}]}]` |

**Verification**: ✅ At least one `map_nested` override exists (ClickHouse and SQL Server)

### Override Registry Summary

| Connector | Entity Type | Override Type | Field |
|-----------|-------------|---------------|-------|
| Snowflake | destinations | map_string | `auto_qa_dedupe_table_mapping` |
| ClickHouse | destinations | map_nested | `topics_config_map` |
| SQL Server | sources | map_nested | `snapshot_custom_table_config` |

### deprecations.json Structure

**Location**: `cmd/tfgen/deprecations.json`

The deprecations file contains **10 deprecated field definitions** for backward compatibility:

```json
{
  "deprecated_fields": [
    {
      "connector": "<connector_name>",
      "entity_type": "<sources|destinations>",
      "deprecated_attr": "<old_attribute_name>",
      "new_attr": "<new_attribute_name>",
      "type": "<string|bool|int64>"
    }
  ]
}
```

### Deprecation Registry Summary

#### PostgreSQL Source (9 deprecated fields)

| Deprecated Attribute | New Attribute | Type |
|---------------------|---------------|------|
| `insert_static_key_field_1` | `transforms_insert_static_key1_static_field` | string |
| `insert_static_key_value_1` | `transforms_insert_static_key1_static_value` | string |
| `insert_static_value_field_1` | `transforms_insert_static_value1_static_field` | string |
| `insert_static_value_1` | `transforms_insert_static_value1_static_value` | string |
| `insert_static_key_field_2` | `transforms_insert_static_key2_static_field` | string |
| `insert_static_key_value_2` | `transforms_insert_static_key2_static_value` | string |
| `insert_static_value_field_2` | `transforms_insert_static_value2_static_field` | string |
| `insert_static_value_2` | `transforms_insert_static_value2_static_value` | string |
| `predicates_istopictoenrich_pattern` | `predicates_is_topic_to_enrich_pattern` | string |

#### Snowflake Destination (1 deprecated field)

| Deprecated Attribute | New Attribute | Type |
|---------------------|---------------|------|
| `auto_schema_creation` | `create_schema_auto` | bool |

### How Overrides and Deprecations Are Applied

The JSON configuration files are processed during code generation:

1. **Generator Initialization**: `cmd/tfgen/generator.go` loads both JSON files at startup
2. **Override Application**: For each connector, overrides are merged with auto-generated fields
3. **Deprecation Application**: Deprecated attributes are added to wrapper files with `DeprecationMessage`
4. **Field Mapping**: Both deprecated and new attributes map to the same API field name

```
┌─────────────────────────┐     ┌──────────────────────────┐
│ configuration.latest.json │     │ overrides.json           │
│ (backend config)         │     │ (complex type overrides) │
└───────────┬─────────────┘     └───────────┬──────────────┘
            │                               │
            ▼                               ▼
┌─────────────────────────────────────────────────────────┐
│                   Code Generator (tfgen)                 │
│  - Parses backend config                                │
│  - Applies type overrides                               │
│  - Generates schemas, models, mappings                  │
└───────────┬─────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────┐
│                   Wrapper Files                          │
│  - Extend generated schemas with deprecated fields      │
│  - Add DeprecationMessage for old attribute names       │
│  - Map deprecated + new attrs to same API field         │
└─────────────────────────────────────────────────────────┘
            │
            │     ┌──────────────────────────┐
            └────►│ deprecations.json        │
                  │ (deprecated field aliases)│
                  └──────────────────────────┘
```

### Key Design Benefits

| Benefit | Description |
|---------|-------------|
| **Centralized Configuration** | All overrides and deprecations in two JSON files for easy audit |
| **Version Control Friendly** | JSON changes are easy to review in PRs |
| **Separation of Concerns** | Generator logic separate from configuration data |
| **Extensibility** | Adding new overrides/deprecations requires only JSON changes |
| **Type Safety** | Nested field validators in overrides support full validation rules |

### Typecheck Verification

```bash
$ go build ./...
# Completed with no errors
```

---

## Code Regeneration Test

This section documents the code regeneration stability verification.

### Test Status: SKIPPED

**Reason**: No backend path configured

The `STREAMKAP_BACKEND_PATH` environment variable is not set. This variable should point to the local clone of the Streamkap Python FastAPI backend repository, which contains the `configuration.latest.json` files used for code generation.

### Required Environment Variable

| Variable | Purpose | Current Status |
|----------|---------|----------------|
| `STREAMKAP_BACKEND_PATH` | Path to local backend repository | **Not set** |

### Command That Would Be Run

If the backend path was available, the following command would verify regeneration stability:

```bash
# With backend path set
export STREAMKAP_BACKEND_PATH=/path/to/python-be-streamkap

# Regenerate all schemas
go generate ./...

# Check for any differences in generated files
git diff --stat internal/generated/

# Expected: Empty diff (or whitespace-only changes)
```

### Alternative Verification

Without the backend path, regeneration stability can be inferred from:

1. **Schema Backward Compatibility Tests**: All 16 tests passed (see [Section 9](#schema-backward-compatibility-tests))
2. **Build Verification**: `go build ./...` completes without errors
3. **Generated Files Present**: 52 generated schema files exist with proper `DO NOT EDIT` markers

### Regeneration Stability Indicators

| Indicator | Status | Evidence |
|-----------|--------|----------|
| Generated files have DO NOT EDIT markers | ✅ Verified | US-003 verification |
| Generated file count matches expected | ✅ Verified | 52 files in `internal/generated/` |
| Build succeeds with generated files | ✅ Verified | `go build ./...` passes |
| Schema compatibility tests pass | ✅ Verified | 16/16 tests pass |

### Recommendation

For complete verification, run regeneration test with backend access before production release:

```bash
# Set backend path
export STREAMKAP_BACKEND_PATH=/path/to/python-be-streamkap

# Regenerate and verify
go generate ./... && git diff --stat internal/generated/

# If any differences, review them to ensure they are expected
```

### Typecheck Verification

```bash
$ go build ./...
# Completed with no errors
```

---
