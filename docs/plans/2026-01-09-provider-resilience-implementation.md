# Provider Resilience & Testing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add configurable timeouts, retry logic, and comprehensive tests to the Terraform provider.

**Architecture:** Leverage terraform-plugin-framework-timeouts for user-configurable timeouts, implement a retry helper in the API client layer, and create comprehensive unit/acceptance tests.

**Tech Stack:** Go 1.21, terraform-plugin-framework v1.8.0, terraform-plugin-framework-timeouts

---

## Pre-Implementation Audit Summary

### Backend Investigation Results

| Component | Configuration | Source |
|-----------|---------------|--------|
| KC Timeout | 15 seconds (env: `KAFKA_CONNECT_TIMEOUT`) | `kafka_connect_utils.py:12` |
| KC Retries | 1 per server (env: `KAFKA_CONNECT_RETRIES`) | `kafka_connect_utils.py:13` |
| Backend Retry | Yes - tries multiple KC servers on ReadTimeout | `kafka_connect_utils.py:58-79` |
| Operation Retry | Yes - stop/resume/delete have 15 attempts, 2s delay | `kafka_connect_utils.py:234-280` |

### Error Response Mapping

| Scenario | HTTP Status | Error Pattern | Retryable? |
|----------|-------------|---------------|------------|
| KC timeout (all servers exhausted) | 500 | `KafkaConnectTimeout` | Yes |
| KC API error | Varies | `ApiErrorCodeException` | Depends |
| Validation error | 400 | `{"detail": "..."}` | No |
| Not found | 404 | `{"detail": "..."}` | No |
| Gateway timeout | 502/503/504 | Various | Yes |

### Codebase Structure

```
internal/resource/
├── connector/base.go       # Generic base for sources/destinations (639 lines)
├── transform/base.go       # Generic base for transforms (similar pattern)
├── pipeline/pipeline.go    # Standalone pipeline resource
├── source/*_generated.go   # Generated source configs
├── destination/*_generated.go
└── transform/*_generated.go
```

### Key Design Decisions

1. **Timeouts added at schema level**: Each resource's schema gets a `timeouts` block
2. **Retry in API client**: `doRequest()` handles retry, not individual resources
3. **No topic validation**: Topics use names not IDs; backend validates at create time
4. **Conservative retry**: 10s min delay since backend already retries internally

---

## Task 1: Add terraform-plugin-framework-timeouts Dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Step 1: Add the dependency**

```bash
go get github.com/hashicorp/terraform-plugin-framework-timeouts@latest
```

**Step 2: Verify go.mod updated**

Run: `grep terraform-plugin-framework-timeouts go.mod`
Expected: Line showing the dependency

**Step 3: Run go mod tidy**

```bash
go mod tidy
```

**Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add terraform-plugin-framework-timeouts dependency"
```

---

## Task 2: Create Retry Helper in API Client

**Files:**
- Create: `internal/api/retry.go`
- Create: `internal/api/retry_test.go`

**Step 1: Write the failing test for IsRetryableError**

```go
// internal/api/retry_test.go
package api

