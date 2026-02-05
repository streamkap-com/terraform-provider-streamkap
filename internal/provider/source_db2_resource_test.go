package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceDB2Hostname = os.Getenv("TF_VAR_source_db2_hostname")
var sourceDB2Password = os.Getenv("TF_VAR_source_db2_password")

func TestAccSourceDB2Resource(t *testing.T) {
	if sourceDB2Hostname == "" || sourceDB2Password == "" {
		t.Skip("Skipping TestAccSourceDB2Resource: TF_VAR_source_db2_hostname or TF_VAR_source_db2_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_db2_hostname" {
	type        = string
	description = "The hostname of the DB2 database"
}
variable "source_db2_password" {
	type        = string
	sensitive   = true
	description = "The password of the DB2 database"
}
resource "streamkap_source_db2" "test" {
	name                                     = "tf-acc-test-source-db2"
	database_hostname                        = var.source_db2_hostname
	database_port                            = 50000
	database_user                            = "db2admin"
	database_password                        = var.source_db2_password
	database_dbname                          = "testdb"
	schema_include_list                      = "STREAMKAP"
	table_include_list                       = "STREAMKAP.CUSTOMER"
	signal_data_collection_schema_or_database = "STREAMKAP"
	ssh_enabled                              = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_db2.test", "name", "tf-acc-test-source-db2"),
					resource.TestCheckResourceAttr("streamkap_source_db2.test", "database_hostname", sourceDB2Hostname),
					resource.TestCheckResourceAttr("streamkap_source_db2.test", "database_port", "50000"),
					resource.TestCheckResourceAttr("streamkap_source_db2.test", "database_user", "db2admin"),
					resource.TestCheckResourceAttr("streamkap_source_db2.test", "database_password", sourceDB2Password),
					resource.TestCheckResourceAttr("streamkap_source_db2.test", "database_dbname", "testdb"),
					resource.TestCheckResourceAttr("streamkap_source_db2.test", "schema_include_list", "STREAMKAP"),
					resource.TestCheckResourceAttr("streamkap_source_db2.test", "table_include_list", "STREAMKAP.CUSTOMER"),
					resource.TestCheckResourceAttr("streamkap_source_db2.test", "signal_data_collection_schema_or_database", "STREAMKAP"),
					resource.TestCheckResourceAttr("streamkap_source_db2.test", "ssh_enabled", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_db2.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_db2_hostname" {
	type        = string
	description = "The hostname of the DB2 database"
}
variable "source_db2_password" {
	type        = string
	sensitive   = true
	description = "The password of the DB2 database"
}
resource "streamkap_source_db2" "test" {
	name                                     = "tf-acc-test-source-db2-updated"
	database_hostname                        = var.source_db2_hostname
	database_port                            = 50000
	database_user                            = "db2admin"
	database_password                        = var.source_db2_password
	database_dbname                          = "testdb"
	schema_include_list                      = "STREAMKAP"
	table_include_list                       = "STREAMKAP.CUSTOMER,STREAMKAP.ORDERS"
	signal_data_collection_schema_or_database = "STREAMKAP"
	ssh_enabled                              = false
	schema_history_internal_store_only_captured_databases_ddl = true
	schema_history_internal_store_only_captured_tables_ddl    = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_db2.test", "name", "tf-acc-test-source-db2-updated"),
					resource.TestCheckResourceAttr("streamkap_source_db2.test", "table_include_list", "STREAMKAP.CUSTOMER,STREAMKAP.ORDERS"),
					resource.TestCheckResourceAttr("streamkap_source_db2.test", "schema_history_internal_store_only_captured_databases_ddl", "true"),
					resource.TestCheckResourceAttr("streamkap_source_db2.test", "schema_history_internal_store_only_captured_tables_ddl", "true"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
