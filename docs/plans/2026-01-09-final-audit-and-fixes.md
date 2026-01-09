# Final Audit and Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Complete all remaining work from the refactor: add deprecated attribute aliases for backward compatibility, add transform tests, update all documentation, clean up files, and create migration guide.

**Architecture:** Add deprecated aliases as thin wrappers (no code duplication), update docs, add tests, clean up.

**Tech Stack:** Go, Terraform Plugin Framework, Markdown

---

## Critical Finding: Breaking Changes Detected

The refactor introduced breaking changes. We will add **deprecated aliases** for backward compatibility.

### PostgreSQL Source Breaking Changes
1. `database_port` type: `Int64` → `String` (CANNOT add alias - type change)
2. 9 attribute names changed (CAN add deprecated aliases)
3. `signal_data_collection_schema_or_database`: optional → required

### Snowflake Destination Breaking Changes
1. `auto_schema_creation` → `create_schema_auto` (CAN add deprecated alias)
2. `auto_qa_dedupe_table_mapping` type: `Map` → `String` (CANNOT add alias - type change)

### Approach: Deprecated Aliases

For renamed attributes, we add the OLD name as a deprecated alias that maps to the new name. Users see deprecation warnings but their configs still work.

---

## Task 0: Verify Previous Plans Completed

**Purpose:** Verify that all work from previous implementation plans (2026-01-08-provider-refactor-design.md and 2026-01-09-remaining-work-plan.md) has been properly completed.

**Step 1: Run build and tests**

```bash
go build ./...
go test -short ./...
```
Expected: All pass

**Step 2: Verify all 13 connectors exist**

Check these files exist and have proper schemas:
- Sources: mongodb, mysql, postgresql, dynamodb, sqlserver, kafkadirect (6)
- Destinations: snowflake, clickhouse, databricks, postgresql, s3, iceberg, kafka (7)

**Step 3: Verify transform resources exist**

Check these files exist in `internal/resource/transform/`:
- base.go (BaseTransformResource)
- map_filter_generated.go, enrich_generated.go, enrich_async_generated.go
- sql_join_generated.go, rollup_generated.go, fan_out_generated.go (6 wrappers)

**Step 4: Verify Transform API client**

Check `internal/api/transform.go` has:
- CreateTransform, UpdateTransform, GetTransform, DeleteTransform

**Step 5: Log verification result**

Document: "Previous plans verified complete - all 13 connectors, 6 transforms, and API client confirmed"

---

## Task 1: Add Deprecated Aliases Helper

**Files:**
- Create: `internal/helper/deprecated.go`

**Step 1: Create deprecated alias helper**

```go
package helper

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DeprecatedAlias represents a mapping from old attribute name to new attribute name
type DeprecatedAlias struct {
	OldName string
	NewName string
}

// PostgreSQLDeprecatedAliases contains all deprecated attribute mappings for PostgreSQL source
var PostgreSQLDeprecatedAliases = []DeprecatedAlias{
	{OldName: "insert_static_key_field_1", NewName: "transforms_insert_static_key1_static_field"},
	{OldName: "insert_static_key_value_1", NewName: "transforms_insert_static_key1_static_value"},
	{OldName: "insert_static_value_field_1", NewName: "transforms_insert_static_value1_static_field"},
	{OldName: "insert_static_value_1", NewName: "transforms_insert_static_value1_static_value"},
	{OldName: "insert_static_key_field_2", NewName: "transforms_insert_static_key2_static_field"},
	{OldName: "insert_static_key_value_2", NewName: "transforms_insert_static_key2_static_value"},
	{OldName: "insert_static_value_field_2", NewName: "transforms_insert_static_value2_static_field"},
	{OldName: "insert_static_value_2", NewName: "transforms_insert_static_value2_static_value"},
	{OldName: "predicates_istopictoenrich_pattern", NewName: "predicates_is_topic_to_enrich_pattern"},
}

// SnowflakeDeprecatedAliases contains all deprecated attribute mappings for Snowflake destination
var SnowflakeDeprecatedAliases = []DeprecatedAlias{
	{OldName: "auto_schema_creation", NewName: "create_schema_auto"},
}

// CreateDeprecatedStringAttribute creates a deprecated string attribute that warns users
func CreateDeprecatedStringAttribute(newName string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		DeprecationMessage:  "This attribute is deprecated. Use '" + newName + "' instead.",
		Description:         "DEPRECATED: Use '" + newName + "' instead.",
		MarkdownDescription: "**DEPRECATED:** Use `" + newName + "` instead.",
		Default:             stringdefault.StaticString(""),
	}
}

// MigrateDeprecatedValues copies values from deprecated attributes to new attributes if set
// Call this in Create/Update before processing the model
func MigrateDeprecatedValues(ctx context.Context, config map[string]any, aliases []DeprecatedAlias) map[string]any {
	for _, alias := range aliases {
		if oldVal, exists := config[alias.OldName]; exists && oldVal != nil && oldVal != "" {
			tflog.Warn(ctx, "Deprecated attribute used",
				map[string]any{
					"deprecated": alias.OldName,
					"use":        alias.NewName,
				})
			// Only copy if new value is not set
			if newVal, newExists := config[alias.NewName]; !newExists || newVal == nil || newVal == "" {
				config[alias.NewName] = oldVal
			}
			// Remove old key to avoid API confusion
			delete(config, alias.OldName)
		}
	}
	return config
}
```

