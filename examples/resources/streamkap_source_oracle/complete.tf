# Complete Oracle CDC source configuration
# This example shows all available configuration options for capturing changes
# from Oracle database tables using LogMiner

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

variable "source_oracle_hostname" {
  type        = string
  description = "The hostname of the Oracle database"
}

variable "source_oracle_password" {
  type        = string
  sensitive   = true
  description = "The password of the Oracle database"
}

resource "streamkap_source_oracle" "example-source-oracle" {
  # Display name for this source in Streamkap UI
  name = "example-source-oracle"

  # Connection settings
  database_hostname = var.source_oracle_hostname
  database_port     = "1521"
  database_user     = "c##streamkap"
  database_password = var.source_oracle_password
  database_dbname   = "ORCL" # CDB name for container databases

  # For Container Database (CDB) installations, specify the PDB
  # database_pdb_name = "ORCLPDB1"

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
  column_exclude_list = "HR.EMPLOYEES.SSN,HR.EMPLOYEES.SALARY"

  # SSH tunnel settings (optional)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host = "bastion.example.com"
  # ssh_port = "22"
  # ssh_user = "streamkap"
}

output "example-source-oracle" {
  value = streamkap_source_oracle.example-source-oracle.id
}
