package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceHubSpotToken = os.Getenv("TF_VAR_source_hubspot_token")

func testAccSourceHubSpotConfig(name, resources string) string {
	return providerConfig + fmt.Sprintf(`
variable "source_hubspot_token" {
	type        = string
	sensitive   = true
	description = "HubSpot private app token"
}
resource "streamkap_source_hubspot" "test" {
	name      = %q
	token     = var.source_hubspot_token
	resources = %s
}
`, name, resources)
}

func TestAccSourceHubSpotResource(t *testing.T) {
	if sourceHubSpotToken == "" {
		t.Skip("Skipping TestAccSourceHubSpotResource: TF_VAR_source_hubspot_token not set")
	}

	name := acctestName(t, "main")
	nameUpdated := acctestName(t, "updated")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSourceHubSpotConfig(name, `["contacts"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_hubspot.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_source_hubspot.test", "connector", "hubspot"),
					resource.TestCheckResourceAttr("streamkap_source_hubspot.test", "resources.#", "1"),
					resource.TestCheckResourceAttr("streamkap_source_hubspot.test", "resources.0", "contacts"),
					resource.TestCheckResourceAttrSet("streamkap_source_hubspot.test", "id"),
				),
			},
			{
				ResourceName:            "streamkap_source_hubspot.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			{
				Config: testAccSourceHubSpotConfig(nameUpdated, `["contacts", "companies"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_hubspot.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_source_hubspot.test", "resources.#", "2"),
				),
			},
		},
	})
}