**Step 2: Create unit test for deprecated helper**

Create: `internal/helper/deprecated_test.go`

```go
package helper

import (
	"context"
	"testing"
)

func TestMigrateDeprecatedValues(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		config   map[string]any
		aliases  []DeprecatedAlias
		expected map[string]any
	}{
		{
			name: "migrates deprecated value when new value not set",
			config: map[string]any{
				"old_name": "test_value",
			},
			aliases: []DeprecatedAlias{
				{OldName: "old_name", NewName: "new_name"},
			},
			expected: map[string]any{
				"new_name": "test_value",
			},
		},
		{
			name: "does not overwrite existing new value",
			config: map[string]any{
				"old_name": "old_value",
				"new_name": "new_value",
			},
			aliases: []DeprecatedAlias{
				{OldName: "old_name", NewName: "new_name"},
			},
			expected: map[string]any{
				"new_name": "new_value",
			},
		},
		{
			name: "handles nil deprecated value",
			config: map[string]any{
				"old_name": nil,
			},
			aliases: []DeprecatedAlias{
				{OldName: "old_name", NewName: "new_name"},
			},
			expected: map[string]any{
				"old_name": nil,
			},
		},
		{
			name: "handles empty deprecated value",
			config: map[string]any{
				"old_name": "",
			},
			aliases: []DeprecatedAlias{
				{OldName: "old_name", NewName: "new_name"},
			},
			expected: map[string]any{
				"old_name": "",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MigrateDeprecatedValues(ctx, tc.config, tc.aliases)
			for key, expectedVal := range tc.expected {
				if result[key] != expectedVal {
					t.Errorf("expected %s=%v, got %v", key, expectedVal, result[key])
				}
			}
		})
	}
}

func TestPostgreSQLDeprecatedAliases(t *testing.T) {
	if len(PostgreSQLDeprecatedAliases) != 9 {
		t.Errorf("expected 9 PostgreSQL deprecated aliases, got %d", len(PostgreSQLDeprecatedAliases))
	}
}

func TestSnowflakeDeprecatedAliases(t *testing.T) {
	if len(SnowflakeDeprecatedAliases) != 1 {
		t.Errorf("expected 1 Snowflake deprecated alias, got %d", len(SnowflakeDeprecatedAliases))
	}
}
```

**Step 3: Verify helper and test compile**

Run: `go build ./...`
Run: `go test ./internal/helper/... -v`
Expected: Build succeeds, tests pass

**Step 4: Commit deprecated helper with tests**

