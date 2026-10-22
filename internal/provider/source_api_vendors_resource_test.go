package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// apiSourceAcceptance runs create, import and a resource update for one API
// source. The vendor block holds everything but name and resources; each
// TF_VAR_<variable> it reads must be set, or the test skips.
type apiSourceAcceptance struct {
	resourceType string
	variables    map[string]bool // name -> sensitive
	vendor       string
	resources    [2]string
}

func (a apiSourceAcceptance) run(t *testing.T) {
	var declarations strings.Builder
	for name, sensitive := range a.variables {
		if os.Getenv("TF_VAR_"+name) == "" {
			t.Skipf("Skipping %s: TF_VAR_%s not set", t.Name(), name)
		}
		fmt.Fprintf(&declarations, "variable %q {\n\ttype      = string\n\tsensitive = %t\n}\n", name, sensitive)
	}
	address := a.resourceType + ".test"
	config := func(name, resources string) string {
		return providerConfig + declarations.String() + fmt.Sprintf(`
resource %q "test" {
	name      = %q
	resources = %s
%s}
`, a.resourceType, name, resources, a.vendor)
	}
	name := acctestName(t, "main")
	nameUpdated := acctestName(t, "updated")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: config(name, a.resources[0]),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "name", name),
					resource.TestCheckResourceAttrSet(address, "id"),
					resource.TestCheckResourceAttrSet(address, "kc_cluster_id"),
				),
			},
			{
				ResourceName: address, ImportState: true, ImportStateVerify: true,
				// The stored backfill_start is the normalized UTC datetime.
				ImportStateVerifyIgnore: []string{"connector_status", "backfill_start"},
			},
			{
				Config: config(nameUpdated, a.resources[1]),
				Check:  resource.TestCheckResourceAttr(address, "name", nameUpdated),
			},
		},
	})
}

func TestAccSourceStripeResource(t *testing.T) {
	apiSourceAcceptance{
		resourceType: "streamkap_source_stripe",
		variables:    map[string]bool{"source_stripe_token": true},
		vendor:       "\ttoken = var.source_stripe_token\n",
		resources:    [2]string{`["customers"]`, `["customers", "charges"]`},
	}.run(t)
}

func TestAccSourceGoogleAnalyticsResource(t *testing.T) {
	apiSourceAcceptance{
		resourceType: "streamkap_source_google_analytics",
		variables:    map[string]bool{"source_google_analytics_property_id": false, "source_google_analytics_service_account_key": true},
		vendor:       "\tproperty_id = var.source_google_analytics_property_id\n\tservice_account_key = var.source_google_analytics_service_account_key\n",
		resources:    [2]string{`["website_overview"]`, `["website_overview", "devices"]`},
	}.run(t)
}

func TestAccSourceFacebookAdsResource(t *testing.T) {
	apiSourceAcceptance{
		resourceType: "streamkap_source_facebook_ads",
		variables:    map[string]bool{"source_facebook_ads_account_id": false, "source_facebook_ads_access_token": true},
		vendor:       "\taccount_id = var.source_facebook_ads_account_id\n\taccess_token = var.source_facebook_ads_access_token\n",
		resources:    [2]string{`["campaigns"]`, `["campaigns", "ads"]`},
	}.run(t)
}

func TestAccSourceGoogleAdsResource(t *testing.T) {
	apiSourceAcceptance{
		resourceType: "streamkap_source_google_ads",
		variables:    map[string]bool{"source_google_ads_customer_id": false, "source_google_ads_service_account_key": true},
		vendor:       "\tauth_mode = \"service_account\"\n\tcustomer_id = var.source_google_ads_customer_id\n\tservice_account_key = var.source_google_ads_service_account_key\n",
		resources:    [2]string{`["campaign"]`, `["campaign", "customer"]`},
	}.run(t)
}
