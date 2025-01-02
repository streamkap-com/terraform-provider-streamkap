terraform {
  required_providers {
    streamkap = {
      source = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

data "streamkap_tag" "development-tag" {
  id = "670e5ca40afe1d3983ce0c22" # Development tag
}

data "streamkap_tag" "production-tag" {
  id = "670e5bab0d119c0d1f8cda9d" # Production tag
}

variable "source_postgresql_hostname" {
  type        = string
  description = "The hostname of the PostgreSQL database"
}

variable "source_postgresql_password" {
  type        = string
  sensitive   = true
  description = "The password of the PostgreSQL database"
}

resource "streamkap_source_postgresql" "example-source-postgresql" {
  name                                      = "example-source-postgresql"
  database_hostname                         = var.source_postgresql_hostname
  database_port                             = 5432
  database_user                             = "postgresql"
  database_password                         = var.source_postgresql_password
  database_dbname                           = "postgres"
  database_sslmode                          = "require"
  schema_include_list                       = "public"
  table_include_list                        = "public.users,public.itst_scen20240528100603,public.itst_scen20240528103635,public.itst_scen20240530141046"
  signal_data_collection_schema_or_database = "streamkap"
  heartbeat_enabled                         = false
  include_source_db_name_in_table_name      = false
  slot_name                                 = "terraform_pgoutput_slot"
  publication_name                          = "terraform_pub"
  binary_handling_mode                      = "bytes"
  ssh_enabled                               = false
}

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
  ingestion_mode                   = "append"
  hard_delete                      = true
  use_hybrid_tables                = false
  apply_dynamic_table_script       = false
  dynamic_table_target_lag         = 15
  cleanup_task_schedule            = 60
  dedupe_table_mapping = {
    users                   = "JUNIT.USERS",
    itst_scen20240528103635 = "ITST_SCEN20240528103635"
  }
}

data "streamkap_transform" "example-transform" {
  id = "63975020676fa8f369d55001"
}

data "streamkap_transform" "another-example-transform" {
  id = "63975020676fa8f369d55005"
}

resource "streamkap_pipeline" "example-pipeline" {
  name                = "example-pipeline"
  snapshot_new_tables = true
  source = {
    id        = streamkap_source_postgresql.example-source-postgresql.id
    name      = streamkap_source_postgresql.example-source-postgresql.name
    connector = streamkap_source_postgresql.example-source-postgresql.connector
    topics = [
      "public.itst_scen20240530141046",
      "public.itst_scen20240528100603",
      "public.itst_scen20240528103635",
      "public.users",
    ]
  }
  destination = {
    id        = streamkap_destination_snowflake.example-destination-snowflake.id
    name      = streamkap_destination_snowflake.example-destination-snowflake.name
    connector = streamkap_destination_snowflake.example-destination-snowflake.connector
  }
  transforms = [
    {
      id = data.streamkap_transform.example-transform.id
      topics = [
        "public.itst_scen20240530123456",
        "random_topic",
      ]
    },
    {
      id = data.streamkap_transform.another-example-transform.id
      topics = [
        "public.itst_scen20240530654321",
        "public.itst_scen20240528121212",
      ]
    }
  ]
  tags = [
    data.streamkap_tag.development-tag.id,
  ]
}

output "example-pipeline" {
  value = streamkap_pipeline.example-pipeline.id
}