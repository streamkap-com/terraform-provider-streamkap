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

variable "source_mysql_hostname" {
  type        = string
  description = "The hostname of the MySQL database"
}

variable "source_mysql_password" {
  type        = string
  sensitive   = true
  description = "The password of the MySQL database"
}

resource "streamkap_source_mysql" "example-source-mysql" {
  name                         = "example-source-mysql"
  database_hostname            = var.source_mysql_hostname
  database_port                = 3306
  database_user                = "admin"
  database_password            = var.source_mysql_password
  database_include_list        = "crm,ecommerce,tst"
  table_include_list           = "crm.demo,ecommerce.Customers,tst.test_id_timestamp"
  heartbeat_enabled            = false
  database_connection_timezone = "SERVER"
  snapshot_gtid                = true
  binary_handling_mode         = "bytes"
  ssh_enabled                  = false
}

output "example-source-mysql" {
  value = streamkap_source_mysql.example-source-mysql.id
}