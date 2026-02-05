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

variable "destination_motherduck_token" {
  type        = string
  sensitive   = true
  description = "Motherduck authentication token"
}

# Complete Motherduck destination configuration with all options
resource "streamkap_destination_motherduck" "example" {
  name = "example-destination-motherduck"

  # Connection settings (required)
  motherduck_token   = var.destination_motherduck_token
  motherduck_catalog = "my_database"            # Database/catalog name

  # Ingestion settings
  ingestion_mode   = "upsert"                   # Valid values: upsert, append. Default: upsert
  schema_evolution = "basic"                    # Valid values: basic, none. Default: basic
  table_name_prefix = "streamkap"               # Schema for table names. Default: streamkap

  # Delete handling
  hard_delete = true                            # Process DELETE events. Default: false

  # Performance settings
  tasks_max = 5                                 # Max active tasks (1-25). Default: 5

  # Table mapping
  topic2table_map                          = true  # Use Streamkap's default mapping. Default: false
  transforms_change_topic_name_match_regex = "^.*\\.(.*)\\.(.*)$" # Regex for topic name matching
  transforms_change_topic_name_mapping     = "source_table:dest_table" # Source to destination table mapping
}

output "example_destination_motherduck" {
  value = streamkap_destination_motherduck.example.id
}
