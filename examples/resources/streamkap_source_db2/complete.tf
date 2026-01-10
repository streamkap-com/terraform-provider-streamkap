# Complete DB2 CDC source configuration
# This example shows all available configuration options for capturing changes
# from IBM DB2 tables

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

variable "source_db2_hostname" {
  type        = string
  description = "The hostname of the DB2 database"
}

variable "source_db2_password" {
  type        = string
  sensitive   = true
  description = "The password of the DB2 database"
}

resource "streamkap_source_db2" "example-source-db2" {
  # Display name for this source in Streamkap UI
  name = "example-source-db2"

  # Connection settings
  database_hostname = var.source_db2_hostname
  database_port     = "50000" # DB2 default port
  database_user     = "db2admin"
  database_password = var.source_db2_password
  database_dbname   = "SAMPLE"

  # Schema and table selection
  schema_include_list = "MYSCHEMA,ANALYTICS"
  table_include_list  = "MYSCHEMA.ORDERS,MYSCHEMA.CUSTOMERS,ANALYTICS.EVENTS"

  # Signal table for incremental snapshots
  signal_data_collection_schema_or_database = "STREAMKAP"

  # Schema history optimization (for large instances)
  schema_history_internal_store_only_captured_databases_ddl = false
  schema_history_internal_store_only_captured_tables_ddl    = false

  # Column filtering (optional)
  column_exclude_list = "MYSCHEMA.CUSTOMERS.ssn,MYSCHEMA.CUSTOMERS.credit_card"

  # SSH tunnel settings (optional)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host = "bastion.example.com"
  # ssh_port = "22"
  # ssh_user = "streamkap"
}

output "example-source-db2" {
  value = streamkap_source_db2.example-source-db2.id
}
