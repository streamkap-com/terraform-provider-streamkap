---
page_title: "Migrate from v2 to v3"
subcategory: ""
description: |-
  Upgrade the Streamkap Terraform provider from v2 to v3 while preserving existing resources and state.
---

# Migrate from v2 to v3

This guide covers **v2.1.21 and later v2 releases (2.1.x and 2.2.x) → v3.0.0**.
Keep your existing Terraform state, resource addresses, and Streamkap resource
IDs. Schema and state inspection indicates no intermediate v2.2.x upgrade is
needed, but the direct 2.1.x path has not been exercised in the migration
suite. Follow the configuration changes and plan checks below before applying.

v3 adds connectors and access-management resources, and generates connector
schemas from the Streamkap API configuration. Some v2 attributes have compatibility
aliases; others were removed or renamed, and some defaults changed.

- [Upgrade steps](#upgrade-steps)
- [Required configuration changes](#required-configuration-changes)
- [Defaults to review](#defaults-to-review)
- [Optional attribute renames](#optional-attribute-renames)
- [Troubleshooting and rollback](#troubleshooting-and-rollback)
- [Upgrading from a v3 beta](#upgrading-from-a-v3-beta)

v2 receives bug and security fixes only through **15 October 2026**; support
ends **16 October 2026**. Until ready, keep a v2-only version constraint, such
as `~> 2.1.21` to stay on 2.1 patches or `~> 2.2` for the 2.2 line.

## Upgrade steps

Repeat these steps for each Terraform root module and workspace. Test first in
a non-production environment with its own resources and state.

### 1. Save the current configuration and state

Pause automated applies. Keep a copy of the current `.tf` files and
`.terraform.lock.hcl`. In the initialized working directory, confirm the
workspace and take a state backup:

```bash
terraform version
terraform providers
terraform workspace show
terraform state list
umask 077
terraform state pull > terraform-v2-backup.tfstate
```

Check that the backup is nonempty. State and saved plans can contain credentials;
keep them out of Git and public issue reports. Use a separate backup per workspace.

### 2. Update the version and required configuration

Keep authentication unchanged. For an existing Registry installation
(`registry.terraform.io/streamkap-com/streamkap` in `terraform providers`), set
the version in your existing `required_providers` block:

```hcl
terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = "3.0.0"
    }
  }
}
```

If you followed an older example with `source = "github.com/streamkap-com/streamkap"`,
check `terraform providers`. When that exact address is present in state, use
your backup from step 1 and migrate the address before upgrading:

```bash
terraform state replace-provider \
  'github.com/streamkap-com/streamkap' \
  'registry.terraform.io/streamkap-com/streamkap'
terraform providers
```

Review the confirmation prompt: this rewrites the provider address in state.
Update the source address in all affected modules to `streamkap-com/streamkap`.
Skip the command if state already uses the Registry address. If your address
differs from both, stop and contact support before changing state.

Apply the [required changes](#required-configuration-changes) for the resources
you use, including references in outputs and modules. Set explicit values for
[defaults](#defaults-to-review) whose old behavior you need to preserve. Update
any child-module constraints that exclude v3.

Keep resource block names, module paths, and `for_each` keys unchanged. Leave
[supported aliases](#optional-attribute-renames) in place for this first upgrade;
you can rename them separately later.

### 3. Initialize, validate, and review the plan

```bash
terraform init -upgrade
terraform fmt -check -recursive
terraform validate
terraform plan -out=streamkap-v3.tfplan
terraform show streamkap-v3.tfplan
```

`init -upgrade` can upgrade other providers and modules within their constraints.
Review `.terraform.lock.hcl` and keep unrelated dependency upgrades separate.

Review every changed resource:

- A no-op or in-place update can be expected. Added defaults can produce updates
  even when you did not change the resource configuration.
- Check heartbeat, deletes, table/topic selection, credentials, and destination
  paths. An in-place update can still change data behavior.
- **Stop on unexpected creation, replacement, or deletion**, including destruction
  of a deposed instance. Do not use `state rm`, import, or `ignore_changes` just
  to hide a migration error.

Resolve errors and regenerate the plan after any configuration edit.

### 4. Apply the reviewed plan and check the result

```bash
terraform apply streamkap-v3.tfplan
terraform plan
```

The follow-up plan should have no unexplained changes. Confirm resource IDs are
unchanged, then check connector/pipeline health and data delivery in Streamkap.
Commit the updated configuration and lock file; resume automated applies after
verification. For remote execution, use your normal reviewed run and approval
workflow instead of the local saved-plan commands.

## Required configuration changes

Only edit the resources and attributes you use. The names below are Terraform
attribute names, not the dotted API configuration keys.

### Attribute types

Update literals and module variable types where necessary:

| Resource | Attribute | v2 → v3 |
|----------|-----------|---------|
| PostgreSQL, MySQL, MongoDB, SQL Server sources; PostgreSQL destination | `ssh_port` | String → number: `"22"` → `22` |
| MySQL source | `snapshot_gtid` | Boolean → string: `true` → `"Yes"`, `false` → `"No"` |
| Databricks destination | `consumer_wait_time_for_larger_batch_ms` | Number → string: `500` → `"500"` |

Databricks batch wait now accepts only `"500"`, `"5000"`, `"10000"`, `"20000"`,
`"30000"`, `"60000"`, `"120000"`, `"180000"`, `"240000"`, or `"300000"`.
Its default increases from 500 ms to 10,000 ms; set `"500"` to preserve the old
wait time. A longer wait favors larger batches over lower latency.

### SQL Server source

For `streamkap_source_sqlserver`:

- Rename `database_dbname` to `database_names`, preserving the database selection
  as a comma-separated string.
- Remove `snapshot_large_table_threshold` and `snapshot_custom_table_config`.
  There is no replacement for either field. Use `streamkap_snapshot_parallelism`
  (or its `snapshot_parallelism` alias) for snapshot concurrency.

### DynamoDB source

For `streamkap_source_dynamodb`, rename `table_include_list_user_defined` to
`table_include_list`, preserving the selected tables.

### ClickHouse destination

Set `streamkap_destination_clickhouse.database` explicitly to the existing
database name. It was optional in v2 and is required in v3.

### PostgreSQL destination

| v2 name | v3.x name |
|---------|-----------|
| `database_dbname` | `database_database` |
| `database_username` | `connection_username` |
| `database_password` | `connection_password` |
| `database_schema_name` | `table_name_prefix` |
| `hard_delete` | `delete_enabled` |
| `custom_primary_key` | `primary_key_fields` |

Preserve the configured values, including deletion behavior and primary keys.
`table_name_prefix` is the schema name (for example, `public`), not a full
table name or a prefix with a trailing dot.

### Iceberg destination

| v2 name | v3.x name |
|---------|-----------|
| `catalog_type` | `iceberg_catalog_type` |
| `catalog_name` | `iceberg_catalog_name` |
| `catalog_uri` | `iceberg_catalog_uri` |
| `aws_access_key` | `iceberg_catalog_s3_access_key_id` |
| `aws_secret_key` | `iceberg_catalog_s3_secret_access_key` |
| `aws_iam_role` | `iceberg_catalog_client_assume_role_arn` |
| `aws_region` | `iceberg_catalog_client_region` |
| `bucket_path` | `iceberg_catalog_warehouse` |
| `schema` | `table_name_prefix` |
| `primary_key_fields` | `iceberg_tables_default_id_columns` |

For REST or Hive catalogs, when supplying your own S3 credentials also set
`iceberg_catalog_s3_credentials_enabled = true`. Its default is `false`, which
uses catalog-vended credentials. The access-key ID is now sensitive, so outputs
that expose it must also be sensitive. Changing `iceberg_catalog_type` requires
replacement in v3; inspect the plan before applying a catalog change.
`iceberg_catalog_warehouse` is the warehouse identifier: retain the S3 path
for S3-based catalogs; other catalogs may require a warehouse name.
The new `iceberg_tables_hard_delete_enabled` setting defaults to `true`; set it
explicitly to match your intended deletion behavior.

### S3 destination

| v2 name | v3.x name |
|---------|-----------|
| `aws_access_key` | `aws_access_key_id` |
| `aws_secret_key` | `aws_secret_access_key` |
| `aws_region` | `aws_s3_region` |
| `bucket_name` | `aws_s3_bucket_name` |
| `filename_template` | `file_name_template` |
| `compression_type` | `file_compression_type` |
| `output_fields` | `format_output_fields` |

`filename_prefix` was removed. Incorporate the intended prefix into
`file_name_template` and review the resulting object paths before applying.

### Outputs and data sources

`data.streamkap_tag.type` changes from a list to a set. Replace indexed access
such as `data.streamkap_tag.example.type[0]` with membership checks:

```hcl
locals {
  is_source_tag = contains(data.streamkap_tag.example.type, "sources")
}
```

Use `one(data.streamkap_tag.example.type)` only if exactly one type is expected;
set order has no meaning.

DynamoDB `aws_access_key_id` and Iceberg `iceberg_catalog_s3_access_key_id`
are sensitive in v3. Mark root outputs exposing them `sensitive = true`.

```hcl
output "access_key_id" {
  value     = streamkap_source_dynamodb.example.aws_access_key_id
  sensitive = true
}
```

Sensitive values remain in state and can be exposed by JSON or raw output.

### Server-assigned fields and unchanged shapes

Leave `ssh_public_key` unset and read the resource output; the backend supplies
this value. Remove literal `"<SSH.PUBLIC.KEY>"` placeholders. Normally also leave
MongoDB `mongodb_connection_hostname` unset so it can follow the connection string.

PostgreSQL source `database_port` is already numeric in v2.1.21. If you configure
`signal_data_collection_schema_or_database`, use the full signal-table path,
such as `public.streamkap_signal`, rather than the old schema-only default `public`.

Snowflake `auto_qa_dedupe_table_mapping` and ClickHouse `topics_config_map`
remain maps. Do not convert them to JSON strings. Pipeline `source` and
`destination` remain nested objects; keep their resource references.

## Defaults to review

Set explicit values if you want to preserve the old behavior. These are notable
changes, not an exhaustive list of every new default; inspect your actual plan.

| Resource | Attribute | v2 default | v3 default |
|----------|-----------|------------|------------|
| PostgreSQL, MySQL, SQL Server sources | `heartbeat_enabled` | `false` | `true` |
| Snowflake destination | `hard_delete` | `false` | `true` |
| DynamoDB source | `poll_timeout_ms` | `1000` | `180000` |
| DynamoDB source | `incremental_snapshot_chunk_size` | `32768` | `8192` |
| Databricks destination | `ingestion_mode` | `"append"` | `"upsert"` |
| Databricks destination | `consumer_wait_time_for_larger_batch_ms` | `500` (number) | `"10000"` (string) |

Snowflake's default `create_sql_execute` template also changed. If you use
`apply_dynamic_table_script = true`, review that SQL and set it explicitly if
you need to preserve the previous script.

New defaulted fields can also plan an in-place update: for example, Kafka Direct
source `records_carry_streamkap_metadata = false`, or Databricks destination
`connection_timeout = 180` and `preserve_null_values = false`.

`tags` is optional on connectors in v3. Leave it unset to preserve tags managed
outside Terraform. Setting `tags = [...]` makes Terraform manage them;
`tags = []` clears them.

## Optional attribute renames

These aliases remain supported throughout v3.x and are planned for removal in
v4. Rename them separately from the initial upgrade. **Use either the old name
or the new name, never both**, and preserve the value.

For PostgreSQL, MySQL, and MongoDB sources, the following mappings apply to
both suffixes `1` and `2`:

| v2 name (`N` = `1` or `2`) | v3 name |
|---------------------------|---------|
| `insert_static_key_field_N` | `transforms_insert_static_keyN_static_field` |
| `insert_static_key_value_N` | `transforms_insert_static_keyN_static_value` |
| `insert_static_value_field_N` | `transforms_insert_static_valueN_static_field` |
| `insert_static_value_N` | `transforms_insert_static_valueN_static_value` |

For example, `insert_static_value_1` becomes
`transforms_insert_static_value1_static_value`.

| Resource | v2 name | v3 name |
|----------|---------|---------|
| PostgreSQL, MySQL, MongoDB sources | `predicates_istopictoenrich_pattern` | `predicates_is_topic_to_enrich_pattern` |
| MySQL source | `database_connection_timezone` | `database_connection_time_zone` |
| MongoDB source | `array_encoding` | `transforms_unwrap_array_encoding` |
| MongoDB source | `nested_document_encoding` | `transforms_unwrap_document_encoding` |
| SQL Server source | `insert_static_key_field` | `transforms_insert_static_key1_static_field` |
| SQL Server source | `insert_static_key_value` | `transforms_insert_static_key1_static_value` |
| SQL Server source | `insert_static_value_field` | `transforms_insert_static_value1_static_field` |
| SQL Server source | `insert_static_value` | `transforms_insert_static_value1_static_value` |
| SQL Server source | `snapshot_parallelism` | `streamkap_snapshot_parallelism` |
| Kafka Direct source | `kafka_format` | `format` |
| Snowflake destination | `auto_schema_creation` | `create_schema_auto` |

## Troubleshooting and rollback

**Unsupported argument or missing required argument:** use the tables above and
check your modules as well as the root configuration. For exact types and valid
values, consult the [v3.0.0 resource reference](https://registry.terraform.io/providers/streamkap-com/streamkap/3.0.0/docs)
or run `terraform providers schema -json` after initialization.

**SQL Server or S3 v2 errors:** older v2 releases can report an unexpected null
for `snapshot_large_table_threshold` or `filename_prefix` during create, refresh,
or update. v2.2.1 fixes these errors but does not restore the removed settings.
Remove the obsolete attributes when moving to v3. If planning still fails, retain
your state backup and contact support; recreating the resource is not a migration
step.

**Resource already exists:** v3 does not automatically adopt sources,
destinations, pipelines, or transforms on a name collision. A replacement using
`create_before_destroy` needs a distinct remote name for the two objects to
coexist. Changing Terraform's local block label does not change that name.
If current and deposed instances share the same backend ID, stop before applying:
destroying the deposed instance can delete the live object. Contact support for
state recovery. Import is appropriate only when the remote object is missing
from state, not as a routine upgrade step.

**Abandoning an upgrade:** before any v3 apply or other state-writing operation,
restore the old configuration and lock file, discard the saved plan, and run
`terraform init` without `-upgrade`. After v3 has written state or changed remote
resources, changing the version pin alone is not a safe rollback. Keep both state
snapshots and contact support before restoring state; an old snapshot does not
undo remote changes.

**Validation scope:** automated migration tests use selected v2.2.0
configurations, with gaps for SQL Server and S3. They do not cover every
connector or configuration.

For help, contact Streamkap support or open a
[GitHub issue](https://github.com/streamkap-com/terraform-provider-streamkap/issues)
with the old/new provider versions, Terraform version, resource types, and a
redacted error or plan excerpt. Do not attach raw state, saved plans, or credentials.

## Upgrading from a v3 beta

Skip this section when upgrading from v2.

- Mark root outputs of webhook `api_key` or HTTP sink
  `http_headers_authorization` as `sensitive = true`.
- **beta.22–26:** replace `post_processors = "reselector"` with
  `post_processors_reselect_enabled = true` on PostgreSQL, AlloyDB, and Supabase.
  The toggle defaults to `true` there; Oracle and Oracle AWS default to `false`.
- Remove PostgreSQL `streamkap_snapshot_large_table_threshold` and
  `streamkap_snapshot_custom_table_config`, and SQL Server
  `streamkap_snapshot_large_table_threshold` if present.
- **beta.16 and earlier:** use maps for Snowflake `auto_qa_dedupe_table_mapping`
  and ClickHouse `topics_config_map`, not JSON strings.
- `streamkap_topic_metrics` legacy throughput, lag, and latency fields are now
  deprecated null values; `time_interval` and `time_unit` are ignored. Use the
  [topic metrics reference](https://registry.terraform.io/providers/streamkap-com/streamkap/3.0.0/docs/data-sources/topic_metrics)
  for the replacement fields and required topic database IDs. The `streamkap_topics`
  fields `messages_7d` and `messages_30d` are also deprecated and always null.
