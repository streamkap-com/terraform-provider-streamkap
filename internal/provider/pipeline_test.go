package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var pipelineSrcPostgreSQLHostname = os.Getenv("SOURCE_POSTGRESQL_HOSTNAME")
var pipelineSrcPostgreSQLPassword = os.Getenv("SOURCE_POSTGRESQL_PASSWORD")
var pipelineDestSnowflakeURLName = os.Getenv("DESTINATION_SNOWFLAKE_URL_NAME")
var pipelineDestSnowflakePrivateKey = os.Getenv("DESTINATION_SNOWFLAKE_PRIVATE_KEY")
var pipelineDestSnowflakeKeyPassphrase = os.Getenv("DESTINATION_SNOWFLAKE_KEY_PASSPHRASE")

var pipelineSrcPostgreSQLResourceDef = fmt.Sprintf(`
resource "streamkap_source_postgresql" "test" {
	name                                         = "test-source-postgresql"
	database_hostname                            = "%s"
	database_port                                = 5432
	database_user                                = "postgresql"
	database_password                            = "%s"
	database_dbname                              = "postgres"
	database_sslmode                             = "require"
	schema_include_list                          = "public"
	table_include_list                           = "public.users"
	signal_data_collection_schema_or_database    = "streamkap"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	include_source_db_name_in_table_name         = false
	slot_name                                    = "fleetio_pgoutput_slot"
	publication_name                             = "fleetio_pub"
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = false
}
`, pipelineSrcPostgreSQLHostname, pipelineSrcPostgreSQLPassword)

var pipelineDestSnowflakeResourceDef = fmt.Sprintf(`
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
`, pipelineDestSnowflakeURLName, pipelineDestSnowflakePrivateKey, pipelineDestSnowflakeKeyPassphrase)

func TestAccPipelineResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + pipelineSrcPostgreSQLResourceDef + pipelineDestSnowflakeResourceDef + `
resource "streamkap_pipeline" "test" {
	name                             = "test-pipeline"
	source = {
		id        = streamkap_source_postgresql.test.id
		name      = streamkap_source_postgresql.test.name
		connector = streamkap_source_postgresql.test.connector
		topics    = ["public.users"]
		}
	destination = {
		id        = streamkap_destination_snowflake.test.id
		name      = streamkap_destination_snowflake.test.name
		connector = streamkap_destination_snowflake.test.connector
	}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_pipeline.test", "name", "test-pipeline"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_pipeline.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + pipelineSrcPostgreSQLResourceDef + pipelineDestSnowflakeResourceDef + `
resource "streamkap_pipeline" "test" {
	name                             = "test-pipeline-updated"
	source = {
		id        = streamkap_source_postgresql.test.id
		name      = streamkap_source_postgresql.test.name
		connector = streamkap_source_postgresql.test.connector
		topics    = ["public.users"]
		}
	destination = {
		id        = streamkap_destination_snowflake.test.id
		name      = streamkap_destination_snowflake.test.name
		connector = streamkap_destination_snowflake.test.connector
	}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_pipeline.test", "name", "test-pipeline-updated"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
