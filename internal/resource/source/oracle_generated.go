// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// OracleConfig implements the ConnectorConfig interface for Oracle sources.
type OracleConfig struct{}

// Ensure OracleConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*OracleConfig)(nil)

// GetSchema returns the Terraform schema for Oracle source.
func (c *OracleConfig) GetSchema() schema.Schema {
	return generated.SourceOracleSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *OracleConfig) GetFieldMappings() map[string]string {
	return generated.SourceOracleFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *OracleConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for Oracle.
func (c *OracleConfig) GetConnectorCode() string {
	return "oracle"
}

// GetResourceName returns the Terraform resource name.
func (c *OracleConfig) GetResourceName() string {
	return "source_oracle"
}

// NewModelInstance returns a new instance of the Oracle model.
func (c *OracleConfig) NewModelInstance() any {
	return &generated.SourceOracleModel{}
}

// NewOracleResource creates a new Oracle source resource.
func NewOracleResource() resource.Resource {
	return connector.NewBaseConnectorResource(&OracleConfig{})
}
