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

variable "destination_mysql_password" {
  type        = string
  sensitive   = true
  description = "The password to access the MySQL database"
}

# Complete MySQL destination configuration with all options
resource "streamkap_destination_mysql" "example" {
  name = "example-destination-mysql"

  # Connection settings (required)
  database_hostname   = "mysql.example.com"
  database_port       = 3306                    # Default: 3306
  database_database   = "mydb"
  connection_username = "streamkap_user"
  connection_password = var.destination_mysql_password

  # Schema settings
  schema_evolution = "basic"                    # Valid values: basic, none. Default: basic

  # Write behavior
  insert_mode      = "upsert"                   # Valid values: insert, upsert. Default: insert
  delete_enabled   = true                       # Process DELETE events. Default: false
  primary_key_mode = "record_key"               # Valid values: none, record_key, record_value. Default: record_key
  primary_key_fields = "id"                     # Comma-separated list of primary key fields

  # Performance settings
  tasks_max = 5                                 # Max active tasks (1-10). Default: 5

  # Table mapping
  topic2table_map                          = true  # Use Streamkap's default mapping. Default: false
  transforms_change_topic_name_match_regex = "^.*\\.(.*)\\.(.*)$" # Regex for topic name matching
  transforms_change_topic_name_mapping     = "source_table:dest_table" # Source to destination table mapping
}

output "example_destination_mysql" {
  value = streamkap_destination_mysql.example.id
}