```bash
git add internal/helper/deprecated.go
git commit -m "feat(helper): add deprecated attribute alias support

Provides backward compatibility for renamed attributes.
Users see deprecation warnings but configs still work.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Add Deprecated Aliases to PostgreSQL Source Schema

**Files:**
- Modify: `internal/resource/source/postgresql_generated.go`

**Step 1: Add deprecated aliases to schema**

In the `PostgreSQLConfig` struct's `GetSchema()` method, add the deprecated attributes alongside the new ones. The deprecated attributes should be:

```go
// Add these deprecated aliases in the Attributes map:
"insert_static_key_field_1": schema.StringAttribute{
    Optional:           true,
    Computed:           true,
    DeprecationMessage: "Use 'transforms_insert_static_key1_static_field' instead.",
    Description:        "DEPRECATED: Use 'transforms_insert_static_key1_static_field' instead.",
},
"insert_static_key_value_1": schema.StringAttribute{
    Optional:           true,
    Computed:           true,
    DeprecationMessage: "Use 'transforms_insert_static_key1_static_value' instead.",
    Description:        "DEPRECATED: Use 'transforms_insert_static_key1_static_value' instead.",
},
// ... (add all 9 deprecated aliases)
"predicates_istopictoenrich_pattern": schema.StringAttribute{
    Optional:           true,
    Computed:           true,
    DeprecationMessage: "Use 'predicates_is_topic_to_enrich_pattern' instead.",
    Description:        "DEPRECATED: Use 'predicates_is_topic_to_enrich_pattern' instead.",
},
```

**Step 2: Update field mappings**

Add the deprecated field names to `PostgreSQLFieldMappings` so they map to the correct API fields:

```go
// Add deprecated mappings
"insert_static_key_field_1":          "transforms.insert.static.key1.static.field",
"insert_static_key_value_1":          "transforms.insert.static.key1.static.value",
// ... etc
```

**Step 3: Verify build**

Run: `go build ./...`
Expected: Build succeeds

**Step 4: Commit PostgreSQL deprecated aliases**

```bash
git add internal/resource/source/postgresql_generated.go
git commit -m "feat(source/postgresql): add deprecated aliases for renamed attributes

Backward compatibility: old attribute names still work with deprecation warnings.
- insert_static_key_field_1 -> transforms_insert_static_key1_static_field
- insert_static_key_value_1 -> transforms_insert_static_key1_static_value
- (and 7 more)

Users should migrate to new names. Old names will be removed in v3.0.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Add Deprecated Alias to Snowflake Destination Schema

**Files:**
- Modify: `internal/resource/destination/snowflake_generated.go`

**Step 1: Add deprecated alias to schema**

Add the deprecated `auto_schema_creation` attribute:

```go
"auto_schema_creation": schema.BoolAttribute{
    Optional:           true,
    Computed:           true,
    DeprecationMessage: "Use 'create_schema_auto' instead.",
    Description:        "DEPRECATED: Use 'create_schema_auto' instead.",
},
```

**Step 2: Update field mappings**

Add deprecated mapping:

```go
"auto_schema_creation": "create.schema.auto",
```

**Step 3: Verify build**

Run: `go build ./...`
Expected: Build succeeds

**Step 4: Commit Snowflake deprecated alias**

```bash
git add internal/resource/destination/snowflake_generated.go
git commit -m "feat(destination/snowflake): add deprecated alias for auto_schema_creation

Backward compatibility: auto_schema_creation still works with deprecation warning.
Users should migrate to create_schema_auto. Old name will be removed in v3.0.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Create Migration Guide with Deprecation Info

**Files:**
- Create: `docs/MIGRATION.md`

**Step 1: Create comprehensive migration documentation**

```markdown
# Migration Guide: v1.x to v2.0

This guide helps existing users migrate their Terraform configurations to the new version.

## Backward Compatibility

**Good news!** Most attribute renames have **deprecated aliases** that provide backward compatibility. Your existing configurations will continue to work, but you'll see deprecation warnings.

### Deprecated Attributes (Still Work, But Migrate Soon)

These old attribute names still work but show deprecation warnings:

#### PostgreSQL Source

| Deprecated (Old) Name | New Name | Action |
|-----------------------|----------|--------|
| `insert_static_key_field_1` | `transforms_insert_static_key1_static_field` | Rename in config |
| `insert_static_key_value_1` | `transforms_insert_static_key1_static_value` | Rename in config |
| `insert_static_value_field_1` | `transforms_insert_static_value1_static_field` | Rename in config |
| `insert_static_value_1` | `transforms_insert_static_value1_static_value` | Rename in config |
| `insert_static_key_field_2` | `transforms_insert_static_key2_static_field` | Rename in config |
| `insert_static_key_value_2` | `transforms_insert_static_key2_static_value` | Rename in config |
| `insert_static_value_field_2` | `transforms_insert_static_value2_static_field` | Rename in config |
| `insert_static_value_2` | `transforms_insert_static_value2_static_value` | Rename in config |
| `predicates_istopictoenrich_pattern` | `predicates_is_topic_to_enrich_pattern` | Rename in config |

