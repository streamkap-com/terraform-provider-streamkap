// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// ElasticsearchConfig implements the ConnectorConfig interface for Elasticsearch sources.
type ElasticsearchConfig struct{}

// Ensure ElasticsearchConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*ElasticsearchConfig)(nil)

// GetSchema returns the Terraform schema for Elasticsearch source.
func (c *ElasticsearchConfig) GetSchema() schema.Schema {
	return generated.SourceElasticsearchSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *ElasticsearchConfig) GetFieldMappings() map[string]string {
	return generated.SourceElasticsearchFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *ElasticsearchConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for Elasticsearch.
func (c *ElasticsearchConfig) GetConnectorCode() string {
	return "elasticsearch"
}

// GetResourceName returns the Terraform resource name.
func (c *ElasticsearchConfig) GetResourceName() string {
	return "source_elasticsearch"
}

// NewModelInstance returns a new instance of the Elasticsearch model.
func (c *ElasticsearchConfig) NewModelInstance() any {
	return &generated.SourceElasticsearchModel{}
}

// NewElasticsearchResource creates a new Elasticsearch source resource.
func NewElasticsearchResource() resource.Resource {
	return connector.NewBaseConnectorResource(&ElasticsearchConfig{})
}
