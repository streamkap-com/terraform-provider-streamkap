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

variable "destination_db2_password" {
  type        = string
  sensitive   = true
  description = "The password to access the Db2 database"
}

# Complete Db2 destination configuration with all options
resource "streamkap_destination_db2" "example" {
  name = "example-destination-db2"

  # Connection settings (required)
  database_hostname   = "db2.example.com"
  database_port       = 50000                   # Default: 50000
  database_database   = "mydb"
  connection_username = "streamkap_user"
  connection_password = var.destination_db2_password

  # Schema settings
  schema_evolution = "basic"                    # Valid values: basic, none. Default: basic

  # Write behavior
  insert_mode      = "upsert"                   # Valid values: insert, upsert. Default: insert
  delete_enabled   = true                       # Process DELETE events. Default: false
  primary_key_mode = "record_key"               # Valid values: none, record_key, record_value. Default: record_key
  primary_key_fields = "id"                     # Comma-separated list of primary key fields

  # Performance settings
  tasks_max = 5                                 # Max active tasks (1-10). Default: 5
}

output "example_destination_db2" {
  value = streamkap_destination_db2.example.id
}
