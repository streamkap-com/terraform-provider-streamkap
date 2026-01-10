// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// PlanetScaleConfig implements the ConnectorConfig interface for PlanetScale sources.
type PlanetScaleConfig struct{}

// Ensure PlanetScaleConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*PlanetScaleConfig)(nil)

// GetSchema returns the Terraform schema for PlanetScale source.
func (c *PlanetScaleConfig) GetSchema() schema.Schema {
	return generated.SourcePlanetscaleSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *PlanetScaleConfig) GetFieldMappings() map[string]string {
	return generated.SourcePlanetscaleFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *PlanetScaleConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for PlanetScale.
func (c *PlanetScaleConfig) GetConnectorCode() string {
	return "planetscale"
}

// GetResourceName returns the Terraform resource name.
func (c *PlanetScaleConfig) GetResourceName() string {
	return "source_planetscale"
}

// NewModelInstance returns a new instance of the PlanetScale model.
func (c *PlanetScaleConfig) NewModelInstance() any {
	return &generated.SourcePlanetscaleModel{}
}

// NewPlanetScaleResource creates a new PlanetScale source resource.
func NewPlanetScaleResource() resource.Resource {
	return connector.NewBaseConnectorResource(&PlanetScaleConfig{})
}
