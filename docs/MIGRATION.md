# Migration Guide

This guide helps existing users migrate their Terraform configurations between major versions.

---

## v2.x to v3.0

> **v3.0 is in beta.** These examples target `3.0.0-beta.32`; confirm it is
> listed on the Terraform Registry before installing. Do not use beta releases
> in production. If your v2 setup is working, wait for the stable v3.0.0 release.
> The beta may introduce further breaking changes before the final release.

### v2 maintenance policy

When v3.0.0 becomes stable, v2 becomes the legacy maintenance line. It receives
bug fixes and security patches only, through **15 October 2026**. New features
and connectors are v3-only. From **16 October 2026**, v2 receives no further
fixes or support. Published v2 versions remain available, but availability does
not guarantee compatibility with future backend changes.

Keep a constraint such as `version = "~> 2.2"` until you are ready to migrate.
An unconstrained configuration, or one without an upper bound below 3.0, can
select v3 stable on `terraform init -upgrade` or on a fresh init without a lock
file. Back up state, make the edits below, and review the plan before applying.

### What's New in v3.0

New resources and data sources in v3.0 include:

- **`streamkap_destination_weaviate`** — Weaviate vector database destination connector
- **`streamkap_destination_pinecone`** — Pinecone vector database destination connector
- **`streamkap_source_informix`** — Informix CDC source connector
- **`streamkap_source_salesforce_webhook`** — Salesforce webhook source connector
- **`streamkap_source_shopify_webhook`** — Shopify webhook source connector
- **`streamkap_source_stripe_webhook`** — Stripe webhook source connector
- **`streamkap_source_zendesk_webhook`** — Zendesk webhook source connector
- **`streamkap_transform_topic_router`** — Topic router transform
- **`streamkap_kafka_user`** — Kafka user management with ACL-based topic access control
- **`streamkap_client_credential`** — API token management for machine-to-machine authentication
- **`streamkap_roles` data source** — List available roles for client credential assignment
- **`streamkap_tags` data source** — Filter/list tags by name, type, or IDs (alternative to single-by-id `streamkap_tag`)

#### Tags everywhere (additive)

v3.0 surfaces `tags = [...]` (Set of tag IDs) on **every** entity that supports
tagging upstream — sources, destinations, transforms, topics, and pipelines.
Pipelines already had `tags`; the rest are new in v3.

The attribute is **Optional + Computed**:

- Unset in config → the provider preserves whatever tags the backend reports,
  so out-of-band tag attachments (e.g. system-managed environment tags) do not
  show as drift.
- Explicitly set → Terraform owns the field; manual edits made via the
  Streamkap UI show as drift on the next plan and revert on apply.
- Set to `tags = []` → clears all tags on the entity. The provider
  distinguishes `null` from empty-list on the wire so this works correctly.

This is purely additive — existing v2 configs that do not reference `tags`
are intended to retain their tag behavior when re-planned against v3.
The migration tests use v2.2.0 as the stable baseline and cover representative
configurations; other breaking changes
listed in this guide still require configuration updates.

```hcl
resource "streamkap_tag" "prod" {
  name = "production"
  type = ["sources", "destinations", "pipelines"]
}

resource "streamkap_source_postgresql" "orders" {
  # ...
  tags = [streamkap_tag.prod.id]
}
```

The standalone `streamkap_tag` resource also gains plan-time validation of
`type` against the backend `TagTypeEnum`.

#### Breaking — `data.streamkap_tag.type` is now a Set, not a List

In v2, the `streamkap_tag` **data source** returned `type` as a `List[String]`,
which meant you could index into it (`data.streamkap_tag.X.type[0]`). v3
returns `type` as a `Set[String]` to match the `streamkap_tag` **resource**.
Sets are unordered and not indexable.

If your HCL references `type` by index, replace it with one of:

```hcl
# v2 — broken in v3
locals { tag_type = data.streamkap_tag.example.type[0] }

# v3 — pick any one element from the set
locals { tag_type = tolist(data.streamkap_tag.example.type)[0] }

# Or just use `contains()` if you were checking membership
locals { is_source_tag = contains(data.streamkap_tag.example.type, "sources") }
```

Element type is unchanged (`string`); only the collection type changed.

#### Breaking — credential attributes are now sensitive

These attributes are now marked `Sensitive`. Previously their values were
printed in plan output and written to logs in the clear:

