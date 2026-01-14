package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// Migration tests validate behavioral equivalence between OLD provider (v2.1.18)
// and NEW provider (this branch).
//
// Pattern:
// 1. Create resource with OLD provider (ExternalProviders)
// 2. Switch to NEW provider (ProtoV6ProviderFactories)
// 3. Assert terraform plan shows NO changes (ExpectEmptyPlan)
//
// If the plan is not empty, the new provider has a behavioral difference!
//
// TEMPORARY: Delete this entire file after v3.0.0 release is validated.

var sourcePostgreSQLHostnameMigration = os.Getenv("TF_VAR_source_postgresql_hostname")
var sourcePostgreSQLPasswordMigration = os.Getenv("TF_VAR_source_postgresql_password")

func TestAccSourcePostgreSQL_MigrationFromLegacy(t *testing.T) {
	if sourcePostgreSQLHostnameMigration == "" || sourcePostgreSQLPasswordMigration == "" {
		t.Skip("TF_VAR_source_postgresql_hostname and TF_VAR_source_postgresql_password must be set")
	}

	config := providerConfig + `
variable "source_postgresql_hostname" {
	type = string
}
variable "source_postgresql_password" {
	type      = string
	sensitive = true
}
resource "streamkap_source_postgresql" "migration_test" {
	name                                         = "tf-migration-test-postgresql"
	database_hostname                            = var.source_postgresql_hostname
	database_port                                = "5432"
	database_user                                = "postgresql"
	database_password                            = var.source_postgresql_password
	database_dbname                              = "postgres"
	database_sslmode                             = "require"
	schema_include_list                          = "streamkap"
	table_include_list                           = "streamkap.customer"
	signal_data_collection_schema_or_database    = "streamkap"
	heartbeat_data_collection_schema_or_database = "streamkap"
	slot_name                                    = "tf_migration_slot"
	publication_name                             = "tf_migration_pub"
	ssh_enabled                                  = false
}
`

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Create with OLD provider (v2.1.18)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_postgresql.migration_test", "name", "tf-migration-test-postgresql"),
					resource.TestCheckResourceAttrSet("streamkap_source_postgresql.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			// Step 3: Verify update works with NEW provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: providerConfig + `
variable "source_postgresql_hostname" {
	type = string
}
variable "source_postgresql_password" {
	type      = string
	sensitive = true
}
resource "streamkap_source_postgresql" "migration_test" {
	name                                         = "tf-migration-test-postgresql-updated"
	database_hostname                            = var.source_postgresql_hostname
	database_port                                = "5432"
	database_user                                = "postgresql"
	database_password                            = var.source_postgresql_password
	database_dbname                              = "postgres"
	database_sslmode                             = "require"
	schema_include_list                          = "streamkap"
	table_include_list                           = "streamkap.customer"
	signal_data_collection_schema_or_database    = "streamkap"
	heartbeat_data_collection_schema_or_database = "streamkap"
	slot_name                                    = "tf_migration_slot"
	publication_name                             = "tf_migration_pub"
	ssh_enabled                                  = false
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_postgresql.migration_test", "name", "tf-migration-test-postgresql-updated"),
				),
			},
		},
	})
}

