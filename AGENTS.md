# AGENTS.md — AI Coding Agent Guide

Audience: AI agents (or humans) **using** the Streamkap Terraform provider in customer configs. For provider-development guidance, see `CLAUDE.md`.

Provider address: `github.com/streamkap-com/streamkap` · Registry: `streamkap-com/streamkap`

## Versions and branches

- **v2.x (stable)** — what users should pin in production: `version = "~> 2.1"`. Lives on the `main` branch.
- **v3.x (beta)** — new resources (`streamkap_destination_weaviate`, `streamkap_kafka_user`, `streamkap_client_credential`, `streamkap_roles` data source) plus deprecated-attribute removals. Lives on the `develop` branch. **Do not use in production yet.**

`develop` is **not** merged into `main` until v3 is promoted to stable. If you're consuming the provider today, target v2.x; if you're contributing, see the Branches section in `CLAUDE.md`.

## What it manages

- **Sources** — CDC and event connectors (databases, queues, webhooks, S3).
- **Destinations** — warehouses, lakes, queues, vector DBs.
- **Pipelines** — connect a source to a destination, optionally through transforms.
- **Transforms** — in-flight data transformations (JS map/filter, enrich, SQL join, rollup, fan-out).
- **Topics** — Kafka topic configuration.
- **Kafka users** and **Client credentials** (v3+) — access control and machine auth.

## Authentication

OAuth2 client credentials. Prefer env vars over inline literals:

```bash
export STREAMKAP_CLIENT_ID=...
export STREAMKAP_SECRET=...
# optional: export STREAMKAP_HOST=https://api.streamkap.com
```

```hcl
provider "streamkap" {}  # picks up env vars
```

## Minimal example

```hcl
terraform {
  required_providers {
    streamkap = { source = "streamkap-com/streamkap", version = "~> 2.1" }
  }
}

resource "streamkap_source_postgresql" "main" {
  name               = "my-postgres"
  database_hostname  = "db.example.com"
  database_user      = "streamkap"
  database_password  = var.db_password
  database_dbname    = "mydb"
  table_include_list = "public.customers"
}

resource "streamkap_destination_snowflake" "main" {
  name               = "my-snowflake"
  snowflake_url_name = "org-account"
  database           = "STREAMKAP"
  schema_name        = "PUBLIC"
  user               = "STREAMKAP_USER"
  private_key        = var.snowflake_key
}

resource "streamkap_pipeline" "main" {
  name           = "postgres-to-snowflake"
  source_id      = streamkap_source_postgresql.main.id
  destination_id = streamkap_destination_snowflake.main.id
  transforms     = []
}
```

Full per-resource examples: `examples/resources/streamkap_<name>/{basic,complete}.tf`.

## Resources

### Sources
`streamkap_source_postgresql`, `mysql`, `mongodb`, `mongodbhosted`, `dynamodb`, `sqlserver`, `oracle`, `oracleaws`, `db2`, `mariadb`, `alloydb`, `documentdb`, `elasticsearch`, `planetscale`, `redis`, `s3`, `supabase`, `vitess`, `webhook`, `kafkadirect`.

### Destinations
`streamkap_destination_snowflake`, `clickhouse`, `databricks`, `postgresql`, `mysql`, `sqlserver`, `oracle`, `db2`, `cockroachdb`, `bigquery`, `redshift`, `motherduck`, `starburst`, `s3`, `gcs`, `r2`, `azblob`, `iceberg`, `kafka`, `kafkadirect`, `httpsink`, `redis`, `weaviate` (v3).

### Transforms
`streamkap_transform_map_filter`, `enrich`, `enrich_async`, `sql_join`, `rollup`, `fan_out`.

### Other resources
`streamkap_pipeline`, `streamkap_topic`, `streamkap_tag`, `streamkap_kafka_user` (v3), `streamkap_client_credential` (v3).

### Data sources
`streamkap_transform`, `streamkap_tag`, `streamkap_topics`, `streamkap_topic`, `streamkap_topic_metrics`, `streamkap_roles` (v3).

For the canonical, always-up-to-date list, run `terraform providers schema -json` or browse the registry — these tables are a snapshot.

## Patterns

**Sensitive values** — declare `variable { sensitive = true }`; never inline secrets.

**Table selection** — most CDC sources use comma-separated patterns:
```hcl
table_include_list = "public.customers,public.orders,inventory.*"
# or
table_exclude_list = "public.temp_*,public.logs"
```

**Insert modes** — destinations typically support `insert` and `upsert` via `insert_mode`.

**Transform chaining** — `transforms` on a pipeline is an *ordered* list of transform IDs.

**Transform implementation code** — most transform resources accept `implementation_json`:
```hcl
implementation_json = jsonencode({
  language        = "JavaScript"
  value_transform = "return record;"
})
```
Omit it to manage code outside Terraform; Terraform won't overwrite existing implementation.

## Schema discovery

Every attribute has both `Description` and `MarkdownDescription`. Enums are listed inline ("Valid values: `insert`, `upsert`"), defaults are documented ("Defaults to `5432`"), and sensitive fields carry an explicit `**Security:**` note. Use `terraform providers schema -json` or the Terraform MCP Server to consume them programmatically.

## Import

All resources support import; the ID comes from the Streamkap UI or API:

```bash
terraform import streamkap_source_postgresql.main <resource-id>
```

## Errors and retry

The provider auto-retries on HTTP 429, 502/503/504, network timeouts, and transient Kafka errors. Common application-level errors:

- `Unable to Create Streamkap API Client` → check `STREAMKAP_CLIENT_ID` / `STREAMKAP_SECRET`.
- `404 Not Found` on read → resource was deleted out of band; `terraform refresh` will drop it from state.
- `Invalid value for [field]` → check enum values and types in the schema.
- `422 already exists` on create → another resource in the tenant has the same name (or an orphaned record exists in another service of the same tenant — see API quirks in `CLAUDE.md`).

## References

- User docs: <https://docs.streamkap.com/streamkap-provider-for-terraform>
- Streamkap API: <https://api.streamkap.com> · OpenAPI: <https://api.streamkap.com/openapi.json>
- Registry: <https://registry.terraform.io/providers/streamkap-com/streamkap>
- Migration (v2 → v3): `docs/MIGRATION.md`
- Architecture / internals (provider development): `CLAUDE.md`, `docs/ARCHITECTURE.md`, `docs/CODE_GENERATOR.md`
