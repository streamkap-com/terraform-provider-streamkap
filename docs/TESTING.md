# Testing Guide

This document provides comprehensive testing documentation for the Streamkap Terraform Provider.

## Table of Contents

- [Test Tiers Overview](#test-tiers-overview)
- [Unit Tests](#unit-tests)
- [Schema Tests](#schema-tests)
- [Acceptance Tests](#acceptance-tests)
- [Migration Tests](#migration-tests)
- [VCR Cassettes](#vcr-cassettes)
- [Smoke Tests](#smoke-tests)
- [Test Sweepers](#test-sweepers)
- [Environment Variables](#environment-variables)
- [Test Naming Conventions](#test-naming-conventions)
- [Writing New Tests](#writing-new-tests)

---

## Test Tiers Overview

| Tier | Test Pattern | API Required | Duration | When to Run |
|------|--------------|--------------|----------|-------------|
| Unit | `Test[^Acc]` | No | ~5s | Every commit |
| Schema Compat | `TestSchemaBackwardsCompatibility` | No | ~2s | Every PR |
| Validators | `Test.*Validator` | No | ~2s | Every commit |
| Smoke | `TestSmoke*` | No | ~5s | Every PR |
| Integration | `TestIntegration_` | No (VCR) | ~30s | Every PR |
| Acceptance | `TestAcc*` | Yes | ~15m | Nightly/Release |
| Migration | `TestAcc.*Migration` | Yes | ~30m | Pre-release |

---

## Unit Tests

Unit tests verify internal logic without external dependencies.

### Running Unit Tests

```bash
# Run all unit tests (fast, no API needed)
go test -v -short ./...

# Run API client unit tests
go test -v -short ./internal/api/...

# Run helper function tests
go test -v ./internal/helper/...
```

### Test Locations

| Package | Tests | Coverage |
|---------|-------|----------|
| `internal/api/` | 21 tests | Source CRUD, Auth, Retry |
| `internal/helper/` | 10 tests | Type conversion, Deprecation |
| `internal/provider/` | Validators | Enum, Range validators |

### Writing Unit Tests

```go
// File: internal/api/source_test.go
func TestCreateSource_Success(t *testing.T) {
    // Setup mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Assert request
        assert.Equal(t, http.MethodPost, r.Method)

        // Verify created_from is set
        body, _ := io.ReadAll(r.Body)
        assert.Contains(t, string(body), `"created_from":"terraform"`)

        // Return response
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]any{"id": "123"})
    }))
    defer server.Close()

    // Test
    client := NewClient(server.URL, token)
    result, err := client.CreateSource(ctx, req)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, "123", result.ID)
}
```

---

## Schema Tests

Schema tests ensure backward compatibility and catch breaking changes.

### Running Schema Tests

```bash
# Run all schema compatibility tests
go test -v -run 'TestSchemaBackwardsCompatibility' ./internal/provider/...

# Update snapshots after intentional changes
UPDATE_SNAPSHOTS=1 go test -v -run 'TestSchemaBackwardsCompatibility' ./internal/provider/...
```

### Schema Snapshots

Snapshots are stored in `internal/provider/testdata/schemas/`:

```
testdata/schemas/
├── destination_clickhouse_v1.json
├── destination_databricks_v1.json
├── destination_snowflake_v1.json
├── source_postgresql_v1.json
├── source_mysql_v1.json
└── ... (16 total)
```

### Breaking Change Detection

Schema tests detect these breaking changes:

| Change Type | Severity | Example |
|-------------|----------|---------|
| Required attribute removed | **BREAKING** | Removing `database_hostname` |
| Optional → Required | **BREAKING** | Making `port` required |
| Type change | **BREAKING** | String → Int64 |
| Computed removed | Warning | Removing `connector_status` |

### Adding Schema Snapshots

When creating a new resource:

1. Add test case to `schema_compat_test.go`:

```go
{
    name:            "source_newconnector",
    snapshotFile:    "source_newconnector_v1.json",
    resourceFactory: func() resource.Resource { return source.NewNewConnectorResource() },
},
```

2. Generate initial snapshot:

```bash
UPDATE_SNAPSHOTS=1 go test -v -run 'TestSchemaBackwardsCompatibility/source_newconnector' ./internal/provider/...
```

---

## Acceptance Tests

Acceptance tests run full CRUD operations against the Streamkap API.

### Running Acceptance Tests

```bash
# Set required environment variables
export TF_ACC=1
export STREAMKAP_CLIENT_ID="your-client-id"
export STREAMKAP_SECRET="your-secret"

# Run all acceptance tests
go test -v -timeout 120m -run 'TestAcc' ./internal/provider/...

# Run specific resource tests
go test -v -run 'TestAccSourcePostgreSQL' ./internal/provider/...
go test -v -run 'TestAccDestinationSnowflake' ./internal/provider/...
```

### Test Structure

```go
func TestAccSourcePostgreSQLResource(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Step 1: Create and Read
            {
                Config: providerConfig + testConfigCreate,
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "name", "test-source"),
                    resource.TestCheckResourceAttrSet("streamkap_source_postgresql.test", "id"),
                ),
            },
            // Step 2: Import
            {
                ResourceName:      "streamkap_source_postgresql.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
            // Step 3: Update
            {
                Config: providerConfig + testConfigUpdate,
                Check: resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "name", "test-source-updated"),
            },
        },
    })
}
```

### Skipping Without Credentials

Tests should skip gracefully when credentials are missing:

```go
var sourcePostgreSQLHostname = os.Getenv("TF_VAR_source_postgresql_hostname")
var sourcePostgreSQLPassword = os.Getenv("TF_VAR_source_postgresql_password")

func TestAccSourcePostgreSQLResource(t *testing.T) {
    if sourcePostgreSQLHostname == "" || sourcePostgreSQLPassword == "" {
        t.Skip("TF_VAR_source_postgresql_hostname and TF_VAR_source_postgresql_password must be set")
    }
    // ... test body
}
```

---

## Migration Tests

Migration tests validate behavioral equivalence between provider versions.

### Purpose

Migration tests ensure:
1. Resources created with OLD provider (v2.1.18) work with NEW provider
2. Terraform plan shows NO changes after provider upgrade
3. Update operations work correctly after migration

### Running Migration Tests

```bash
# Set credentials
export TF_ACC=1
export STREAMKAP_CLIENT_ID="..."
export STREAMKAP_SECRET="..."
export TF_VAR_source_postgresql_hostname="..."
export TF_VAR_source_postgresql_password="..."

# Run migration tests (takes ~30 minutes)
go test -v -timeout 180m -run 'TestAcc.*Migration' ./internal/provider/...
```

### Test Pattern

```go
func TestAccSourcePostgreSQL_MigrationFromLegacy(t *testing.T) {
    resource.Test(t, resource.TestCase{
        Steps: []resource.TestStep{
            // Step 1: Create with OLD provider (v2.1.18)
            {
                ExternalProviders: legacyProviderConfig(),
                Config:            config,
                Check:             resource.TestCheckResourceAttr(...),
            },
            // Step 2: Switch to NEW provider - MUST produce empty plan
            {
                ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
                Config:                   config,
                ConfigPlanChecks: resource.ConfigPlanChecks{
                    PreApply: []plancheck.PlanCheck{
                        plancheck.ExpectEmptyPlan(),
                    },
                },
            },
            // Step 3: Verify update works with NEW provider
            {
                ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
                Config:                   configUpdated,
            },
        },
    })
}
```

### Interpreting Failures

If migration tests fail with non-empty plan:
- The new provider behaves differently than v2.1.18
- Check the plan output for which attributes differ
- This indicates a potential breaking change

---

## VCR Cassettes

VCR (Video Cassette Recorder) cassettes record HTTP interactions for replay during CI.

### Overview

- Located in `internal/provider/testdata/cassettes/`
- Records API requests/responses as JSON files
- Enables integration tests without live API access
- Automatically redacts sensitive data (Authorization headers, passwords)

### Recording Cassettes

```bash
# Record new cassettes (requires credentials)
UPDATE_CASSETTES=1 go test -v -run 'TestIntegration_' ./internal/provider/...

# Record specific cassette
UPDATE_CASSETTES=1 go test -v -run 'TestIntegration_SourcePostgreSQL' ./internal/provider/...
```

### Cassette Structure

```go
// vcr_test.go - VCR client factory
func newVCRClient(t *testing.T, cassetteName string) (*http.Client, func()) {
    cassettePath := filepath.Join("testdata", "cassettes", cassetteName)

    mode := recorder.ModeReplayOnly
    if os.Getenv("UPDATE_CASSETTES") != "" {
        mode = recorder.ModeRecordOnly
    }

    // Hook to redact sensitive data
    redactHook := func(i *cassette.Interaction) error {
        delete(i.Request.Headers, "Authorization")
        i.Request.Body = redactSensitiveFields(i.Request.Body)
        return nil
    }

    r, err := recorder.New(cassettePath, recorder.WithMode(mode), ...)
}
```

### When to Re-record

Re-record cassettes when:
1. API endpoints change
2. Request/response schemas change
3. New fields are added
4. Authentication flow changes

---

## Smoke Tests

Smoke tests verify schemas compile and basic conversions work without API access.

### Purpose

- Validate generated code works correctly
- Test connectors where we lack test credentials
- Catch schema compilation errors early

### Running Smoke Tests

```bash
# Run all smoke tests
go test -v -run 'TestSmoke' ./internal/provider/...
```

### Test Coverage

Smoke tests verify:
1. Schema compiles without errors
2. Model struct has expected fields
3. Field mappings are valid
4. Required computed fields exist (id, connector)
5. Name field is required

```go
func TestSmokeSourceOracle(t *testing.T) {
    tc := smokeTestCase{
        name:            "source_oracle",
        resourceFactory: func() resource.Resource { return source.NewOracleResource() },
        modelFactory:    func() any { return &generated.SourceOracleModel{} },
        fieldMappings:   generated.SourceOracleFieldMappings,
    }
    runSchemaSmoke(t, tc)
    runModelSmoke(t, tc)
}
```

---

## Test Sweepers

Sweepers clean up orphaned test resources from failed test runs.

### Running Sweepers

```bash
# Run sweepers (requires credentials)
go test -v -run 'TestSweep' ./internal/provider/...

# Or use the resource sweep command
TF_ACC=1 go test -v -tags=sweep -run TestSweep ./internal/provider/...
```

### Sweeper Configuration

Sweepers are registered in `sweep_test.go`:

```go
//go:build sweep

func init() {
    resource.AddTestSweepers("streamkap_source", &resource.Sweeper{
        Name: "streamkap_source",
        F:    sweepSources,
    })

    resource.AddTestSweepers("streamkap_pipeline", &resource.Sweeper{
        Name:         "streamkap_pipeline",
        F:            sweepPipelines,
        Dependencies: []string{"streamkap_source", "streamkap_destination"},
    })
}
```

### Test Resource Prefixes

Resources matching these prefixes are swept:

| Prefix | Purpose |
|--------|---------|
| `tf-acc-test-` | Acceptance tests |
| `tf-migration-test-` | Migration tests |
| `test-source-` | Integration tests |
| `test-destination-` | Integration tests |

---

## Environment Variables

### Required for Acceptance Tests

| Variable | Required | Description |
|----------|----------|-------------|
| `TF_ACC` | Yes | Set to `1` to enable acceptance tests |
| `STREAMKAP_CLIENT_ID` | Yes | OAuth2 client ID |
| `STREAMKAP_SECRET` | Yes | OAuth2 client secret |
| `STREAMKAP_HOST` | No | Override API URL (default: `https://api.streamkap.com`) |

### Connector-Specific Variables

| Variable | Resource | Description |
|----------|----------|-------------|
| `TF_VAR_source_postgresql_hostname` | PostgreSQL source | Database hostname |
| `TF_VAR_source_postgresql_password` | PostgreSQL source | Database password |
| `TF_VAR_source_mysql_hostname` | MySQL source | Database hostname |
| `TF_VAR_source_mysql_password` | MySQL source | Database password |
| `TF_VAR_destination_snowflake_url` | Snowflake destination | Account URL |
| `TF_VAR_destination_snowflake_password` | Snowflake destination | User password |

### Test Control Variables

| Variable | Description |
|----------|-------------|
| `UPDATE_CASSETTES` | Set to any value to re-record VCR cassettes |
| `UPDATE_SNAPSHOTS` | Set to any value to update schema snapshots |
| `TF_LOG` | Terraform log level: TRACE, DEBUG, INFO, WARN, ERROR |

---

## Test Naming Conventions

### Pattern Overview

| Pattern | Example | Purpose |
|---------|---------|---------|
| `TestAcc*` | `TestAccSourcePostgreSQLResource` | Full CRUD acceptance |
| `TestAcc*_Migration*` | `TestAccSourcePostgreSQL_MigrationFromLegacy` | Version migration |
| `TestSmoke*` | `TestSmokeSourceOracle` | Schema validation |
| `TestIntegration_*` | `TestIntegration_SourcePostgreSQL` | VCR-based integration |
| `Test*Validator` | `TestSSLModeValidator` | Validator unit tests |
| `TestSchemaBackwardsCompatibility` | - | Schema compat |
| `TestAPI*` | `TestAPIError401` | API error handling |

### Resource Naming in Tests

Test resources use these naming conventions:

```hcl
# Acceptance tests
resource "streamkap_source_postgresql" "test" {
    name = "tf-acc-test-postgresql"
}

# Migration tests
resource "streamkap_source_postgresql" "migration_test" {
    name = "tf-migration-test-postgresql"
}
```

---

## Writing New Tests

### Adding Acceptance Test for New Connector

1. Create test file: `internal/provider/source_newconnector_resource_test.go`

2. Define environment variables:

```go
var sourceNewConnectorHostname = os.Getenv("TF_VAR_source_newconnector_hostname")
var sourceNewConnectorPassword = os.Getenv("TF_VAR_source_newconnector_password")
```

3. Write skip logic:

```go
func TestAccSourceNewConnectorResource(t *testing.T) {
    if sourceNewConnectorHostname == "" || sourceNewConnectorPassword == "" {
        t.Skip("TF_VAR_source_newconnector_hostname and TF_VAR_source_newconnector_password must be set")
    }
    // ...
}
```

4. Define test configuration:

```go
config := providerConfig + `
variable "source_newconnector_hostname" {
    type = string
}
variable "source_newconnector_password" {
    type      = string
    sensitive = true
}
resource "streamkap_source_newconnector" "test" {
    name     = "tf-acc-test-newconnector"
    hostname = var.source_newconnector_hostname
    password = var.source_newconnector_password
}
`
```

5. Run tests:

```bash
TF_ACC=1 TF_VAR_source_newconnector_hostname="host" TF_VAR_source_newconnector_password="pass" \
    go test -v -run 'TestAccSourceNewConnector' ./internal/provider/...
```

### Adding Smoke Test

```go
func TestSmokeSourceNewConnector(t *testing.T) {
    tc := smokeTestCase{
        name:            "source_newconnector",
        resourceFactory: func() resource.Resource { return source.NewNewConnectorResource() },
        modelFactory:    func() any { return &generated.SourceNewConnectorModel{} },
        fieldMappings:   generated.SourceNewConnectorFieldMappings,
    }
    runSchemaSmoke(t, tc)
    runModelSmoke(t, tc)
}
```

### Adding Schema Snapshot

```go
// In schema_compat_test.go
{
    name:            "source_newconnector",
    snapshotFile:    "source_newconnector_v1.json",
    resourceFactory: func() resource.Resource { return source.NewNewConnectorResource() },
},
```

Then run:

```bash
UPDATE_SNAPSHOTS=1 go test -v -run 'TestSchemaBackwardsCompatibility/source_newconnector' ./internal/provider/...
```

---

## CI Integration

### Recommended CI Pipeline

```yaml
test:
  stages:
    - unit:
        command: go test -v -short ./...
        duration: ~5s

    - schema:
        command: go test -v -run 'TestSchemaBackwardsCompatibility' ./internal/provider/...
        duration: ~2s

    - validators:
        command: go test -v -run 'Test.*Validator' ./internal/provider/...
        duration: ~2s

    - smoke:
        command: go test -v -run 'TestSmoke' ./internal/provider/...
        duration: ~5s

acceptance:
  schedule: nightly
  env:
    TF_ACC: "1"
    STREAMKAP_CLIENT_ID: ${{ secrets.STREAMKAP_CLIENT_ID }}
    STREAMKAP_SECRET: ${{ secrets.STREAMKAP_SECRET }}
  command: go test -v -timeout 120m -run 'TestAcc' ./internal/provider/...
```

### Pre-release Checklist

1. All unit tests pass: `go test -v -short ./...`
2. Schema compatibility verified: `go test -v -run 'TestSchemaBackwardsCompatibility' ./...`
3. Smoke tests pass: `go test -v -run 'TestSmoke' ./...`
4. Acceptance tests pass (with credentials): `TF_ACC=1 go test -v -timeout 120m -run 'TestAcc' ./...`
5. Migration tests pass: `TF_ACC=1 go test -v -timeout 180m -run 'TestAcc.*Migration' ./...`
6. Run sweepers to clean up: `TF_ACC=1 go test -v -tags=sweep -run TestSweep ./...`
