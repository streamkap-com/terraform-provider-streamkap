package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceSQLServerHostname = os.Getenv("TF_VAR_source_sqlserver_hostname")
var sourceSQLServerPassword = os.Getenv("TF_VAR_source_sqlserver_password")
var sourceSQLServerSSHHost = os.Getenv("TF_VAR_source_sqlserver_ssh_host")
var sourceSQLServerEntraHostname = os.Getenv("TF_VAR_source_sqlserver_entra_hostname")
var sourceSQLServerEntraTenantID = os.Getenv("TF_VAR_source_sqlserver_entra_tenant_id")
var sourceSQLServerEntraClientID = os.Getenv("TF_VAR_source_sqlserver_entra_client_id")
var sourceSQLServerEntraClientSecret = os.Getenv("TF_VAR_source_sqlserver_entra_client_secret")

func TestAccSourceSQLServerResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create and Read testing
			{
				Config: providerConfig + `
variable "source_sqlserver_hostname" {
	type        = string
	description = "The hostname of the SQLServer database"
}
variable "source_sqlserver_password" {
	type        = string
	sensitive   = true
	description = "The password of the SQLServer database"
}
resource "streamkap_source_sqlserver" "test" {
	name                                         = "test-source-sqlserver"
	database_hostname                            = var.source_sqlserver_hostname
	database_port                                = 1433
	database_user                                = "admin"
	database_password                            = var.source_sqlserver_password
	database_dbname                              = "sqlserverdemo"
	schema_include_list                          = "dbo"
	table_include_list                           = "dbo.Orders,dbo.Customers"
	signal_data_collection_schema_or_database    = "streamkap"
	column_exclude_list                          = ""
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	binary_handling_mode                         = "bytes"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "name", "test-source-sqlserver"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_hostname", sourceSQLServerHostname),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_port", "1433"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_user", "admin"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_password", sourceSQLServerPassword),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_dbname", "sqlserverdemo"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "schema_include_list", "dbo"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "table_include_list", "dbo.Orders,dbo.Customers"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "signal_data_collection_schema_or_database", "streamkap"),
					// Check defaults for unset attributes
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "binary_handling_mode", "bytes"),
				),
			},
			// Step 2: ImportState testing
			{
				ResourceName:      "streamkap_source_sqlserver.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update and Read testing
			{
				Config: providerConfig + `
variable "source_sqlserver_hostname" {
	type        = string
	description = "The hostname of the SQLServer database"
}
variable "source_sqlserver_password" {
	type        = string
	sensitive   = true
	description = "The password of the SQLServer database"
}
resource "streamkap_source_sqlserver" "test" {
	name                                         = "test-source-sqlserver-updated"
	database_hostname                            = var.source_sqlserver_hostname
	database_port                                = 1433
	database_user                                = "admin"
	database_password                            = var.source_sqlserver_password
	database_dbname                              = "sqlserverdemo"
	schema_include_list                          = "dbo"
	table_include_list                           = "dbo.Orders, dbo.Customers"
	signal_data_collection_schema_or_database    = "streamkap"
	column_exclude_list                          = ""
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "name", "test-source-sqlserver-updated"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_hostname", sourceSQLServerHostname),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_port", "1433"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_user", "admin"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_password", sourceSQLServerPassword),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_dbname", "sqlserverdemo"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "schema_include_list", "dbo"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "table_include_list", "dbo.Orders, dbo.Customers"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "signal_data_collection_schema_or_database", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "ssh_enabled", "false"),
				),
			},
			// Step 4: Update to test column_exclude_list
			{
				Config: providerConfig + `
variable "source_sqlserver_hostname" {
	type        = string
	description = "The hostname of the SQLServer database"
}
variable "source_sqlserver_password" {
	type        = string
	sensitive   = true
	description = "The password of the SQLServer database"
}
resource "streamkap_source_sqlserver" "test" {
	name                                         = "test-source-sqlserver-exclude"
	database_hostname                            = var.source_sqlserver_hostname
	database_port                                = 1433
	database_user                                = "admin"
	database_password                            = var.source_sqlserver_password
	database_dbname                              = "sqlserverdemo"
	schema_include_list                          = "dbo"
	table_include_list                           = "dbo.Orders, dbo.Customers"
	signal_data_collection_schema_or_database    = "streamkap"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = false
	column_exclude_list                          = "streamkap.customer.name"  # Switch to exclude list
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "name", "test-source-sqlserver-exclude"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_hostname", sourceSQLServerHostname),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_port", "1433"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_user", "admin"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_password", sourceSQLServerPassword),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_dbname", "sqlserverdemo"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "schema_include_list", "dbo"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "table_include_list", "dbo.Orders, dbo.Customers"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "signal_data_collection_schema_or_database", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "ssh_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "column_exclude_list", "streamkap.customer.name"),
				),
			},
			// Step 5: Update to test SSH enabled
			{
				Config: providerConfig + `
variable "source_sqlserver_hostname" {
	type        = string
	description = "The hostname of the SQLServer database"
}
variable "source_sqlserver_password" {
	type        = string
	sensitive   = true
	description = "The password of the SQLServer database"
}
variable "source_sqlserver_ssh_host" {
	type        = string
	description = "The SSH host for the SQLServer database"
}
resource "streamkap_source_sqlserver" "test" {
	name                                         = "test-source-sqlserver-ssh"
	database_hostname                            = var.source_sqlserver_hostname
	database_port                                = 1433
	database_user                                = "admin"
	database_password                            = var.source_sqlserver_password
	database_dbname                              = "sqlserverdemo"
	schema_include_list                          = "dbo"
	table_include_list                           = "dbo.Orders, dbo.Customers"
	signal_data_collection_schema_or_database    = "streamkap"
	column_exclude_list                          = ""
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = true
	ssh_host                                     = var.source_sqlserver_ssh_host
	ssh_port                                     = "22"
	ssh_user                                     = "streamkap"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "name", "test-source-sqlserver-ssh"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_hostname", sourceSQLServerHostname),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_port", "1433"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_user", "admin"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_password", sourceSQLServerPassword),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_dbname", "sqlserverdemo"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "schema_include_list", "dbo"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "table_include_list", "dbo.Orders, dbo.Customers"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "signal_data_collection_schema_or_database", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "ssh_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "ssh_host", sourceSQLServerSSHHost),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "ssh_port", "22"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "ssh_user", "streamkap"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccSourceSQLServer_entraServicePrincipal(t *testing.T) {
	if sourceSQLServerEntraHostname == "" || sourceSQLServerEntraTenantID == "" || sourceSQLServerEntraClientID == "" || sourceSQLServerEntraClientSecret == "" {
		t.Skip("TF_VAR_source_sqlserver_entra_{hostname,tenant_id,client_id,client_secret} must be set")
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_sqlserver_entra_hostname" {
	type        = string
	description = "The hostname of the Azure SQL database"
}
variable "source_sqlserver_entra_tenant_id" {
	type        = string
	description = "Azure Entra tenant ID"
}
variable "source_sqlserver_entra_client_id" {
	type        = string
	description = "Azure Entra client ID"
}
variable "source_sqlserver_entra_client_secret" {
	type        = string
	sensitive   = true
	description = "Azure Entra client secret"
}
resource "streamkap_source_sqlserver" "test_entra" {
	name                                      = "test-source-sqlserver-entra"
	database_hostname                         = var.source_sqlserver_entra_hostname
	database_port                             = 1433
	auth_mode                                 = "entra_service_principal"
	azure_tenant_id                           = var.source_sqlserver_entra_tenant_id
	azure_client_id                           = var.source_sqlserver_entra_client_id
	azure_client_secret                       = var.source_sqlserver_entra_client_secret
	database_dbname                           = "sqlserverdemo"
	schema_include_list                       = "dbo"
	table_include_list                        = "dbo.Orders"
	signal_data_collection_schema_or_database = "streamkap"
	heartbeat_enabled                         = false
	binary_handling_mode                      = "bytes"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test_entra", "name", "test-source-sqlserver-entra"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test_entra", "database_hostname", sourceSQLServerEntraHostname),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test_entra", "auth_mode", "entra_service_principal"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test_entra", "azure_tenant_id", sourceSQLServerEntraTenantID),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test_entra", "azure_client_id", sourceSQLServerEntraClientID),
					resource.TestCheckResourceAttrSet("streamkap_source_sqlserver.test_entra", "azure_client_secret"),
					// database.user / database.password resolve to azure_client_id / secret server-side — they must not leak back into the TF model.
					resource.TestCheckNoResourceAttr("streamkap_source_sqlserver.test_entra", "database_user"),
					resource.TestCheckNoResourceAttr("streamkap_source_sqlserver.test_entra", "database_password"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