func TestAccDestinationSnowflake_MigrationFromLegacy(t *testing.T) {
	// Get required env vars for skip check
	sfURL := os.Getenv("TF_VAR_destination_snowflake_url_name")
	sfPrivateKey := os.Getenv("TF_VAR_destination_snowflake_private_key")
	// Note: passphrase is optional, other fields use test defaults

	if sfURL == "" || sfPrivateKey == "" {
		t.Skip("TF_VAR_destination_snowflake_url_name and TF_VAR_destination_snowflake_private_key must be set")
	}

	// Config matches main branch test patterns - hardcodes test-specific values
	config := providerConfig + `
variable "destination_snowflake_url_name" { type = string }
variable "destination_snowflake_private_key" { type = string; sensitive = true }
variable "destination_snowflake_key_passphrase" { type = string; sensitive = true; default = "" }

resource "streamkap_destination_snowflake" "migration_test" {
	name                             = "tf-migration-test-snowflake"
	snowflake_url_name               = var.destination_snowflake_url_name
	snowflake_user_name              = "STREAMKAP_USER_JUNIT"
	snowflake_private_key            = var.destination_snowflake_private_key
	snowflake_private_key_passphrase = var.destination_snowflake_key_passphrase
	sfwarehouse                      = "STREAMKAP_WH"
	snowflake_database_name          = "JUNIT"
	snowflake_schema_name            = "JUNIT"
	snowflake_role_name              = "STREAMKAP_ROLE_JUNIT"
}
`

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Create with OLD provider
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.migration_test", "name", "tf-migration-test-snowflake"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccPipeline_MigrationFromLegacy(t *testing.T) {
	// Pipeline Migration Strategy:
	// Pipelines have dependencies on source and destination connectors.
	// We test pipeline migration in a single test that:
	// 1. Creates source + destination + pipeline with OLD provider
	// 2. Switches to NEW provider and verifies empty plan for ALL resources
	//
	// This ensures the entire resource graph migrates correctly.

	// Get required env vars for skip check
	// Note: All other vars are passed via TF_VAR_* env vars to terraform
	pgHostname := os.Getenv("TF_VAR_source_postgresql_hostname")
	sfURL := os.Getenv("TF_VAR_destination_snowflake_url_name")

	if pgHostname == "" || sfURL == "" {
		t.Skip("Pipeline migration requires both source and destination env vars")
	}

	config := providerConfig + `
variable "source_postgresql_hostname" { type = string }
variable "source_postgresql_password" { type = string; sensitive = true }
variable "destination_snowflake_url_name" { type = string }
variable "destination_snowflake_user_name" { type = string }
variable "destination_snowflake_private_key" { type = string; sensitive = true }
variable "destination_snowflake_private_key_passphrase" { type = string; sensitive = true; default = "" }
variable "destination_snowflake_database_name" { type = string }
variable "destination_snowflake_schema_name" { type = string }

resource "streamkap_source_postgresql" "pipeline_test" {
	name                                         = "tf-migration-pipeline-source"
	database_hostname                            = var.source_postgresql_hostname
	database_port                                = "5432"
	database_user                                = "postgresql"
	database_password                            = var.source_postgresql_password
	database_dbname                              = "postgres"
	database_sslmode                             = "require"
	schema_include_list                          = "streamkap"
	table_include_list                           = "streamkap.customer"
	signal_data_collection_schema_or_database    = "streamkap"
	heartbeat_data_collection_schema_or_database = "streamkap"
	slot_name                                    = "tf_pipeline_slot"
	publication_name                             = "tf_pipeline_pub"
	ssh_enabled                                  = false
}

resource "streamkap_destination_snowflake" "pipeline_test" {
	name                             = "tf-migration-pipeline-dest"
	snowflake_url_name               = var.destination_snowflake_url_name
	snowflake_user_name              = var.destination_snowflake_user_name
	snowflake_private_key            = var.destination_snowflake_private_key
	snowflake_private_key_passphrase = var.destination_snowflake_private_key_passphrase
	snowflake_database_name          = var.destination_snowflake_database_name
	snowflake_schema_name            = var.destination_snowflake_schema_name
}

resource "streamkap_pipeline" "pipeline_test" {
	name           = "tf-migration-pipeline"
	source_id      = streamkap_source_postgresql.pipeline_test.id
	destination_id = streamkap_destination_snowflake.pipeline_test.id
	# Note: Add snapshot_new_tables and other optional fields as needed
}
`

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Create entire graph with OLD provider
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_postgresql.pipeline_test", "name", "tf-migration-pipeline-source"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.pipeline_test", "name", "tf-migration-pipeline-dest"),
					resource.TestCheckResourceAttr("streamkap_pipeline.pipeline_test", "name", "tf-migration-pipeline"),
					resource.TestCheckResourceAttrSet("streamkap_pipeline.pipeline_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - ALL resources must produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// ============================================================================
// SOURCE MIGRATION TESTS
// ============================================================================

func TestAccSourceMySQL_MigrationFromLegacy(t *testing.T) {
	mysqlHostname := os.Getenv("TF_VAR_source_mysql_hostname")
	mysqlPassword := os.Getenv("TF_VAR_source_mysql_password")

	if mysqlHostname == "" || mysqlPassword == "" {
		t.Skip("TF_VAR_source_mysql_hostname and TF_VAR_source_mysql_password must be set")
	}

	config := providerConfig + `
variable "source_mysql_hostname" { type = string }
variable "source_mysql_password" { type = string; sensitive = true }

resource "streamkap_source_mysql" "migration_test" {
	name                                         = "tf-migration-test-mysql"
	database_hostname                            = var.source_mysql_hostname
	database_port                                = "3306"
	database_user                                = "streamkap"
	database_password                            = var.source_mysql_password
	database_include_list                        = "streamkap"
	table_include_list                           = "streamkap.customer"
	signal_data_collection_schema_or_database    = "streamkap"
	heartbeat_data_collection_schema_or_database = "streamkap"
	ssh_enabled                                  = false
}
`

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Create with OLD provider (v2.1.18)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mysql.migration_test", "name", "tf-migration-test-mysql"),
					resource.TestCheckResourceAttrSet("streamkap_source_mysql.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccSourceMongoDB_MigrationFromLegacy(t *testing.T) {
	mongoConnectionString := os.Getenv("TF_VAR_source_mongodb_connection_string")

	if mongoConnectionString == "" {
		t.Skip("TF_VAR_source_mongodb_connection_string must be set")
	}

	config := providerConfig + `
variable "source_mongodb_connection_string" { type = string; sensitive = true }

resource "streamkap_source_mongodb" "migration_test" {
	name                                      = "tf-migration-test-mongodb"
	mongodb_connection_string                 = var.source_mongodb_connection_string
	database_include_list                     = "streamkap"
	collection_include_list                   = "streamkap.customer"
	signal_data_collection_schema_or_database = "streamkap"
	ssh_enabled                               = false
}
`

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Create with OLD provider (v2.1.18)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mongodb.migration_test", "name", "tf-migration-test-mongodb"),
					resource.TestCheckResourceAttrSet("streamkap_source_mongodb.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccSourceDynamoDB_MigrationFromLegacy(t *testing.T) {
	awsRegion := os.Getenv("TF_VAR_source_dynamodb_aws_region")
	awsAccessKeyID := os.Getenv("TF_VAR_source_dynamodb_aws_access_key_id")
	awsSecretKey := os.Getenv("TF_VAR_source_dynamodb_aws_secret_key")

	if awsRegion == "" || awsAccessKeyID == "" || awsSecretKey == "" {
		t.Skip("DynamoDB environment variables must be set")
	}

	// Config matches main branch test patterns
	config := providerConfig + `
variable "source_dynamodb_aws_region" { type = string }
variable "source_dynamodb_aws_access_key_id" { type = string }
variable "source_dynamodb_aws_secret_key" { type = string; sensitive = true }

resource "streamkap_source_dynamodb" "migration_test" {
	name                             = "tf-migration-test-dynamodb"
	aws_region                       = var.source_dynamodb_aws_region
	aws_access_key_id                = var.source_dynamodb_aws_access_key_id
	aws_secret_key                   = var.source_dynamodb_aws_secret_key
	s3_export_bucket_name            = "streamkap-export"
	table_include_list               = "migration-test-table"
	batch_size                       = 1024
	poll_timeout_ms                  = 1000
	incremental_snapshot_chunk_size  = 32768
	incremental_snapshot_max_threads = 8
	full_export_expiration_time_ms   = 86400000
	signal_kafka_poll_timeout_ms     = 1000
	array_encoding_json              = true
	struct_encoding_json             = true
	tasks_max                        = 3
}
`

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Create with OLD provider (v2.1.18)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.migration_test", "name", "tf-migration-test-dynamodb"),
					resource.TestCheckResourceAttrSet("streamkap_source_dynamodb.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccSourceSQLServer_MigrationFromLegacy(t *testing.T) {
	sqlserverHostname := os.Getenv("TF_VAR_source_sqlserver_hostname")
	sqlserverPassword := os.Getenv("TF_VAR_source_sqlserver_password")

	if sqlserverHostname == "" || sqlserverPassword == "" {
		t.Skip("TF_VAR_source_sqlserver_hostname and TF_VAR_source_sqlserver_password must be set")
	}

	config := providerConfig + `
variable "source_sqlserver_hostname" { type = string }
variable "source_sqlserver_password" { type = string; sensitive = true }

resource "streamkap_source_sqlserver" "migration_test" {
	name                                      = "tf-migration-test-sqlserver"
	database_hostname                         = var.source_sqlserver_hostname
	database_port                             = "1433"
	database_user                             = "sa"
	database_password                         = var.source_sqlserver_password
	database_names                            = "streamkap"
	schema_include_list                       = "dbo"
	table_include_list                        = "dbo.customer"
	signal_data_collection_schema_or_database = "dbo"
	ssh_enabled                               = false
}
`

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Create with OLD provider (v2.1.18)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.migration_test", "name", "tf-migration-test-sqlserver"),
					resource.TestCheckResourceAttrSet("streamkap_source_sqlserver.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccSourceKafkaDirect_MigrationFromLegacy(t *testing.T) {
	// KafkaDirect doesn't require external credentials - uses Streamkap's internal Kafka
	config := providerConfig + `
resource "streamkap_source_kafkadirect" "migration_test" {
	name               = "tf-migration-test-kafkadirect"
	topic_prefix       = "migration-test_"
	format             = "json"
	schemas_enable     = true
	topic_include_list = "migration-test_topic1, migration-test_topic2"
}
`

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Create with OLD provider (v2.1.18)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.migration_test", "name", "tf-migration-test-kafkadirect"),
					resource.TestCheckResourceAttrSet("streamkap_source_kafkadirect.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// ============================================================================
// DESTINATION MIGRATION TESTS
// ============================================================================

func TestAccDestinationClickHouse_MigrationFromLegacy(t *testing.T) {
	clickhouseHostname := os.Getenv("TF_VAR_destination_clickhouse_hostname")
	clickhouseUsername := os.Getenv("TF_VAR_destination_clickhouse_connection_username")
	clickhousePassword := os.Getenv("TF_VAR_destination_clickhouse_connection_password")

	if clickhouseHostname == "" || clickhouseUsername == "" || clickhousePassword == "" {
		t.Skip("ClickHouse environment variables must be set")
	}

	config := providerConfig + `
variable "destination_clickhouse_hostname" { type = string }
variable "destination_clickhouse_connection_username" { type = string }
variable "destination_clickhouse_connection_password" { type = string; sensitive = true }

resource "streamkap_destination_clickhouse" "migration_test" {
	name                = "tf-migration-test-clickhouse"
	hostname            = var.destination_clickhouse_hostname
	connection_username = var.destination_clickhouse_connection_username
	connection_password = var.destination_clickhouse_connection_password
	ingestion_mode      = "upsert"
	hard_delete         = true
	tasks_max           = 3
	port                = 8443
	database            = "demo"
	ssl                 = true
	schema_evolution    = "basic"
}
`

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Create with OLD provider (v2.1.18)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.migration_test", "name", "tf-migration-test-clickhouse"),
					resource.TestCheckResourceAttrSet("streamkap_destination_clickhouse.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccDestinationDatabricks_MigrationFromLegacy(t *testing.T) {
	databricksConnectionUrl := os.Getenv("TF_VAR_destination_databricks_connection_url")
	databricksToken := os.Getenv("TF_VAR_destination_databricks_token")

	if databricksConnectionUrl == "" || databricksToken == "" {
		t.Skip("Databricks environment variables must be set")
	}

	config := providerConfig + `
variable "destination_databricks_connection_url" { type = string }
variable "destination_databricks_token" { type = string; sensitive = true }

resource "streamkap_destination_databricks" "migration_test" {
	name              = "tf-migration-test-databricks"
	connection_url    = var.destination_databricks_connection_url
	databricks_token  = var.destination_databricks_token
	table_name_prefix = "streamkap"
	ingestion_mode    = "upsert"
	partition_mode    = "by_topic"
	hard_delete       = true
	tasks_max         = 3
	schema_evolution  = "basic"
}
`

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Create with OLD provider (v2.1.18)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_databricks.migration_test", "name", "tf-migration-test-databricks"),
					resource.TestCheckResourceAttrSet("streamkap_destination_databricks.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccDestinationPostgreSQL_MigrationFromLegacy(t *testing.T) {
	destPostgresqlHostname := os.Getenv("TF_VAR_destination_postgresql_hostname")
	destPostgresqlPassword := os.Getenv("TF_VAR_destination_postgresql_password")

	if destPostgresqlHostname == "" || destPostgresqlPassword == "" {
		t.Skip("TF_VAR_destination_postgresql_hostname and TF_VAR_destination_postgresql_password must be set")
	}

	config := providerConfig + `
variable "destination_postgresql_hostname" { type = string }
variable "destination_postgresql_password" { type = string; sensitive = true }

resource "streamkap_destination_postgresql" "migration_test" {
	name                = "tf-migration-test-postgresql-dest"
	database_hostname   = var.destination_postgresql_hostname
	database_port       = "5432"
	database_database   = "postgres"
	connection_username = "postgresql"
	connection_password = var.destination_postgresql_password
	table_name_prefix   = "streamkap"
	schema_evolution    = "basic"
	insert_mode         = "insert"
	delete_enabled      = false
	ssh_enabled         = false
}
`

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Create with OLD provider (v2.1.18)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.migration_test", "name", "tf-migration-test-postgresql-dest"),
					resource.TestCheckResourceAttrSet("streamkap_destination_postgresql.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccDestinationS3_MigrationFromLegacy(t *testing.T) {
	s3AwsAccessKey := os.Getenv("TF_VAR_s3_aws_access_key")
	s3AwsSecretKey := os.Getenv("TF_VAR_s3_aws_secret_key")

	if s3AwsAccessKey == "" || s3AwsSecretKey == "" {
		t.Skip("S3 environment variables must be set")
	}

	config := providerConfig + `
variable "s3_aws_access_key" { type = string }
variable "s3_aws_secret_key" { type = string; sensitive = true }

resource "streamkap_destination_s3" "migration_test" {
	name                  = "tf-migration-test-s3"
	aws_access_key_id     = var.s3_aws_access_key
	aws_secret_access_key = var.s3_aws_secret_key
	aws_s3_region         = "us-west-2"
	aws_s3_bucket_name    = "migration-test-bucket"
	format                = "JSON Array"
}
`

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Create with OLD provider (v2.1.18)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_s3.migration_test", "name", "tf-migration-test-s3"),
					resource.TestCheckResourceAttrSet("streamkap_destination_s3.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccDestinationIceberg_MigrationFromLegacy(t *testing.T) {
	icebergAwsAccessKey := os.Getenv("TF_VAR_iceberg_aws_access_key")
	icebergAwsSecretKey := os.Getenv("TF_VAR_iceberg_aws_secret_key")

	if icebergAwsAccessKey == "" || icebergAwsSecretKey == "" {
		t.Skip("Iceberg environment variables must be set")
	}

	config := providerConfig + `
variable "iceberg_aws_access_key" { type = string }
variable "iceberg_aws_secret_key" { type = string; sensitive = true }

resource "streamkap_destination_iceberg" "migration_test" {
	name                                = "tf-migration-test-iceberg"
	iceberg_catalog_type                = "rest"
	iceberg_catalog_name                = "migration_test_catalog"
	iceberg_catalog_uri                 = "migration_test_catalog_uri"
	iceberg_catalog_s3_access_key_id    = var.iceberg_aws_access_key
	iceberg_catalog_s3_secret_access_key = var.iceberg_aws_secret_key
	iceberg_catalog_client_region       = "us-west-2"
	iceberg_catalog_warehouse           = "migration_test_bucket_path"
	table_name_prefix                   = "migration_test_schema"
}
`

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Create with OLD provider (v2.1.18)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.migration_test", "name", "tf-migration-test-iceberg"),
					resource.TestCheckResourceAttrSet("streamkap_destination_iceberg.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// ============================================================================
// OTHER RESOURCE MIGRATION TESTS
// ============================================================================

// NOTE: Topic resource migration test is not implemented.
//
// The Topic resource requires a valid topic_id that is dynamically generated
// when a source connector creates topics. This makes stateless migration testing
// impractical because:
// 1. topic_id format: "source_{source_id}.{schema}.{table}"
// 2. source_id changes with each test run
// 3. Topics are created asynchronously after source connector starts
//
// Topic compatibility is implicitly validated through:
// - Source connector migration tests (same underlying data structures)
// - Pipeline migration tests (pipelines use topics internally)
//
// If explicit Topic migration testing is needed, consider:
// - Using a pre-existing source with known topic_ids in the test environment
// - Creating a multi-step test that captures the topic_id after source creation
