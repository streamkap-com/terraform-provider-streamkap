# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Terraform provider for Streamkap (data streaming platform). Built with Terraform Plugin Framework. Provider address: `github.com/streamkap-com/streamkap`

## Related Backend

This Terraform provider is built against the Streamkap Python FastAPI backend. Set the `STREAMKAP_BACKEND_PATH` environment variable to point to your local clone of the backend repository. Use this repository to validate API logic, understand endpoint behavior, check request/response schemas, and debug integration issues.

**OpenAPI Specification**: https://api.streamkap.com/openapi.json — use this to explore available endpoints, request/response schemas, and API documentation.

### Key Backend Files
When debugging or adding new connectors, these backend locations are most relevant:
- `app/api/sources_api.py`, `app/api/destinations_api.py` — API endpoint definitions
- `app/models/api/sources/common.py`, `app/models/api/destinations/common.py` — Pydantic request/response models
- `app/sources/plugins/{connector}/` — Source connector plugins (config schemas, validation)
- `app/destinations/plugins/{connector}/` — Destination connector plugins
- `app/utils/entity_changes.py` — CRUD logic, `created_from` handling

### Plugin Structure
Each connector plugin folder contains:
- `configuration.latest.json` — Current config schema (required fields, defaults, validation rules)
- `dynamic_utils.py` — Runtime config resolution and derived values

### Connector Status Values
The `connector_status` field (read-only, computed) can be: `Active`, `Paused`, `Stopped`, `Broken`, `Starting`, `Unassigned`, `Unknown`

### Detailed Reference Documents
For in-depth understanding of backend patterns, see the audit documents in `docs/audits/`:
- **Entity Configuration Schema Audit** (`docs/audits/entity-config-schema-audit.md`) — Complete reference for `configuration.latest.json` schema structure, control types, value objects, conditional logic, and Terraform mapping rules
- **Backend Code Reference Guide** (`docs/audits/backend-code-reference.md`) — API endpoints, Pydantic models, CRUD flows, dynamic resolution, multi-tenancy, and debugging guide

## Provider Refactor Plan
For the ongoing architecture refactor (config-driven code generation), see:
- **Design Document** (`docs/plans/2026-01-08-provider-refactor-design.md`) — Full refactor plan with best practices, implementation tasks, and success metrics

## Commands

### Build and Install
```bash
go install .              # Build and install provider to $GOPATH/bin
go generate              # Generate/update documentation
```

### Testing
```bash
make testacc            # Run acceptance tests (creates real resources)

# Run tests with specific settings
TF_ACC=1 STREAMKAP_HOST=https://api.streamkap.com STREAMKAP_CLIENT_ID=client_id STREAMKAP_SECRET=secret go test ./... -v -timeout 120m

# Run single test
go test ./internal/provider -v -run TestAccSourcePostgreSQL_basic
```

### Local Development Setup
Configure `~/.terraformrc` to use local build:
```hcl
provider_installation {
  dev_overrides {
    "github.com/streamkap-com/streamkap" = "$GOBIN_PATH"  # Replace with your $GOPATH/bin
  }
  direct {}
}
```

After setup: `go install .` then use provider in Terraform configs. See examples in `examples/` directory.

## Architecture

### Core Components
- **Provider** (`internal/provider/provider.go`): Registers all resources/datasources, handles authentication via OAuth2 token exchange
- **API Client** (`internal/api/`): HTTP client implementing `StreamkapAPI` interface with methods for each resource type
- **Resources** (`internal/resource/`): Source connectors (PostgreSQL, MySQL, MongoDB, DynamoDB, SQL Server, KafkaDirect), Destination connectors (Snowflake, ClickHouse, Databricks, PostgreSQL, S3, Iceberg, Kafka), Transform resources (MapFilter, Enrich, EnrichAsync, SQLJoin, Rollup, FanOut), Pipelines, Topics
- **Data Sources** (`internal/datasource/`): Transforms, Tags

### Transform API
The API client provides CRUD operations for Transform resources:
- `CreateTransform` - Create a new transform
- `GetTransform` - Retrieve a transform by ID
- `UpdateTransform` - Update an existing transform
- `DeleteTransform` - Delete a transform

