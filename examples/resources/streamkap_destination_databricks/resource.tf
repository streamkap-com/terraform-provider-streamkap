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
  name               = "example-destination-databricks"
  table_name_prefix  = "streamkap"
  ingestion_mode     = "append"
  partition_mode     = "by_topic"
  hard_delete        = true
  tasks_max          = 5
  connection_url     = var.destination_databricks_connection_url
  databricks_token   = var.destination_databricks_token
  databricks_catalog = "hive_metastore"
  schema_evolution   = "basic"
}

output "example-destination-databricks" {
  value = streamkap_destination_databricks.example-destination-databricks.id
}