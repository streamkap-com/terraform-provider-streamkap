package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourcePostgreSQLHostname = os.Getenv("TF_VAR_source_postgresql_hostname")
var sourcePostgreSQLPassword = os.Getenv("TF_VAR_source_postgresql_password")
var sourcePostgreSQLSSHHost = os.Getenv("TF_VAR_source_postgresql_ssh_host")

func TestAccSourcePostgreSQLResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create and Read testing
			{
				Config: providerConfig + `
variable "source_postgresql_hostname" {
	type        = string
	description = "The hostname of the PostgreSQL database"
}
variable "source_postgresql_password" {
	type        = string
	sensitive   = true
	description = "The password of the PostgreSQL database"
}
resource "streamkap_source_postgresql" "test" {
	name                                         = "test-source-postgresql"
	database_hostname                            = var.source_postgresql_hostname
	database_port                                = "5432"
	database_user                                = "postgresql"
	database_password                            = var.source_postgresql_password
	database_dbname                              = "postgres"
	snapshot_read_only                           = "No"
	database_sslmode                             = "require"
	schema_include_list                          = "streamkap"
	table_include_list                           = "streamkap.customer,streamkap.customer2"
	signal_data_collection_schema_or_database    = "streamkap"
	column_include_list                          = "streamkap[.]customer[.](id|name)"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	include_source_db_name_in_table_name         = false
	slot_name                                    = "terraform_pgoutput_slot"
	publication_name                             = "terraform_pub"
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "name", "test-source-postgresql"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_hostname", sourcePostgreSQLHostname),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_port", "5432"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_user", "postgresql"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_password", sourcePostgreSQLPassword),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_dbname", "postgres"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "snapshot_read_only", "No"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_sslmode", "require"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "schema_include_list", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "table_include_list", "streamkap.customer,streamkap.customer2"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "signal_data_collection_schema_or_database", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "column_include_list", "streamkap[.]customer[.](id|name)"),
					// Check defaults for unset attributes
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "include_source_db_name_in_table_name", "false"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "slot_name", "terraform_pgoutput_slot"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "publication_name", "terraform_pub"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "ssh_enabled", "false"),
				),
			},
			// Step 2: ImportState testing
			{
				ResourceName:      "streamkap_source_postgresql.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update and Read testing
			{
				Config: providerConfig + `
variable "source_postgresql_hostname" {
	type        = string
	description = "The hostname of the PostgreSQL database"
}
variable "source_postgresql_password" {
	type        = string
	sensitive   = true
	description = "The password of the PostgreSQL database"
}
resource "streamkap_source_postgresql" "test" {
	name                                         = "test-source-postgresql-updated"
	database_hostname                            = var.source_postgresql_hostname
	database_port                                = "5432"
	database_user                                = "postgresql"
	database_password                            = var.source_postgresql_password
	database_dbname                              = "postgres"
	snapshot_read_only                           = "Yes"
	database_sslmode                             = "require"
	schema_include_list                          = "streamkap"
	table_include_list                           = "streamkap.customer"
	signal_data_collection_schema_or_database    = "streamkap"
	column_include_list                          = "streamkap[.]customer[.](id|name)"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	include_source_db_name_in_table_name         = false
	slot_name                                    = "terraform_pgoutput_slot"
	publication_name                             = "terraform_pub"
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "name", "test-source-postgresql-updated"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_hostname", sourcePostgreSQLHostname),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_port", "5432"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_user", "postgresql"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_password", sourcePostgreSQLPassword),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_dbname", "postgres"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "snapshot_read_only", "Yes"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_sslmode", "require"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "schema_include_list", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "table_include_list", "streamkap.customer"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "signal_data_collection_schema_or_database", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "column_include_list", "streamkap[.]customer[.](id|name)"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "include_source_db_name_in_table_name", "false"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "slot_name", "terraform_pgoutput_slot"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "publication_name", "terraform_pub"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "ssh_enabled", "false"),
				),
			},
			// Step 4: Update to test column_exclude_list
			{
				Config: providerConfig + `
variable "source_postgresql_hostname" {
	type        = string
	description = "The hostname of the PostgreSQL database"
}
variable "source_postgresql_password" {
	type        = string
	sensitive   = true
	description = "The password of the PostgreSQL database"
}
resource "streamkap_source_postgresql" "test" {
	name                                         = "test-source-postgresql-exclude"
	database_hostname                            = var.source_postgresql_hostname
	database_port                                = "5432"
	database_user                                = "postgresql"
	database_password                            = var.source_postgresql_password
	database_dbname                              = "postgres"
	snapshot_read_only                           = "No"
	database_sslmode                             = "require"
	schema_include_list                          = "streamkap"
	table_include_list                           = "streamkap.customer"
	signal_data_collection_schema_or_database    = "streamkap"
	column_exclude_list                          = "streamkap.customer.name"  # Switch to exclude list
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	include_source_db_name_in_table_name         = false
	slot_name                                    = "terraform_pgoutput_slot"
	publication_name                             = "terraform_pub"
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "name", "test-source-postgresql-exclude"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_hostname", sourcePostgreSQLHostname),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_port", "5432"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_user", "postgresql"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_password", sourcePostgreSQLPassword),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_dbname", "postgres"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "snapshot_read_only", "No"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_sslmode", "require"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "schema_include_list", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "table_include_list", "streamkap.customer"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "signal_data_collection_schema_or_database", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "column_exclude_list", "streamkap.customer.name"),
					// Verify column_include_list is not set (null)
					resource.TestCheckNoResourceAttr("streamkap_source_postgresql.test", "column_include_list"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "include_source_db_name_in_table_name", "false"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "slot_name", "terraform_pgoutput_slot"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "publication_name", "terraform_pub"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "ssh_enabled", "false"),
				),
			},
			// Step 5: Update to test SSH enabled
			{
				Config: providerConfig + `
variable "source_postgresql_hostname" {
	type        = string
	description = "The hostname of the PostgreSQL database"
}
variable "source_postgresql_password" {
	type        = string
	sensitive   = true
	description = "The password of the PostgreSQL database"
}
variable "source_postgresql_ssh_host" {
	type        = string
	description = "The SSH host for the PostgreSQL database"
}
resource "streamkap_source_postgresql" "test" {
	name                                         = "test-source-postgresql-ssh"
	database_hostname                            = var.source_postgresql_hostname
	database_port                                = "5432"
	database_user                                = "postgresql"
	database_password                            = var.source_postgresql_password
	database_dbname                              = "postgres"
	snapshot_read_only                           = "No"
	database_sslmode                             = "require"
	schema_include_list                          = "streamkap"
	table_include_list                           = "streamkap.customer"
	signal_data_collection_schema_or_database    = "streamkap"
	column_include_list                          = "streamkap[.]customer[.](id|name)"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	include_source_db_name_in_table_name         = false
	slot_name                                    = "terraform_pgoutput_slot"
	publication_name                             = "terraform_pub"
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = true
	ssh_host                                     = var.source_postgresql_ssh_host
	ssh_port                                     = "22"
	ssh_user                                     = "streamkap"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "name", "test-source-postgresql-ssh"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_hostname", sourcePostgreSQLHostname),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_port", "5432"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_user", "postgresql"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_password", sourcePostgreSQLPassword),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_dbname", "postgres"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "snapshot_read_only", "No"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_sslmode", "require"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "schema_include_list", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "table_include_list", "streamkap.customer"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "signal_data_collection_schema_or_database", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "column_include_list", "streamkap[.]customer[.](id|name)"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "include_source_db_name_in_table_name", "false"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "slot_name", "terraform_pgoutput_slot"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "publication_name", "terraform_pub"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "ssh_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "ssh_host", sourcePostgreSQLSSHHost),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "ssh_port", "22"),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "ssh_user", "streamkap"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccSourcePostgreSQLResource_WithTimeout(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
variable "source_postgresql_hostname" {
	type = string
}
variable "source_postgresql_password" {
	type      = string
	sensitive = true
}
resource "streamkap_source_postgresql" "test_timeout" {
	name              = "test-source-postgresql-timeout"
	database_hostname = var.source_postgresql_hostname
	database_port     = "5432"
	database_user     = "postgresql"
	database_password = var.source_postgresql_password
	database_dbname   = "postgres"
	database_sslmode  = "require"
	schema_include_list = "streamkap"
	table_include_list  = "streamkap.customer"
	signal_data_collection_schema_or_database = "streamkap"
	slot_name         = "terraform_timeout_test_slot"
	publication_name  = "terraform_timeout_test_pub"
	ssh_enabled       = false

	timeouts {
		create = "30m"
		update = "30m"
		delete = "15m"
	}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test_timeout", "name", "test-source-postgresql-timeout"),
				),
			},
		},
	})
}
