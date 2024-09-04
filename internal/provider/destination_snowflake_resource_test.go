package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationSnowflakeURLName = os.Getenv("TF_VAR_destination_snowflake_url_name")
var destinationSnowflakePrivateKey = os.Getenv("TF_VAR_destination_snowflake_private_key")
var destinationSnowflakeKeyPassphrase = os.Getenv("TF_VAR_destination_snowflake_key_passphrase")

func TestAccDestinationSnowflakeResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "destination_snowflake_url_name" {
	type        = string
	description = "The URL name of the Snowflake database"
}
variable "destination_snowflake_private_key" {
	type        = string
	sensitive   = true
	description = "The private key of the Snowflake database"
}
variable "destination_snowflake_key_passphrase" {
	type        = string
	sensitive   = true
	description = "The passphrase of the private key of the Snowflake database"
}
resource "streamkap_destination_snowflake" "test" {
	name                             = "test-destination-snowflake"
	snowflake_url_name               = var.destination_snowflake_url_name
	snowflake_user_name              = "STREAMKAP_USER_JUNIT"
	snowflake_private_key            = var.destination_snowflake_private_key
	snowflake_private_key_passphrase = var.destination_snowflake_key_passphrase
	sfwarehouse                      = "STREAMKAP_WH"
	snowflake_database_name          = "JUNIT"
	snowflake_schema_name            = "JUNIT"
	snowflake_role_name              = "STREAMKAP_ROLE_JUNIT"
	ingestion_mode                   = "upsert"
	hard_delete                      = true
	use_hybrid_tables                = false
	apply_dynamic_table_script       = false
	dynamic_table_target_lag         = 60
	cleanup_task_schedule            = 120
	dedupe_table_mapping = {
		users                   = "JUNIT.USERS",
		itst_scen20240528103635 = "ITST_SCEN20240528103635"
	}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "name", "test-destination-snowflake"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_url_name", destinationSnowflakeURLName),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_user_name", "STREAMKAP_USER_JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key", destinationSnowflakePrivateKey),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key_passphrase", destinationSnowflakeKeyPassphrase),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "sfwarehouse", "STREAMKAP_WH"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_database_name", "JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_schema_name", "JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_role_name", "STREAMKAP_ROLE_JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "ingestion_mode", "upsert"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "hard_delete", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "use_hybrid_tables", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "apply_dynamic_table_script", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "dynamic_table_target_lag", "60"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "cleanup_task_schedule", "120"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "dedupe_table_mapping.users", "JUNIT.USERS"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "dedupe_table_mapping.itst_scen20240528103635", "ITST_SCEN20240528103635"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_destination_snowflake.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "destination_snowflake_url_name" {
	type        = string
	description = "The URL name of the Snowflake database"
}
variable "destination_snowflake_private_key" {
	type        = string
	sensitive   = true
	description = "The private key of the Snowflake database"
}
variable "destination_snowflake_key_passphrase" {
	type        = string
	sensitive   = true
	description = "The passphrase of the private key of the Snowflake database"
}
resource "streamkap_destination_snowflake" "test" {
	name                             = "test-destination-snowflake-updated"
	snowflake_url_name               = var.destination_snowflake_url_name
	snowflake_user_name              = "STREAMKAP_USER_JUNIT"
	snowflake_private_key            = var.destination_snowflake_private_key
	snowflake_private_key_passphrase = var.destination_snowflake_key_passphrase
	sfwarehouse                      = "STREAMKAP_WH"
	snowflake_database_name          = "JUNIT"
	snowflake_schema_name            = "JUNIT"
	snowflake_role_name              = "STREAMKAP_ROLE_JUNIT"
	ingestion_mode                   = "append"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "name", "test-destination-snowflake-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_url_name", destinationSnowflakeURLName),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_user_name", "STREAMKAP_USER_JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key", destinationSnowflakePrivateKey),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key_passphrase", destinationSnowflakeKeyPassphrase),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "sfwarehouse", "STREAMKAP_WH"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_database_name", "JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_schema_name", "JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_role_name", "STREAMKAP_ROLE_JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "ingestion_mode", "append"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "hard_delete", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "use_hybrid_tables", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "apply_dynamic_table_script", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "dynamic_table_target_lag", "15"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "cleanup_task_schedule", "60"),
					resource.TestCheckNoResourceAttr("streamkap_destination_snowflake.test", "dedupe_table_mapping"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
