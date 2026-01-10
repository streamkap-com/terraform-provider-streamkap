// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// VitessConfig implements the ConnectorConfig interface for Vitess sources.
type VitessConfig struct{}

// Ensure VitessConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*VitessConfig)(nil)

// GetSchema returns the Terraform schema for Vitess source.
func (c *VitessConfig) GetSchema() schema.Schema {
	return generated.SourceVitessSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *VitessConfig) GetFieldMappings() map[string]string {
	return generated.SourceVitessFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *VitessConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for Vitess.
func (c *VitessConfig) GetConnectorCode() string {
	return "vitess"
}

// GetResourceName returns the Terraform resource name.
func (c *VitessConfig) GetResourceName() string {
	return "source_vitess"
}

// NewModelInstance returns a new instance of the Vitess model.
func (c *VitessConfig) NewModelInstance() any {
	return &generated.SourceVitessModel{}
}

// NewVitessResource creates a new Vitess source resource.
func NewVitessResource() resource.Resource {
	return connector.NewBaseConnectorResource(&VitessConfig{})
}
