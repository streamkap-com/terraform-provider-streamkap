package source

import (
	"context"
	"slices"
	"strings"
	"testing"

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

func TestAPISourceSchemas(t *testing.T) {
	for _, test := range []struct {
		name string
		new  func() connector.ConnectorConfig
	}{
		{"hubspot", func() connector.ConnectorConfig {
			return NewHubSpotResource().(*connector.BaseConnectorResource).Config()
		}},
		{"salesforce", func() connector.ConnectorConfig {
			return NewSalesforceResource().(*connector.BaseConnectorResource).Config()
		}},
		{"netsuite", func() connector.ConnectorConfig {
			return NewNetSuiteResource().(*connector.BaseConnectorResource).Config()
		}},
		{"stripe", func() connector.ConnectorConfig {
			return NewStripeResource().(*connector.BaseConnectorResource).Config()
		}},
		{"zendesk", func() connector.ConnectorConfig {
			return NewZendeskResource().(*connector.BaseConnectorResource).Config()
		}},
		{"google_analytics", func() connector.ConnectorConfig {
			return NewGoogleAnalyticsResource().(*connector.BaseConnectorResource).Config()
		}},
		{"facebook_ads", func() connector.ConnectorConfig {
			return NewFacebookAdsResource().(*connector.BaseConnectorResource).Config()
		}},
		{"google_ads", func() connector.ConnectorConfig {
			return NewGoogleAdsResource().(*connector.BaseConnectorResource).Config()
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := test.new().GetSchema()
			if _, ok := s.Attributes["database_hostname"]; ok {
				t.Fatal("API source schema contains CDC fields")
			}
			cluster := s.Attributes["kc_cluster_id"].(schema.StringAttribute)
			if !cluster.Computed || cluster.Optional || cluster.Required {
				t.Fatal("API source cluster must be server selected")
			}
			resources := s.Attributes["resources"].(schema.ListAttribute)
			if !resources.Required || len(resources.Validators) != 1 {
				t.Fatal("resources must be required and validate nonempty selection")
			}
			validateResources := func(items []attr.Value) bool {
				var response validator.ListResponse
				resources.Validators[0].ValidateList(context.Background(), validator.ListRequest{
					Path: path.Root("resources"), ConfigValue: types.ListValueMust(types.StringType, items),
				}, &response)
				return response.Diagnostics.HasError()
			}
			if !validateResources(nil) || validateResources([]attr.Value{types.StringValue("custom_object")}) {
				t.Fatal("resources validator must reject an empty list and allow custom vendor objects")
			}
		})
	}
	s := NewSalesforceResource().(*connector.BaseConnectorResource).Config().GetSchema()
	for _, field := range []string{"refresh_token", "org_id", "environment"} {
		if _, ok := s.Attributes[field]; ok {
			t.Errorf("system-managed Salesforce field %s is configurable", field)
		}
	}
	if s.Attributes["client_secret"].(schema.StringAttribute).Required {
		t.Fatal("Salesforce service credential must be conditional on auth_mode")
	}
	if !s.Attributes["client_secret"].(schema.StringAttribute).Sensitive {
		t.Fatal("Salesforce secret must remain sensitive")
	}
}

func TestAPISourceCredentialModes(t *testing.T) {
	salesforce := &generated.SourceSalesforceModel{AuthMode: types.StringValue("service")}
	config := NewSalesforceResource().(*connector.BaseConnectorResource).Config().(*apiSourceConfig)
	if diags := config.ValidateConfiguration(salesforce, true); !diags.HasError() {
		t.Fatal("Salesforce service mode accepted missing credentials")
	}
	salesforce.AuthMode = types.StringNull()
	if diags := config.ValidateConfiguration(salesforce, true); !diags.HasError() {
		t.Fatal("Salesforce default service mode accepted missing credentials")
	}
	salesforce.AuthMode = types.StringValue("service")
	salesforce.Domain = types.StringValue("https://example.my.salesforce.com")
	salesforce.ClientID = types.StringValue("id")
	salesforce.ClientSecret = types.StringValue("secret")
	if diags := config.ValidateConfiguration(salesforce, true); diags.HasError() {
		t.Fatalf("Salesforce service mode rejected complete credentials: %v", diags)
	}
	salesforce.AuthMode = types.StringValue("oauth")
	salesforce.Domain = types.StringNull()
	salesforce.ClientID = types.StringNull()
	salesforce.ClientSecret = types.StringNull()
	if diags := config.ValidateConfiguration(salesforce, true); !diags.HasError() {
		t.Fatal("Terraform create cannot complete interactive Salesforce OAuth without a grant")
	}
	if diags := config.ValidateConfiguration(salesforce, false); diags.HasError() {
		t.Fatalf("imported Salesforce OAuth source must remain manageable: %v", diags)
	}
	salesforce.OauthGrantID = types.StringValue("grant")
	if diags := config.ValidateConfiguration(salesforce, true); diags.HasError() {
		t.Fatalf("Salesforce OAuth create with a grant must not need pasted credentials: %v", diags)
	}
	salesforce.AuthMode = types.StringValue("jwt")
	salesforce.OauthGrantID = types.StringNull()
	salesforce.Domain = types.StringValue("https://example.my.salesforce.com")
	salesforce.ClientID = types.StringValue("id")
	if diags := config.ValidateConfiguration(salesforce, true); !diags.HasError() {
		t.Fatal("Salesforce JWT mode accepted a missing username and private key")
	}
	salesforce.Username = types.StringValue("integration@example.com")
	salesforce.PrivateKey = types.StringValue("key")
	if diags := config.ValidateConfiguration(salesforce, true); diags.HasError() {
		t.Fatalf("Salesforce JWT mode must not need the client secret: %v", diags)
	}

	netsuite := &generated.SourceNetsuiteModel{AuthMode: types.StringValue("certificate")}
	config = NewNetSuiteResource().(*connector.BaseConnectorResource).Config().(*apiSourceConfig)
	if diags := config.ValidateConfiguration(netsuite, true); !diags.HasError() {
		t.Fatal("NetSuite certificate mode accepted missing credentials")
	}
	netsuite.ClientID = types.StringValue("id")
	netsuite.CertificateID = types.StringValue("certificate")
	netsuite.PrivateKey = types.StringValue("key")
	if diags := config.ValidateConfiguration(netsuite, true); diags.HasError() {
		t.Fatalf("NetSuite certificate mode rejected complete credentials: %v", diags)
	}
	netsuite.AuthMode = types.StringValue("tba")
	netsuite.ConsumerKey = types.StringValue("key")
	netsuite.ConsumerSecret = types.StringValue("secret")
	netsuite.TokenID = types.StringValue("token")
	netsuite.TokenSecret = types.StringValue("token-secret")
	if diags := config.ValidateConfiguration(netsuite, true); diags.HasError() {
		t.Fatalf("NetSuite TBA mode required certificate credentials: %v", diags)
	}
}

func TestAPISourceListConfigRoundTrip(t *testing.T) {
	model := &generated.SourceHubspotModel{Resources: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("contacts"), types.StringValue("custom_object")})}
	config, err := shared.ModelToConfigMap(context.Background(), model, generated.SourceHubspotFieldMappings, nil)
	if err != nil {
		t.Fatal(err)
	}
	resources, ok := config["resources"].([]string)
	if !ok || len(resources) != 2 || resources[1] != "custom_object" {
		t.Fatalf("API source resources serialized as %#v", config["resources"])
	}
}

