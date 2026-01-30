// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// OracleAWSConfig implements the ConnectorConfig interface for Oracle AWS (RDS) sources.
type OracleAWSConfig struct{}

// Ensure OracleAWSConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*OracleAWSConfig)(nil)

// GetSchema returns the Terraform schema for Oracle AWS source.
func (c *OracleAWSConfig) GetSchema() schema.Schema {
	return generated.SourceOracleawsSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *OracleAWSConfig) GetFieldMappings() map[string]string {
	return generated.SourceOracleawsFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *OracleAWSConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for Oracle AWS.
func (c *OracleAWSConfig) GetConnectorCode() string {
	return "oracleaws"
}

// GetResourceName returns the Terraform resource name.
func (c *OracleAWSConfig) GetResourceName() string {
	return "source_oracleaws"
}

// NewModelInstance returns a new instance of the Oracle AWS model.
func (c *OracleAWSConfig) NewModelInstance() any {
	return &generated.SourceOracleawsModel{}
}

// NewOracleAWSResource creates a new Oracle AWS source resource.
func NewOracleAWSResource() resource.Resource {
	return connector.NewBaseConnectorResource(&OracleAWSConfig{})
}
