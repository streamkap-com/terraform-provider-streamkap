// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/shared"
)

// sourceSqlserverModelWithDeprecated embeds the generated model and adds
// deprecated field aliases (v2.1.19 unsuffixed names) so reflection-based
// marshaling can find them. The v2.1.19 schema had a single insert_static
// pair; the new schema uses the _1 suffix.
type sourceSqlserverModelWithDeprecated struct {
	generated.SourceSqlserverawsModel
	InsertStaticKeyField   types.String `tfsdk:"insert_static_key_field"`
	InsertStaticKeyValue   types.String `tfsdk:"insert_static_key_value"`
	InsertStaticValueField types.String `tfsdk:"insert_static_value_field"`
	InsertStaticValue      types.String `tfsdk:"insert_static_value"`
	SnapshotParallelismOld types.Int64  `tfsdk:"snapshot_parallelism"`
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
	return mappings
}()

// SQLServerConfig implements the ConnectorConfig interface for SQL Server sources.
type SQLServerConfig struct{}

// Ensure SQLServerConfig implements ConnectorConfig.
var (
	_ connector.ConnectorConfig          = (*SQLServerConfig)(nil)
	_ connector.ConnectorConfigValidator = (*SQLServerConfig)(nil)
)

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
	// snapshot_large_table_threshold has no alias: the backend dropped
	// streamkap.snapshot.large.table.threshold, so there is nothing to point at.
	int64Aliases := []struct {
		oldName, newName string
	}{
		{"snapshot_parallelism", "streamkap_snapshot_parallelism"},
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

// ValidateConfiguration enforces mutual exclusivity between the SQL and Entra
// Service Principal auth modes. The backend does not reject cross-mode
// credentials on write, so the provider refuses them at plan time. Unknown
// values defer to apply: a validator cannot decide on a value not yet resolved.
func (c *SQLServerConfig) ValidateConfiguration(model any, _ any) diag.Diagnostics {
	var diags diag.Diagnostics
	values := shared.CaptureFields(model, []string{
		"auth_mode",
		"database_user",
		"database_password",
		"azure_tenant_id",
		"azure_client_id",
		"azure_client_secret",
	})
	mode, known := sqlServerAuthMode(values["auth_mode"])
	if !known {
		return diags
	}
	sqlFields := []string{"database_user", "database_password"}
	entraFields := []string{"azure_tenant_id", "azure_client_id", "azure_client_secret"}
	switch mode {
	case "entra_service_principal":
		requireSet(&diags, values, entraFields, mode)
		requireUnset(&diags, values, sqlFields, mode)
	default: // "sql" and any other value stringvalidator.OneOf allows through
		requireSet(&diags, values, sqlFields, mode)
		requireUnset(&diags, values, entraFields, mode)
	}
	return diags
}

// sqlServerAuthMode returns the effective auth_mode. Null falls back to the
// schema default "sql"; known is false while the value is still unknown.
func sqlServerAuthMode(v attr.Value) (mode string, known bool) {
	s, ok := v.(types.String)
	if !ok || s.IsUnknown() {
		return "", false
	}
	if s.IsNull() {
		return "sql", true
	}
	return s.ValueString(), true
}

// requireSet reports each named attribute that is explicitly null in the
// configuration; unknown values defer to apply.
func requireSet(diags *diag.Diagnostics, values map[string]attr.Value, names []string, mode string) {
	for _, name := range names {
		v, ok := values[name].(types.String)
		if !ok || v.IsUnknown() {
			continue
		}
		if v.IsNull() {
			diags.AddAttributeError(path.Root(name), "Missing "+name,
				name+" is required when auth_mode is "+formatAuthMode(mode)+".")
		}
	}
}

// requireUnset reports each named attribute that is set in the configuration
// but is not used by the selected auth mode; unknown values defer to apply.
func requireUnset(diags *diag.Diagnostics, values map[string]attr.Value, names []string, mode string) {
	for _, name := range names {
		v, ok := values[name].(types.String)
		if !ok || v.IsUnknown() || v.IsNull() {
			continue
		}
		diags.AddAttributeError(path.Root(name), "Unexpected "+name,
			name+" is not used when auth_mode is "+formatAuthMode(mode)+"; remove it.")
	}
}

func formatAuthMode(mode string) string { return `"` + mode + `"` }