#### Snowflake Destination

| Deprecated (Old) Name | New Name | Action |
|-----------------------|----------|--------|
| `auto_schema_creation` | `create_schema_auto` | Rename in config |

> **Note:** Deprecated attributes will be removed in v3.0. Please migrate before then.

## Breaking Changes (Require Immediate Action)

These changes do NOT have backward compatibility and require updates:

### PostgreSQL Source

#### Type Change: database_port

**Before (v1.x):**
```hcl
resource "streamkap_source_postgresql" "example" {
  database_port = 5432  # Integer - NO LONGER WORKS
}
```

**After (v2.0):**
```hcl
resource "streamkap_source_postgresql" "example" {
  database_port = "5432"  # String - REQUIRED
}
```

#### New Required Field

`signal_data_collection_schema_or_database` is now required:

```hcl
resource "streamkap_source_postgresql" "example" {
  signal_data_collection_schema_or_database = "public"
}
```

### Snowflake Destination

#### Type Change: auto_qa_dedupe_table_mapping

**Before (v1.x):**
```hcl
resource "streamkap_destination_snowflake" "example" {
  auto_qa_dedupe_table_mapping = {
    rawTable1 = "dedupeTable1"
  }
}
```

**After (v2.0):**
```hcl
resource "streamkap_destination_snowflake" "example" {
  auto_qa_dedupe_table_mapping = "rawTable1:dedupeTable1"
}
```

## Default Value Changes

These affect NEW resources only. Existing resources are unaffected:

| Resource | Attribute | Old Default | New Default |
|----------|-----------|-------------|-------------|
| PostgreSQL Source | `heartbeat_enabled` | `false` | `true` |
| Snowflake Destination | `hard_delete` | `false` | `true` |

## Migration Steps

### Step 1: Backup Your State

```bash
cp terraform.tfstate terraform.tfstate.backup
```

### Step 2: Fix Breaking Changes

Update these in your `.tf` files:
1. Change `database_port = 5432` to `database_port = "5432"`
2. Add `signal_data_collection_schema_or_database = "public"` if missing
3. Change map-style `auto_qa_dedupe_table_mapping` to string format

### Step 3: (Optional) Fix Deprecated Attributes

While not required immediately, update deprecated attribute names to avoid warnings:
- Rename attributes as shown in the tables above

### Step 4: Verify

```bash
terraform init -upgrade
terraform plan
```

You should see:
- No errors for breaking changes (if fixed)
- Deprecation warnings for old attribute names (if not yet migrated)

### Step 5: Apply

```bash
terraform apply
```

## New Features in v2.0

### Transform Resources (NEW)

Six new transform resource types:

- `streamkap_transform_map_filter` - Transform/Filter Records
- `streamkap_transform_enrich` - Enrich transforms
- `streamkap_transform_enrich_async` - Async Enrich transforms
- `streamkap_transform_sql_join` - SQL Join transforms
- `streamkap_transform_rollup` - Rollup transforms
- `streamkap_transform_fan_out` - Fan Out transforms

Example:
```hcl
resource "streamkap_transform_map_filter" "example" {
  name                                   = "my-transform"
  transforms_input_topic_pattern         = "source-topic"
  transforms_output_topic_pattern        = "output-topic"
  transforms_input_serialization_format  = "AVRO"
  transforms_output_serialization_format = "AVRO"
}
```

## Deprecation Timeline

| Version | Status |
|---------|--------|
| v2.0 | Deprecated attributes work with warnings |
| v2.x | Deprecated attributes continue to work |
| v3.0 | Deprecated attributes REMOVED |

## Getting Help

