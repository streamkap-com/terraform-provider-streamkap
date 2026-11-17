package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceSQLServerHostname = os.Getenv("TF_VAR_source_sqlserver_hostname")
var sourceSQLServerPassword = os.Getenv("TF_VAR_source_sqlserver_password")

func TestAccSourceSQLServerResource(t *testing.T) {
	name := acctestName(t, "main")
	nameUpdated := acctestName(t, "updated")
	nameExclude := acctestName(t, "exclude")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
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
	name                                         = %q
	database_hostname                            = var.source_sqlserver_hostname
	database_port                                = 1433
	database_user                                = "sa"
	database_password                            = var.source_sqlserver_password
	database_names                              = "demo"
	schema_include_list                          = "dbo"
	table_include_list                           = "dbo.Orders,dbo.Customers"
	signal_data_collection_schema_or_database    = "streamkap"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	binary_handling_mode                         = "bytes"
	streamkap_snapshot_max_split_size_bytes      = 21474836480
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_hostname", sourceSQLServerHostname),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_port", "1433"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_user", "sa"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_password", sourceSQLServerPassword),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_names", "demo"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "schema_include_list", "dbo"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "table_include_list", "dbo.Orders,dbo.Customers"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "signal_data_collection_schema_or_database", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "streamkap_snapshot_max_split_size_bytes", "21474836480"),
					// Check defaults for unset attributes
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "binary_handling_mode", "bytes"),
				),
			},
			// Step 2: ImportState testing
			{
				ResourceName:            "streamkap_source_sqlserver.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Step 3: Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
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
	name                                         = %q
	database_hostname                            = var.source_sqlserver_hostname
	database_port                                = 1433
	database_user                                = "sa"
	database_password                            = var.source_sqlserver_password
	database_names                              = "demo"
	schema_include_list                          = "dbo"
	table_include_list                           = "dbo.Orders, dbo.Customers"
	signal_data_collection_schema_or_database    = "streamkap"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = false
	streamkap_snapshot_max_split_size_bytes      = 107374182400
}
`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_hostname", sourceSQLServerHostname),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_port", "1433"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_user", "sa"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_password", sourceSQLServerPassword),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_names", "demo"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "schema_include_list", "dbo"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "table_include_list", "dbo.Orders, dbo.Customers"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "signal_data_collection_schema_or_database", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "ssh_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "streamkap_snapshot_max_split_size_bytes", "107374182400"),
				),
			},
			// Step 4: Update to test column_exclude_list
			{
				Config: providerConfig + fmt.Sprintf(`
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
	name                                         = %q
	database_hostname                            = var.source_sqlserver_hostname
	database_port                                = 1433
	database_user                                = "sa"
	database_password                            = var.source_sqlserver_password
	database_names                              = "demo"
	schema_include_list                          = "dbo"
	table_include_list                           = "dbo.Orders, dbo.Customers"
	signal_data_collection_schema_or_database    = "streamkap"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = false
	column_exclude_list                          = "streamkap.customer.name"  # Switch to exclude list
}
`, nameExclude),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "name", nameExclude),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_hostname", sourceSQLServerHostname),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_port", "1433"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_user", "sa"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_password", sourceSQLServerPassword),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_names", "demo"),
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
			// 			{
			// 				Config: providerConfig + `
			// variable "source_sqlserver_hostname" {
			// 	type        = string
			// 	description = "The hostname of the SQLServer database"
			// }
			// variable "source_sqlserver_password" {
			// 	type        = string
			// 	sensitive   = true
			// 	description = "The password of the SQLServer database"
			// }
			// variable "source_sqlserver_ssh_host" {
			// 	type        = string
			// 	description = "The SSH host for the SQLServer database"
			// }
			// resource "streamkap_source_sqlserver" "test" {
			// 	name                                         = "test-source-sqlserver-ssh"
			// 	database_hostname                            = var.source_sqlserver_hostname
			// 	database_port                                = 1433
			// 	database_user                                = "sa"
			// 	database_password                            = var.source_sqlserver_password
			// 	database_names                              = "demo"
			// 	schema_include_list                          = "dbo"
			// 	table_include_list                           = "dbo.Orders, dbo.Customers"
			// 	signal_data_collection_schema_or_database    = "streamkap"
			// 	column_exclude_list                          = ""
			// 	heartbeat_enabled                            = false
			// 	heartbeat_data_collection_schema_or_database = null
			// 	binary_handling_mode                         = "bytes"
			// 	ssh_enabled                                  = false
			// 	ssh_host                                     = var.source_sqlserver_ssh_host
			// 	ssh_port                                     = "22"
			// 	ssh_user                                     = "streamkap"
			// }
			// `,
			// 				Check: resource.ComposeAggregateTestCheckFunc(
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "name", "test-source-sqlserver-ssh"),
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_hostname", sourceSQLServerHostname),
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_port", "1433"),
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_user", "sa"),
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_password", sourceSQLServerPassword),
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "database_names", "demo"),
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "schema_include_list", "dbo"),
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "table_include_list", "dbo.Orders, dbo.Customers"),
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "signal_data_collection_schema_or_database", "streamkap"),
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "heartbeat_enabled", "false"),
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "binary_handling_mode", "bytes"),
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "ssh_enabled", "false"),
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "ssh_host", sourceSQLServerSSHHost),
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "ssh_port", "22"),
			// 					resource.TestCheckResourceAttr("streamkap_source_sqlserver.test", "ssh_user", "streamkap"),
			// 				),
			// 			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// TestAccSourceSQLServerResource_EntraServicePrincipal covers the Entra SP
