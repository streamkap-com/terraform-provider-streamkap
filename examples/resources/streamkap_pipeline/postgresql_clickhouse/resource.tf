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
  sensitive   = true
  description = "The password to connect to the Clickhouse server"
}

resource "streamkap_destination_clickhouse" "example-destination-clickhouse" {
  name                = "example-destination-clickhouse"
  ingestion_mode      = "append"
  tasks_max           = 5
  hostname            = var.destination_clickhouse_hostname
  connection_username = var.destination_clickhouse_connection_username
  connection_password = var.destination_clickhouse_connection_password
  port                = 8443
  database            = "demo"
  ssl                 = true
  topics_config_map = {
    "public.users" = {
      delete_sql_execute = "SELECT 1;"
    },
    "public.itst_scen20240528100603" = {
      delete_sql_execute = "SELECT 1;"
    },
  }
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
    id        = streamkap_destination_clickhouse.example-destination-clickhouse.id
    name      = streamkap_destination_clickhouse.example-destination-clickhouse.name
    connector = streamkap_destination_clickhouse.example-destination-clickhouse.connector
  }
}

output "example-pipeline" {
  value = streamkap_pipeline.example-pipeline.id
}