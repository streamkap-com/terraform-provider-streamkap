// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// sourceSqlserverModelWithDeprecated embeds the generated model and adds
// deprecated field aliases (v2.1.19 unsuffixed names) so reflection-based
// marshaling can find them. The v2.1.19 schema had a single insert_static
// pair; the new schema uses the _1 suffix.
type sourceSqlserverModelWithDeprecated struct {
	generated.SourceSqlserverawsModel
	InsertStaticKeyField           types.String `tfsdk:"insert_static_key_field"`
	InsertStaticKeyValue           types.String `tfsdk:"insert_static_key_value"`
	InsertStaticValueField         types.String `tfsdk:"insert_static_value_field"`
	InsertStaticValue              types.String `tfsdk:"insert_static_value"`
	SnapshotParallelismOld         types.Int64  `tfsdk:"snapshot_parallelism"`
	SnapshotLargeTableThresholdOld types.Int64  `tfsdk:"snapshot_large_table_threshold"`
}

// sqlserverFieldMappings extends the generated field mappings with deprecated aliases.
var sqlserverFieldMappings = func() map[string]string {
	mappings := make(map[string]string)
	for k, v := range generated.SourceSqlserverawsFieldMappings {
		mappings[k] = v
	}
	// Deprecated aliases - map to same API fields as the _1 new names
	mappings["insert_static_key_field"] = "transforms.InsertStaticKey1.static.field"
	mappings["insert_static_key_value"] = "transforms.InsertStaticKey1.static.value"
	mappings["insert_static_value_field"] = "transforms.InsertStaticValue1.static.field"
	mappings["insert_static_value"] = "transforms.InsertStaticValue1.static.value"
	mappings["snapshot_parallelism"] = "streamkap.snapshot.parallelism"
	mappings["snapshot_large_table_threshold"] = "streamkap.snapshot.large.table.threshold"
	return mappings
}()

// SQLServerConfig implements the ConnectorConfig interface for SQL Server sources.
type SQLServerConfig struct{}

// Ensure SQLServerConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*SQLServerConfig)(nil)

// GetSchema returns the Terraform schema for SQL Server source.
func (c *SQLServerConfig) GetSchema() schema.Schema {
	s := generated.SourceSqlserverawsSchema()
	// Deprecated aliases — v2.1.19 unsuffixed names map to the _1 new names.
	// Computed so the provider can populate them from the API on Read.
	// ConflictsWith prevents setting both old and new names simultaneously.
	deprecatedAliases := []struct {
		oldName, newName string
	}{
		{"insert_static_key_field", "transforms_insert_static_key1_static_field"},
		{"insert_static_key_value", "transforms_insert_static_key1_static_value"},
		{"insert_static_value_field", "transforms_insert_static_value1_static_field"},
		{"insert_static_value", "transforms_insert_static_value1_static_value"},
	}
	for _, alias := range deprecatedAliases {
		s.Attributes[alias.oldName] = schema.StringAttribute{
			Optional:           true,
			Computed:           true,
			DeprecationMessage: "Use '" + alias.newName + "' instead.",
			Description:        "DEPRECATED: Use '" + alias.newName + "' instead.",
			Validators: []validator.String{
				stringvalidator.ConflictsWith(path.MatchRoot(alias.newName)),
			},
		}
	}
	// Int64 deprecated aliases (snapshot fields were renamed with streamkap_ prefix).
	int64Aliases := []struct {
		oldName, newName string
	}{
		{"snapshot_parallelism", "streamkap_snapshot_parallelism"},
		{"snapshot_large_table_threshold", "streamkap_snapshot_large_table_threshold"},
	}
	for _, alias := range int64Aliases {
		s.Attributes[alias.oldName] = schema.Int64Attribute{
			Optional:           true,
			Computed:           true,
			DeprecationMessage: "Use '" + alias.newName + "' instead.",
			Description:        "DEPRECATED: Use '" + alias.newName + "' instead.",
			Validators: []validator.Int64{
				int64validator.ConflictsWith(path.MatchRoot(alias.newName)),
			},
		}
	}
	return s
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *SQLServerConfig) GetFieldMappings() map[string]string {
	return sqlserverFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *SQLServerConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for SQL Server.
func (c *SQLServerConfig) GetConnectorCode() string {
	return "sqlserveraws"
}

// GetResourceName returns the Terraform resource name.
func (c *SQLServerConfig) GetResourceName() string {
	return "source_sqlserver"
}

// NewModelInstance returns a new instance of the SQL Server model with deprecated fields.
func (c *SQLServerConfig) NewModelInstance() any {
	return &sourceSqlserverModelWithDeprecated{}
}

// NewSQLServerResource creates a new SQL Server source resource.
func NewSQLServerResource() resource.Resource {
	return connector.NewBaseConnectorResource(&SQLServerConfig{})
}
