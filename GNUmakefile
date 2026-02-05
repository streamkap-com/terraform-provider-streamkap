BINARY=terraform-provider-streamkap
GOBIN ?= $(shell go env GOPATH)/bin

default: build

# Build the provider
.PHONY: build
build:
	go build -o $(BINARY) .

# Install the provider locally
.PHONY: install
install:
	go install .

# Generate documentation
.PHONY: generate
generate:
	go generate ./...

# Run unit tests (fast, no API needed)
.PHONY: test
test:
	go test -v -short ./...

# Run schema compatibility tests
.PHONY: test-schema
test-schema:
	go test -v -run 'TestSchemaBackwardsCompatibility' ./internal/provider/...

# Run validator tests
.PHONY: test-validators
test-validators:
	go test -v -run 'Test.*Validator' ./internal/provider/...

# Run integration tests with VCR cassettes
.PHONY: test-integration
test-integration:
	go test -v -run 'TestIntegration_' ./internal/provider/...

# Run acceptance tests (requires API credentials)
.PHONY: testacc
testacc:
	TF_ACC=1 go test -v -timeout 120m -run 'TestAcc' ./internal/provider/...

# Run migration tests
.PHONY: test-migration
test-migration:
	TF_ACC=1 go test -v -timeout 180m -run 'TestAcc.*Migration' ./internal/provider/...

# Run all tests except acceptance
.PHONY: test-all
test-all: test test-schema test-validators test-integration

# Run linter
.PHONY: lint
lint:
	golangci-lint run ./...

# Format code
.PHONY: fmt
fmt:
	go fmt ./...
	gofmt -s -w .

# Tidy dependencies
.PHONY: tidy
tidy:
	go mod tidy

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY)
	go clean

# Record new VCR cassettes (requires API credentials)
.PHONY: cassettes
cassettes:
	UPDATE_CASSETTES=1 go test -v -run 'TestIntegration_' ./internal/provider/...

# Update schema snapshots after intentional changes
.PHONY: snapshots
snapshots:
	UPDATE_SNAPSHOTS=1 go test -v -run 'TestSchemaBackwardsCompatibility' ./internal/provider/...

# Run test sweepers to clean up orphaned resources
.PHONY: sweep
sweep:
	go test -v -run 'TestSweep' ./internal/provider/...

# Validate example files
.PHONY: validate-examples
validate-examples:
	@for f in examples/resources/*/basic.tf examples/resources/*/complete.tf; do \
		if [ -f "$$f" ]; then \
			echo "Validating $$f"; \
			terraform -chdir=$$(dirname $$f) validate || exit 1; \
		fi \
	done

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build            - Build the provider binary"
	@echo "  install          - Install provider to GOBIN"
	@echo "  generate         - Generate documentation"
	@echo "  test             - Run unit tests (fast)"
	@echo "  test-schema      - Run schema compatibility tests"
	@echo "  test-validators  - Run validator tests"
	@echo "  test-integration - Run integration tests with VCR"
	@echo "  testacc          - Run acceptance tests (requires API)"
	@echo "  test-migration   - Run migration tests (requires API)"
	@echo "  test-all         - Run all tests except acceptance"
	@echo "  lint             - Run golangci-lint"
	@echo "  fmt              - Format Go code"
	@echo "  tidy             - Tidy Go modules"
	@echo "  clean            - Remove build artifacts"
	@echo "  cassettes        - Record new VCR cassettes"
	@echo "  snapshots        - Update schema snapshots"
	@echo "  sweep            - Clean up orphaned test resources"
	@echo "  validate-examples - Validate example Terraform files"
