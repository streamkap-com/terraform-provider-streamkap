package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationSnowflakeURLName = os.Getenv("DESTINATION_SNOWFLAKE_URL_NAME")
var destinationSnowflakePrivateKey = os.Getenv("DESTINATION_SNOWFLAKE_PRIVATE_KEY")
var destinationSnowflakeKeyPassphrase = os.Getenv("DESTINATION_SNOWFLAKE_KEY_PASSPHRASE")

func TestAccDestinationSnowflakeResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_destination_snowflake" "test" {
	name                             = "test-destination-snowflake"
	snowflake_url_name               = "%s"
	snowflake_user_name              = "STREAMKAP_USER_POSTGRESQL"
	snowflake_private_key            = "%s"
	snowflake_private_key_passphrase = "%s"
	snowflake_database_name          = "STREAMKAP_POSTGRESQL"
	snowflake_schema_name            = "STREAMKAP"
	snowflake_role_name              = "STREAMKAP_ROLE"
}
`, destinationSnowflakeURLName, destinationSnowflakePrivateKey, destinationSnowflakeKeyPassphrase),
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
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_destination_snowflake" "test" {
	name                             = "test-destination-snowflake-updated"
	snowflake_url_name               = "%s"
	snowflake_user_name              = "STREAMKAP_USER_POSTGRESQL"
	snowflake_private_key            = "%s"
	snowflake_private_key_passphrase = "%s"
	snowflake_database_name          = "STREAMKAP_POSTGRESQL"
	snowflake_schema_name            = "STREAMKAP"
	snowflake_role_name              = "STREAMKAP_ROLE"
}
`, destinationSnowflakeURLName, destinationSnowflakePrivateKey, destinationSnowflakeKeyPassphrase),
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
