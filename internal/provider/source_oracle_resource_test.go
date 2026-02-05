package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceOracleHostname = os.Getenv("TF_VAR_source_oracle_hostname")
var sourceOraclePassword = os.Getenv("TF_VAR_source_oracle_password")

func TestAccSourceOracleResource(t *testing.T) {
	if sourceOracleHostname == "" || sourceOraclePassword == "" {
		t.Skip("Skipping TestAccSourceOracleResource: TF_VAR_source_oracle_hostname or TF_VAR_source_oracle_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_oracle_hostname" {
	type        = string
	description = "The hostname of the Oracle database"
}
variable "source_oracle_password" {
	type        = string
	sensitive   = true
	description = "The password of the Oracle database"
}
resource "streamkap_source_oracle" "test" {
	name                                         = "tf-acc-test-source-oracle"
	database_hostname                            = var.source_oracle_hostname
	database_port                                = 1521
	database_user                                = "streamkap"
	database_password                            = var.source_oracle_password
	database_dbname                              = "ORCL"
	schema_include_list                          = "STREAMKAP"
	table_include_list                           = "STREAMKAP.CUSTOMER"
	signal_data_collection_schema_or_database    = "STREAMKAP"
	heartbeat_enabled                            = true
	heartbeat_data_collection_schema_or_database = "STREAMKAP"
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "name", "tf-acc-test-source-oracle"),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "database_hostname", sourceOracleHostname),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "database_port", "1521"),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "database_user", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "database_password", sourceOraclePassword),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "database_dbname", "ORCL"),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "schema_include_list", "STREAMKAP"),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "table_include_list", "STREAMKAP.CUSTOMER"),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "signal_data_collection_schema_or_database", "STREAMKAP"),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "heartbeat_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "heartbeat_data_collection_schema_or_database", "STREAMKAP"),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "ssh_enabled", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_oracle.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_oracle_hostname" {
	type        = string
	description = "The hostname of the Oracle database"
}
variable "source_oracle_password" {
	type        = string
	sensitive   = true
	description = "The password of the Oracle database"
}
resource "streamkap_source_oracle" "test" {
	name                                         = "tf-acc-test-source-oracle-updated"
	database_hostname                            = var.source_oracle_hostname
	database_port                                = 1521
	database_user                                = "streamkap"
	database_password                            = var.source_oracle_password
	database_dbname                              = "ORCL"
	schema_include_list                          = "STREAMKAP"
	table_include_list                           = "STREAMKAP.CUSTOMER,STREAMKAP.ORDERS"
	signal_data_collection_schema_or_database    = "STREAMKAP"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = "STREAMKAP"
	binary_handling_mode                         = "base64"
	ssh_enabled                                  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "name", "tf-acc-test-source-oracle-updated"),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "table_include_list", "STREAMKAP.CUSTOMER,STREAMKAP.ORDERS"),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_oracle.test", "binary_handling_mode", "base64"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