| Resource | Attribute |
|----------|-----------|
| `streamkap_source_webhook`, `streamkap_source_shopify_webhook`, `streamkap_source_stripe_webhook`, `streamkap_source_salesforce_webhook`, `streamkap_source_zendesk_webhook` | `api_key` |
| `streamkap_destination_httpsink` | `http_headers_authorization` |
| `streamkap_source_dynamodb` | `aws_access_key_id` |
| `streamkap_destination_iceberg` | `iceberg_catalog_s3_access_key_id` (formerly `aws_access_key`) |

Terraform refuses to expose a sensitive value through a root module output
unless the output itself is marked sensitive, so this config now fails at plan
time with *"Output refers to sensitive values"*:

```hcl
# v3.0.0-beta.24 and earlier — errors after upgrading
output "webhook_key" {
  value = streamkap_source_webhook.example.api_key
}

# fixed
output "webhook_key" {
  value     = streamkap_source_webhook.example.api_key
  sensitive = true
}
```

The value is still readable — `terraform output -raw webhook_key`, or
`terraform show -json` — so you can still copy it into Shopify or Stripe when
registering the webhook URL. `Sensitive` only stops it appearing in plan diffs
and logs. Passing `api_key` into another module needs no change.

### Deprecated Attribute Removal (Planned)

The deprecated attributes below keep working, with a deprecation warning, for the
whole of v3.x — including v3.0.0 stable. They are removed in **v4.0**.

Removing them at v3.0.0 stable would defeat their purpose: they exist so a v2
configuration can move to v3 without a rewrite. Migrate to the replacement names
at your own pace during v3.x.

