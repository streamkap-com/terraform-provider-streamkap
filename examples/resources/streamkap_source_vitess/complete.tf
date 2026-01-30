# Complete Vitess CDC source configuration
# This example shows all available configuration options for capturing changes
# from self-hosted Vitess clusters

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

variable "source_vitess_vtgate_hostname" {
  type        = string
  description = "The hostname of the Vitess VTGate server"
}

variable "source_vitess_vtgate_password" {
  type        = string
  sensitive   = true
  description = "The password for VTGate authentication"
}

variable "source_vitess_vtctld_password" {
  type        = string
  sensitive   = true
  description = "The password for VTCtld authentication"
}

resource "streamkap_source_vitess" "example-source-vitess" {
  # Display name for this source in Streamkap UI
  name = "example-source-vitess"

  # VTGate connection settings
  database_hostname = var.source_vitess_vtgate_hostname
  database_port     = "15991"
  database_user     = "vt_user"        # Optional for unauthenticated gRPC
  database_password = var.source_vitess_vtgate_password # Optional

  # Vitess keyspace
  vitess_keyspace = "ecommerce"

  # Tablet type for streaming
  vitess_tablet_type = "MASTER" # Options: MASTER, REPLICA, RDONLY

  # VTCtld connection (required for schema discovery)
  vitess_vtctld_host     = "vtctld.example.com"
  vitess_vtctld_port     = "15999"
  vitess_vtctld_user     = "admin"
  vitess_vtctld_password = var.source_vitess_vtctld_password

  # Table selection
  table_include_list = "ecommerce.orders,ecommerce.customers,ecommerce.products"

  # Column filtering (optional)
  column_exclude_list = "ecommerce.customers.ssn"

  # SSH tunnel settings (optional)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host = "bastion.example.com"
  # ssh_port = "22"
  # ssh_user = "streamkap"
}

output "example-source-vitess" {
  value = streamkap_source_vitess.example-source-vitess.id
}
