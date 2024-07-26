package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var pipelineSrcPostgreSQLResourceDef = `
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
	database_port                                = 5432
	database_user                                = "postgresql"
	database_password                            = var.source_postgresql_password
	database_dbname                              = "postgres"
	database_sslmode                             = "require"
	schema_include_list                          = "public"
	table_include_list                           = "public.users"
	signal_data_collection_schema_or_database    = "streamkap"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	include_source_db_name_in_table_name         = false
	slot_name                                    = "terraform_pgoutput_slot"
	publication_name                             = "terraform_pub"
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = false
}
`

var pipelineDestSnowflakeResourceDef = `
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
`

var pipelineTransformsDef = `
data "streamkap_transform" "test-transform" {
	id = "63975020676fa8f369d55001"
}

data "streamkap_transform" "another-test-transform" {
	id = "63975020676fa8f369d55005"
}
`

func TestAccPipelineResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + pipelineSrcPostgreSQLResourceDef + pipelineDestSnowflakeResourceDef + pipelineTransformsDef + `
resource "streamkap_pipeline" "test" {
	name                = "test-pipeline"
	snapshot_new_tables = true
	source = {
		id        = streamkap_source_postgresql.test.id
		name      = streamkap_source_postgresql.test.name
		connector = streamkap_source_postgresql.test.connector
		topics    = [
			"public.users",
			"public.itst_scen20240530141046",
			"public.itst_scen20240528100603",
			"public.itst_scen20240528103635",
		]
	}
	destination = {
		id        = streamkap_destination_snowflake.test.id
		name      = streamkap_destination_snowflake.test.name
		connector = streamkap_destination_snowflake.test.connector
	}
	transforms = [
		{
			id     = data.streamkap_transform.test-transform.id
			topics = [
				"public.itst_scen20240530123456",
				"random_topic",
			]
		},
		{
			id     = data.streamkap_transform.another-test-transform.id
			topics = [
				"public.itst_scen20240530654321",
				"public.itst_scen20240528121212",
			]
		}
	]
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
				Config: providerConfig + pipelineSrcPostgreSQLResourceDef + pipelineDestSnowflakeResourceDef + pipelineTransformsDef + `
resource "streamkap_pipeline" "test" {
	name                = "test-pipeline-updated"
	snapshot_new_tables = true
	source = {
		id        = streamkap_source_postgresql.test.id
		name      = streamkap_source_postgresql.test.name
		connector = streamkap_source_postgresql.test.connector
		topics    = [
			"public.itst_scen20240530141046",
			"public.itst_scen20240528103635",
		]
	}
	destination = {
		id        = streamkap_destination_snowflake.test.id
		name      = streamkap_destination_snowflake.test.name
		connector = streamkap_destination_snowflake.test.connector
	}
	transforms = [
		{
			id     = data.streamkap_transform.test-transform.id
			topics = [
				"public.itst_scen20240530123456",
			]
		},
	]
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
