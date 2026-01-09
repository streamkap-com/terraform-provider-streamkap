// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// S3Config implements the ConnectorConfig interface for S3 destinations.
type S3Config struct{}

// Ensure S3Config implements ConnectorConfig.
var _ connector.ConnectorConfig = (*S3Config)(nil)

// GetSchema returns the Terraform schema for S3 destination.
func (c *S3Config) GetSchema() schema.Schema {
	return generated.DestinationS3Schema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *S3Config) GetFieldMappings() map[string]string {
	return generated.DestinationS3FieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *S3Config) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for S3.
func (c *S3Config) GetConnectorCode() string {
	return "s3"
}

// GetResourceName returns the Terraform resource name.
func (c *S3Config) GetResourceName() string {
	return "streamkap_destination_s3"
}

// NewModelInstance returns a new instance of the S3 model.
func (c *S3Config) NewModelInstance() any {
	return &generated.DestinationS3Model{}
}

// NewS3Resource creates a new S3 destination resource.
func NewS3Resource() resource.Resource {
	return connector.NewBaseConnectorResource(&S3Config{})
}
