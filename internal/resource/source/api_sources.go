package source

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/shared"
)

type apiSourceConfig struct {
	code         string
	schema       func() schema.Schema
	mappings     map[string]string
	model        func() any
	requirements []generated.APIRequirement
	oauth        bool
}

var _ connector.ConnectorConfig = (*apiSourceConfig)(nil)
var _ connector.ConnectorConfigValidator = (*apiSourceConfig)(nil)

func (c *apiSourceConfig) GetSchema() schema.Schema            { return c.schema() }
func (c *apiSourceConfig) GetFieldMappings() map[string]string { return c.mappings }
func (c *apiSourceConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}
func (c *apiSourceConfig) GetConnectorCode() string { return c.code }
func (c *apiSourceConfig) GetResourceName() string  { return "source_" + c.code }
func (c *apiSourceConfig) NewModelInstance() any    { return c.model() }
func (c *apiSourceConfig) ValidateConfiguration(model any, creating bool) diag.Diagnostics {
	var diags diag.Diagnostics
	names := make([]string, 0, len(c.requirements)*2+1)
	for _, requirement := range c.requirements {
		names = append(names, requirement.Field, requirement.ConditionField)
	}
	if c.oauth {
		names = append(names, "auth_mode")
	}
	values := shared.CaptureFields(model, names)
	if c.oauth && creating && stringValue(values["auth_mode"]) == "oauth" {
		diags.AddAttributeError(path.Root("auth_mode"), "OAuth setup is interactive", "Create the OAuth source with the CLI or MCP, then import it into Terraform.")
	}
	for _, requirement := range c.requirements {
		if condition := values[requirement.ConditionField]; condition != nil && condition.IsUnknown() {
			continue
		}
		conditionValue := stringValue(values[requirement.ConditionField])
		if conditionValue == "" {
			conditionValue = requirement.ConditionDefault
		}
		if conditionValue != requirement.ConditionValue {
			continue
		}
		value := values[requirement.Field]
		if value != nil && !value.IsUnknown() && (value.IsNull() || emptyValue(value)) {
			diags.AddAttributeError(path.Root(requirement.Field), "Missing API source value", requirement.Field+" is required when "+requirement.ConditionField+" is "+requirement.ConditionValue+".")
		}
	}
	return diags
}

func stringValue(value attr.Value) string {
	if v, ok := value.(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		return v.ValueString()
	}
	return ""
}

func emptyValue(value attr.Value) bool {
	switch v := value.(type) {
	case types.String:
		return v.ValueString() == ""
	case types.List:
		return len(v.Elements()) == 0
	}
	return false
}

func NewHubSpotResource() resource.Resource {
	return connector.NewBaseConnectorResource(&apiSourceConfig{
		code: "hubspot", schema: generated.SourceHubspotSchema,
		mappings:     generated.SourceHubspotFieldMappings,
		model:        func() any { return &generated.SourceHubspotModel{} },
		requirements: generated.SourceHubspotAPIRequirements,
		oauth:        generated.SourceHubspotAPIOAuth,
	})
}

func NewSalesforceResource() resource.Resource {
	return connector.NewBaseConnectorResource(&apiSourceConfig{
		code: "salesforce", schema: generated.SourceSalesforceSchema,
		mappings:     generated.SourceSalesforceFieldMappings,
		model:        func() any { return &generated.SourceSalesforceModel{} },
		requirements: generated.SourceSalesforceAPIRequirements,
		oauth:        generated.SourceSalesforceAPIOAuth,
	})
}

func NewNetSuiteResource() resource.Resource {
	return connector.NewBaseConnectorResource(&apiSourceConfig{
		code: "netsuite", schema: generated.SourceNetsuiteSchema,
		mappings:     generated.SourceNetsuiteFieldMappings,
		model:        func() any { return &generated.SourceNetsuiteModel{} },
		requirements: generated.SourceNetsuiteAPIRequirements,
		oauth:        generated.SourceNetsuiteAPIOAuth,
	})
}
