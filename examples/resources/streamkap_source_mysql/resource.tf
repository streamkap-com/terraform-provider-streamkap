terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = "~> 2.2"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

variable "source_mysql_hostname" {
  type        = string
  description = "The hostname of the MySQL database"
}

variable "source_mysql_password" {
  type        = string
  sensitive   = true
  description = "The password of the MySQL database"
}

resource "streamkap_source_mysql" "example-source-mysql" {
  name                                      = "orders-mysql"
  database_hostname                         = var.source_mysql_hostname
  database_port                             = 3306
  database_user                             = "admin"
  database_password                         = var.source_mysql_password
  database_include_list                     = "commerce"
  table_include_list                        = "commerce.orders,commerce.customers"
  signal_data_collection_schema_or_database = "commerce.streamkap_signal"
  database_connection_timezone              = "SERVER"
  snapshot_gtid                             = true
  binary_handling_mode                      = "bytes"
  ssh_enabled                               = false

  # Heartbeat keeps the connector polling on low-traffic sources.
  #   - leave heartbeat_data_collection_schema_or_database unset -> Kafka-only mode (no source-DB write)
  #   - set it to a database containing a streamkap_heartbeat table -> source-table mode
  heartbeat_enabled = true
}

output "example-source-mysql" {
  value = streamkap_source_mysql.example-source-mysql.id
}
