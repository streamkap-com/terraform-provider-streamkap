# Complete MariaDB CDC source configuration
# This example shows all available configuration options for capturing changes
# from MariaDB tables using binary log replication

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

variable "source_mariadb_hostname" {
  type        = string
  description = "The hostname of the MariaDB database"
}

variable "source_mariadb_password" {
  type        = string
  sensitive   = true
  description = "The password of the MariaDB database"
}

resource "streamkap_source_mariadb" "example-source-mariadb" {
  # Display name for this source in Streamkap UI
  name = "example-source-mariadb"

  # Connection settings
  database_hostname = var.source_mariadb_hostname
  database_port     = "3306"
  database_user     = "streamkap_user"
  database_password = var.source_mariadb_password

  # Database and table selection
  database_include_list = "ecommerce,analytics"
  table_include_list    = "ecommerce.orders,ecommerce.customers,analytics.events"

  # Signal table for incremental snapshots (optional)
  signal_data_collection_schema_or_database = "streamkap"

  # Heartbeat configuration
  heartbeat_enabled                            = true
  heartbeat_data_collection_schema_or_database = "streamkap"

  # Timezone configuration
  database_connection_time_zone = "SERVER" # Options: SERVER, UTC, or specific timezone

  # GTID mode for read-only connections
  snapshot_gtid = "Yes" # MariaDB has GTID enabled by default

  # Schema history optimization
  schema_history_internal_store_only_captured_databases_ddl = false
  schema_history_internal_store_only_captured_tables_ddl    = false

  # Binary data handling
  binary_handling_mode = "bytes" # Options: bytes, base64, base64-url-safe, hex

  # SSL/TLS mode
  database_ssl_mode = "required" # Options: required, disabled

  # Column filtering (optional)
  column_exclude_list = "ecommerce.customers.ssn"

  # SSH tunnel settings (optional)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host = "bastion.example.com"
  # ssh_port = "22"
  # ssh_user = "streamkap"
}

output "example-source-mariadb" {
  value = streamkap_source_mariadb.example-source-mariadb.id
}
