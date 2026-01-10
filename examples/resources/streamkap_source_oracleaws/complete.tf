# Complete Oracle RDS CDC source configuration
# This example shows all available configuration options for capturing changes
# from AWS Oracle RDS tables

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

variable "source_oracle_rds_hostname" {
  type        = string
  description = "The hostname of the Oracle RDS database"
}

variable "source_oracle_rds_password" {
  type        = string
  sensitive   = true
  description = "The password of the Oracle RDS database"
}

resource "streamkap_source_oracleaws" "example-source-oracleaws" {
  # Display name for this source in Streamkap UI
  name = "example-source-oracleaws"

  # Connection settings
  database_hostname = var.source_oracle_rds_hostname
  database_port     = "1521"
  database_user     = "admin"
  database_password = var.source_oracle_rds_password
  database_dbname   = "ORCL"

  # Schema and table selection
  schema_include_list = "HR,SALES"
  table_include_list  = "HR.EMPLOYEES,HR.DEPARTMENTS,SALES.ORDERS"

  # Signal table for incremental snapshots
  signal_data_collection_schema_or_database = "STREAMKAP"

  # Heartbeat configuration
  heartbeat_enabled                            = true
  heartbeat_data_collection_schema_or_database = "STREAMKAP"

  # Schema history optimization
  schema_history_internal_store_only_captured_databases_ddl = false
  schema_history_internal_store_only_captured_tables_ddl    = false

  # Binary data handling
  binary_handling_mode = "bytes" # Options: bytes, base64, base64-url-safe, hex

  # Column filtering (optional)
  column_exclude_list = "HR.EMPLOYEES.SSN"

  # SSH tunnel settings (optional)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host = "bastion.example.com"
  # ssh_port = "22"
  # ssh_user = "streamkap"
}

output "example-source-oracleaws" {
  value = streamkap_source_oracleaws.example-source-oracleaws.id
}
