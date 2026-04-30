# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Branches

- **`main`** — v2.x stable (this branch). Bug fixes and back-compatible changes only. Default target for hotfix branches that need to ship to current released users.
- **`develop`** — v3.x beta. New architecture (codegen-driven schemas via `cmd/tfgen`, additional resources like `kafka_user` and `client_credential`, deprecated-attribute aliases). Default target for new feature work.

**Do not merge `develop` into `main`** until v3 is promoted from beta to stable. Until then, treat the two lines as independent: a fix that needs to ship to v2 users goes to `main` directly (and may need a separate cherry-pick to `develop`); v3-only work stays on `develop`. Cut feature branches from the line you're targeting.

If you're reading this file, you're on `main` — assume v2.x semantics: hand-written resource schemas in `internal/resource/{source,destination}/`, no code generation, no `internal/generated/` directory.

## Public repository — content hygiene

This repo is public. Do not commit internal ticket IDs/URLs, customer or tenant identifiers, real credentials, internal hostnames, verbatim production traces, or unannounced roadmap details. Public GitHub issue numbers (`#60`) are fine. Prefer `https://api.streamkap.com` and `https://docs.streamkap.com` when referencing surfaces.

## Related backend

The provider is built against the Streamkap Python FastAPI backend. Useful areas when reasoning about API behavior:

- `app/api/{sources,destinations,auth}_api.py` — endpoint definitions
- `app/models/api/{sources,destinations,app_auth}/` — Pydantic request/response models
- `app/utils/entity_changes.py` — CRUD logic and `created_from` handling

OpenAPI: `https://api.streamkap.com/openapi.json`.

## Overview

Terraform provider for Streamkap (data streaming platform). Built with Terraform Plugin Framework. Provider address: `github.com/streamkap-com/streamkap`.

## Documentation map

| Doc | What it covers |
|---|---|
| `README.md` | User-facing intro, install, basic usage. |
| `docs/index.md` | Auto-generated registry landing page (do not hand-edit; regen via `go generate`). |
| `docs/resources/`, `docs/data-sources/` | Auto-generated per-resource pages (regen via `go generate`). |
| `CHANGELOG.md` | User-visible changes per release. Update before tagging. |
| `examples/` | Example configs surfaced on the registry; per-resource and provider-level. |

If you add or remove a resource or data source, update the registration in `internal/provider/provider.go`, add an `examples/resources/streamkap_<name>/` example, and re-run `go generate` to refresh `docs/`. Add a `CHANGELOG.md` entry for any user-visible behavior change.

## Commands

```bash
# Build and install
go install .              # Build and install provider to $GOBIN (used by dev_overrides)
go generate               # Regenerate docs/ from descriptions + examples/

# Testing
make testacc              # Acceptance tests, TF_ACC=1, hits real API, ~120m timeout

# Run tests with explicit settings
TF_ACC=1 STREAMKAP_HOST=https://api.streamkap.com STREAMKAP_CLIENT_ID=... STREAMKAP_SECRET=... \
  go test ./... -v -timeout 120m

# Single test
go test ./internal/provider -v -run TestAccSourcePostgreSQL_basic
```

Acceptance tests create and destroy real resources against the configured tenant — run them against a non-production tenant or one you own.

### Local Development Setup

Configure `~/.terraformrc` to use the local build:

```hcl
provider_installation {
  dev_overrides {
    "github.com/streamkap-com/streamkap" = "$GOBIN_PATH"  # your $GOPATH/bin
  }
  direct {}
}
```

After setup: `go install .` then reference the provider from your Terraform configs. See `examples/` for working configs.

## Architecture

