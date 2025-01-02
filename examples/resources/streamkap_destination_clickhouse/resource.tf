terraform {
  required_providers {
    streamkap = {
      source = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

variable "destination_clickhouse_hostname" {
  type        = string
  description = "The hostname of the Clickhouse server"
}

variable "destination_clickhouse_connection_username" {
  type        = string
  description = "The username to connect to the Clickhouse server"
}

variable "destination_clickhouse_connection_password" {
  type        = string
  description = "The password to connect to the Clickhouse server"
}

resource "streamkap_destination_clickhouse" "example-destination-clickhouse" {
  name                = "example-destination-clickhouse"
  ingestion_mode      = "append"
  tasks_max           = 5
  hostname            = var.destination_clickhouse_hostname
  connection_username = var.destination_clickhouse_connection_username
  connection_password = var.destination_clickhouse_connection_password
  port                = 8443
  database            = "demo"
  ssl                 = true
  topics_config_map = {
    "public.users" = {
      delete_sql_execute = "SELECT 1;"
    }
  }
}

output "example-destination-clickhouse" {
  value = streamkap_destination_clickhouse.example-destination-clickhouse.id
}