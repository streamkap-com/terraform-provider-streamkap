# AGENTS.md - AI Coding Agent Guide

This document provides context and instructions for AI coding agents working with the Streamkap Terraform provider.

## Provider Overview

**Provider Address:** `github.com/streamkap-com/streamkap`

The Streamkap Terraform provider manages data streaming infrastructure. It creates and configures:
- **Sources** - Database and event connectors that capture changes (CDC)
- **Destinations** - Data warehouses, lakes, and messaging systems
- **Pipelines** - Connections between sources and destinations
- **Transforms** - Data transformations applied in-flight
- **Topics** - Kafka topics for the data streams

## Quick Start Pattern

```hcl
terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.1.0"
    }
  }
}

provider "streamkap" {
  # Uses STREAMKAP_CLIENT_ID, STREAMKAP_SECRET env vars
}

# 1. Create a source
resource "streamkap_source_postgresql" "main" {
  name              = "my-postgres"
  database_hostname = "db.example.com"
  database_user     = "streamkap"
  database_password = var.db_password
  database_dbname   = "mydb"
  table_include_list = "public.customers"
}

# 2. Create a destination
resource "streamkap_destination_snowflake" "main" {
  name               = "my-snowflake"
  snowflake_url_name = "org-account"
  database           = "STREAMKAP"
  schema_name        = "PUBLIC"
  user               = "STREAMKAP_USER"
  private_key        = var.snowflake_key
}

# 3. Connect them with a pipeline
resource "streamkap_pipeline" "main" {
  name           = "postgres-to-snowflake"
  source_id      = streamkap_source_postgresql.main.id
  destination_id = streamkap_destination_snowflake.main.id
  transforms     = []  # optional
}
```

## Available Resources

### Sources (20 connectors)
| Resource | Description |
|----------|-------------|
| `streamkap_source_postgresql` | PostgreSQL CDC source |
| `streamkap_source_mysql` | MySQL CDC source |
| `streamkap_source_mongodb` | MongoDB CDC source |
| `streamkap_source_dynamodb` | DynamoDB CDC source |
| `streamkap_source_sqlserver` | SQL Server CDC source |
| `streamkap_source_kafkadirect` | Kafka source (direct) |
| `streamkap_source_alloydb` | AlloyDB CDC source |
| `streamkap_source_db2` | IBM DB2 CDC source |
| `streamkap_source_documentdb` | DocumentDB CDC source |
| `streamkap_source_elasticsearch` | Elasticsearch source |
| `streamkap_source_mariadb` | MariaDB CDC source |
| `streamkap_source_mongodbhosted` | MongoDB Atlas source |
| `streamkap_source_oracle` | Oracle CDC source |
| `streamkap_source_oracleaws` | Oracle on AWS CDC source |
| `streamkap_source_planetscale` | PlanetScale source |
| `streamkap_source_redis` | Redis source |
| `streamkap_source_s3` | S3 source |
| `streamkap_source_supabase` | Supabase CDC source |
| `streamkap_source_vitess` | Vitess CDC source |
| `streamkap_source_webhook` | Webhook source |

### Destinations (22 connectors)
| Resource | Description |
|----------|-------------|
| `streamkap_destination_snowflake` | Snowflake data warehouse |
| `streamkap_destination_clickhouse` | ClickHouse database |
| `streamkap_destination_databricks` | Databricks lakehouse |
| `streamkap_destination_postgresql` | PostgreSQL database |
| `streamkap_destination_s3` | Amazon S3 storage |
| `streamkap_destination_iceberg` | Apache Iceberg tables |
| `streamkap_destination_kafka` | Kafka (Confluent) |
| `streamkap_destination_azblob` | Azure Blob Storage |
| `streamkap_destination_bigquery` | Google BigQuery |
| `streamkap_destination_cockroachdb` | CockroachDB |
| `streamkap_destination_db2` | IBM DB2 |
| `streamkap_destination_gcs` | Google Cloud Storage |
| `streamkap_destination_httpsink` | HTTP endpoint |
| `streamkap_destination_kafkadirect` | Kafka (direct) |
| `streamkap_destination_motherduck` | MotherDuck |
| `streamkap_destination_mysql` | MySQL database |
| `streamkap_destination_oracle` | Oracle database |
| `streamkap_destination_r2` | Cloudflare R2 |
| `streamkap_destination_redis` | Redis |
| `streamkap_destination_redshift` | Amazon Redshift |
| `streamkap_destination_sqlserver` | SQL Server |
| `streamkap_destination_starburst` | Starburst Galaxy |

