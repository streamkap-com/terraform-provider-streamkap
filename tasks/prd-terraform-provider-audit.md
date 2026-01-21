# PRD: Comprehensive Terraform Provider Audit & Verification

## Introduction

This PRD defines a complete end-to-end audit of the refactored Streamkap Terraform Provider (`ralph-refactored-terraform` branch). The audit will verify architecture integrity, implementation correctness, backward compatibility with `main` branch, code generator accuracy, backend alignment, AI-agent optimization, and testing completeness.

The audit compares the old (`main`) vs new (`ralph-refactored-terraform`) architecture, validates improvements, then achieves equal coverage across all 53 resources. Critical issues discovered during the audit will be fixed immediately before continuing.

**Key Decisions:**
- Deliverables: Comprehensive report + GitHub issues + automated fixes for critical issues
- Execution: Parallel execution of independent phases where possible
- Testing: Write full acceptance tests for ALL 37 missing resources
- Backend Alignment: Combination of manual verification + automated tooling
- Critical Issues: Pause audit, fix immediately, then resume
- VCR Cassettes: Record for all acceptance tests (CI-friendly)
- Test Sweepers: Implement cleanup logic for orphaned resources
- Missing Credentials: Add env var placeholders + smoke tests for untestable connectors
- Negative Tests: Write error/edge case tests as part of audit
- Documentation: Create/update all missing documentation, inline comments, and examples

---

## Goals

- Compare `main` vs `ralph-refactored-terraform` branch architecture patterns and document findings
- Validate the refactored three-layer architecture follows best practices
- Verify all CRUD operations work correctly for all 53 resources
- Confirm backward compatibility with existing `main` branch Terraform configs
- Validate `tfgen` code generator produces correct, compilable, functional schemas
- Verify field mappings match backend `configuration.latest.json` (100% alignment)
- Audit AI-agent descriptions for Terraform MCP Server optimization
- Achieve >80% test coverage with full acceptance tests for all resources
- Document all findings and create actionable GitHub issues

---

## User Stories

### Phase 1: Architecture Comparison Audit

#### US-001: Compare Main vs Refactored Architecture Patterns
**Description:** As a developer, I need to understand the architectural differences between branches so I can validate the refactoring improvements.

**Acceptance Criteria:**
- [ ] Document `main` branch architecture pattern (manual schemas, inline CRUD, ~500 LOC per connector)
- [ ] Document `ralph-refactored-terraform` three-layer pattern (generated/wrapper/base)
- [ ] Create comparison matrix covering: code per connector, schema definition, field mappings, CRUD implementation, type conversion, deprecation handling, timeout/import support
- [ ] Identify improvements and trade-offs for each architectural decision
- [ ] Typecheck passes (`go build ./...`)

#### US-002: Verify Layer Separation
**Description:** As a developer, I need to confirm the three layers (generated, wrapper, base) are properly separated.

