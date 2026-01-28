package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourcePlanetScaleHostname = os.Getenv("TF_VAR_source_planetscale_hostname")
var sourcePlanetScalePassword = os.Getenv("TF_VAR_source_planetscale_password")

func TestAccSourcePlanetScaleResource(t *testing.T) {
	if sourcePlanetScaleHostname == "" || sourcePlanetScalePassword == "" {
		t.Skip("Skipping TestAccSourcePlanetScaleResource: TF_VAR_source_planetscale_hostname or TF_VAR_source_planetscale_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_planetscale_hostname" {
	type        = string
	description = "The hostname of the PlanetScale database (VTGate)"
}
variable "source_planetscale_password" {
	type        = string
	sensitive   = true
	description = "The password of the PlanetScale database"
}
resource "streamkap_source_planetscale" "test" {
	name                = "tf-acc-test-source-planetscale"
	database_hostname   = var.source_planetscale_hostname
	database_port       = 443
	database_user       = "eu0akgouilvei5flomiy"
	database_password   = var.source_planetscale_password
	vitess_keyspace     = "sandbox"
	vitess_tablet_type  = "MASTER"
	table_include_list  = "sandbox.customer"
	ssh_enabled         = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_planetscale.test", "name", "tf-acc-test-source-planetscale"),
					resource.TestCheckResourceAttr("streamkap_source_planetscale.test", "database_hostname", sourcePlanetScaleHostname),
					resource.TestCheckResourceAttr("streamkap_source_planetscale.test", "database_port", "443"),
					resource.TestCheckResourceAttr("streamkap_source_planetscale.test", "database_user", "eu0akgouilvei5flomiy"),
					resource.TestCheckResourceAttr("streamkap_source_planetscale.test", "database_password", sourcePlanetScalePassword),
					resource.TestCheckResourceAttr("streamkap_source_planetscale.test", "vitess_keyspace", "sandbox"),
					resource.TestCheckResourceAttr("streamkap_source_planetscale.test", "vitess_tablet_type", "MASTER"),
					resource.TestCheckResourceAttr("streamkap_source_planetscale.test", "table_include_list", "sandbox.customer"),
					resource.TestCheckResourceAttr("streamkap_source_planetscale.test", "ssh_enabled", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_planetscale.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_planetscale_hostname" {
	type        = string
	description = "The hostname of the PlanetScale database (VTGate)"
}
variable "source_planetscale_password" {
	type        = string
	sensitive   = true
	description = "The password of the PlanetScale database"
}
resource "streamkap_source_planetscale" "test" {
	name                = "tf-acc-test-source-planetscale-updated"
	database_hostname   = var.source_planetscale_hostname
	database_port       = 443
	database_user       = "eu0akgouilvei5flomiy"
	database_password   = var.source_planetscale_password
	vitess_keyspace     = "sandbox"
	vitess_tablet_type  = "REPLICA"
	table_include_list  = "sandbox.customer,sandbox.orders"
	ssh_enabled         = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_planetscale.test", "name", "tf-acc-test-source-planetscale-updated"),
					resource.TestCheckResourceAttr("streamkap_source_planetscale.test", "vitess_tablet_type", "REPLICA"),
					resource.TestCheckResourceAttr("streamkap_source_planetscale.test", "table_include_list", "sandbox.customer,sandbox.orders"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
