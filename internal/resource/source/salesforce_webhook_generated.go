// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// SalesforceWebhookConfig implements the ConnectorConfig interface for Salesforce webhook sources.
type SalesforceWebhookConfig struct{}

var _ connector.ConnectorConfig = (*SalesforceWebhookConfig)(nil)

func (c *SalesforceWebhookConfig) GetSchema() schema.Schema {
	return generated.SourceSalesforceWebhookSchema()
}

func (c *SalesforceWebhookConfig) GetFieldMappings() map[string]string {
	return generated.SourceSalesforceWebhookFieldMappings
}

func (c *SalesforceWebhookConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

func (c *SalesforceWebhookConfig) GetConnectorCode() string {
	return "salesforce_webhook"
}

func (c *SalesforceWebhookConfig) GetResourceName() string {
	return "source_salesforce_webhook"
}

func (c *SalesforceWebhookConfig) NewModelInstance() any {
	return &generated.SourceSalesforceWebhookModel{}
}

// NewSalesforceWebhookResource creates a new Salesforce webhook source resource.
func NewSalesforceWebhookResource() resource.Resource {
	return connector.NewBaseConnectorResource(&SalesforceWebhookConfig{})
}
