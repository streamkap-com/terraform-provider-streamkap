package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceSalesforceDomain = os.Getenv("TF_VAR_source_salesforce_domain")
var sourceSalesforceClientID = os.Getenv("TF_VAR_source_salesforce_client_id")
var sourceSalesforceClientSecret = os.Getenv("TF_VAR_source_salesforce_client_secret")

func testAccSourceSalesforceConfig(name, resources string) string {
	return providerConfig + fmt.Sprintf(`
variable "source_salesforce_domain" {
	type        = string
	description = "Salesforce My Domain URL"
}
variable "source_salesforce_client_id" {
	type        = string
	description = "External Client App consumer key"
}
variable "source_salesforce_client_secret" {
	type        = string
	sensitive   = true
	description = "External Client App consumer secret"
}
resource "streamkap_source_salesforce" "test" {
	name          = %q
	auth_mode     = "service"
	domain        = var.source_salesforce_domain
	client_id     = var.source_salesforce_client_id
	client_secret = var.source_salesforce_client_secret
	resources     = %s
}
`, name, resources)
}

func TestAccSourceSalesforceResource(t *testing.T) {
	if sourceSalesforceDomain == "" || sourceSalesforceClientID == "" || sourceSalesforceClientSecret == "" {
		t.Skip("Skipping TestAccSourceSalesforceResource: TF_VAR_source_salesforce_domain, TF_VAR_source_salesforce_client_id or TF_VAR_source_salesforce_client_secret not set")
	}

	name := acctestName(t, "main")
	nameUpdated := acctestName(t, "updated")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSourceSalesforceConfig(name, `["Account"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_salesforce.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_source_salesforce.test", "connector", "salesforce"),
					resource.TestCheckResourceAttr("streamkap_source_salesforce.test", "auth_mode", "service"),
					resource.TestCheckResourceAttr("streamkap_source_salesforce.test", "resources.#", "1"),
					resource.TestCheckResourceAttrSet("streamkap_source_salesforce.test", "id"),
				),
			},
			{
				ResourceName:            "streamkap_source_salesforce.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			{
				Config: testAccSourceSalesforceConfig(nameUpdated, `["Account", "Contact"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_salesforce.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_source_salesforce.test", "resources.#", "2"),
				),
			},
		},
	})
}