func apiSource(t *testing.T, r func() resource.Resource) *apiSourceConfig {
	t.Helper()
	return r().(*connector.BaseConnectorResource).Config().(*apiSourceConfig)
}

func TestAPISourceHubSpotModes(t *testing.T) {
	config := apiSource(t, NewHubSpotResource)
	hubspot := &generated.SourceHubspotModel{}
	if diags := config.ValidateConfiguration(hubspot, true); !diags.HasError() {
		t.Fatal("HubSpot default token mode accepted a missing token")
	}
	hubspot.Token = types.StringValue("pat-token")
	if diags := config.ValidateConfiguration(hubspot, true); diags.HasError() {
		t.Fatalf("HubSpot token mode rejected a token: %v", diags)
	}
	hubspot = &generated.SourceHubspotModel{AuthMode: types.StringValue("oauth")}
	diags := config.ValidateConfiguration(hubspot, true)
	if !diags.HasError() || !strings.Contains(diags[0].Detail(), "start-source-oauth-connect hubspot") {
		t.Fatalf("HubSpot OAuth create without a grant must explain the handoff: %v", diags)
	}
	hubspot.OauthGrantID = types.StringUnknown()
	if diags := config.ValidateConfiguration(hubspot, true); diags.HasError() {
		t.Fatalf("an unknown grant must wait for apply: %v", diags)
	}
}