If you encounter issues:
1. Check [GitHub Issues](https://github.com/streamkap-com/terraform-provider-streamkap/issues)
2. Open a new issue with your error message and config (redact secrets)
```

**Step 2: Commit migration guide**

```bash
git add docs/MIGRATION.md
git commit -m "docs: add comprehensive migration guide for v2.0

Includes:
- Deprecated attributes (backward compatible with warnings)
- Breaking changes requiring immediate action
- Step-by-step migration instructions
- Deprecation timeline

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Add Transform Acceptance Tests (3 types)

**Files:**
- Create: `internal/provider/transform_map_filter_resource_test.go`
- Create: `internal/provider/transform_enrich_resource_test.go`
- Create: `internal/provider/transform_sql_join_resource_test.go`

### Step 1: Create map_filter transform test

```go
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformMapFilterResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTransformMapFilterResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test", "name", "tf-acc-test-transform-map-filter"),
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test", "transforms_input_serialization_format", "AVRO"),
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test", "transforms_output_serialization_format", "AVRO"),
					resource.TestCheckResourceAttrSet("streamkap_transform_map_filter.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_map_filter.test", "transform_type"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_transform_map_filter.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccTransformMapFilterResourceConfig() string {
	return `
resource "streamkap_transform_map_filter" "test" {
  name                                   = "tf-acc-test-transform-map-filter"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "AVRO"
  transforms_output_serialization_format = "AVRO"
  transforms_language                    = "PYTHON"
}
`
}
```

### Step 2: Create enrich transform test

```go
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformEnrichResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformEnrichResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_enrich.test", "name", "tf-acc-test-transform-enrich"),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich.test", "transform_type"),
				),
			},
			{
				ResourceName:      "streamkap_transform_enrich.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccTransformEnrichResourceConfig() string {
	return `
resource "streamkap_transform_enrich" "test" {
  name                                   = "tf-acc-test-transform-enrich"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "AVRO"
  transforms_output_serialization_format = "AVRO"
}
`
}
```

### Step 3: Create sql_join transform test

```go
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformSqlJoinResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformSqlJoinResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_sql_join.test", "name", "tf-acc-test-transform-sql-join"),
					resource.TestCheckResourceAttrSet("streamkap_transform_sql_join.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_sql_join.test", "transform_type"),
				),
			},
			{
				ResourceName:      "streamkap_transform_sql_join.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccTransformSqlJoinResourceConfig() string {
	return `
resource "streamkap_transform_sql_join" "test" {
  name                                   = "tf-acc-test-transform-sql-join"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "AVRO"
  transforms_output_serialization_format = "AVRO"
}
`
}
```

### Step 4: Verify test files compile

Run: `go build ./...`
Expected: Build succeeds

### Step 5: Commit transform tests

```bash
git add internal/provider/transform_*_resource_test.go
git commit -m "test: add acceptance tests for 3 transform resource types

Added tests for:
- transform_map_filter
- transform_enrich
- transform_sql_join

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 5.5: Add Destination Kafka Acceptance Test

**Files:**
- Create: `internal/provider/destination_kafka_resource_test.go`

**Step 1: Create destination_kafka test**

```go
package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDestinationKafkaResource_basic(t *testing.T) {
	bootstrapServers := os.Getenv("TF_VAR_destination_kafka_bootstrap_servers")
	if bootstrapServers == "" {
		t.Skip("TF_VAR_destination_kafka_bootstrap_servers not set, skipping test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDestinationKafkaResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "name", "tf-acc-test-destination-kafka"),
					resource.TestCheckResourceAttrSet("streamkap_destination_kafka.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_destination_kafka.test", "connector"),
				),
			},
			{
				ResourceName:            "streamkap_destination_kafka.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"sasl_password"},
			},
		},
	})
}

func testAccDestinationKafkaResourceConfig() string {
	return `
variable "destination_kafka_bootstrap_servers" {
  type = string
}

variable "destination_kafka_sasl_username" {
  type    = string
  default = ""
}

variable "destination_kafka_sasl_password" {
  type      = string
  sensitive = true
  default   = ""
}

resource "streamkap_destination_kafka" "test" {
  name              = "tf-acc-test-destination-kafka"
  bootstrap_servers = var.destination_kafka_bootstrap_servers
  security_protocol = "PLAINTEXT"
}
`
}
```

**Step 2: Verify test file compiles**

Run: `go build ./...`
Expected: Build succeeds

**Step 3: Commit destination_kafka test**

```bash
git add internal/provider/destination_kafka_resource_test.go
git commit -m "test: add acceptance test for destination_kafka resource

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Update CLAUDE.md with Transform Resources

**Files:**
- Modify: `CLAUDE.md`

**Step 1: Update Resources section**

Add Transform resources to the Resources list:
```markdown
- **Resources** (`internal/resource/`): Source connectors (PostgreSQL, MySQL, MongoDB, DynamoDB, SQL Server, KafkaDirect), Destination connectors (Snowflake, ClickHouse, Databricks, PostgreSQL, S3, Iceberg, Kafka), Transform resources (MapFilter, Enrich, EnrichAsync, SqlJoin, Rollup, FanOut), Pipelines, Topics
```

