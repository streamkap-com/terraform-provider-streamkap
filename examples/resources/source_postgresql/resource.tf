terraform {
  required_providers {
    streamkap = {
      source = "github.com/streamkap-com/streamkap"
    }
  }
  required_version = ">= 1.0"
}

provider "streamkap" {}

resource "streamkap_source_postgresql" "example-source-postgresql" {
  name                                      = "example-source-postgresql"
  database_hostname                         = "sandbox-postgresql.cluster-cfpneamopmi0.us-west-2.rds.amazonaws.com"
  database_port                             = 5432
  database_user                             = "postgresql"
  database_password                         = "sy_h$j#fxZZVg0CZ<X)Y(JC.F%Gd"
  database_dbname                           = "postgres"
  database_sslmode                          = "require"
  schema_include_list                       = "public"
  table_include_list                        = "public.users"
  signal_data_collection_schema_or_database = "streamkap"
  heartbeat_enabled                         = false
  include_source_db_name_in_table_name      = false
  slot_name                                 = "streamkap_pgoutput_slot"
  publication_name                          = "streamkap_pub"
  binary_handling_mode                      = "bytes"
  ssh_enabled                               = false
}

output "example-source-postgresql" {
  value = streamkap_source_postgresql.example-source-postgresql.id
}