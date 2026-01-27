# Complete PlanetScale CDC source configuration
# This example shows all available configuration options for capturing changes
# from PlanetScale (Vitess) tables

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

variable "source_planetscale_hostname" {
  type        = string
  description = "The hostname of the PlanetScale database (VTGate)"
}

variable "source_planetscale_password" {
  type        = string
  sensitive   = true
  description = "The password of the PlanetScale branch"
}

resource "streamkap_source_planetscale" "example-source-planetscale" {
  # Display name for this source in Streamkap UI
  name = "example-source-planetscale"

  # Connection settings
  database_hostname = var.source_planetscale_hostname
  database_port     = "443"
  database_user     = "streamkap"
  database_password = var.source_planetscale_password

  # Vitess keyspace (database name)
  vitess_keyspace = "ecommerce"

  # Tablet type for streaming
  vitess_tablet_type = "MASTER" # Options: MASTER, REPLICA, RDONLY

  # Table selection
  table_include_list = "ecommerce.orders,ecommerce.customers,ecommerce.products"

  # Tinyint to boolean conversion
  converter_tinyint_bool = false

  # Schema history optimization
  schema_history_internal_store_only_captured_databases_ddl = false
  schema_history_internal_store_only_captured_tables_ddl    = false

  # Column filtering (optional)
  column_exclude_list = "ecommerce.customers.ssn"

  # SSH tunnel settings (optional)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host = "bastion.example.com"
  # ssh_port = "22"
  # ssh_user = "streamkap"
}

output "example-source-planetscale" {
  value = streamkap_source_planetscale.example-source-planetscale.id
}
