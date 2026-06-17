package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceInformixHostname = os.Getenv("TF_VAR_source_informix_hostname")
var sourceInformixPassword = os.Getenv("TF_VAR_source_informix_password")

func TestAccSourceInformixResource(t *testing.T) {
	if sourceInformixHostname == "" || sourceInformixPassword == "" {
		t.Skip("Skipping TestAccSourceInformixResource: TF_VAR_source_informix_hostname or TF_VAR_source_informix_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_informix_hostname" {
	type        = string
	description = "The hostname of the Informix database"
}
variable "source_informix_password" {
	type        = string
	sensitive   = true
	description = "The password of the Informix database"
}
resource "streamkap_source_informix" "test" {
	name                                      = "tf-acc-test-source-informix"
	database_hostname                         = var.source_informix_hostname
	database_port                             = 9088
	database_user                             = "streamkap_user"
	database_password                         = var.source_informix_password
	database_dbname                           = "stores_demo"
	schema_include_list                       = "informix"
	table_include_list                        = "informix.orders"
	signal_data_collection_schema_or_database = "streamkap"
	ssh_enabled                               = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_informix.test", "name", "tf-acc-test-source-informix"),
					resource.TestCheckResourceAttr("streamkap_source_informix.test", "database_hostname", sourceInformixHostname),
					resource.TestCheckResourceAttr("streamkap_source_informix.test", "database_port", "9088"),
					resource.TestCheckResourceAttr("streamkap_source_informix.test", "database_user", "streamkap_user"),
					resource.TestCheckResourceAttr("streamkap_source_informix.test", "database_password", sourceInformixPassword),
					resource.TestCheckResourceAttr("streamkap_source_informix.test", "database_dbname", "stores_demo"),
					resource.TestCheckResourceAttr("streamkap_source_informix.test", "schema_include_list", "informix"),
					resource.TestCheckResourceAttr("streamkap_source_informix.test", "table_include_list", "informix.orders"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "streamkap_source_informix.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_informix_hostname" {
	type        = string
	description = "The hostname of the Informix database"
}
variable "source_informix_password" {
	type        = string
	sensitive   = true
	description = "The password of the Informix database"
}
resource "streamkap_source_informix" "test" {
	name                                      = "tf-acc-test-source-informix-updated"
	database_hostname                         = var.source_informix_hostname
	database_port                             = 9088
	database_user                             = "streamkap_user"
	database_password                         = var.source_informix_password
	database_dbname                           = "stores_demo"
	schema_include_list                       = "informix"
	table_include_list                        = "informix.orders,informix.customer"
	signal_data_collection_schema_or_database = "streamkap"
	ssh_enabled                               = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_informix.test", "name", "tf-acc-test-source-informix-updated"),
					resource.TestCheckResourceAttr("streamkap_source_informix.test", "table_include_list", "informix.orders,informix.customer"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
