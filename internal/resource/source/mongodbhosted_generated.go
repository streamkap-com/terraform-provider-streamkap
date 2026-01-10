// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// MongoDBHostedConfig implements the ConnectorConfig interface for MongoDB Hosted sources.
type MongoDBHostedConfig struct{}

// Ensure MongoDBHostedConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*MongoDBHostedConfig)(nil)

// GetSchema returns the Terraform schema for MongoDB Hosted source.
func (c *MongoDBHostedConfig) GetSchema() schema.Schema {
	return generated.SourceMongodbhostedSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *MongoDBHostedConfig) GetFieldMappings() map[string]string {
	return generated.SourceMongodbhostedFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *MongoDBHostedConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for MongoDB Hosted.
func (c *MongoDBHostedConfig) GetConnectorCode() string {
	return "mongodbhosted"
}

// GetResourceName returns the Terraform resource name.
func (c *MongoDBHostedConfig) GetResourceName() string {
	return "source_mongodbhosted"
}

// NewModelInstance returns a new instance of the MongoDB Hosted model.
func (c *MongoDBHostedConfig) NewModelInstance() any {
	return &generated.SourceMongodbhostedModel{}
}

// NewMongoDBHostedResource creates a new MongoDB Hosted source resource.
func NewMongoDBHostedResource() resource.Resource {
	return connector.NewBaseConnectorResource(&MongoDBHostedConfig{})
}
