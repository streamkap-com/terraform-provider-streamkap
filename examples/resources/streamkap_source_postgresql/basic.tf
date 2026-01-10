# Minimal PostgreSQL CDC source configuration
# This example captures changes from PostgreSQL tables using logical replication

resource "streamkap_source_postgresql" "example" {
  name = "my-postgres-source"

  # Connection details
  database_hostname = "db.example.com"
  database_port     = 5432
  database_dbname   = "mydb"
  database_user     = "streamkap_user"
  database_password = var.db_password # Use variables for secrets

  # Tables to capture (schema.table format)
  schema_include_list = "public"
  table_include_list  = "public.orders,public.customers"

  # Replication configuration
  slot_name        = "streamkap_slot"
  publication_name = "streamkap_pub"

  # Signal table for incremental snapshots (required)
  signal_data_collection_schema_or_database = "streamkap"
}

variable "db_password" {
  description = "PostgreSQL database password"
  type        = string
  sensitive   = true
}
