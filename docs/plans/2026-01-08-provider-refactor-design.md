# Streamkap Terraform Provider: Architecture Refactor Design

**Date**: 2026-01-08
**Status**: Ready for Implementation
**Author**: Architecture Review

---

## Executive Summary

The Streamkap Terraform provider suffers from severe architectural issues: ~5,200 lines of duplicated CRUD code across 13 connector resources, no abstraction layer, and manual synchronization with backend APIs. Additionally, Transform and Tag are missing full CRUD resources (only read-only datasources exist). This document defines the complete refactor to a config-driven, generated provider that reduces maintenance burden by 70% and enables automatic synchronization with backend evolution.

**Scope:**
- **Tier 1 - Generated:** Sources (20), Destinations (23), Transforms (8) - all have `configuration.latest.json`
- **Tier 2 - New Resource:** Tag - backend has full CRUD, TF only has datasource
- **Tier 3 - Enhancement:** Topic - add Delete for unused topics
- **Out of Scope:** Pipeline (unique composition), Kafka Users, Consumer Groups (future consideration)

**Key Decisions:**
1. Build a custom generator that outputs **static typed Go code** (not runtime-interpreted data structures)
2. Generate per-connector: typed model structs, typed schema functions, and field mappings - all validated by Go compiler
3. Implement a single generic `BaseConnectorResource` with shared CRUD logic that uses generated typed code
4. Add `streamkap_tag` as a full resource (simple CRUD, handwritten)
5. Enhance `streamkap_topic` with proper Delete implementation for unused topics
6. Keep Pipeline unchanged (unique composition logic, already complete)
7. No dependency on HashiCorp's generator (doesn't generate CRUD) or Speakeasy (commercial)
8. Implement CI/CD pipeline for automated regeneration on backend changes

### Supporting Documentation

The following audit documents provide detailed reference material for implementation:

| Document | Purpose |
|----------|---------|
| [Entity Configuration Schema Audit](../audits/entity-config-schema-audit.md) | Complete schema reference for `configuration.latest.json` files - control types, value objects, conditional logic, Terraform mapping rules |
| [Backend Code Reference Guide](../audits/backend-code-reference.md) | Key backend code locations, API patterns, CRUD flows, multi-tenancy, debugging guide |

These documents should be consulted during code generator implementation (Section 3.3) and for debugging API integration issues.

---

## 1. Best Practices Summary

### 1.1 Modern Terraform Provider Architecture (2024-2025)

| Practice | Description | Source |
|----------|-------------|--------|
| **terraform-plugin-framework** | Modern SDK with better null handling, type safety, plan modifiers | HashiCorp |
| **OpenAPI as Single Source of Truth** | Schemas drive SDKs, Terraform, and docs simultaneously | Cloudflare |
| **Provider Code Specification** | Intermediate format enabling multiple input sources | HashiCorp |
| **Centralized Marshaling** | Unified conversion with struct tags vs per-resource code | Cloudflare |
| **CI Schema Validation** | Lint OpenAPI before generation to prevent cascading failures | Cloudflare |

### 1.2 Code Generation Tool Comparison

| Tool | Approach | Generates | Best For |
|------|----------|-----------|----------|
| **HashiCorp tfplugingen-openapi** | OpenAPI → Spec → Code | Schema, types | Schema generation baseline |
| **Speakeasy** | OpenAPI + annotations | Full CRUD provider | Commercial, full automation |
| **Custom Generator** | Configuration JSON → Go | Full provider | Domain-specific patterns |

### 1.3 Key Industry Insights

