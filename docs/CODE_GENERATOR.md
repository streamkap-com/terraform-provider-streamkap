# Code Generator (tfgen)

This document provides comprehensive documentation for the `tfgen` code generator tool that generates Terraform provider schemas from backend configuration files.

## Table of Contents

- [Overview](#overview)
- [CLI Usage](#cli-usage)
- [Type Mapping](#type-mapping)
- [overrides.json](#overridesjson)
- [deprecations.json](#deprecationsjson)
- [Adding a New Connector](#adding-a-new-connector)
- [Generated Code Structure](#generated-code-structure)
- [Troubleshooting](#troubleshooting)

## Overview

The `tfgen` tool automates Terraform provider schema generation by parsing backend `configuration.latest.json` files. This eliminates manual boilerplate and ensures consistency between the backend API and Terraform provider.

### Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│  Backend Repository (python-be-streamkap)                            │
│  app/{sources,destinations,transforms}/plugins/*/configuration.latest.json
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
│  - Load deprecations from deprecations.json                          │
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

### Key Benefits

1. **Consistency**: All connectors follow the same pattern
2. **Maintainability**: Changes to backend configs automatically propagate
3. **Type Safety**: Generates proper Go types with compile-time checks
4. **AI-Agent Compatibility**: Generates rich descriptions for AI tooling
5. **Backward Compatibility**: Supports deprecated field aliases

### Source Files

| File | Purpose |
|------|---------|
| `cmd/tfgen/main.go` | CLI entry point, command parsing |
| `cmd/tfgen/parser.go` | JSON config parsing, field extraction |
| `cmd/tfgen/generator.go` | Code generation, templates |
| `cmd/tfgen/overrides.json` | Custom type overrides for map fields |
| `cmd/tfgen/deprecations.json` | Deprecated field aliases |

## CLI Usage

### Installation

The tool is built with the provider:

```bash
go install ./cmd/tfgen
```

### Basic Usage

```bash
# Generate all connectors
tfgen generate --backend-path=/path/to/python-be-streamkap

# Generate specific entity type
tfgen generate --backend-path=/path/to/backend --entity-type=sources
tfgen generate --backend-path=/path/to/backend --entity-type=destinations
tfgen generate --backend-path=/path/to/backend --entity-type=transforms

# Generate specific connector
tfgen generate --backend-path=/path/to/backend --entity-type=sources --connector=postgresql

# Custom output directory
tfgen generate --backend-path=/path/to/backend --output=internal/generated
```

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--backend-path` | Yes | - | Path to the Streamkap backend repository |
| `--output` | No | `internal/generated` | Output directory for generated code |
| `--entity-type` | No | `all` | Entity type: `sources`, `destinations`, `transforms`, or `all` |
| `--connector` | No | - | Specific connector to generate (e.g., `postgresql`) |

### Using go generate

The recommended way to run tfgen is via `go generate`:

```bash
# Set the backend path
export STREAMKAP_BACKEND_PATH=/path/to/python-be-streamkap

# Run generation
go generate ./...
```

The `go generate` directive is typically configured in `internal/generated/doc.go`:

```go
//go:generate go run ../../cmd/tfgen generate --backend-path=$STREAMKAP_BACKEND_PATH
```

### Example Output

```
$ tfgen generate --backend-path=/Users/dev/python-be-streamkap
Backend path: /Users/dev/python-be-streamkap
Output: internal/generated
Entity type: all

Loaded 3 field overrides from cmd/tfgen/overrides.json
Loaded 10 deprecated field definitions from cmd/tfgen/deprecations.json

Generating source_alloydb.go...
Generating source_db2.go...
Generating source_documentdb.go...
...
Generating destination_snowflake.go...
Generating destination_clickhouse.go...
...
Generating transform_map_filter.go...
Generating transform_enrich.go...
...

Generation complete! Generated 51 schema files.
```

## Type Mapping

The generator maps backend control types to Terraform attribute types:

### Control Type → Terraform Type

| Backend Control | Terraform Type | Go Type | Notes |
|----------------|----------------|---------|-------|
| `string` | `schema.StringAttribute` | `types.String` | Basic string input |
| `password` | `schema.StringAttribute` | `types.String` | Marked `Sensitive: true` |
| `textarea` | `schema.StringAttribute` | `types.String` | Multi-line text |
| `json` | `schema.StringAttribute` | `types.String` | JSON as string |
| `datetime` | `schema.StringAttribute` | `types.String` | ISO datetime string |
| `number` | `schema.Int64Attribute` | `types.Int64` | Integer values |
| `boolean` | `schema.BoolAttribute` | `types.Bool` | True/false toggle |
| `toggle` | `schema.BoolAttribute` | `types.Bool` | True/false toggle |
| `one-select` | `schema.StringAttribute` | `types.String` | Enum with `OneOf` validator |
| `multi-select` | `schema.ListAttribute` | `types.List` | List of strings |
| `slider` | `schema.Int64Attribute` | `types.Int64` | Range with `Between` validator |

### Automatic Type Conversions

The generator applies smart conversions:

1. **Port fields**: Fields named `port` or ending with `_port` are converted to `Int64` even if stored as strings in the backend
2. **Sensitive fields**: Fields with `encrypt: true` or `control: "password"` are marked `Sensitive: true`
3. **Set-once fields**: Fields with `set_once: true` get `RequiresReplace()` plan modifier

### Default Value Handling

Defaults are generated based on type:

```go
// String default
Default: stringdefault.StaticString("upsert"),

// Int64 default
Default: int64default.StaticInt64(5),

// Bool default
Default: booldefault.StaticBool(true),
```

### Validator Generation

Validators are automatically generated from backend config:

```go
// one-select with raw_values: ["insert", "upsert"]
Validators: []validator.String{
    stringvalidator.OneOf("insert", "upsert"),
},

// slider with min: 1, max: 100
Validators: []validator.Int64{
    int64validator.Between(1, 100),
},
```

## overrides.json

The `cmd/tfgen/overrides.json` file defines custom field handling for complex types that can't be automatically inferred from the backend config.

### When to Use Overrides

Use overrides for:
- Map types (`map[string]T`)
- Nested map types (`map[string]struct{...}`)
- Fields requiring custom type handling

### Schema Structure

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

### Override Fields

| Field | Required | Description |
|-------|----------|-------------|
| `connector` | Yes | Connector code (e.g., `snowflake`, `clickhouse`) |
| `entity_type` | Yes | `sources`, `destinations`, or `transforms` |
| `api_field_name` | Yes | Backend API field name with dots |
| `terraform_attr_name` | Yes | Terraform attribute name with underscores |
| `type` | Yes | `map_string` or `map_nested` |
| `optional` | Yes | Whether the field is optional |
| `description` | Yes | Field description |

### Map String Type

Simple key-value map with string values:

```json
{
  "connector": "snowflake",
  "entity_type": "destinations",
  "api_field_name": "auto.qa.dedupe.table.mapping",
  "terraform_attr_name": "auto_qa_dedupe_table_mapping",
  "type": "map_string",
  "optional": true,
  "description": "Mapping between tables..."
}
```

Generates:

```go
// In Model struct
AutoQADedupeTableMapping map[string]types.String `tfsdk:"auto_qa_dedupe_table_mapping"`

// In Schema
"auto_qa_dedupe_table_mapping": schema.MapAttribute{
    Optional:    true,
    ElementType: types.StringType,
    Description: "Mapping between tables...",
}
```

### Map Nested Type

Map with structured values:

```json
{
  "connector": "clickhouse",
  "entity_type": "destinations",
  "api_field_name": "topics.config.map",
  "terraform_attr_name": "topics_config_map",
  "type": "map_nested",
  "optional": true,
  "description": "Per topic configuration in JSON format",
  "nested_model_name": "ClickHouseTopicsConfigMapItemModel",
  "nested_fields": [
    {
      "name": "delete_sql_execute",
      "terraform_attr_name": "delete_sql_execute",
      "type": "string",
      "optional": true
    }
  ]
}
```

Generates:

```go
// Nested model type
type clickHouseTopicsConfigMapItemModel struct {
    DeleteSQLExecute types.String `tfsdk:"delete_sql_execute"`
}

// In Model struct
TopicsConfigMap map[string]clickHouseTopicsConfigMapItemModel `tfsdk:"topics_config_map"`

// In Schema
"topics_config_map": schema.MapNestedAttribute{
    Optional: true,
    NestedObject: schema.NestedAttributeObject{
        Attributes: map[string]schema.Attribute{
            "delete_sql_execute": schema.StringAttribute{
                Optional: true,
            },
        },
    },
    Description: "Per topic configuration in JSON format",
}
```

### Nested Field Configuration

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Backend field name |
| `terraform_attr_name` | Yes | Terraform attribute name |
| `type` | Yes | `string`, `int64`, or `bool` |
| `optional` | No | Whether the field is optional |
| `required` | No | Whether the field is required |
| `validators` | No | Array of validator configs |

### Nested Field Validators

```json
{
  "name": "chunks",
  "terraform_attr_name": "chunks",
  "type": "int64",
  "required": true,
  "validators": [
    {
      "type": "int64_at_least",
      "value": 1
    }
  ]
}
```

Supported validator types:
- `int64_at_least`: Minimum value validator

## deprecations.json

The `cmd/tfgen/deprecations.json` file defines deprecated field aliases for backward compatibility.

### When to Use Deprecations

Use deprecations when:
- A field is being renamed but old configs should still work
- Maintaining backward compatibility during migrations
- Gradual transition to new field names

### Schema Structure

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

### Deprecation Fields

| Field | Required | Description |
|-------|----------|-------------|
| `connector` | Yes | Connector code (e.g., `postgresql`) |
| `entity_type` | Yes | `sources`, `destinations`, or `transforms` |
| `deprecated_attr` | Yes | Old/deprecated attribute name |
| `new_attr` | Yes | New attribute name to map to |
| `type` | Yes | `string`, `int64`, or `bool` |

### Generated Code

Deprecated fields are added to the Model struct only:

```go
type SourcePostgresqlModel struct {
    // ... regular fields ...

    // Deprecated fields - kept for backward compatibility
    InsertStaticKeyField1 types.String `tfsdk:"insert_static_key_field_1"`
    // ... more deprecated fields ...

    Timeouts timeouts.Value `tfsdk:"timeouts"`
}
```

The schema and field mappings for deprecated fields are added by the wrapper files in `internal/resource/source/` or `internal/resource/destination/`.

### Example: PostgreSQL Deprecations

```json
[
  {
    "connector": "postgresql",
    "entity_type": "sources",
    "deprecated_attr": "insert_static_key_field_1",
    "new_attr": "transforms_insert_static_key1_static_field",
    "type": "string"
  },
  {
    "connector": "postgresql",
    "entity_type": "sources",
    "deprecated_attr": "predicates_istopictoenrich_pattern",
    "new_attr": "predicates_is_topic_to_enrich_pattern",
    "type": "string"
  }
]
```

## Adding a New Connector

Follow these steps to add a new connector to the Terraform provider.

### Prerequisites

1. Backend has `configuration.latest.json` for the connector
2. API endpoints exist for CRUD operations

### Step 1: Generate the Schema

```bash
# Generate just the new connector
tfgen generate \
  --backend-path=/path/to/python-be-streamkap \
  --entity-type=sources \
  --connector=mynewconnector

# Or regenerate all
go generate ./...
```

### Step 2: Create the Wrapper File

Create `internal/resource/source/mynewconnector_generated.go` (or `destination/`):

```go
package source

import (
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
    "github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// Compile-time interface check
var _ connector.ConnectorConfig = (*mynewconnectorConfig)(nil)

type mynewconnectorConfig struct{}

func NewMynewconnectorResource() *connector.BaseConnectorResource {
    return connector.NewBaseConnectorResource(&mynewconnectorConfig{})
}

func (c *mynewconnectorConfig) GetSchema() schema.Schema {
    return generated.SourceMynewconnectorSchema()
}

func (c *mynewconnectorConfig) GetFieldMappings() map[string]string {
    return generated.SourceMynewconnectorFieldMappings
}

func (c *mynewconnectorConfig) GetConnectorType() string {
    return "source"
}

func (c *mynewconnectorConfig) GetConnectorCode() string {
    return "mynewconnector"
}

func (c *mynewconnectorConfig) GetResourceName() string {
    return "streamkap_source_mynewconnector"
}

func (c *mynewconnectorConfig) NewModelInstance() any {
    return &generated.SourceMynewconnectorModel{}
}
```

### Step 3: Register the Resource

Add to `internal/provider/provider.go`:

```go
func (p *StreamkapProvider) Resources(ctx context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        // ... existing resources ...
        source.NewMynewconnectorResource,
    }
}
```

### Step 4: Add Example Files

Create `examples/resources/streamkap_source_mynewconnector/`:

**basic.tf**:
```hcl
resource "streamkap_source_mynewconnector" "example" {
  name     = "my-connector"
  hostname = "db.example.com"
  port     = 5432
  username = "user"
  password = "secret"
  database = "mydb"
}
```

**complete.tf**:
```hcl
resource "streamkap_source_mynewconnector" "example" {
  name     = "my-connector"
  hostname = "db.example.com"
  port     = 5432
  username = "user"
  password = "secret"
  database = "mydb"

  # All optional fields with comments
  ssl_mode = "require"  # Valid values: disable, require, verify-ca, verify-full
}
```

**import.sh**:
```bash
terraform import streamkap_source_mynewconnector.example <connector-id>
```

### Step 5: Write Tests

Create `internal/provider/source_mynewconnector_resource_test.go`:

```go
package provider

import (
    "os"
    "testing"

    "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSourceMynewconnector_basic(t *testing.T) {
    if os.Getenv("MYNEWCONNECTOR_HOSTNAME") == "" {
        t.Skip("Set MYNEWCONNECTOR_HOSTNAME to run this test")
    }

    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccSourceMynewconnectorConfig(),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttrSet(
                        "streamkap_source_mynewconnector.test", "id"),
                ),
            },
        },
    })
}

func testAccSourceMynewconnectorConfig() string {
    return `
resource "streamkap_source_mynewconnector" "test" {
  name     = "tf-acc-test-mynewconnector"
  hostname = "` + os.Getenv("MYNEWCONNECTOR_HOSTNAME") + `"
  ...
}
`
}
```

### Step 6: Build and Verify

```bash
# Build
go build ./...

# Run schema compatibility tests
go test -v -run 'TestSchemaBackwardsCompatibility' ./internal/provider/...

# Run your new tests (if credentials available)
TF_ACC=1 go test -v -run 'TestAccSourceMynewconnector' ./internal/provider/...
```

### Step 7: Update Documentation

1. Add to README.md feature list
2. Run `go generate ./...` to update docs

## Generated Code Structure

### File Output

Generated files are placed in `internal/generated/`:

```
internal/generated/
├── doc.go                          # Package documentation
├── source_postgresql.go            # PostgreSQL source schema
├── source_mysql.go                 # MySQL source schema
├── source_mongodb.go               # MongoDB source schema
├── ...
├── destination_snowflake.go        # Snowflake destination schema
├── destination_clickhouse.go       # ClickHouse destination schema
├── ...
├── transform_map_filter.go         # MapFilter transform schema
├── transform_enrich.go             # Enrich transform schema
└── ...
```

### Generated File Structure

Each generated file contains:

```go
// Code generated by tfgen. DO NOT EDIT.

package generated

import (
    // Required imports
)

// Nested model types (if map_nested overrides present)
type clickHouseTopicsConfigMapItemModel struct {
    DeleteSQLExecute types.String `tfsdk:"delete_sql_execute"`
}

// Main model struct
type SourcePostgresqlModel struct {
    ID              types.String `tfsdk:"id"`
    Name            types.String `tfsdk:"name"`
    Connector       types.String `tfsdk:"connector"`
    DatabaseHostname types.String `tfsdk:"database_hostname"`
    // ... all fields ...

    // Deprecated fields - kept for backward compatibility
    InsertStaticKeyField1 types.String `tfsdk:"insert_static_key_field_1"`

    Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Schema function
func SourcePostgresqlSchema() schema.Schema {
    return schema.Schema{
        Description:         "Manages a PostgreSQL source connector.",
        MarkdownDescription: "Manages a **PostgreSQL source connector**.\n\n...",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed: true,
                // ...
            },
            // ... all attributes ...
        },
    }
}

// Field mappings
var SourcePostgresqlFieldMappings = map[string]string{
    "database_hostname": "database.hostname.user.defined",
    "database_port":     "database.port.user.defined",
    // ... all mappings ...
}
```

### DO NOT EDIT

Generated files include the header:

```go
// Code generated by tfgen. DO NOT EDIT.
```

**Never manually edit generated files.** Instead:
- Fix the backend `configuration.latest.json`
- Add entries to `overrides.json` or `deprecations.json`
- Re-run the generator

## Troubleshooting

### Common Issues

#### "backend path does not exist"

```
Error: backend path does not exist: /path/to/backend
```

**Solution**: Verify the backend path exists and contains the plugin directories.

#### "no configuration.latest.json found"

```
Skipping source postgresql: no configuration.latest.json found
```

**Solution**: Ensure the connector has a `configuration.latest.json` file in its plugin directory.

#### "failed to format generated code"

```
Error: failed to format generated code (see internal/generated/source_xyz.go.unformatted)
```

**Solution**: Check the `.unformatted` file for syntax errors. Usually caused by template issues.

#### "unknown entity type"

```
Error: unknown entity type: source (valid: sources, destinations, transforms, all)
```

**Solution**: Use plural form: `--entity-type=sources`

### Debugging Tips

1. **Check raw output**: Look for `.unformatted` files when generation fails
2. **Verify JSON**: Use `jq` to validate `configuration.latest.json`
3. **Check overrides**: Ensure override `api_field_name` matches exactly
4. **Build test**: Run `go build ./...` after generation

### Re-generating All Schemas

When backend configs change significantly:

```bash
# Clean and regenerate
rm internal/generated/*.go
go generate ./...

# Verify
go build ./...
go test -v -short ./...
```