import (
	"errors"
	"testing"
)

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		// KC timeout - backend exhausted retries
		{"KC timeout", errors.New("KafkaConnectTimeout"), true},
		{"request timeout", errors.New("Request timed out"), true},
		{"socket timeout", errors.New("SocketTimeoutException: connect timed out"), true},
		// Gateway errors - infrastructure issues
		{"503 error", errors.New("503 Service Unavailable"), true},
		{"502 error", errors.New("502 Bad Gateway"), true},
		{"504 error", errors.New("504 Gateway Timeout"), true},
		// Connection errors - network issues
		{"connection refused", errors.New("connection refused"), true},
		{"connection reset", errors.New("connection reset by peer"), true},
		{"i/o timeout", errors.New("i/o timeout"), true},
		// Kafka-specific retryable errors
		{"rebalance", errors.New("REBALANCE_IN_PROGRESS"), true},
		{"leader not available", errors.New("LEADER_NOT_AVAILABLE"), true},
		// Non-retryable errors
		{"auth error", errors.New("401 Unauthorized"), false},
		{"validation error", errors.New("Invalid configuration"), false},
		{"not found", errors.New("404 Not Found"), false},
		{"bad request", errors.New("400 Bad Request"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/api -v -run TestIsRetryableError`
Expected: FAIL - function not defined

**Step 3: Write IsRetryableError implementation**

```go
// internal/api/retry.go
package api

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// IsRetryableError checks if an error is transient and should be retried.
// Note: The Streamkap backend already retries Kafka Connect operations internally
// (tries multiple KC servers on ReadTimeout). This function identifies errors
// that indicate the backend exhausted its retries OR infrastructure issues.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())

	// Patterns indicating backend exhausted its KC retries
	retryablePatterns := []string{
		"kafkaconnecttimeout",       // Backend's custom timeout exception
		"request timed out",         // KC timeout message
		"sockettimeoutexception",    // Java socket timeout
	}

	// Gateway/infrastructure errors - Streamkap API issues
	gatewayPatterns := []string{
		"502", "503", "504",         // Gateway errors
		"bad gateway",
		"service unavailable",
		"gateway timeout",
	}

	// Network errors - connection to Streamkap API
	networkPatterns := []string{
		"connection refused",
		"connection reset",
		"no such host",
		"network unreachable",
		"i/o timeout",
	}

	// Kafka-specific transient errors
	kafkaPatterns := []string{
		"rebalance_in_progress",
		"leader_not_available",
		"not_leader_for_partition",
	}

	allPatterns := append(retryablePatterns, gatewayPatterns...)
	allPatterns = append(allPatterns, networkPatterns...)
	allPatterns = append(allPatterns, kafkaPatterns...)

	for _, pattern := range allPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries int
	MinDelay   time.Duration
	MaxDelay   time.Duration
}

// DefaultRetryConfig returns sensible defaults for Streamkap API operations.
// Uses conservative delays because the backend already retries KC operations.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		MinDelay:   10 * time.Second, // Conservative: backend may be retrying
		MaxDelay:   60 * time.Second, // Cap to avoid excessive waits
	}
}

