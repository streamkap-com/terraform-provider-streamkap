# Complete SQL Server CDC source configuration
# This example shows all available configuration options for capturing changes
# from SQL Server tables using Change Data Capture (CDC)

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
  description = "The hostname of the SQL Server database"
}

variable "source_sqlserver_password" {
  type        = string
  sensitive   = true
  description = "The password of the SQL Server database"
}

resource "streamkap_source_sqlserver" "example-source-sqlserver" {
  # Display name for this source in Streamkap UI
  name = "example-source-sqlserver"

  # Connection settings
  database_hostname = var.source_sqlserver_hostname # Database server hostname or IP
  database_port     = 1433                          # SQL Server port (default: 1433)
  database_user     = "streamkap_user"              # User with CDC access
  database_password = var.source_sqlserver_password # Password (use variables for secrets)
  database_encrypt  = "true"                        # Enable TLS encryption (true/false)

  # Database and table selection
  database_names      = "demo"       # Database to stream from
  schema_include_list = "dbo"        # Schemas to include
  table_include_list  = "dbo.Orders" # Tables to capture (schema.table format)

  # Signal table for incremental snapshots (optional, defaults to "streamkap")
  signal_data_collection_schema_or_database = "streamkap"

  # Column filtering (optional)
  # column_exclude_list = "dbo.Orders.sensitive_column"

  # Heartbeat configuration (for monitoring replication lag)
  heartbeat_enabled                            = false       # Enable heartbeat messages
  heartbeat_data_collection_schema_or_database = "streamkap" # Schema containing heartbeat table

  # Schema history optimization (for large instances)
  schema_history_internal_store_only_captured_databases_ddl = false
  schema_history_internal_store_only_captured_tables_ddl    = false

  # Binary data handling
  binary_handling_mode = "bytes" # Options: bytes, base64, base64-url-safe, hex

  # Snapshot parallelization settings
  streamkap_snapshot_parallelism           = 2     # Parallel chunk requests (1-10)
  streamkap_snapshot_large_table_threshold = 12000 # MB threshold for parallel chunking

  # Custom table configuration for parallelization
  # Format: JSON object mapping table names to chunk counts
  snapshot_custom_table_config = {
    "dbo.Orders" = {
      chunks = 2
    }
  }

  # SSH tunnel settings (optional, for secure connections)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host = "bastion.example.com"
  # ssh_port = 22
  # ssh_user = "streamkap"

  # Static field insertion (optional)
  # Add static key/value pairs to messages
  transforms_insert_static_key1_static_field   = "key_field"
  transforms_insert_static_key1_static_value   = "key_value"
  transforms_insert_static_value1_static_field = "value_field"
  transforms_insert_static_value1_static_value = "value_value"
}

output "example-source-sqlserver" {
  value = streamkap_source_sqlserver.example-source-sqlserver.id
}
