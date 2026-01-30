// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// WebhookConfig implements the ConnectorConfig interface for Webhook sources.
type WebhookConfig struct{}

// Ensure WebhookConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*WebhookConfig)(nil)

// GetSchema returns the Terraform schema for Webhook source.
func (c *WebhookConfig) GetSchema() schema.Schema {
	return generated.SourceWebhookSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *WebhookConfig) GetFieldMappings() map[string]string {
	return generated.SourceWebhookFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *WebhookConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for Webhook.
func (c *WebhookConfig) GetConnectorCode() string {
	return "webhook"
}

// GetResourceName returns the Terraform resource name.
func (c *WebhookConfig) GetResourceName() string {
	return "source_webhook"
}

// NewModelInstance returns a new instance of the Webhook model.
func (c *WebhookConfig) NewModelInstance() any {
	return &generated.SourceWebhookModel{}
}

// NewWebhookResource creates a new Webhook source resource.
func NewWebhookResource() resource.Resource {
	return connector.NewBaseConnectorResource(&WebhookConfig{})
}
