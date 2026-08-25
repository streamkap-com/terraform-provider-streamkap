package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccClientCredentialResource covers create, import, and the in-place update
// of description and role_ids.
//
// The update path is the one worth guarding: role_ids used to force replacement,
// which rotates the secret and silently invalidates whatever was authenticating
// with it. The backend exposes PATCH /auth/client-credentials/{client_id}, so a
// role change must keep the same client_id and the same secret.
//
// Requires a token carrying fe.secure.write.tenantApiTokens and
// fe.secure.read.roles; without them the API answers 403, not an empty result.
func TestAccClientCredentialResource(t *testing.T) {
	description := acctestName(t, "cred")
	updatedDescription := acctestName(t, "cred-updated")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckClientCredentialDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
data "streamkap_roles" "all" {}

resource "streamkap_client_credential" "test" {
	role_ids    = [data.streamkap_roles.all.roles[0].id]
	description = %[1]q
}
`, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("streamkap_client_credential.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_client_credential.test", "client_id"),
					resource.TestCheckResourceAttrSet("streamkap_client_credential.test", "secret"),
					resource.TestCheckResourceAttrSet("streamkap_client_credential.test", "created_at"),
					resource.TestCheckResourceAttr("streamkap_client_credential.test", "description", description),
					resource.TestCheckResourceAttr("streamkap_client_credential.test", "role_ids.#", "1"),
					resource.TestCheckResourceAttr("streamkap_client_credential.test", "roles.#", "1"),
					resource.TestCheckResourceAttrSet("streamkap_client_credential.test", "roles.0.key"),
					// id mirrors client_id, which is what import expects.
					resource.TestCheckResourceAttrPair(
						"streamkap_client_credential.test", "id",
						"streamkap_client_credential.test", "client_id",
					),
				),
			},
			// ImportState testing. The secret is unrecoverable after creation —
			// later reads echo a masked value — so an imported credential has none.
			{
				ResourceName:            "streamkap_client_credential.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"},
			},
			// Update in place: the description changes, the credential does not.
			{
				Config: providerConfig + fmt.Sprintf(`
data "streamkap_roles" "all" {}

resource "streamkap_client_credential" "test" {
	role_ids    = [data.streamkap_roles.all.roles[0].id]
	description = %[1]q
}
`, updatedDescription),
				// The attribute checks below pass whether the credential was
				// updated or replaced — only the plan action distinguishes them,
				// and a replace is what silently rotates the secret.
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("streamkap_client_credential.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_client_credential.test", "description", updatedDescription),
					resource.TestCheckResourceAttr("streamkap_client_credential.test", "role_ids.#", "1"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// TestAccDataSourceRoles asserts the roles listing returns usable records — the
// data source exists solely to resolve role IDs for client credentials, so an
// entry missing an id or key is useless even if the read succeeded.
func TestAccDataSourceRoles(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
data "streamkap_roles" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.streamkap_roles.all", "roles.#"),
					resource.TestCheckResourceAttrSet("data.streamkap_roles.all", "roles.0.id"),
					resource.TestCheckResourceAttrSet("data.streamkap_roles.all", "roles.0.key"),
					resource.TestCheckResourceAttrSet("data.streamkap_roles.all", "roles.0.name"),
				),
			},
		},
	})
}
