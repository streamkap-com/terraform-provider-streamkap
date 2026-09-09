# Migration Guide

This guide helps existing users migrate their Terraform configurations between major versions.

---

## v2.x to v3.0

> **The v3.0.0 beta line is available for testing** (latest: `3.0.0-beta.31`). This is a
> pre-release — do not use it in production. If you are on v2.x and your setup is working,
> **there is no need to migrate yet**. Wait for the stable v3.0.0 release. The beta may
> introduce further breaking changes before the final release.

### What's New in v3.0

v3.0 adds new resource types and data sources on top of everything in v2.x:

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
      version = "3.0.0-beta.31"
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

#### Non-Aliasable Renames (Config Edit Required)

The following v2.1.19 attributes cannot be aliased because the underlying API contract
changed. If your v2.1.19 configuration uses them, rename them before upgrading to v3.x.

##### SQL Server Source

| v2.1.19 Name | v3.x Name | Why no alias |
|--------------|-----------|---------------|
| `database_dbname` (string) | `database_names` (comma-separated) | Backend API field changed from `database.dbname` to `database.names` to support multiple databases per connector. |

##### DynamoDB Source

| v2.1.19 Name | v3.x Name | Why no alias |
|--------------|-----------|---------------|
| `table_include_list_user_defined` | `table_include_list` | v2.1.19 field was `Required`, so a deprecated alias would still force a plan-time choice between names. A straight rename is the cleanest migration. |

#### Attributes Removed by the Backend (Config Edit Required)

A production backend release dropped these connector config fields. There is no
replacement attribute and no alias — remove them from your configuration.

| Resource | Removed attribute | Notes |
|----------|-------------------|-------|
| `streamkap_source_postgresql` | `streamkap_snapshot_large_table_threshold` | Backend dropped `streamkap.snapshot.large.table.threshold`. |
| `streamkap_source_postgresql` | `streamkap_snapshot_custom_table_config` | Backend dropped the field. |
| `streamkap_source_sqlserver` | `streamkap_snapshot_large_table_threshold` | The `snapshot_large_table_threshold` v2 alias is removed with it. |
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

`database_port` is numeric in v3 (`database_port = 5432`); it defaults to 5432.
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

This is an update, never a replacement, and no data is lost. Review the plan
before applying: if a new default is not what you want, set the attribute
explicitly to the value you intend.

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
