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

variable "destination_postgresql_hostname" {
  type        = string
  description = "The hostname of the PostgreSQL database"
}
variable "destination_postgresql_password" {
  type        = string
  sensitive   = true
  description = "The password of the PostgreSQL database"
}

resource "streamkap_destination_postgresql" "example-destination-postgresql" {
  name                 = "example-destination-postgresql"
  database_hostname    = var.destination_postgresql_hostname
  database_port        = 5432
  database_dbname      = "postgres"
  database_username    = "postgresql"
  database_password    = var.destination_postgresql_password
  database_schema_name = "streamkap"
  schema_evolution     = "basic"
  insert_mode          = "insert"
  hard_delete          = false
  ssh_enabled          = false
}

output "example-destination-postgresql" {
  value = streamkap_destination_postgresql.example-destination-postgresql.id
}