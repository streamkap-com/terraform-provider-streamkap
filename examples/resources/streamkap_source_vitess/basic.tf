# Minimal Vitess CDC source configuration
# This example captures changes from self-hosted Vitess clusters

resource "streamkap_source_vitess" "example" {
  name = "my-vitess-source"

  # VTGate connection details
  database_hostname = "vtgate.example.com"
  database_port     = "15991"

  # Vitess keyspace
  vitess_keyspace = "ecommerce"

  # VTCtld connection (required for schema discovery)
  vitess_vtctld_host     = "vtctld.example.com"
  vitess_vtctld_port     = "15999"
  vitess_vtctld_user     = "admin"
  vitess_vtctld_password = var.vtctld_password

  # Tables to capture
  table_include_list = "ecommerce.orders,ecommerce.customers"
}

variable "vtctld_password" {
  description = "VTCtld password"
  type        = string
  sensitive   = true
}
