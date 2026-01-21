# Complete Un-Nesting transform configuration with all options
# Flattens arrays and nested records into separate output records

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

resource "streamkap_transform_un_nesting" "example" {
  name = "unnest-order-items"

  # Regex pattern to match input topics with nested data
  transforms_input_topic_pattern = "my-source\\.public\\.orders"

  # Output topic pattern for flattened records
  # $0 refers to the matched input topic name
  transforms_output_topic_pattern = "unnested-$0"

  # Format of the input topics
  # Valid values: Any, Avro, Json
  transforms_input_serialization_format = "Avro"

  # Format of the output topics
  # Valid values: Any, Avro, Json
  transforms_output_serialization_format = "Avro"
}

output "transform_un_nesting_id" {
  value = streamkap_transform_un_nesting.example.id
}
