# Changelog

## [Unreleased]

### Fixed

- Report pipeline and ClickHouse `topics_config_map` response-mapping errors
  instead of silently saving incomplete Terraform state.

### Documentation

- Correct v2 installation examples and stable-v3 guidance, and link to the
  migration guide. Keep example version constraints on v2 and replace test
  fixture IDs with explicit inputs.

## [2.2.1] - 2026-09-15

### Fixed

- Keep legacy SQL Server snapshot thresholds and per-table chunk settings, and
  S3 filename prefixes, in Terraform state without sending them to the current
  backend, which no longer supports these settings. This avoids null-echo apply
  failures while warning that the settings no longer affect connector behavior.

- Stop logging source and destination request bodies and raw resource models,
  which could expose configured credentials in provider logs.

- Patch gRPC and Go networking/text dependencies for reported vulnerabilities,
  build with Go 1.27.1, and require vulnerability analysis before publishing.

- Run credential checks inside acceptance-test setup so offline release checks
  work without API credentials.

- Validate release tags, branch ancestry, changelog entries and offline checks
  before publishing v2 patches, including after the legacy branch is renamed.

### Maintenance

- Preserve this release line on `v2`; v3 development now lives on `main`.

- From v3 stable, v2 receives bug and security fixes only through 15 October
  2026. Support ends on 16 October 2026.

## 2.2.0 (June 22, 2026)

### Added

* **PostgreSQL source**: New optional `heartbeat_use_logical_message` (bool, default `false`). When `heartbeat_enabled = true` and this is set, the connector runs `SELECT pg_logical_emit_message(true, ...)` on each beat to advance the replication slot — works on PG14+ primaries with a SELECT-only role and is compatible with read-only mode. No `streamkap_heartbeat` table or write grant required on the source.

### Changed

* **PostgreSQL/MySQL sources**: Document Kafka-only heartbeat mode. Setting `heartbeat_enabled = true` while leaving `heartbeat_data_collection_schema_or_database` unset/null now keeps the connector polling on low-traffic sources without requiring a `streamkap_heartbeat` table or write grant in the source DB. No schema change — the field has always been optional in the provider; this just lights up an existing path that the backend used to reject.

## 2.1.22 (June 22, 2026)

### Fixed

* **Provider error handling**: API errors that return a non-JSON body (gateway HTML pages, WAF blocks, proxy 5xx) now surface as `unexpected <status> <status text> from <method> <url>: <body snippet>` instead of the cryptic `invalid character '<' looking for beginning of value`. The HTTP status, URL, and a truncated body snippet are included directly in the Terraform error, so failures like 504 gateway timeouts on long-running operations are diagnosable without re-running with `TF_LOG=DEBUG`.

* **Resource read**: When a resource (source, destination, pipeline) is deleted out-of-band — for example via the Streamkap UI, ops cleanup, or a prior failed `terraform destroy` — the `Read` handler now removes it from Terraform state instead of returning `... does not exist`. `terraform refresh` / `terraform plan` recover automatically and propose recreating the resource; previously the only workaround was `terraform state rm`.
