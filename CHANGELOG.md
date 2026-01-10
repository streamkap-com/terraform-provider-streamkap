# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Configurable timeouts for all resources (create, update, delete operations)
- Automatic retry with exponential backoff for transient errors
- Unit tests for helper functions and retry logic

### Changed
- Improved error handling with retry for transient failures
- Default timeouts: create/update/delete 20m

### Technical Details
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
