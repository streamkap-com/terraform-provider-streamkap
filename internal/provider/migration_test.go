package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// Migration tests create state with v2.2.0, apply the documented v3 configuration,
// and verify that resource IDs survive and the refreshed plan converges.

func migrationResourceIDCheck(address string) resource.TestCheckFunc {
	var legacyID string
	return resource.TestCheckResourceAttrWith(address, "id", func(id string) error {
		if id == "" {
			return fmt.Errorf("%s: empty resource ID", address)
		}
		if legacyID == "" {
			legacyID = id
		} else if id != legacyID {
			return fmt.Errorf("%s: resource ID changed during migration", address)
		}
		return nil
	})
}

func TestAccSourcePostgreSQL_MigrationFromLegacy(t *testing.T) {
	sourcePostgreSQLHostnameMigration := os.Getenv("TF_VAR_source_postgresql_hostname")
	sourcePostgreSQLPasswordMigration := os.Getenv("TF_VAR_source_postgresql_password")
	if sourcePostgreSQLHostnameMigration == "" || sourcePostgreSQLPasswordMigration == "" {
		t.Skip("TF_VAR_source_postgresql_hostname and TF_VAR_source_postgresql_password must be set")
	}

	name := acctestName(t, "migration")
	slotName := fmt.Sprintf("tf_migration_%d", time.Now().UnixNano())
	config := providerConfig + fmt.Sprintf(`
variable "source_postgresql_hostname" {
	type = string
}
variable "source_postgresql_password" {
	type      = string
	sensitive = true
}
resource "streamkap_source_postgresql" "migration_test" {
	name                                         = %q
	database_hostname                            = var.source_postgresql_hostname
	database_port                                = 5432
	database_user                                = "postgresql"
	database_password                            = var.source_postgresql_password
	database_dbname                              = "postgres"
	database_sslmode                             = "require"
	schema_include_list                          = "streamkap"
	table_include_list                           = "streamkap.customer"
	signal_data_collection_schema_or_database    = "streamkap.streamkap_signal"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	slot_name                                    = %q
	publication_name                             = %q
	ssh_enabled                                  = false

	# Deprecated v2 aliases (v3: transforms_insert_static_key1_static_field /
	# ..._static_value). Set here so the empty-plan check covers the alias echo.
	insert_static_key_field_1                    = "tf_migration_key_field"
	insert_static_key_value_1                    = "tf_migration_key_value"
}
`, name, slotName, slotName+"_pub")

	idCheck := migrationResourceIDCheck("streamkap_source_postgresql.migration_test")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Step 1: Create with stable provider (v2.2.0)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					idCheck,
					resource.TestCheckResourceAttr("streamkap_source_postgresql.migration_test", "name", name),
					resource.TestCheckResourceAttr("streamkap_source_postgresql.migration_test", "insert_static_key_field_1", "tf_migration_key_field"),
					resource.TestCheckResourceAttrSet("streamkap_source_postgresql.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				Check:                    idCheck,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			// Step 3: Verify update works with NEW provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   strings.Replace(config, fmt.Sprintf("%q", name), fmt.Sprintf("%q", name+"-updated"), 1),
				Check: resource.ComposeTestCheckFunc(
					idCheck,
					resource.TestCheckResourceAttr("streamkap_source_postgresql.migration_test", "name", name+"-updated"),
					// The alias must survive an update applied by the NEW provider.
					resource.TestCheckResourceAttr("streamkap_source_postgresql.migration_test", "insert_static_key_field_1", "tf_migration_key_field"),
				),
			},
		},
	})
}

func TestAccDestinationSnowflake_MigrationFromLegacy(t *testing.T) {
	sfURL := os.Getenv("TF_VAR_destination_snowflake_url_name")
	sfPrivateKey := os.Getenv("TF_VAR_destination_snowflake_private_key")

	if sfURL == "" || sfPrivateKey == "" {
		t.Skip("TF_VAR_destination_snowflake_url_name and TF_VAR_destination_snowflake_private_key must be set")
	}

	name := acctestName(t, "migration")
	config := providerConfig + fmt.Sprintf(`
variable "destination_snowflake_url_name" { type = string }
variable "destination_snowflake_private_key" {
  type      = string
  sensitive = true
}
variable "destination_snowflake_key_passphrase" {
  type      = string
  sensitive = true
  default   = ""
}

resource "streamkap_destination_snowflake" "migration_test" {
	name                             = %q
	snowflake_url_name               = var.destination_snowflake_url_name
	snowflake_user_name              = "STREAMKAP_USER_JUNIT"
	snowflake_private_key            = var.destination_snowflake_private_key
	snowflake_private_key_passphrase = var.destination_snowflake_key_passphrase
	sfwarehouse                      = "STREAMKAP_WH"
	hard_delete                      = false
	snowflake_database_name          = "JUNIT"
	snowflake_schema_name            = "JUNIT"
	snowflake_role_name              = "STREAMKAP_ROLE_JUNIT"

	# Deprecated v2 alias (v3: create_schema_auto).
	auto_schema_creation             = true
}
`, name)

	idCheck := migrationResourceIDCheck("streamkap_destination_snowflake.migration_test")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Step 1: Create with OLD provider
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					idCheck,
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.migration_test", "name", name),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.migration_test", "auto_schema_creation", "true"),
				),
			},
			// Apply the v3 configuration without replacing the existing resource.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				Check:                    idCheck,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("streamkap_destination_snowflake.migration_test", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