func TestAPISourceZendeskNeedsGrantOnCreate(t *testing.T) {
	config := apiSource(t, NewZendeskResource)
	zendesk := &generated.SourceZendeskModel{Subdomain: types.StringValue("acme")}
	diags := config.ValidateConfiguration(zendesk, true)
	if !diags.HasError() || !strings.Contains(diags[0].Detail(), "no pasted-credential mode") {
		t.Fatalf("Zendesk create without a grant must say Connect is its only sign-in: %v", diags)
	}
	if diags := config.ValidateConfiguration(zendesk, false); diags.HasError() {
		t.Fatalf("an existing Zendesk source must update without a new grant: %v", diags)
	}
	zendesk.OauthGrantID = types.StringValue("grant")
	if diags := config.ValidateConfiguration(zendesk, true); diags.HasError() {
		t.Fatalf("Zendesk create with a grant: %v", diags)
	}
}

func TestAPISourceGoogleAdsModes(t *testing.T) {
	config := apiSource(t, NewGoogleAdsResource)
	for _, test := range []struct {
		name    string
		model   generated.SourceGoogleAdsModel
		missing []string
	}{
		{"oauth", generated.SourceGoogleAdsModel{AuthMode: types.StringValue("oauth")}, []string{"client_id", "client_secret", "oauth_grant_id"}},
		{"oauth with client and grant", generated.SourceGoogleAdsModel{AuthMode: types.StringValue("oauth"), ClientID: types.StringValue("id"), ClientSecret: types.StringValue("secret"), OauthGrantID: types.StringValue("grant")}, nil},
		{"refresh token", generated.SourceGoogleAdsModel{AuthMode: types.StringValue("refresh_token")}, []string{"client_id", "client_secret", "refresh_token"}},
		{"service account", generated.SourceGoogleAdsModel{AuthMode: types.StringValue("service_account")}, []string{"service_account_key"}},
		{"service account complete", generated.SourceGoogleAdsModel{AuthMode: types.StringValue("service_account"), ServiceAccountKey: types.StringValue("{}")}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			var missing []string
			for _, d := range config.ValidateConfiguration(&test.model, true) {
				if withPath, ok := d.(diag.DiagnosticWithPath); ok {
					missing = append(missing, withPath.Path().String())
				}
			}
			slices.Sort(missing)
			if !slices.Equal(missing, test.missing) {
				t.Errorf("missing = %v, want %v", missing, test.missing)
			}
		})
	}
}

