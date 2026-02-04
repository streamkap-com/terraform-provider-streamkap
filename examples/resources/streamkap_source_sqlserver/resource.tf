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

variable "source_sqlserver_hostname" {
  type        = string
  description = "The hostname of the SQLServer database"
}
variable "source_sqlserver_password" {
  type        = string
  sensitive   = true
  description = "The password of the SQLServer database"
}

resource "streamkap_source_sqlserver" "example-source-sqlserver" {
  name                                         = "example-source-sqlserver"
  database_hostname                            = var.source_sqlserver_hostname
  database_port                                = 1433
  database_user                                = "admin"
  database_password                            = var.source_sqlserver_password
  database_dbname                              = "sqlserverdemo"
  schema_include_list                          = "dbo"
  table_include_list                           = "dbo.Orders"
  signal_data_collection_schema_or_database    = "dbo.streamkap_signal"
  heartbeat_enabled                            = false
  heartbeat_data_collection_schema_or_database = null
  binary_handling_mode                         = "bytes"
  ssh_enabled                                  = false
  insert_static_key_field                      = "key_field"
  insert_static_key_value                      = "key_value"
  insert_static_value_field                    = "value_field"
  insert_static_value                          = "value_value"
  snapshot_parallelism                         = 2
  snapshot_large_table_threshold               = 12000
  snapshot_custom_table_config = {
    "dbo.Orders" = {
      chunks = 2
    }
  }

}

output "example-source-sqlserver" {
  value = streamkap_source_sqlserver.example-source-sqlserver.id
}