func TestAccPipeline_MigrationFromLegacy(t *testing.T) {
	for _, key := range []string{
		"TF_VAR_source_postgresql_hostname", "TF_VAR_source_postgresql_password",
		"TF_VAR_destination_snowflake_url_name", "TF_VAR_destination_snowflake_private_key",
	} {
		if os.Getenv(key) == "" {
			t.Skipf("%s must be set", key)
		}
	}
	name := acctestName(t, "pipeline-migration")
	connectors := pipelineSrcPostgreSQLResourceDef(acctestName(t, "source")) +
		pipelineDestSnowflakeResourceDef(acctestName(t, "destination"))
	slotName := fmt.Sprintf("tf_migration_%d", time.Now().UnixNano())
	connectors = strings.NewReplacer(
		"terraform_timeout_test_slot", slotName,
		"terraform_timeout_test_pub", slotName+"_pub",
	).Replace(connectors)
	config := providerConfig + connectors + fmt.Sprintf(`
resource "streamkap_pipeline" "migration_test" {
  name = %q
  snapshot_new_tables = false
  source = {
    id = streamkap_source_postgresql.test.id
    name = streamkap_source_postgresql.test.name
    connector = streamkap_source_postgresql.test.connector
    topics = ["streamkap.customer"]
  }
  destination = {
    id = streamkap_destination_snowflake.test.id
    name = streamkap_destination_snowflake.test.name
    connector = streamkap_destination_snowflake.test.connector
  }
}
`, name)
	pipelineID := migrationResourceIDCheck("streamkap_pipeline.migration_test")
	sourceID := migrationResourceIDCheck("streamkap_source_postgresql.test")
	destinationID := migrationResourceIDCheck("streamkap_destination_snowflake.test")
	checkIDs := resource.ComposeTestCheckFunc(pipelineID, sourceID, destinationID)
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckPipelineDestroy, testAccCheckSourceDestroy, testAccCheckDestinationDestroy,
		),
		Steps: []resource.TestStep{
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check:             checkIDs,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				Check:                    checkIDs,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("streamkap_pipeline.migration_test", plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("streamkap_source_postgresql.test", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("streamkap_destination_snowflake.test", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
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

	name := acctestName(t, "migration")
	config := providerConfig + fmt.Sprintf(`
variable "source_mysql_hostname" { type = string }
variable "source_mysql_password" {
  type      = string
  sensitive = true
}

resource "streamkap_source_mysql" "migration_test" {
	name                                         = %q
	database_hostname                            = var.source_mysql_hostname
	database_port                                = 3306
	database_user                                = "streamkap"
	database_password                            = var.source_mysql_password
	database_include_list                        = "streamkap"
	table_include_list                           = "streamkap.customer"
	signal_data_collection_schema_or_database    = "streamkap.streamkap_signal"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	ssh_enabled                                  = false

	# Deprecated v2 aliases (v3: transforms_insert_static_key1_static_field /
	# ..._static_value, database_connection_time_zone).
	insert_static_key_field_1                    = "tf_migration_key_field"
	insert_static_key_value_1                    = "tf_migration_key_value"
	database_connection_timezone                 = "SERVER"
}
`, name)

	idCheck := migrationResourceIDCheck("streamkap_source_mysql.migration_test")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Step 1: Create with stable provider (v2.2.0)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					idCheck,
					resource.TestCheckResourceAttr("streamkap_source_mysql.migration_test", "name", name),
					resource.TestCheckResourceAttr("streamkap_source_mysql.migration_test", "database_connection_timezone", "SERVER"),
					resource.TestCheckResourceAttrSet("streamkap_source_mysql.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				Check:                    idCheck,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
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

	name := acctestName(t, "migration")
	config := providerConfig + fmt.Sprintf(`
variable "source_mongodb_connection_string" {
  type      = string
  sensitive = true
}

resource "streamkap_source_mongodb" "migration_test" {
	name                                      = %q
	mongodb_connection_string                 = var.source_mongodb_connection_string
	database_include_list                     = "streamkap"
	collection_include_list                   = "streamkap.customer"
	signal_data_collection_schema_or_database = "streamkap.streamkap_signal"
	ssh_enabled                               = false

	# Deprecated v2 aliases (v3: transforms_insert_static_key1_static_field /
	# ..._static_value, transforms_unwrap_array_encoding).
	insert_static_key_field_1                 = "tf_migration_key_field"
	insert_static_key_value_1                 = "tf_migration_key_value"
	array_encoding                            = "array_string"
}
`, name)

	idCheck := migrationResourceIDCheck("streamkap_source_mongodb.migration_test")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Step 1: Create with stable provider (v2.2.0)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					idCheck,
					resource.TestCheckResourceAttr("streamkap_source_mongodb.migration_test", "name", name),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.migration_test", "array_encoding", "array_string"),
					resource.TestCheckResourceAttrSet("streamkap_source_mongodb.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				Check:                    idCheck,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
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

	name := acctestName(t, "migration")
	config := providerConfig + fmt.Sprintf(`
variable "source_dynamodb_aws_region" { type = string }
variable "source_dynamodb_aws_access_key_id" { type = string }
variable "source_dynamodb_aws_secret_key" {
  type      = string
  sensitive = true
}

resource "streamkap_source_dynamodb" "migration_test" {
	name                             = %q
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
`, name)

	legacyConfig := strings.NewReplacer(
		"\ttable_include_list ", "\ttable_include_list_user_defined ",
	).Replace(config)
	idCheck := migrationResourceIDCheck("streamkap_source_dynamodb.migration_test")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Step 1: Create with stable provider (v2.2.0)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            legacyConfig,
				Check: resource.ComposeTestCheckFunc(
					idCheck,
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.migration_test", "name", name),
					resource.TestCheckResourceAttrSet("streamkap_source_dynamodb.migration_test", "id"),
				),
			},
			// Apply the v3 configuration without replacing the existing resource.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				Check:                    idCheck,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("streamkap_source_dynamodb.migration_test", plancheck.ResourceActionUpdate),
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

	name := acctestName(t, "migration")
	config := providerConfig + fmt.Sprintf(`
variable "source_sqlserver_hostname" { type = string }
variable "source_sqlserver_password" {
  type      = string
  sensitive = true
}

resource "streamkap_source_sqlserver" "migration_test" {
	name                                      = %q
	database_hostname                         = var.source_sqlserver_hostname
	database_port                             = 1433
	database_user                             = "sa"
	database_password                         = var.source_sqlserver_password
	database_names                            = "streamkap"
	heartbeat_enabled                         = false
	schema_include_list                       = "dbo"
	table_include_list                        = "dbo.customer"
	signal_data_collection_schema_or_database = "dbo.streamkap_signal"
	ssh_enabled                               = false

	# Deprecated v2 aliases (v3: transforms_insert_static_key1_static_field /
	# ..._static_value, streamkap_snapshot_parallelism). The unsuffixed
	# insert_static_* names are the v2 spelling; v3 keeps them as aliases onto
	# the _1 API fields.
	insert_static_key_field                   = "tf_migration_key_field"
	insert_static_key_value                   = "tf_migration_key_value"
	snapshot_parallelism                      = 2
}
`, name)

	legacyConfig := strings.NewReplacer(
		"\tdatabase_names ", "\tdatabase_dbname ",
	).Replace(config)
	idCheck := migrationResourceIDCheck("streamkap_source_sqlserver.migration_test")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Step 1: Create with stable provider (v2.2.0)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            legacyConfig,
				Check: resource.ComposeTestCheckFunc(
					idCheck,
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.migration_test", "name", name),
					resource.TestCheckResourceAttr("streamkap_source_sqlserver.migration_test", "snapshot_parallelism", "2"),
					resource.TestCheckResourceAttrSet("streamkap_source_sqlserver.migration_test", "id"),
				),
			},
			// Apply the v3 configuration without replacing the existing resource.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				Check:                    idCheck,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("streamkap_source_sqlserver.migration_test", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

func TestAccSourceKafkaDirect_MigrationFromLegacy(t *testing.T) {
	// KafkaDirect doesn't require external credentials - uses Streamkap's internal Kafka.
	//
	// `kafka_format` is the v2 attribute name and v3's deprecated alias for
	// `format` (they ConflictsWith each other, so only one may be set). The v2
	// provider has no `format` attribute at all, so the alias is the only
	// spelling a shared config can use.
	name := acctestName(t, "migration")
	config := providerConfig + fmt.Sprintf(`
resource "streamkap_source_kafkadirect" "migration_test" {
	name               = %q
	topic_prefix       = "migration-test_"
	kafka_format       = "json"
	schemas_enable     = true
	topic_include_list = "migration-test_topic1, migration-test_topic2"
}
`, name)

	idCheck := migrationResourceIDCheck("streamkap_source_kafkadirect.migration_test")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Step 1: Create with stable provider (v2.2.0)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					idCheck,
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.migration_test", "name", name),
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.migration_test", "kafka_format", "json"),
					resource.TestCheckResourceAttrSet("streamkap_source_kafkadirect.migration_test", "id"),
				),
			},
			// Apply the v3 configuration without replacing the existing resource.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				Check:                    idCheck,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("streamkap_source_kafkadirect.migration_test", plancheck.ResourceActionUpdate),
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

	name := acctestName(t, "migration")
	config := providerConfig + fmt.Sprintf(`
variable "destination_clickhouse_hostname" { type = string }
variable "destination_clickhouse_connection_username" { type = string }
variable "destination_clickhouse_connection_password" {
  type      = string
  sensitive = true
}

resource "streamkap_destination_clickhouse" "migration_test" {
	name                = %q
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
`, name)

	idCheck := migrationResourceIDCheck("streamkap_destination_clickhouse.migration_test")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Step 1: Create with stable provider (v2.2.0)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					idCheck,
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.migration_test", "name", name),
					resource.TestCheckResourceAttrSet("streamkap_destination_clickhouse.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				Check:                    idCheck,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
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

	name := acctestName(t, "migration")
	config := providerConfig + fmt.Sprintf(`
variable "destination_databricks_connection_url" { type = string }
variable "destination_databricks_token" {
  type      = string
  sensitive = true
}

resource "streamkap_destination_databricks" "migration_test" {
	name              = %q
	connection_url    = var.destination_databricks_connection_url
	databricks_token  = var.destination_databricks_token
	table_name_prefix = "streamkap"
	ingestion_mode    = "upsert"
	partition_mode    = "by_topic"
	hard_delete       = true
	tasks_max         = 3
	schema_evolution  = "basic"
}
`, name)

	idCheck := migrationResourceIDCheck("streamkap_destination_databricks.migration_test")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Step 1: Create with stable provider (v2.2.0)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					idCheck,
					resource.TestCheckResourceAttr("streamkap_destination_databricks.migration_test", "name", name),
					resource.TestCheckResourceAttrSet("streamkap_destination_databricks.migration_test", "id"),
				),
			},
			// Apply the v3 configuration without replacing the existing resource.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				Check:                    idCheck,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("streamkap_destination_databricks.migration_test", plancheck.ResourceActionUpdate),
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

	name := acctestName(t, "migration")
	config := providerConfig + fmt.Sprintf(`
variable "destination_postgresql_hostname" { type = string }
variable "destination_postgresql_password" {
  type      = string
  sensitive = true
}

resource "streamkap_destination_postgresql" "migration_test" {
	name                = %q
	database_hostname   = var.destination_postgresql_hostname
	database_port       = 5432
	database_database   = "postgres"
	connection_username = "postgresql"
	connection_password = var.destination_postgresql_password
	table_name_prefix   = "streamkap"
	schema_evolution    = "basic"
	insert_mode         = "insert"
	delete_enabled      = false
	ssh_enabled         = false
}
`, name)

	legacyConfig := strings.NewReplacer(
		"\tdatabase_database ", "\tdatabase_dbname ",
		"\tdelete_enabled ", "\thard_delete ",
		"\tconnection_username ", "\tdatabase_username ",
		"\tconnection_password ", "\tdatabase_password ",
		"\ttable_name_prefix ", "\tdatabase_schema_name ",
	).Replace(config)
	idCheck := migrationResourceIDCheck("streamkap_destination_postgresql.migration_test")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Step 1: Create with stable provider (v2.2.0)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            legacyConfig,
				Check: resource.ComposeTestCheckFunc(
					idCheck,
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.migration_test", "name", name),
					resource.TestCheckResourceAttrSet("streamkap_destination_postgresql.migration_test", "id"),
				),
			},
			// Apply the v3 configuration without replacing the existing resource.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				Check:                    idCheck,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("streamkap_destination_postgresql.migration_test", plancheck.ResourceActionUpdate),
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

	name := acctestName(t, "migration")
	config := providerConfig + fmt.Sprintf(`
variable "s3_aws_access_key" { type = string }
variable "s3_aws_secret_key" {
  type      = string
  sensitive = true
}

resource "streamkap_destination_s3" "migration_test" {
	name                  = %q
	aws_access_key_id     = var.s3_aws_access_key
	aws_secret_access_key = var.s3_aws_secret_key
	aws_s3_region         = "us-west-2"
	aws_s3_bucket_name    = "migration-test-bucket"
	format                = "JSON Array"
}
`, name)

	legacyConfig := strings.NewReplacer(
		"\taws_access_key_id ", "\taws_access_key ",
		"\taws_secret_access_key ", "\taws_secret_key ",
		"\taws_s3_region ", "\taws_region ",
		"\taws_s3_bucket_name ", "\tbucket_name ",
	).Replace(config)
	idCheck := migrationResourceIDCheck("streamkap_destination_s3.migration_test")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Step 1: Create with stable provider (v2.2.0)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            legacyConfig,
				Check: resource.ComposeTestCheckFunc(
					idCheck,
					resource.TestCheckResourceAttr("streamkap_destination_s3.migration_test", "name", name),
					resource.TestCheckResourceAttrSet("streamkap_destination_s3.migration_test", "id"),
				),
			},
			// Step 2: Switch to NEW provider - MUST produce empty plan
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				Check:                    idCheck,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
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

	name := acctestName(t, "migration")
	config := providerConfig + fmt.Sprintf(`
variable "iceberg_aws_access_key" { type = string }
variable "iceberg_aws_secret_key" {
  type      = string
  sensitive = true
}

resource "streamkap_destination_iceberg" "migration_test" {
	name                                = %q
	iceberg_catalog_type                = "rest"
	iceberg_catalog_s3_credentials_enabled = true
	iceberg_catalog_name                = "migration_test_catalog"
	iceberg_catalog_uri                 = "migration_test_catalog_uri"
	iceberg_catalog_s3_access_key_id    = var.iceberg_aws_access_key
	iceberg_catalog_s3_secret_access_key = var.iceberg_aws_secret_key
	iceberg_catalog_client_region       = "us-west-2"
	iceberg_catalog_warehouse           = "migration_test_bucket_path"
	table_name_prefix                   = "migration_test_schema"
}
`, name)

	legacyConfig := strings.NewReplacer(
		"\ticeberg_catalog_type ", "\tcatalog_type ",
		"\ticeberg_catalog_s3_credentials_enabled = true\n", "",
		"\ticeberg_catalog_name ", "\tcatalog_name ",
		"\ticeberg_catalog_uri ", "\tcatalog_uri ",
		"\ticeberg_catalog_s3_access_key_id ", "\taws_access_key ",
		"\ticeberg_catalog_s3_secret_access_key ", "\taws_secret_key ",
		"\ticeberg_catalog_client_region ", "\taws_region ",
		"\ticeberg_catalog_warehouse ", "\tbucket_path ",
		"\ttable_name_prefix ", "\tschema ",
	).Replace(config)
	idCheck := migrationResourceIDCheck("streamkap_destination_iceberg.migration_test")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Step 1: Create with stable provider (v2.2.0)
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            legacyConfig,
				Check: resource.ComposeTestCheckFunc(
					idCheck,
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.migration_test", "name", name),
					resource.TestCheckResourceAttrSet("streamkap_destination_iceberg.migration_test", "id"),
				),
			},
			// Apply the v3 configuration without replacing the existing resource.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				Check:                    idCheck,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("streamkap_destination_iceberg.migration_test", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

func TestAccDestinationKafka_MigrationFromLegacy(t *testing.T) {
	if os.Getenv("TF_VAR_destination_kafka_bootstrap_servers") == "" {
		t.Skip("TF_VAR_destination_kafka_bootstrap_servers must be set")
	}
	name := acctestName(t, "migration")
	config := providerConfig + fmt.Sprintf(`
variable "destination_kafka_bootstrap_servers" { type = string }
resource "streamkap_destination_kafka" "migration_test" {
  name = %q
  kafka_sink_bootstrap = var.destination_kafka_bootstrap_servers
  destination_format = "json"
  json_schema_enable = false
}
`, name)
	idCheck := migrationResourceIDCheck("streamkap_destination_kafka.migration_test")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: legacyProviderConfig(),
				Config:            config,
				Check:             idCheck,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				Check:                    idCheck,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("streamkap_destination_kafka.migration_test", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
