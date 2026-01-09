# Development Guide

## Prerequisites

- **Go 1.21+** - [Install Go](https://golang.org/doc/install)
- **Terraform 1.0+** - [Install Terraform](https://developer.hashicorp.com/terraform/downloads)
- **Streamkap API credentials** - Client ID and Secret from [Streamkap Services](https://app.streamkap.com)
- **Backend repository access** (for schema generation) - `/path/to/python-be-streamkap`

## Quick Start

```bash
# Clone repository
git clone https://github.com/streamkap-com/terraform-provider-streamkap.git
cd terraform-provider-streamkap

# Copy environment template and fill in credentials
cp .env.example .env
# Edit .env with your credentials

# Build
go build ./...

# Run unit tests
go test -v -short ./...

# Run acceptance tests (creates real resources)
export $(grep -v '^#' .env | xargs)
go test -v ./internal/provider -timeout 30m
```

## Local Development Setup

### 1. Configure Terraform Dev Overrides

Create or edit `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "github.com/streamkap-com/streamkap" = "/path/to/your/go/bin"
  }
  direct {}
}
```

Replace `/path/to/your/go/bin` with your `$GOPATH/bin` (usually `~/go/bin`).

### 2. Build and Install

```bash
go install .
```

### 3. Test with Terraform

```bash
cd examples/provider
terraform plan
```

Note: With dev_overrides, you don't need `terraform init`.

## Code Generation

The provider uses code generation from backend `configuration.latest.json` files.

### Generate a Single Connector

```bash
go run cmd/tfgen/main.go generate \
  --config /path/to/backend/app/sources/plugins/postgresql/configuration.latest.json \
  --type source \
  --output internal/generated/
```

### Regenerate All Connectors

```bash
# Sources
for config in /path/to/backend/app/sources/plugins/*/configuration.latest.json; do
  connector=$(basename $(dirname "$config"))
  go run cmd/tfgen/main.go generate --config "$config" --type source --output internal/generated/
done

# Destinations
for config in /path/to/backend/app/destinations/plugins/*/configuration.latest.json; do
  connector=$(basename $(dirname "$config"))
  go run cmd/tfgen/main.go generate --config "$config" --type destination --output internal/generated/
done
```

### Generated Files

- `internal/generated/source_<name>.go` - Schema and model for source
- `internal/generated/destination_<name>.go` - Schema and model for destination

## Adding a New Connector

### Step 1: Locate Backend Config

Find the `configuration.latest.json` file:
```
backend/app/sources/plugins/<connector>/configuration.latest.json
backend/app/destinations/plugins/<connector>/configuration.latest.json
```

### Step 2: Generate Schema

```bash
# For sources:
go run cmd/tfgen/main.go generate \
  --config /path/to/configuration.latest.json \
  --type source \
  --output internal/generated/

# For destinations, use --type destination instead
```

### Step 3: Create Config Wrapper

Create `internal/resource/source/<connector>_generated.go`:

```go
package source

import (
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"

    "github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
    "github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

type MyConnectorConfig struct{}

var _ connector.ConnectorConfig = (*MyConnectorConfig)(nil)

func (c *MyConnectorConfig) GetSchema() schema.Schema {
    return generated.SourceMyconnectorSchema()
}

func (c *MyConnectorConfig) GetFieldMappings() map[string]string {
    return generated.SourceMyconnectorFieldMappings
}

func (c *MyConnectorConfig) GetConnectorType() connector.ConnectorType {
    return connector.ConnectorTypeSource
}

func (c *MyConnectorConfig) GetConnectorCode() string {
    return "myconnector"
}

func (c *MyConnectorConfig) GetResourceName() string {
    return "source_myconnector"
}

func (c *MyConnectorConfig) NewModelInstance() any {
    return &generated.SourceMyconnectorModel{}
}

func NewMyConnectorResource() resource.Resource {
    return connector.NewBaseConnectorResource(&MyConnectorConfig{})
}
```

### Step 4: Register in Provider

Edit `internal/provider/provider.go`:

```go
func (p *streamkapProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        // ... existing resources
        source.NewMyConnectorResource,
    }
}
```

### Step 5: Add Acceptance Test

Create `internal/provider/source_myconnector_resource_test.go`:

```go
func TestAccSourceMyConnectorResource(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: providerConfig + `
resource "streamkap_source_myconnector" "test" {
    name = "test-connector"
    // ... required fields
}
`,
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("streamkap_source_myconnector.test", "name", "test-connector"),
                ),
            },
            {
                ResourceName:      "streamkap_source_myconnector.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
        },
    })
}
```

### Step 6: Add Example

Create `examples/resources/streamkap_source_myconnector/resource.tf`:

```hcl
resource "streamkap_source_myconnector" "example" {
  name = "my-connector"
  // ... configuration
}
```

### Step 7: Generate Documentation

```bash
go generate ./...
```

## Running Tests

### Unit Tests

```bash
go test -v -short ./...
```

### Specific Test

```bash
go test -v ./internal/provider -run TestAccSourceKafkaDirectResource
```

### All Acceptance Tests

```bash
export TF_ACC=1
export $(grep -v '^#' .env | xargs)
go test -v ./internal/provider -timeout 30m
```

### Generator Tests

```bash
go test -v ./cmd/tfgen/...
```

## Code Style

- Run `go fmt ./...` before committing
- Run `golangci-lint run` for linting
- Use pre-commit hooks: `pre-commit install`

## Troubleshooting

### "Resource type not found"

Ensure:
1. Resource is registered in `provider.go`
2. `GetResourceName()` returns the correct suffix (without `streamkap_` prefix)
3. Provider is rebuilt: `go install .`

### "Unknown attribute"

Check that:
1. Field exists in generated schema
2. Field name in TF matches `tfsdk` tag in model
3. Field mappings are correct

### API Errors

Enable debug logging:
```bash
TF_LOG=DEBUG terraform plan
```

## CI/CD Workflows

- **ci.yml** - Build, lint, test on PRs
- **security.yml** - Trivy and Checkov security scans
- **regenerate.yml** - Manual schema regeneration
- **release.yml** - GoReleaser on tags
