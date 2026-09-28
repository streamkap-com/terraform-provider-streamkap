package source

import (
	"slices"
	"strings"

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

const oauthGrantAttr = "oauth_grant_id"

type apiSourceConfig struct {
	code        string
	displayName string
	schema      func() schema.Schema
	mappings    map[string]string
	model       func() any
	conditions  []generated.APICondition
	oauth       generated.APIOAuth
}

var _ connector.ConnectorConfig = (*apiSourceConfig)(nil)
var _ connector.ConnectorConfigValidator = (*apiSourceConfig)(nil)
var _ connector.ConnectorConfigRequestFilter = (*apiSourceConfig)(nil)
var _ connector.ConnectorConfigPlanAdjuster = (*apiSourceConfig)(nil)
var _ connector.ConnectorConfigKeepsConfiguredForm = (*apiSourceConfig)(nil)

func (c *apiSourceConfig) GetSchema() schema.Schema            { return c.schema() }
func (c *apiSourceConfig) GetFieldMappings() map[string]string { return c.mappings }
func (c *apiSourceConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}
func (c *apiSourceConfig) GetConnectorCode() string { return c.code }
func (c *apiSourceConfig) GetResourceName() string  { return "source_" + c.code }
func (c *apiSourceConfig) NewModelInstance() any    { return c.model() }

// KeepsConfiguredForm opts API sources into keeping the configured spelling of
// a value the backend normalizes (see connector.ConnectorConfigKeepsConfiguredForm).
func (c *apiSourceConfig) KeepsConfiguredForm() {}

// conditionValues captures every condition field, and the fields they gate.
func (c *apiSourceConfig) conditionValues(model any, extra ...string) map[string]attr.Value {
	names := make([]string, 0, len(c.conditions)*2+len(extra))
	for _, condition := range c.conditions {
		names = append(names, condition.Field, condition.ConditionField)
	}
	return shared.CaptureFields(model, append(names, extra...))
}

// shown reports whether a condition holds; known is false while its condition
// field is still unknown at plan time.
func shown(condition generated.APICondition, values map[string]attr.Value) (show bool, value string, known bool) {
	if field := values[condition.ConditionField]; field != nil && field.IsUnknown() {
		return false, "", false
	}
	value = conditionString(values[condition.ConditionField])
	if value == "" {
		value = condition.ConditionDefault
	}
	return slices.Contains(condition.ConditionValues, value), value, true
}

func (c *apiSourceConfig) ValidateConfiguration(model any, creating bool) diag.Diagnostics {
	var diags diag.Diagnostics
	var extra []string
	if c.oauth.Enabled {
		extra = append(extra, oauthGrantAttr)
		if c.oauth.AuthModeField != "" {
			extra = append(extra, c.oauth.AuthModeField)
		}
	}
	values := c.conditionValues(model, extra...)
	if creating && c.oauth.Enabled {
		diags.Append(c.validateOAuthCreate(values)...)
	}
	for _, condition := range c.conditions {
		show, conditionValue, known := shown(condition, values)
		if !condition.Required || !known || !show {
			continue
		}
		value := values[condition.Field]
		if value != nil && !value.IsUnknown() && (value.IsNull() || emptyValue(value)) {
			diags.AddAttributeError(path.Root(condition.Field), "Missing API source value", condition.Field+" is required when "+condition.ConditionField+" is "+conditionValue+".")
		}
	}
	return diags
}

// PlanAdjustments plans form-gated fields the configuration leaves unset.
// They are Computed so that values an OAuth grant filled in survive, but a
// field the user removed must otherwise be cleared: HubSpot rejects properties
// once sync_all_properties is back on, and a secret of the previous auth mode
// would stay stored. In OAuth mode the form does not say which hidden fields
// the grant fills, so they are kept while the grant is unchanged, and planned
// unknown when a new grant is sent: the update then sends null, the grant
// refills its own fields, and the previous mode's credentials are cleared.
func (c *apiSourceConfig) PlanAdjustments(config any, state any) (unset []string, unknown []string) {
	values := c.conditionValues(config, oauthGrantAttr)
	newGrant := false
	if grant := values[oauthGrantAttr]; grant != nil && !grant.IsNull() {
		newGrant = state == nil || grant.IsUnknown() || !grant.Equal(shared.CaptureFields(state, []string{oauthGrantAttr})[oauthGrantAttr])
	}
	for _, condition := range c.conditions {
		_, value, known := shown(condition, values)
		field := values[condition.Field]
		if !known || field == nil || !field.IsNull() {
			continue
		}
		if c.oauth.Enabled && condition.ConditionField == c.oauth.AuthModeField && value == c.oauth.AuthModeValue {
			if newGrant {
				unknown = append(unknown, condition.Field)
			}
			continue
		}
		unset = append(unset, condition.Field)
	}
	return unset, unknown
}

// validateOAuthCreate requires the Connect handoff on a create in OAuth mode:
// the consent step runs in a browser, which Terraform cannot drive.
func (c *apiSourceConfig) validateOAuthCreate(values map[string]attr.Value) diag.Diagnostics {
	var diags diag.Diagnostics
	if c.oauth.AuthModeField != "" {
		mode := values[c.oauth.AuthModeField]
		if mode != nil && mode.IsUnknown() {
			return diags
		}
		value := conditionString(mode)
		if value == "" {
			value = c.oauth.AuthModeDefault
		}
		if value != c.oauth.AuthModeValue {
			return diags
		}
	}
	grant := values[oauthGrantAttr]
	if grant != nil && (grant.IsUnknown() || (!grant.IsNull() && !emptyValue(grant))) {
		return diags
	}
	summary := c.displayName + " sign-in needs an OAuth grant"
	detail := "Terraform cannot complete the browser consent of Connect with " + c.displayName + ". "
	if c.oauth.AuthModeField == "" {
		detail = c.displayName + " has no pasted-credential mode: it signs in only through Connect with " + c.displayName + ", which Terraform cannot complete. "
	} else {
		detail += "Choose another " + c.oauth.AuthModeField + ", or "
	}
	start := "`streamkap sources start-source-oauth-connect " + c.code + "`"
	if len(c.oauth.ClientFields) > 0 {
		start = "`streamkap sources start-source-oauth-connect " + c.code + " --body <file>` with your OAuth client's " + strings.Join(c.oauth.ClientFields, " and ") + " in the JSON file"
	}
	detail += "Run Connect in the Streamkap UI, or run " + start + ", open the returned authorize_url, then run `streamkap sources poll-source-oauth-grant " + c.code + " --state <state>`. " +
		"Set oauth_grant_id to the returned grant and apply before it expires. Alternatively, create the source in the Streamkap UI and import it."
	diags.AddAttributeError(path.Root(oauthGrantAttr), summary, detail)
	return diags
}

// FilterRequestConfig shapes the config body to the API-source contract. A
// create omits unset fields. An update omits every secret the plan did not
// change: the backend keeps a secret that is absent, while resending the prior
// value would overwrite a token the backend has rotated since, and a spent
// oauth_grant_id cannot be redeemed twice. An explicit null still clears a
// secret the configuration removed.
func (c *apiSourceConfig) FilterRequestConfig(configMap map[string]any, plan any, state any) {
	if state == nil {
		for key, value := range configMap {
			if value == nil {
				delete(configMap, key)
			}
		}
		return
	}
	secrets := shared.SensitiveStringAttrNames(c.schema())
	planned := shared.CaptureFields(plan, secrets)
	prior := shared.CaptureFields(state, secrets)
	for _, name := range secrets {
		apiField, ok := c.mappings[name]
		if !ok {
			continue
		}
		before, after := prior[name], planned[name]
		if before == nil || after == nil || before.Equal(after) {
			delete(configMap, apiField)
		}
	}
}

func conditionString(value attr.Value) string {
	switch v := value.(type) {
	case types.String:
		if !v.IsNull() && !v.IsUnknown() {
			return v.ValueString()
		}
	case types.Bool:
		if !v.IsNull() && !v.IsUnknown() {
			if v.ValueBool() {
				return "true"
			}
			return "false"
		}
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
		code: "hubspot", displayName: "HubSpot", schema: generated.SourceHubspotSchema,
		mappings:   generated.SourceHubspotFieldMappings,
		model:      func() any { return &generated.SourceHubspotModel{} },
		conditions: generated.SourceHubspotAPIConditions,
		oauth:      generated.SourceHubspotAPIOAuth,
	})
}

