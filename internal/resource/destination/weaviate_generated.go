package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

type WeaviateDestConfig struct{}

var _ connector.ConnectorConfig = (*WeaviateDestConfig)(nil)

func (c *WeaviateDestConfig) GetSchema() schema.Schema { return generated.DestinationWeaviateSchema() }
func (c *WeaviateDestConfig) GetFieldMappings() map[string]string {
	return generated.DestinationWeaviateFieldMappings
}
func (c *WeaviateDestConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}
func (c *WeaviateDestConfig) GetConnectorCode() string { return "weaviate" }
func (c *WeaviateDestConfig) GetResourceName() string  { return "destination_weaviate" }
func (c *WeaviateDestConfig) NewModelInstance() any    { return &generated.DestinationWeaviateModel{} }

func NewWeaviateResource() resource.Resource {
	return connector.NewBaseConnectorResource(&WeaviateDestConfig{})
}
