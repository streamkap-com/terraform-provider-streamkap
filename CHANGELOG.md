# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- **`streamkap_source_kafkadirect` and `streamkap_destination_kafkadirect` no
  longer expose configuration attributes the backend rejects.** tfgen merged
  `configurations_for_all.json` (the shared source/destination config) into every
  connector, but the backend's `_load_global_configuration()` returns `{}` for
  `kafkadirect` — it resolves against its plugin config alone. As a result both
  resources advertised ~30 phantom attributes (`quote_identifiers`,
  `preserve_null_values`, every `transforms_*`, `consumer_override_max_poll_records`,
  the source's `insert_topic_name_enabled` / `transforms_value_to_key_*`, etc.)
  that are not valid Kafka Direct config. tfgen now skips the common-config merge
  for `kafkadirect`, matching the backend. The destination keeps `password` and
  `whitelist_ips`; the source keeps `topic_prefix`, `topic_include_list`,
  `format`, and `schemas_enable` (plus the `kafka_format` deprecated alias).
  Configurations that set any of the removed attributes must drop them.

## [3.0.0-beta.20] - 2026-06-09 (Pre-release)

### Fixed
- **Numeric defaults stored as strings in the backend schema are now emitted
  correctly instead of `0`.** tfgen read `number`/`slider`-control defaults via
  a getter that only handled JSON numbers, so fields whose backend default is a
  string (e.g. `"180000"`) fell through to `0`. With `Optional + Computed +
  Default`, omitting such a field made Terraform send `0` to the API, overriding
  the backend's intended default. Affected attributes and their corrected
  defaults: `streamkap_source_dynamodb` `poll_timeout_ms` (`0` → `180000`),
  `signal_kafka_poll_timeout_ms` (`0` → `1000`), `incremental_snapshot_chunk_size`
  (`0` → `32768`), `incremental_snapshot_max_threads` (`0` → `8`),
  `full_export_expiration_time_ms` (`0` → `86400000`); and
  `streamkap_destination_databricks` `connection_timeout` (`0` → `180`).
  Resources that left these unset will show a one-time plan diff aligning them
  to the correct default.

## [3.0.0-beta.19] - 2026-06-03 (Pre-release)

### Fixed
- **`streamkap_destination_postgresql` docs now list the `TopicRegexRouter`
  attributes.** The schema fields shipped in beta.18, but their registry
  documentation was missing because `go generate ./...` ran `tfplugindocs`
  (root `main.go`) before `tfgen` (`internal/generated`), rendering docs against
  the pre-regen schema. The `make generate` target now runs `tfgen` first, then
  `tfplugindocs`, so docs always match the freshly generated schemas.

## [3.0.0-beta.18] - 2026-06-03 (Pre-release)

### Added
- **`streamkap_destination_postgresql` now exposes the `TopicRegexRouter`
  transforms.** Four optional attributes let you rename destination tables by
  applying up to two regex-based rewrite rules to the incoming topic name:
  `transforms_topic_regex_router1_regex` / `transforms_topic_regex_router1_replacement`
  and `transforms_topic_regex_router2_regex` / `transforms_topic_regex_router2_replacement`.
  Both replacement fields default to `$0` (the full match); leave a regex empty
  to skip that rule.

### Fixed
- **Regenerated provider docs to match the committed schemas.** Several
  `docs/resources/*.md` pages had drifted ahead of the generated Go schemas
  (showing `post_processors`, `heartbeat_use_logical_message`,
  ClickHouse `table_name_prefix`, and richer heartbeat/SSH descriptions that
  only exist on backend feature branches, not production). The docs now reflect
  the schemas the provider actually ships. No schema change.

## [3.0.0-beta.17] - 2026-05-21 (Pre-release)

### Added
- **`streamkap_pipeline` now supports `topic_auto_discovery_transforms`.** This
  optional list of `{ transform_id, regex }` rules lets a pipeline automatically
  pick up a transform's output topics whose names match a regex, without
  enumerating them in `transforms[].topics`. Use it when a transform produces
  topics whose names are generated dynamically (e.g. a topic-router / fan-out
  transform) and are therefore not known when the pipeline is created. Topic
  resolution happens server-side; the list is stored and returned unchanged.

### Fixed
- **Map config attributes are restored to their correct `map` type** (regression
  introduced in earlier v3 betas). A code-generation bug (`tfgen` resolved
  `overrides.json` relative to the working directory, so `go generate ./...`
  loaded zero overrides) caused three map attributes to be emitted as plain
  strings — and `snapshot_custom_table_config` to be renamed to
  `streamkap_snapshot_custom_table_config`:
    - `streamkap_destination_snowflake.auto_qa_dedupe_table_mapping`
    - `streamkap_destination_clickhouse.topics_config_map`
    - `streamkap_source_sqlserver.snapshot_custom_table_config`
  These now match v2.1.x (and the intended v3 design): map attributes under their
  original names. **If you set any of these as a string on a v3 beta ≤ beta.16,
  revert to the map form** (see `docs/MIGRATION.md`). Migrating from v2.1.x
  requires no change — these were always maps there. The override path is now
  resolved cwd-independently so a full regen can't silently drop it again.
- **`connector_status` no longer triggers "Provider produced inconsistent
  result after apply" when updating a source or destination.** Updates are sent
  with `wait=false`, so the API response reports the transient desired-state
  `Pending Update` while the plan still carries the prior known status
  (e.g. `Active`). The provider was overwriting state with that transient value,
  which Terraform rejected as plan/apply divergence. The Update path now keeps
  the planned status (it is volatile and refreshed by Read; the backend
  reconciler settles the connector back to `Active` within seconds).
- **Boolean connector config fields no longer drift on every apply.** The
  backend returns connector config as string-encoded values, so booleans arrive
  as `"true"` / `"false"`. `helper.GetTfCfgBool` only accepted a native Go
  `bool` and returned null for the string form, so any bool field the backend
  backfills with a default (e.g. PostgreSQL destination
  `transforms_mark_columns_as_required_fields_include_all`,
  `transforms_oversized_records_replace_null_with_default`,
  `transforms_to_decimal_j_truncate_to_max_precision`,
  `transforms_to_jsonb_j_convert_all_json`) read back null and showed a
  perpetual diff. `GetTfCfgBool` now coerces string-encoded booleans, mirroring
  `GetTfCfgInt64` / `GetTfCfgFloat64`. Fixes the drift for every bool attribute
  on every connector.
- **`streamkap_pipeline` `source.topics` is hardened against the issue #78 bug
  class.** `api2Model` now strips a leading `source_<id>.` prefix from returned
  source topics (only that exact prefix — never a naive split, so dotted names
  like `default.MyTable` survive), so state stays stable across plan/apply even
  if the API echoes the raw `topic_id` form or state was written by an older
  beta. Note for DynamoDB sources: send source topics in the source's catalog
  form (e.g. `default.<table>`) so they register as selected in the pipeline.

## [3.0.0-beta.16] - 2026-05-11 (Pre-release)

### Changed
- **`streamkap_pipeline` and `streamkap_transform_*` no longer auto-adopt on
  duplicate-name conflicts (409 / 422 "already exists").** The previous
  adopt-on-conflict path was unsafe under
  `lifecycle { create_before_destroy = true }`: Terraform creates the new
  instance first while the deposed instance still occupies the
  `{tenant_id, service_id, name}` slot, the provider would adopt the
  deposed's backend record (returning the same backend id from Create), and
  Terraform's subsequent destroy of the deposed entry would DELETE the same
  backend record the live state had just been pointed at — silently
  destroying the customer's live pipeline / transform. A customer hit this on
  pipelines and had deposed transforms in the same state that would have
  triggered the same issue once the pipeline path was unblocked
  (trace in `0_Terraform Apply.txt`). The provider now refuses to auto-adopt
  on these two resources and surfaces a clear error with two recovery paths:
    1. Remove `create_before_destroy = true` from the resource's
       `lifecycle` (Streamkap enforces unique pipeline / transform names per
       tenant + service, so that lifecycle does not work).
    2. `terraform import streamkap_pipeline.<name> <pipeline_id>` or
       `terraform import streamkap_transform_<type>.<name> <transform_id>`
       when recovering from a previous apply whose response was lost.
  The original adopt-on-conflict still applies to `streamkap_source_*`,
  `streamkap_destination_*`, and `streamkap_tag`. The **same data-loss path
  exists in principle** for those resources — adopt-on-conflict under
  `create_before_destroy` is unsafe for any backend that enforces
  name-uniqueness, regardless of resource type. We are holding their adopt
  path back from this PR because (a) the reported customer wasn't hitting
  it on those resources, (b) existing acceptance tests rely on adopt-on-422
  behavior there, and (c) tightening them would break legitimate
  timed-out-create recovery workflows. This is **not** a statement that
  they are safe; track removing their adopt path in a follow-up before any
  customer relies on `lifecycle { create_before_destroy = true }` against
  those resources.
- **`streamkap_transform_*` — `deploy = true` with `replay_window = "0"`
  failed with "Replay window must be a valid duration like 3d, 10m, 2h, 30s".**
  The provider's schema documents `"0"` as "continue from last position"
  (semantically identical to leaving the field unset → backend's
  `start_time = None` → use the latest offset). But the backend's preview-
  deploy parser uses a strict `(\d+)([smhd])` regex
  (`python-be-streamkap/app/utils/api/v2/api_transforms_utils.py:352`) that
  rejects a bare `"0"`. The deploy client now treats `"0"` and `""` as
  equivalent and omits the query parameter in both cases, restoring the
  documented behavior without requiring a backend change.
- **Non-JSON error responses surface a useful message.** When an upstream
  layer (nginx, load balancer, ingress) returned HTML or another non-JSON
  body for a 5xx, the client previously bubbled up a bare
  `invalid character '<' looking for beginning of value` JSON-decode error
  — the HTTP status and body content were silently dropped, which dead-ended
  customer debugging. Error responses that fail to decode as the standard
  `{"detail": "..."}` shape now surface the actual status code and a
  truncated body snippet so the real failure is visible.

## [3.0.0-beta.15] - 2026-05-08 (Pre-release)

### Added
- **Tags on every entity.** `streamkap_source_*`, `streamkap_destination_*`,
  `streamkap_transform_*`, and `streamkap_topic` now expose a `tags = [...]`
  attribute (Set of tag IDs) — at parity with `streamkap_pipeline`, which
  already had it. Optional+Computed: unset in config preserves backend tags so
  out-of-band attachments don't churn plans; explicitly set means Terraform
  owns the field and reverts manual UI edits; `tags = []` clears the entity's
  tags. Backend distinguishes `null`/absent (keep) from `[]` (clear) on the
  wire and the provider honors that.
- **`streamkap_tags` data source.** Lists tags filtered by `filter_name`,
  `filter_type`, or `filter_ids`. Routes through `GET /tags`, falls back to
  `POST /tags/search` automatically when `filter_ids` is large enough that the
  URL would risk a length limit. Use it instead of hardcoding tag IDs.
- **`streamkap_tag.type` plan-time validation.** Element values are now
  validated against the backend `TagTypeEnum` (`environment`, `general`,
  `sources`, `destinations`, `pipelines`, `transforms`, `topics`, `services`,
  `users`, `tenant`). Typos fail at plan, not after a round-trip.
- **Two previously-unwired source connectors** (backend exposed them, the
  provider didn't): `streamkap_source_salesforce_webhook`,
  `streamkap_source_zendesk_webhook`.

### Fixed
- **`streamkap_topic` Read silently no-op'd against the API response.** The
  internal API struct mapped `topic_id` and a top-level `partition_count`, but
  the backend's `GET /topics/{id}` returns `id` and `kafka.partitions.count` —
  so every Read decoded into an empty struct and the resource only looked
  unchanged because prior state survived. `Topic.TopicID` now decodes from
  `id`, and `partition_count` is lifted from `kafka.partitions.count` after
  Unmarshal.
- **`streamkap_tag` adopt-on-exists.** When `CreateTag` returns "A tag with
  identical properties already exists" (e.g. a leaked test fixture or a
  parallel-apply race), the provider now looks the tag up by name via the
  list endpoint and adopts it instead of failing the apply — matching the
  existing behavior for sources, destinations, and transforms.
- **`streamkap_pipeline` — "planned set element does not correlate with any element in actual" on `transforms[].topics`.**
  When a transform attached to a pipeline had been deployed, the backend stamps its
  `topic_ids` with a `transform_<id>_<version>` prefix
  (`python-be-streamkap` `app/utils/api/v2/api_transforms_utils.py:427`). The pipeline
  read path then swapped that internal id into the Terraform state in place of the
  pretty topic name, so the next apply saw the user's pretty name in the plan and
  the full topic_id in state and aborted. State now always carries the pretty
  `topic` field returned by the backend, matching what users write in
  `transforms[*].topics` (e.g. `<schema>.<table>`).
- **`streamkap_transform_*` — `replay_window` left Unknown after Create.**
  `replay_window` is `Optional+Computed` with no default and the backend never
  echoes it back, so on initial Create with no prior state the framework's
  `UseStateForUnknown` plan modifier could not resolve the planned Unknown,
  leaving Terraform to error with "still indicated an unknown value." Create
  now concretizes the field to Null when the user did not set it.

---

## [3.0.0-beta.1] - 2026-03-04 (Pre-release)

> **Beta Release** — This is a pre-release version intended for testing and early feedback.
> It is **not recommended for production use**. If you are currently on v2.x, there is no
> urgency to upgrade. Wait for the stable v3.0.0 release unless you specifically need the
> new resources below and are comfortable with potential breaking changes in future betas.
>
> Install the beta explicitly:
> ```hcl
> version = "3.0.0-beta.1"
> ```

### Added
- **`streamkap_destination_weaviate` resource** - Weaviate vector database destination connector
- **`streamkap_kafka_user` resource** - Kafka user management with ACL-based topic access control
- **`streamkap_client_credential` resource** - API token (client credential) management for machine-to-machine authentication
- **`streamkap_roles` data source** - List available roles for client credential assignment
- **Data source tests** - Acceptance tests for all 6 data sources (tag, topic, topics, topic_metrics, transform, roles)
- **Schema backward compatibility snapshots** - Coverage for all 54 resources (was 16, now complete)
- **AGENTS.md** - AI coding agent guide following emerging standards for AI-assisted development
- **Provider Configuration Validation Tests** - Tests for missing credentials, empty values, environment variable fallbacks
- **Example File Validation Tests** - Tests to verify all example .tf files are valid HCL and have required files
- **429 Rate Limit Handling** - Retry logic now handles HTTP 429 (Too Many Requests) responses

### Changed
- **GNUmakefile** - Enhanced with comprehensive build, test, lint, and development targets including:
  - `make test` - Run unit tests
  - `make test-schema` - Run schema compatibility tests
  - `make test-validators` - Run validator tests
  - `make test-integration` - Run VCR integration tests
  - `make testacc` - Run acceptance tests
  - `make lint` - Run golangci-lint
  - `make cassettes` - Record VCR cassettes
  - `make snapshots` - Update schema snapshots
  - `make sweep` - Clean up orphaned test resources
  - `make validate-examples` - Validate example Terraform files

---

## [2.1.19+] - 2026-02-05 (Development)

### Added
- **Comprehensive Test Suite**
  - Acceptance tests for all 14 new source connectors (AlloyDB, DB2, DocumentDB, Elasticsearch, MariaDB, MongoDB Hosted, Oracle, Oracle AWS, PlanetScale, Redis, S3, Supabase, Vitess, Webhook)
  - Acceptance tests for all 15 new destination connectors (Azure Blob, BigQuery, CockroachDB, DB2, GCS, HTTP Sink, Kafka Direct, Motherduck, MySQL, Oracle, R2, Redis, Redshift, SQL Server, Starburst)
  - Acceptance tests for 3 additional transforms (EnrichAsync, FanOut, Rollup)
  - Migration tests for backward compatibility validation
  - Smoke tests for connectors without live credentials (Oracle, BigQuery, Redshift, Starburst, Motherduck)
  - Negative tests for API error handling (401, 404, 422, 5xx)
  - Schema validation tests for required fields, enums, and sliders
  - State conflict tests for drift detection and external modifications
  - Test sweepers for sources, destinations, transforms, and pipelines
- **Documentation**
  - Comprehensive testing documentation (`docs/TESTING.md`)
  - Code generator documentation (`docs/CODE_GENERATOR.md`)
  - Architecture comparison audit report (`docs/audits/architecture-comparison-report.md`)
  - Entity config schema audit (`docs/audits/entity-config-schema-audit.md`)
  - Backend code reference guide (`docs/audits/backend-code-reference.md`)
- **Examples**
  - Examples for Weaviate destination
  - All 20 sources, 23 destinations, and 6 transforms now have basic.tf and complete.tf examples
- **API Client Enhancements**
  - ListSources, ListDestinations, ListTransforms, ListPipelines methods for sweepers
- **Full Connector Coverage** - All 42 connectors now exposed:
  - **14 new source connectors**: AlloyDB, DB2, DocumentDB, Elasticsearch, MariaDB, MongoDB Hosted, Oracle, Oracle AWS, PlanetScale, Redis, S3, Supabase, Vitess, Webhook
  - **15 new destination connectors**: Azure Blob, BigQuery, CockroachDB, DB2, GCS, HTTP Sink, Kafka Direct, Motherduck, MySQL, Oracle, R2, Redis, Redshift, SQL Server, Starburst
  - Total: 20 sources + 22 destinations
- **AI-Agent Compatibility (Terraform MCP Server)**
  - All resources now have both `Description` (plain text) and `MarkdownDescription` (rich text)
  - Enum fields list all valid values in descriptions
  - Default values are documented in field descriptions
  - Sensitive fields include security notes
- Basic and complete example configurations for all resources
  - `basic.tf` - minimal required configuration
  - `complete.tf` - all available options with comments
- Examples for all 6 transform types (previously missing)
- Example for tag resource (previously missing)
- Resource-level documentation with links to Streamkap docs
- Configurable timeouts for all resources (create, update, delete operations)
- Automatic retry with exponential backoff for transient errors
- Unit tests for helper functions and retry logic

### Changed
- Updated inline GoDoc comments in `internal/resource/connector/base.go`, `cmd/tfgen/parser.go`, `cmd/tfgen/generator.go`
- Added comprehensive architecture documentation for three-layer pattern (Generated Schemas → Thin Wrappers → Shared Base Resource)
- Improved README with accurate connector counts (20 sources, 23 destinations, 8 transforms)
- Regenerated all connector schemas with comprehensive descriptions
- Restructured examples directory with `basic.tf` and `complete.tf` patterns
- Improved error handling with retry for transient failures
- Default timeouts: create/update/delete 20m

### Fixed
- Fixed typo "Tranform" → "Transform" in transform data source
- Fixed incorrect attribute names in destination examples (S3, Iceberg, PostgreSQL)
- Fixed pipeline example output reference bug
- Added missing `sensitive = true` to ClickHouse password variable

### Technical Details
- tfgen code generator now outputs both Description and MarkdownDescription
- Backend investigation confirmed KC timeout is 15s per server
- Conservative retry strategy: 3 attempts, 10s minimum delay
- Retry only on mutating operations (not reads)

## [2.0.0] - 2026-01-09

### Added
- **Transform Resources**: 6 new resource types for data transformation
  - `streamkap_transform_map_filter` - Filter and transform records
  - `streamkap_transform_enrich` - Enrich data with external sources
  - `streamkap_transform_enrich_async` - Async enrichment transforms
  - `streamkap_transform_sql_join` - SQL-based join transforms
  - `streamkap_transform_rollup` - Aggregation/rollup transforms
  - `streamkap_transform_fan_out` - Fan-out transforms for multiple outputs
- **Destination Kafka**: New Kafka destination connector
- **Destination Iceberg**: New Iceberg destination connector
- Comprehensive migration guide (`docs/MIGRATION.md`)
- Architecture documentation (`docs/ARCHITECTURE.md`)
- Developer guide (`docs/DEVELOPMENT.md`)

### Changed
- **Code Generation Architecture**: Provider now uses config-driven code generation
  - BaseConnectorResource pattern for sources and destinations
  - BaseTransformResource pattern for transforms
  - Generated schemas from backend configuration files
- Standardized attribute naming across all connectors

### Deprecated
- PostgreSQL source attributes (will be removed in v3.0):
  - `insert_static_key_field_1` → use `transforms_insert_static_key1_static_field`
  - `insert_static_key_value_1` → use `transforms_insert_static_key1_static_value`
  - `insert_static_value_field_1` → use `transforms_insert_static_value1_static_field`
  - `insert_static_value_1` → use `transforms_insert_static_value1_static_value`
  - `insert_static_key_field_2` → use `transforms_insert_static_key2_static_field`
  - `insert_static_key_value_2` → use `transforms_insert_static_key2_static_value`
  - `insert_static_value_field_2` → use `transforms_insert_static_value2_static_field`
  - `insert_static_value_2` → use `transforms_insert_static_value2_static_value`
  - `predicates_istopictoenrich_pattern` → use `predicates_is_topic_to_enrich_pattern`
- Snowflake destination attributes (will be removed in v3.0):
  - `auto_schema_creation` → use `create_schema_auto`

### Breaking Changes
- **PostgreSQL Source**:
  - `database_port`: Type changed from `Int64` to `String`
  - `signal_data_collection_schema_or_database`: Now required (was optional)
- **Snowflake Destination**:
  - `auto_qa_dedupe_table_mapping`: Type changed from `Map` to `String`

### Removed
- Legacy connector implementation files (replaced by generated code)

## [1.x.x] - Previous Releases

See [GitHub Releases](https://github.com/streamkap-com/terraform-provider-streamkap/releases) for previous versions.
