package source

import (
	"slices"

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
	generated.APISource
}

var _ connector.ConnectorConfig = (*apiSourceConfig)(nil)
var _ connector.ConnectorConfigValidator = (*apiSourceConfig)(nil)
var _ connector.ConnectorConfigRequestFilter = (*apiSourceConfig)(nil)
var _ connector.ConnectorConfigPlanAdjuster = (*apiSourceConfig)(nil)
var _ connector.ConnectorConfigKeepsConfiguredForm = (*apiSourceConfig)(nil)

func (c *apiSourceConfig) GetSchema() schema.Schema            { return c.Schema() }
func (c *apiSourceConfig) GetFieldMappings() map[string]string { return c.FieldMappings }
func (c *apiSourceConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}
func (c *apiSourceConfig) GetConnectorCode() string { return c.Code }
func (c *apiSourceConfig) GetResourceName() string  { return "source_" + c.Code }
func (c *apiSourceConfig) NewModelInstance() any    { return c.NewModel() }

// KeepsConfiguredForm opts API sources into keeping the configured spelling of
// a value the backend normalizes (see connector.ConnectorConfigKeepsConfiguredForm).
func (c *apiSourceConfig) KeepsConfiguredForm() {}

// conditionValues captures every condition field, and the fields they gate.
func (c *apiSourceConfig) conditionValues(model any, extra ...string) map[string]attr.Value {
	names := make([]string, 0, len(c.Conditions)*2+len(extra))
	for _, condition := range c.Conditions {
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

func (c *apiSourceConfig) ValidateConfiguration(model any, prior any) diag.Diagnostics {
	var diags diag.Diagnostics
	var extra []string
	if c.OAuth.Enabled {
		extra = append(extra, oauthGrantAttr)
		if c.OAuth.AuthModeField != "" {
			extra = append(extra, c.OAuth.AuthModeField)
		}
		if c.OAuth.HostField != "" {
			extra = append(extra, c.OAuth.HostField)
		}
	}
	for _, dependency := range c.Dependencies {
		extra = append(extra, dependency.Field)
		extra = append(extra, dependency.Requires...)
	}
	values := c.conditionValues(model, extra...)
	if prior == nil && c.OAuth.Enabled {
		diags.Append(c.validateOAuthCreate(values)...)
	}
	if prior != nil && c.OAuth.HostField != "" {
		diags.Append(c.validateHostChange(values, shared.CaptureFields(prior, []string{c.OAuth.HostField, oauthGrantAttr}))...)
	}
	secrets := shared.SensitiveStringAttrNames(c.Schema())
	for _, condition := range c.Conditions {
		show, conditionValue, known := shown(condition, values)
		if !known {
			continue
		}
		value := values[condition.Field]
		if show && condition.Required && isUnset(value) {
			diags.AddAttributeError(path.Root(condition.Field), "Missing API source value", condition.Field+" is required when "+condition.ConditionField+" is "+conditionValue+".")
		}
		// The backend refuses a secret typed for another auth mode. In OAuth
		// mode the grant may use secrets the form hides, so only the pasted
		// modes, where the form shows exactly the secrets they sign in with,
		// are checked.
		if !show && slices.Contains(secrets, condition.Field) && condition.ConditionField == c.OAuth.AuthModeField &&
			conditionValue != "" && conditionValue != c.OAuth.AuthModeValue && isSet(value) {
			diags.AddAttributeError(path.Root(condition.Field), "API source value for another auth mode", condition.Field+" is not used when "+condition.ConditionField+" is "+conditionValue+"; remove it, or switch "+condition.ConditionField+".")
		}
	}
	for _, dependency := range c.Dependencies {
		if !isSet(values[dependency.Field]) {
			continue
		}
		for _, name := range dependency.Requires {
			if isUnset(values[name]) {
				diags.AddAttributeError(path.Root(name), "Missing API source value", name+" is required when "+dependency.Field+" is set.")
			}
		}
	}
	return diags
}

// isSet and isUnset are false for a value unknown at plan time.
func isSet(value attr.Value) bool {
	return value != nil && !value.IsUnknown() && !value.IsNull() && !emptyValue(value)
}

func isUnset(value attr.Value) bool {
	return value != nil && !value.IsUnknown() && (value.IsNull() || emptyValue(value))
}

// PlanAdjustments plans form-gated fields the configuration leaves unset.
// They are Computed so that values an OAuth grant filled in survive, but a
// field the user removed must otherwise be planned null. The backend drops a
// previous auth mode's secrets itself; without the null, state would keep the
// stale value, since Read refills a secret the API echoes as null. In OAuth
// mode the form does not say which hidden fields the grant fills, so they are
// kept while the grant is unchanged, and planned unknown when a new grant is
// sent, so state takes whatever the backend then holds.
func (c *apiSourceConfig) PlanAdjustments(config any, state any) (unset []string, unknown []string) {
	values := c.conditionValues(config, oauthGrantAttr)
	newGrant := false
	if grant := values[oauthGrantAttr]; grant != nil && !grant.IsNull() {
		newGrant = state == nil || grant.IsUnknown() || !grant.Equal(shared.CaptureFields(state, []string{oauthGrantAttr})[oauthGrantAttr])
	}
	for _, condition := range c.Conditions {
		_, value, known := shown(condition, values)
		field := values[condition.Field]
		if !known || field == nil || !field.IsNull() {
			continue
		}
		if c.OAuth.Enabled && condition.ConditionField == c.OAuth.AuthModeField && value == c.OAuth.AuthModeValue {
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
	if c.OAuth.AuthModeField != "" {
		mode := values[c.OAuth.AuthModeField]
		if mode != nil && mode.IsUnknown() {
			return diags
		}
		value := conditionString(mode)
		if value == "" {
			value = c.OAuth.AuthModeDefault
		}
		if value != c.OAuth.AuthModeValue {
			return diags
		}
	}
	grant := values[oauthGrantAttr]
	if grant != nil && (grant.IsUnknown() || (!grant.IsNull() && !emptyValue(grant))) {
		return diags
	}
	summary := c.DisplayName + " sign-in needs an OAuth grant"
	detail := "Terraform cannot complete the browser consent of Connect with " + c.DisplayName + ". "
	if c.OAuth.AuthModeField == "" {
		detail = c.DisplayName + " has no pasted-credential mode: it signs in only through Connect with " + c.DisplayName + ", which Terraform cannot complete. "
	} else {
		detail += "Choose another " + c.OAuth.AuthModeField + ", or "
	}
	detail += "Run Connect in the Streamkap UI, or run " + c.connectSteps() + " " +
		"Set oauth_grant_id to the returned grant and apply within 10 minutes. Alternatively, create the source in the Streamkap UI and import it."
	diags.AddAttributeError(path.Root(oauthGrantAttr), summary, detail)
	return diags
}

// validateHostChange refuses a new host (the Zendesk subdomain) without a new
// grant: the stored grant was authorized for the old account, and the backend
// rejects the update.
func (c *apiSourceConfig) validateHostChange(values, prior map[string]attr.Value) diag.Diagnostics {
	var diags diag.Diagnostics
	host, before := values[c.OAuth.HostField], prior[c.OAuth.HostField]
	if !isSet(host) || !isSet(before) || host.Equal(before) {
		return diags
	}
	if grant := values[oauthGrantAttr]; grant != nil && (grant.IsUnknown() || (isSet(grant) && !grant.Equal(prior[oauthGrantAttr]))) {
		return diags
	}
	diags.AddAttributeError(path.Root(c.OAuth.HostField), "Changing "+c.OAuth.HostField+" needs a new OAuth grant",
		"The current grant was authorized for the previous "+c.OAuth.HostField+". Run Connect with "+c.DisplayName+" for the new one in the Streamkap UI, or run "+
			c.connectSteps()+" Set oauth_grant_id to the new grant in the same apply.")
	return diags
}

// connectSteps is the CLI handoff for an OAuth grant.
func (c *apiSourceConfig) connectSteps() string {
	return c.OAuth.ConnectCommand + ", open the returned authorize_url, then run `streamkap sources poll-source-oauth-grant " + c.Code + " --state <state>`."
}

// FilterRequestConfig shapes the config body to the API-source contract. A
// create omits unset fields. An update resends every secret the current
// auth mode shows, so a changed location (Salesforce domain, NetSuite
// account_id) never pairs with a kept secret, which the backend refuses. It
// omits an unchanged secret the form hides, which the grant filled and the
// backend may have rotated since, and an unchanged oauth_grant_id, which is
// spent. An explicit null still clears a secret the configuration removed.
func (c *apiSourceConfig) FilterRequestConfig(configMap map[string]any, plan any, state any) {
	if state == nil {
		for key, value := range configMap {
			if value == nil {
				delete(configMap, key)
			}
		}
		return
	}
	secrets := shared.SensitiveStringAttrNames(c.Schema())
	planned := shared.CaptureFields(plan, secrets)
	prior := shared.CaptureFields(state, secrets)
	values := c.conditionValues(plan)
	for _, name := range secrets {
		apiField, ok := c.FieldMappings[name]
		if !ok {
			continue
		}
		if name != oauthGrantAttr && c.showsField(name, values) {
			continue
		}
		before, after := prior[name], planned[name]
		if before == nil || after == nil || before.Equal(after) || (name == oauthGrantAttr && !isSet(after)) {
			delete(configMap, apiField)
		}
	}
}

// showsField reports whether the form shows a field for the planned values: a
// field no condition gates is always shown.
func (c *apiSourceConfig) showsField(name string, values map[string]attr.Value) bool {
	for _, condition := range c.Conditions {
		if condition.Field == name {
			show, _, known := shown(condition, values)
			return show && known
		}
	}
	return true
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

func newAPISource(source generated.APISource) resource.Resource {
	return connector.NewBaseConnectorResource(&apiSourceConfig{source})
}

func NewHubSpotResource() resource.Resource { return newAPISource(generated.SourceHubspotAPISource) }
func NewSalesforceResource() resource.Resource {
	return newAPISource(generated.SourceSalesforceAPISource)
}
func NewNetSuiteResource() resource.Resource { return newAPISource(generated.SourceNetsuiteAPISource) }
func NewStripeResource() resource.Resource   { return newAPISource(generated.SourceStripeAPISource) }
func NewZendeskResource() resource.Resource  { return newAPISource(generated.SourceZendeskAPISource) }
func NewGoogleAnalyticsResource() resource.Resource {
	return newAPISource(generated.SourceGoogleAnalyticsAPISource)
}
func NewFacebookAdsResource() resource.Resource {
	return newAPISource(generated.SourceFacebookAdsAPISource)
}
func NewGoogleAdsResource() resource.Resource {
	return newAPISource(generated.SourceGoogleAdsAPISource)
}
