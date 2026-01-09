// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// MongoDBConfig implements the ConnectorConfig interface for MongoDB sources.
type MongoDBConfig struct{}

// Ensure MongoDBConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*MongoDBConfig)(nil)

// GetSchema returns the Terraform schema for MongoDB source.
func (c *MongoDBConfig) GetSchema() schema.Schema {
	return generated.SourceMongodbSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *MongoDBConfig) GetFieldMappings() map[string]string {
	return generated.SourceMongodbFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *MongoDBConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for MongoDB.
func (c *MongoDBConfig) GetConnectorCode() string {
	return "mongodb"
}

// GetResourceName returns the Terraform resource name.
func (c *MongoDBConfig) GetResourceName() string {
	return "streamkap_source_mongodb"
}

// NewModelInstance returns a new instance of the MongoDB model.
func (c *MongoDBConfig) NewModelInstance() any {
	return &generated.SourceMongodbModel{}
}

// NewMongoDBResource creates a new MongoDB source resource.
func NewMongoDBResource() resource.Resource {
	return connector.NewBaseConnectorResource(&MongoDBConfig{})
}
