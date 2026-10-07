# Using the provider

Configuration patterns for people and AI agents writing Streamkap Terraform
configs. Install, quick start and authentication are in the [README](../README.md);
per-attribute reference is on the
[Registry](https://registry.terraform.io/providers/streamkap-com/streamkap) and in
`docs/resources/`. Provider internals are in [ARCHITECTURE.md](ARCHITECTURE.md).

Terraform source address: `streamkap-com/streamkap`
(`registry.terraform.io/streamkap-com/streamkap`).

## What it manages

- **Sources** — CDC and event connectors (databases, queues, webhooks, S3).
- **Destinations** — warehouses, lakes, queues, vector DBs.
- **Pipelines** — connect a source to a destination, optionally through transforms.
- **Transforms** — in-flight data transformations (JS map/filter, enrich, SQL join, rollup, fan-out).
- **Topics** — Kafka topic configuration.
- **Kafka users** and **Client credentials** (v3+) — access control and machine auth.

## Minimal example

```hcl
terraform {
  required_providers {
    streamkap = { source = "streamkap-com/streamkap", version = "3.0.1" }
  }
}

variable "db_password" {
  type      = string
  sensitive = true
}

variable "snowflake_key" {
  type        = string
  sensitive   = true
  description = "Snowflake private key without PEM headers, footers, or line breaks."
}

resource "streamkap_source_postgresql" "main" {
  name                = "my-postgres"
  database_hostname   = "db.example.com"
  database_user       = "streamkap"
  database_password   = var.db_password
  database_dbname     = "mydb"
  schema_include_list = "public"
  table_include_list  = "public.customers"
}

resource "streamkap_destination_snowflake" "main" {
  name                    = "my-snowflake"
  snowflake_url_name      = "org-account.snowflakecomputing.com"
  snowflake_user_name     = "STREAMKAP_USER"
  snowflake_private_key   = var.snowflake_key
  snowflake_database_name = "STREAMKAP"
  snowflake_schema_name   = "PUBLIC"
}

resource "streamkap_pipeline" "main" {
  name = "postgres-to-snowflake"

  source = {
    id        = streamkap_source_postgresql.main.id
    name      = streamkap_source_postgresql.main.name
    connector = streamkap_source_postgresql.main.connector
    topics    = ["public.customers"]
  }

  destination = {
    id        = streamkap_destination_snowflake.main.id
    name      = streamkap_destination_snowflake.main.name
    connector = streamkap_destination_snowflake.main.connector
  }
}
```

`source` and `destination` are **nested objects**, not bare IDs — each carries
`id`, `name`, and `connector` (the source additionally takes `topics`). Wire them
from the resources' own attributes as above rather than hardcoding.

Full per-resource examples: `examples/resources/streamkap_<name>/{basic,complete}.tf`.

## Resources

For the canonical, always-up-to-date list, run `terraform providers schema -json`
or browse the registry — these tables are a snapshot.

### Sources
`streamkap_source_postgresql`, `mysql`, `mongodb`, `mongodbhosted`, `dynamodb`, `sqlserver`, `oracle`, `oracleaws`, `db2`, `informix` (v3), `mariadb`, `alloydb`, `documentdb`, `elasticsearch`, `planetscale`, `redis`, `s3`, `supabase`, `vitess`, `webhook`, `salesforce_webhook` (v3), `zendesk_webhook` (v3), `shopify_webhook` (v3), `stripe_webhook` (v3), `kafkadirect`.

### Destinations
`streamkap_destination_snowflake`, `clickhouse`, `databricks`, `postgresql`, `mysql`, `sqlserver`, `oracle`, `db2`, `cockroachdb`, `bigquery`, `redshift`, `motherduck`, `starburst`, `s3`, `gcs`, `r2`, `azblob`, `iceberg`, `kafka`, `kafkadirect`, `httpsink`, `redis`, `weaviate` (v3), `pinecone` (v3).

### Transforms
`streamkap_transform_map_filter`, `enrich`, `enrich_async`, `sql_join`, `rollup`, `fan_out`, `topic_router` (v3).

### Other resources
`streamkap_pipeline`, `streamkap_topic`, `streamkap_tag`, `streamkap_kafka_user` (v3), `streamkap_client_credential` (v3).

### Data sources
`streamkap_transform`, `streamkap_tag`, `streamkap_tags` (v3), `streamkap_topics`, `streamkap_topic`, `streamkap_topic_metrics`, `streamkap_roles` (v3).

## Patterns

**Tags** — every source, destination, transform, topic, and pipeline accepts an
optional `tags = [streamkap_tag.<name>.id, ...]`. Tags are managed by Terraform
when set in config (drift detected and reverted on apply); when unset, the
provider preserves whatever tags the backend has attached out-of-band. To clear
tags from an entity, set `tags = []` (the provider distinguishes `null` from
empty-list on the wire and the backend respects the difference).

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

Look up existing tag IDs (instead of hardcoding) via the `streamkap_tags`
data source, filtered by `filter_name`, `filter_type`, or `filter_ids`. Check
that the results match the intended tags before referencing their IDs.

**Sensitive values** — declare `variable { sensitive = true }`; never inline secrets.

**Table selection** — most CDC sources use comma-separated patterns:
```hcl
table_include_list = "public.customers,public.orders,inventory.*"
# or
table_exclude_list = "public.temp_*,public.logs"
```

**Insert modes** — destinations typically support `insert` and `upsert` via `insert_mode`.

**Transform chaining** — `transforms` on a pipeline is an *ordered* list of
objects, each carrying the transform's `id` and the `topics` it should process
(both required):

```hcl
transforms = [
  { id = streamkap_transform_map_filter.redact.id, topics = ["public.customers"] },
]
```

**Auto-discovering transform output topics** — when a transform produces topics
whose names are generated dynamically (e.g. a topic-router / fan-out transform)
and can't be listed up front in `transforms[].topics`, use
`topic_auto_discovery_transforms = [{ transform_id = ..., regex = "..." }]` on
the pipeline. The backend matches the regex against the transform's output topic
names server-side and adds them automatically.

**Transform implementation code** — most transform resources accept `implementation_json`:
```hcl
implementation_json = jsonencode({
  language        = "JAVASCRIPT"
  value_transform = "function _streamkap_transform(inputObj) { return inputObj; }"
})
```
Omit it to manage code outside Terraform; Terraform won't overwrite existing implementation.

## Schema discovery

Every attribute has both `Description` and `MarkdownDescription`. Enums are
listed inline ("Valid values: `insert`, `upsert`"), defaults are documented
("Defaults to `5432`"), and sensitive fields carry an explicit `**Security:**`
note. Use `terraform providers schema -json` or the Terraform MCP Server to
consume them programmatically.

## Import

All resources support import; the ID comes from the Streamkap UI or API:

```bash
terraform import streamkap_source_postgresql.main <resource-id>
```

## Errors and retry

The provider retries eligible operations on HTTP 429, 502/503/504, network
timeouts, and transient Kafka errors. Client-credential creation does not retry
failed responses; the API has no idempotency key, so replaying a request can
issue an unmanaged credential. Common application-level errors:

- `Unable to Create Streamkap API Client` → check `STREAMKAP_CLIENT_ID` / `STREAMKAP_SECRET`.
- `404 Not Found` on read → resource was deleted out of band; `terraform refresh` will drop it from state.
- `Invalid value for [field]` → check enum values and types in the schema.
- `422 already exists` on create → another resource in the tenant has the same
  name, or an orphaned record exists in another service of the same tenant. No
  connector resource adopts the existing record: use `terraform import` when a
  previous apply created it but lost the response, and inspect state for a
  deposed `create_before_destroy` instance otherwise (see
  [ARCHITECTURE.md](ARCHITECTURE.md#backend-contract-quirks)).

## References

- User docs: <https://docs.streamkap.com/streamkap-provider-for-terraform>
- Streamkap API: <https://api.streamkap.com> · OpenAPI: <https://api.streamkap.com/openapi.json>
- Registry: <https://registry.terraform.io/providers/streamkap-com/streamkap>
- Migration (v2 → v3): [guides/migration.md](guides/migration.md)
- Provider internals: [ARCHITECTURE.md](ARCHITECTURE.md), [CODE_GENERATOR.md](CODE_GENERATOR.md)