**Step 2: Update API Client section**

Add Transform CRUD operations:
```markdown
- CreateTransform, UpdateTransform, GetTransform, DeleteTransform
```

**Step 3: Add deprecation note**

Add note about deprecated attributes and migration guide.

**Step 4: Commit CLAUDE.md**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md with Transform resources and deprecation info

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 7: Update README.md with Transform Resources

**Files:**
- Modify: `README.md`

**Step 1: Add Transform section**

Add Transform resources listing all 6 types.

**Step 2: Fix examples/full path**

Change `cd examples/full` to `cd examples/provider`.

**Step 3: Update project structure**

Add `internal/resource/transform/` to directory structure.

**Step 4: Add migration guide reference**

Add link to MIGRATION.md for users upgrading from v1.x.

**Step 5: Commit README.md**

```bash
git add README.md
git commit -m "docs: update README with Transform resources, fix paths, add migration link

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 8: Update DEVELOPMENT.md with Transforms

**Files:**
- Modify: `docs/DEVELOPMENT.md`

**Step 1: Update generated files section**

Add `transform_*.go` to the list of generated files.

**Step 2: Add Transform resources mention**

Mention Transform resources where sources and destinations are discussed.

**Step 3: Commit DEVELOPMENT.md**

```bash
git add docs/DEVELOPMENT.md
git commit -m "docs: update DEVELOPMENT.md with Transform resources

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 9: Update ARCHITECTURE.md with Transforms

**Files:**
- Modify: `docs/ARCHITECTURE.md`

**Step 1: Update resource counts**

Change to include 6 Transform resources.

**Step 2: Add transform directory**

Add `internal/resource/transform/` to directory structure.

**Step 3: Update API client description**

Add Transform APIs.

**Step 4: Commit ARCHITECTURE.md**

```bash
git add docs/ARCHITECTURE.md
git commit -m "docs: update ARCHITECTURE.md with Transform resources

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 10: Update .env.example

**Files:**
- Modify: `.env.example`

**Step 1: Add Kafka destination variables**

```bash
# ================================
# Destination: Kafka
# ================================
TF_VAR_destination_kafka_bootstrap_servers=
TF_VAR_destination_kafka_security_protocol=
TF_VAR_destination_kafka_sasl_mechanism=
TF_VAR_destination_kafka_sasl_username=
TF_VAR_destination_kafka_sasl_password=
```

**Step 2: Add KafkaDirect source variables if missing**

```bash
# ================================
# Source: KafkaDirect
# ================================
TF_VAR_source_kafkadirect_bootstrap_servers=
TF_VAR_source_kafkadirect_topic=
```

**Step 3: Commit .env.example**

```bash
git add .env.example
git commit -m "docs: add Kafka and KafkaDirect variables to .env.example

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 11: Clean Up Uncommitted Files

**Files:**
- Update: `.gitignore`

**Step 1: Update .gitignore**

Add entries to ignore local tooling:
```
# Local tooling
.mcp.json
.serena/
tfgen
```

**Step 2: Commit .gitignore**

```bash
git add .gitignore
git commit -m "chore: update .gitignore for local tooling files

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

**Step 3: Commit audit and plan documentation**

```bash
git add docs/audits/ docs/plans/
git commit -m "docs: add audit documents and implementation plans

These documents provide valuable context about the refactor:
- Backend code reference guide
- Entity config schema audit
- Provider refactor design plan
- Remaining work plan
- Final audit and fixes plan

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 12: Final Verification

**Step 1: Run build**

```bash
go build ./...
```
Expected: Success

**Step 2: Run all tests**

```bash
go test -short ./...
```
Expected: All tests pass

**Step 3: Verify git status is clean**

```bash
git status
```
Expected: Only untracked local files (.mcp.json, .serena/, tfgen) which are in .gitignore

**Step 4: Review commit history**

```bash
git log --oneline -20
```
Expected: Clean commit history

---

## Task 13: Create CHANGELOG.md

**Files:**
- Create: `CHANGELOG.md`

**Step 1: Create changelog file**

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2026-01-09

