package source

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	if diags := config.ValidateConfiguration(salesforce, true); !diags.HasError() {
		t.Fatal("Terraform create cannot complete interactive Salesforce OAuth")
	}
	if diags := config.ValidateConfiguration(salesforce, false); diags.HasError() {
		t.Fatalf("imported Salesforce OAuth source must remain manageable: %v", diags)
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
