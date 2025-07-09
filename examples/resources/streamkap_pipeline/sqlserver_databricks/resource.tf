terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com-test/streamkap"
      version = ">= 2.0.0"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

variable "source_sqlserver_hostname" {
  type        = string
  description = "The hostname of the SQLServer database"
}
variable "source_sqlserver_password" {
  type        = string
  sensitive   = true
  description = "The password of the SQLServer database"
}

resource "streamkap_source_sqlserver" "example-source-sqlserver" {
  name                                         = "example-source-sqlserver"
  database_hostname                            = var.source_sqlserver_hostname
  database_port                                = 1433
  database_user                                = "admin"
  database_password                            = var.source_sqlserver_password
  database_dbname                              = "sqlserverdemo"
  schema_include_list                          = "dbo"
  table_include_list                           = "dbo.Orders,dbo.Customers"
  signal_data_collection_schema_or_database    = "streamkap"
  heartbeat_enabled                            = false
  heartbeat_data_collection_schema_or_database = null
  binary_handling_mode                         = "bytes"
  ssh_enabled                                  = false
}

variable "destination_databricks_connection_url" {
  type        = string
  description = "The JDBC url of the databricks server"
}

variable "destination_databricks_token" {
  type        = string
  sensitive   = true
  description = "The token for the Databricks database"
}

resource "streamkap_destination_databricks" "example-destination-databricks" {
  name                = "example-destination-databricks"
  table_name_prefix   = "streamkap"
  ingestion_mode      = "append"
  partition_mode      = "by_topic"
  hard_delete         = true
  tasks_max           = 5
  connection_url      = var.destination_databricks_connection_url
  databricks_token    = var.destination_databricks_token
  schema_evolution    = "basic"
}

resource "streamkap_pipeline" "example-pipeline" {
  name                = "example-pipeline"
  snapshot_new_tables = true
  source = {
    id        = streamkap_source_sqlserver.example-source-sqlserver.id
    name      = streamkap_source_sqlserver.example-source-sqlserver.name
    connector = streamkap_source_sqlserver.example-source-sqlserver.connector
    topics = [
      "default.warehouse-test-2",
    ]
  }
  destination = {
    id        = streamkap_destination_databricks.example-destination-databricks.id
    name      = streamkap_destination_databricks.example-destination-databricks.name
    connector = streamkap_destination_databricks.example-destination-databricks.connector
  }
}

output "example-pipeline" {
  value = streamkap_pipeline.example-pipeline.id
}