See [the alias tables](#deprecated-attributes-still-work-but-migrate-soon) for replacements.

### Trying the Beta

If you want to test the beta in a non-production environment:

```hcl
terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = "3.0.0-beta.32"
    }
  }
}
```

> **Important:** Pin the **exact** beta version you tested, not a range like
> `">= 3.0.0"` or `"~> 3.0.0-beta"`. Each beta may introduce further breaking
> changes; an exact pin stops Terraform pulling a newer beta — or the stable
> release — before you're ready. Check the
> [Terraform Registry](https://registry.terraform.io/providers/streamkap-com/streamkap/latest)
> for the latest `3.0.0-beta.x`.

### Reporting Issues

If you encounter problems with the beta, please report them:
1. [GitHub Issues](https://github.com/streamkap-com/terraform-provider-streamkap/issues)
2. Include your provider version, Terraform version, and relevant config (redact secrets)

---

### Backward Compatibility

The aliases below preserve v2 attribute names through v3.x. Other removals, type
changes and newly required fields still require configuration updates.

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

#### Renames Without Compatibility Aliases (Config Edit Required)

These attributes from v2.2.0 have no compatibility aliases in v3. Rename them
before upgrading. Review the semantic changes alongside each table.

##### SQL Server Source

| v2 name | v3.x name | Notes |
|--------------|-----------|---------------|
| `database_dbname` (string) | `database_names` (comma-separated) | The Terraform name now reflects the plural backend field `database.names`, which v2.2.0 already used. Set the same database selection under the new name. |

##### DynamoDB Source

| v2 name | v3.x name | Notes |
|--------------|-----------|---------------|
| `table_include_list_user_defined` | `table_include_list` | Both names map to `table.include.list.user.defined`. Preserve the configured table selection. |

##### PostgreSQL Destination

| v2 name | v3.x name |
|---------|-----------|
| `database_dbname` | `database_database` |
| `database_username` | `connection_username` |
| `database_password` | `connection_password` |
| `database_schema_name` | `table_name_prefix` |
| `hard_delete` | `delete_enabled` |
| `custom_primary_key` | `primary_key_fields` |

Preserve the configured values, including deletion behavior and primary keys.
`ssh_port` is now numeric; use `ssh_port = 22`.

##### Iceberg Destination

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

When supplying your own AWS credentials, also set
`iceberg_catalog_s3_credentials_enabled = true`. Its default is `false`, which
uses catalog-vended credentials. The access-key ID is now sensitive, so outputs
that expose it must also be sensitive. Changing `iceberg_catalog_type` requires
replacement in v3; inspect the plan before applying a catalog change.
The new `iceberg_tables_hard_delete_enabled` setting defaults to `true`; set it
explicitly to match your intended deletion behavior.

##### S3 Destination

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

#### Attributes Removed by the Backend (Config Edit Required)

A production backend release dropped these connector config fields. There is no
replacement attribute and no alias — remove them from your configuration.

| Resource | Removed attribute | Notes |
|----------|-------------------|-------|
| `streamkap_source_postgresql` | `streamkap_snapshot_large_table_threshold` | Early-v3-beta field, absent from v2.2.0. Backend dropped `streamkap.snapshot.large.table.threshold`. |
| `streamkap_source_postgresql` | `streamkap_snapshot_custom_table_config` | Early-v3-beta field, absent from v2.2.0. Backend dropped the field. |
| `streamkap_source_sqlserver` | `snapshot_large_table_threshold` | The v2 attribute and its early-v3 replacement `streamkap_snapshot_large_table_threshold` are removed. |
| `streamkap_source_sqlserver` | `snapshot_custom_table_config` | The backend has no `streamkap.snapshot.custom.table.config.user.defined` field, so **every value ever set here was silently discarded** — it never reached the connector. Per-table chunk counts are no longer configurable; the backend sizes chunks itself. Use `snapshot_parallelism` (and, if needed, `streamkap_snapshot_chunk_size_bytes`) to tune snapshot throughput. |

`streamkap_snapshot_parallelism` is unaffected and keeps its
`snapshot_parallelism` alias.

> **Why `snapshot_custom_table_config` was a no-op.** The provider generated the
> attribute from a `cmd/tfgen/overrides.json` entry rather than from the backend
> plugin spec, so nothing ever checked that the API field it mapped to still
> existed. It did not. Terraform accepted the value, sent it, and the backend
> ignored the unknown key. tfgen now fails the build when an override targets a
> field the backend does not declare, so this class of dead attribute cannot
> ship again.

### Breaking Changes (Require Immediate Action)

These changes do NOT have backward compatibility and require updates:

#### Newly Required Fields

`streamkap_destination_clickhouse.database` was optional in v2.2.0 and is
required in v3. Set it explicitly before upgrading, using the existing database
name.

Databricks `connection_url` and `databricks_token` were already required in
v2.2.0. CockroachDB is a new destination in v3, so its required fields are not
changes to an existing v2 resource.

#### SSH public keys and server-assigned fields

`ssh_public_key` remains optional and computed, but the backend resolves it
server-side and ignores whatever the configuration sends — on create it returns
the tenant tunnel key, and on update it returns the already-stored key. Leave it
unset and read the resource output. Setting it has no effect and will show as
drift once the backend's own value lands in state. Remove any literal
`"<SSH.PUBLIC.KEY>"` placeholder left by early versions. If you need a specific
key installed, contact Streamkap support — it cannot be set through Terraform.

The `mongodb_connection_hostname` attribute in MongoDB and MongoDB Hosted remains optional and
computed. When omitted, updates can refresh the derived hostname after the
connection string changes. Webhook `webhook_url` and `api_key` also remain
optional and computed for configuration compatibility. Normally leave these
server-populated attributes unset and read their resource outputs.

#### Topic metrics data source

`streamkap_topic_metrics` now reads the backend's topic-keyed table metrics.
Use `partition_count`, `replication_factor`, `retention_ms`,
`last_message_timestamp`, `snapshot_status_json`, and `record_error_total`.
Missing metrics remain null; a reported zero remains zero.

Existing `time_interval` and `time_unit` inputs remain accepted but are
deprecated and ignored. The legacy throughput, lag and latency result fields
remain in the schema as deprecated null values because the API does not return
them. Update expressions that expect numeric values before relying on these
results. Source and transform requests require topic database IDs alongside
topic IDs; see the data source example.

#### PostgreSQL source types and signal table

`database_port` was already numeric in v2.2.0 and remains numeric
(`database_port = 5432`). SSH ports changed from strings to numbers on the
PostgreSQL destination and SQL Server source; use `ssh_port = 22` there.
`signal_data_collection_schema_or_database` is optional and computed. If set,
use a full table path such as `public.streamkap_signal`, not just `public`.

#### Snowflake Destination & ClickHouse Destination — map fields are unchanged

`auto_qa_dedupe_table_mapping` (Snowflake) and `topics_config_map` (ClickHouse)
are **unchanged** between v2.1.x and v3.x — both remain map attributes with the
same names and shapes. No migration is required.

```hcl
resource "streamkap_destination_snowflake" "example" {
  auto_qa_dedupe_table_mapping = {
    rawTable1 = "dedupeTable1"
  }
}
```

> **If you adopted an early v3 beta (≤ `beta.16`):** a code-generation bug
> exposed these map fields as plain strings (and dropped `topics_config_map` /
> `auto_qa_dedupe_table_mapping` to a JSON-string form). That is fixed — revert
> any string values back to the map form shown above.

### Default Value Changes

Changed defaults can produce a plan for existing resources when the attribute
is omitted from configuration. Set an explicit value to preserve the previous
behavior, and inspect the plan before applying:

| Resource | Attribute | Old Default | New Default |
|----------|-----------|-------------|-------------|
| PostgreSQL Source | `heartbeat_enabled` | `false` | `true` |
| Snowflake Destination | `hard_delete` | `false` | `true` |
| SQL Server Source | `heartbeat_enabled` | `false` | `true` |
| DynamoDB Source | `poll_timeout_ms` | `1000` | `180000` |
| DynamoDB Source | `incremental_snapshot_chunk_size` | `32768` | `8192` |

#### Expect an in-place update on your first v3 plan

Beyond the table above, v3 **adds** optional attributes to existing connectors,
and many of them carry a client-side default. For a resource created under v2
the default was never part of the config, so the first v3 plan writes it and the
resource shows as an in-place `update` even though nothing in your
configuration changed. Migration validation observes this on the Kafka Direct
source (`format` defaults to `string`, `records_carry_streamkap_metadata` to
`false`) and the Databricks destination (`connection_timeout` to `180`,
`preserve_null_values` to `false`, plus the `transforms_*` family), and it can
occur on any connector that gained a defaulted attribute.

An in-place update can change connector behavior, including heartbeat and
delete handling. Set explicit values to preserve the behavior you need. Stop
and investigate an unexpected replacement or deletion before applying. The
migration suite covers representative configurations, not every configuration
or downstream data effect; skipped tests provide no upgrade evidence.

### Migration Steps

#### Step 1: Backup Your State

```bash
terraform state pull > terraform.tfstate.backup
```

#### Step 2: Fix Breaking Changes

Update these in your `.tf` files:
1. Apply the required renames and removals listed above.
2. Set any newly required destination fields.
3. Review defaults and any planned replacements before applying.

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

## Known limitations

### `streamkap_pipeline` and `streamkap_transform_*` do not support `lifecycle { create_before_destroy = true }`

Streamkap enforces unique pipeline and transform names per
`{tenant_id, service_id}`. The `create_before_destroy` lifecycle requires the
old and new instances to coexist briefly during the apply, which the backend
will reject with a 409 (pipelines) / 422 (transforms) conflict. Symptoms:

- `Error: streamkap_pipeline "<name>" already exists ...` or
  `Error: streamkap_transform "<name>" already exists ...`, with a backend
  409 / 422 in the debug log.
- After repeated retries with the old provider version, deposed instances
  accumulate in state (visible as
  `streamkap_pipeline.<name> (destroy deposed <key>)` or
  `streamkap_transform_<type>.<name> (destroy deposed <key>)` in
  `terraform plan`).

The provider now refuses to auto-adopt on this collision and surfaces an
actionable error. To recover:

1. **Remove the lifecycle directive** — for `streamkap_pipeline` and the
   `streamkap_transform_*` resources, use the default destroy-then-create. If
   you specifically need create-before-destroy semantics, rename the resource
   so old and new can coexist by name.
2. **If you have accumulated deposed entries**, back up the state with
   `terraform state pull` and inspect the plan before applying. Terraform normally
   removes deposed objects during apply. If an old provider adopted the same
   backend ID into both current and deposed instances, stop before applying: a
   deposed destroy could delete the current object. Resolve that state collision
   using a supported recovery procedure for your Terraform version. Do not append
   `.deposed.<key>` to a resource address; that is not valid `terraform state rm`
   syntax.
3. **If the resource already exists on the backend but not in state** (e.g.
   from a previous apply whose response was lost), use `terraform import`
   to bring it into state and then `terraform apply` to reconcile any drift.
   - Pipelines:
     `terraform import streamkap_pipeline.<name> <pipeline_id>`
   - Transforms: the resource type matches the existing record's `transform`
     field (e.g. `sql_join` → `streamkap_transform_sql_join`,
     `map_filter` → `streamkap_transform_map_filter`):
     `terraform import streamkap_transform_sql_join.<name> <transform_id>`

The id (and `transform` type for transforms) is visible in the Streamkap UI
or via `GET /pipelines?partial_name=<name>` /
`GET /transforms?partial_name=<name>`.

## Deprecation Timeline

| Version | Status |
|---------|--------|
| v2.0 | Deprecated attributes work with warnings |
| v2.x | Deprecated attributes continue to work |
| v3.0.0-beta.x | Deprecated attributes still work (pre-release, not for production) |
| v3.0.0 (stable) | Deprecated attributes work, with a deprecation warning |
| v3.x | Deprecated attributes continue to work, with a deprecation warning |
| v4.0 | Deprecated attributes **REMOVED** |

## Getting Help

If you encounter issues:
1. Check [GitHub Issues](https://github.com/streamkap-com/terraform-provider-streamkap/issues)
2. Open a new issue with your error message and config (redact secrets)