### Deprecated Attributes
Some attribute names have been deprecated but still work with backward compatibility. See [MIGRATION.md](docs/MIGRATION.md) for the full list of deprecated attributes and migration guidance.
- **Helpers** (`internal/helper/helper.go`): Type conversion between API responses and Terraform types

### API Client Pattern
All API operations go through `streamkapAPI.doRequest()` which:
- Adds `Authorization: Bearer <token>` header
- Handles errors from API `detail` field
- All Create operations inject `created_from: constants.TERRAFORM` to track resource origin

### Resource Implementation Pattern
Each resource (source/destination):
1. Has a `connector_code` string identifying the integration type (e.g., "postgresql", "snowflake")
2. Stores connector-specific config in `Config map[string]any` field (flat structure, not nested)
3. Implements standard Terraform Plugin Framework interfaces: `Resource`, `ResourceWithConfigure`, `ResourceWithImportState`
4. Uses helper functions to convert API map responses to typed Terraform attributes:
   - `helper.GetTfCfgString(cfg, "key")` → `types.String`
   - `helper.GetTfCfgInt64(cfg, "key")` → `types.Int64` (handles string or numeric)
   - `helper.GetTfCfgBool(cfg, "key")` → `types.Bool`
   - `helper.GetTfCfgListString(ctx, cfg, "key")` → `types.List`

### Provider Configuration
Three parameters (all support env vars as fallback):
- `host` - API endpoint (default: `https://api.streamkap.com`, env: `STREAMKAP_HOST`)
- `client_id` - Required (env: `STREAMKAP_CLIENT_ID`)
- `secret` - Required, sensitive (env: `STREAMKAP_SECRET`)

### Adding New Resources
1. Create file in `internal/resource/source/` or `internal/resource/destination/`
2. Define model struct with `tfsdk` tags
3. Implement Schema() with fields, defaults, validators, plan modifiers
4. Implement CRUD methods using generic API client methods
5. Register in `internal/provider/provider.go` Resources() list
6. Add example to `examples/resources/streamkap_<name>/`
7. Add test in `internal/provider/<name>_resource_test.go`

### API Quirks
- Source Create/Read operations use `?secret_returned=true` query parameter to include sensitive fields in response
- Use `stringplanmodifier.UseStateForUnknown()` for computed fields to prevent spurious diffs

## AI-Agent Description Standards

This provider is optimized for the Terraform MCP Server. When adding or modifying resources, follow these patterns:

### Schema-Level Descriptions
Every resource/data source must have both `Description` and `MarkdownDescription`:

```go
Description: "Manages a PostgreSQL source connector.",
MarkdownDescription: "Manages a **PostgreSQL source connector**.\n\n" +
    "This resource creates and manages a PostgreSQL source for Streamkap data pipelines.\n\n" +
    "[Documentation](https://docs.streamkap.com/streamkap-provider-for-terraform)",
```

### Attribute Descriptions
All attributes must include:

1. **Enum fields** - List valid values:
   ```go
   Description: "Insert mode. Valid values: insert, upsert."
   MarkdownDescription: "Insert mode. Valid values: `insert`, `upsert`."
   ```

2. **Fields with defaults** - Document the default:
   ```go
   Description: "Database port. Defaults to \"5432\"."
   MarkdownDescription: "Database port. Defaults to `5432`."
   ```

3. **Sensitive fields** - Add security note:
   ```go
   Description: "Database password. This value is sensitive and will not appear in logs or CLI output."
   MarkdownDescription: "Database password.\n\n**Security:** This value is marked sensitive and will not appear in CLI output or logs."
   ```

### Example Files
Each resource needs two example files:
- `examples/resources/streamkap_<name>/basic.tf` - Minimal required configuration
- `examples/resources/streamkap_<name>/complete.tf` - All available options with comments

### tfgen Code Generator
The `cmd/tfgen` tool automatically generates schemas with these patterns from backend `configuration.latest.json` files:

```bash
# Regenerate all schemas
STREAMKAP_BACKEND_PATH=/path/to/python-be-streamkap go generate ./...

# Generate specific connector
go run ./cmd/tfgen generate --backend-path=$STREAMKAP_BACKEND_PATH --entity-type sources --connector postgresql
```

See `docs/AI_AGENT_COMPATIBILITY.md` for complete AI integration guidelines.