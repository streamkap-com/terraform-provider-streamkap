# Streamkap Terraform Provider

Terraform provider for [Streamkap](https://streamkap.com) - a real-time data streaming platform.

v3 is the stable release line and lives on `main`. The legacy v2 maintenance
line lives on `v2` and receives bug and security fixes only through
15 October 2026. Review the [migration guide](docs/guides/migration.md) before upgrading
from v2.

## Features

### Source Connectors
- PostgreSQL, MySQL, MongoDB, SQL Server, DynamoDB, Kafka Direct
- AlloyDB, DB2, DocumentDB, Elasticsearch, Informix, MariaDB, MongoDB Hosted
- Oracle, Oracle AWS, PlanetScale, Redis, S3, Supabase, Vitess
- Webhook, Salesforce Webhook, Shopify Webhook, Stripe Webhook, Zendesk Webhook
- API sources: HubSpot, Salesforce, NetSuite

### Destination Connectors
- Snowflake, ClickHouse, Databricks, PostgreSQL, S3, Iceberg, Kafka
- Azure Blob, BigQuery, CockroachDB, DB2, GCS, HTTP Sink, Kafka Direct
- Motherduck, MySQL, Oracle, Pinecone, R2, Redis, Redshift, SQL Server
- Starburst, Weaviate

### Transform Resources
- Map Filter
- Enrich
- Enrich Async
- SQL Join
- Rollup
- Fan Out
- Topic Router

### Other Resources
- Pipelines
- Topics
- Topic Destinations (`streamkap_topic_destination` — sends one API source topic to a destination)
- Tags (`streamkap_tag` — manages individual tag definitions)
- Kafka Users (ACL-based Kafka access control)
- Client Credentials (API token management)

Every source, destination, transform, topic, and pipeline accepts an optional
`tags = [streamkap_tag.<name>.id, ...]` attribute for organizing and filtering
resources in the Streamkap UI (v3+).

### Data Sources
- Topics, Topic, Topic Metrics
- Tag (`streamkap_tag` — single-tag-by-id lookup)
- Tags (`streamkap_tags` — list/filter tags by name, type, or IDs)
- Transforms
- Roles

## Quick Start

Configure a source, destination, and pipeline. The database and Snowflake account
must already be configured for Streamkap access.

### 1. Configure Provider

```hcl
terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = "3.0.1"
    }
  }
}

provider "streamkap" {
  # Credentials from environment variables (recommended)
}
```

Set your credentials:
```bash
export STREAMKAP_CLIENT_ID="your-client-id"
export STREAMKAP_SECRET="your-secret"
```

### 2. Create a Source

```hcl
resource "streamkap_source_postgresql" "my_source" {
  name                = "production-postgres"
  database_hostname   = "db.example.com"
  database_port       = 5432
  database_user       = "streamkap"
  database_password   = var.db_password
  database_dbname     = "mydb"
  schema_include_list = "public"
  table_include_list  = "public.orders,public.customers"
}

variable "db_password" {
  type      = string
  sensitive = true
}
```

### 3. Create a Destination

```hcl
resource "streamkap_destination_snowflake" "my_dest" {
  name                    = "analytics-snowflake"
  snowflake_url_name      = "account.snowflakecomputing.com"
  snowflake_user_name     = "streamkap"
  snowflake_private_key   = var.snowflake_private_key
  snowflake_database_name = "STREAMKAP_DB"
  snowflake_schema_name   = "PUBLIC"
}

variable "snowflake_private_key" {
  type        = string
  sensitive   = true
  description = "Snowflake private key without PEM headers, footers, or line breaks."
}

# Connect them with a pipeline
resource "streamkap_pipeline" "my_pipeline" {
  name = "postgres-to-snowflake"

  source = {
    id        = streamkap_source_postgresql.my_source.id
    name      = streamkap_source_postgresql.my_source.name
    connector = streamkap_source_postgresql.my_source.connector
    topics    = ["public.orders", "public.customers"]
  }

  destination = {
    id        = streamkap_destination_snowflake.my_dest.id
    name      = streamkap_destination_snowflake.my_dest.name
    connector = streamkap_destination_snowflake.my_dest.connector
  }
}
```

Set `TF_VAR_db_password` and `TF_VAR_snowflake_private_key`, then initialize
and review the plan:

```bash
terraform init
terraform plan -out=streamkap.tfplan
terraform apply streamkap.tfplan
```

After applying, check connector health and data delivery in Streamkap.

See [examples/](./examples/) for connector examples to adapt to your environment.

---

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.27.1 (for development — see the `go` directive in `go.mod`)

## Authentication

Set environment variables:
```bash
export STREAMKAP_CLIENT_ID="your-client-id"
export STREAMKAP_SECRET="your-secret"
# Optional: custom API host
export STREAMKAP_HOST="https://api.streamkap.com"
```

Or configure in the provider block (not recommended for security):
```hcl
provider "streamkap" {
  client_id = "your-client-id"
  secret    = "your-secret"
}
```

## Development

### Building

```bash
go install .
```

### Running Tests

```bash
# Unit + schema-compat + validator tests (fast, no API calls)
make test-all

# Acceptance tests (creates real resources — takes ~15 minutes)
export STREAMKAP_CLIENT_ID="your-client-id"
export STREAMKAP_SECRET="your-secret"
make testacc
```

Prefer the `make` targets over a bare `go test ./...`: tests auto-load `.env`, and
a `TF_ACC=1` line there turns any unfiltered run into a live-API acceptance run.
`make test` clears `TF_ACC` for exactly this reason. Run `make help` for the full
target list.

### Local Development

Configure `~/.terraformrc`:
```hcl
provider_installation {
  dev_overrides {
    "streamkap-com/streamkap" = "/path/to/go/bin"
  }
  direct {}
}
```

Then build and use:
```bash
go install .
cd examples/provider
terraform plan
```

### Code Generation

Connector schemas in `internal/generated/` are produced by `cmd/tfgen` from the backend's `configuration.latest.json` plugin specs. API-source schemas also use each connector's `form.schema.json` for conditional credentials and defaults. Regenerate via:

```bash
STREAMKAP_BACKEND_PATH=/path/to/python-be-streamkap make generate
```

`make generate` runs `tfgen` and then `tfplugindocs`, in that order. Do not use
`go generate ./...` — it renders the docs *before* regenerating the schemas, so
new fields land in `internal/generated/` but never reach `docs/resources/`.

Never hand-edit `internal/generated/` — it is overwritten on every regen. Fix the
generator instead. See [docs/CODE_GENERATOR.md](docs/CODE_GENERATOR.md) for the
generator internals, the override system, and the walkthrough for adding a new
connector.

HubSpot, Salesforce and NetSuite API sources use `streamkap_source_hubspot`, `streamkap_source_salesforce` and `streamkap_source_netsuite`. Their topics reach destinations through `streamkap_topic_destination`, one resource per full topic ID and destination ID. API sources reconcile automatically; do not use `streamkap_pipeline` or a manual deploy step for them. Salesforce browser OAuth must be completed through the CLI or UI before importing that source into Terraform. See the generated resource pages for fields and examples.

### Project Structure

```
├── cmd/tfgen/           # Schema generator CLI
├── internal/
│   ├── api/             # API client
│   ├── generated/       # Generated schemas and models
│   ├── provider/        # Provider and tests
│   ├── resource/
│   │   ├── connector/   # Generic base resource
│   │   ├── source/      # Source connector configs
│   │   ├── destination/ # Destination connector configs
│   │   ├── transform/   # Transform resource configs
│   │   ├── pipeline/    # Pipeline resource
│   │   ├── topic/       # Topic resource
│   │   ├── tag/         # Tag resource
│   │   ├── kafka_user/  # Kafka user resource
│   │   └── client_credential/ # Client credential resource
│   ├── datasource/      # Data sources
│   └── helper/          # Utility functions
└── examples/            # Example Terraform configs
```

## Documentation

- [Streamkap Provider on Terraform Registry](https://registry.terraform.io/providers/streamkap-com/streamkap)
- [Streamkap Documentation](https://docs.streamkap.com)
- [API Reference](https://api.streamkap.com/openapi.json)
- [Changelog](CHANGELOG.md) - Version history and breaking changes
- [Architecture](docs/ARCHITECTURE.md) - Provider design and code structure

## Schema discovery

Use `terraform providers schema -json` to inspect attributes, types, sensitivity
and descriptions. Resource examples live under `examples/resources/`; the
[agent guide](AGENTS.md) documents configuration patterns and provider development.

## Upgrading

Follow the [v2 → v3 migration guide](docs/guides/migration.md) for the upgrade procedure,
required configuration edits, changed defaults, and supported aliases. Pin the
release you have validated and review the plan before applying.

For release history, see the [changelog](CHANGELOG.md).

## License

Mozilla Public License 2.0
