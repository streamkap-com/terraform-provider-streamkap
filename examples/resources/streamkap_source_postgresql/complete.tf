# Complete PostgreSQL CDC source configuration
# This example shows all available configuration options for capturing changes
# from PostgreSQL tables using logical replication (pgoutput plugin)

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
  # Display name for this source in Streamkap UI
  name = "example-source-postgresql"

  # Connection settings
  database_hostname = var.source_postgresql_hostname # Database server hostname or IP
  database_port     = 5432                           # PostgreSQL port (default: 5432)
  database_user     = "postgresql"                   # User with replication privileges
  database_password = var.source_postgresql_password # Password (use variables for secrets)
  database_dbname   = "postgres"                     # Database name to connect to

  # SSL configuration
  database_sslmode = "require" # Options: disable, allow, prefer, require, verify-ca, verify-full

  # Snapshot settings
  snapshot_read_only = "No" # "Yes" to prevent DDL during snapshots

  # Table selection (comma-separated, supports regex patterns)
  schema_include_list = "streamkap"                        # Schemas to include
  table_include_list  = "streamkap.customer,streamkap.customer2" # Tables to capture (schema.table format)

  # Column filtering (optional, uses regex pattern: schema[.]table[.](col1|col2))
  column_include_list = "streamkap[.]customer[.](id|name)"

  # Signal table for incremental snapshots (required)
  # This schema must contain a 'streamkap_signal' table for snapshot coordination
  signal_data_collection_schema_or_database = "streamkap"

  # Heartbeat configuration (optional, for monitoring replication lag)
  heartbeat_enabled                            = false # Enable heartbeat messages
  heartbeat_data_collection_schema_or_database = null  # Schema containing heartbeat table

  # Output configuration
  include_source_db_name_in_table_name = false # Prefix table names with database name

  # Replication slot and publication (must be pre-created in PostgreSQL)
  slot_name        = "terraform_pgoutput_slot" # Logical replication slot name
  publication_name = "terraform_pub"           # Publication name for the tables

  # Binary data handling
  binary_handling_mode = "bytes" # Options: bytes, base64, base64-url-safe, hex

  # SSH tunnel settings (optional, for secure connections)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host     = "bastion.example.com"
  # ssh_port     = 22
  # ssh_user     = "ssh_user"
  # ssh_password = var.ssh_password  # or use ssh_private_key
}

output "example-source-postgresql" {
  value = streamkap_source_postgresql.example-source-postgresql.id
}
