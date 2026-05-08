// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// ZendeskWebhookConfig implements the ConnectorConfig interface for Zendesk webhook sources.
type ZendeskWebhookConfig struct{}

var _ connector.ConnectorConfig = (*ZendeskWebhookConfig)(nil)

func (c *ZendeskWebhookConfig) GetSchema() schema.Schema {
	return generated.SourceZendeskWebhookSchema()
}

func (c *ZendeskWebhookConfig) GetFieldMappings() map[string]string {
	return generated.SourceZendeskWebhookFieldMappings
}

func (c *ZendeskWebhookConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

func (c *ZendeskWebhookConfig) GetConnectorCode() string {
	return "zendesk_webhook"
}

func (c *ZendeskWebhookConfig) GetResourceName() string {
	return "source_zendesk_webhook"
}

func (c *ZendeskWebhookConfig) NewModelInstance() any {
	return &generated.SourceZendeskWebhookModel{}
}

// NewZendeskWebhookResource creates a new Zendesk webhook source resource.
func NewZendeskWebhookResource() resource.Resource {
	return connector.NewBaseConnectorResource(&ZendeskWebhookConfig{})
}
