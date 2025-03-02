package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceMySQLHostname = os.Getenv("TF_VAR_source_mysql_hostname")
var sourceMySQLPassword = os.Getenv("TF_VAR_source_mysql_password")

func TestAccSourceMySQLResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_mysql_hostname" {
	type        = string
	description = "The hostname of the MySQL database"
}
variable "source_mysql_password" {
	type        = string
	sensitive   = true
	description = "The password of the MySQL database"
}
resource "streamkap_source_mysql" "test" {
	name                         = "test-source-mysql"
	database_hostname            = var.source_mysql_hostname
	database_port                = 3306
	database_user                = "admin"
	database_password            = var.source_mysql_password
	database_include_list        = "crm,ecommerce,tst"
	table_include_list           = "crm.demo,ecommerce.Customers,tst.test_id_timestamp"
	heartbeat_enabled            = false
	database_connection_timezone = "SERVER"
	snapshot_gtid                = true
	binary_handling_mode         = "bytes"
	ssh_enabled                  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "name", "test-source-mysql"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_hostname", sourceMySQLHostname),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_port", "3306"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_user", "admin"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_password", sourceMySQLPassword),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_include_list", "crm,ecommerce,tst"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "table_include_list", "crm.demo,ecommerce.Customers,tst.test_id_timestamp"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_connection_timezone", "SERVER"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "snapshot_gtid", "true"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "binary_handling_mode", "bytes"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_mysql.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_mysql_hostname" {
	type        = string
	description = "The hostname of the MySQL database"
}
variable "source_mysql_password" {
	type        = string
	sensitive   = true
	description = "The password of the MySQL database"
}
resource "streamkap_source_mysql" "test" {
	name                         = "test-source-mysql-updated"
	database_hostname            = var.source_mysql_hostname
	database_port                = 3306
	database_user                = "admin"
	database_password            = var.source_mysql_password
	database_include_list        = "crm"
	table_include_list           = "crm.demo"
	heartbeat_enabled            = false
	database_connection_timezone = "SERVER"
	snapshot_gtid                = true
	binary_handling_mode         = "bytes"
	ssh_enabled                  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "name", "test-source-mysql-updated"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_hostname", sourceMySQLHostname),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_port", "3306"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_user", "admin"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_password", sourceMySQLPassword),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_include_list", "crm"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "table_include_list", "crm.demo"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_connection_timezone", "SERVER"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "snapshot_gtid", "true"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "binary_handling_mode", "bytes"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
