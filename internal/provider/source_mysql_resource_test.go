package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceMySQLHostname = os.Getenv("TF_VAR_source_mysql_hostname")
var sourceMySQLPassword = os.Getenv("TF_VAR_source_mysql_password")

func TestAccSourceMySQLResource(t *testing.T) {
	name := acctestName(t, "main")
	nameUpdated := acctestName(t, "updated")
	nameExclude := acctestName(t, "exclude")
	nameStatic := acctestName(t, "static")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
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
	name                                      = %q
	database_hostname                         = var.source_mysql_hostname
	database_port                             = 3306
	database_user                             = "root"
	database_password                         = var.source_mysql_password
	database_include_list                     = "crm,ecommerce,tst"
	table_include_list                        = "crm.demo,ecommerce.customers,tst.test_id_timestamp"
	signal_data_collection_schema_or_database = "crm"
	column_include_list                       = "crm[.]demo[.](id|name),ecommerce[.]customers[.](customer_id|email)"
	heartbeat_data_collection_schema_or_database = "crm"
	database_connection_time_zone              = "SERVER"
	snapshot_gtid                             = "Yes"
	binary_handling_mode                      = "bytes"
	ssh_enabled                               = false
	inconsistent_schema_handling_mode         = "Warn"
	streamkap_snapshot_chunk_size_bytes       = 262144
	streamkap_snapshot_max_split_size_bytes   = 21474836480
	streamkap_snapshot_state_refresh_ms       = 15000
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_hostname", sourceMySQLHostname),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_port", "3306"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_user", "root"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_password", sourceMySQLPassword),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_include_list", "crm,ecommerce,tst"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "table_include_list", "crm.demo,ecommerce.customers,tst.test_id_timestamp"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "signal_data_collection_schema_or_database", "crm"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "column_include_list", "crm[.]demo[.](id|name),ecommerce[.]customers[.](customer_id|email)"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "heartbeat_data_collection_schema_or_database", "crm"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_connection_time_zone", "SERVER"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "snapshot_gtid", "Yes"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "ssh_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "inconsistent_schema_handling_mode", "Warn"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "streamkap_snapshot_chunk_size_bytes", "262144"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "streamkap_snapshot_max_split_size_bytes", "21474836480"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "streamkap_snapshot_state_refresh_ms", "15000"),
					// Check defaults for unset attributes
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "heartbeat_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "streamkap_snapshot_parallelism", "1"),
				),
			},
			// Step 2: ImportState testing
			{
				ResourceName:            "streamkap_source_mysql.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Step 3: Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
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
	name                                      = %q
	database_hostname                         = var.source_mysql_hostname
	database_port                             = 3306
	database_user                             = "root"
	database_password                         = var.source_mysql_password
	database_include_list                     = "crm"
	table_include_list                        = "crm.demo"
	signal_data_collection_schema_or_database = "crm"
	column_include_list                       = "crm[.]demo[.](id|name)"
	heartbeat_enabled                         = true
	heartbeat_data_collection_schema_or_database = "crm"
	database_connection_time_zone              = "SERVER"
	snapshot_gtid                             = "Yes"
	binary_handling_mode                      = "bytes"
	ssh_enabled                               = false
	inconsistent_schema_handling_mode         = "Skip"
	streamkap_snapshot_chunk_size_bytes       = 1048576
	streamkap_snapshot_max_split_size_bytes   = 107374182400
	streamkap_snapshot_state_refresh_ms       = 45000
}
`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "database_include_list", "crm"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "table_include_list", "crm.demo"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "heartbeat_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "heartbeat_data_collection_schema_or_database", "crm"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "column_include_list", "crm[.]demo[.](id|name)"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "inconsistent_schema_handling_mode", "Skip"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "streamkap_snapshot_chunk_size_bytes", "1048576"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "streamkap_snapshot_max_split_size_bytes", "107374182400"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "streamkap_snapshot_state_refresh_ms", "45000"),
				),
			},
			// Step 4: Update to test column_exclude_list
			{
				Config: providerConfig + fmt.Sprintf(`
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
	name                                      = %q
	database_hostname                         = var.source_mysql_hostname
	database_port                             = 3306
	database_user                             = "root"
	database_password                         = var.source_mysql_password
	database_include_list                     = "crm"
	table_include_list                        = "crm.demo"
	signal_data_collection_schema_or_database = "crm"
	column_exclude_list                       = "crm.demo.name"
	heartbeat_enabled                         = true
	heartbeat_data_collection_schema_or_database = "crm"
	database_connection_time_zone              = "SERVER"
	snapshot_gtid                             = "Yes"
	binary_handling_mode                      = "bytes"
	ssh_enabled                               = false
}
`, nameExclude),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "name", nameExclude),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "column_exclude_list", "crm.demo.name"),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "column_include_list", "crm[.]demo[.](id|name)"),
				),
			},
			// Step 5: Update to test insert_static_* fields
			{
				Config: providerConfig + fmt.Sprintf(`
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
	name                                      = %q
	database_hostname                         = var.source_mysql_hostname
	database_port                             = 3306
	database_user                             = "root"
	database_password                         = var.source_mysql_password
	database_include_list                     = "crm"
	table_include_list                        = "crm.demo"
	signal_data_collection_schema_or_database = "crm"
	column_include_list                       = "crm[.]demo[.](id|name)"
	heartbeat_enabled                         = true
	heartbeat_data_collection_schema_or_database = "crm"
	database_connection_time_zone              = "SERVER"
	snapshot_gtid                             = "Yes"
	binary_handling_mode                      = "bytes"
	ssh_enabled                               = false
}
`, nameStatic),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "name", nameStatic),
					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "column_include_list", "crm[.]demo[.](id|name)"),
				),
			},
			// Step 6: Update to test SSH enabled
			// 			{
			// 				Config: providerConfig + `
			// variable "source_mysql_hostname" {
			// 	type        = string
			// 	description = "The hostname of the MySQL database"
			// }
			// variable "source_mysql_password" {
			// 	type        = string
			// 	sensitive   = true
			// 	description = "The password of the MySQL database"
			// }
			// variable "source_mysql_ssh_host" {
			// 	type        = string
			// 	description = "The SSH host for the MySQL database"
			// }
			// resource "streamkap_source_mysql" "test" {
			// 	name                                      = "test-source-mysql-ssh"
			// 	database_hostname                         = var.source_mysql_hostname
			// 	database_port                             = 3306
			// 	database_user                             = "root"
			// 	database_password                         = var.source_mysql_password
			// 	database_include_list                     = "crm"
			// 	table_include_list                        = "crm.demo"
			// 	signal_data_collection_schema_or_database = "crm"
			// 	column_include_list                       = "crm[.]demo[.](id|name)"
			// 	heartbeat_enabled                         = true
			// 	heartbeat_data_collection_schema_or_database = "crm"
			// 	database_connection_time_zone              = "SERVER"
			// 	snapshot_gtid                             = "Yes"
			// 	binary_handling_mode                      = "bytes"
			// 	ssh_enabled                               = true
			// 	ssh_host                                  = var.source_mysql_ssh_host
			// 	ssh_port                                  = "22"
			// 	ssh_user                                  = "streamkap"
			// }
			// `,
			// 				Check: resource.ComposeAggregateTestCheckFunc(
			// 					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "name", "test-source-mysql-ssh"),
			// 					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "ssh_enabled", "true"),
			// 					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "ssh_host", sourceMySQLSSHHost),
			// 					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "ssh_port", "22"),
			// 					resource.TestCheckResourceAttr("streamkap_source_mysql.test", "ssh_user", "streamkap"),
			// 				),
			// 			},
			// Delete testing is handled automatically by TestCase
		},
	})
}
