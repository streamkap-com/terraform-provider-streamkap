package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceMySQLHostname = os.Getenv("TF_VAR_source_mysql_hostname")
var sourceMySQLPassword = os.Getenv("TF_VAR_source_mysql_password")
var sourceMySQLSSHHost = os.Getenv("TF_VAR_source_mysql_ssh_host")

func TestAccSourceMySQLResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create and Read testing
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
	name                                      = "test-source-mysql"
	database_hostname                         = var.source_mysql_hostname
	database_port                             = 3306
	database_user                             = "admin"
	database_password                         = var.source_mysql_password
	database_include_list                     = "crm,ecommerce,tst"
	table_include_list                        = "crm.demo,ecommerce.customers,tst.test_id_timestamp"
	signal_data_collection_schema_or_database = "crm"
	column_include_list                       = "crm[.]demo[.](id|name),ecommerce[.]customers[.](customer_id|email)"
	database_connection_timezone              = "SERVER"
	snapshot_gtid                             = true
	binary_handling_mode                      = "bytes"
	ssh_enabled                               = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "name", "test-source-mysql"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_hostname", sourceMySQLHostname),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_port", "3306"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_user", "admin"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_password", sourceMySQLPassword),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_include_list", "crm,ecommerce,tst"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "table_include_list", "crm.demo,ecommerce.customers,tst.test_id_timestamp"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "signal_data_collection_schema_or_database", "crm"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "column_include_list", "crm[.]demo[.](id|name),ecommerce[.]customers[.](customer_id|email)"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_connection_timezone", "SERVER"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "snapshot_gtid", "true"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "ssh_enabled", "false"),
					// Check defaults for unset attributes
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "heartbeat_enabled", "false"),
				),
			},
			// Step 2: ImportState testing
			{
				ResourceName:      "streamkap_source_mysql.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update and Read testing
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
	name                                      = "test-source-mysql-updated"
	database_hostname                         = var.source_mysql_hostname
	database_port                             = 3306
	database_user                             = "admin"
	database_password                         = var.source_mysql_password
	database_include_list                     = "crm"
	table_include_list                        = "crm.demo"
	signal_data_collection_schema_or_database = "crm"
	column_include_list                       = "crm[.]demo[.](id|name)"
	heartbeat_enabled                         = true
	heartbeat_data_collection_schema_or_database = "crm"
	database_connection_timezone              = "SERVER"
	snapshot_gtid                             = true
	binary_handling_mode                      = "bytes"
	ssh_enabled                               = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "name", "test-source-mysql-updated"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_include_list", "crm"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "table_include_list", "crm.demo"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "heartbeat_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "heartbeat_data_collection_schema_or_database", "crm"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "column_include_list", "crm[.]demo[.](id|name)"),
				),
			},
			// Step 4: Update to test column_exclude_list
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
	name                                      = "test-source-mysql-exclude"
	database_hostname                         = var.source_mysql_hostname
	database_port                             = 3306
	database_user                             = "admin"
	database_password                         = var.source_mysql_password
	database_include_list                     = "crm"
	table_include_list                        = "crm.demo"
	signal_data_collection_schema_or_database = "crm"
	column_exclude_list                       = "crm.demo.name"
	heartbeat_enabled                         = true
	heartbeat_data_collection_schema_or_database = "crm"
	database_connection_timezone              = "SERVER"
	snapshot_gtid                             = true
	binary_handling_mode                      = "bytes"
	ssh_enabled                               = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "name", "test-source-mysql-exclude"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "column_exclude_list", "crm.demo.name"),
					// Since column_include_list is not set, it should be null; we can skip explicit check unless needed
				),
			},
			// Step 5: Update to test insert_static_* fields
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
	name                                      = "test-source-mysql-static"
	database_hostname                         = var.source_mysql_hostname
	database_port                             = 3306
	database_user                             = "admin"
	database_password                         = var.source_mysql_password
	database_include_list                     = "crm"
	table_include_list                        = "crm.demo"
	signal_data_collection_schema_or_database = "crm"
	column_include_list                       = "crm[.]demo[.](id|name)"
	insert_static_key_field                   = "static_key"
	insert_static_key_value                   = "key_value"
	insert_static_value_field                 = "static_value"
	insert_static_value                       = "value"
	heartbeat_enabled                         = true
	heartbeat_data_collection_schema_or_database = "crm"
	database_connection_timezone              = "SERVER"
	snapshot_gtid                             = true
	binary_handling_mode                      = "bytes"
	ssh_enabled                               = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "name", "test-source-mysql-static"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "insert_static_key_field", "static_key"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "insert_static_key_value", "key_value"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "insert_static_value_field", "static_value"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "insert_static_value", "value"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "column_include_list", "crm[.]demo[.](id|name)"),
				),
			},
			// Step 6: Update to test SSH enabled
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
variable "source_mysql_ssh_host" {
	type        = string
	description = "The SSH host for the MySQL database"
}
resource "streamkap_source_mysql" "test" {
	name                                      = "test-source-mysql-ssh"
	database_hostname                         = var.source_mysql_hostname
	database_port                             = 3306
	database_user                             = "admin"
	database_password                         = var.source_mysql_password
	database_include_list                     = "crm"
	table_include_list                        = "crm.demo"
	signal_data_collection_schema_or_database = "crm"
	column_include_list                       = "crm[.]demo[.](id|name)"
	insert_static_key_field                   = "static_key"
	insert_static_key_value                   = "key_value"
	insert_static_value_field                 = "static_value"
	insert_static_value                       = "value"
	heartbeat_enabled                         = true
	heartbeat_data_collection_schema_or_database = "crm"
	database_connection_timezone              = "SERVER"
	snapshot_gtid                             = true
	binary_handling_mode                      = "bytes"
	ssh_enabled                               = true
	ssh_host                                  = var.source_mysql_ssh_host
	ssh_port                                  = "22"
	ssh_user                                  = "streamkap"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "name", "test-source-mysql-ssh"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "ssh_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "ssh_host", sourceMySQLSSHHost),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "ssh_port", "22"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "ssh_user", "streamkap"),
				),
			},
			// Delete testing is handled automatically by TestCase
		},
	})
}
