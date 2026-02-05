package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceMariaDBHostname = os.Getenv("TF_VAR_source_mariadb_hostname")
var sourceMariaDBPassword = os.Getenv("TF_VAR_source_mariadb_password")

func TestAccSourceMariaDBResource(t *testing.T) {
	if sourceMariaDBHostname == "" || sourceMariaDBPassword == "" {
		t.Skip("Skipping TestAccSourceMariaDBResource: TF_VAR_source_mariadb_hostname or TF_VAR_source_mariadb_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_mariadb_hostname" {
	type        = string
	description = "The hostname of the MariaDB database"
}
variable "source_mariadb_password" {
	type        = string
	sensitive   = true
	description = "The password of the MariaDB database"
}
resource "streamkap_source_mariadb" "test" {
	name                                         = "tf-acc-test-source-mariadb"
	database_hostname                            = var.source_mariadb_hostname
	database_port                                = 3306
	database_user                                = "streamkap"
	database_password                            = var.source_mariadb_password
	database_include_list                        = "streamkap"
	table_include_list                           = "streamkap.customer"
	heartbeat_enabled                            = true
	heartbeat_data_collection_schema_or_database = "streamkap"
	database_connection_time_zone                = "SERVER"
	snapshot_gtid                                = "Yes"
	binary_handling_mode                         = "bytes"
	database_ssl_mode                            = "required"
	ssh_enabled                                  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "name", "tf-acc-test-source-mariadb"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "database_hostname", sourceMariaDBHostname),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "database_port", "3306"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "database_user", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "database_password", sourceMariaDBPassword),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "database_include_list", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "table_include_list", "streamkap.customer"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "heartbeat_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "heartbeat_data_collection_schema_or_database", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "database_connection_time_zone", "SERVER"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "snapshot_gtid", "Yes"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "database_ssl_mode", "required"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "ssh_enabled", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_mariadb.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_mariadb_hostname" {
	type        = string
	description = "The hostname of the MariaDB database"
}
variable "source_mariadb_password" {
	type        = string
	sensitive   = true
	description = "The password of the MariaDB database"
}
resource "streamkap_source_mariadb" "test" {
	name                                         = "tf-acc-test-source-mariadb-updated"
	database_hostname                            = var.source_mariadb_hostname
	database_port                                = 3306
	database_user                                = "streamkap"
	database_password                            = var.source_mariadb_password
	database_include_list                        = "streamkap"
	table_include_list                           = "streamkap.customer,streamkap.orders"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = "streamkap"
	database_connection_time_zone                = "UTC"
	snapshot_gtid                                = "No"
	binary_handling_mode                         = "base64"
	database_ssl_mode                            = "required"
	ssh_enabled                                  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "name", "tf-acc-test-source-mariadb-updated"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "table_include_list", "streamkap.customer,streamkap.orders"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "database_connection_time_zone", "UTC"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "snapshot_gtid", "No"),
					resource.TestCheckResourceAttr("streamkap_source_mariadb.test", "binary_handling_mode", "base64"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
