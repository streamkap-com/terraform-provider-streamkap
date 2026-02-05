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

variable "destination_azblob_connection_string" {
  type        = string
  sensitive   = true
  description = "The Azure Blob Storage connection string or SAS URL"
}

# Complete Azure Blob Storage destination configuration with all options
resource "streamkap_destination_azblob" "example" {
  name = "example-destination-azblob"

  # Connection settings (required)
  azblob_connection_string = var.destination_azblob_connection_string

  # Container configuration (optional)
  azblob_container_name = "streamkap-data"

  # File format settings
  format                 = "json"               # Valid values: json, csv, avro, parquet. Default: json
  format_csv_write_headers = false              # Include column headers in CSV files. Default: false

  # File organization
  topics_dir        = "data/topics"             # Top level directory for storing data
  file_name_template = "{{topic}}-{{partition}}-{{start_offset}}" # Default filename template

  # Performance settings
  flush_size          = 1000                    # Number of records per file. Default: 1000
  file_size           = 65536                   # Minimum file size in bytes. Default: 65536
  rotate_interval_ms  = -1                      # Max wait time in ms before writing. Default: -1 (disabled)

  # Compression
  compression = "gzip"                          # Compression type for output files
}

output "example_destination_azblob" {
  value = streamkap_destination_azblob.example.id
}
