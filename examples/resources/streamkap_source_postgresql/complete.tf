# Complete PostgreSQL CDC source configuration
# Capturing changes from PostgreSQL tables using logical replication (pgoutput plugin)

terraform {
  required_providers {
    streamkap = {
      source = "streamkap-com/streamkap"
      # Set version to the release matching this documentation.
      # Beta releases require an exact prerelease version; omission selects stable.
    }
  }
  required_version = ">= 1.5.0"
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
  database_user     = "streamkap"                    # User with replication privileges
  database_password = var.source_postgresql_password # Password (use variables for secrets)
  database_dbname   = "commerce"                     # Database name to connect to

  # SSL configuration
  database_sslmode = "require" # Options: disable, allow, prefer, require, verify-ca, verify-full

  # Snapshot settings
  snapshot_read_only = "No" # "Yes" to prevent DDL during snapshots

  # Table selection (comma-separated, supports regex patterns)
  schema_include_list = "public"                         # Schemas to include
  table_include_list  = "public.orders,public.customers" # Tables to capture (schema.table format)

  # Signal table for incremental snapshots
  # This schema must contain a 'streamkap_signal' table for snapshot coordination
  signal_data_collection_schema_or_database = "public.streamkap_signal"

  # Heartbeat configuration (optional, for monitoring replication lag)
  heartbeat_enabled                            = false # Enable heartbeat messages
  heartbeat_data_collection_schema_or_database = null  # Schema containing heartbeat table

  # Output configuration
  include_source_db_name_in_table_name = false # Prefix table names with database name

  # The publication must be pre-created in PostgreSQL.
  slot_name        = "streamkap_slot"        # Logical replication slot name
  publication_name = "streamkap_publication" # Publication name for the tables

  # Binary data handling
  binary_handling_mode = "bytes" # Options: bytes, base64, base64-url-safe, hex

  # TOAST column re-selection
  # Enabled by default. Disable it to leave unavailable TOAST values as
  # placeholders instead of querying the source for them.
  post_processors_reselect_enabled        = true
  reselector_reselect_error_handling_mode = "fail" # Options: fail, warn

  # SSH tunnel settings (optional, for secure connections)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host     = "bastion.example.com"
  # ssh_port     = 22
  # ssh_user     = "ssh_user"
  # ssh_password = var.ssh_password  # or use ssh_private_key

  # Tags — optional set of tag IDs (manage via streamkap_tag or look up via
  # the streamkap_tags data source). Set explicitly to manage; omit to let the
  # backend keep whatever tags it has attached out-of-band; set to [] to clear.
  tags = [streamkap_tag.production.id]
}

resource "streamkap_tag" "production" {
  name        = "production"
  description = "Production environment resources"
  type        = ["sources", "destinations", "pipelines"]
}

output "example-source-postgresql" {
  value = streamkap_source_postgresql.example-source-postgresql.id
}
