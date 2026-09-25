package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceNetSuiteAccountID = os.Getenv("TF_VAR_source_netsuite_account_id")
var sourceNetSuiteClientID = os.Getenv("TF_VAR_source_netsuite_client_id")
var sourceNetSuiteCertificateID = os.Getenv("TF_VAR_source_netsuite_certificate_id")
var sourceNetSuitePrivateKey = os.Getenv("TF_VAR_source_netsuite_private_key")

func testAccSourceNetSuiteConfig(name, resources string) string {
	return providerConfig + fmt.Sprintf(`
variable "source_netsuite_account_id" {
	type        = string
	description = "NetSuite account ID"
}
variable "source_netsuite_client_id" {
	type        = string
	description = "NetSuite integration client ID"
}
variable "source_netsuite_certificate_id" {
	type        = string
	description = "NetSuite OAuth 2.0 certificate ID"
}
variable "source_netsuite_private_key" {
	type        = string
	sensitive   = true
	description = "PEM private key for the NetSuite certificate"
}
resource "streamkap_source_netsuite" "test" {
	name           = %q
	account_id     = var.source_netsuite_account_id
	auth_mode      = "certificate"
	client_id      = var.source_netsuite_client_id
	certificate_id = var.source_netsuite_certificate_id
	private_key    = var.source_netsuite_private_key
	resources      = %s
}
`, name, resources)
}

func TestAccSourceNetSuiteResource(t *testing.T) {
	if sourceNetSuiteAccountID == "" || sourceNetSuiteClientID == "" || sourceNetSuiteCertificateID == "" || sourceNetSuitePrivateKey == "" {
		t.Skip("Skipping TestAccSourceNetSuiteResource: TF_VAR_source_netsuite_account_id, TF_VAR_source_netsuite_client_id, TF_VAR_source_netsuite_certificate_id or TF_VAR_source_netsuite_private_key not set")
	}

	name := acctestName(t, "main")
	nameUpdated := acctestName(t, "updated")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSourceNetSuiteConfig(name, `["customer"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_netsuite.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_source_netsuite.test", "connector", "netsuite"),
					resource.TestCheckResourceAttr("streamkap_source_netsuite.test", "auth_mode", "certificate"),
					resource.TestCheckResourceAttr("streamkap_source_netsuite.test", "resources.#", "1"),
					resource.TestCheckResourceAttrSet("streamkap_source_netsuite.test", "id"),
				),
			},
			{
				ResourceName:            "streamkap_source_netsuite.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			{
				Config: testAccSourceNetSuiteConfig(nameUpdated, `["customer", "salesOrder"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_netsuite.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_source_netsuite.test", "resources.#", "2"),
				),
			},
		},
	})
}
