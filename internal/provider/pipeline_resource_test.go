package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Test PostgreSQL -> Snowflake ----------------------------------------------------------
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

func TestAccPostgreSQLSnowflakePipelineResource(t *testing.T) {
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

// Test DynamoDB -> ClickHouse ----------------------------------------------------------
var pipelineSrcDynamoDBResourceDef = `
variable "source_dynamodb_aws_region" {
	type        = string
	description = "AWS Region"
}

variable "source_dynamodb_aws_access_key_id" {
	type        = string
	description = "AWS Access Key ID"
}

variable "source_dynamodb_aws_secret_key" {
	type        = string
	sensitive   = true
	description = "AWS Secret Key"
}

resource "streamkap_source_dynamodb" "test" {
	name                             = "test-source-dynamodb"
	aws_region                       = var.source_dynamodb_aws_region
	aws_access_key_id                = var.source_dynamodb_aws_access_key_id
	aws_secret_key                   = var.source_dynamodb_aws_secret_key
	s3_export_bucket_name            = "streamkap-export"
	table_include_list_user_defined  = "warehouse-test-2"
	batch_size                       = 1024
	poll_timeout_ms                  = 1000
	incremental_snapshot_chunk_size  = 32768
	incremental_snapshot_max_threads = 8
	incremental_snapshot_interval_ms = 8
	full_export_expiration_time_ms   = 86400000
	signal_kafka_poll_timeout_ms     = 1000
}
`

var pipelineDestClickHouseResourceDef = `
variable "destination_clickhouse_hostname" {
	type        = string
	description = "The hostname of the Clickhouse server"
}

variable "destination_clickhouse_connection_username" {
	type        = string
	description = "The username to connect to the Clickhouse server"
}

variable "destination_clickhouse_connection_password" {
	type        = string
	description = "The password to connect to the Clickhouse server"
}

resource "streamkap_destination_clickhouse" "test" {
	name                = "test-destination-clickhouse"
	ingestion_mode      = "append"
	tasks_max           = 5
	hostname            = var.destination_clickhouse_hostname
	connection_username = var.destination_clickhouse_connection_username
	connection_password = var.destination_clickhouse_connection_password
	port                = 8443
	database            = "demo"
	ssl                 = true
}
`

func TestAccDynamoDBClickHousePipelineResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + pipelineSrcDynamoDBResourceDef + pipelineDestClickHouseResourceDef + pipelineTransformsDef + `
resource "streamkap_pipeline" "test" {
	name                = "test-pipeline"
	snapshot_new_tables = true
	source = {
		id        = streamkap_source_dynamodb.test.id
		name      = streamkap_source_dynamodb.test.name
		connector = streamkap_source_dynamodb.test.connector
		topics    = [
			"default.warehouse-test-2",
		]
	}
	destination = {
		id        = streamkap_destination_clickhouse.test.id
		name      = streamkap_destination_clickhouse.test.name
		connector = streamkap_destination_clickhouse.test.connector
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
				Config: providerConfig + pipelineSrcDynamoDBResourceDef + pipelineDestClickHouseResourceDef + pipelineTransformsDef + `
resource "streamkap_pipeline" "test" {
	name                = "test-pipeline-updated"
	snapshot_new_tables = true
	source = {
		id        = streamkap_source_dynamodb.test.id
		name      = streamkap_source_dynamodb.test.name
		connector = streamkap_source_dynamodb.test.connector
		topics    = [
			"default.warehouse-test-2",
		]
	}
	destination = {
		id        = streamkap_destination_clickhouse.test.id
		name      = streamkap_destination_clickhouse.test.name
		connector = streamkap_destination_clickhouse.test.connector
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
