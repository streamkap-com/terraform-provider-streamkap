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
  signal_data_collection_schema_or_database    = "streamkap.streamkap_signal"
  column_include_list                          = "streamkap[.]customer[.](id|name)"
  heartbeat_enabled                            = false
  heartbeat_data_collection_schema_or_database = null
  include_source_db_name_in_table_name         = false
  slot_name                                    = "terraform_pgoutput_slot"
  publication_name                             = "terraform_pub"
  binary_handling_mode                         = "bytes"
  ssh_enabled                                  = false
}

output "example-source-postgresql" {
  value = streamkap_source_postgresql.example-source-postgresql.id
}