// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// StripeWebhookConfig implements the ConnectorConfig interface for Stripe webhook sources.
type StripeWebhookConfig struct{}

var _ connector.ConnectorConfig = (*StripeWebhookConfig)(nil)

func (c *StripeWebhookConfig) GetSchema() schema.Schema {
	return generated.SourceStripeWebhookSchema()
}

func (c *StripeWebhookConfig) GetFieldMappings() map[string]string {
	return generated.SourceStripeWebhookFieldMappings
}

func (c *StripeWebhookConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

func (c *StripeWebhookConfig) GetConnectorCode() string {
	return "stripe_webhook"
}

func (c *StripeWebhookConfig) GetResourceName() string {
	return "source_stripe_webhook"
}

func (c *StripeWebhookConfig) NewModelInstance() any {
	return &generated.SourceStripeWebhookModel{}
}

// NewStripeWebhookResource creates a new Stripe webhook source resource.
func NewStripeWebhookResource() resource.Resource {
	return connector.NewBaseConnectorResource(&StripeWebhookConfig{})
}