func NewSalesforceResource() resource.Resource {
	return connector.NewBaseConnectorResource(&apiSourceConfig{
		code: "salesforce", displayName: "Salesforce", schema: generated.SourceSalesforceSchema,
		mappings:   generated.SourceSalesforceFieldMappings,
		model:      func() any { return &generated.SourceSalesforceModel{} },
		conditions: generated.SourceSalesforceAPIConditions,
		oauth:      generated.SourceSalesforceAPIOAuth,
	})
}

func NewNetSuiteResource() resource.Resource {
	return connector.NewBaseConnectorResource(&apiSourceConfig{
		code: "netsuite", displayName: "NetSuite", schema: generated.SourceNetsuiteSchema,
		mappings:   generated.SourceNetsuiteFieldMappings,
		model:      func() any { return &generated.SourceNetsuiteModel{} },
		conditions: generated.SourceNetsuiteAPIConditions,
		oauth:      generated.SourceNetsuiteAPIOAuth,
	})
}

func NewStripeResource() resource.Resource {
	return connector.NewBaseConnectorResource(&apiSourceConfig{
		code: "stripe", displayName: "Stripe", schema: generated.SourceStripeSchema,
		mappings:   generated.SourceStripeFieldMappings,
		model:      func() any { return &generated.SourceStripeModel{} },
		conditions: generated.SourceStripeAPIConditions,
		oauth:      generated.SourceStripeAPIOAuth,
	})
}