func TestAPISourcePlansRemovedGatedFields(t *testing.T) {
	hubspot := apiSource(t, NewHubSpotResource)
	for _, test := range []struct {
		sync  types.Bool
		unset bool
	}{{types.BoolNull(), true}, {types.BoolValue(true), true}, {types.BoolValue(false), true}, {types.BoolUnknown(), false}} {
		unset, _ := hubspot.PlanAdjustments(&generated.SourceHubspotModel{SyncAllProperties: test.sync, Properties: types.ListNull(types.StringType)}, nil)
		if slices.Contains(unset, "properties") != test.unset {
			t.Errorf("sync_all_properties=%v: unset = %v", test.sync, unset)
		}
	}

	netsuite := apiSource(t, NewNetSuiteResource)
	unset, _ := netsuite.PlanAdjustments(&generated.SourceNetsuiteModel{
		AuthMode: types.StringValue("tba"), ConsumerKey: types.StringValue("key"), ConsumerSecret: types.StringValue("secret"),
		TokenID: types.StringValue("token"), TokenSecret: types.StringValue("token-secret"),
	}, &generated.SourceNetsuiteModel{})
	slices.Sort(unset)
	if !slices.Equal(unset, []string{"certificate_id", "client_id", "private_key", "private_key_passphrase"}) {
		t.Errorf("the previous auth mode's removed fields must be cleared: %v", unset)
	}

	salesforce := apiSource(t, NewSalesforceResource)
	oauthConfig := func(grant string) *generated.SourceSalesforceModel {
		return &generated.SourceSalesforceModel{AuthMode: types.StringValue("oauth"), OauthGrantID: types.StringValue(grant)}
	}
	priorService := &generated.SourceSalesforceModel{
		AuthMode: types.StringValue("service"), Domain: types.StringValue("https://acme.my.salesforce.com"),
		ClientID: types.StringValue("id"), ClientSecret: types.StringValue("secret"),
	}
	unset, unknown := salesforce.PlanAdjustments(oauthConfig("grant-1"), priorService)
	slices.Sort(unknown)
	if len(unset) != 0 || !slices.Equal(unknown, []string{"client_id", "client_secret", "domain", "private_key", "username"}) {
		t.Errorf("a new grant must replan every field hidden in OAuth mode, so the grant refills its own and the rest are cleared: unset=%v unknown=%v", unset, unknown)
	}
	priorOAuth := &generated.SourceSalesforceModel{AuthMode: types.StringValue("oauth"), OauthGrantID: types.StringValue("grant-1"), Domain: types.StringValue("https://acme.my.salesforce.com")}
	unset, unknown = salesforce.PlanAdjustments(oauthConfig("grant-1"), priorOAuth)
	if len(unset) != 0 || len(unknown) != 0 {
		t.Errorf("with the grant unchanged, the values it filled in must be kept: unset=%v unknown=%v", unset, unknown)
	}
}

func TestAPISourceRequestConfig(t *testing.T) {
	config := apiSource(t, NewGoogleAdsResource)
	create := map[string]any{"customer_id": "1234567890", "login_customer_id": nil, "oauth_grant_id": nil}
	config.FilterRequestConfig(create, &generated.SourceGoogleAdsModel{}, nil)
	if _, ok := create["login_customer_id"]; ok || len(create) != 1 {
		t.Errorf("create must omit unset fields: %v", create)
	}

	prior := &generated.SourceGoogleAdsModel{
		ClientSecret: types.StringValue("secret"), RefreshToken: types.StringValue("rotated-by-backend"),
		ServiceAccountKey: types.StringValue("key"), OauthGrantID: types.StringValue("spent"),
	}
	plan := &generated.SourceGoogleAdsModel{
		ClientSecret: types.StringValue("new-secret"), RefreshToken: types.StringValue("rotated-by-backend"),
		ServiceAccountKey: types.StringNull(), OauthGrantID: types.StringValue("spent"),
	}
	update := map[string]any{"customer_id": "1234567890", "login_customer_id": nil, "client_secret": "new-secret", "refresh_token": "rotated-by-backend", "service_account_key": nil, "oauth_grant_id": "spent"}
	config.FilterRequestConfig(update, plan, prior)
	if update["client_secret"] != "new-secret" {
		t.Error("a changed secret must be sent")
	}
	if _, ok := update["refresh_token"]; ok {
		t.Error("an unchanged secret must be omitted so the backend keeps its current value")
	}
	if _, ok := update["oauth_grant_id"]; ok {
		t.Error("a spent grant must not be resent")
	}
	if value, ok := update["service_account_key"]; !ok || value != nil {
		t.Error("a secret removed from the configuration must be cleared with an explicit null")
	}
	if value, ok := update["login_customer_id"]; !ok || value != nil {
		t.Error("an update replaces the whole config, so unset non-secret fields stay explicit")
	}
}

func TestAPISourceOAuthClientFieldsInHandoff(t *testing.T) {
	diags := apiSource(t, NewGoogleAdsResource).ValidateConfiguration(&generated.SourceGoogleAdsModel{AuthMode: types.StringValue("oauth"), ClientID: types.StringValue("id"), ClientSecret: types.StringValue("secret")}, true)
	if !diags.HasError() || !strings.Contains(diags[0].Detail(), "--body <file>` with your OAuth client's client_id and client_secret") {
		t.Fatalf("a flow on the customer's own OAuth client must say how to pass it: %v", diags)
	}
}

