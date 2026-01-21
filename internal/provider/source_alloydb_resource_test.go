package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceAlloyDBHostname = os.Getenv("TF_VAR_source_alloydb_hostname")
var sourceAlloyDBPassword = os.Getenv("TF_VAR_source_alloydb_password")

func TestAccSourceAlloyDBResource(t *testing.T) {
	if sourceAlloyDBHostname == "" || sourceAlloyDBPassword == "" {
		t.Skip("Skipping TestAccSourceAlloyDBResource: TF_VAR_source_alloydb_hostname or TF_VAR_source_alloydb_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_alloydb_hostname" {
	type        = string
	description = "The hostname of the AlloyDB database"
}
variable "source_alloydb_password" {
	type        = string
	sensitive   = true
	description = "The password of the AlloyDB database"
}
resource "streamkap_source_alloydb" "test" {
	name                                         = "tf-acc-test-source-alloydb"
	database_hostname                            = var.source_alloydb_hostname
	database_port                                = 5432
	database_user                                = "alloydb"
	database_password                            = var.source_alloydb_password
	database_dbname                              = "postgres"
	snapshot_read_only                           = "Yes"
	database_sslmode                             = "require"
	schema_include_list                          = "streamkap"
	table_include_list                           = "streamkap.customer"
	signal_data_collection_schema_or_database    = "streamkap"
	heartbeat_enabled                            = true
	heartbeat_data_collection_schema_or_database = "streamkap"
	slot_name                                    = "streamkap_pgoutput_slot"
	publication_name                             = "streamkap_pub"
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "name", "tf-acc-test-source-alloydb"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "database_hostname", sourceAlloyDBHostname),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "database_port", "5432"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "database_user", "alloydb"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "database_password", sourceAlloyDBPassword),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "database_dbname", "postgres"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "snapshot_read_only", "Yes"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "database_sslmode", "require"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "schema_include_list", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "table_include_list", "streamkap.customer"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "signal_data_collection_schema_or_database", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "heartbeat_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "heartbeat_data_collection_schema_or_database", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "slot_name", "streamkap_pgoutput_slot"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "publication_name", "streamkap_pub"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "ssh_enabled", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_alloydb.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_alloydb_hostname" {
	type        = string
	description = "The hostname of the AlloyDB database"
}
variable "source_alloydb_password" {
	type        = string
	sensitive   = true
	description = "The password of the AlloyDB database"
}
resource "streamkap_source_alloydb" "test" {
	name                                         = "tf-acc-test-source-alloydb-updated"
	database_hostname                            = var.source_alloydb_hostname
	database_port                                = 5432
	database_user                                = "alloydb"
	database_password                            = var.source_alloydb_password
	database_dbname                              = "postgres"
	snapshot_read_only                           = "No"
	database_sslmode                             = "require"
	schema_include_list                          = "streamkap"
	table_include_list                           = "streamkap.customer,streamkap.orders"
	signal_data_collection_schema_or_database    = "streamkap"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = "streamkap"
	slot_name                                    = "streamkap_pgoutput_slot"
	publication_name                             = "streamkap_pub"
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "name", "tf-acc-test-source-alloydb-updated"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "snapshot_read_only", "No"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "table_include_list", "streamkap.customer,streamkap.orders"),
					resource.TestCheckResourceAttr("streamkap_source_alloydb.test", "heartbeat_enabled", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