### Transforms (6 types)
| Resource | Description |
|----------|-------------|
| `streamkap_transform_map_filter` | JavaScript map/filter transformation |
| `streamkap_transform_enrich` | Synchronous data enrichment |
| `streamkap_transform_enrich_async` | Asynchronous data enrichment |
| `streamkap_transform_sql_join` | SQL-based join transformation |
| `streamkap_transform_rollup` | Aggregation/rollup transformation |
| `streamkap_transform_fan_out` | Fan-out to multiple destinations |

### Other Resources
| Resource | Description |
|----------|-------------|
| `streamkap_pipeline` | Pipeline connecting source to destination |
| `streamkap_topic` | Kafka topic configuration |
| `streamkap_tag` | Resource tagging |

### Data Sources
| Data Source | Description |
|-------------|-------------|
| `streamkap_transform` | Query existing transforms |
| `streamkap_tag` | Query existing tags |
| `streamkap_topics` | List topics |
| `streamkap_topic` | Query single topic |
| `streamkap_topic_metrics` | Topic metrics |

## Authentication

The provider uses OAuth2 with client credentials:

```hcl
provider "streamkap" {
  client_id = var.streamkap_client_id  # or STREAMKAP_CLIENT_ID env var
  secret    = var.streamkap_secret     # or STREAMKAP_SECRET env var
  host      = "https://api.streamkap.com"  # optional, defaults to production
}
```

**Best Practice:** Use environment variables for credentials:
```bash
export STREAMKAP_CLIENT_ID="your-client-id"
export STREAMKAP_SECRET="your-secret"
```

## Common Patterns

### Sensitive Values
Always use variables for credentials:

```hcl
variable "db_password" {
  type      = string
  sensitive = true
}

resource "streamkap_source_postgresql" "main" {
  database_password = var.db_password  # Never hardcode
}
```

### Table Selection
Most sources use comma-separated table patterns:

```hcl
resource "streamkap_source_postgresql" "main" {
  # Include specific tables
  table_include_list = "public.customers,public.orders,inventory.*"

  # Or exclude tables (use one or the other)
  table_exclude_list = "public.temp_*,public.logs"
}
```

### Insert Modes
Destinations support different insert strategies:

```hcl
resource "streamkap_destination_snowflake" "main" {
  insert_mode = "upsert"  # Valid values: insert, upsert
}
```

### Transforms in Pipeline
Transforms are applied in order:

```hcl
resource "streamkap_pipeline" "main" {
  source_id      = streamkap_source_postgresql.main.id
  destination_id = streamkap_destination_snowflake.main.id
  transforms     = [
    streamkap_transform_map_filter.filter_pii.id,
    streamkap_transform_enrich.add_metadata.id,
  ]
}
```

## Schema Discovery

All resources have rich descriptions with:
- **Enum values** listed inline (e.g., "Valid values: `insert`, `upsert`")
- **Defaults** documented (e.g., "Defaults to `5432`")
- **Security notes** on sensitive fields

Use `terraform providers schema -json` or the Terraform MCP Server to inspect schemas programmatically.

## Error Handling

### Common Errors

**Authentication Failed:**
```
Error: Unable to Create Streamkap API Client
```
→ Check `client_id` and `secret` values/environment variables

**Resource Not Found:**
```
Error: 404 Not Found
```
→ Resource was deleted outside Terraform; run `terraform refresh`

**Validation Error:**
```
Error: Invalid value for [field]
```
→ Check enum values and type constraints in schema

### Retry Behavior
The provider automatically retries on:
- HTTP 429 (Too Many Requests)
- HTTP 502, 503, 504 (Gateway errors)
- Network timeouts
- Kafka transient errors

## Import Existing Resources

All resources support import:

```bash
terraform import streamkap_source_postgresql.main <resource-id>
```

Resource IDs can be found in the Streamkap UI or API.

## Examples Location

Complete examples are in `examples/resources/`:
- `basic.tf` - Minimal required configuration
- `complete.tf` - All available options

## Development Reference

For provider development, see:
- `CLAUDE.md` - Development instructions
- `docs/ARCHITECTURE.md` - Code architecture
- `docs/AI_AGENT_COMPATIBILITY.md` - AI integration patterns
- `docs/TESTING.md` - Test execution guide

## API Documentation

- [Streamkap Documentation](https://docs.streamkap.com)
- [OpenAPI Spec](https://api.streamkap.com/openapi.json)
- [Terraform Registry](https://registry.terraform.io/providers/streamkap-com/streamkap)
