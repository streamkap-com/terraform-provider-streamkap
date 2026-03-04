package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClientCredentialResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckClientCredentialDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
data "streamkap_roles" "all" {}

resource "streamkap_client_credential" "test" {
	role_ids    = [data.streamkap_roles.all.roles[0].id]
	description = "tf-acc-test-client-credential"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("streamkap_client_credential.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_client_credential.test", "client_id"),
					resource.TestCheckResourceAttrSet("streamkap_client_credential.test", "secret"),
					resource.TestCheckResourceAttrSet("streamkap_client_credential.test", "created_at"),
					resource.TestCheckResourceAttr("streamkap_client_credential.test", "description", "tf-acc-test-client-credential"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "streamkap_client_credential.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