**From Cloudflare's Experience** ([Blog Post](https://blog.cloudflare.com/automatically-generating-cloudflares-terraform-provider/)):
- **Schema Linting Before Generation**: "Notice patterns of problems in the schema behaviors and apply CI lint rules within the schemas before it got into the code generation pipeline"
- **Unified Marshaling**: Decorate Terraform models with struct tags for consistent marshaling/unmarshaling - bug fixes propagate everywhere automatically
- **Incremental Bug Detection**: Test each generation stage independently to catch cascading failures
- **Migration Tooling**: Used [Grit](https://www.grit.io/) for automated Terraform configuration migrations
- Terraform's "known values" requirement differs fundamentally from SDK patterns

**From HashiCorp's Design** ([Documentation](https://developer.hashicorp.com/terraform/plugin/code-generation)):
- OpenAPI doesn't fully align with Terraform design principles
- Some manual configuration/mapping is always required
- Update/Delete operations don't currently affect schema mapping
- Provider code generation is designed to be extensible
- `terraform-plugin-codegen-framework` only generates schemas, NOT CRUD logic

**From Community Discussions** ([HashiCorp Forums](https://discuss.hashicorp.com/t/sdk-provider-development-anyone-ever-used-code-generation-or-other-tools-to-simplify-their-provider-development/20301)):
- "When developing a provider it feels like acting as a translator from an API to a terraform resource"
- Custom generators from config files outperform OpenAPI-only approaches for domain-specific APIs
- Go-Swagger supports custom templates for generating Terraform wrapper code

### 1.4 Available Automation Tools

| Tool | Use Case | Status | Relevance |
|------|----------|--------|-----------|
| [hashicorp/ghaction-terraform-provider-release](https://github.com/hashicorp/ghaction-terraform-provider-release) | Release automation with GoReleaser | Stable | **USE** |
| [hashicorp/terraform-provider-scaffolding](https://github.com/hashicorp/terraform-provider-scaffolding) | Template with .goreleaser.yml | Reference | **USE** |
| [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs) | Auto-generate docs from schema | Stable | **USE** |
| [tfplugingen-openapi](https://github.com/hashicorp/terraform-plugin-codegen-openapi) | OpenAPI → Provider spec | Tech Preview | Reference only |
| [dikhan/terraform-provider-openapi](https://github.com/dikhan/terraform-provider-openapi) | Runtime dynamic provider | Stable | Not applicable |
| [Speakeasy](https://www.speakeasy.com/docs/create-terraform) | Full generation from OpenAPI | Commercial | Not applicable |
| [Trivy](https://github.com/aquasecurity/trivy) | Security scanning | Stable | **ADD** |
| [Checkov](https://github.com/bridgecrewio/checkov) | Policy-as-code validation | Stable | **ADD** |

---

## 2. Current-State Audit

### 2.1 Codebase Statistics

| Component | Files | Lines | Avg Lines/File |
|-----------|-------|-------|----------------|
| Source Resources | 6 | 2,879 | 480 |
| Destination Resources | 7 | 2,870 | 410 |
| API Client | 8 | ~1,800 | 225 |
| Provider Core | 1 | 227 | - |
| Helpers | 1 | 56 | - |
| **Total Resource Code** | **13** | **~5,749** | **442** |

### 2.2 Architectural Smells Identified

#### Smell 1: Massive Code Duplication (Critical)
**Evidence:** Each connector resource repeats:
- Identical CRUD method structure (~100-150 lines each)
- Identical `Configure()` implementation
- Identical `ImportState()` implementation
- Common schema fields (id, name, connector) repeated 13 times
- Similar `model2ConfigMap()` / `configMap2Model()` patterns

**Impact:** Adding a field requires changes in multiple places. Bug fixes must be applied 13 times.

**Quantification:**
```
PostgreSQL:  589 lines (34 fields)
MySQL:       624 lines (27 fields)
SQLServer:   562 lines (25 fields)
Snowflake:   503 lines (17 fields)
ClickHouse:  466 lines (19 fields)
...
Estimated duplicated code: ~70% = ~4,000 lines
```

#### Smell 2: No Base Abstraction
**Evidence:** No `BaseConnectorResource` or similar abstraction exists. Every resource implements the full `resource.Resource` interface from scratch.

**Impact:** Impossible to make cross-cutting changes. No code reuse.

#### Smell 3: Manual Field Mapping
**Evidence:** Each resource has handwritten `model2ConfigMap()` that manually maps each field:
```go
// internal/resource/source/postgresql.go:478-540
configMap["database.hostname.user.defined"] = model.DatabaseHostname.ValueString()
configMap["database.port.user.defined"] = model.DatabasePort.ValueInt64()
// ... 30+ more manual mappings
```

**Impact:** Error-prone, no validation, no sync with backend schema.

#### Smell 4: Test Separation
**Evidence:** Tests live in `internal/provider/*_resource_test.go` while implementations are in `internal/resource/source/*.go` and `internal/resource/destination/*.go`.

**Impact:** Low cohesion, harder to maintain, tests don't live near the code they test.

#### Smell 5: Weak Type Conversion
**Evidence:** `helper.go` (56 lines) has silent failure modes:
```go
func GetTfCfgInt64(cfg map[string]any, key string) types.Int64 {
    // Returns null on any conversion failure - no error reporting
}
```

**Impact:** Configuration errors silently become null values.

#### Smell 6: No Schema Versioning
**Evidence:** No mechanism to handle schema evolution or deprecation.

**Impact:** Breaking changes require manual migration, no backward compatibility.

### 2.3 Current Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Terraform CLI                                 │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    provider.go (227 lines)                          │
│  - OAuth2 token exchange                                            │
│  - Registers 15 resources, 2 datasources                            │
└─────────────────────────────────────────────────────────────────────┘
                                  │
           ┌──────────────────────┼──────────────────────┐
           ▼                      ▼                      ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ postgresql.go   │    │ mysql.go        │    │ snowflake.go    │
│ 589 lines       │    │ 624 lines       │    │ 503 lines       │
│ - Schema()      │    │ - Schema()      │    │ - Schema()      │
│ - Create()      │    │ - Create()      │    │ - Create()      │
│ - Read()        │    │ - Read()        │    │ - Read()        │
│ - Update()      │    │ - Update()      │    │ - Update()      │
│ - Delete()      │    │ - Delete()      │    │ - Delete()      │
│ - model2Cfg()   │    │ - model2Cfg()   │    │ - model2Cfg()   │
│ - cfg2Model()   │    │ - cfg2Model()   │    │ - cfg2Model()   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
           │                      │                      │
           └──────────────────────┼──────────────────────┘
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      api/client.go                                   │
│  - StreamkapAPI interface                                            │
│  - doRequest() with auth header injection                            │
│  - CreateSource(), GetSource(), UpdateSource(), DeleteSource()       │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Streamkap Backend API                             │
│                  https://api.streamkap.com                           │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.4 Duplication Pattern Analysis

Extracted common patterns across all 13 connector resources:

| Pattern | Lines per Resource | Total Duplicated |
|---------|-------------------|------------------|
| Schema boilerplate (id, name, connector) | ~30 | ~390 |
| Configure() method | ~15 | ~195 |
| Create() method structure | ~60 | ~780 |
| Read() method structure | ~50 | ~650 |
| Update() method structure | ~55 | ~715 |
| Delete() method structure | ~25 | ~325 |
| ImportState() method | ~5 | ~65 |
| model2ConfigMap() pattern | ~80 | ~1,040 |
| configMap2Model() pattern | ~80 | ~1,040 |
| **Total Duplicated** | **~400** | **~5,200** |

### 2.5 Entity Type Coverage Analysis

Current state of all entities, backend capabilities, and TF implementation status:

| Entity Type | Backend CRUD | TF Status | Config JSON? | Action Required |
|-------------|--------------|-----------|--------------|-----------------|
| **Sources** | Full CRUD | 6 Resources | ✅ 20 plugins | **GENERATE** - Generic pattern |
| **Destinations** | Full CRUD | 7 Resources | ✅ 23 plugins | **GENERATE** - Generic pattern |
| **Transforms** | Full CRUD | ❌ DataSource only | ✅ 8 types | **GENERATE** - Has config.json! |
| **Tags** | Full CRUD | ❌ DataSource only | ❌ No | **ADD RESOURCE** - Simple CRUD |
| **Topics** | Read/Update/Delete | Partial Resource | ❌ No | **ENHANCE** - Add Delete |
| **Pipeline** | Full CRUD | ✅ Complete | ❌ No | NO CHANGE - Unique logic |
| **Kafka Users** | Full CRUD | ❌ Missing | ❌ No | FUTURE - Access control |
| **Consumer Groups** | Read + Reset | ❌ Missing | ❌ No | FUTURE - Offset management |

#### Transform Configuration Files

Transform entity has `configuration.latest.json` files for all 8 types:
- `app/transforms/plugins/map_filter/configuration.latest.json`
- `app/transforms/plugins/sql_join/configuration.latest.json`
- `app/transforms/plugins/enrich/configuration.latest.json`
- `app/transforms/plugins/enrich_async/configuration.latest.json`
- `app/transforms/plugins/rollup/configuration.latest.json`
- `app/transforms/plugins/un_nesting/configuration.latest.json`
- `app/transforms/plugins/fan_out/configuration.latest.json`
- `app/transforms/plugins/toast_handling/configuration.latest.json`

Backend has full Transform CRUD:
- `POST /transforms` - Create transform
- `GET /transforms/{id}` - Read transform
- `PUT /transforms` - Update transform
- `DELETE /transforms` - Delete transform
- `POST /transforms/{id}/clone` - Clone transform

TF currently only has read-only datasource.

#### Tag Backend CRUD

Backend Tag API supports:
- `POST /tags` - Create tag
- `PUT /tags/{id}` - Update tag
- `DELETE /tags/{id}` - Delete tag

TF currently only has read-only datasource.

Tag validation rules:
- `environment` and `general` tags cannot be updated/deleted
- Only ONE `environment` tag per tenant
- Tags in use by entities cannot be deleted
- Simple structure: name, type (list), description

#### Topic Delete

Backend has `DELETE /topics/{topic_id}` endpoint with constraints:
- Topics in use by pipelines CANNOT be deleted
- Unused topics CAN be deleted
- Removes from both MongoDB and Kafka

TF current implementation has empty `Delete()` method.

#### Why Pipeline Remains Out of Scope

Pipeline is a composition resource with unique logic:
- References Sources, Destinations, Transforms by ID
- Fetches Transform details via API for validation
- Has audit configuration with source/destination pair validation
- Has topic auto-discovery logic
- **No `configuration.latest.json`** - structure is code-defined

#### Coverage Gaps Summary

**Current TF:** 13 connectors + 1 pipeline + 1 topic (partial) + 2 datasources = 17 entities
**Backend Total:** 43 connectors + 8 transforms + tags + topic + pipeline = 53+ entities

**Missing from TF:**
- 30 connectors (14 sources + 16 destinations)
- 8 transform types (as full resources)
- Tag (as full resource)
- Topic Delete functionality
- Kafka Users (optional - access control)
- Consumer Groups (optional - offset management)

---

## 3. Target-State Architecture

### 3.1 Design Principles

1. **Single Implementation, N Connectors**: One generic connector resource driven by configuration
2. **Schema-Driven**: Connector schemas derived from backend `configuration.latest.json`
3. **Generated Where Possible**: Schema definitions, field mappings, validators generated from OpenAPI/config
4. **Handwritten Where Necessary**: Authentication, custom business logic, edge cases
5. **Backward Compatible**: Existing Terraform state must continue to work

### 3.2 Target Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                     CODE GENERATION PIPELINE                         │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐              │
│  │ OpenAPI     │    │ Connector   │    │ Generator   │              │
│  │ Spec        │───▶│ Config JSON │───▶│ Tool        │              │
│  │ (backend)   │    │ (backend)   │    │ (custom)    │              │
│  └─────────────┘    └─────────────┘    └─────────────┘              │
│                                              │                       │
│                      ┌───────────────────────┤                       │
│                      ▼                       ▼                       │
│            ┌─────────────────┐    ┌─────────────────┐               │
│            │ generated/      │    │ generated/      │               │
│            │ schemas.go      │    │ mappings.go     │               │
│            └─────────────────┘    └─────────────────┘               │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        RUNTIME PROVIDER                              │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    provider.go (enhanced)                    │    │
│  │  - OAuth2 token exchange                                     │    │
│  │  - Dynamic resource registration from registry               │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                  │                                   │
│                                  ▼                                   │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │              connector_resource.go (SINGLE FILE)             │    │
│  │                                                              │    │
│  │  type ConnectorResource struct {                             │    │
│  │      client         api.StreamkapAPI                         │    │
│  │      connectorType  string        // "source" or "destination"│   │
│  │      connectorCode  string        // "postgresql", "snowflake"│   │
│  │      schemaSpec     SchemaSpec    // generated schema config │    │
│  │      fieldMappings  FieldMappings // generated mappings      │    │
│  │  }                                                           │    │
│  │                                                              │    │
│  │  - Schema() → uses schemaSpec to build dynamic schema        │    │
│  │  - Create() → generic, uses fieldMappings                    │    │
│  │  - Read()   → generic, uses fieldMappings                    │    │
│  │  - Update() → generic, uses fieldMappings                    │    │
│  │  - Delete() → generic                                        │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                  │                                   │
│                                  ▼                                   │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    api/client.go (unchanged)                 │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.3 Core Components

#### 3.3.1 Static Typed Code Generation (Compile-Time Safety)

**Key Principle:** Generate actual Go code, not data structures interpreted at runtime. This ensures:
- Go compiler catches typos and type mismatches immediately
- IDE autocomplete works on generated code
- `golangci-lint` can analyze generated code for issues
- AI-generated mistakes are caught at `go build`, not at runtime

**What gets generated per connector:**

```go
// generated/source_postgresql.go - AUTO-GENERATED, DO NOT EDIT

package generated

import (
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// SourcePostgreSQLModel - typed model with compile-time field validation
type SourcePostgreSQLModel struct {
    ID               types.String `tfsdk:"id"`
    Name             types.String `tfsdk:"name"`
    Connector        types.String `tfsdk:"connector"`
    DatabaseHostname types.String `tfsdk:"database_hostname"`
    DatabasePort     types.Int64  `tfsdk:"database_port"`
    DatabaseUser     types.String `tfsdk:"database_user"`
    DatabasePassword types.String `tfsdk:"database_password"`
    DatabaseDbname   types.String `tfsdk:"database_dbname"`
    // ... all fields are explicit Go struct fields
}

// SourcePostgreSQLSchema - typed schema with compile-time attribute validation
func SourcePostgreSQLSchema() schema.Schema {
    return schema.Schema{
        MarkdownDescription: "PostgreSQL CDC source connector",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed:            true,
                MarkdownDescription: "Source identifier",
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "name": schema.StringAttribute{
                Required:            true,
                MarkdownDescription: "Source name",
            },
            "database_hostname": schema.StringAttribute{
                Required:            true,
                MarkdownDescription: "Database hostname",
            },
            "database_port": schema.Int64Attribute{
                Optional:            true,
                Computed:            true,
                MarkdownDescription: "Database port",
                Default:             int64default.StaticInt64(5432),
            },
            // ... all attributes are explicit Go code
        },
    }
}

// SourcePostgreSQLFieldMappings - maps TF field names to API field names
var SourcePostgreSQLFieldMappings = map[string]string{
    "database_hostname": "database.hostname.user.defined",
    "database_port":     "database.port.user.defined",
    "database_user":     "database.user.user.defined",
    "database_password": "database.password.user.defined",
    // ... explicit mappings
}
```

**Why this is better than data-driven schemas:**

| Aspect | Data-Driven (SchemaSpec) | Static Typed Code |
|--------|--------------------------|-------------------|
| Typo in field name | Runtime error | Compile error |
| Wrong attribute type | Runtime error | Compile error |
| Missing attribute | Runtime error | Compile error |
| IDE autocomplete | ❌ No | ✅ Yes |
| golangci-lint | ❌ Can't analyze | ✅ Full analysis |
| Debugging | Runtime stack traces | Line numbers in generated code |

#### 3.3.2 Unified Marshaling Pattern (Cloudflare-inspired)

Use struct tags for consistent marshaling/unmarshaling across all resources:

```go
// internal/generated/models.go - Generated model with tags

type PostgreSQLSourceModel struct {
    ID               types.String `tfsdk:"id" api:"-" computed:"true"`
    Name             types.String `tfsdk:"name" api:"name" required:"true"`
    DatabaseHostname types.String `tfsdk:"database_hostname" api:"database.hostname.user.defined" required:"true"`
    DatabasePort     types.Int64  `tfsdk:"database_port" api:"database.port.user.defined" default:"5432"`
    DatabaseUser     types.String `tfsdk:"database_user" api:"database.user.user.defined" required:"true"`
    DatabasePassword types.String `tfsdk:"database_password" api:"database.password.user.defined" sensitive:"true"`
    // ... all fields have consistent tags
}

// internal/resource/connector/marshal.go - Unified conversion

func ModelToAPIConfig(model interface{}) (map[string]any, error) {
    // Uses reflection on `api` tags to build config map
    // One implementation, all connectors
}

func APIConfigToModel(config map[string]any, model interface{}) error {
    // Uses reflection on `api` tags to populate model
    // Bug fixes here propagate to ALL connectors
}
```

**Benefits of this pattern:**
- Bug fix in marshaling logic fixes all connectors automatically
- Consistent handling of nulls, defaults, and type conversions
- Generated models are the single source of truth

#### 3.3.3 Generic Connector Resource (Handwritten CRUD, Generated Schema)

The generic resource uses **generated static typed code** for schemas and models, while providing shared CRUD logic:

```go
// internal/resource/connector/base.go - HANDWRITTEN (shared CRUD logic)

// ConnectorConfig interface - implemented by each generated connector
type ConnectorConfig interface {
    GetSchema() schema.Schema
    GetFieldMappings() map[string]string
    GetConnectorType() string  // "source" or "destination"
    GetConnectorCode() string  // "postgresql", "snowflake", etc.
    NewModel() any             // returns *SourcePostgreSQLModel, etc.
}

// BaseConnectorResource - generic CRUD implementation
type BaseConnectorResource struct {
    client api.StreamkapAPI
    config ConnectorConfig
}

func (r *BaseConnectorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    // Uses generated schema function - no runtime interpretation
    resp.Schema = r.config.GetSchema()
}

func (r *BaseConnectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    model := r.config.NewModel()
    resp.Diagnostics.Append(req.Plan.Get(ctx, model)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Convert model to API config using generated field mappings
    configMap := ModelToAPIConfig(model, r.config.GetFieldMappings())

    var result *api.ConnectorResponse
    var err error
    if r.config.GetConnectorType() == "source" {
        result, err = r.client.CreateSource(ctx, r.config.GetConnectorCode(), configMap)
    } else {
        result, err = r.client.CreateDestination(ctx, r.config.GetConnectorCode(), configMap)
    }

    if err != nil {
        resp.Diagnostics.AddError("Create failed", err.Error())
        return
    }

    // Update model with computed values from API response
    APIConfigToModel(result.Config, model, r.config.GetFieldMappings())
    resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Similar Read, Update, Delete use the same pattern
```

**Generated connector wiring:**

```go
// generated/source_postgresql_resource.go - AUTO-GENERATED

// Implements ConnectorConfig interface
type sourcePostgreSQLConfig struct{}

func (c *sourcePostgreSQLConfig) GetSchema() schema.Schema {
    return SourcePostgreSQLSchema()  // Generated typed schema function
}

func (c *sourcePostgreSQLConfig) GetFieldMappings() map[string]string {
    return SourcePostgreSQLFieldMappings  // Generated typed mapping
}

func (c *sourcePostgreSQLConfig) GetConnectorType() string { return "source" }
func (c *sourcePostgreSQLConfig) GetConnectorCode() string { return "postgresql" }

func (c *sourcePostgreSQLConfig) NewModel() any {
    return &SourcePostgreSQLModel{}  // Generated typed model
}

// Resource constructor
func NewSourcePostgreSQLResource() resource.Resource {
    return &BaseConnectorResource{config: &sourcePostgreSQLConfig{}}
}
```

**Key insight:** CRUD logic is handwritten once. Schema, model, and mappings are generated as static typed Go code. The compiler validates everything at build time.

#### 3.3.4 Provider Registration (Simplified)

```go
// internal/provider/provider.go

import "github.com/streamkap-com/terraform-provider-streamkap/internal/generated"

func (p *streamkapProvider) Resources(ctx context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        // Generated connectors - each is a typed constructor function
        generated.NewSourcePostgreSQLResource,
        generated.NewSourceMySQLResource,
        generated.NewSourceMongoDBResource,
        generated.NewSourceDynamoDBResource,
        generated.NewSourceSQLServerResource,
        generated.NewSourceKafkaDirectResource,
        // ... all 20 source connectors

        generated.NewDestinationSnowflakeResource,
        generated.NewDestinationClickHouseResource,
        generated.NewDestinationDatabricksResource,
        generated.NewDestinationPostgreSQLResource,
        generated.NewDestinationS3Resource,
        generated.NewDestinationIcebergResource,
        generated.NewDestinationKafkaResource,
        // ... all 23 destination connectors

        generated.NewTransformMapFilterResource,
        generated.NewTransformSQLJoinResource,
        // ... all 8 transform types

        // Non-connector resources (handwritten)
        pipeline.NewPipelineResource,
        topic.NewTopicResource,
        tag.NewTagResource,
    }
}
```

**Note:** Each `generated.New*Resource` is a typed function returning a concrete resource. The Go compiler verifies all function signatures at build time.

### 3.4 What Gets Generated vs Handwritten

| Component | Generated | Handwritten | Rationale |
|-----------|-----------|-------------|-----------|
| Model structs (typed fields) | ✅ | | Compile-time field validation |
| Schema functions (typed attributes) | ✅ | | Compile-time schema validation |
| Field mappings (TF ↔ API) | ✅ | | Mechanical transformation |
| Validators (enums, ranges) | ✅ | | Derived from backend schema |
| Defaults | ✅ | | Derived from backend schema |
| Descriptions | ✅ | | Derived from backend schema |
| ConnectorConfig implementations | ✅ | | Wires generated code to base resource |
| Resource constructors | ✅ | | Type-safe factory functions |
| BaseConnectorResource (CRUD logic) | | ✅ | Core framework, rarely changes |
| API client | | ✅ | Authentication, error handling |
| Provider configuration | | ✅ | OAuth2, environment variables |
| Complex resources (Pipeline) | | ✅ | Non-standard patterns |
| Tests | Partially | Partially | Generated structure, custom assertions |

---

## 4. Backend OpenAPI Requirements Checklist

For the code generation strategy to work, the backend OpenAPI specification and connector configuration files need specific metadata.

### 4.1 OpenAPI Specification Requirements

| Requirement | Current State | Action Needed |
|-------------|---------------|---------------|
| OpenAPI 3.0+ version | ✅ 3.1.0 | None |
| Source/Destination CRUD endpoints | ✅ Present | None |
| Request/Response schemas | ✅ Present | Enhance with details |
| Explicit `default` values | ⚠️ Partial | Add defaults to all optional fields |
| `nullable` annotations | ⚠️ Missing | Add for optional fields |
| `readOnly` for computed fields | ⚠️ Missing | Mark id, connector, status |
| `writeOnly` for sensitive fields | ⚠️ Missing | Mark passwords, keys |
| Enum constraints | ⚠️ Partial | Add for all constrained fields |
| Field descriptions | ✅ Present | Ensure completeness |

### 4.2 Connector Configuration JSON Requirements

The `configuration.latest.json` files are the richest source of metadata. Required enhancements:

| Requirement | Current State | Action Needed |
|-------------|---------------|---------------|
| `name` (API field name) | ✅ Present | None |
| `required` flag | ✅ Present | None |
| `user_defined` flag | ✅ Present | None |
| `description` | ✅ Present | None |
| `control` type | ✅ Present | None |
| `default` values | ⚠️ Partial | Add explicit defaults |
| `encrypt` for sensitive | ✅ Present | None |
| `raw_values` for enums | ✅ Present | None |
| **NEW: `terraform_name`** | ❌ Missing | Add snake_case TF field name |
| **NEW: `computed`** | ❌ Missing | Mark server-computed fields |
| **NEW: `immutable`** | ❌ Missing | Mark fields that can't change after create |
| **NEW: `deprecated`** | ❌ Missing | Mark deprecated fields with replacement |

### 4.3 Example Enhanced Configuration Entry

Current:
```json
{
  "name": "database.hostname.user.defined",
  "description": "PostgreSQL Hostname",
  "user_defined": true,
  "required": true,
  "value": {"control": "string"}
}
```

Enhanced for Terraform generation:
```json
{
  "name": "database.hostname.user.defined",
  "terraform_name": "database_hostname",
  "description": "PostgreSQL Hostname. Example: postgres.something.rds.amazonaws.com",
  "user_defined": true,
  "required": true,
  "computed": false,
  "immutable": false,
  "deprecated": null,
  "value": {
    "control": "string",
    "default": null
  }
}
```

### 4.4 Metadata Mapping Table

| Backend Metadata | Terraform Schema Property |
|------------------|---------------------------|
| `required: true` | `Required: true` |
| `required: false` + `default` | `Optional: true` + `Default` |
| `user_defined: false` | `Computed: true` |
| `encrypt: true` | `Sensitive: true` |
| `control: "password"` | `Sensitive: true` |
| `control: "one-select"` + `raw_values` | `Validators: stringvalidator.OneOf(...)` |
| `control: "slider"` + min/max | `Validators: int64validator.Between(...)` |
| `set_once: true` | `PlanModifiers: RequiresReplace()` |
| `conditions` | Complex validator or documentation |

---

## 5. Automation & Regeneration Strategy

### 5.1 Generator Tool Design

Create a custom Go-based generator tool that:

1. **Inputs:**
   - Backend `configuration.latest.json` files for each connector
   - OpenAPI specification for API structure
   - Generator configuration (overrides, customizations)

2. **Outputs:**
   - `internal/generated/schemas.go` - Schema specifications
   - `internal/generated/mappings.go` - Field mapping tables
   - `internal/generated/validators.go` - Validator definitions
   - `internal/generated/defaults.go` - Default value constants

3. **Process:**
```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ Backend Repo    │     │ Generator       │     │ Provider Repo   │
│                 │     │                 │     │                 │
│ plugins/        │────▶│ tfgen/          │────▶│ generated/      │
│   postgresql/   │     │   main.go       │     │   schemas.go    │
│     config.json │     │   parser.go     │     │   mappings.go   │
│   snowflake/    │     │   generator.go  │     │                 │
│     config.json │     │   templates/    │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

### 5.2 Generator Configuration

```yaml
# tfgen.yaml
version: 1

backend:
  repo: ../python-be-streamkap
  plugins_path: app/sources/plugins
  destinations_path: app/destinations/plugins

output:
  package: generated
  path: internal/generated

connectors:
  sources:
    - code: postgresql
      enabled: true
      overrides:
        database_hostname:
          description: "Custom description override"
    - code: mysql
      enabled: true
    # ... etc

  destinations:
    - code: snowflake
      enabled: true
    # ... etc

global_overrides:
  # Fields to always mark as sensitive
  sensitive_patterns:
    - "password"
    - "secret"
    - "private_key"

  # Fields to always mark as computed
  computed_fields:
    - "id"
    - "connector"
    - "connector_status"
```

### 5.3 CI/CD Integration

```yaml
# .github/workflows/regenerate.yml
name: Regenerate Provider Code

on:
  repository_dispatch:
    types: [backend-schema-updated]
  workflow_dispatch:
  schedule:
    - cron: '0 0 * * *'  # Daily check

jobs:
  regenerate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          path: provider

      - uses: actions/checkout@v4
        with:
          repository: streamkap-com/python-be-streamkap
          path: backend
          token: ${{ secrets.BACKEND_REPO_TOKEN }}

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Run Generator
        run: |
          cd provider
          go run ./cmd/tfgen --backend ../backend --output internal/generated

      - name: Run Tests
        run: |
          cd provider
          go test ./...

      - name: Check for Changes
        id: changes
        run: |
          cd provider
          git diff --quiet || echo "changes=true" >> $GITHUB_OUTPUT

      - name: Create PR
        if: steps.changes.outputs.changes == 'true'
        uses: peter-evans/create-pull-request@v5
        with:
          path: provider
          title: "chore: regenerate provider from backend schema"
          body: |
            Automated regeneration of provider code from backend configuration schemas.

            Please review the changes and ensure tests pass.
          branch: auto/regenerate-schemas
```

### 5.4 Regeneration Workflow

```
Developer updates backend connector config
                    │
                    ▼
        Backend CI runs, pushes event
                    │
                    ▼
    Provider CI triggered (repository_dispatch)
                    │
                    ▼
        Generator pulls latest configs
                    │
                    ▼
        Generator produces new code
                    │
                    ▼
            Tests run automatically
                    │
           ┌────────┴────────┐
           │                 │
        Pass              Fail
           │                 │
           ▼                 ▼
    Create PR          Alert team
           │           (Schema incompatibility)
           ▼
    Human review & merge
```

### 5.5 Backward Compatibility Strategy

1. **Schema Versioning**: Track schema versions in generated code
2. **Deprecation Workflow**:
   - Mark field deprecated in backend config
   - Generator adds `DeprecationMessage` to schema
   - After N versions, remove field
3. **State Migration**:
   - Use `UpgradeState` for breaking changes
   - Document migration path in release notes

### 5.6 Schema Validation (Cloudflare Pattern)

Before running the generator, validate backend config files to catch issues early:

```yaml
# .github/workflows/validate-schemas.yml
name: Validate Backend Schemas

on:
  pull_request:
    paths:
      - 'app/sources/plugins/**/configuration.latest.json'
      - 'app/destinations/plugins/**/configuration.latest.json'
      - 'app/transforms/plugins/**/configuration.latest.json'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Validate JSON Syntax
        run: |
          find app/*/plugins -name "configuration.latest.json" -exec jq . {} \;

      - name: Schema Lint Rules
        run: |
          # Check all required fields exist
          # Check terraform_name follows snake_case
          # Check no duplicate field names
          # Check sensitive fields have encrypt: true
          python scripts/lint-connector-schemas.py
```

**Lint Rules to Implement:**
| Rule | Check | Error Level |
|------|-------|-------------|
| `required-fields` | Every entry has `name`, `description`, `value` | Error |
| `terraform-name-format` | `terraform_name` is snake_case | Error |
| `sensitive-detection` | Password/secret/key fields have `encrypt: true` | Warning |
| `default-consistency` | Optional fields have explicit defaults | Warning |
| `enum-consistency` | `one-select` controls have `raw_values` | Error |

---

## 6. Release Automation

> **Note:** The provider already has `.goreleaser.yml` and `.github/workflows/release.yml` properly configured. This section documents the current setup and required updates.

### 6.1 GoReleaser Configuration (Already Exists ✅)

The existing `.goreleaser.yml` follows [HashiCorp's scaffolding](https://github.com/hashicorp/terraform-provider-scaffolding) pattern:

```yaml
# .goreleaser.yml
version: 2

before:
  hooks:
    - go mod tidy
    - go generate ./...

builds:
  - env:
      - CGO_ENABLED=0
    mod_timestamp: '{{ .CommitTimestamp }}'
    flags:
      - -trimpath
    ldflags:
      - '-s -w -X main.version={{.Version}} -X main.commit={{.Commit}}'
    goos:
      - freebsd
      - windows
      - linux
      - darwin
    goarch:
      - amd64
      - '386'
      - arm
      - arm64
    ignore:
      - goos: darwin
        goarch: '386'
    binary: '{{ .ProjectName }}_v{{ .Version }}'

archives:
  - format: zip
    name_template: '{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}'

checksum:
  name_template: '{{ .ProjectName }}_{{ .Version }}_SHA256SUMS'
  algorithm: sha256

signs:
  - artifacts: checksum
    args:
      - "--batch"
      - "--local-user"
      - "{{ .Env.GPG_FINGERPRINT }}"
      - "--output"
      - "${signature}"
      - "--detach-sign"
      - "${artifact}"

release:
  draft: true
```

### 6.2 Release Workflow (Exists - Needs Version Updates)

The workflow exists at `.github/workflows/release.yml`. **Update action versions:**

```yaml
# .github/workflows/release.yml - UPDATED VERSIONS
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6        # Updated from v4.1.0
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v6        # Updated from v4.1.0
        with:
          go-version-file: 'go.mod'
          cache: true

      - name: Import GPG key
        uses: crazy-max/ghaction-import-gpg@v6  # Updated from v6.0.0
        id: import_gpg
        with:
          gpg_private_key: ${{ secrets.GPG_PRIVATE_KEY }}
          passphrase: ${{ secrets.PASSPHRASE }}

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6   # Updated from v5.0.0
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}
```

### 6.3 Terraform Registry Publishing

After release, the provider is automatically published to the [Terraform Registry](https://registry.terraform.io/) if:
1. Repository is public on GitHub
2. Release is signed with GPG
3. Repository has the `terraform-provider-` prefix or is registered

**Registry Requirements:**
- GPG public key registered in Terraform Registry settings
- Semantic versioning (`v1.2.3` format)
- SHA256SUMS and signature files included in release

### 6.4 Documentation Generation

Use [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs) to auto-generate documentation:

```yaml
# Makefile
generate-docs:
	go generate ./...
	tfplugindocs generate --rendered-provider-name "Streamkap"
```

```go
// main.go
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
```

Documentation is generated from:
- Schema `MarkdownDescription` fields
- `examples/` directory for usage examples
- `templates/` for customization

---

## 7. Security & Quality Gates

### 7.1 CI Security Scanning (New - To Be Added)

```yaml
# .github/workflows/security.yml
name: Security Scan

on: [push, pull_request]

jobs:
  trivy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6

      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@0.33.1  # Latest as of 2025-01
        with:
          scan-type: 'fs'
          scan-ref: '.'
          exit-code: '1'
          ignore-unfixed: true
          severity: 'CRITICAL,HIGH'

  checkov:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6

      - name: Run Checkov
        uses: bridgecrewio/checkov-action@v12.1347.0  # Latest as of 2025-01
        with:
          directory: examples/
          framework: terraform
          quiet: true
          soft_fail: false
```

### 7.2 Quality Gates

| Gate | Tool | Threshold | When |
|------|------|-----------|------|
| Vulnerability scan | Trivy | No CRITICAL/HIGH | Every PR |
| IaC policy check | Checkov | Pass all policies | Every PR |
| Test coverage | go test -cover | >80% | Every PR |
| Linting | golangci-lint | No errors | Every PR |
| Generated code validity | go build | Compiles | After regeneration |
| Acceptance tests | TF_ACC=1 | All pass | Before release |

### 7.3 Pre-commit Hooks

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/tekwizely/pre-commit-golang
    rev: v1.0.0-rc.1
    hooks:
      - id: go-fmt
      - id: go-vet
      - id: go-lint
      - id: go-build

  - repo: local
    hooks:
      - id: tfplugindocs
        name: terraform plugin docs
        entry: go generate ./...
        language: system
        pass_filenames: false
```

---

## 8. Phased Refactor Plan

### Phase 0: Preparation (Week 1-2)

**Objective:** Set up infrastructure without changing existing code

| Task | Owner | Deliverable |
|------|-------|-------------|
| Create `cmd/tfgen/` generator skeleton | Provider team | Working CLI tool |
| Create `internal/generated/` package | Provider team | Empty package with doc.go |
| Document backend config JSON schema | Backend team | JSON Schema file |
| Add missing `terraform_name` to 3 connector configs | Backend team | Updated configs |
| Set up CI workflow (disabled) | DevOps | Workflow YAML |

**Exit Criteria:** Generator CLI exists, backend team aligned on schema changes

### Phase 1: Generator MVP (Week 3-4)

**Objective:** Generate schema specs for one connector (PostgreSQL source)

| Task | Owner | Deliverable |
|------|-------|-------------|
| Implement config JSON parser | Provider team | `parser.go` |
| Implement schema spec generator | Provider team | `generator.go` |
| Generate PostgreSQL source schema | Provider team | `schemas.go` with 1 entry |
| Write unit tests for generator | Provider team | 80% coverage |

**Exit Criteria:** `go run ./cmd/tfgen` produces valid `schemas.go` for PostgreSQL

### Phase 2: Generic Resource Framework (Week 5-7)

**Objective:** Implement single generic connector resource using generated schema

| Task | Owner | Deliverable |
|------|-------|-------------|
| Implement `ConnectorResource` struct | Provider team | `connector_resource.go` |
| Implement dynamic `Schema()` | Provider team | Schema from spec |
| Implement generic `Create()` | Provider team | Working create |
| Implement generic `Read()` | Provider team | Working read |
| Implement generic `Update()` | Provider team | Working update |
| Implement generic `Delete()` | Provider team | Working delete |
| Implement `extractConfigMap()` | Provider team | TF → API mapping |
| Implement `applyToState()` | Provider team | API → TF mapping |

**Exit Criteria:** PostgreSQL source works via generic resource, passes existing tests

### Phase 3: Migration - Sources (Week 8-9)

**Objective:** Migrate all source connectors to generic framework

| Task | Owner | Deliverable |
|------|-------|-------------|
| Add `terraform_name` to all source configs | Backend team | Updated configs |
| Generate schemas for all sources | Provider team | Complete `schemas.go` |
| Register all sources via generic resource | Provider team | Updated `provider.go` |
| Delete old source resource files | Provider team | Remove 6 files |
| Update/verify all source tests | Provider team | Green tests |

**Exit Criteria:** All 6 source resources use generic framework, tests pass

### Phase 4: Migration - Destinations (Week 10-11)

**Objective:** Migrate all destination connectors to generic framework

| Task | Owner | Deliverable |
|------|-------|-------------|
| Add `terraform_name` to all destination configs | Backend team | Updated configs |
| Generate schemas for all destinations | Provider team | Updated `schemas.go` |
| Register all destinations via generic resource | Provider team | Updated `provider.go` |
| Delete old destination resource files | Provider team | Remove 7 files |
| Update/verify all destination tests | Provider team | Green tests |

**Exit Criteria:** All 7 destination resources use generic framework, tests pass

### Phase 5: Automation & Polish (Week 12-13)

**Objective:** Enable full automation and clean up

| Task | Owner | Deliverable |
|------|-------|-------------|
| Enable CI regeneration workflow | DevOps | Active workflow |
| Add backend webhook trigger | Backend team | repository_dispatch |
| Improve error messages | Provider team | Better diagnostics |
| Update documentation | Provider team | Updated docs/ |
| Performance testing | QA | Benchmark results |
| Security review | Security | Sign-off |

**Exit Criteria:** Fully automated, documented, reviewed

### Phase 6: Future Enhancements (Ongoing)

| Enhancement | Priority | Description |
|-------------|----------|-------------|
| Test generation | Medium | Generate test scaffolds from config |
| Validation generation | Medium | Generate complex validators |
| Import generation | Low | Generate import examples |
| Diff optimization | Low | Smart diff for large configs |

---

## 9. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Backend config schema incompatible | Medium | High | Early prototype with real data |
| Generic resource doesn't handle edge cases | Medium | Medium | Keep escape hatches for custom logic |
| Performance regression | Low | Medium | Benchmark before/after |
| Breaking changes to existing users | Medium | High | **See Section 10 - Backward Compatibility** |
| Generator bugs cause silent failures | Medium | High | Comprehensive generator tests |

---

## 10. Backward Compatibility Strategy

**Critical Requirement:** Existing Terraform configurations and state files MUST continue to work without modification after the refactor. Breaking existing users is unacceptable.

### 10.1 What Must Be Preserved (Schema Contract)

For each existing resource, the following must be **exactly preserved**:

| Aspect | Example | Breaking if Changed |
|--------|---------|---------------------|
| Resource type name | `streamkap_source_postgresql` | Yes - state references break |
| Attribute names | `database_hostname`, `database_port` | Yes - configs and state break |
| Attribute types | `database_port` = Int64 | Yes - type mismatch errors |
| Required vs Optional | `name` = Required | Yes if Optional→Required |
| Computed flag | `id` = Computed | Yes - plan/apply behavior changes |
| Sensitive flag | `database_password` = Sensitive | Partial - state handling changes |
| Default values | `database_port` default 5432 | Yes - causes unexpected drift |
| Validators | `snapshot_read_only` OneOf values | Partial - may reject valid configs |

### 10.2 Golden File Schema Testing

**Before any migration**, capture current schemas as golden files:

```bash
# Generate golden schema files from current implementation
go run ./tools/schema-extractor \
  --resources=streamkap_source_postgresql,streamkap_source_mysql,... \
  --output=testdata/golden/schemas/
```

**Golden file format** (JSON):
```json
{
  "resource_type": "streamkap_source_postgresql",
  "attributes": {
    "id": {"type": "string", "computed": true, "required": false, "optional": false},
    "name": {"type": "string", "computed": false, "required": true, "optional": false},
    "database_hostname": {"type": "string", "computed": false, "required": true, "optional": false},
    "database_port": {"type": "int64", "computed": true, "required": false, "optional": true, "default": 5432},
    "database_password": {"type": "string", "sensitive": true, "required": true}
  }
}
```

**CI validation** compares generated schemas against golden files:
```go
func TestSchemaCompatibility(t *testing.T) {
    golden := loadGoldenSchema("streamkap_source_postgresql")
    generated := generated.SourcePostgreSQLSchema()

    for attrName, goldenAttr := range golden.Attributes {
        genAttr, exists := generated.Attributes[attrName]
        if !exists {
            t.Errorf("BREAKING: Attribute %q removed", attrName)
            continue
        }
        if goldenAttr.Type != getAttrType(genAttr) {
            t.Errorf("BREAKING: Attribute %q type changed: %s → %s",
                attrName, goldenAttr.Type, getAttrType(genAttr))
        }
        if goldenAttr.Required && !isRequired(genAttr) {
            t.Errorf("BREAKING: Attribute %q was required, now optional", attrName)
        }
        if !goldenAttr.Required && isRequired(genAttr) {
            t.Errorf("BREAKING: Attribute %q was optional, now required", attrName)
        }
        // ... more checks
    }

    // Check for new required attributes (would break existing configs)
    for attrName, genAttr := range generated.Attributes {
        if _, existed := golden.Attributes[attrName]; !existed && isRequired(genAttr) {
            t.Errorf("BREAKING: New required attribute %q added", attrName)
        }
    }
}
```

### 10.3 State Compatibility Testing

**Test procedure:** Apply generated provider to existing real state files (sanitized).

```go
func TestStateCompatibility(t *testing.T) {
    // Load real state file (with secrets redacted)
    stateFile := loadTestState("testdata/states/production-sanitized.tfstate")

    for _, resource := range stateFile.Resources {
        if !strings.HasPrefix(resource.Type, "streamkap_") {
            continue
        }

        // Attempt to read resource with generated provider
        ctx := context.Background()
        resp := &resource.ReadResponse{}

        // This should NOT error if schemas are compatible
        generatedResource := getResource(resource.Type)
        generatedResource.Read(ctx, resource.ReadRequest{State: resource.State}, resp)

        if resp.Diagnostics.HasError() {
            t.Errorf("State incompatibility for %s: %v", resource.Type, resp.Diagnostics)
        }
    }
}
```

### 10.4 Acceptance Test Reuse

**All existing acceptance tests must pass with generated resources:**

```go
// internal/provider/source_postgresql_resource_test.go (EXISTING - unchanged)
func TestAccSourcePostgreSQL_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccSourcePostgreSQLConfig_basic,
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttrSet("streamkap_source_postgresql.test", "id"),
                    resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "name", "test-pg-source"),
                    // ... existing checks
                ),
            },
        },
    })
}
```

**Migration validation:** Same test configs must work with both old and new implementations.

### 10.5 Generator Attribute Name Mapping

The generator must map backend config field names to **existing TF attribute names**:

```go
// tools/generator/mappings.go

// ExistingAttributeMappings ensures generated attributes match current provider
var ExistingAttributeMappings = map[string]map[string]string{
    "source_postgresql": {
        // Backend config key → Existing TF attribute name
        "database.hostname.user.defined": "database_hostname",
        "database.port.user.defined":     "database_port",
        "database.user.user.defined":     "database_user",
        "database.password.user.defined": "database_password",
        "database.dbname.user.defined":   "database_dbname",
        // ... all fields must be mapped
    },
    // ... other connectors
}

// ValidateNoNewRequiredAttributes ensures we don't add new required fields
func ValidateNoNewRequiredAttributes(connectorCode string, generatedSchema SchemaSpec) error {
    golden := loadGoldenSchema(connectorCode)
    for _, attr := range generatedSchema.Attributes {
        if attr.Required {
            if _, existed := golden.Attributes[attr.TFName]; !existed {
                return fmt.Errorf("BREAKING: new required attribute %q would break existing configs", attr.TFName)
            }
        }
    }
    return nil
}
```

### 10.6 CI Breaking Change Detection

Add CI job that **blocks merges** if breaking changes detected:

```yaml
# .github/workflows/compatibility.yml
name: Schema Compatibility Check

on:
  pull_request:
    paths:
      - 'internal/generated/**'
      - 'tools/generator/**'

jobs:
  compatibility:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run schema compatibility tests
        run: go test ./internal/provider -run TestSchemaCompatibility -v

      - name: Run state compatibility tests
        run: go test ./internal/provider -run TestStateCompatibility -v

      - name: Verify existing acceptance tests pass
        run: make testacc
        env:
          TF_ACC: 1
          # ... credentials
```

### 10.7 Version Strategy

Follow [Semantic Versioning](https://semver.org/) strictly:

| Change Type | Version Bump | Example |
|-------------|--------------|---------|
| Bug fix, no schema change | PATCH (0.0.X) | v1.2.3 → v1.2.4 |
| New optional attribute | MINOR (0.X.0) | v1.2.3 → v1.3.0 |
| New resource type | MINOR (0.X.0) | v1.2.3 → v1.3.0 |
| **Remove attribute** | MAJOR (X.0.0) | v1.2.3 → v2.0.0 |
| **Change attribute type** | MAJOR (X.0.0) | v1.2.3 → v2.0.0 |
| **Make optional→required** | MAJOR (X.0.0) | v1.2.3 → v2.0.0 |
| **Rename attribute** | MAJOR (X.0.0) | v1.2.3 → v2.0.0 |

**Goal:** Refactor should be released as a MINOR version (e.g., v1.5.0) with zero breaking changes.

### 10.8 Migration Validation Checklist

Before releasing refactored provider:

- [ ] All golden file schema tests pass
- [ ] All state compatibility tests pass with production-like state files
- [ ] All existing acceptance tests pass unchanged
- [ ] No new required attributes added
- [ ] No attribute names changed
- [ ] No attribute types changed
- [ ] Default values match exactly
- [ ] Sensitive flags match exactly
- [ ] Resource type names unchanged
- [ ] `terraform plan` shows no changes on existing infrastructure
- [ ] Documentation clearly states "internal refactor, no user action required"

### 10.9 Rollback Plan

If issues discovered post-release:

1. **Immediate:** Publish patch release reverting to old implementation
2. **Communication:** Notify users via GitHub release notes, Terraform Registry
3. **Analysis:** Identify compatibility gap in testing
4. **Fix Forward:** Add test coverage for missed case, then re-release

### 10.10 Deprecation Strategy (Last Resort)

**Important Clarification:** Backward compatibility is ONLY about customer-facing Terraform schemas. Internally, we can (and will) completely replace the implementation:

| Aspect | Backward Compatible? | Notes |
|--------|---------------------|-------|
| Resource type names (`streamkap_source_postgresql`) | ✅ Yes | Customers reference these |
| Attribute names (`database_hostname`) | ✅ Yes | Customers' `.tf` files use these |
| Attribute types, defaults, required/optional | ✅ Yes | Affects plan/apply behavior |
| Internal Go code structure | ❌ No - delete everything | Implementation detail |
| File organization | ❌ No - reorganize freely | Implementation detail |
| Internal function names | ❌ No - rename freely | Implementation detail |
| Old patterns/abstractions | ❌ No - remove completely | Implementation detail |

**When v2 resources are needed:** Only if the **customer-facing schema** is fundamentally broken and cannot be fixed without changing attribute names/types.

**Pattern:** Keep old schema available, add new "v2" resource with improved schema:

```hcl
# Old resource (deprecated schema, for existing customers)
resource "streamkap_source_postgresql" "existing" {
  # Same attribute names as before - customers don't change anything
}

# New resource (improved schema, for new customers)
resource "streamkap_source_postgresql_v2" "new" {
  # Better attribute names, types, structure
}
```

**Implementation:** Both resources use the NEW internal code - only the schema interface differs:

```go
// Both use BaseConnectorResource - same internal implementation
// Only the schema/model mapping differs

func NewSourcePostgreSQLResource() resource.Resource {
    return &BaseConnectorResource{
        config: &sourcePostgreSQLV1Config{}, // Old schema, maps to new internals
    }
}

func NewSourcePostgreSQLV2Resource() resource.Resource {
    return &BaseConnectorResource{
        config: &sourcePostgreSQLV2Config{}, // New schema
    }
}
```

**Migration path for users:**

```hcl
# Step 1: Import existing infrastructure to new resource
terraform import streamkap_source_postgresql_v2.new <id>

# Step 2: Remove old resource from state (don't destroy!)
terraform state rm streamkap_source_postgresql.existing

# Step 3: Update configuration to use new resource
```

**Timeline:**
- v1.x: Deprecation warning added to old schema
- v2.0: Both schemas available
- v3.0: Old schema removed (MAJOR version bump)

**Expected usage:** Hopefully zero resources need this. The goal is 100% schema compatibility so customers don't notice the refactor.

---

## 11. Appendix: File Inventory

### 11.1 Files to Delete (After Migration)

```
internal/resource/source/postgresql.go (589 lines)
internal/resource/source/mysql.go (624 lines)
internal/resource/source/mongodb.go (435 lines)
internal/resource/source/dynamodb.go (384 lines)
internal/resource/source/sqlserver.go (562 lines)
internal/resource/source/kafkadirect.go (285 lines)
internal/resource/destination/snowflake.go (503 lines)
internal/resource/destination/clickhouse.go (466 lines)
internal/resource/destination/databricks.go (375 lines)
internal/resource/destination/postgresql.go (419 lines)
internal/resource/destination/s3.go (397 lines)
internal/resource/destination/iceberg.go (374 lines)
internal/resource/destination/kafka.go (336 lines)

Total: 13 files, ~5,749 lines
```

### Files to Create

```
# Generator Tool
cmd/tfgen/main.go (~100 lines)
cmd/tfgen/parser.go (~300 lines)
cmd/tfgen/generator.go (~400 lines)
cmd/tfgen/templates/ (Go templates)

# Generated Code
internal/generated/schemas.go (generated, ~2000 lines)
internal/generated/models.go (generated, ~1500 lines) - Struct models with tags
internal/generated/mappings.go (generated, ~500 lines)

# Generic Resource Framework
internal/resource/connector/connector_resource.go (~400 lines)
internal/resource/connector/schema_builder.go (~200 lines)
internal/resource/connector/marshal.go (~150 lines) - Unified marshaling

# New Resources
internal/resource/tag/tag_resource.go (~200 lines) - Handwritten
internal/resource/transform/transform_resource.go (~400 lines) - Generic

# CI/CD Workflows
.github/workflows/regenerate.yml (~80 lines)
.github/workflows/release.yml (~50 lines)
.github/workflows/security.yml (~40 lines)
.github/workflows/validate-schemas.yml (~30 lines) - Backend repo

# Configuration
tfgen.yaml (~100 lines)
.goreleaser.yml (~60 lines)
.pre-commit-config.yaml (~30 lines)

Total: ~18 files, ~6,540 lines (including ~4,000 generated)
```

### Net Change

- **Deleted:** ~5,749 lines of handwritten, duplicated code
- **Added:** ~2,540 lines of handwritten framework + generator + CI
- **Added:** ~4,000 lines of generated code
- **Net handwritten reduction:** ~3,200 lines (56% reduction)
- **Automation gain:** Full CI/CD, security scanning, schema validation, auto-releases

---

## 12. Key Decisions

These decisions are based on comprehensive analysis of the backend codebase, OpenAPI spec, current provider architecture, and industry best practices.

### 12.1 Code Generation Strategy

**Decision:** Build a custom Go-based generator that reads backend `configuration.latest.json` files for Sources, Destinations, AND Transforms.

**Rationale:**
- Backend config JSON has richer metadata than OpenAPI (user_defined, required, control types, conditions)
- All three entity types (Sources: 20, Destinations: 23, Transforms: 8) have config.json files
- HashiCorp's `tfplugingen-openapi` only generates schemas, not CRUD logic
- Speakeasy requires commercial licensing and doesn't leverage our existing config files
- Custom generator gives us full control and integrates with existing backend structure

### 12.2 Resource Architecture

**Decision:** Implement generic resource structs for connector-like entities:
- `ConnectorResource` - handles all 43 source/destination types
- `TransformResource` - handles all 8 transform types

**Rationale:**
- Connector and Transform differences are DATA (field names, types, validators), not LOGIC
- CRUD operations are identical within each category
- Single implementation per category means bug fixes apply everywhere automatically
- Reduces ~5,200+ lines of duplicated code to ~600 lines of framework

### 12.3 Refactor Scope

**Decision:** Complete scope includes:

| Entity | Action | Approach |
|--------|--------|----------|
| Sources (20) | Generate | Generic ConnectorResource |
| Destinations (23) | Generate | Generic ConnectorResource |
| Transforms (8) | Generate | Generic TransformResource |
| Tag | Add Resource | Handwritten (simple CRUD) |
| Topic | Enhance | Add Delete for unused topics |
| Pipeline | No Change | Already complete, unique logic |

**Rationale:**
- Transforms have `configuration.latest.json` files - can be generated like connectors
- Tag has full backend CRUD but only TF datasource - should be full resource
- Topic has backend Delete endpoint but TF ignores it - should implement
- Pipeline is unique composition logic - no generation possible

### 12.4 Schema Source of Truth

**Decision:** Use backend `configuration.latest.json` as the primary schema source, not OpenAPI.

**Rationale:**
- Config JSON contains Terraform-specific metadata: user_defined, required, control types, conditions
- OpenAPI is generic and lacks this richness
- Config JSON already exists for all 51 generatable entities (20+23+8)
- Backend team maintains these files, ensuring automatic sync

### 12.5 Test Location

**Decision:** Keep tests in `internal/provider/*_resource_test.go` (current location).

**Rationale:**
- Minimizes disruption during migration
- Tests still work with new architecture
- Colocating tests can be done as a separate cleanup later

### 12.6 Backend Schema Requirements

**Decision:** Backend team will add `terraform_name` field to all config entries (connectors AND transforms).

**Rationale:**
- Generator needs explicit snake_case field names for Terraform
- Current API names use dot.notation (`database.hostname.user.defined`, `transforms.input.topic.pattern`)
- Adding `terraform_name` is minimal effort vs. implementing name conversion logic

### 12.7 API Client Completeness

**Decision:** Add missing API client methods:
- `CreateTag`, `UpdateTag`, `DeleteTag` - for new Tag resource
- `DeleteTopic` - for Topic enhancement
- `CreateTransform`, `UpdateTransform`, `DeleteTransform` - for new Transform resources

**Rationale:**
- Current API client has gaps that prevent full Terraform coverage
- Backend APIs exist, just not exposed in TF client

### 12.8 Release Automation

**Decision:** Implement full release automation using:
- GoReleaser for multi-platform builds and signing
- GitHub Actions for CI/CD
- GPG signing for Terraform Registry compatibility
- terraform-plugin-docs for documentation generation

**Rationale:**
- Industry standard for Terraform providers ([HashiCorp scaffolding](https://github.com/hashicorp/terraform-provider-scaffolding))
- Enables automatic registry publishing on tag push
- Reduces manual release burden

### 12.9 Schema Validation Pipeline

**Decision:** Implement Cloudflare-style schema linting in backend CI:
- Validate JSON syntax
- Check required fields present
- Verify `terraform_name` format
- Detect sensitive field annotation gaps

**Rationale:**
- [Cloudflare's lesson](https://blog.cloudflare.com/lessons-from-building-an-automated-sdk-pipeline/): "Notice patterns of problems and apply CI lint rules within the schemas before it got into the code generation pipeline"
- Catches issues before they cascade into generated code
- Backend team gets immediate feedback on schema quality

### 12.10 Unified Marshaling

**Decision:** Generate struct models with unified marshaling tags (`tfsdk`, `api`, `sensitive`, etc.) and implement reflection-based conversion.

**Rationale:**
- [Cloudflare's experience](https://blog.cloudflare.com/automatically-generating-cloudflares-terraform-provider/): "Once you identify a bug with a particular type of field, fixing that in the unified interface fixes it for other occurrences"
- Single implementation for all connector conversions
- Bug fixes automatically propagate to all resources

---

## 13. Implementation Tasks

### Phase 0: Preparation & Quick Wins

**Objective:** Set up infrastructure + implement low-hanging fruit

**CI/CD & Tooling Setup (Existing - Update Only):**
- [x] ~~Add `.goreleaser.yml`~~ → Already exists, properly configured
- [x] ~~Add `.github/workflows/release.yml`~~ → Already exists
- [x] ~~Set up GPG key for signing~~ → Already configured with secrets
- [ ] **UPDATE** `.github/workflows/release.yml` action versions:
  - `actions/checkout@v6` (currently v4.1.0)
  - `actions/setup-go@v6` (currently v4.1.0)
  - `crazy-max/ghaction-import-gpg@v6` (currently v6.0.0)
  - `goreleaser/goreleaser-action@v6` (currently v5.0.0)

**CI/CD & Tooling Setup (New):**
- [ ] Add `.github/workflows/security.yml` with Trivy + Checkov
- [ ] Add `.pre-commit-config.yaml` for local dev quality
- [ ] Add `go generate` directive for terraform-plugin-docs

**Generator Infrastructure:**
- [ ] Create `cmd/tfgen/` generator CLI skeleton
- [ ] Create `internal/generated/` package with doc.go
- [ ] Create disabled CI workflow in `.github/workflows/regenerate.yml`

**Quick Wins (Immediate Value):**
- [ ] **Quick Win:** Add `DeleteTopic` to API client (`internal/api/topic.go`)
- [ ] **Quick Win:** Implement Topic Delete in resource (call API only for unused topics)
- [ ] **Quick Win:** Add `CreateTag`, `UpdateTag`, `DeleteTag` to API client
- [ ] **Quick Win:** Create `streamkap_tag` resource (handwritten, ~200 lines)
- [ ] Add tests for Tag resource and Topic Delete

**Backend Team Tasks:**
- [ ] Add schema linter script to backend repo
- [ ] Create `.github/workflows/validate-schemas.yml` in backend repo

### Phase 1: Generator MVP (Connectors)

**Objective:** Generate schema specs for one connector (PostgreSQL source)

- [ ] Backend: Add `terraform_name` to PostgreSQL source config (pilot)
- [ ] Implement config JSON parser (`cmd/tfgen/parser.go`)
- [ ] Implement schema spec generator (`cmd/tfgen/generator.go`)
- [ ] Generate PostgreSQL source schema to `internal/generated/schemas.go`
- [ ] Write generator unit tests (80% coverage)

### Phase 2: Generic ConnectorResource Framework

**Objective:** Implement single generic connector resource using generated schema

**Framework Implementation:**
- [ ] Implement `ConnectorResource` struct (`internal/resource/connector/`)
- [ ] Implement dynamic `Schema()` method from SchemaSpec
- [ ] Implement generic `Create()`, `Read()`, `Update()`, `Delete()`

**Unified Marshaling (Cloudflare Pattern):**
- [ ] Implement `marshal.go` with reflection-based `ModelToAPIConfig()`
- [ ] Implement `APIConfigToModel()` using struct tags
- [ ] Add comprehensive tests for type conversions (null handling, defaults)
- [ ] Document struct tag format: `tfsdk:"name" api:"api.path" sensitive:"true"`

**Validation:**
- [ ] Verify PostgreSQL source passes existing acceptance tests
- [ ] Verify no state drift after migration

### Phase 3: Source Migration

**Objective:** Migrate all source connectors to generic framework

- [ ] Backend: Add `terraform_name` to all source configs (20 total)
- [ ] Generate schemas for all 20 sources
- [ ] Register all sources via generic resource in `provider.go`
- [ ] Delete 6 old source resource files
- [ ] Verify all source tests pass
- [ ] Add 14 missing sources (alloydb, db2, documentdb, etc.)

### Phase 4: Destination Migration

**Objective:** Migrate all destination connectors to generic framework

- [ ] Backend: Add `terraform_name` to all destination configs (23 total)
- [ ] Generate schemas for all 23 destinations
- [ ] Register all destinations via generic resource
- [ ] Delete 7 old destination resource files
- [ ] Verify all destination tests pass
- [ ] Add 16 missing destinations (bigquery, redshift, etc.)

### Phase 5: Transform Resources

**Objective:** Add full CRUD Transform resources using generation

- [ ] Add `CreateTransform`, `UpdateTransform`, `DeleteTransform` to API client
- [ ] Backend: Add `terraform_name` to all 8 transform configs
- [ ] Extend generator to handle transform config structure
- [ ] Implement `TransformResource` struct (similar to ConnectorResource)
- [ ] Generate schemas for all 8 transform types
- [ ] Register transform resources: `streamkap_transform_map_filter`, `streamkap_transform_sql_join`, etc.
- [ ] Handle transform implementation code (JS/Python/SQL) as string attributes
- [ ] Add transform deployment support (optional: deploy on create/update)
- [ ] Add tests for all transform resources
- [ ] Keep existing transform datasource for backward compatibility

### Phase 6: Automation & Polish

**Objective:** Enable full automation and clean up

**CI/CD Activation:**
- [ ] Enable CI regeneration workflow (remove `disabled` flag)
- [ ] Add backend webhook trigger (repository_dispatch)
- [ ] Test full regeneration cycle: backend change → PR → merge → release
- [ ] Set up scheduled drift detection (weekly `terraform plan` on examples)

**Documentation:**
- [ ] Run terraform-plugin-docs to generate all docs
- [ ] Update provider documentation for all new resources
- [ ] Add examples for Tag, Transform resources in `examples/`
- [ ] Create migration guide for existing users

**Quality Assurance:**
- [ ] Performance testing (compare old vs new response times)
- [ ] Security review (focus on sensitive field handling)
- [ ] Run full acceptance test suite
- [ ] Test state import for all resource types

**Release:**
- [ ] Create first release with new architecture (v2.0.0)
- [ ] Verify Terraform Registry publishing works
- [ ] Announce deprecation timeline for old patterns

### Phase 7: Future Enhancements (Optional)

| Enhancement | Priority | Description |
|-------------|----------|-------------|
| Kafka Users resource | Medium | Manage Kafka access credentials |
| Consumer Groups datasource | Low | Read consumer group offsets |
| Transform clone support | Low | Clone transforms via TF |
| Bulk operations | Low | Bulk create/delete for pipelines |

---

## 14. Success Metrics

| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| Lines of connector code | ~5,749 | ~600 | `wc -l internal/resource/**/*.go` |
| Entity coverage | 17 | 53+ | Count of TF resources + datasources |
| Connector coverage | 13/43 | 43/43 | Sources + Destinations |
| Transform coverage | 0/8 | 8/8 | Full CRUD resources |
| Time to add new connector | ~4 hours | ~5 min | Backend config + regenerate |
| Time to add new transform | N/A | ~5 min | Backend config + regenerate |
| Test coverage | Unknown | >80% | `go test -cover` |
| Sync lag with backend | Days/weeks | <1 day | PR creation time |

### 13.1 Testing Strategy

**Framework:** Use [terraform-plugin-testing](https://github.com/hashicorp/terraform-plugin-testing) (the official testing module).

**Matrix Testing:** Parameterize acceptance tests across connector types to maximize coverage with minimal test code:

```go
func TestAccConnectorResource(t *testing.T) {
    connectorTypes := []string{"postgresql", "mysql", "mongodb", "snowflake", "clickhouse"}
    for _, ct := range connectorTypes {
        t.Run(ct, func(t *testing.T) {
            resource.Test(t, resource.TestCase{
                ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
                Steps: []resource.TestStep{
                    {
                        Config: testAccConnectorConfig(ct),
                        Check:  testAccConnectorChecks(ct),
                    },
                },
            })
        })
    }
}
```

**Schema Validation:** Use built-in Framework capabilities:

```go
func TestConnectorSchema(t *testing.T) {
    ctx := context.Background()
    schemaResp := &resource.SchemaResponse{}
    NewConnectorResource("postgresql").Schema(ctx, resource.SchemaRequest{}, schemaResp)
    if schemaResp.Diagnostics.HasError() {
        t.Fatalf("Schema validation failed: %+v", schemaResp.Diagnostics)
    }
}
```

---

## 15. Edge Cases & Considerations

### Protocol v6 Computed Attribute Strictness

**Critical:** Terraform Plugin Protocol v6 enforces strict computed value handling. If a computed attribute returns a value different from what was planned, Terraform raises an **error** (not a warning as in v5).

**Implications:**
- API response parsing must be precise - don't let API quirks (extra fields, different casing) cause plan/apply mismatches
- Use `UseStateForUnknown()` plan modifier for computed fields that don't change after creation
- For fields that may change (e.g., `connector_status`), ensure the schema marks them as computed AND the Read() correctly updates them

```go
// Correct: API may update status independently
"connector_status": schema.StringAttribute{
    Computed:    true,
    Description: "Current connector status (Active, Paused, Broken, etc.)",
    // No UseStateForUnknown - status can change
},

// Correct: ID never changes after creation
"id": schema.StringAttribute{
    Computed: true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.UseStateForUnknown(),
    },
},
```

### WriteOnly Attributes (Terraform 1.11+)

For sensitive data that shouldn't persist in state, Terraform 1.11+ introduces WriteOnly attributes. Consider for:
- API keys that are write-once
- Passwords that should not be readable from state

**Note:** Only use if backend supports write-only semantics. Most Streamkap sensitive fields need to persist in state for updates.

### Import Functionality

Every resource (including generic ConnectorResource) must implement `ImportState`:

```go
func (r *ConnectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

For connectors, import requires the connector ID from the Streamkap API. Users can find this in the UI or via API.

### Transform Implementation Handling

Transform resources need special handling for user code:
```hcl
resource "streamkap_transform_map_filter" "example" {
  name     = "my_transform"
  language = "JAVASCRIPT"

  # Code can be inline or from file
  value_transform = file("${path.module}/transforms/value.js")

  # Optional deployment
  deploy_live = true
}
```

- Implementation code stored as string attribute
- Sensitive code should use `sensitive = true`
- Deploy triggers available: `deploy_live`, `deploy_preview`

### Topic Delete Constraints

Topic Delete will fail if topic is in use:
```
Error: Cannot delete topic - in use by pipeline(s): my-pipeline
```

Terraform should surface this error clearly. Consider:
- Adding `force_delete` attribute (dangerous, may break pipelines)
- Or just let backend error propagate

### Tag Validation

Tags have complex validation:
- Cannot delete tags in use by entities
- Cannot update `environment`/`general` types
- Only one `environment` tag per tenant

TF should handle these errors gracefully and provide clear messages.

### Backward Compatibility

- Keep existing datasources (`streamkap_transform`, `streamkap_tag`) for read operations
- New resources use different names where needed (e.g., `streamkap_transform_map_filter`)
- Existing state files should continue to work after migration

---

## 16. References & Sources

### Internal Audit Documents

| Document | Path | Purpose |
|----------|------|---------|
| Entity Configuration Schema Audit | `docs/audits/entity-config-schema-audit.md` | Complete schema reference for backend `configuration.latest.json` files |
| Backend Code Reference Guide | `docs/audits/backend-code-reference.md` | Key backend code locations, API patterns, CRUD flows |

### Official HashiCorp Resources

| Resource | Version/Status | Notes |
|----------|----------------|-------|
| [Terraform Provider Code Generation](https://developer.hashicorp.com/terraform/plugin/code-generation) | Tech Preview | Doesn't generate CRUD - reference only |
| [terraform-plugin-codegen-framework](https://github.com/hashicorp/terraform-plugin-codegen-framework) | Tech Preview | Actively maintained, generates schema only |
| [terraform-plugin-codegen-openapi](https://github.com/hashicorp/terraform-plugin-codegen-openapi) | Tech Preview | Maps OpenAPI to provider spec |
| [terraform-provider-scaffolding-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework) | Stable | Template for new providers |
| [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs) | v0.24.0 (Oct 2025) | ✅ Actively maintained - USE |
| [Release and Publish Tutorial](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-release-publish) | Current | Official guide |

### GitHub Actions

| Action | Latest Version | Notes |
|--------|----------------|-------|
| [actions/checkout](https://github.com/actions/checkout) | v6.0.1 | ✅ Actively maintained |
| [actions/setup-go](https://github.com/actions/setup-go) | v6 | ✅ Actively maintained |
| [goreleaser/goreleaser-action](https://github.com/goreleaser/goreleaser-action) | v6 | ✅ Requires GoReleaser v2 |
| [crazy-max/ghaction-import-gpg](https://github.com/crazy-max/ghaction-import-gpg) | v6.3.0 | ✅ Actively maintained |
| [aquasecurity/trivy-action](https://github.com/aquasecurity/trivy-action) | v0.33.1 | ✅ Actively maintained |
| [bridgecrewio/checkov-action](https://github.com/bridgecrewio/checkov-action) | v12.1347.0 | ✅ Actively maintained |

### Industry Case Studies

- [Cloudflare: Automatically Generating Terraform Provider](https://blog.cloudflare.com/automatically-generating-cloudflares-terraform-provider/) - Key learnings on schema linting, unified marshaling
- [Cloudflare: Lessons from Building an Automated SDK Pipeline](https://blog.cloudflare.com/lessons-from-building-an-automated-sdk-pipeline/) - CI lint rules, incremental validation
- [LogicMonitor: Custom Terraform Provider with OpenAPI](https://www.logicmonitor.com/blog/how-to-write-a-custom-terraform-provider-automatically-with-openapi) - Go-Swagger template customization

### Community Discussions

- [HashiCorp Forum: Code Generation for Provider Development](https://discuss.hashicorp.com/t/sdk-provider-development-anyone-ever-used-code-generation-or-other-tools-to-simplify-their-provider-development/20301)

### Security & Quality Tools

| Tool | Latest Version | Last Updated | Status |
|------|----------------|--------------|--------|
| [Trivy](https://github.com/aquasecurity/trivy) | v0.68.2 | Jan 8, 2026 | ✅ Very active (30k+ stars) |
| [Checkov](https://github.com/bridgecrewio/checkov) | v3.2.497 | Dec 30, 2025 | ✅ Very active (7.3k+ stars) |
| [GoReleaser](https://goreleaser.com/) | v2.13.2 | Dec 24, 2025 | ✅ Actively maintained (581 releases) |
| [pre-commit](https://pre-commit.com/) | Stable | - | ✅ Standard tooling |

### Core Framework

| Tool | Latest Version | Last Updated | Status |
|------|----------------|--------------|--------|
| [terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework) | v1.17.0 | Dec 2, 2025 | ✅ GA, actively maintained |
| [terraform-plugin-testing](https://github.com/hashicorp/terraform-plugin-testing) | v1.12.0 | Dec 2025 | ✅ GA, official testing module |
| [terraform-plugin-codegen-framework](https://github.com/hashicorp/terraform-plugin-codegen-framework) | v0.7.0 | Dec 27, 2025 | ✅ Tech Preview, actively maintained |
| [terraform-provider-scaffolding-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework) | Template | Jan 7, 2026 | ✅ Active reference |
| [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs) | v0.24.0 | Oct 2025 | ✅ Stable, actively maintained |

### Alternative Approaches (Evaluated, Not Selected)

| Tool | Reason Not Selected | Maintenance Status |
|------|---------------------|-------------------|
| [Speakeasy Terraform Generation](https://www.speakeasy.com/docs/create-terraform) | Commercial licensing required | Active |
| [dikhan/terraform-provider-openapi](https://github.com/dikhan/terraform-provider-openapi) | Runtime dynamic - doesn't leverage our config.json files | ⚠️ **DORMANT** (last code push Nov 2023) |
| [HashiCorp tfplugingen-openapi](https://github.com/hashicorp/terraform-plugin-codegen-openapi) | Tech preview, only generates schema (not CRUD) | Active |

### CI/CD Best Practices

- [Spacelift: Terraform in CI/CD](https://spacelift.io/blog/terraform-in-ci-cd)
- [Buildkite: Terraform CI/CD Workflows](https://buildkite.com/resources/blog/best-practices-for-terraform-ci-cd/)
- [Terraform Drift Detection Guide](https://dev.to/env0/the-ultimate-guide-to-terraform-drift-detection-how-to-detect-prevent-and-remediate-5hji)

---

*End of Design Document*
