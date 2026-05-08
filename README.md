# Streamkap Terraform Provider

Terraform provider for [Streamkap](https://streamkap.com) - a real-time data streaming platform.

## Features

### Source Connectors
- PostgreSQL, MySQL, MongoDB, SQL Server, DynamoDB, Kafka Direct
- AlloyDB, DB2, DocumentDB, Elasticsearch, MariaDB, MongoDB Hosted
- Oracle, Oracle AWS, PlanetScale, Redis, S3, Salesforce Webhook, Supabase
- Vitess, Webhook, Zendesk Webhook

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
      version = ">= 2.0.0"
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
  database_port     = "5432"
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
- [Go](https://golang.org/doc/install) >= 1.21 (for development)

## Installation

```hcl
terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
}

provider "streamkap" {
  # Credentials loaded from environment variables:
  # STREAMKAP_CLIENT_ID and STREAMKAP_SECRET
}
```

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

## Example Usage

```hcl
# Create a PostgreSQL source
resource "streamkap_source_postgresql" "example" {
  name              = "my-postgresql-source"
  database_hostname = "db.example.com"
  database_port     = "5432"
  database_user     = "streamkap"
  database_password = "secret"
  database_dbname   = "mydb"
}

# Create a Snowflake destination
resource "streamkap_destination_snowflake" "example" {
  name              = "my-snowflake-dest"
  snowflake_url_name = "account.snowflakecomputing.com"
  snowflake_user_name = "streamkap"
  snowflake_private_key = file("~/.ssh/snowflake_key.pem")
  snowflake_database_name = "STREAMKAP_DB"
  snowflake_schema_name = "PUBLIC"
}

# Tag for organizing related resources in the Streamkap UI
resource "streamkap_tag" "production" {
  name        = "production"
  description = "Production environment resources"
  type        = ["sources", "destinations", "pipelines"]
}

# Create a pipeline connecting source to destination
resource "streamkap_pipeline" "example" {
  name = "my-pipeline"

  source = {
    id        = streamkap_source_postgresql.example.id
    name      = streamkap_source_postgresql.example.name
    connector = streamkap_source_postgresql.example.connector
    topics    = ["public.orders"]
  }

  destination = {
    id        = streamkap_destination_snowflake.example.id
    name      = streamkap_destination_snowflake.example.name
    connector = streamkap_destination_snowflake.example.connector
  }

  tags = [streamkap_tag.production.id]
}
```

See the [examples](./examples/) directory for more complete examples.

## Development

### Building

```bash
go install .
```

### Running Tests

```bash
# Unit tests
go test -v -short ./...

# Acceptance tests (creates real resources)
export TF_ACC=1
export STREAMKAP_CLIENT_ID="your-client-id"
export STREAMKAP_SECRET="your-secret"
go test -v ./internal/provider -timeout 30m
```

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

The provider uses code generation for connector schemas. To regenerate:

```bash
# Generate schema for a specific connector
go run cmd/tfgen/main.go generate \
  --config /path/to/backend/app/sources/plugins/postgresql/configuration.latest.json \
  --type source \
  --output internal/generated/

# Run tests to verify
go test -v ./cmd/tfgen/...
```

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

## AI-Agent Compatibility

This provider is optimized for use with AI assistants via the [Terraform MCP Server](https://github.com/hashicorp/terraform-mcp-server). AI agents can leverage:

- **Rich Schema Descriptions**: All resources have detailed `MarkdownDescription` fields with valid values, defaults, and security notes
- **Structured Examples**: Each resource includes `basic.tf` (minimal config) and `complete.tf` (all options) examples
- **Semantic Documentation**: Enum fields list valid values, sensitive fields include security warnings

### Using with AI Assistants

When working with AI coding assistants (Claude, Copilot, etc.), the provider's enhanced schema descriptions enable:

1. **Accurate code generation** - AI can suggest correct attribute names and valid values
2. **Security awareness** - Sensitive fields are clearly marked
3. **Default value knowledge** - AI knows what defaults are applied

Example prompt for AI assistants:
```
Create a Streamkap PostgreSQL source connected to a Snowflake destination
with CDC enabled and SSL required.
```

See [AI_AGENT_COMPATIBILITY.md](docs/AI_AGENT_COMPATIBILITY.md) for detailed AI integration guidelines.

## Upgrading

See [MIGRATION.md](docs/MIGRATION.md) for guidance on upgrading between major versions, including breaking changes and deprecated attributes.

### v3.0 (Beta)

The v3.0 line adds:
- **Tags on every entity** — every source, destination, transform, topic, and pipeline accepts an optional `tags = [...]` attribute (Set of tag IDs).
- **`streamkap_tags` data source** — list/filter tags by name, type, or IDs (alternative to single-tag lookup).
- **New connectors** — `streamkap_source_salesforce_webhook`, `streamkap_source_zendesk_webhook`, `streamkap_destination_pinecone`, `streamkap_destination_weaviate`, `streamkap_transform_topic_router`.
- **New non-connector resources** — `streamkap_kafka_user` (ACL-based Kafka access), `streamkap_client_credential` (API tokens).
- **New data sources** — `streamkap_roles`, `streamkap_tags`.

See [docs/MIGRATION.md](docs/MIGRATION.md) for the full v2 → v3 changelog and any breaking changes.

To try a beta release, pin the latest `3.0.0-beta.x` available on the [Terraform Registry](https://registry.terraform.io/providers/streamkap-com/streamkap/latest):

```hcl
version = "~> 3.0.0-beta"  # any 3.0.0-beta.x; resolves to the most recent on the registry
```

## License

Mozilla Public License 2.0
