# Streamkap Terraform Provider

Terraform provider for [Streamkap](https://streamkap.com) - a real-time data streaming platform.

## Features

### Source Connectors
- PostgreSQL
- MySQL
- MongoDB
- SQL Server
- DynamoDB
- Kafka Direct

### Destination Connectors
- Snowflake
- ClickHouse
- Databricks
- PostgreSQL
- S3
- Iceberg
- Kafka

### Transform Resources
- Map Filter
- Enrich
- Enrich Async
- SQL Join
- Rollup
- Fan Out

### Other Resources
- Pipelines
- Topics
- Tags

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (for development)

## Installation

```hcl
terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.1.18"
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

# Create a pipeline connecting source to destination
resource "streamkap_pipeline" "example" {
  name        = "my-pipeline"
  source_id   = streamkap_source_postgresql.example.id
  destination_id = streamkap_destination_snowflake.example.id
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
│   │   └── tag/         # Tag resource
│   ├── datasource/      # Data sources
│   └── helper/          # Utility functions
└── examples/            # Example Terraform configs
```

## Documentation

- [Streamkap Provider on Terraform Registry](https://registry.terraform.io/providers/streamkap-com/streamkap)
- [Streamkap Documentation](https://docs.streamkap.com)
- [API Reference](https://api.streamkap.com/openapi.json)

## Upgrading

See [MIGRATION.md](docs/MIGRATION.md) for guidance on upgrading from previous versions, including breaking changes and deprecated attributes.

## License

Mozilla Public License 2.0
