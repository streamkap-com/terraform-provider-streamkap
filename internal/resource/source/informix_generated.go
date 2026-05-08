// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// InformixConfig implements the ConnectorConfig interface for Informix sources.
type InformixConfig struct{}

var _ connector.ConnectorConfig = (*InformixConfig)(nil)

func (c *InformixConfig) GetSchema() schema.Schema {
	return generated.SourceInformixSchema()
}

func (c *InformixConfig) GetFieldMappings() map[string]string {
	return generated.SourceInformixFieldMappings
}

func (c *InformixConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

func (c *InformixConfig) GetConnectorCode() string {
	return "informix"
}

func (c *InformixConfig) GetResourceName() string {
	return "source_informix"
}

func (c *InformixConfig) NewModelInstance() any {
	return &generated.SourceInformixModel{}
}

// NewInformixResource creates a new Informix source resource.
func NewInformixResource() resource.Resource {
	return connector.NewBaseConnectorResource(&InformixConfig{})
}
