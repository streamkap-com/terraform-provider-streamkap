package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceOracleAWSHostname = os.Getenv("TF_VAR_source_oracleaws_hostname")
var sourceOracleAWSPassword = os.Getenv("TF_VAR_source_oracleaws_password")

func TestAccSourceOracleAWSResource(t *testing.T) {
	if sourceOracleAWSHostname == "" || sourceOracleAWSPassword == "" {
		t.Skip("Skipping TestAccSourceOracleAWSResource: TF_VAR_source_oracleaws_hostname or TF_VAR_source_oracleaws_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_oracleaws_hostname" {
	type        = string
	description = "The hostname of the Oracle RDS database"
}
variable "source_oracleaws_password" {
	type        = string
	sensitive   = true
	description = "The password of the Oracle RDS database"
}
resource "streamkap_source_oracleaws" "test" {
	name                                         = "tf-acc-test-source-oracleaws"
	database_hostname                            = var.source_oracleaws_hostname
	database_port                                = 1521
	database_user                                = "streamkap"
	database_password                            = var.source_oracleaws_password
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
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "name", "tf-acc-test-source-oracleaws"),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "database_hostname", sourceOracleAWSHostname),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "database_port", "1521"),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "database_user", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "database_password", sourceOracleAWSPassword),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "database_dbname", "ORCL"),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "schema_include_list", "STREAMKAP"),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "table_include_list", "STREAMKAP.CUSTOMER"),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "signal_data_collection_schema_or_database", "STREAMKAP"),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "heartbeat_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "heartbeat_data_collection_schema_or_database", "STREAMKAP"),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "ssh_enabled", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_oracleaws.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_oracleaws_hostname" {
	type        = string
	description = "The hostname of the Oracle RDS database"
}
variable "source_oracleaws_password" {
	type        = string
	sensitive   = true
	description = "The password of the Oracle RDS database"
}
resource "streamkap_source_oracleaws" "test" {
	name                                         = "tf-acc-test-source-oracleaws-updated"
	database_hostname                            = var.source_oracleaws_hostname
	database_port                                = 1521
	database_user                                = "streamkap"
	database_password                            = var.source_oracleaws_password
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
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "name", "tf-acc-test-source-oracleaws-updated"),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "table_include_list", "STREAMKAP.CUSTOMER,STREAMKAP.ORDERS"),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_oracleaws.test", "binary_handling_mode", "base64"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