func NewZendeskResource() resource.Resource {
	return connector.NewBaseConnectorResource(&apiSourceConfig{
		code: "zendesk", displayName: "Zendesk", schema: generated.SourceZendeskSchema,
		mappings:   generated.SourceZendeskFieldMappings,
		model:      func() any { return &generated.SourceZendeskModel{} },
		conditions: generated.SourceZendeskAPIConditions,
		oauth:      generated.SourceZendeskAPIOAuth,
	})
}

func NewGoogleAnalyticsResource() resource.Resource {
	return connector.NewBaseConnectorResource(&apiSourceConfig{
		code: "google_analytics", displayName: "Google Analytics 4", schema: generated.SourceGoogleAnalyticsSchema,
		mappings:   generated.SourceGoogleAnalyticsFieldMappings,
		model:      func() any { return &generated.SourceGoogleAnalyticsModel{} },
		conditions: generated.SourceGoogleAnalyticsAPIConditions,
		oauth:      generated.SourceGoogleAnalyticsAPIOAuth,
	})
}

func NewFacebookAdsResource() resource.Resource {
	return connector.NewBaseConnectorResource(&apiSourceConfig{
		code: "facebook_ads", displayName: "Facebook Ads", schema: generated.SourceFacebookAdsSchema,
		mappings:   generated.SourceFacebookAdsFieldMappings,
		model:      func() any { return &generated.SourceFacebookAdsModel{} },
		conditions: generated.SourceFacebookAdsAPIConditions,
		oauth:      generated.SourceFacebookAdsAPIOAuth,
	})
}

func NewGoogleAdsResource() resource.Resource {
	return connector.NewBaseConnectorResource(&apiSourceConfig{
		code: "google_ads", displayName: "Google Ads", schema: generated.SourceGoogleAdsSchema,
		mappings:   generated.SourceGoogleAdsFieldMappings,
		model:      func() any { return &generated.SourceGoogleAdsModel{} },
		conditions: generated.SourceGoogleAdsAPIConditions,
		oauth:      generated.SourceGoogleAdsAPIOAuth,
	})
}
