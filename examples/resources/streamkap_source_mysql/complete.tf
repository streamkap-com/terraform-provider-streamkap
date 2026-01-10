# Complete MySQL CDC source configuration
# This example shows all available configuration options for capturing changes
# from MySQL tables using binary log replication

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
  # Display name for this source in Streamkap UI
  name = "example-source-mysql"

  # Connection settings
  database_hostname = var.source_mysql_hostname # Database server hostname or IP
  database_port     = 3306                      # MySQL port (default: 3306)
  database_user     = "admin"                   # User with replication privileges
  database_password = var.source_mysql_password # Password (use variables for secrets)

  # Database and table selection (comma-separated)
  database_include_list = "crm,ecommerce,tst"
  table_include_list    = "crm.demo,ecommerce.customers,tst.test_id_timestamp"

  # Signal table for incremental snapshots (optional)
  signal_data_collection_schema_or_database = "crm"

  # Column filtering (optional, uses regex pattern: schema[.]table[.](col1|col2))
  column_include_list = "crm[.]demo[.](id|name),ecommerce[.]customers[.](customer_id|email)"

  # Heartbeat configuration (for monitoring replication lag)
  heartbeat_enabled                            = true # Enable heartbeat messages
  heartbeat_data_collection_schema_or_database = "crm" # Database containing heartbeat table

  # Timezone configuration
  database_connection_time_zone = "SERVER" # Options: SERVER, UTC, or specific timezone

  # GTID mode for read-only snapshot connections
  snapshot_gtid = "Yes" # Requires GTID mode enabled on MySQL (Yes/No)

  # Schema history optimization (for large instances)
  schema_history_internal_store_only_captured_databases_ddl = false
  schema_history_internal_store_only_captured_tables_ddl    = false

  # Binary data handling
  binary_handling_mode = "bytes" # Options: bytes, base64, base64-url-safe, hex

  # SSH tunnel settings (optional, for secure connections)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host = "bastion.example.com"
  # ssh_port = 22
  # ssh_user = "streamkap"
}

output "example-source-mysql" {
  value = streamkap_source_mysql.example-source-mysql.id
}
