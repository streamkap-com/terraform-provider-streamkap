// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// destinationSnowflakeModelWithDeprecated embeds the generated model and adds
// the deprecated auto_schema_creation alias so reflection-based marshaling can
// find it. Without this wrapper, the alias is invisible to field-index
// construction and silently reads as null on refresh.
type destinationSnowflakeModelWithDeprecated struct {
	generated.DestinationSnowflakeModel
	AutoSchemaCreationOld types.Bool `tfsdk:"auto_schema_creation"`
}

// snowflakeFieldMappings extends the generated field mappings with deprecated aliases.
var snowflakeFieldMappings = func() map[string]string {
	mappings := make(map[string]string)
	for k, v := range generated.DestinationSnowflakeFieldMappings {
		mappings[k] = v
	}
	// Deprecated alias - maps to same API field as create_schema_auto.
	mappings["auto_schema_creation"] = "create.schema.auto"
	return mappings
}()

// SnowflakeConfig implements the ConnectorConfig interface for Snowflake destinations.
type SnowflakeConfig struct{}

// Ensure SnowflakeConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*SnowflakeConfig)(nil)

// GetSchema returns the Terraform schema for Snowflake destination.
func (c *SnowflakeConfig) GetSchema() schema.Schema {
	s := generated.DestinationSnowflakeSchema()
	// Add deprecated alias. ConflictsWith prevents users from setting both the
	// old and new attribute name at the same time, which would otherwise race
	// on map-iteration order during marshaling.
	s.Attributes["auto_schema_creation"] = schema.BoolAttribute{
		Optional:           true,
		Computed:           true,
		DeprecationMessage: "Use 'create_schema_auto' instead.",
		Description:        "DEPRECATED: Use 'create_schema_auto' instead.",
		Validators: []validator.Bool{
			boolvalidator.ConflictsWith(path.MatchRoot("create_schema_auto")),
		},
	}
	return s
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *SnowflakeConfig) GetFieldMappings() map[string]string {
	return snowflakeFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *SnowflakeConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for Snowflake.
func (c *SnowflakeConfig) GetConnectorCode() string {
	return "snowflake"
}

// GetResourceName returns the Terraform resource name.
func (c *SnowflakeConfig) GetResourceName() string {
	return "destination_snowflake"
}

// NewModelInstance returns a new instance of the Snowflake model. We return
// the wrapper struct so the deprecated alias is visible to reflection-based
// marshaling (see destinationSnowflakeModelWithDeprecated above).
func (c *SnowflakeConfig) NewModelInstance() any {
	return &destinationSnowflakeModelWithDeprecated{}
}

// NewSnowflakeResource creates a new Snowflake destination resource.
func NewSnowflakeResource() resource.Resource {
	return connector.NewBaseConnectorResource(&SnowflakeConfig{})
}
