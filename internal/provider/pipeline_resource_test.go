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
	snapshot_read_only                           = "No"
	database_sslmode                             = "require"
	schema_include_list                          = "streamkap"
	table_include_list                           = "streamkap.customer,streamkap.customer2"
	signal_data_collection_schema_or_database    = "streamkap"
	column_include_list                          = "streamkap[.]customer[.](id|name)"
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
	auto_qa_dedupe_table_mapping = {
		users                   = "JUNIT.USERS",
		itst_scen20240528103635 = "ITST_SCEN20240528103635"
	}
}
`

var pipelineTransformsDef = `
data "streamkap_transform" "test-transform" {
	id = "67d43b4ed21e8f093edae34b"
}

data "streamkap_transform" "another-test-transform" {
	id = "67dbe945308e0871a4e1fc49"
}
`

var pipelineTagsDef = `
data "streamkap_tag" "development-tag" {
  id = "670e5ca40afe1d3983ce0c22" # Development tag
}

data "streamkap_tag" "production-tag" {
  id = "670e5bab0d119c0d1f8cda9d" # Production tag
}
`

func TestAccPostgreSQLSnowflakePipelineResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPipelineDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + pipelineSrcPostgreSQLResourceDef + pipelineDestSnowflakeResourceDef + pipelineTransformsDef + pipelineTagsDef + `
resource "streamkap_pipeline" "test" {
	name                = "test-pipeline"
	snapshot_new_tables = true
	source = {
		id        = streamkap_source_postgresql.test.id
		name      = streamkap_source_postgresql.test.name
		connector = streamkap_source_postgresql.test.connector
		topics    = [
			"streamkap.customer",
			"streamkap.customer2",
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
				"public.test_transformed",
			]
		},
		{
			id     = data.streamkap_transform.another-test-transform.id
			topics = [
				"test",
			]
		}
	]
	tags = [
		data.streamkap_tag.production-tag.id,
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
				Config: providerConfig + pipelineSrcPostgreSQLResourceDef + pipelineDestSnowflakeResourceDef + pipelineTransformsDef + pipelineTagsDef + `
resource "streamkap_pipeline" "test" {
	name                = "test-pipeline-updated"
	snapshot_new_tables = true
	source = {
		id        = streamkap_source_postgresql.test.id
		name      = streamkap_source_postgresql.test.name
		connector = streamkap_source_postgresql.test.connector
		topics    = [
			"streamkap.customer",
			"streamkap.customer2",
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
				"public.test_transformed",
			]
		},
	]
	tags = [
		data.streamkap_tag.production-tag.id,
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
	table_include_list               = "warehouse-test-2"
	batch_size                       = 1024
	poll_timeout_ms                  = 1000
	incremental_snapshot_chunk_size  = 32768
	incremental_snapshot_max_threads = 8
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
		CheckDestroy:             testAccCheckPipelineDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + pipelineSrcDynamoDBResourceDef + pipelineDestClickHouseResourceDef + pipelineTransformsDef + pipelineTagsDef + `
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
	tags = [
		data.streamkap_tag.development-tag.id,
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
				Config: providerConfig + pipelineSrcDynamoDBResourceDef + pipelineDestClickHouseResourceDef + pipelineTransformsDef + pipelineTagsDef + `
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
	tags = [
		data.streamkap_tag.development-tag.id,
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
