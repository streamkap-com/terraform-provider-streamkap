# Streamkap Terraform Provider

Terraform provider for [Streamkap](https://streamkap.com) - a real-time data streaming platform.

## Features

### Source Connectors
- PostgreSQL, MySQL, MongoDB, SQL Server, DynamoDB, Kafka Direct
- AlloyDB, DB2, DocumentDB, Elasticsearch, Informix, MariaDB, MongoDB Hosted
- Oracle, Oracle AWS, PlanetScale, Redis, S3, Supabase, Vitess
- Webhook, Salesforce Webhook, Shopify Webhook, Stripe Webhook, Zendesk Webhook

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

Get up and running with Streamkap in 3 steps:

### 1. Configure Provider

```hcl
terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = "~> 2.1"
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
  name              = "production-postgres"
  database_hostname = "db.example.com"
  database_port     = 5432
  database_user     = "streamkap"
  database_password = var.db_password
  database_dbname   = "mydb"
}
```

### 3. Create a Destination

```hcl
resource "streamkap_destination_snowflake" "my_dest" {
  name                    = "analytics-snowflake"
  snowflake_url_name      = "account.snowflakecomputing.com"
  snowflake_user_name     = "streamkap"
  snowflake_private_key   = file("~/.ssh/snowflake_key.pem")
  snowflake_database_name = "STREAMKAP_DB"
  snowflake_schema_name   = "PUBLIC"
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

Run `terraform apply` and your data pipeline is ready.

See [examples/](./examples/) for complete configurations covering every supported connector.

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
    "github.com/streamkap-com/streamkap" = "/path/to/go/bin"
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

Connector schemas in `internal/generated/` are produced by `cmd/tfgen` from the
backend's `configuration.latest.json` plugin specs. Regenerate **only** via:

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

See [MIGRATION.md](docs/MIGRATION.md) for guidance on upgrading between major versions, including breaking changes and deprecated attributes.

### v3.0 (Beta)

The v3.0 line adds:
- **Tags on every entity** — every source, destination, transform, topic, and pipeline accepts an optional `tags = [...]` attribute (Set of tag IDs).
- **`streamkap_tags` data source** — list/filter tags by name, type, or IDs (alternative to single-tag lookup).
- **New connectors** — `streamkap_source_informix`, `streamkap_source_salesforce_webhook`, `streamkap_source_shopify_webhook`, `streamkap_source_stripe_webhook`, `streamkap_source_zendesk_webhook`, `streamkap_destination_pinecone`, `streamkap_destination_weaviate`, `streamkap_transform_topic_router`.
- **New non-connector resources** — `streamkap_kafka_user` (ACL-based Kafka access), `streamkap_client_credential` (API tokens).
- **New data sources** — `streamkap_roles`, `streamkap_tags`.

See [docs/MIGRATION.md](docs/MIGRATION.md) for the full v2 → v3 changelog and any breaking changes.

To try a beta release, pin the **exact** `3.0.0-beta.x` you tested — check the
[Terraform Registry](https://registry.terraform.io/providers/streamkap-com/streamkap/latest)
for the latest one. Each beta may introduce further breaking changes, so a range
constraint can pull one you haven't validated:

```hcl
version = "3.0.0-beta.31" # exact pin; do not use a range for pre-releases
```

## License

Mozilla Public License 2.0
