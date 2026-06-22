# Changelog

## Unreleased

## 2.2.0 (June 22, 2026)

### Added

* **PostgreSQL source**: New optional `heartbeat_use_logical_message` (bool, default `false`). When `heartbeat_enabled = true` and this is set, the connector runs `SELECT pg_logical_emit_message(true, ...)` on each beat to advance the replication slot — works on PG14+ primaries with a SELECT-only role and is compatible with read-only mode. No `streamkap_heartbeat` table or write grant required on the source. Resolves ENG-2398.

### Changed

* **PostgreSQL/MySQL sources**: Document Kafka-only heartbeat mode. Setting `heartbeat_enabled = true` while leaving `heartbeat_data_collection_schema_or_database` unset/null now keeps the connector polling on low-traffic sources without requiring a `streamkap_heartbeat` table or write grant in the source DB. No schema change — the field has always been optional in the provider; this just lights up an existing path that the backend used to reject.

## 2.1.22 (June 22, 2026)

### Fixed

* **Provider error handling**: API errors that return a non-JSON body (gateway HTML pages, WAF blocks, proxy 5xx) now surface as `unexpected <status> <status text> from <method> <url>: <body snippet>` instead of the cryptic `invalid character '<' looking for beginning of value`. The HTTP status, URL, and a truncated body snippet are included directly in the Terraform error, so failures like 504 gateway timeouts on long-running operations are diagnosable without re-running with `TF_LOG=DEBUG`. Resolves ENG-2460.

* **Resource read**: When a resource (source, destination, pipeline) is deleted out-of-band — for example via the Streamkap UI, ops cleanup, or a prior failed `terraform destroy` — the `Read` handler now removes it from Terraform state instead of returning `... does not exist`. `terraform refresh` / `terraform plan` recover automatically and propose recreating the resource; previously the only workaround was `terraform state rm`. Resolves ENG-2461.
