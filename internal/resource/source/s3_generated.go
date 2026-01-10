// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// S3SourceConfig implements the ConnectorConfig interface for S3 sources.
type S3SourceConfig struct{}

// Ensure S3SourceConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*S3SourceConfig)(nil)

// GetSchema returns the Terraform schema for S3 source.
func (c *S3SourceConfig) GetSchema() schema.Schema {
	return generated.SourceS3Schema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *S3SourceConfig) GetFieldMappings() map[string]string {
	return generated.SourceS3FieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *S3SourceConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for S3.
func (c *S3SourceConfig) GetConnectorCode() string {
	return "s3"
}

// GetResourceName returns the Terraform resource name.
func (c *S3SourceConfig) GetResourceName() string {
	return "source_s3"
}

// NewModelInstance returns a new instance of the S3 model.
func (c *S3SourceConfig) NewModelInstance() any {
	return &generated.SourceS3Model{}
}

// NewS3SourceResource creates a new S3 source resource.
func NewS3SourceResource() resource.Resource {
	return connector.NewBaseConnectorResource(&S3SourceConfig{})
}
