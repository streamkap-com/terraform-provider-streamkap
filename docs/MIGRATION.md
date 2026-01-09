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

### New Destination Connectors

- `streamkap_destination_kafka` - Kafka destination
- `streamkap_destination_iceberg` - Iceberg destination

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
