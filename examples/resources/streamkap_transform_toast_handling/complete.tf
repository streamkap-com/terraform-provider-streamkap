# Complete TOAST Handling transform configuration with all options
# Handles PostgreSQL TOAST (The Oversized-Attribute Storage Technique) columns
# by reconstructing large values that were stored externally

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

resource "streamkap_transform_toast_handling" "example" {
  name = "toast-handling-orders"

  # Language for the transform
  # Valid values: JavaScript, Python
  transforms_language = "Python"

  # Regex pattern to match input topics containing TOAST data
  transforms_input_topic_pattern = "my-source\\.public\\.orders"

  # Output topic pattern for processed records with TOAST values reconstructed
  # $0 refers to the matched input topic name
  transforms_output_topic_pattern = "toast-handled-$0"

  # Format of the input topics
  # Valid values: Any, Avro, Json
  transforms_input_serialization_format = "Avro"

  # Format of the output topics
  # Valid values: Any, Avro, Json
  transforms_output_serialization_format = "Avro"
}

output "transform_toast_handling_id" {
  value = streamkap_transform_toast_handling.example.id
}
