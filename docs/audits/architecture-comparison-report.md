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

*Section to be completed in US-003*

---

## Comparison Matrix

*Section to be completed in US-005*

---

## Trade-offs

*Section to be completed in US-005*

---
