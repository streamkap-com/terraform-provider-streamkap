# Complete Informix CDC source configuration.
# Shows the commonly used options for capturing changes from IBM Informix.

terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 3.0.0"
    }
  }
  required_version = ">= 1.5.0"
}

provider "streamkap" {}

variable "source_informix_hostname" {
  type        = string
  description = "The hostname of the Informix database"
}

variable "source_informix_password" {
  type        = string
  sensitive   = true
  description = "The password of the Informix database"
}

resource "streamkap_source_informix" "example-source-informix" {
  # Display name for this source in Streamkap UI
  name = "example-source-informix"

  # Connection settings
  database_hostname = var.source_informix_hostname
  database_port     = 9088 # Informix default port
  database_user     = "streamkap_user"
  database_password = var.source_informix_password
  database_dbname   = "stores_demo"

  # Schema and table selection
  schema_include_list = "informix"
  table_include_list  = "informix.orders,informix.customer,informix.items"

  # Signal table for incremental snapshots (defaults to "streamkap")
  signal_data_collection_schema_or_database = "streamkap"

  # Schema history optimization (for large instances)
  schema_history_internal_store_only_captured_databases_ddl = false
  schema_history_internal_store_only_captured_tables_ddl    = false

  # Column filtering (optional)
  column_exclude_list = "informix.customer.ssn"

  # Heartbeat keeps offsets advancing on low-traffic sources
  heartbeat_enabled = true

  # SSH tunnel settings (optional)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host = "bastion.example.com"
  # ssh_port = 22
  # ssh_user = "streamkap"
}

output "example-source-informix" {
  value = streamkap_source_informix.example-source-informix.id
}
