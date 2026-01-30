# Minimal AlloyDB CDC source configuration
# This example captures changes from AlloyDB (PostgreSQL-compatible) tables

resource "streamkap_source_alloydb" "example" {
  name = "my-alloydb-source"

  # Connection details
  database_hostname = "alloydb.example.com"
  database_port     = "5432"
  database_user     = "streamkap_user"
  database_password = var.db_password
  database_dbname   = "mydb"

  # Schemas and tables to capture
  schema_include_list = "public"
  table_include_list  = "public.orders,public.customers"

  # Signal and heartbeat tables (required)
  signal_data_collection_schema_or_database    = "streamkap"
  heartbeat_data_collection_schema_or_database = "streamkap"
}

variable "db_password" {
  description = "AlloyDB database password"
  type        = string
  sensitive   = true
}
