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
  description = "The password for the PostgreSQL database"
}

# Complete PostgreSQL destination configuration with all options
resource "streamkap_destination_postgresql" "example" {
  name = "example-destination-postgresql"

  # Connection settings (required)
  database_hostname   = var.destination_postgresql_hostname
  database_port       = 5432
  connection_username = "streamkap"
  connection_password = var.destination_postgresql_password

  # Database name
  database_database = "sandbox"

  # Schema prefix for table names (required)
  table_name_prefix = "streamkap"

  # Schema evolution: basic or none
  # Use 'none' for pre-created destination tables
  schema_evolution = "basic"

  # Insert mode: insert or upsert
  insert_mode = "upsert"

  # Enable hard deletes (propagate DELETE operations)
  delete_enabled = true

  # Primary key configuration
  # primary_key_mode: kafka, record_key, record_value, none
  # primary_key_fields: comma-separated list of fields (when mode is record_value)

  # Parallel tasks (default: 1)
  tasks_max = 2

  # SSH tunnel configuration (optional)
  ssh_enabled = false
  # ssh_host = "bastion.example.com"
  # ssh_port = "22"
  # ssh_user = "streamkap"

  # Topic to table mapping (optional)
  # topic2table_map = true
  # transforms_change_topic_name_match_regex = ".*\\.(.*)"
  # transforms_change_topic_name_mapping     = "$1"
}

output "postgresql_destination_id" {
  value = streamkap_destination_postgresql.example.id
}
