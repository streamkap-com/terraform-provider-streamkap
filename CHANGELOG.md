# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
