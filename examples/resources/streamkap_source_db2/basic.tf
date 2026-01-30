# Minimal DB2 CDC source configuration
# This example captures changes from IBM DB2 tables

resource "streamkap_source_db2" "example" {
  name = "my-db2-source"

  # Connection details
  database_hostname = "db2.example.com"
  database_port     = "50000"
  database_user     = "streamkap_user"
  database_password = var.db_password
  database_dbname   = "SAMPLE"

  # Schemas and tables to capture
  schema_include_list = "MYSCHEMA"
  table_include_list  = "MYSCHEMA.ORDERS,MYSCHEMA.CUSTOMERS"

  # Signal table for incremental snapshots (required)
  signal_data_collection_schema_or_database = "STREAMKAP"
}

variable "db_password" {
  description = "DB2 database password"
  type        = string
  sensitive   = true
}