func diagPaths(diags diag.Diagnostics) []string {
	var paths []string
	for _, d := range diags {
		if withPath, ok := d.(diag.DiagnosticWithPath); ok {
			paths = append(paths, withPath.Path().String())
		}
	}
	slices.Sort(paths)
	return paths
}

func TestAPISourceFacebookAdsAppPair(t *testing.T) {
	config := apiSource(t, NewFacebookAdsResource)
	for _, test := range []struct {
		name    string
		model   generated.SourceFacebookAdsModel
		missing []string
	}{
		{"neither", generated.SourceFacebookAdsModel{}, nil},
		{"both", generated.SourceFacebookAdsModel{AppID: types.StringValue("42"), AppSecret: types.StringValue("secret")}, nil},
		{"id only", generated.SourceFacebookAdsModel{AppID: types.StringValue("42")}, []string{"app_secret"}},
		{"secret only", generated.SourceFacebookAdsModel{AppSecret: types.StringValue("secret")}, []string{"app_id"}},
		{"secret unknown", generated.SourceFacebookAdsModel{AppID: types.StringValue("42"), AppSecret: types.StringUnknown()}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := diagPaths(config.ValidateConfiguration(&test.model, false)); !slices.Equal(got, test.missing) {
				t.Errorf("missing = %v, want %v", got, test.missing)
			}
		})
	}
}

func TestAPISourceHubSpotPropertiesRequiredWhenNotSyncingAll(t *testing.T) {
	config := apiSource(t, NewHubSpotResource)
	model := &generated.SourceHubspotModel{Token: types.StringValue("pat"), SyncAllProperties: types.BoolValue(false), Properties: types.ListNull(types.StringType)}
	if got := diagPaths(config.ValidateConfiguration(model, true)); !slices.Equal(got, []string{"properties"}) {
		t.Errorf("sync_all_properties=false without properties: %v", got)
	}
	model.SyncAllProperties = types.BoolNull()
	if diags := config.ValidateConfiguration(model, true); diags.HasError() {
		t.Errorf("sync_all_properties defaults to true: %v", diags)
	}
}

func TestAPISourceRefusesAnotherModesSecret(t *testing.T) {
	googleAds := apiSource(t, NewGoogleAdsResource)
	model := &generated.SourceGoogleAdsModel{AuthMode: types.StringValue("service_account"), ServiceAccountKey: types.StringValue("{}"), RefreshToken: types.StringValue("stale")}
	if got := diagPaths(googleAds.ValidateConfiguration(model, false)); !slices.Equal(got, []string{"refresh_token"}) {
		t.Errorf("a refresh token typed in service_account mode: %v", got)
	}
	model = &generated.SourceGoogleAdsModel{AuthMode: types.StringValue("oauth"), ClientID: types.StringValue("id"), ClientSecret: types.StringValue("secret"), RefreshToken: types.StringValue("pasted")}
	if diags := googleAds.ValidateConfiguration(model, false); diags.HasError() {
		t.Errorf("OAuth mode must not guess which hidden secrets the grant uses: %v", diags)
	}
	salesforce := apiSource(t, NewSalesforceResource)
	sf := &generated.SourceSalesforceModel{AuthMode: types.StringValue("jwt"), Domain: types.StringValue("https://acme.my.salesforce.com"), ClientID: types.StringValue("id"),
		Username: types.StringValue("u"), PrivateKey: types.StringValue("key"), ClientSecret: types.StringValue("service-secret")}
	if got := diagPaths(salesforce.ValidateConfiguration(sf, false)); !slices.Equal(got, []string{"client_secret"}) {
		t.Errorf("a service-mode secret typed in jwt mode: %v", got)
	}
}

func TestGoogleAdsAuthModeHasNoProviderDefault(t *testing.T) {
	a := apiSource(t, NewGoogleAdsResource).GetSchema().Attributes["auth_mode"].(schema.StringAttribute)
	if !a.Required || a.Default != nil {
		t.Fatal("the served auth_mode default depends on whether the deployment provisions Connect, so Terraform must require it")
	}
	h := apiSource(t, NewHubSpotResource).GetSchema().Attributes["auth_mode"].(schema.StringAttribute)
	if h.Required || h.Default == nil {
		t.Fatal("a vendor whose default is a pasted mode keeps its default")
	}
}