### Core Components
- **Provider** (`internal/provider/provider.go`): Registers all resources and data sources, handles authentication via OAuth2 token exchange.
- **API Client** (`internal/api/`): HTTP client implementing `StreamkapAPI` interface with one file per resource type (`auth.go`, `client.go`, `source.go`, `destination.go`, `pipeline.go`, `topic.go`, `tag.go`, `transform.go`).
- **Resources** (`internal/resource/`): `source/` (PostgreSQL, MySQL, MongoDB, DynamoDB, SQL Server, Kafka Direct), `destination/` (Snowflake, ClickHouse, Databricks, PostgreSQL, S3, Iceberg, Kafka), `pipeline/`, `topic/`.
- **Data Sources** (`internal/datasource/`): `transform`, `tag`.
- **Helpers** (`internal/helper/helper.go`): Type conversion between API responses (`map[string]any`) and Terraform attribute types.

### API Client Pattern
All API operations go through `streamkapAPI.doRequest()` which:
- Adds `Authorization: Bearer <token>` header.
- Unwraps API errors from the `detail` field.
- All Create operations inject `created_from: constants.TERRAFORM` to track resource origin.

### Resource Implementation Pattern
Each connector resource (source / destination):
1. Has a `connector_code` string identifying the integration (e.g., `"postgresql"`, `"snowflake"`).
2. Stores connector-specific config in a flat `Config map[string]any` field (no nested objects on the wire).
3. Implements the standard Terraform Plugin Framework interfaces: `Resource`, `ResourceWithConfigure`, `ResourceWithImportState`.
4. Uses helper functions to convert API map values to typed Terraform attributes:
   - `helper.GetTfCfgString(cfg, "key")` → `types.String`
   - `helper.GetTfCfgInt64(cfg, "key")` → `types.Int64` (accepts string or numeric input)
   - `helper.GetTfCfgBool(cfg, "key")` → `types.Bool`
   - `helper.GetTfCfgListString(ctx, cfg, "key")` → `types.List`

### Provider Configuration
Three parameters (all support env vars as fallback):
- `host` — API endpoint (default: `https://api.streamkap.com`, env: `STREAMKAP_HOST`)
- `client_id` — required (env: `STREAMKAP_CLIENT_ID`)
- `secret` — required, sensitive (env: `STREAMKAP_SECRET`)

### Adding New Resources
1. Create the file in `internal/resource/source/` or `internal/resource/destination/`.
2. Define the model struct with `tfsdk` tags.
3. Implement `Schema()` with fields, defaults, validators, plan modifiers.
4. Implement CRUD methods using the generic API client.
5. Register in `internal/provider/provider.go` `Resources()` list.
6. Add an example to `examples/resources/streamkap_<name>/`.
7. Add an acceptance test in `internal/provider/<name>_resource_test.go`.
8. Run `go generate` to refresh `docs/`.

## Generated Files — Do Not Edit

The `docs/` directory is generated by `tfplugindocs` (via `go generate`). **Never modify files under `docs/` directly** — they regenerate from schema descriptions plus `examples/` on every run, so hand-edits are silently overwritten.

## API Quirks (non-obvious)

- **Sources Create/Read/Update/Delete** all use `?secret_returned=true` so sensitive fields round-trip in responses.
- **List endpoints** default `page_size=10` (max 100). `ListSources / ListDestinations / ListPipelines` should paginate until `resp.Total`; anything else silently truncates tenants with more than 10 resources.
- **`/sources`, `/destinations`, `/pipelines` filters** accept `partial_name` only — no exact-name filter.
- **Create returns 422 "already exists"** when a non-deleted record with the same `{tenant_id, name}` exists. List additionally filters by `service_id`, so a record orphaned in a different service of the same tenant triggers 422 but is invisible to list.
- **Plan modifiers**: use `stringplanmodifier.UseStateForUnknown()` for computed fields that don't change to avoid spurious diffs.
- **Connector status** values (read-only): `Active`, `Paused`, `Stopped`, `Broken`, `Starting`, `Unassigned`, `Unknown`.

## Conventions

- Provider address: `github.com/streamkap-com/streamkap`. Go 1.x as pinned in `go.mod`. Terraform Plugin Framework.
- Every resource and data source needs both `Description` and `MarkdownDescription` so registry docs and AI-assisted tools have something to read.
- Each resource needs at least one example under `examples/resources/streamkap_<name>/`.
- For sensitive fields, mark `Sensitive: true` and document the security implication in the description.