**Acceptance Criteria:**
- [ ] Verify `internal/generated/*.go` files have "DO NOT EDIT" markers
- [ ] Verify `internal/resource/source/*_generated.go` wrapper files implement `ConnectorConfig` interface
- [ ] Verify `internal/resource/connector/base.go` contains shared CRUD logic
- [ ] Confirm no cross-layer pollution (generated code doesn't contain business logic)
- [ ] Count files: 48+ generated schemas, 48 wrapper files
- [ ] Typecheck passes

#### US-003: Document Architectural Decisions
**Description:** As a developer, I need a record of why architectural decisions were made for future reference.

**Acceptance Criteria:**
- [ ] Document rationale for code generation approach
- [ ] Document rationale for reflection-based marshaling
- [ ] Document rationale for thin wrapper pattern
- [ ] Document rationale for JSON-based deprecations/overrides
- [ ] Save to `docs/audits/architecture-comparison-report.md`

---

### Phase 2: Implementation Verification

#### US-004: Verify API Client Operations
**Description:** As a developer, I need to confirm the API client correctly handles authentication, headers, and error parsing.

**Acceptance Criteria:**
- [ ] OAuth2 token exchange works (`internal/api/client.go`)
- [ ] Authorization header set correctly on all requests
- [ ] `?secret_returned=true` query param applied for source Create/Read
- [ ] API `detail` field errors extracted and propagated
- [ ] Retry logic handles 502, 503, 504, timeouts (`internal/api/retry.go`)
- [ ] `created_from: terraform` injected on all create operations
- [ ] Typecheck passes

#### US-005: Verify Helper Functions
**Description:** As a developer, I need to confirm helper functions handle edge cases correctly.

**Acceptance Criteria:**
- [ ] `GetTfCfgString` handles nil, missing keys, type conversion
- [ ] `GetTfCfgInt64` handles float64, string numbers, nil
- [ ] `GetTfCfgBool` handles nil, missing keys
- [ ] `GetTfCfgListString` handles empty lists, nil elements
- [ ] `MigrateDeprecatedValues` copies old→new, removes old key
- [ ] Unit tests pass for all helper functions
- [ ] Typecheck passes

#### US-006: Verify Base Resource Implementation
**Description:** As a developer, I need to confirm the base connector resource handles all Terraform types correctly.

**Acceptance Criteria:**
- [ ] `modelToConfigMap` converts model struct to API map format
- [ ] `configMapToModel` converts API response to model struct
- [ ] `extractTerraformValue` handles String, Int64, Bool, List types
- [ ] `setTerraformValue` uses correct helper functions for each type
- [ ] Timeout context properly applied to all CRUD operations
- [ ] Import state passthrough works for all resource types
- [ ] Typecheck passes

---

### Phase 3: Backward Compatibility Verification

#### US-007: Run Migration Tests
**Description:** As a developer, I need to verify existing Terraform configs work unchanged with the refactored provider.

**Acceptance Criteria:**
- [ ] All 13 existing migration tests pass:
  - `TestAccSourcePostgreSQL_Migration`
  - `TestAccSourceMySQL_Migration`
  - `TestAccSourceMongoDB_Migration`
  - `TestAccSourceDynamoDB_Migration`
  - `TestAccSourceSQLServer_Migration`
  - `TestAccSourceKafkaDirect_Migration`
  - `TestAccDestinationSnowflake_Migration`
  - `TestAccDestinationClickhouse_Migration`
  - `TestAccDestinationDatabricks_Migration`
  - `TestAccDestinationPostgreSQL_Migration`
  - `TestAccDestinationS3_Migration`
  - `TestAccDestinationIceberg_Migration`
  - `TestAccDestinationKafka_Migration`
- [ ] Each test produces empty plan (no changes detected vs v2.1.18)
- [ ] Run: `TF_ACC=1 go test -v -timeout 180m -run 'TestAcc.*Migration' ./internal/provider/...`

#### US-008: Verify Deprecated Field Handling
**Description:** As a developer, I need to confirm deprecated fields work with proper warnings.

**Acceptance Criteria:**
- [ ] All deprecated fields in `cmd/tfgen/deprecations.json` are present in schemas
- [ ] Deprecated fields have `DeprecationMessage` set
- [ ] Using deprecated field shows warning but works correctly
- [ ] Verify PostgreSQL deprecated fields (8 fields):
  - `insert_static_key_field_1` → `transforms_insert_static_key1_static_field`
  - `insert_static_key_value_1` → `transforms_insert_static_key1_static_value`
  - (and 6 more per deprecations.json)
- [ ] Verify Snowflake deprecated fields:
  - `auto_schema_creation` → `create_schema_auto`
- [ ] Typecheck passes

#### US-009: Run Schema Compatibility Tests
**Description:** As a developer, I need to detect any breaking schema changes vs v2.1.18.

**Acceptance Criteria:**
- [ ] Run: `go test -v -run 'TestSchemaBackwardsCompatibility' ./internal/provider/...`
- [ ] All 16 existing schema snapshots pass
- [ ] No "required attribute removed" errors
- [ ] No "optional changed to required" errors
- [ ] Document any warnings (computed attribute changes)
- [ ] Typecheck passes

---

### Phase 4: Code Generator Verification

#### US-010: Verify Parser Implementation
**Description:** As a developer, I need to confirm the parser correctly reads backend configuration files.

**Acceptance Criteria:**
- [ ] JSON parsing reads `configuration.latest.json` correctly
- [ ] Only `user_defined: true` fields extracted
- [ ] Field metadata extracted: control, type, default, required, sensitive
- [ ] Enum values extracted from `raw_values` for one-select
- [ ] Slider bounds extracted (min/max/step)
- [ ] Conditional fields parsed from conditions array
- [ ] Typecheck passes

#### US-011: Verify Generator Implementation
**Description:** As a developer, I need to confirm the generator produces correct Go code.

**Acceptance Criteria:**
- [ ] Type mapping correct:
  - `string` → `types.String`
  - `password` → `types.String` (sensitive)
  - `number` → `types.Int64`
  - `boolean/toggle` → `types.Bool`
  - `one-select` → `types.String` + validator
  - `multi-select` → `types.List`
  - `slider` → `types.Int64` + range validator
- [ ] Port fields (ending in `_port`) convert to Int64
- [ ] Go abbreviations uppercased: ID, SSH, SSL, SQL, DB, URL, API, AWS, ARN, QA
- [ ] Default generation uses correct functions (stringdefault, int64default, booldefault)
- [ ] Validators generated correctly (OneOf, Between)
- [ ] Plan modifiers generated (UseStateForUnknown, RequiresReplace)
- [ ] `encrypt=true` → `Sensitive=true`
- [ ] Descriptions enhanced with defaults, valid values, security notes
- [ ] Typecheck passes

#### US-012: Verify Override System
**Description:** As a developer, I need to confirm the override system handles special field types.

**Acceptance Criteria:**
- [ ] `cmd/tfgen/overrides.json` loaded correctly
- [ ] Override types work:
  - `map_string` type (Snowflake `auto_qa_dedupe_table_mapping`)
  - `map_nested` type (ClickHouse `topics_config_map`, SQLServer `snapshot_custom_table_config`)
- [ ] Overridden fields generate correct schema attributes
- [ ] Typecheck passes

#### US-013: Verify Deprecation System
**Description:** As a developer, I need to confirm deprecated fields are generated correctly.

**Acceptance Criteria:**
- [ ] `cmd/tfgen/deprecations.json` loaded correctly
- [ ] Model fields added for deprecated attributes
- [ ] Wrapper files add schema attributes with DeprecationMessage
- [ ] Field mappings extended with deprecated aliases
- [ ] Typecheck passes

#### US-014: Run Regeneration Test
**Description:** As a developer, I need to confirm regenerating produces identical output.

**Acceptance Criteria:**
- [ ] Run: `STREAMKAP_BACKEND_PATH=/path/to/backend go generate ./...`
- [ ] Run: `git diff internal/generated/`
- [ ] No changes (or only whitespace/formatting differences)
- [ ] All generated files compile without errors
- [ ] Typecheck passes

---

### Phase 5: Backend Alignment Verification

#### US-015: Create Backend Alignment Tooling
**Description:** As a developer, I need automated tooling to verify Terraform schemas match backend configs.

**Acceptance Criteria:**
- [ ] Create script/tool that parses backend `configuration.latest.json`
- [ ] Extract all `user_defined: true` fields from backend
- [ ] Compare with generated `FieldMappings` in Terraform
- [ ] Report missing fields (in backend but not Terraform)
- [ ] Report extra fields (in Terraform but not backend)
- [ ] Output structured report (JSON or markdown)
- [ ] Typecheck passes

#### US-016: Verify Source Connector Field Mappings
**Description:** As a developer, I need to confirm all 20 source connectors match their backend configs.

**Acceptance Criteria:**
- [ ] Verify field alignment for all sources:
  - AlloyDB, DB2, DocumentDB, DynamoDB, Elasticsearch
  - KafkaDirect, MariaDB, MongoDB, MongoDBHosted, MySQL
  - Oracle, OracleAWS, PlanetScale, PostgreSQL, Redis
  - S3, SQLServer, Supabase, Vitess, Webhook
- [ ] 100% field coverage for each connector
- [ ] Document any discrepancies found
- [ ] Typecheck passes

#### US-017: Verify Destination Connector Field Mappings
**Description:** As a developer, I need to confirm all 22 destination connectors match their backend configs.

**Acceptance Criteria:**
- [ ] Verify field alignment for all destinations:
  - AzBlob, BigQuery, ClickHouse, CockroachDB, Databricks
  - DB2, GCS, HTTPSink, Iceberg, Kafka, KafkaDirect
  - Motherduck, MySQL, Oracle, PostgreSQL, R2, Redis
  - Redshift, S3, Snowflake, SQLServer, Starburst
- [ ] 100% field coverage for each connector
- [ ] Document any discrepancies found
- [ ] Typecheck passes

#### US-018: Verify Transform Field Mappings
**Description:** As a developer, I need to confirm all 6 transform resources match their backend configs.

**Acceptance Criteria:**
- [ ] Verify field alignment for all transforms:
  - Enrich, EnrichAsync, FanOut, MapFilter, Rollup, SqlJoin
- [ ] 100% field coverage for each transform
- [ ] Document any discrepancies found
- [ ] Typecheck passes

#### US-019: Verify Dynamic Field Exclusion
**Description:** As a developer, I need to confirm dynamically-resolved fields are NOT in Terraform schemas.

**Acceptance Criteria:**
- [ ] `database.hostname` (computed) NOT in schema
- [ ] `database.port` (computed) NOT in schema
- [ ] `connection.url` (computed) NOT in schema
- [ ] Only `*.user.defined` variants exposed to users
- [ ] Typecheck passes

---

### Phase 6: Entity Coverage - Source Connectors

#### US-020: Write Acceptance Tests for Missing Source Connectors
**Description:** As a developer, I need acceptance tests for all 14 untested source connectors.

**Acceptance Criteria:**
- [ ] Write `TestAccSourceAlloyDB` - basic CRUD test
- [ ] Write `TestAccSourceDB2` - basic CRUD test
- [ ] Write `TestAccSourceDocumentDB` - basic CRUD test
- [ ] Write `TestAccSourceElasticsearch` - basic CRUD test
- [ ] Write `TestAccSourceMariaDB` - basic CRUD test
- [ ] Write `TestAccSourceMongoDBHosted` - basic CRUD test
- [ ] Write `TestAccSourceOracle` - basic CRUD test
- [ ] Write `TestAccSourceOracleAWS` - basic CRUD test
- [ ] Write `TestAccSourcePlanetScale` - basic CRUD test
- [ ] Write `TestAccSourceRedis` - basic CRUD test
- [ ] Write `TestAccSourceS3` - basic CRUD test
- [ ] Write `TestAccSourceSupabase` - basic CRUD test
- [ ] Write `TestAccSourceVitess` - basic CRUD test
- [ ] Write `TestAccSourceWebhook` - basic CRUD test
- [ ] All tests pass: `TF_ACC=1 go test -v -run 'TestAccSource' ./internal/provider/...`
- [ ] Typecheck passes

#### US-021: Create Schema Snapshots for Source Connectors
**Description:** As a developer, I need schema snapshots for all source connectors for backward compatibility testing.

**Acceptance Criteria:**
- [ ] Create snapshots for 14 missing sources
- [ ] Run: `UPDATE_SNAPSHOTS=1 go test -v -run 'TestSchemaBackwardsCompatibility' ./internal/provider/...`
- [ ] All 20 source snapshots exist in `testdata/snapshots/`
- [ ] Typecheck passes

---

### Phase 7: Entity Coverage - Destination Connectors

#### US-022: Write Acceptance Tests for Missing Destination Connectors
**Description:** As a developer, I need acceptance tests for all 15 untested destination connectors.

**Acceptance Criteria:**
- [ ] Write `TestAccDestinationAzBlob` - basic CRUD test
- [ ] Write `TestAccDestinationBigQuery` - basic CRUD test
- [ ] Write `TestAccDestinationCockroachDB` - basic CRUD test
- [ ] Write `TestAccDestinationDB2` - basic CRUD test
- [ ] Write `TestAccDestinationGCS` - basic CRUD test
- [ ] Write `TestAccDestinationHTTPSink` - basic CRUD test
- [ ] Write `TestAccDestinationKafkaDirect` - basic CRUD test
- [ ] Write `TestAccDestinationMotherduck` - basic CRUD test
- [ ] Write `TestAccDestinationMySQL` - basic CRUD test
- [ ] Write `TestAccDestinationOracle` - basic CRUD test
- [ ] Write `TestAccDestinationR2` - basic CRUD test
- [ ] Write `TestAccDestinationRedis` - basic CRUD test
- [ ] Write `TestAccDestinationRedshift` - basic CRUD test
- [ ] Write `TestAccDestinationSQLServer` - basic CRUD test
- [ ] Write `TestAccDestinationStarburst` - basic CRUD test
- [ ] All tests pass: `TF_ACC=1 go test -v -run 'TestAccDestination' ./internal/provider/...`
- [ ] Typecheck passes

#### US-023: Create Schema Snapshots for Destination Connectors
**Description:** As a developer, I need schema snapshots for all destination connectors.

**Acceptance Criteria:**
- [ ] Create snapshots for 15 missing destinations
- [ ] Run: `UPDATE_SNAPSHOTS=1 go test -v -run 'TestSchemaBackwardsCompatibility' ./internal/provider/...`
- [ ] All 22 destination snapshots exist in `testdata/snapshots/`
- [ ] Typecheck passes

---

### Phase 8: Entity Coverage - Transform Resources

#### US-024: Write Acceptance Tests for Missing Transform Resources
**Description:** As a developer, I need acceptance tests for the 3 untested transform resources.

**Acceptance Criteria:**
- [ ] Write `TestAccTransformEnrichAsync` - basic CRUD test
- [ ] Write `TestAccTransformFanOut` - basic CRUD test
- [ ] Write `TestAccTransformRollup` - basic CRUD test
- [ ] All tests pass: `TF_ACC=1 go test -v -run 'TestAccTransform' ./internal/provider/...`
- [ ] Typecheck passes

#### US-025: Create Schema Snapshots for Transform Resources
**Description:** As a developer, I need schema snapshots for all transform resources.

**Acceptance Criteria:**
- [ ] Create snapshots for 3 missing transforms
- [ ] All 6 transform snapshots exist in `testdata/snapshots/`
- [ ] Typecheck passes

---

### Phase 9: AI-Agent Optimization Audit

#### US-026: Audit Description Quality
**Description:** As a developer, I need to verify all schema descriptions meet AI-agent standards.

**Acceptance Criteria:**
- [ ] All attributes have both `Description` and `MarkdownDescription`
- [ ] Enum fields document: "Valid values: `x`, `y`, `z`."
- [ ] Fields with defaults document: "Defaults to `value`."
- [ ] Sensitive fields include security note
- [ ] Resource-level descriptions follow pattern:
  ```
  Description: "Manages a {DisplayName} {type} connector."
  MarkdownDescription: "Manages a **{DisplayName}**..."
  ```
- [ ] Typecheck passes

#### US-027: Verify Example Files
**Description:** As a developer, I need to verify all resources have proper example files.

**Acceptance Criteria:**
- [ ] All 53 resources have `examples/resources/streamkap_{name}/` directory
- [ ] Each has `basic.tf` with minimal required config
- [ ] Each has `complete.tf` with all options and comments
- [ ] Examples are syntactically valid Terraform
- [ ] Run: `terraform fmt -check examples/`

#### US-028: Test Terraform MCP Server Compatibility
**Description:** As a developer, I need to verify the provider works with Terraform MCP Server.

**Acceptance Criteria:**
- [ ] Provider overview accessible via MCP
- [ ] Resource schemas queryable
- [ ] Field descriptions helpful for AI agents
- [ ] Valid values visible in schema metadata

---

### Phase 10: Documentation Audit

#### US-029: Verify Core Documentation
**Description:** As a developer, I need to ensure all core documentation is accurate and complete.

**Acceptance Criteria:**
- [ ] `CLAUDE.md` reflects current architecture and commands
- [ ] `docs/ARCHITECTURE.md` matches three-layer implementation
- [ ] `docs/DEVELOPMENT.md` has accurate setup instructions
- [ ] `docs/MIGRATION.md` lists all deprecated fields with migration paths
- [ ] `docs/AI_AGENT_COMPATIBILITY.md` describes MCP integration
- [ ] `CHANGELOG.md` is up to date
- [ ] `README.md` has accurate overview

#### US-030: Verify Audit Documents
**Description:** As a developer, I need to ensure audit reference documents are accurate.

**Acceptance Criteria:**
- [ ] `docs/audits/entity-config-schema-audit.md` matches current JSON schema patterns
- [ ] `docs/audits/backend-code-reference.md` matches current API patterns
- [ ] No TODOs or placeholders in documentation

#### US-031: Create Comprehensive Audit Report
**Description:** As a developer, I need to compile all audit findings into a comprehensive report.

**Acceptance Criteria:**
- [ ] Create `docs/audits/comprehensive-audit-report.md`
- [ ] Include: Executive Summary, Architecture Comparison, Implementation Assessment
- [ ] Include: Compatibility Assessment, Entity Coverage Matrix
- [ ] Include: Code Generator Assessment, Backend Alignment
- [ ] Include: AI-Agent Assessment, Test Coverage
- [ ] Include: Documentation Assessment, Recommendations
- [ ] Include: Appendices with detailed test output

---

### Phase 10B: Documentation Creation & Updates

#### US-045: Create Missing Resource Documentation
**Description:** As a developer, I need documentation created for all resources that are missing proper docs.

**Acceptance Criteria:**
- [ ] Audit `docs/resources/` for missing documentation files
- [ ] Create `docs/resources/streamkap_source_*.md` for each source connector
- [ ] Create `docs/resources/streamkap_destination_*.md` for each destination connector
- [ ] Create `docs/resources/streamkap_transform_*.md` for each transform type
- [ ] Each doc includes: Overview, Example Usage, Argument Reference, Attribute Reference, Import
- [ ] Documentation generated via `go generate` produces valid markdown

#### US-046: Update Inline Code Comments
**Description:** As a developer, I need inline code comments added/updated for complex logic.

**Acceptance Criteria:**
- [ ] Add/update comments in `internal/resource/connector/base.go` explaining reflection logic
- [ ] Add/update comments in `cmd/tfgen/parser.go` explaining JSON parsing rules
- [ ] Add/update comments in `cmd/tfgen/generator.go` explaining type mapping decisions
- [ ] Add/update comments in `internal/api/client.go` explaining OAuth2 flow
- [ ] Add/update comments in `internal/api/retry.go` explaining retry strategy
- [ ] All exported functions have GoDoc comments
- [ ] Typecheck passes

#### US-047: Create Test Documentation
**Description:** As a developer, I need documentation for the test suite explaining how to run and extend tests.

**Acceptance Criteria:**
- [ ] Create/update `docs/TESTING.md` with comprehensive test guide
- [ ] Document all test tiers (unit, schema, validator, integration, acceptance, migration)
- [ ] Document VCR cassette recording and playback
- [ ] Document test sweeper usage
- [ ] Document environment variable requirements for each test type
- [ ] Document how to add tests for new resources
- [ ] Include troubleshooting section for common test failures

#### US-048: Create Code Generator Documentation
**Description:** As a developer, I need documentation explaining how to use and extend the code generator.

**Acceptance Criteria:**
- [ ] Create/update `docs/CODE_GENERATOR.md`
- [ ] Document `cmd/tfgen` CLI usage and flags
- [ ] Document `overrides.json` schema and usage
- [ ] Document `deprecations.json` schema and usage
- [ ] Document how to add a new connector via code generation
- [ ] Document type mapping rules (control → Terraform type)
- [ ] Include examples for common customization scenarios

#### US-049: Update CHANGELOG for Audit Findings
**Description:** As a developer, I need the CHANGELOG updated to reflect all fixes and improvements from this audit.

**Acceptance Criteria:**
- [ ] Add "Unreleased" section to `CHANGELOG.md`
- [ ] Document all bug fixes found and resolved
- [ ] Document all new tests added
- [ ] Document all documentation improvements
- [ ] Document any breaking changes or deprecations
- [ ] Follow Keep a Changelog format

#### US-050: Create/Update Example Files for All Resources
**Description:** As a developer, I need example Terraform configurations for all resources.

**Acceptance Criteria:**
- [ ] Audit `examples/resources/` for missing directories
- [ ] Create `examples/resources/streamkap_{resource}/` for each missing resource
- [ ] Each directory contains `basic.tf` (minimal config)
- [ ] Each directory contains `complete.tf` (all options with comments)
- [ ] Each directory contains `import.sh` (import command example)
- [ ] All examples pass `terraform fmt -check`
- [ ] All examples pass `terraform validate` (syntax check)

#### US-051: Update README with Quick Start Guide
**Description:** As a developer, I need the README updated with a clear quick start guide.

**Acceptance Criteria:**
- [ ] Add "Quick Start" section to `README.md`
- [ ] Include minimal provider configuration example
- [ ] Include example source and destination creation
- [ ] Include example pipeline creation
- [ ] Link to full documentation for each resource type
- [ ] Add badges for build status, version, license

---

### Phase 11: Issue Tracking and Fixes

#### US-032: Create GitHub Issues for Findings
**Description:** As a developer, I need to document all audit findings as trackable issues.

**Acceptance Criteria:**
- [ ] Create issues for all critical bugs found
- [ ] Create issues for missing test coverage
- [ ] Create issues for documentation gaps
- [ ] Create issues for enhancement opportunities
- [ ] Label issues appropriately (bug, enhancement, documentation)
- [ ] Link issues to audit report

#### US-033: Fix Critical Issues During Audit
**Description:** As a developer, I need to fix critical issues immediately when found.

**Acceptance Criteria:**
- [ ] Critical issues (blocking functionality) fixed immediately
- [ ] Fixes verified with tests before continuing audit
- [ ] Fixes documented in audit report
- [ ] Typecheck and all existing tests pass after fixes

---

### Phase 12: VCR Cassette Recording

#### US-034: Record VCR Cassettes for Source Connector Tests
**Description:** As a developer, I need VCR cassettes recorded for all source connector acceptance tests so CI can run without live API access.

**Acceptance Criteria:**
- [ ] Record cassettes for all 20 source connector tests
- [ ] Cassettes stored in `testdata/cassettes/sources/`
- [ ] Sensitive data (tokens, passwords) redacted from cassettes
- [ ] Run: `UPDATE_CASSETTES=1 go test -v -run 'TestAccSource' ./internal/provider/...`
- [ ] Verify cassette playback works: `go test -v -run 'TestIntegration_Source' ./internal/provider/...`
- [ ] Typecheck passes

#### US-035: Record VCR Cassettes for Destination Connector Tests
**Description:** As a developer, I need VCR cassettes recorded for all destination connector acceptance tests.

**Acceptance Criteria:**
- [ ] Record cassettes for all 22 destination connector tests
- [ ] Cassettes stored in `testdata/cassettes/destinations/`
- [ ] Sensitive data redacted from cassettes
- [ ] Run: `UPDATE_CASSETTES=1 go test -v -run 'TestAccDestination' ./internal/provider/...`
- [ ] Verify cassette playback works
- [ ] Typecheck passes

#### US-036: Record VCR Cassettes for Transform and Other Resource Tests
**Description:** As a developer, I need VCR cassettes recorded for transform, pipeline, topic, and tag tests.

**Acceptance Criteria:**
- [ ] Record cassettes for all 6 transform tests
- [ ] Record cassettes for Pipeline and Topic tests
- [ ] Record cassettes for Tag tests
- [ ] Cassettes stored in `testdata/cassettes/transforms/` and `testdata/cassettes/other/`
- [ ] Sensitive data redacted from cassettes
- [ ] Typecheck passes

---

### Phase 13: Test Sweeper Implementation

#### US-037: Implement Source Connector Test Sweepers
**Description:** As a developer, I need test sweepers to clean up orphaned source resources from failed test runs.

**Acceptance Criteria:**
- [ ] Implement `sweepSources` function in `internal/provider/sweep_test.go`
- [ ] Sweeper lists all sources via API
- [ ] Sweeper deletes resources matching test naming convention (prefix `tf-acc-test-`)
- [ ] Sweeper handles pagination for large resource lists
- [ ] Run: `go test -v -run 'TestSweepSources' ./internal/provider/...`
- [ ] Typecheck passes

#### US-038: Implement Destination Connector Test Sweepers
**Description:** As a developer, I need test sweepers to clean up orphaned destination resources.

**Acceptance Criteria:**
- [ ] Implement `sweepDestinations` function
- [ ] Sweeper lists all destinations via API
- [ ] Sweeper deletes resources matching test naming convention
- [ ] Sweeper handles pagination
- [ ] Run: `go test -v -run 'TestSweepDestinations' ./internal/provider/...`
- [ ] Typecheck passes

#### US-039: Implement Transform and Pipeline Test Sweepers
**Description:** As a developer, I need test sweepers for transforms and pipelines.

**Acceptance Criteria:**
- [ ] Implement `sweepTransforms` function
- [ ] Implement `sweepPipelines` function
- [ ] Sweepers delete resources matching test naming convention
- [ ] Run: `go test -v -run 'TestSweep' ./internal/provider/...`
- [ ] Typecheck passes

---

### Phase 14: Environment Variable Assessment

#### US-040: Audit and Document Required Environment Variables
**Description:** As a developer, I need to assess which connectors have credentials available and document all required environment variables.

**Acceptance Criteria:**
- [ ] Audit existing `.env` and `.env.example` files
- [ ] Create comprehensive list of all connector-specific env vars needed
- [ ] Add missing env var placeholders to `.env.example`
- [ ] Document which connectors have full credentials vs smoke-test only
- [ ] Update `docs/DEVELOPMENT.md` with env var requirements
- [ ] Typecheck passes

#### US-041: Implement Smoke Tests for Connectors Without Credentials
**Description:** As a developer, I need smoke tests for connectors where we lack full infrastructure credentials.

**Acceptance Criteria:**
- [ ] Identify connectors without credentials (Oracle, BigQuery, etc.)
- [ ] Create `TestSmoke*` tests that verify:
  - Schema compiles correctly
  - Model↔API conversion works
  - Validation rules trigger correctly
- [ ] Compare schema behavior with `main` branch for parity
- [ ] Document smoke-tested connectors in audit report
- [ ] Typecheck passes

---

### Phase 15: Negative/Error Case Tests

#### US-042: Write Negative Tests for API Error Handling
**Description:** As a developer, I need tests that verify proper error handling when the API returns errors.

**Acceptance Criteria:**
- [ ] Test 401 Unauthorized error handling
- [ ] Test 403 Forbidden error handling
- [ ] Test 404 Not Found error handling (resource deleted externally)
- [ ] Test 422 Validation error handling
- [ ] Test 500/502/503/504 server error handling and retry logic
- [ ] Test network timeout handling
- [ ] All error messages propagate to Terraform correctly
- [ ] Typecheck passes

#### US-043: Write Negative Tests for Schema Validation
**Description:** As a developer, I need tests that verify schema validators reject invalid input.

**Acceptance Criteria:**
- [ ] Test required field validation (missing required fields)
- [ ] Test enum validation (invalid values rejected)
- [ ] Test range validation (slider min/max bounds)
- [ ] Test format validation (ports, hostnames, etc.)
- [ ] Test sensitive field handling (not exposed in logs/state)
- [ ] Validators produce helpful error messages
- [ ] Typecheck passes

#### US-044: Write Negative Tests for Resource State Conflicts
**Description:** As a developer, I need tests that verify proper handling of state conflicts and drift.

**Acceptance Criteria:**
- [ ] Test resource modified externally (drift detection)
- [ ] Test resource deleted externally (proper error on refresh)
- [ ] Test concurrent modification handling
- [ ] Test import of non-existent resource
- [ ] Test import with wrong ID format
- [ ] Typecheck passes

---

## Functional Requirements

### Architecture
- FR-1: The refactored architecture must maintain three clear layers: generated, wrapper, and base
- FR-2: Generated code must be reproducible by running `go generate ./...`
- FR-3: All 53 resources must be registered in `provider.go`

### CRUD Operations
- FR-4: All resources must support Create, Read, Update, Delete operations
- FR-5: All resources must support Import via ID
- FR-6: All resources must support configurable timeouts

### Backward Compatibility
- FR-7: All deprecated fields must continue to work with deprecation warnings
- FR-8: Migration from v2.1.18 must produce empty plan (no changes)
- FR-9: Schema compatibility tests must pass for all resources

### Code Generation
- FR-10: Parser must extract all `user_defined: true` fields from backend configs
- FR-11: Generator must produce compilable Go code for all field types
- FR-12: Override system must handle special field types (map_string, map_nested)
- FR-13: Deprecation system must add proper schema attributes and field mappings

### Backend Alignment
- FR-14: All Terraform schema fields must map to backend config fields
- FR-15: Dynamic/computed fields must NOT appear in Terraform schemas
- FR-16: Field types must match backend control types

### AI-Agent Optimization
- FR-17: All schema attributes must have Description and MarkdownDescription
- FR-18: All resources must have basic.tf and complete.tf examples
- FR-19: Enum fields must document valid values

### Testing
- FR-20: All 53 resources must have acceptance tests
- FR-21: All resources must have schema snapshots for compatibility testing
- FR-22: Unit tests must cover all helper functions and API client methods
- FR-23: VCR cassettes must be recorded for all acceptance tests
- FR-24: Test sweepers must clean up resources matching `tf-acc-test-` prefix
- FR-25: Negative tests must verify error handling for API errors (401, 403, 404, 422, 5xx)
- FR-26: Negative tests must verify schema validation rejects invalid input
- FR-27: Smoke tests must exist for connectors without full credentials

### Environment Variables
- FR-28: All required connector env vars must be documented in `.env.example`
- FR-29: `docs/DEVELOPMENT.md` must list all env vars with descriptions
- FR-30: Connectors without credentials must have smoke tests as alternative

### Documentation
- FR-31: All resources must have documentation in `docs/resources/`
- FR-32: All exported functions must have GoDoc comments
- FR-33: Complex logic must have inline comments explaining rationale
- FR-34: `docs/TESTING.md` must document all test tiers and procedures
- FR-35: `docs/CODE_GENERATOR.md` must document tfgen usage and extension
- FR-36: All resources must have `basic.tf` and `complete.tf` examples
- FR-37: CHANGELOG must document all audit fixes and improvements
- FR-38: README must include quick start guide with working examples

---

## Non-Goals (Out of Scope)

- Backend Python codebase modifications
- Terraform Registry publishing
- CI/CD pipeline implementation (documentation only)
- New connector development
- Performance optimization beyond fixing critical issues
- UI/UX changes (this is a CLI provider)

---

## Technical Considerations

### Environment Requirements
- Go 1.24+
- `STREAMKAP_BACKEND_PATH` set to local backend repository
- `STREAMKAP_CLIENT_ID` and `STREAMKAP_SECRET` for acceptance tests
- `TF_ACC=1` for acceptance test execution

### Parallel Execution Strategy
Phases that can run in parallel:
- Phase 1 (Architecture) + Phase 4 (Code Generator) - independent analysis
- Phase 6 (Sources) + Phase 7 (Destinations) + Phase 8 (Transforms) - independent test writing
- Phase 9 (AI-Agent) + Phase 10 (Documentation) + Phase 10B (Doc Creation) - independent audits
- Phase 12 (VCR Sources) + Phase 12 (VCR Destinations) + Phase 12 (VCR Other) - after acceptance tests
- Phase 14 (Env Vars) can start early, parallel with Phase 1
- Phase 15 (Negative Tests) can run parallel with Phase 6-8

Sequential dependencies:
- Phase 2 (Implementation) depends on Phase 1
- Phase 3 (Compatibility) depends on Phase 2
- Phase 5 (Backend Alignment) depends on Phase 4
- Phase 12 (VCR) depends on Phase 6-8 (acceptance tests must exist first)
- Phase 13 (Sweepers) can run anytime after Phase 2
- Phase 11 (Issues) depends on all other phases

### Critical Issue Thresholds
**Stop and fix immediately:**
- Compilation failures
- CRUD operation failures
- Data loss scenarios
- Security vulnerabilities
- Breaking changes vs v2.1.18

**Document and continue:**
- Missing test coverage
- Documentation gaps
- Minor description improvements
- Enhancement opportunities

---

## Success Metrics

- All 53 resources pass CRUD acceptance tests
- Schema compatibility tests detect no breaking changes vs v2.1.18
- All deprecated fields work with deprecation warnings
- Code generator reproduces identical output when re-run
- API field mappings match 100% of backend config fields
- All schema descriptions include valid values, defaults, and sensitivity notes
- Test coverage >80% for critical paths
- Zero outstanding TODOs in core documentation
- Comprehensive audit report completed
- VCR cassettes recorded for all 53 resources (CI-friendly testing)
- Test sweepers implemented and functional for all resource types
- All required env vars documented in `.env.example`
- Negative/error case tests pass for all error scenarios
- Smoke tests pass for connectors without full credentials
- All 53 resources have documentation in `docs/resources/`
- All 53 resources have example files (`basic.tf`, `complete.tf`)
- `docs/TESTING.md` and `docs/CODE_GENERATOR.md` complete
- All exported functions have GoDoc comments
- CHANGELOG updated with all audit findings

---

## Open Questions (Resolved)

All open questions have been resolved and incorporated into the PRD:

1. ✅ **VCR Cassettes**: Record for all acceptance tests during this audit → **Phase 12 added (US-034 to US-036)**
2. ✅ **Test Sweepers**: Include cleanup logic for orphaned resources → **Phase 13 added (US-037 to US-039)**
3. ✅ **Infrastructure Requirements**: Assess via `.env` file; add missing env vars; use smoke testing for connectors without credentials → **Phase 14 added (US-040 to US-041)**
4. ✅ **Negative Tests**: Write error case tests as part of this audit → **Phase 15 added (US-042 to US-044)**

---

## Appendix A: Resource Inventory

### Source Connectors (20)
AlloyDB, DB2, DocumentDB, DynamoDB, Elasticsearch, KafkaDirect, MariaDB, MongoDB, MongoDBHosted, MySQL, Oracle, OracleAWS, PlanetScale, PostgreSQL, Redis, S3, SQLServer, Supabase, Vitess, Webhook

### Destination Connectors (22)
AzBlob, BigQuery, ClickHouse, CockroachDB, Databricks, DB2, GCS, HTTPSink, Iceberg, Kafka, KafkaDirect, Motherduck, MySQL, Oracle, PostgreSQL, R2, Redis, Redshift, S3, Snowflake, SQLServer, Starburst

### Transform Resources (6)
Enrich, EnrichAsync, FanOut, MapFilter, Rollup, SqlJoin

### Other Resources (3)
Pipeline, Topic, Tag

### Data Sources (2)
Transform, Tag

**Total: 53 Terraform resources**

---

## Appendix B: Test Commands Reference

```bash
# Environment setup
export STREAMKAP_BACKEND_PATH=/path/to/python-be-streamkap
export TF_ACC=1
export STREAMKAP_CLIENT_ID=xxx
export STREAMKAP_SECRET=xxx

# Build and install
go install .

# Unit tests
go test -v -short ./...

# Schema compatibility
go test -v -run 'TestSchemaBackwardsCompatibility' ./internal/provider/...

# Update snapshots
UPDATE_SNAPSHOTS=1 go test -v -run 'TestSchemaBackwardsCompatibility' ./internal/provider/...

# Acceptance tests
TF_ACC=1 go test -v -timeout 120m -run 'TestAcc' ./internal/provider/...

# Migration tests
TF_ACC=1 go test -v -timeout 180m -run 'TestAcc.*Migration' ./internal/provider/...

# Regenerate schemas
go generate ./...

# Check for drift
git diff internal/generated/
```

---

*PRD Version: 1.1*
*Created: 2026-01-21*
*Updated: 2026-01-21*
*Based on: Comprehensive Audit Planning Document*

**Version History:**
- v1.0: Initial PRD with 33 user stories across 11 phases
- v1.1: Added VCR cassettes (Phase 12), test sweepers (Phase 13), env var assessment (Phase 14), negative tests (Phase 15), and documentation creation/updates (Phase 10B) - now 51 user stories across 15 phases
