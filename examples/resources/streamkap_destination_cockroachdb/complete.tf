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

variable "destination_cockroachdb_password" {
  type        = string
  sensitive   = true
  description = "The password to access the CockroachDB database"
}

# Complete CockroachDB destination configuration with all options
resource "streamkap_destination_cockroachdb" "example" {
  name = "example-destination-cockroachdb"

  # Connection settings (required)
  database_hostname   = "cockroachdb.example.com"
  database_port       = 26257                   # Default: 26257
  database_database   = "mydb"
  connection_username = "streamkap_user"
  connection_password = var.destination_cockroachdb_password

  # Schema settings
  table_name_prefix = "public"                  # Schema for table names. Default: public
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

output "example_destination_cockroachdb" {
  value = streamkap_destination_cockroachdb.example.id
}
