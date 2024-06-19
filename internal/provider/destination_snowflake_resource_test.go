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
	snowflake_user_name              = "STREAMKAP_USER_POSTGRESQL"
	snowflake_private_key            = var.destination_snowflake_private_key
	snowflake_private_key_passphrase = var.destination_snowflake_key_passphrase
	snowflake_database_name          = "STREAMKAP_POSTGRESQL"
	snowflake_schema_name            = "STREAMKAP"
	snowflake_role_name              = "STREAMKAP_ROLE"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "name", "test-destination-snowflake"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_url_name", destinationSnowflakeURLName),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_user_name", "STREAMKAP_USER_POSTGRESQL"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key", destinationSnowflakePrivateKey),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key_passphrase", destinationSnowflakeKeyPassphrase),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_database_name", "STREAMKAP_POSTGRESQL"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_schema_name", "STREAMKAP"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_role_name", "STREAMKAP_ROLE"),
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
	snowflake_user_name              = "STREAMKAP_USER_POSTGRESQL"
	snowflake_private_key            = var.destination_snowflake_private_key
	snowflake_private_key_passphrase = var.destination_snowflake_key_passphrase
	snowflake_database_name          = "STREAMKAP_POSTGRESQL"
	snowflake_schema_name            = "STREAMKAP"
	snowflake_role_name              = "STREAMKAP_ROLE"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "name", "test-destination-snowflake-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_url_name", destinationSnowflakeURLName),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_user_name", "STREAMKAP_USER_POSTGRESQL"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key", destinationSnowflakePrivateKey),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key_passphrase", destinationSnowflakeKeyPassphrase),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_database_name", "STREAMKAP_POSTGRESQL"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_schema_name", "STREAMKAP"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_role_name", "STREAMKAP_ROLE"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
