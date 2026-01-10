# Minimal Supabase CDC source configuration
# This example captures changes from Supabase (PostgreSQL) tables

resource "streamkap_source_supabase" "example" {
  name = "my-supabase-source"

  # Connection details
  database_hostname = "db.xxxx.supabase.co"
  database_port     = "5432"
  database_user     = "postgres"
  database_password = var.db_password
  database_dbname   = "postgres"

  # Schemas and tables to capture
  schema_include_list = "public"
  table_include_list  = "public.orders,public.customers"

  # Signal and heartbeat tables (required)
  signal_data_collection_schema_or_database    = "streamkap"
  heartbeat_data_collection_schema_or_database = "streamkap"
}

variable "db_password" {
  description = "Supabase database password"
  type        = string
  sensitive   = true
}
