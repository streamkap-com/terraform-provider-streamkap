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
	name                                      = "tf-migration-test-postgresql"
	database_hostname                         = var.source_postgresql_hostname
	database_port                             = "5432"
	database_user                             = "postgresql"
	database_password                         = var.source_postgresql_password
	database_dbname                           = "postgres"
	database_sslmode                          = "require"
	schema_include_list                       = "streamkap"
	table_include_list                        = "streamkap.customer"
	signal_data_collection_schema_or_database = "streamkap"
	slot_name                                 = "tf_migration_slot"
	publication_name                          = "tf_migration_pub"
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
	name                                      = "tf-migration-test-postgresql-updated"
	database_hostname                         = var.source_postgresql_hostname
	database_port                             = "5432"
	database_user                             = "postgresql"
	database_password                         = var.source_postgresql_password
	database_dbname                           = "postgres"
	database_sslmode                          = "require"
	schema_include_list                       = "streamkap"
	table_include_list                        = "streamkap.customer"
	signal_data_collection_schema_or_database = "streamkap"
	slot_name                                 = "tf_migration_slot"
	publication_name                          = "tf_migration_pub"
	ssh_enabled                               = false
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
	sfUser := os.Getenv("TF_VAR_destination_snowflake_user_name")
	sfPrivateKey := os.Getenv("TF_VAR_destination_snowflake_private_key")
	// Note: sfKeyPassphrase, sfDatabase, sfSchema are passed via TF_VAR_* env vars to terraform

	if sfURL == "" || sfUser == "" || sfPrivateKey == "" {
		t.Skip("Snowflake environment variables must be set")
	}

	config := providerConfig + `
variable "destination_snowflake_url_name" { type = string }
variable "destination_snowflake_user_name" { type = string }
variable "destination_snowflake_private_key" { type = string; sensitive = true }
variable "destination_snowflake_private_key_passphrase" { type = string; sensitive = true; default = "" }
variable "destination_snowflake_database_name" { type = string }
variable "destination_snowflake_schema_name" { type = string }

resource "streamkap_destination_snowflake" "migration_test" {
	name                   = "tf-migration-test-snowflake"
	snowflake_url_name     = var.destination_snowflake_url_name
	snowflake_user_name    = var.destination_snowflake_user_name
	snowflake_private_key  = var.destination_snowflake_private_key
	snowflake_private_key_passphrase = var.destination_snowflake_private_key_passphrase
	snowflake_database_name = var.destination_snowflake_database_name
	snowflake_schema_name  = var.destination_snowflake_schema_name
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
	name                                      = "tf-migration-pipeline-source"
	database_hostname                         = var.source_postgresql_hostname
	database_port                             = "5432"
	database_user                             = "postgresql"
	database_password                         = var.source_postgresql_password
	database_dbname                           = "postgres"
	database_sslmode                          = "require"
	schema_include_list                       = "streamkap"
	table_include_list                        = "streamkap.customer"
	signal_data_collection_schema_or_database = "streamkap"
	slot_name                                 = "tf_pipeline_slot"
	publication_name                          = "tf_pipeline_pub"
	ssh_enabled                               = false
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

// Add more migration tests for each resource type following the same pattern:
// - TestAccSourceMySQL_MigrationFromLegacy
// - TestAccSourceMongoDB_MigrationFromLegacy
// - TestAccDestinationClickHouse_MigrationFromLegacy
// - TestAccTransformMapFilter_MigrationFromLegacy
// etc.
