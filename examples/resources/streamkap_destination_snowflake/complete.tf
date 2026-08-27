terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
  required_version = ">= 1.5.0"
}

provider "streamkap" {}

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
resource "streamkap_destination_snowflake" "example-destination-snowflake" {
  name                             = "example-destination-snowflake"
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
  snowflake_topic2table_map        = "REGEX_MATCHER>^([-\\w]+\\.)([-\\w]+\\.)?([-\\w]+\\.)?([-\\w]+\\.)?([-\\w]+):$5"
  auto_qa_dedupe_table_mapping = {
    users                   = "JUNIT.USERS",
    itst_scen20240528103635 = "ITST_SCEN20240528103635"
  }
}

output "example-destination-snowflake" {
  value = streamkap_destination_snowflake.example-destination-snowflake.id
}

# Append mode destination writing into a pre-existing Snowflake-managed Iceberg table.
# The Iceberg table must already exist in Snowflake, and the connector's Snowflake
# role must hold ALTER/schema-evolution privilege on it.
resource "streamkap_destination_snowflake" "example-destination-snowflake-iceberg" {
  name                                = "example-destination-snowflake-iceberg"
  snowflake_url_name                  = var.destination_snowflake_url_name
  snowflake_user_name                 = "STREAMKAP_USER_JUNIT"
  snowflake_private_key               = var.destination_snowflake_private_key
  snowflake_private_key_passphrase    = var.destination_snowflake_key_passphrase
  sfwarehouse                         = "STREAMKAP_WH"
  snowflake_database_name             = "JUNIT"
  snowflake_schema_name               = "JUNIT"
  snowflake_role_name                 = "STREAMKAP_ROLE_JUNIT"
  ingestion_mode                      = "append"
  apply_dynamic_table_script          = false
  snowflake_streaming_iceberg_enabled = true
}

output "example-destination-snowflake-iceberg" {
  value = streamkap_destination_snowflake.example-destination-snowflake-iceberg.id
}
