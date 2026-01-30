// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// DynamoDBConfig implements the ConnectorConfig interface for DynamoDB sources.
type DynamoDBConfig struct{}

// Ensure DynamoDBConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*DynamoDBConfig)(nil)

// GetSchema returns the Terraform schema for DynamoDB source.
func (c *DynamoDBConfig) GetSchema() schema.Schema {
	return generated.SourceDynamodbSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *DynamoDBConfig) GetFieldMappings() map[string]string {
	return generated.SourceDynamodbFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *DynamoDBConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for DynamoDB.
func (c *DynamoDBConfig) GetConnectorCode() string {
	return "dynamodb"
}

// GetResourceName returns the Terraform resource name.
func (c *DynamoDBConfig) GetResourceName() string {
	return "source_dynamodb"
}

// NewModelInstance returns a new instance of the DynamoDB model.
func (c *DynamoDBConfig) NewModelInstance() any {
	return &generated.SourceDynamodbModel{}
}

// NewDynamoDBResource creates a new DynamoDB source resource.
func NewDynamoDBResource() resource.Resource {
	return connector.NewBaseConnectorResource(&DynamoDBConfig{})
}