// auth mode. database_user and database_password must be omitted; the backend
// derives the connection credentials from azure_client_id/azure_client_secret.
func TestAccSourceSQLServerResource_EntraServicePrincipal(t *testing.T) {
	azureTenantID := os.Getenv("TF_VAR_source_sqlserver_azure_tenant_id")
	azureClientID := os.Getenv("TF_VAR_source_sqlserver_azure_client_id")
	azureClientSecret := os.Getenv("TF_VAR_source_sqlserver_azure_client_secret")
	if azureTenantID == "" || azureClientID == "" || azureClientSecret == "" {
		t.Skip("set TF_VAR_source_sqlserver_azure_tenant_id, TF_VAR_source_sqlserver_azure_client_id and TF_VAR_source_sqlserver_azure_client_secret to run this test")
	}

	name := acctestName(t, "entra")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
variable "source_sqlserver_hostname" {
	type        = string
	description = "The hostname of the SQLServer database"
}
variable "source_sqlserver_azure_tenant_id" {
	type        = string
	description = "Azure AD tenant ID"
}
variable "source_sqlserver_azure_client_id" {
	type        = string
	description = "Entra app registration client ID"
}
variable "source_sqlserver_azure_client_secret" {
	type        = string
	sensitive   = true
	description = "Entra app registration client secret"
}
resource "streamkap_source_sqlserver" "entra" {
	name                                      = %q
	database_hostname                         = var.source_sqlserver_hostname
	database_port                             = 1433
	auth_mode                                 = "entra_service_principal"
	azure_tenant_id                           = var.source_sqlserver_azure_tenant_id
	azure_client_id                           = var.source_sqlserver_azure_client_id
	azure_client_secret                       = var.source_sqlserver_azure_client_secret
	database_names                            = "demo"
	schema_include_list                       = "dbo"
	table_include_list                        = "dbo.Orders"
	signal_data_collection_schema_or_database = "streamkap"
	heartbeat_enabled                         = false
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.entra", "name", name),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.entra", "auth_mode", "entra_service_principal"),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.entra", "azure_tenant_id", azureTenantID),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.entra", "azure_client_id", azureClientID),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.entra", "azure_client_secret", azureClientSecret),
					// Entra mode: the backend reflection reads database_user only from
					// the .user.defined wire key, so Read echoes null for both.
					resource.TestCheckNoResourceAttr("streamkap_source_sqlserver.entra", "database_user"),
					resource.TestCheckNoResourceAttr("streamkap_source_sqlserver.entra", "database_password"),
				),
			},
			{
				// azure_client_secret round-trips via secret_returned=true,
				// matching database_password; no ignore needed.
				ResourceName:            "streamkap_source_sqlserver.entra",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
		},
	})
}
