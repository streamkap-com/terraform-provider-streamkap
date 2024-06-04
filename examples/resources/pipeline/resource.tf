terraform {
  required_providers {
    streamkap = {
      source = "github.com/streamkap-com/streamkap"
    }
  }
  required_version = ">= 1.0"
}

provider "streamkap" {}

resource "streamkap_source_postgresql" "example" {
  name                                         = "example-source-postgresql"
  database_hostname                            = ""
  database_port                                = 5432
  database_user                                = "postgresql"
  database_password                            = ""
  database_dbname                              = "postgres"
  database_sslmode                             = "require"
  schema_include_list                          = "public"
  table_include_list                           = "public.users"
  signal_data_collection_schema_or_database    = "streamkap"
  heartbeat_enabled                            = false
  heartbeat_data_collection_schema_or_database = null
  include_source_db_name_in_table_name         = false
  slot_name                                    = "streamkap_pgoutput_slot"
  publication_name                             = "streamkap_pub"
  binary_handling_mode                         = "bytes"
  ssh_enabled                                  = false
}

resource "streamkap_destination_snowflake" "example" {
  name                             = "example-destination-snowflake"
  snowflake_url_name               = ""
  snowflake_user_name              = "STREAMKAP_USER_POSTGRESQL"
  snowflake_private_key            = ""
  snowflake_private_key_passphrase = ""
  snowflake_database_name          = "STREAMKAP_POSTGRESQL"
  snowflake_schema_name            = "STREAMKAP"
  snowflake_role_name              = "STREAMKAP_ROLE"
}

resource "streamkap_pipeline" "example-pipeline" {
  name = "example-pipeline"
  source = {
    id        = streamkap_source_postgresql.example.id
    name      = streamkap_source_postgresql.example.name
    connector = streamkap_source_postgresql.example.connector
    topics    = ["public.users"]
  }
  destination = {
    id        = streamkap_destination_snowflake.example.id
    name      = streamkap_destination_snowflake.example.name
    connector = streamkap_destination_snowflake.example.connector
  }
}

output "example-pipeline" {
  value = streamkap_pipeline.example-pipeline.id
}