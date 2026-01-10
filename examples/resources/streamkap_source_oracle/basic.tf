# Minimal Oracle CDC source configuration
# This example captures changes from Oracle database tables using LogMiner

resource "streamkap_source_oracle" "example" {
  name = "my-oracle-source"

  # Connection details
  database_hostname = "oracle.example.com"
  database_port     = "1521"
  database_user     = "streamkap_user"
  database_password = var.db_password
  database_dbname   = "ORCL"

  # Schemas and tables to capture
  schema_include_list = "HR"
  table_include_list  = "HR.EMPLOYEES,HR.DEPARTMENTS"

  # Signal and heartbeat tables (required)
  signal_data_collection_schema_or_database    = "STREAMKAP"
  heartbeat_data_collection_schema_or_database = "STREAMKAP"
}

variable "db_password" {
  description = "Oracle database password"
  type        = string
  sensitive   = true
}
