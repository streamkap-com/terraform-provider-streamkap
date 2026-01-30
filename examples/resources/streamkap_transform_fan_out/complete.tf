# Complete Fan Out transform configuration with all options
# Splits single records into multiple output records using JavaScript

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

resource "streamkap_transform_fan_out" "example" {
  name = "fanout-batch-events"

  # Implementation language (JavaScript only for Fan Out)
  transforms_language = "JavaScript"

  # Regex pattern to match input topics
  transforms_input_topic_pattern = "my-source\\.public\\.batch_events"

  # Output topic pattern for fanned-out records
  transforms_output_topic_pattern = "fanout-$0"

  # Input serialization format: Any, Avro, or Json
  transforms_input_serialization_format = "Avro"

  # Output serialization format: Any, Avro, or Json
  transforms_output_serialization_format = "Avro"
}

output "transform_fan_out_id" {
  value = streamkap_transform_fan_out.example.id
}
