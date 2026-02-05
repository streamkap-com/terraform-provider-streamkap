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

variable "destination_sqlserver_password" {
  type        = string
  sensitive   = true
  description = "The password to access the SQL Server database"
}

# Complete SQL Server destination configuration with all options
resource "streamkap_destination_sqlserver" "example" {
  name = "example-destination-sqlserver"

  # Connection settings (required)
  database_hostname   = "sqlserver.example.com"
  database_port       = 1433                    # Default: 1433
  database_database   = "mydb"
  connection_username = "streamkap_user"
  connection_password = var.destination_sqlserver_password

  # Schema settings (required)
  table_name_prefix = "dbo"                     # Schema for table names
  schema_evolution  = "basic"                   # Valid values: basic, none. Default: basic

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

output "example_destination_sqlserver" {
  value = streamkap_destination_sqlserver.example.id
}
