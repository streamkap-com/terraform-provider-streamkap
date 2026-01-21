package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceVitessHostname = os.Getenv("TF_VAR_source_vitess_hostname")
var sourceVitessVtctldHost = os.Getenv("TF_VAR_source_vitess_vtctld_host")
var sourceVitessVtctldPassword = os.Getenv("TF_VAR_source_vitess_vtctld_password")

func TestAccSourceVitessResource(t *testing.T) {
	if sourceVitessHostname == "" || sourceVitessVtctldHost == "" || sourceVitessVtctldPassword == "" {
		t.Skip("Skipping TestAccSourceVitessResource: TF_VAR_source_vitess_hostname, TF_VAR_source_vitess_vtctld_host, or TF_VAR_source_vitess_vtctld_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_vitess_hostname" {
	type        = string
	description = "The hostname of the Vitess database server (VTGate)"
}
variable "source_vitess_vtctld_host" {
	type        = string
	description = "The hostname of the VTCtld server"
}
variable "source_vitess_vtctld_password" {
	type        = string
	sensitive   = true
	description = "The password of the VTCtld server"
}
resource "streamkap_source_vitess" "test" {
	name                   = "tf-acc-test-source-vitess"
	database_hostname      = var.source_vitess_hostname
	database_port          = 15991
	vitess_keyspace        = "streamkap"
	vitess_tablet_type     = "MASTER"
	vitess_vtctld_host     = var.source_vitess_vtctld_host
	vitess_vtctld_port     = 15999
	vitess_vtctld_user     = "streamkap"
	vitess_vtctld_password = var.source_vitess_vtctld_password
	table_include_list     = "streamkap.customer"
	ssh_enabled            = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_vitess.test", "name", "tf-acc-test-source-vitess"),
					resource.TestCheckResourceAttr("streamkap_source_vitess.test", "database_hostname", sourceVitessHostname),
					resource.TestCheckResourceAttr("streamkap_source_vitess.test", "database_port", "15991"),
					resource.TestCheckResourceAttr("streamkap_source_vitess.test", "vitess_keyspace", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_vitess.test", "vitess_tablet_type", "MASTER"),
					resource.TestCheckResourceAttr("streamkap_source_vitess.test", "vitess_vtctld_host", sourceVitessVtctldHost),
					resource.TestCheckResourceAttr("streamkap_source_vitess.test", "vitess_vtctld_port", "15999"),
					resource.TestCheckResourceAttr("streamkap_source_vitess.test", "vitess_vtctld_user", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_vitess.test", "vitess_vtctld_password", sourceVitessVtctldPassword),
					resource.TestCheckResourceAttr("streamkap_source_vitess.test", "table_include_list", "streamkap.customer"),
					resource.TestCheckResourceAttr("streamkap_source_vitess.test", "ssh_enabled", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_vitess.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_vitess_hostname" {
	type        = string
	description = "The hostname of the Vitess database server (VTGate)"
}
variable "source_vitess_vtctld_host" {
	type        = string
	description = "The hostname of the VTCtld server"
}
variable "source_vitess_vtctld_password" {
	type        = string
	sensitive   = true
	description = "The password of the VTCtld server"
}
resource "streamkap_source_vitess" "test" {
	name                   = "tf-acc-test-source-vitess-updated"
	database_hostname      = var.source_vitess_hostname
	database_port          = 15991
	vitess_keyspace        = "streamkap"
	vitess_tablet_type     = "REPLICA"
	vitess_vtctld_host     = var.source_vitess_vtctld_host
	vitess_vtctld_port     = 15999
	vitess_vtctld_user     = "streamkap"
	vitess_vtctld_password = var.source_vitess_vtctld_password
	table_include_list     = "streamkap.customer,streamkap.orders"
	ssh_enabled            = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_vitess.test", "name", "tf-acc-test-source-vitess-updated"),
					resource.TestCheckResourceAttr("streamkap_source_vitess.test", "vitess_tablet_type", "REPLICA"),
					resource.TestCheckResourceAttr("streamkap_source_vitess.test", "table_include_list", "streamkap.customer,streamkap.orders"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
