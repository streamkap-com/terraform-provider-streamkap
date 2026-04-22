# Migration Guide

This guide helps existing users migrate their Terraform configurations between major versions.

---

## v2.x to v3.0

> **v3.0.0-beta.1 is available for testing.** This is a pre-release — do not use it in
> production. If you are on v2.x and your setup is working, **there is no need to migrate
> yet**. Wait for the stable v3.0.0 release. The beta may introduce further breaking changes
> before the final release.

### What's New in v3.0

v3.0 adds new resource types and data sources on top of everything in v2.x:

- **`streamkap_destination_weaviate`** — Weaviate vector database destination connector
- **`streamkap_kafka_user`** — Kafka user management with ACL-based topic access control
- **`streamkap_client_credential`** — API token management for machine-to-machine authentication
- **`streamkap_roles` data source** — List available roles for client credential assignment

### Deprecated Attribute Removal (Planned)

v3.0 is planned to **remove** the deprecated attributes introduced in v2.0. These attributes currently still work with deprecation warnings in the beta, but will be removed before the stable release:

| Resource | Deprecated Attribute | Replacement |
|----------|---------------------|-------------|
| PostgreSQL Source | `insert_static_key_field_1` | `transforms_insert_static_key1_static_field` |
| PostgreSQL Source | `insert_static_key_value_1` | `transforms_insert_static_key1_static_value` |
| PostgreSQL Source | `insert_static_value_field_1` | `transforms_insert_static_value1_static_field` |
| PostgreSQL Source | `insert_static_value_1` | `transforms_insert_static_value1_static_value` |
| PostgreSQL Source | `insert_static_key_field_2` | `transforms_insert_static_key2_static_field` |
| PostgreSQL Source | `insert_static_key_value_2` | `transforms_insert_static_key2_static_value` |
| PostgreSQL Source | `insert_static_value_field_2` | `transforms_insert_static_value2_static_field` |
| PostgreSQL Source | `insert_static_value_2` | `transforms_insert_static_value2_static_value` |
| PostgreSQL Source | `predicates_istopictoenrich_pattern` | `predicates_is_topic_to_enrich_pattern` |
| Snowflake Destination | `auto_schema_creation` | `create_schema_auto` |

**Action required before v3.0 stable:** If you use any of the deprecated attribute names listed above, rename them to the new names now. They will stop working in the final v3.0.0 release.

### Trying the Beta

If you want to test the beta in a non-production environment:

```hcl
terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = "3.0.0-beta.1"
    }
  }
}
```

> **Important:** Pin to the exact beta version (`"3.0.0-beta.1"`), not a range like `">= 3.0.0"`. This prevents accidentally pulling a future beta or the stable release before you're ready.

### Reporting Issues

