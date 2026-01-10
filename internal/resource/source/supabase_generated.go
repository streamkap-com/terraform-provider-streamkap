// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// SupabaseConfig implements the ConnectorConfig interface for Supabase sources.
type SupabaseConfig struct{}

// Ensure SupabaseConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*SupabaseConfig)(nil)

// GetSchema returns the Terraform schema for Supabase source.
func (c *SupabaseConfig) GetSchema() schema.Schema {
	return generated.SourceSupabaseSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *SupabaseConfig) GetFieldMappings() map[string]string {
	return generated.SourceSupabaseFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *SupabaseConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for Supabase.
func (c *SupabaseConfig) GetConnectorCode() string {
	return "supabase"
}

// GetResourceName returns the Terraform resource name.
func (c *SupabaseConfig) GetResourceName() string {
	return "source_supabase"
}

// NewModelInstance returns a new instance of the Supabase model.
func (c *SupabaseConfig) NewModelInstance() any {
	return &generated.SourceSupabaseModel{}
}

// NewSupabaseResource creates a new Supabase source resource.
func NewSupabaseResource() resource.Resource {
	return connector.NewBaseConnectorResource(&SupabaseConfig{})
}
