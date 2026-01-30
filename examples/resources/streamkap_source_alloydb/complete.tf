# Complete AlloyDB CDC source configuration
# This example shows all available configuration options for capturing changes
# from AlloyDB tables using PostgreSQL logical replication

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

variable "source_alloydb_hostname" {
  type        = string
  description = "The hostname of the AlloyDB database"
}

variable "source_alloydb_password" {
  type        = string
  sensitive   = true
  description = "The password of the AlloyDB database"
}

resource "streamkap_source_alloydb" "example-source-alloydb" {
  # Display name for this source in Streamkap UI
  name = "example-source-alloydb"

  # Connection settings
  database_hostname = var.source_alloydb_hostname
  database_port     = "5432"
  database_user     = "streamkap_user"
  database_password = var.source_alloydb_password
  database_dbname   = "mydb"

  # SSL mode for secure connections
  database_sslmode = "require" # Options: require, disable

  # Read-only mode for snapshots
  snapshot_read_only = "Yes" # Options: Yes, No

  # Signal table for incremental snapshots
  signal_data_collection_schema_or_database = "streamkap"

  # Schema and table selection
  schema_include_list = "public,analytics"
  table_include_list  = "public.orders,public.customers,analytics.events"

  # Column filtering
  column_include_list_toggled = true
  column_include_list         = "public[.]orders[.](id|customer_id|total)"

  # Heartbeat configuration
  heartbeat_enabled                            = true
  heartbeat_data_collection_schema_or_database = "streamkap"

  # Replication slot settings
  slot_name        = "streamkap_pgoutput_slot"
  publication_name = "streamkap_pub"

  # Topic naming
  include_source_db_name_in_table_name = false

  # Binary data handling
  binary_handling_mode = "bytes" # Options: bytes, base64, base64-url-safe, hex

  # SSH tunnel settings (optional)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host = "bastion.example.com"
  # ssh_port = "22"
  # ssh_user = "streamkap"
}

output "example-source-alloydb" {
  value = streamkap_source_alloydb.example-source-alloydb.id
}
