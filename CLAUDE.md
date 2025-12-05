# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Terraform provider for Streamkap (data streaming platform). Built with Terraform Plugin Framework. Provider address: `github.com/streamkap-com/streamkap`

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
- **Resources** (`internal/resource/`): Source connectors (PostgreSQL, MySQL, MongoDB, DynamoDB, SQL Server), Destination connectors (Snowflake, ClickHouse, Databricks, PostgreSQL, S3, Iceberg), Pipelines, Topics
- **Data Sources** (`internal/datasource/`): Transforms, Tags
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