// RetryWithBackoff retries an operation with exponential backoff.
// Only retries transient errors; validation/auth errors fail immediately.
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, operation func() error) error {
	var lastErr error
	delay := cfg.MinDelay

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		lastErr = operation()
		if lastErr == nil {
			return nil
		}

		if !IsRetryableError(lastErr) {
			return lastErr // Non-retryable, fail immediately
		}

		if attempt == cfg.MaxRetries {
			break // Last attempt failed
		}

		tflog.Debug(ctx, "Retryable error, will retry", map[string]interface{}{
			"attempt": attempt + 1,
			"delay":   delay.String(),
			"error":   lastErr.Error(),
		})

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		// Exponential backoff with cap
		delay = delay * 2
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return lastErr
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/api -v -run TestIsRetryableError`
Expected: PASS

**Step 5: Add test for RetryWithBackoff**

```go
func TestRetryWithBackoff_SucceedsOnFirstTry(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{MaxRetries: 3, MinDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond}

	attempts := 0
	err := RetryWithBackoff(ctx, cfg, func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetryWithBackoff_RetriesOnTransientError(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{MaxRetries: 3, MinDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond}

	attempts := 0
	err := RetryWithBackoff(ctx, cfg, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("503 Service Unavailable")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryWithBackoff_FailsImmediatelyOnNonRetryable(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{MaxRetries: 3, MinDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond}

	attempts := 0
	err := RetryWithBackoff(ctx, cfg, func() error {
		attempts++
		return errors.New("400 Bad Request")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retry), got %d", attempts)
	}
}
```

**Step 6: Run all retry tests**

Run: `go test ./internal/api -v -run "TestRetry"`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/api/retry.go internal/api/retry_test.go
git commit -m "feat: add retry helper with error classification

- IsRetryableError classifies transient vs permanent errors
- Conservative retry config (10s min delay) since backend already retries
- RetryWithBackoff implements exponential backoff with context support"
```

---

## Task 3: Integrate Retry into API Client

**Files:**
- Modify: `internal/api/client.go`

**Step 1: Read current client.go**

Verify current structure of doRequest method.

**Step 2: Add doRequestWithRetry method**

Add to `internal/api/client.go`:

```go
// doRequestWithRetry wraps doRequest with retry logic for transient errors.
// Use this for Create/Update/Delete operations only, not for Read.
func (s *streamkapAPI) doRequestWithRetry(ctx context.Context, req *http.Request, result interface{}) error {
	cfg := DefaultRetryConfig()

	return RetryWithBackoff(ctx, cfg, func() error {
		// Clone the request for retry (body may have been consumed)
		reqCopy := req.Clone(ctx)
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewBuffer(body))
			reqCopy.Body = io.NopCloser(bytes.NewBuffer(body))
		}
		return s.doRequest(ctx, reqCopy, result)
	})
}
```

**Step 3: Update Create/Update/Delete methods to use retry**

For Source operations in `internal/api/source.go`:

```go
// CreateSource - change from doRequest to doRequestWithRetry
func (s *streamkapAPI) CreateSource(ctx context.Context, reqPayload Source) (*Source, error) {
	// ... existing code to build request ...
	var resp Source
	err = s.doRequestWithRetry(ctx, req, &resp)  // Changed from doRequest
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
```

Apply same pattern to:
- `CreateSource`, `UpdateSource`, `DeleteSource` in `source.go`
- `CreateDestination`, `UpdateDestination`, `DeleteDestination` in `destination.go`
- `CreatePipeline`, `UpdatePipeline`, `DeletePipeline` in `pipeline.go`
- `CreateTransform`, `UpdateTransform`, `DeleteTransform` in `transform.go`
- `UpdateTopic`, `DeleteTopic` in `topic.go`

**IMPORTANT:** Do NOT add retry to Get/Read methods - they should fail fast.

**Step 4: Add required imports to client.go**

```go
import (
	"bytes"
	"io"
	// ... existing imports
)
```

**Step 5: Build and test**

Run: `go build ./...`
Expected: Build succeeds

**Step 6: Commit**

```bash
git add internal/api/
git commit -m "feat: add retry logic to Create/Update/Delete API operations

- doRequestWithRetry wraps requests with exponential backoff
- Only applied to mutating operations (not Read/Get)
- Conservative retry: 3 attempts, 10s min delay"
```

---

## Task 4: Add Timeouts to Base Connector Resource

**Files:**
- Modify: `internal/resource/connector/base.go`

**Step 1: Add imports**

```go
import (
	// ... existing imports
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
)
```

**Step 2: Define default timeouts**

Add near the top of the file:

```go
const (
	defaultCreateTimeout = 5 * time.Minute
	defaultUpdateTimeout = 5 * time.Minute
	defaultDeleteTimeout = 3 * time.Minute
)
```

**Step 3: Modify GetSchema to include timeouts**

The `ConnectorConfig.GetSchema()` returns the schema. We need to modify it after retrieval.

In `Schema()` method:

```go
func (r *BaseConnectorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	baseSchema := r.config.GetSchema()

	// Add timeouts block to the schema
	baseSchema.Blocks = map[string]schema.Block{
		"timeouts": timeouts.Block(ctx, timeouts.Opts{
			Create: true,
			Update: true,
			Delete: true,
		}),
	}

	resp.Schema = baseSchema
}
```

**Step 4: Update Create to use timeout**

```go
func (r *BaseConnectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Resource",
			"Expected configured API client. Please report this issue to the provider developers.",
		)
		return
	}

	// Get timeout from config
	var timeoutsValue timeouts.Value
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("timeouts"), &timeoutsValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := timeoutsValue.Create(ctx, defaultCreateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	// ... rest of existing create logic ...

	// The API client retry will respect the context timeout
}
```

**Step 5: Update Update and Delete similarly**

Apply same pattern:
- Get `timeoutsValue` from config
- Call `timeoutsValue.Update()` or `timeoutsValue.Delete()`
- Wrap context with timeout
- Existing API calls will respect context cancellation

**Step 6: Build and test**

Run: `go build ./...`
Expected: Build succeeds

**Step 7: Commit**

```bash
git add internal/resource/connector/base.go
git commit -m "feat: add configurable timeouts to base connector resource

- Add timeouts block (create: 5m, update: 5m, delete: 3m defaults)
- Context timeout integrates with API client retry logic"
```

---

## Task 5: Add Timeouts to Transform Base Resource

**Files:**
- Modify: `internal/resource/transform/base.go`

**Step 1: Apply same pattern as connector base**

Add imports, constants, and modify Schema/Create/Update/Delete methods exactly as in Task 4.

**Step 2: Build and test**

Run: `go build ./...`

**Step 3: Commit**

```bash
git add internal/resource/transform/base.go
git commit -m "feat: add configurable timeouts to base transform resource"
```

---

## Task 6: Add Timeouts to Pipeline Resource

**Files:**
- Modify: `internal/resource/pipeline/pipeline.go`

**Step 1: Add imports**

```go
import (
	// ... existing imports
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
)
```

**Step 2: Define default timeouts**

```go
const (
	defaultCreateTimeout = 5 * time.Minute
	defaultUpdateTimeout = 5 * time.Minute
	defaultDeleteTimeout = 3 * time.Minute
)
```

**Step 3: Add timeouts block to Schema**

In the `Schema()` method, add:

```go
Blocks: map[string]schema.Block{
	"timeouts": timeouts.Block(ctx, timeouts.Opts{
		Create: true,
		Update: true,
		Delete: true,
	}),
},
```

**Step 4: Update Create/Update/Delete with timeout handling**

Same pattern as connector base - get timeout value, create context with deadline.

**Step 5: Build and test**

Run: `go build ./...`

**Step 6: Commit**

```bash
git add internal/resource/pipeline/pipeline.go
git commit -m "feat: add configurable timeouts to pipeline resource"
```

---

## Task 7: Add Unit Tests for Helper Functions

**Files:**
- Create: `internal/helper/helper_test.go`

**Step 1: Write tests for type conversion helpers**

```go
// internal/helper/helper_test.go
package helper

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGetTfCfgString(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]any
		key      string
		expected string
		isNull   bool
	}{
		{"string value", map[string]any{"key": "value"}, "key", "value", false},
		{"missing key", map[string]any{}, "key", "", true},
		{"nil value", map[string]any{"key": nil}, "key", "", true},
		{"non-string", map[string]any{"key": 123}, "key", "", false}, // Returns empty string, not null
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTfCfgString(tt.cfg, tt.key)
			if tt.isNull && !result.IsNull() {
				t.Errorf("Expected null, got: %v", result)
			}
			if !tt.isNull && result.ValueString() != tt.expected {
				t.Errorf("GetTfCfgString() = %v, want %v", result.ValueString(), tt.expected)
			}
		})
	}
}

func TestGetTfCfgInt64(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]any
		key      string
		expected int64
		isNull   bool
	}{
		{"float64 value", map[string]any{"key": float64(42)}, "key", 42, false},
		{"string int", map[string]any{"key": "42"}, "key", 42, false},
		{"missing key", map[string]any{}, "key", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTfCfgInt64(tt.cfg, tt.key)
			if tt.isNull && !result.IsNull() {
				t.Errorf("Expected null, got: %v", result)
			}
			if !tt.isNull && result.ValueInt64() != tt.expected {
				t.Errorf("GetTfCfgInt64() = %v, want %v", result.ValueInt64(), tt.expected)
			}
		})
	}
}

func TestGetTfCfgBool(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]any
		key      string
		expected bool
		isNull   bool
	}{
		{"true", map[string]any{"key": true}, "key", true, false},
		{"false", map[string]any{"key": false}, "key", false, false},
		{"missing", map[string]any{}, "key", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTfCfgBool(tt.cfg, tt.key)
			if tt.isNull && !result.IsNull() {
				t.Errorf("Expected null, got: %v", result)
			}
			if !tt.isNull && result.ValueBool() != tt.expected {
				t.Errorf("GetTfCfgBool() = %v, want %v", result.ValueBool(), tt.expected)
			}
		})
	}
}

func TestGetTfCfgListString(t *testing.T) {
	ctx := context.Background()

	t.Run("valid list", func(t *testing.T) {
		cfg := map[string]any{"key": []interface{}{"a", "b", "c"}}
		result := GetTfCfgListString(ctx, cfg, "key")
		if result.IsNull() {
			t.Error("Expected non-null list")
		}
		if len(result.Elements()) != 3 {
			t.Errorf("Expected 3 elements, got %d", len(result.Elements()))
		}
	})

	t.Run("missing key", func(t *testing.T) {
		cfg := map[string]any{}
		result := GetTfCfgListString(ctx, cfg, "key")
		if !result.IsNull() {
			t.Error("Expected null list")
		}
	})
}
```

**Step 2: Run helper tests**

Run: `go test ./internal/helper -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/helper/helper_test.go
git commit -m "test: add unit tests for helper type conversion functions"
```

---

## Task 8: Add Acceptance Test with Timeout Configuration

**Files:**
- Modify: `internal/provider/source_postgresql_resource_test.go`

**Step 1: Add test case with custom timeout**

Add to the existing test file:

```go
func TestAccSourcePostgreSQLResource_WithTimeout(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
variable "source_postgresql_hostname" {
	type = string
}
variable "source_postgresql_password" {
	type      = string
	sensitive = true
}
resource "streamkap_source_postgresql" "test_timeout" {
	name              = "test-source-postgresql-timeout"
	database_hostname = var.source_postgresql_hostname
	database_port     = "5432"
	database_user     = "postgresql"
	database_password = var.source_postgresql_password
	database_dbname   = "postgres"
	database_sslmode  = "require"
	schema_include_list = "streamkap"
	table_include_list  = "streamkap.customer"
	signal_data_collection_schema_or_database = "streamkap"
	slot_name         = "terraform_timeout_test_slot"
	publication_name  = "terraform_timeout_test_pub"
	ssh_enabled       = false

	timeouts {
		create = "10m"
		update = "10m"
		delete = "5m"
	}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test_timeout", "name", "test-source-postgresql-timeout"),
				),
			},
		},
	})
}
```

**Step 2: Add similar test for pipeline**

Add to `internal/provider/pipeline_resource_test.go`:

```go
// Add timeouts block to existing pipeline test configuration
// Add TestAccPipelineResource_WithTimeout test case
```

**Step 3: Run a smoke test locally (if credentials available)**

Run: `TF_ACC=1 go test ./internal/provider -v -run TestAccSourcePostgreSQLResource_WithTimeout -timeout 30m`

**Step 4: Commit**

```bash
git add internal/provider/*_resource_test.go
git commit -m "test: add acceptance tests for timeout configuration"
```

---

## Task 9: Update Documentation

**Files:**
- Modify: `docs/index.md`
- Regenerate: `docs/resources/*.md` (via go generate)

**Step 1: Add operational section to docs/index.md**

Add after the provider configuration section:

```markdown
## Operational Considerations

### Timeouts

Streamkap uses Kafka Connect under the hood, which can have variable response times
depending on cluster load and connector complexity. All resources support configurable
timeouts:

\`\`\`hcl
resource "streamkap_source_postgresql" "example" {
  # ... configuration

  timeouts {
    create = "10m"  # Default: 5m
    update = "10m"  # Default: 5m
    delete = "5m"   # Default: 3m
  }
}
\`\`\`

### Retry Behavior

The provider automatically retries transient errors with exponential backoff:
- Gateway errors (502, 503, 504)
- Connection timeouts
- Kafka Connect rebalancing

Validation errors and authentication failures are NOT retried.

### Resource Dependencies

Source connectors automatically create Kafka topics when they start streaming.
If you need to reference topics in pipelines, ensure proper ordering:

\`\`\`hcl
resource "streamkap_source_postgresql" "orders" {
  name = "orders-source"
  # ...
}

resource "streamkap_pipeline" "orders_pipeline" {
  name = "orders-to-snowflake"
  # ...

  # Ensure source is created first
  depends_on = [streamkap_source_postgresql.orders]
}
\`\`\`
```

**Step 2: Regenerate resource documentation**

Run: `go generate ./...`

**Step 3: Commit**

```bash
git add docs/
git commit -m "docs: add operational considerations and timeout documentation

- Document timeout configuration for all resources
- Explain retry behavior and transient error handling
- Add resource dependency guidance"
```

---

## Task 10: Update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

**Step 1: Add unreleased section**

```markdown
## [Unreleased]

### Added
- Configurable timeouts for all resources (create, update, delete operations)
- Automatic retry with exponential backoff for transient errors
- Unit tests for helper functions and retry logic

### Changed
- Improved error handling with retry for transient failures
- Default timeouts: create/update 5m, delete 3m

### Technical Details
- Backend investigation confirmed KC timeout is 15s (not 90s)
- Conservative retry strategy: 3 attempts, 10s minimum delay
- Retry only on mutating operations (not reads)
```

**Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: update CHANGELOG for resilience improvements"
```

---

## Task 11: Final Verification

**Step 1: Run all unit tests**

```bash
go test ./... -v
```

**Step 2: Build and install**

```bash
go install .
```

**Step 3: Run acceptance tests (if credentials available)**

```bash
make testacc
```

**Step 4: Manual smoke test**

Create a test Terraform configuration and verify:
1. Timeout block is accepted
2. Custom timeouts work
3. Retry behavior on transient errors

**Step 5: Create final summary commit (if any changes needed)**

```bash
git add -A
git status
# Only commit if there are changes
```

---

## Execution Notes

**Recommended Execution Order:**
1. Tasks 1-3 (dependencies, retry logic, API integration) - Foundation
2. Tasks 4-6 (timeouts for all resource types) - Core functionality
3. Tasks 7-8 (tests) - Verification
4. Tasks 9-11 (docs and finalization) - Polish

**Key Decision Points:**
- After Task 3: Verify retry logic compiles and tests pass
- After Task 6: Verify all resources build with timeout blocks
- After Task 8: Run acceptance tests if credentials available

**What Was Removed from Original Plan:**
- Topic validation: Topics use names, not IDs; backend validates at create time
- terraform-plugin-sdk/v2/helper/retry: Using custom implementation instead

**Parallel Opportunities:**
- Tasks 4-6 can run in parallel (independent files)
- Tasks 7-8 can run in parallel (independent test files)
- Tasks 9-10 can run in parallel (independent docs)