If you encounter problems with the beta, please report them:
1. [GitHub Issues](https://github.com/streamkap-com/terraform-provider-streamkap/issues)
2. Include your provider version, Terraform version, and relevant config (redact secrets)

---

## v1.x to v2.0

### Backward Compatibility

**Good news!** Most attribute renames have **deprecated aliases** that provide backward compatibility. Your existing configurations will continue to work, but you'll see deprecation warnings.

#### Deprecated Attributes (Still Work, But Migrate Soon)

These old attribute names still work but show deprecation warnings:

##### PostgreSQL Source

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

##### MySQL Source

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
| `database_connection_timezone` | `database_connection_time_zone` | Rename in config |

##### MongoDB Source

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
| `array_encoding` | `transforms_unwrap_array_encoding` | Rename in config |
| `nested_document_encoding` | `transforms_unwrap_document_encoding` | Rename in config |

##### SQL Server Source

The v2.1.19 schema had a single `insert_static_*` pair without a numeric suffix. v3.x
uses the `_1` suffix to align with the other connectors. All aliases below work with
deprecation warnings.

| Deprecated (Old) Name | New Name | Action |
|-----------------------|----------|--------|
| `insert_static_key_field` | `transforms_insert_static_key1_static_field` | Rename in config |
| `insert_static_key_value` | `transforms_insert_static_key1_static_value` | Rename in config |
| `insert_static_value_field` | `transforms_insert_static_value1_static_field` | Rename in config |
| `insert_static_value` | `transforms_insert_static_value1_static_value` | Rename in config |
| `snapshot_parallelism` | `streamkap_snapshot_parallelism` | Rename in config |
| `snapshot_large_table_threshold` | `streamkap_snapshot_large_table_threshold` | Rename in config |

##### KafkaDirect Source

| Deprecated (Old) Name | New Name | Action |
|-----------------------|----------|--------|
| `kafka_format` | `format` | Rename in config |

##### Snowflake Destination

| Deprecated (Old) Name | New Name | Action |
|-----------------------|----------|--------|
| `auto_schema_creation` | `create_schema_auto` | Rename in config |

> **Note:** Deprecated attributes work with a warning during v3.x and are
> scheduled for removal in the next major version (v4.0). Migrate at your
> convenience; no immediate action is required.

#### Non-Aliasable Renames (Config Edit Required)

The following v2.1.19 attributes cannot be aliased because the underlying API contract
changed. If your v2.1.19 configuration uses them, rename them before upgrading to v3.x.

##### SQL Server Source

| v2.1.19 Name | v3.x Name | Why no alias |
|--------------|-----------|---------------|
| `database_dbname` (string) | `database_names` (comma-separated) | Backend API field changed from `database.dbname` to `database.names` to support multiple databases per connector. |
| `snapshot_custom_table_config` (map of objects) | `streamkap_snapshot_custom_table_config` (JSON string) | Type changed from `map<string, {chunks: int}>` to a JSON-serialized string, so the attribute's type definition cannot be aliased. |

##### DynamoDB Source

| v2.1.19 Name | v3.x Name | Why no alias |
|--------------|-----------|---------------|
| `table_include_list_user_defined` | `table_include_list` | v2.1.19 field was `Required`, so a deprecated alias would still force a plan-time choice between names. A straight rename is the cleanest migration. |

### Breaking Changes (Require Immediate Action)

These changes do NOT have backward compatibility and require updates:

#### Newly Required Fields (added in v3.x)

Some destination fields that were optional in v2.1.x have become required in the
current backend schema. Update your `.tf` files to set them explicitly before
upgrading, or `terraform plan` will fail with "Missing required argument".

| Resource | Field | Action |
|----------|-------|--------|
| `streamkap_destination_cockroachdb` | `database_database` | Add `database_database = "<db_name>"` |
| `streamkap_destination_databricks` | `connection_url` | Add `connection_url = "<JDBC_URL>"` |
| `streamkap_destination_databricks` | `databricks_token` (sensitive) | Add `databricks_token = "<TOKEN>"`. Keep the value out of source control — source from a variable or secret manager. |
| `streamkap_destination_clickhouse` | `database` | Add `database = "<db_name>"` (was Optional with a default in v2.1.x). |

Example for Databricks:

```hcl
resource "streamkap_destination_databricks" "example" {
  name             = "my-warehouse"
  hostname         = "dbc-xxxx.cloud.databricks.com"
  connection_url   = var.databricks_jdbc_url    # now required
  databricks_token = var.databricks_token       # now required, sensitive
  # ...existing fields...
}
```

#### `ssh_public_key` is server-assigned — don't set it in Terraform

For every source/destination that supports SSH tunnelling (`streamkap_source_*`
with `ssh_enabled`, `streamkap_destination_postgresql`, etc.), the
`ssh_public_key` attribute is **computed** — the backend generates and returns
the real key, and the provider stores it in Terraform state. v2.1.x and early
v3.x betas (<= beta.9) emitted a placeholder default (`"<SSH.PUBLIC.KEY>"`)
that caused a "provider produced inconsistent result after apply" error on
first apply (GitHub issue #72).

**Action:**
- If you previously set `ssh_public_key = "<SSH.PUBLIC.KEY>"` or any explicit
  value in your `.tf` files, **remove the line**. Leave the attribute unset.
- If an old state file already contains the literal `"<SSH.PUBLIC.KEY>"` (from
  the bug), the next `terraform apply` will show state drift once the backend
  returns the real key; accept the drift. A `terraform refresh` before the
  apply is also safe.
- If you truly need to set a specific SSH public key, work directly with
  Streamkap support — the backend currently overrides user-provided values.

#### PostgreSQL Source

##### Type Change: database_port

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

##### New Required Field

`signal_data_collection_schema_or_database` is now required:

```hcl
resource "streamkap_source_postgresql" "example" {
  signal_data_collection_schema_or_database = "public"
}
```

#### Snowflake Destination

##### Type Change: auto_qa_dedupe_table_mapping

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

### Default Value Changes

These affect NEW resources only. Existing resources are unaffected:

| Resource | Attribute | Old Default | New Default |
|----------|-----------|-------------|-------------|
| PostgreSQL Source | `heartbeat_enabled` | `false` | `true` |
| Snowflake Destination | `hard_delete` | `false` | `true` |

### Migration Steps

#### Step 1: Backup Your State

```bash
cp terraform.tfstate terraform.tfstate.backup
```

#### Step 2: Fix Breaking Changes

Update these in your `.tf` files:
1. Change `database_port = 5432` to `database_port = "5432"`
2. Add `signal_data_collection_schema_or_database = "public"` if missing
3. Change map-style `auto_qa_dedupe_table_mapping` to string format

#### Step 3: (Optional) Fix Deprecated Attributes

While not required immediately, update deprecated attribute names to avoid warnings:
- Rename attributes as shown in the tables above

#### Step 4: Verify

```bash
terraform init -upgrade
terraform plan
```

You should see:
- No errors for breaking changes (if fixed)
- Deprecation warnings for old attribute names (if not yet migrated)

#### Step 5: Apply

```bash
terraform apply
```

### New Features in v2.0

#### Transform Resources

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

#### New Destination Connectors

- `streamkap_destination_kafka` - Kafka destination
- `streamkap_destination_iceberg` - Iceberg destination

## Deprecation Timeline

| Version | Status |
|---------|--------|
| v2.0 | Deprecated attributes work with warnings |
| v2.x | Deprecated attributes continue to work |
| v3.0.0-beta.1 | Deprecated attributes still work (pre-release, not for production) |
| v3.0.0 (stable) | Deprecated attributes **REMOVED** |

## Getting Help

If you encounter issues:
1. Check [GitHub Issues](https://github.com/streamkap-com/terraform-provider-streamkap/issues)
2. Open a new issue with your error message and config (redact secrets)
