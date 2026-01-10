# Minimal PlanetScale CDC source configuration
# This example captures changes from PlanetScale (Vitess) tables

resource "streamkap_source_planetscale" "example" {
  name = "my-planetscale-source"

  # Connection details
  database_hostname = "aws.connect.psdb.cloud"
  database_port     = "443"
  database_user     = "branch_user"
  database_password = var.db_password

  # Vitess keyspace (database)
  vitess_keyspace = "myapp"

  # Tables to capture
  table_include_list = "myapp.orders,myapp.customers"
}

variable "db_password" {
  description = "PlanetScale branch password"
  type        = string
  sensitive   = true
}
