terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
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
  name                                         = "example-source-postgresql"
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
  hard_delete         = true
  tasks_max           = 5
  hostname            = var.destination_clickhouse_hostname
  connection_username = var.destination_clickhouse_connection_username
  connection_password = var.destination_clickhouse_connection_password
  port                = 8443
  database            = "demo"
  ssl                 = true
  topics_config_map = {
    "streamkap.customer" = {
      delete_sql_execute = "SELECT 1;"
    }
  }
  schema_evolution = "basic"
}

resource "streamkap_pipeline" "example-pipeline" {
  name                = "example-pipeline"
  snapshot_new_tables = true
  source = {
    id        = streamkap_source_postgresql.example-source-postgresql.id
    name      = streamkap_source_postgresql.example-source-postgresql.name
    connector = streamkap_source_postgresql.example-source-postgresql.connector
    topics = [
      "streamkap.customer",
      "streamkap.customer2",
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