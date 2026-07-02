// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// ShopifyWebhookConfig implements the ConnectorConfig interface for Shopify webhook sources.
type ShopifyWebhookConfig struct{}

var _ connector.ConnectorConfig = (*ShopifyWebhookConfig)(nil)

func (c *ShopifyWebhookConfig) GetSchema() schema.Schema {
	return generated.SourceShopifyWebhookSchema()
}

func (c *ShopifyWebhookConfig) GetFieldMappings() map[string]string {
	return generated.SourceShopifyWebhookFieldMappings
}

func (c *ShopifyWebhookConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

func (c *ShopifyWebhookConfig) GetConnectorCode() string {
	return "shopify_webhook"
}

func (c *ShopifyWebhookConfig) GetResourceName() string {
	return "source_shopify_webhook"
}

func (c *ShopifyWebhookConfig) NewModelInstance() any {
	return &generated.SourceShopifyWebhookModel{}
}

// NewShopifyWebhookResource creates a new Shopify webhook source resource.
func NewShopifyWebhookResource() resource.Resource {
	return connector.NewBaseConnectorResource(&ShopifyWebhookConfig{})
}