### Added
- **Transform Resources**: 6 new resource types for data transformation
  - `streamkap_transform_map_filter` - Filter and transform records
  - `streamkap_transform_enrich` - Enrich data with external sources
  - `streamkap_transform_enrich_async` - Async enrichment transforms
  - `streamkap_transform_sql_join` - SQL-based join transforms
  - `streamkap_transform_rollup` - Aggregation/rollup transforms
  - `streamkap_transform_fan_out` - Fan-out transforms for multiple outputs
- **Destination Kafka**: New Kafka destination connector
- **Destination Iceberg**: New Iceberg destination connector
- Comprehensive migration guide (`docs/MIGRATION.md`)
- Architecture documentation (`docs/ARCHITECTURE.md`)
- Developer guide (`docs/DEVELOPMENT.md`)

### Changed
- **Code Generation Architecture**: Provider now uses config-driven code generation
  - BaseConnectorResource pattern for sources and destinations
  - BaseTransformResource pattern for transforms
  - Generated schemas from backend configuration files
- Standardized attribute naming across all connectors

### Deprecated
- PostgreSQL source attributes (will be removed in v3.0):
  - `insert_static_key_field_1` → use `transforms_insert_static_key1_static_field`
  - `insert_static_key_value_1` → use `transforms_insert_static_key1_static_value`
  - `insert_static_value_field_1` → use `transforms_insert_static_value1_static_field`
  - `insert_static_value_1` → use `transforms_insert_static_value1_static_value`
  - `insert_static_key_field_2` → use `transforms_insert_static_key2_static_field`
  - `insert_static_key_value_2` → use `transforms_insert_static_key2_static_value`
  - `insert_static_value_field_2` → use `transforms_insert_static_value2_static_field`
  - `insert_static_value_2` → use `transforms_insert_static_value2_static_value`
  - `predicates_istopictoenrich_pattern` → use `predicates_is_topic_to_enrich_pattern`
- Snowflake destination attributes (will be removed in v3.0):
  - `auto_schema_creation` → use `create_schema_auto`

### Breaking Changes
- **PostgreSQL Source**:
  - `database_port`: Type changed from `Int64` to `String`
  - `signal_data_collection_schema_or_database`: Now required (was optional)
- **Snowflake Destination**:
  - `auto_qa_dedupe_table_mapping`: Type changed from `Map` to `String`

### Removed
- Legacy connector implementation files (replaced by generated code)

## [1.x.x] - Previous Releases

See [GitHub Releases](https://github.com/streamkap-com/terraform-provider-streamkap/releases) for previous versions.
```

**Step 2: Commit changelog**

```bash
git add CHANGELOG.md
git commit -m "docs: add CHANGELOG.md for v2.0.0 release

Documents all additions, changes, deprecations, and breaking changes.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Summary Checklist

- [ ] Task 0: Previous plans verified complete
- [ ] Task 1: Deprecated aliases helper created (with unit tests)
- [ ] Task 2: PostgreSQL deprecated aliases added
- [ ] Task 3: Snowflake deprecated alias added
- [ ] Task 4: Migration guide created (docs/MIGRATION.md)
- [ ] Task 5: Transform acceptance tests added (3 types)
- [ ] Task 5.5: Destination Kafka acceptance test added
- [ ] Task 6: CLAUDE.md updated
- [ ] Task 7: README.md updated
- [ ] Task 8: DEVELOPMENT.md updated
- [ ] Task 9: ARCHITECTURE.md updated
- [ ] Task 10: .env.example updated
- [ ] Task 11: .gitignore updated, docs committed
- [ ] Task 12: Final verification passed
- [ ] Task 13: CHANGELOG.md created

---

## Customer Impact Summary

**Q: If we merge this, will customers feel any difference?**

**A: Minimal disruption with deprecated aliases in place.**

### What Customers Experience:

1. **Deprecated Attributes (10 total):**
   - Old names STILL WORK
   - Deprecation warnings shown during `terraform plan/apply`
   - Users can migrate at their own pace before v3.0

2. **Breaking Changes (3 total - require immediate action):**
   - `database_port` type change: Integer → String
   - `signal_data_collection_schema_or_database` now required
   - `auto_qa_dedupe_table_mapping` type change: Map → String

3. **New Features:**
   - 6 Transform resource types
   - Improved schema validation

### Customer Action Required:

1. Read `docs/MIGRATION.md`
2. Fix 3 breaking changes (5 minutes)
3. (Optional) Migrate deprecated attributes to avoid warnings
4. Run `terraform plan` to verify
